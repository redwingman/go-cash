package app

import (
	"pandora-pay/blockchain"
	"pandora-pay/blockchain/forging"
	"pandora-pay/gui"
	"pandora-pay/mempool"
	"pandora-pay/settings"
	"pandora-pay/store"
	"pandora-pay/wallet"
)

var (
	Settings *settings.Settings
	Wallet   *wallet.Wallet
	Forging  *forging.Forging
	Mempool  *mempool.Mempool
	Chain    *blockchain.Blockchain
)

func Close() {
	store.DBClose()
	gui.GUI.Close()
	Forging.Close()
	Chain.Close()
	Wallet.Close()
}
