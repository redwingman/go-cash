package blockchain

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"pandora-pay/blockchain/blockchain_sync"
	"pandora-pay/blockchain/blockchain_types"
	"pandora-pay/blockchain/blocks/block/difficulty"
	"pandora-pay/blockchain/blocks/block_complete"
	"pandora-pay/blockchain/data_storage"
	"pandora-pay/blockchain/data_storage/assets/asset"
	"pandora-pay/blockchain/forging/forging_block_work"
	"pandora-pay/blockchain/transactions/transaction"
	"pandora-pay/blockchain/transactions/transaction/transaction_type"
	"pandora-pay/blockchain/transactions/transaction/transaction_zether"
	"pandora-pay/blockchain/transactions/transaction/transaction_zether/transaction_zether_payload/transaction_zether_payload_extra"
	"pandora-pay/blockchain/transactions/transaction/transaction_zether/transaction_zether_payload/transaction_zether_payload_script"
	"pandora-pay/config"
	"pandora-pay/config/config_coins"
	"pandora-pay/config/config_stake"
	"pandora-pay/gui"
	"pandora-pay/helpers/generics"
	"pandora-pay/helpers/multicast"
	"pandora-pay/mempool"
	"pandora-pay/network/websocks/connection/advanced_connection_types"
	"pandora-pay/store"
	"pandora-pay/store/store_db/store_db_interface"
	"pandora-pay/txs_validator"
	"strconv"
	"sync"
	"time"
)

type blockchain struct {
	ChainData                               *generics.Value[*BlockchainData]
	Sync                                    *blockchain_sync.BlockchainSync
	mutex                                   *sync.Mutex //writing mutex
	updatesQueue                            *blockchainUpdatesQueue
	ForgingSolutionCn                       chan *blockchain_types.BlockchainSolution
	UpdateNewChain                          *multicast.MulticastChannel[uint64]
	UpdateNewChainDataUpdate                *multicast.MulticastChannel[*BlockchainDataUpdate]
	UpdateNewChainUpdate                    *multicast.MulticastChannel[*blockchain_types.BlockchainUpdates]
	UpdateSocketsSubscriptionsTransactions  *multicast.MulticastChannel[[]*blockchain_types.BlockchainTransactionUpdate]
	UpdateSocketsSubscriptionsNotifications *multicast.MulticastChannel[*data_storage.DataStorage]
	NextBlockCreatedCn                      chan *forging_block_work.ForgingWork
}

var Blockchain *blockchain

func (self *blockchain) validateBlocks(blocksComplete []*block_complete.BlockComplete) (err error) {

	if len(blocksComplete) == 0 {
		return errors.New("Blocks length is ZERO")
	}

	for _, blkComplete := range blocksComplete {

		if err = blkComplete.Verify(); err != nil {
			return
		}

		if err = txs_validator.TxsValidator.ValidateTxs(blkComplete.Txs); err != nil {
			return
		}

	}

	return
}

