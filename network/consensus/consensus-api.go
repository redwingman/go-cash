package consensus

import (
	"encoding/json"
	"github.com/tevino/abool"
	"pandora-pay/blockchain"
	block_complete "pandora-pay/blockchain/block-complete"
	"pandora-pay/network/websocks/connection"
	"sync/atomic"
)

func (consensus *Consensus) chainGet(conn *connection.AdvancedConnection, values []byte) ([]byte, error) {
	conn.SendJSON([]byte("chain"), consensus.getUpdateNotification(nil))
	return nil, nil
}

func (consensus *Consensus) chainUpdate(conn *connection.AdvancedConnection, values []byte) (out []byte, err error) {

	chainUpdateNotification := new(ChainUpdateNotification)
	if err := json.Unmarshal(values, &chainUpdateNotification); err != nil {
		return nil, err
	}

	forkFound, exists := consensus.forks.hashes.Load(string(chainUpdateNotification.Hash))
	if exists {
		fork := forkFound.(*Fork)
		fork.AddConn(conn, false)
		return
	}

	chainLastUpdate := consensus.chain.GetChainData()

	if chainLastUpdate.BigTotalDifficulty.Cmp(chainUpdateNotification.BigTotalDifficulty) < 0 {

		_, exists := consensus.forks.hashes.Load(string(chainUpdateNotification.Hash))
		if exists {
			return
		} //already found

		found, exists := consensus.forks.hashes.Load(string(chainUpdateNotification.PrevHash))
		if exists {
			prevFork := (found).(*Fork)

			_, exists = consensus.forks.hashes.LoadOrStore(string(chainUpdateNotification.Hash), prevFork)
			if exists {
				return
			} //meanwhile it was found

			if prevFork.readyForDownloading.IsNotSet() {

				prevFork.Lock()
				defer prevFork.Unlock()

				atomic.StoreUint64(&prevFork.end, chainUpdateNotification.End)

				if prevFork.readyForDownloading.IsNotSet() {
					prevFork.current = chainUpdateNotification.End
					prevFork.start = chainUpdateNotification.End
					prevFork.hashes = append(prevFork.hashes, chainUpdateNotification.Hash)
					prevFork.prevHash = chainUpdateNotification.PrevHash
					prevFork.bigTotalDifficulty = chainUpdateNotification.BigTotalDifficulty
					prevFork.AddConn(conn, true)
				}

				prevFork.Unlock()
			}

			return
		}

		fork := &Fork{
			start:               chainUpdateNotification.End,
			end:                 chainUpdateNotification.End,
			current:             chainUpdateNotification.End,
			hashes:              [][]byte{chainUpdateNotification.Hash},
			prevHash:            chainUpdateNotification.PrevHash,
			bigTotalDifficulty:  chainUpdateNotification.BigTotalDifficulty,
			readyForDownloading: abool.New(),
			blocks:              make([]*block_complete.BlockComplete, 0),
			conns:               []*connection.AdvancedConnection{conn},
		}
		_, exists = consensus.forks.hashes.LoadOrStore(string(chainUpdateNotification.Hash), fork)
		if !exists {
			consensus.forks.listMutex.Lock()
			list := consensus.forks.list.Load().([]*Fork)
			list = append(list, fork)
			consensus.forks.list.Store(list)
			consensus.forks.listMutex.Unlock()
		}

	} else {
		//let's notify him tha we have a better chain
		conn.SendJSON([]byte("chain"), consensus.getUpdateNotification(nil))
	}

	return nil, nil
}

func (consensus *Consensus) broadcast(newChainData *blockchain.BlockchainData) {
	consensus.httpServer.Websockets.BroadcastJSON([]byte("chain"), consensus.getUpdateNotification(newChainData))
}

func (consensus *Consensus) getUpdateNotification(newChainData *blockchain.BlockchainData) *ChainUpdateNotification {
	if newChainData == nil {
		newChainData = consensus.chain.GetChainData()
	}
	return &ChainUpdateNotification{
		End:                newChainData.Height,
		Hash:               newChainData.Hash,
		PrevHash:           newChainData.PrevHash,
		BigTotalDifficulty: newChainData.BigTotalDifficulty,
	}
}
