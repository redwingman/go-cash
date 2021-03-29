package forging

import (
	"github.com/tevino/abool"
	"math/big"
	"pandora-pay/blockchain/block-complete"
	"pandora-pay/config"
	"pandora-pay/gui"
	"pandora-pay/mempool"
)

type Forging struct {
	mempool    *mempool.Mempool
	Wallet     *ForgingWallet
	workCn     chan *ForgingWork
	started    *abool.AtomicBool
	SolutionCn chan *block_complete.BlockComplete
}

func ForgingInit(mempool *mempool.Mempool) (forging *Forging, err error) {

	forging = &Forging{
		mempool:    mempool,
		workCn:     nil,
		started:    abool.New(),
		SolutionCn: make(chan *block_complete.BlockComplete),
		Wallet: &ForgingWallet{
			addressesMap: make(map[string]*ForgingWalletAddress),
		},
	}

	gui.Log("Forging Init")

	return
}

func (forging *Forging) StartForging() bool {

	if !forging.started.SetToIf(false, true) {
		return false
	}

	forging.workCn = make(chan *ForgingWork)
	forgingThread := createForgingThread(config.CPU_THREADS, forging.mempool, forging.SolutionCn, forging.workCn, forging.Wallet)
	go forgingThread.startForging()

	return true
}

func (forging *Forging) StopForging() bool {
	if forging.started.SetToIf(true, false) {
		close(forging.workCn) //this will close the thread
		return true
	}
	return false
}

//thread safe
func (forging *Forging) ForgingNewWork(blkComplete *block_complete.BlockComplete, target *big.Int) {

	work := &ForgingWork{
		blkComplete: blkComplete,
		target:      target,
	}

	if forging.started.IsSet() {
		forging.workCn <- work
	}
}

func (forging *Forging) Close() {
	forging.StopForging()
	close(forging.SolutionCn) //this will close the thread
}