func (self *blockchain) AddBlocks(blocksComplete []*block_complete.BlockComplete, calledByForging bool, exceptSocketUUID advanced_connection_types.UUID) (kernelHash []byte, err error) {

	if err = self.validateBlocks(blocksComplete); err != nil {
		return
	}

	//avoid processing the same function twice
	self.mutex.Lock()
	defer self.mutex.Unlock()

	chainData := self.GetChainData()

	if calledByForging && blocksComplete[len(blocksComplete)-1].Height == chainData.Height-1 && chainData.ConsecutiveSelfForged > 0 {
		err = errors.New("Block was already forged by a different thread")
		return
	}

	gui.GUI.Info("Including blocks " + strconv.FormatUint(blocksComplete[0].Height, 10) + " ... " + strconv.FormatUint(blocksComplete[len(blocksComplete)-1].Height, 10))

	allTransactionsChanges := []*blockchain_types.BlockchainTransactionUpdate{}

	insertedBlocks := []*block_complete.BlockComplete{}

	//remove blocks which are different
	removedTxHashes := make(map[string][]byte)
	insertedTxs := make(map[string]*transaction.Transaction)

	var removedTxsList [][]byte                    //ordered list
	var insertedTxsList []*transaction.Transaction //ordered list

	removedBlocksHeights := []uint64{}
	removedBlocksTransactionsCount := uint64(0)

	var dataStorage *data_storage.DataStorage
	var newChainData *BlockchainData

	err = func() (err error) {

		mempool.Mempool.SuspendProcessingCn <- struct{}{}

		err = store.StoreBlockchain.DB.Update(func(writer store_db_interface.StoreDBTransactionInterface) (err error) {

			chainData = self.GetChainData()
			newChainData = chainData.clone()

			defer func() {
				if errReturned := recover(); errReturned != nil {
					err = errReturned.(error)
				}
			}()

			savedBlock := false

			dataStorage = data_storage.NewDataStorage(writer)

			//let's filter existing blocks
			for i := len(blocksComplete) - 1; i >= 0; i-- {

				blkComplete := blocksComplete[i]

				if blkComplete.Block.Height < newChainData.Height {
					var hash []byte
					if hash, err = self.LoadBlockHash(writer, blkComplete.Block.Height); err != nil {
						return
					}
					if bytes.Equal(hash, blkComplete.Block.Bloom.Hash) {
						blocksComplete = blocksComplete[i+1:]
						break
					}
				}

			}

			if len(blocksComplete) == 0 {
				return errors.New("blocks are identical now")
			}

			firstBlockComplete := blocksComplete[0]
			if firstBlockComplete.Block.Height < newChainData.Height {

				index := newChainData.Height - 1
				for {

					removedBlocksHeights = append(removedBlocksHeights, 0)
					copy(removedBlocksHeights[1:], removedBlocksHeights)
					removedBlocksHeights[0] = index

					if allTransactionsChanges, err = self.removeBlockComplete(writer, index, removedTxHashes, allTransactionsChanges, dataStorage); err != nil {
						return
					}

					if index > firstBlockComplete.Block.Height {
						index -= 1
					} else {
						break
					}
				}

				removedBlocksTransactionsCount = newChainData.TransactionsCount

				if firstBlockComplete.Block.Height == 0 {
					gui.GUI.Info("chain.createGenesisBlockchainData called")
					newChainData = self.createGenesisBlockchainData()
					removedBlocksTransactionsCount = 0
				} else {
					newChainData = &BlockchainData{}
					if err = newChainData.loadBlockchainInfo(writer, firstBlockComplete.Block.Height); err != nil {
						return
					}
				}

				if err = dataStorage.CommitChanges(); err != nil {
					return
				}

			}

			if blocksComplete[0].Block.Height != newChainData.Height {
				return errors.New("First block hash is not matching")
			}

			if !bytes.Equal(firstBlockComplete.Block.PrevHash, newChainData.Hash) {
				return fmt.Errorf("First block hash is not matching chain hash %d %s %s ", firstBlockComplete.Block.Height, base64.StdEncoding.EncodeToString(firstBlockComplete.Bloom.Hash), base64.StdEncoding.EncodeToString(newChainData.Hash))
			}

			if !bytes.Equal(firstBlockComplete.Block.PrevKernelHash, newChainData.KernelHash) {
				return errors.New("First block kernel hash is not matching chain prev kerneh lash")
			}

			err = func() (err error) {

				for _, blkComplete := range blocksComplete {

					//check block height
					if blkComplete.Block.Height != newChainData.Height {
						return errors.New("Block Height is not right!")
					}

					//check existance of a tx with payloads
					var foundStakingRewardTx *transaction.Transaction
					for index, tx := range blkComplete.Txs {
						if tx.Version == transaction_type.TX_ZETHER {
							txBase := tx.TransactionBaseInterface.(*transaction_zether.TransactionZether)
							if len(txBase.Payloads) == 2 && txBase.Payloads[0].PayloadScript == transaction_zether_payload_script.SCRIPT_STAKING && txBase.Payloads[1].PayloadScript == transaction_zether_payload_script.SCRIPT_STAKING_REWARD {
								if foundStakingRewardTx != nil {
									return errors.New("Multiple txs with staking & reward payloads")
								}
								foundStakingRewardTx = tx
								if index != len(blkComplete.Txs)-1 {
									return errors.New("Staking reward tx should be the last one")
								}
								continue
							}
							for _, payload := range txBase.Payloads {
								if payload.PayloadScript == transaction_zether_payload_script.SCRIPT_STAKING || payload.PayloadScript == transaction_zether_payload_script.SCRIPT_STAKING_REWARD {
									return errors.New("Block contains other staking/reward payloads")
								}
							}
						}
					}

					// not staking and reward tx
					if foundStakingRewardTx == nil {
						return errors.New("Block is missing Staking and Reward Transaction")
					}

					//check blkComplete balance
					foundStakingRewardTxBase := foundStakingRewardTx.TransactionBaseInterface.(*transaction_zether.TransactionZether)
					if foundStakingRewardTxBase.Payloads[0].BurnValue < config_stake.GetRequiredStake(blkComplete.Block.Height) {
						return errors.New("Staked amount is not enough!")
					}

					//verify staking amount
					if foundStakingRewardTxBase.Payloads[0].BurnValue != blkComplete.StakingAmount {
						return errors.New("Staked amount is different that the burn value")
					}

					if !bytes.Equal(foundStakingRewardTxBase.Payloads[0].Proof.Nonce(), blkComplete.StakingNonce) {
						return errors.New("Staked Proof Nonce is not matching with the one specified in the block")
					}

					//verify forger reward
					var reward, finalForgerReward uint64
					if reward, finalForgerReward, err = blockchain_types.ComputeBlockReward(blkComplete.Height, blkComplete.Txs); err != nil {
						return
					}

					if foundStakingRewardTxBase.Payloads[1].Extra.(*transaction_zether_payload_extra.TransactionZetherPayloadExtraStakingReward).Reward > finalForgerReward {
						return fmt.Errorf("Payload Reward %d is bigger than it should be %d", foundStakingRewardTxBase.Payloads[1].Extra.(*transaction_zether_payload_extra.TransactionZetherPayloadExtraStakingReward).Reward, finalForgerReward)
					}

					//increase supply
					var ast *asset.Asset
					if ast, err = dataStorage.Asts.Get(string(config_coins.NATIVE_ASSET_FULL)); err != nil {
						return
					}

					if err = ast.AddNativeSupply(true, reward); err != nil {
						return
					}
					if err = dataStorage.Asts.Update(string(config_coins.NATIVE_ASSET_FULL), ast); err != nil {
						return
					}

					newChainData.Supply = ast.Supply

					if difficulty.CheckKernelHashBig(blkComplete.Block.Bloom.KernelHashStaked, newChainData.Target) != true {
						return errors.New("KernelHash Difficulty is not met")
					}

					if !bytes.Equal(blkComplete.Block.PrevHash, newChainData.Hash) {
						return errors.New("PrevHash doesn't match Genesis prevHash")
					}

					if !bytes.Equal(blkComplete.Block.PrevKernelHash, newChainData.KernelHash) {
						return errors.New("PrevHash doesn't match Genesis prevKernelHash")
					}

					if blkComplete.Block.Timestamp < newChainData.Timestamp {
						return errors.New("Timestamp has to be greater than the last timestmap")
					}

					if blkComplete.Block.Timestamp > uint64(time.Now().UTC().Unix())+config.NETWORK_TIMESTAMP_DRIFT_MAX {
						return errors.New("Timestamp is too much into the future")
					}

					if err = blkComplete.IncludeBlockComplete(dataStorage); err != nil {
						return fmt.Errorf("Error including block %d into Blockchain: %s", blkComplete.Height, err.Error())
					}

					if err = dataStorage.ProcessPendingStakes(blkComplete.Height); err != nil {
						return errors.New("Error Processing Pending Stakes: " + err.Error())
					}

					if err = dataStorage.ProcessConditionalPayments(blkComplete.Height); err != nil {
						return errors.New("Error Processing Pending Future: " + err.Error())
					}

					//to detect if the savedBlock was done correctly
					savedBlock = false

					if allTransactionsChanges, err = self.saveBlockComplete(writer, blkComplete, newChainData.TransactionsCount, removedTxHashes, allTransactionsChanges, dataStorage); err != nil {
						return errors.New("Error saving block complete: " + err.Error())
					}

					if len(removedBlocksHeights) > 0 {
						removedBlocksHeights = removedBlocksHeights[1:]
					}

					newChainData.PrevHash = newChainData.Hash
					newChainData.Hash = blkComplete.Block.Bloom.Hash
					newChainData.PrevKernelHash = newChainData.KernelHash
					newChainData.KernelHash = blkComplete.Block.Bloom.KernelHash
					newChainData.Timestamp = blkComplete.Block.Timestamp

					difficultyBigInt := difficulty.ConvertTargetToDifficulty(newChainData.Target)
					newChainData.BigTotalDifficulty = new(big.Int).Add(newChainData.BigTotalDifficulty, difficultyBigInt)

					if newChainData.Target, err = newChainData.computeNextTargetBig(writer); err != nil {
						return
					}

					newChainData.Height += 1
					newChainData.TransactionsCount += uint64(len(blkComplete.Txs))
					insertedBlocks = append(insertedBlocks, blkComplete)

					newChainData.saveTotalDifficultyExtra(writer)

					newChainData.saveBlockchainHeight(writer)
					if err = newChainData.saveBlockchainInfo(writer); err != nil {
						return
					}

					savedBlock = true
				}

				return
			}()

			//recover, but in case the chain was correctly saved and the mewChainDifficulty is higher than
			//we should store it
			if savedBlock && chainData.BigTotalDifficulty.Cmp(newChainData.BigTotalDifficulty) < 0 {

				//let's recompute removedTxHashes
				removedTxHashes = make(map[string][]byte)
				for _, change := range allTransactionsChanges {
					if !change.Inserted {
						removedTxHashes[change.TxHashStr] = change.TxHash
					}
				}
				for _, change := range allTransactionsChanges {
					if change.Inserted {
						insertedTxs[change.TxHashStr] = change.Tx
						delete(removedTxHashes, change.TxHashStr)
					}
				}

				if calledByForging {
					newChainData.ConsecutiveSelfForged += 1
				} else {
					newChainData.ConsecutiveSelfForged = 0
				}

				if err = newChainData.saveBlockchain(writer); err != nil {
					return errors.New("Error saving Blockchain " + err.Error())
				}

				if len(removedBlocksHeights) > 0 {

					//remove unused blocks
					for _, removedBlock := range removedBlocksHeights {
						if err = self.deleteUnusedBlocksComplete(writer, removedBlock, dataStorage); err != nil {
							return
						}
					}

					//removing unused transactions
					if config.NODE_PROVIDE_EXTENDED_INFO_APP {
						removeUnusedTransactions(writer, newChainData.TransactionsCount, removedBlocksTransactionsCount)
					}
				}

				//let's keep the order as well
				var removedCount, insertedCount int
				for _, change := range allTransactionsChanges {
					if !change.Inserted && removedTxHashes[change.TxHashStr] != nil && insertedTxs[change.TxHashStr] == nil {
						removedCount += 1
					}
					if change.Inserted && insertedTxs[change.TxHashStr] != nil && removedTxHashes[change.TxHashStr] == nil {
						insertedCount += 1
					}
				}
				removedTxsList = make([][]byte, removedCount)
				insertedTxsList = make([]*transaction.Transaction, insertedCount)
				removedCount, insertedCount = 0, 0

				for _, change := range allTransactionsChanges {
					if !change.Inserted && removedTxHashes[change.TxHashStr] != nil && insertedTxs[change.TxHashStr] == nil {
						removedTxsList[removedCount] = writer.Get("tx:" + change.TxHashStr) //required because the garbage collector sometimes it deletes the underlying buffers
						writer.Delete("tx:" + change.TxHashStr)
						writer.Delete("txHash:" + change.TxHashStr)
						writer.Delete("txBlock:" + change.TxHashStr)
						removedCount += 1
					}
					if change.Inserted && insertedTxs[change.TxHashStr] != nil && removedTxHashes[change.TxHashStr] == nil {
						insertedTxsList[insertedCount] = change.Tx
						insertedCount += 1
					}
				}

				if config.NODE_PROVIDE_EXTENDED_INFO_APP {
					removeTxsInfo(writer, removedTxHashes)
				}

				if err = self.saveBlockchainHashmaps(dataStorage); err != nil {
					return
				}

				newChainData.AssetsCount = dataStorage.Asts.Count
				newChainData.AccountsCount = dataStorage.Regs.Count + dataStorage.PlainAccs.Count

			} else if err == nil { //only rollback
				err = errors.New("Rollback")
			}

			if dataStorage != nil {
				dataStorage.SetTx(nil)
			}

			return
		})

		return
	}()

	if err == nil && len(insertedBlocks) == 0 {
		err = errors.New("No blocks were inserted")
	}

	if err == nil && newChainData != nil {
		kernelHash = newChainData.KernelHash
		self.ChainData.Store(newChainData)
		mempool.Mempool.ContinueProcessingCn <- mempool.CONTINUE_PROCESSING_NO_ERROR
	} else {
		mempool.Mempool.ContinueProcessingCn <- mempool.CONTINUE_PROCESSING_ERROR
	}

	update := &blockchainUpdate{
		err:              err,
		calledByForging:  calledByForging,
		exceptSocketUUID: exceptSocketUUID,
	}

	if err == nil && newChainData != nil {
		update.newChainData = newChainData
		update.dataStorage = dataStorage
		update.removedTxsList = removedTxsList
		update.removedTxHashes = removedTxHashes
		update.insertedTxs = insertedTxs
		update.insertedTxsList = insertedTxsList
		update.insertedBlocks = insertedBlocks
		update.allTransactionsChanges = allTransactionsChanges

		gui.GUI.Warning("-------------------------------------------")
		gui.GUI.Warning(fmt.Sprintf("Included blocks %v - %d | TXs: %d | Hash %s", update.calledByForging, len(update.insertedBlocks), len(update.insertedTxs), base64.StdEncoding.EncodeToString(update.newChainData.Hash)))
		gui.GUI.Warning(update.newChainData.Height, base64.StdEncoding.EncodeToString(update.newChainData.Hash), update.newChainData.Target.Text(10), update.newChainData.BigTotalDifficulty.Text(10))
		gui.GUI.Warning("-------------------------------------------")
	}

	self.updatesQueue.updatesCn <- update

	return
}

