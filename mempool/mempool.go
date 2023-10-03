package mempool

import (
	"context"
	"errors"
	"pandora-pay/blockchain/transactions/transaction"
	"pandora-pay/blockchain/transactions/transaction/transaction_simple"
	"pandora-pay/blockchain/transactions/transaction/transaction_type"
	"pandora-pay/blockchain/transactions/transaction/transaction_zether"
	"pandora-pay/blockchain/transactions/transaction/transaction_zether/transaction_zether_payload/transaction_zether_payload_script"
	"pandora-pay/config/config_fees"
	"pandora-pay/gui"
	"pandora-pay/helpers"
	"pandora-pay/helpers/generics"
	"pandora-pay/helpers/recovery"
	"pandora-pay/network/websocks/connection/advanced_connection_types"
	"pandora-pay/txs_validator"
	"runtime"
	"sync/atomic"
	"time"
)

type mempoolTx struct {
	Tx          *transaction.Transaction `json:"tx" msgpack:"tx"`
	Added       int64                    `json:"added" msgpack:"added"`
	Mine        bool                     `json:"mine" msgpack:"mine"`
	FeePerByte  uint64                   `json:"feePerByte" msgpack:"feePerByte"`
	ChainHeight uint64                   `json:"chainHeight" msgpack:"chainHeight"`
}

type mempool struct {
	result                    *generics.Value[*mempoolResult]
	SuspendProcessingCn       chan struct{}
	ContinueProcessingCn      chan ContinueProcessingType
	newWorkCn                 chan *mempoolWork
	addTransactionCn          chan *mempoolWorkerAddTx
	removeTransactionsCn      chan *mempoolWorkerRemoveTxs
	insertTransactionsCn      chan *mempoolWorkerInsertTxs
	Txs                       *mempoolTxs
	OnBroadcastNewTransaction func([]*transaction.Transaction, bool, bool, advanced_connection_types.UUID, context.Context) []error
	workers                   []*mempoolWorker
}

var Mempool *mempool

func (self *mempool) ContinueProcessing(continueProcessingType ContinueProcessingType) {
	self.ContinueProcessingCn <- continueProcessingType
}

func (self *mempool) RemoveInsertedTxsFromBlockchain(txs []string) bool {
	answerCn := make(chan bool)
	self.removeTransactionsCn <- &mempoolWorkerRemoveTxs{txs, answerCn}
	return <-answerCn
}

func (self *mempool) InsertRemovedTxsFromBlockchain(txs []*transaction.Transaction, height uint64) bool {

	finalTxs, _ := self.processTxsToMempool(txs, height, context.Background())

	insertTxs := make([]*mempoolTx, len(finalTxs))
	for i, it := range finalTxs {
		if it != nil {
			insertTxs[i] = it
		}
	}

	answerCn := make(chan bool)
	self.insertTransactionsCn <- &mempoolWorkerInsertTxs{insertTxs, answerCn}
	return <-answerCn

}

func (self *mempool) AddTxToMempool(tx *transaction.Transaction, height uint64, justCreated bool, awaitAnswer, awaitBroadcasting bool, exceptSocketUUID advanced_connection_types.UUID, ctx context.Context) error {
	result := self.AddTxsToMempool([]*transaction.Transaction{tx}, height, justCreated, awaitAnswer, awaitBroadcasting, exceptSocketUUID, ctx)
	return result[0]
}

func (self *mempool) processTxsToMempool(txs []*transaction.Transaction, height uint64, ctx context.Context) (finalTxs []*mempoolTx, errs []error) {

	finalTxs = make([]*mempoolTx, len(txs))
	errs = make([]error, len(txs))

	for i, tx := range txs {

		select {
		case <-ctx.Done():
			return
		default:
		}

		if tx.Version == transaction_type.TX_ZETHER {
			txBase := tx.TransactionBaseInterface.(*transaction_zether.TransactionZether)
			for _, payload := range txBase.Payloads {
				if payload.PayloadScript == transaction_zether_payload_script.SCRIPT_STAKING_REWARD || payload.PayloadScript == transaction_zether_payload_script.SCRIPT_STAKING {
					errs[i] = errors.New("Transaction is not accepted in the mempool")
					return
				}
			}
		}

		if errs[i] = txs_validator.TxsValidator.ValidateTx(tx); errs[i] != nil {
			return
		}

		if self.Txs.Exists(tx.Bloom.HashStr) {
			continue
		}

		checkFee := true

		minerFee, err := tx.GetAllFee()
		if err != nil {
			errs[i] = err
			continue
		}

		computedFeePerByte := minerFee
		if errs[i] = helpers.SafeUint64Sub(&computedFeePerByte, tx.SpaceExtra*config_fees.FEE_PER_BYTE_EXTRA_SPACE); errs[i] != nil {
			continue
		}

		computedFeePerByte = minerFee / tx.Bloom.Size

		requiredFeePerByte := uint64(0)
		switch tx.Version {
		case transaction_type.TX_SIMPLE:
			requiredFeePerByte = config_fees.FEE_PER_BYTE
			txBase := tx.TransactionBaseInterface.(*transaction_simple.TransactionSimple)
			if txBase.TxScript == transaction_simple.SCRIPT_RESOLUTION_CONDITIONAL_PAYMENT {
				checkFee = false
			}
		case transaction_type.TX_ZETHER:
			requiredFeePerByte = config_fees.FEE_PER_BYTE_ZETHER
		default:
			errs[i] = errors.New("Invalid Tx.Version")
			continue
		}

		if checkFee {
			if computedFeePerByte < requiredFeePerByte {
				errs[i] = errors.New("Transaction fee was not accepted")
				continue
			}
		}

		finalTxs[i] = &mempoolTx{
			Tx:          tx,
			Added:       time.Now().Unix(),
			FeePerByte:  computedFeePerByte,
			ChainHeight: height,
		}

	}

	return
}

