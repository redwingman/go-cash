package forging

import (
	"bytes"
	"fmt"
	"pandora-pay/blockchain/blockchain_types"
	"pandora-pay/blockchain/blocks/block_complete"
	"pandora-pay/blockchain/forging/forging_block_work"
	"pandora-pay/blockchain/transactions/transaction"
	"pandora-pay/gui"
	"pandora-pay/helpers/advanced_buffers"
	"pandora-pay/helpers/generics"
	"pandora-pay/helpers/recovery"
	"pandora-pay/mempool"
	"strconv"
	"sync/atomic"
	"time"
)

type forgingThread struct {
	threads                   int                                         //number of threads
	solutionCn                chan<- *blockchain_types.BlockchainSolution //broadcasting that a solution thread was received
	nextBlockCreatedCn        <-chan *forging_block_work.ForgingWork      //detect if a new work was published
	workers                   []*ForgingWorkerThread
	workersCreatedCn          chan []*ForgingWorkerThread
	workersDestroyedCn        chan struct{}
	lastPrevKernelHash        *generics.Value[[]byte]
	createForgingTransactions func(*block_complete.BlockComplete, []byte, uint64, []*transaction.Transaction) (*transaction.Transaction, error)
}

func (self *forgingThread) stopForging() {
	self.workersDestroyedCn <- struct{}{}
	for i := 0; i < len(self.workers); i++ {
		close(self.workers[i].workCn)
	}
}

func (self *forgingThread) startForging() {

	self.workers = make([]*ForgingWorkerThread, self.threads)

	forgingWorkerSolutionCn := make(chan *ForgingSolution)
	for i := 0; i < len(self.workers); i++ {
		self.workers[i] = createForgingWorkerThread(i, forgingWorkerSolutionCn)
		recovery.SafeGo(self.workers[i].forge)
	}
	self.workersCreatedCn <- self.workers

	recovery.SafeGo(func() {
		for {

			s := ""
			for i := 0; i < self.threads; i++ {
				hashesPerSecond := atomic.SwapUint32(&self.workers[i].hashes, 0)
				s += strconv.FormatUint(uint64(hashesPerSecond), 10) + " "
			}
			gui.GUI.InfoUpdate("Hashes/s", s)

			time.Sleep(time.Second)
		}
	})

	recovery.SafeGo(func() {
		var err error
		var newKernelHash []byte

		for {
			solution, ok := <-forgingWorkerSolutionCn
			if !ok {
				return
			}

			lastPrevKernelHash := self.lastPrevKernelHash.Load()
			if lastPrevKernelHash != nil && solution.blkComplete.Height > 1 && !bytes.Equal(solution.blkComplete.PrevKernelHash, lastPrevKernelHash) {
				continue
			}

			if newKernelHash, err = self.publishSolution(solution); err != nil {
				gui.GUI.Error(fmt.Errorf("Error publishing solution: %d error: %s ", solution.blkComplete.Height, err))
			} else {
				gui.GUI.Info(fmt.Errorf("Block was forged! %d ", solution.blkComplete.Height))
				self.lastPrevKernelHash.Store(newKernelHash)
			}

		}
	})

	recovery.SafeGo(func() {
		for {
			newWork, ok := <-self.nextBlockCreatedCn
			if !ok {
				return
			}

			self.lastPrevKernelHash.Store(newWork.BlkComplete.PrevKernelHash)

			for i := 0; i < self.threads; i++ {
				self.workers[i].workCn <- newWork
			}

			gui.GUI.InfoUpdate("Hash Block", strconv.FormatUint(newWork.BlkHeight, 10))
		}
	})

}

func (self *forgingThread) publishSolution(solution *ForgingSolution) ([]byte, error) {

	newBlk := block_complete.CreateEmptyBlockComplete()
	if err := newBlk.Deserialize(advanced_buffers.NewBufferReader(solution.blkComplete.SerializeToBytes())); err != nil {
		return nil, err
	}

	newBlk.Block.StakingNonce = solution.stakingNonce
	newBlk.Block.Timestamp = solution.timestamp
	newBlk.Block.StakingAmount = solution.stakingAmount

	txs, _ := mempool.Mempool.GetNextTransactionsToInclude(newBlk.Block.PrevHash)

	txStakingReward, err := self.createForgingTransactions(newBlk, solution.publicKey, solution.decryptedStakingBalance, txs)
	if err != nil {
		return nil, err
	}

	newBlk.Txs = append(txs, txStakingReward)

	newBlk.Block.MerkleHash = newBlk.MerkleHash()

	newBlk.Bloom = nil
	if err = newBlk.BloomAll(); err != nil {
		return nil, err
	}

	//send message to blockchain
	result := make(chan *blockchain_types.BlockchainSolutionAnswer)
	self.solutionCn <- &blockchain_types.BlockchainSolution{
		newBlk,
		result,
	}

	res := <-result
	return res.ChainKernelHash, res.Err
}

func createForgingThread(threads int, createForgingTransactions func(*block_complete.BlockComplete, []byte, uint64, []*transaction.Transaction) (*transaction.Transaction, error), solutionCn chan<- *blockchain_types.BlockchainSolution, nextBlockCreatedCn <-chan *forging_block_work.ForgingWork) *forgingThread {
	return &forgingThread{
		threads,
		solutionCn,
		nextBlockCreatedCn,
		[]*ForgingWorkerThread{},
		make(chan []*ForgingWorkerThread),
		make(chan struct{}),
		&generics.Value[[]byte]{},
		createForgingTransactions,
	}
}
