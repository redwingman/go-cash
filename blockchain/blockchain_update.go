package blockchain

import (
	"pandora-pay/blockchain/blockchain_sync"
	"pandora-pay/blockchain/blockchain_types"
	"pandora-pay/blockchain/blocks/block_complete"
	"pandora-pay/blockchain/data_storage"
	"pandora-pay/blockchain/transactions/transaction"
	"pandora-pay/gui"
	"pandora-pay/helpers/multicast"
	"pandora-pay/helpers/recovery"
	"pandora-pay/network/websocks/connection/advanced_connection_types"
)

type BlockchainDataUpdate struct {
	Update        *BlockchainData
	ChainSyncData *blockchain_sync.BlockchainSyncData
}

type blockchainUpdate struct {
	err                    error
	newChainData           *BlockchainData
	dataStorage            *data_storage.DataStorage
	allTransactionsChanges []*blockchain_types.BlockchainTransactionUpdate
	removedTxHashes        map[string][]byte
	removedTxsList         [][]byte //ordered kept
	insertedTxs            map[string]*transaction.Transaction
	insertedTxsList        []*transaction.Transaction
	insertedBlocks         []*block_complete.BlockComplete
	calledByForging        bool
	exceptSocketUUID       advanced_connection_types.UUID
}

type blockchainUpdatesQueue struct {
	updatesCn            chan *blockchainUpdate //buffered
	updatesMempool       *multicast.MulticastChannel[*blockchainUpdate]
	updatesNotifications *multicast.MulticastChannel[*blockchainUpdate]
	chain                *blockchain
}

func createBlockchainUpdatesQueue() *blockchainUpdatesQueue {
	return &blockchainUpdatesQueue{
		make(chan *blockchainUpdate, 100),
		multicast.NewMulticastChannel[*blockchainUpdate](),
		multicast.NewMulticastChannel[*blockchainUpdate](),
		nil,
	}
}

func (self *blockchainUpdatesQueue) hasCalledByForging(updates []*blockchainUpdate) bool {
	for _, update := range updates {
		if update.calledByForging {
			return true
		}
	}
	return false
}

func (self *blockchainUpdatesQueue) lastSuccess(updates []*blockchainUpdate) *BlockchainData {
	for i := len(updates) - 1; i >= 0; i-- {
		if updates[i].err == nil {
			return updates[i].newChainData
		}
	}

	return nil
}

func (self *blockchainUpdatesQueue) executeUpdate(update *blockchainUpdate) (err error) {

	update.newChainData.updateChainInfo()

	self.chain.UpdateNewChainUpdate.Broadcast(&blockchain_types.BlockchainUpdates{
		update.dataStorage.AccsCollection,
		update.dataStorage.PlainAccs,
		update.dataStorage.Asts,
		update.dataStorage.Regs,
		update.newChainData.Height,
		update.newChainData.Hash,
	})

	chainSyncData := self.chain.Sync.AddBlocksChanged(uint32(len(update.insertedBlocks)), true)

	self.updatesMempool.Broadcast(update)
	self.updatesNotifications.Broadcast(update)

	gui.GUI.Log("self.chain.UpdateNewChain fired")
	self.chain.UpdateNewChain.Broadcast(update.newChainData.Height)

	self.chain.UpdateNewChainDataUpdate.Broadcast(&BlockchainDataUpdate{
		update.newChainData,
		chainSyncData,
	})

	return nil
}

func (self *blockchainUpdatesQueue) processBlockchainUpdatesQueue() {
	recovery.SafeGo(func() {

		for {

			works := make([]*blockchainUpdate, 0)
			update, _ := <-self.updatesCn
			works = append(works, update)

			loop := true
			for loop {
				select {
				case update, _ = <-self.updatesCn:
					works = append(works, update)
				default:
					loop = false
				}
			}

			lastSuccessUpdate := self.lastSuccess(works)
			updateForging := lastSuccessUpdate != nil || self.hasCalledByForging(works)
			for _, update = range works {
				if update.err == nil {
					if err := self.executeUpdate(update); err != nil {
						gui.GUI.Error("Error processUpdate", err)
					}
				}
			}

			chainSyncData := self.chain.Sync.GetSyncData()
			if chainSyncData.Started {
				//create next block and the workers will be automatically reset
				self.chain.createNextBlockForForging(lastSuccessUpdate, updateForging)
			}

		}

	})
}