func (self *blockchain) InitializeChain() (err error) {

	if err = self.loadBlockchain(); err != nil {
		if err.Error() != "Chain not found" {
			return
		}
		if _, err = self.init(); err != nil {
			return
		}
		if err = self.saveBlockchain(); err != nil {
			return
		}
	}

	chainData := self.GetChainData()
	chainData.updateChainInfo()

	return
}

func (self *blockchain) Close() {
	self.UpdateNewChainDataUpdate.CloseAll()
	self.UpdateNewChain.CloseAll()
	close(self.ForgingSolutionCn)
	close(self.NextBlockCreatedCn)
}

func Initialize() error {

	gui.GUI.Log("Blockchain init...")

	Blockchain = &blockchain{
		&generics.Value[*BlockchainData]{},
		blockchain_sync.CreateBlockchainSync(),
		&sync.Mutex{},
		createBlockchainUpdatesQueue(),
		make(chan *blockchain_types.BlockchainSolution),
		multicast.NewMulticastChannel[uint64](),
		multicast.NewMulticastChannel[*BlockchainDataUpdate](),
		multicast.NewMulticastChannel[*blockchain_types.BlockchainUpdates](),
		multicast.NewMulticastChannel[[]*blockchain_types.BlockchainTransactionUpdate](),
		multicast.NewMulticastChannel[*data_storage.DataStorage](),
		make(chan *forging_block_work.ForgingWork),
	}

	Blockchain.updatesQueue.chain = Blockchain
	Blockchain.updatesQueue.processBlockchainUpdatesQueue()
	Blockchain.updatesQueue.processBlockchainUpdateMempool()
	Blockchain.updatesQueue.processBlockchainUpdateNotifications()
	Blockchain.initBlockchainCLI()

	return nil
}
