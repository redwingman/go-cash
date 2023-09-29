package app

import (
	"pandora-pay/blockchain"
	"pandora-pay/blockchain/forging"
	"pandora-pay/gui"
	"pandora-pay/store"
	"pandora-pay/wallet"
)

func Close() {
	store.DBClose()
	gui.GUI.Close()
	forging.Forging.Close()
	blockchain.Blockchain.Close()
	wallet.Wallet.Close()
}
