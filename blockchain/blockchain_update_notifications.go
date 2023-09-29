package blockchain

import (
	"bytes"
	"pandora-pay/blockchain/transactions/transaction"
	"pandora-pay/helpers/advanced_buffers"
	"pandora-pay/helpers/recovery"
	"pandora-pay/mempool"
	"pandora-pay/txs_validator"
)

func (queue *blockchainUpdatesQueue) processBlockchainUpdateNotifications() {
	recovery.SafeGo(func() {

		updatesNotificationsCn := queue.updatesNotifications.AddListener()
		defer queue.updatesNotifications.RemoveChannel(updatesNotificationsCn)

		for {
			update := <-updatesNotificationsCn

			queue.chain.UpdateSocketsSubscriptionsNotifications.Broadcast(update.dataStorage)
		}

	})
}

// async mempool notifications
func (queue *blockchainUpdatesQueue) processBlockchainUpdateMempool() {
	recovery.SafeGo(func() {

		var err error

		updatesMempoolCn := queue.updatesMempool.AddListener()
		defer queue.updatesMempool.RemoveChannel(updatesMempoolCn)

		for {

			update := <-updatesMempoolCn

			//let's remove the transactions from the mempool
			if len(update.insertedTxsList) > 0 {
				hashes := make([]string, len(update.insertedTxsList))
				for i, tx := range update.insertedTxsList {
					if tx != nil {
						hashes[i] = tx.Bloom.HashStr
					}
				}
				mempool.Mempool.RemoveInsertedTxsFromBlockchain(hashes)
			}

			//let's add the transactions in the mempool
			if len(update.removedTxsList) > 0 {

				removedTxs := make([]*transaction.Transaction, len(update.removedTxsList))
				for i, txData := range update.removedTxsList {
					tx := &transaction.Transaction{}
					if err = tx.Deserialize(advanced_buffers.NewBufferReader(txData)); err != nil {
						return
					}
					if err = txs_validator.TxsValidator.MarkAsValidatedTx(tx); err != nil {
						return
					}

					removedTxs[i] = tx
					for _, change := range update.allTransactionsChanges {
						if bytes.Equal(change.TxHash, tx.Bloom.Hash) {
							change.Tx = tx
						}
					}
				}

				mempool.Mempool.InsertRemovedTxsFromBlockchain(removedTxs, update.newChainData.Height)
			}

			queue.chain.UpdateSocketsSubscriptionsTransactions.Broadcast(update.allTransactionsChanges)
		}

	})
}
