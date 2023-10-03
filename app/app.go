package app

import (
	"pandora-pay/blockchain"
	"pandora-pay/blockchain/forging"
	"pandora-pay/gui"
	"pandora-pay/mempool"
	"pandora-pay/store"
	"pandora-pay/wallet"
)

func Close() {

	closedCn := make(chan struct{})
	go func() {

		defer func() {
			close(closedCn)
		}()

		mempool.Mempool.Close()

		forging.Forging.Close()
		blockchain.Blockchain.Close()
		wallet.Wallet.Close()

		if err := store.DBClose(); err != nil {
			gui.GUI.Error("Error closing DB", err)
		}

		gui.GUI.Close()

	}()

	<-closedCn
}
