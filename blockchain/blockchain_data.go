package blockchain

import (
	"encoding/base64"
	"errors"
	"math/big"
	"pandora-pay/blockchain/blocks/block/difficulty"
	"pandora-pay/config"
	"pandora-pay/gui"
	"pandora-pay/helpers"
	"pandora-pay/store/store_db/store_db_interface"
	"strconv"
)

type BlockchainData struct {
	Hash                  []byte   `json:"hash" msgpack:"hash"`                     //32
	PrevHash              []byte   `json:"prevHash" msgpack:"prevHash"`             //32
	KernelHash            []byte   `json:"kernelHash" msgpack:"kernelHash"`         //32
	PrevKernelHash        []byte   `json:"prevKernelHash" msgpack:"prevKernelHash"` //32
	Height                uint64   `json:"height" msgpack:"height"`
	Timestamp             uint64   `json:"timestamp" msgpack:"timestamp"`
	Target                *big.Int `json:"target" msgpack:"target"`
	BigTotalDifficulty    *big.Int `json:"bigTotalDifficulty" msgpack:"bigTotalDifficulty"`
	TransactionsCount     uint64   `json:"transactionsCount" msgpack:"transactionsCount"` //count of the number of txs
	AccountsCount         uint64   `json:"accountsCount" msgpack:"accountsCount"`         //count of the number of assets
	AssetsCount           uint64   `json:"assetsCount" msgpack:"assetsCount"`             //count of the number of assets
	Supply                uint64   `json:"supply" msgpack:"supply"`
	ConsecutiveSelfForged uint64   `json:"consecutiveSelfForged" msgpack:"consecutiveSelfForged"`
}

func (self *BlockchainData) computeNextTargetBig(reader store_db_interface.StoreDBTransactionInterface) (*big.Int, error) {

	if config.DIFFICULTY_BLOCK_WINDOW > self.Height {
		return self.Target, nil
	}

	first := self.Height - config.DIFFICULTY_BLOCK_WINDOW

	firstDifficulty, firstTimestamp, err := self.LoadTotalDifficultyExtra(reader, first+1)
	if err != nil {
		return nil, err
	}

	lastDifficulty := self.BigTotalDifficulty
	lastTimestamp := self.Timestamp

	deltaTotalDifficulty := new(big.Int).Sub(lastDifficulty, firstDifficulty)
	deltaTime := lastTimestamp - firstTimestamp

	//gui.Log("lastDifficulty", lastDifficulty.String(), "chainData.Height", chainData.Height, "chainData.Timestamp", chainData.Timestamp, "chainData.BigTotalDifficulty", chainData.BigTotalDifficulty.String())
	if deltaTotalDifficulty.Cmp(config.BIG_INT_ZERO) == 0 {
		return nil, errors.New("Delta Difficulty is zero")
	}

	return difficulty.NextTargetBig(deltaTotalDifficulty, deltaTime)
}

func (self *BlockchainData) updateChainInfo() {
	gui.GUI.Info2Update("Blocks", strconv.FormatUint(self.Height, 10))
	gui.GUI.Info2Update("Chain  Hash", base64.StdEncoding.EncodeToString(self.Hash))
	gui.GUI.Info2Update("Chain KHash", base64.StdEncoding.EncodeToString(self.KernelHash))
	gui.GUI.Info2Update("TXs", strconv.FormatUint(self.TransactionsCount, 10))
}

func (self *BlockchainData) clone() *BlockchainData {
	return &BlockchainData{
		helpers.CloneBytes(self.Hash),             //atomic copy
		helpers.CloneBytes(self.PrevHash),         //atomic copy
		helpers.CloneBytes(self.KernelHash),       //atomic copy
		helpers.CloneBytes(self.PrevKernelHash),   //atomic copy
		self.Height,                               //atomic copy
		self.Timestamp,                            //atomic copy
		new(big.Int).Set(self.Target),             //atomic copy
		new(big.Int).Set(self.BigTotalDifficulty), //atomic copy
		self.TransactionsCount,                    //atomic copy
		self.AccountsCount,                        //atomic copy
		self.AssetsCount,                          //atomic copy
		self.Supply,
		self.ConsecutiveSelfForged, //atomic copy
	}
}
