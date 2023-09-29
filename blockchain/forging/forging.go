package forging

import (
	"github.com/tevino/abool"
	"pandora-pay/blockchain/blockchain_types"
	"pandora-pay/blockchain/blocks/block_complete"
	"pandora-pay/blockchain/forging/forging_block_work"
	"pandora-pay/blockchain/transactions/transaction"
	"pandora-pay/config"
	"pandora-pay/gui"
	"pandora-pay/helpers/generics"
	"pandora-pay/helpers/multicast"
	"pandora-pay/helpers/recovery"
)

type forging struct {
	Wallet             *forgingWallet
	started            *abool.AtomicBool
	forgingThread      *forgingThread
	nextBlockCreatedCn <-chan *forging_block_work.ForgingWork
	forgingSolutionCn  chan<- *blockchain_types.BlockchainSolution
}

var Forging *forging

func (self *forging) InitializeForging(createForgingTransactions func(*block_complete.BlockComplete, []byte, uint64, []*transaction.Transaction) (*transaction.Transaction, error), nextBlockCreatedCn <-chan *forging_block_work.ForgingWork, updateNewChainUpdate *multicast.MulticastChannel[*blockchain_types.BlockchainUpdates], forgingSolutionCn chan<- *blockchain_types.BlockchainSolution) {

	self.nextBlockCreatedCn = nextBlockCreatedCn
	self.Wallet.updateNewChainUpdate = updateNewChainUpdate
	self.forgingSolutionCn = forgingSolutionCn

	self.forgingThread = createForgingThread(config.CPU_THREADS, createForgingTransactions, self.forgingSolutionCn, self.nextBlockCreatedCn)
	self.Wallet.workersCreatedCn = self.forgingThread.workersCreatedCn
	self.Wallet.workersDestroyedCn = self.forgingThread.workersDestroyedCn

	self.Wallet.initialized.Set()
	recovery.SafeGo(self.Wallet.runProcessUpdates)
	recovery.SafeGo(self.Wallet.runDecryptBalanceAndNotifyWorkers)

}

func (self *forging) StartForging() bool {

	if config.NODE_CONSENSUS != config.NODE_CONSENSUS_TYPE_FULL {
		gui.GUI.Warning(`Staking was not started as "--node-consensus=full" is missing`)
		return false
	}

	if !self.started.SetToIf(false, true) {
		return false
	}

	self.forgingThread.startForging()

	return true
}

func (self *forging) StopForging() bool {
	if self.started.SetToIf(true, false) {
		return true
	}
	return false
}

func (self *forging) Close() {
	self.StopForging()
}

func Initialize() error {

	Forging = &forging{
		&forgingWallet{
			map[string]*forgingWalletAddress{},
			[]int{},
			[]*ForgingWorkerThread{},
			nil,
			make(chan *forgingWalletAddressUpdate),
			nil,
			nil,
			&generics.Map[string, *forgingWalletAddress]{},
			nil,
			abool.New(),
		},
		abool.New(),
		nil, nil, nil,
	}
	Forging.Wallet.forging = Forging

	return nil
}