func (self *mempool) AddTxsToMempool(txs []*transaction.Transaction, height uint64, justCreated, awaitAnswer, awaitBroadcasting bool, exceptSocketUUID advanced_connection_types.UUID, ctx context.Context) []error {

	finalTxs, errs := self.processTxsToMempool(txs, height, ctx)

	//making sure that the transaction is not inserted twice
	if runtime.GOARCH != "wasm" {
		for i, finalTx := range finalTxs {
			if finalTx != nil {

				select {
				case <-ctx.Done():
					return errs
				default:
				}

				var errorResult error

				if awaitAnswer {
					answerCn := make(chan error)
					self.addTransactionCn <- &mempoolWorkerAddTx{finalTx, answerCn}
					errorResult = <-answerCn
				} else {
					self.addTransactionCn <- &mempoolWorkerAddTx{finalTx, nil}
				}

				if errorResult != nil {
					errs[i] = errorResult
					finalTxs[i] = nil
				}

			}
		}
	}

	if exceptSocketUUID != advanced_connection_types.UUID_SKIP_ALL {

		broadcastTxs := make([]*transaction.Transaction, 0)
		for _, finalTx := range finalTxs {
			if finalTx != nil {
				broadcastTxs = append(broadcastTxs, finalTx.Tx)
			}
		}

		errors2 := self.OnBroadcastNewTransaction(broadcastTxs, justCreated, awaitBroadcasting, exceptSocketUUID, ctx)
		for i, err := range errors2 {
			if err != nil {
				errs[i] = err
				finalTxs[i] = nil
			}
		}

	}

	return errs
}

// reset the forger
func (self *mempool) UpdateWork(hash []byte, height uint64) {

	result := &mempoolResult{
		txs:         &generics.Value[[]*mempoolTx]{}, //, appendOnly
		totalSize:   0,
		chainHash:   hash,
		chainHeight: height,
	}
	result.txs.Store([]*mempoolTx{})
	self.result.Store(result)

	newWork := &mempoolWork{
		chainHash:   hash,
		chainHeight: height,
		result:      result,
	}

	self.newWorkCn <- newWork
}

func (self *mempool) ContinueWork() {
	newWork := &mempoolWork{}
	self.newWorkCn <- newWork
}

func (self *mempool) Close() {

	for _, worker := range self.workers {
		worker.closed.Store(true)
	}

	close(self.SuspendProcessingCn)
}

func Initialize() (err error) {

	gui.GUI.Log("Mempool init...")

	Mempool = &mempool{
		&generics.Value[*mempoolResult]{},
		make(chan struct{}),
		make(chan ContinueProcessingType),
		make(chan *mempoolWork),
		make(chan *mempoolWorkerAddTx, 1000),
		make(chan *mempoolWorkerRemoveTxs),
		make(chan *mempoolWorkerInsertTxs),
		createMempoolTxs(),
		nil,
		[]*mempoolWorker{},
	}

	worker := &mempoolWorker{
		nil,
		&atomic.Bool{},
	}
	worker.closed.Store(false)
	Mempool.workers = append(Mempool.workers, worker)

	recovery.SafeGo(func() {
		worker.processing(Mempool.newWorkCn, Mempool.SuspendProcessingCn, Mempool.ContinueProcessingCn, Mempool.addTransactionCn, Mempool.insertTransactionsCn, Mempool.removeTransactionsCn, Mempool.Txs)
	})

	Mempool.initCLI()

	return nil

}
