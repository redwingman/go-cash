package start

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/signal"
	"pandora-pay/address_balance_decrypter"
	"pandora-pay/app"
	"pandora-pay/blockchain"
	"pandora-pay/blockchain/forging"
	"pandora-pay/blockchain/genesis"
	"pandora-pay/chain_network"
	"pandora-pay/config"
	"pandora-pay/config/arguments"
	"pandora-pay/config/config_forging"
	"pandora-pay/config/globals"
	"pandora-pay/cryptography/crypto/balance_decrypter"
	"pandora-pay/gui"
	"pandora-pay/helpers/debugging_pprof"
	"pandora-pay/mempool"
	"pandora-pay/network"
	"pandora-pay/network/network_config"
	"pandora-pay/settings"
	"pandora-pay/store"
	"pandora-pay/testnet"
	"pandora-pay/txs_builder"
	"pandora-pay/txs_validator"
	"pandora-pay/wallet"
	"runtime"
	"strconv"
	"syscall"
)

func StartMainNow() (err error) {

	if !globals.MainStarted.CompareAndSwap(false, true) {
		return
	}

	arguments.VERSION_STRING = config.VERSION_STRING
	if arguments.Arguments["--pprof"] == true {
		if err = debugging_pprof.Start(); err != nil {
			return
		}
	}

	if err = gui.InitGUI(); err != nil {
		return
	}
	globals.MainEvents.BroadcastEvent("main", "GUI initialized")

	if err = store.InitDB(); err != nil {
		return
	}
	globals.MainEvents.BroadcastEvent("main", "database initialized")

	if err = txs_validator.NewTxsValidator(); err != nil {
		return
	}
	globals.MainEvents.BroadcastEvent("main", "txs validator initialized")

	if err = address_balance_decrypter.Initialize(runtime.GOARCH != "wasm"); err != nil {
		return
	}
	globals.MainEvents.BroadcastEvent("main", "address balance decrypter validator initialized")

	if err = mempool.Initialize(); err != nil {
		return
	}
	globals.MainEvents.BroadcastEvent("main", "mempool initialized")

	if err = forging.Initialize(); err != nil {
		return
	}
	globals.MainEvents.BroadcastEvent("main", "forging initialized")

	if err = blockchain.Initialize(); err != nil {
		return
	}
	globals.MainEvents.BroadcastEvent("main", "blockchain initialized")

	if err = wallet.Initialize(); err != nil {
		return
	}
	if err = wallet.Wallet.ProcessWalletArguments(); err != nil {
		return
	}

	globals.MainEvents.BroadcastEvent("main", "wallet initialized")

	if err = genesis.GenesisInit(wallet.Wallet.GetFirstAddressForDevnetGenesisAirdrop); err != nil {
		return
	}
	if err = blockchain.Blockchain.InitializeChain(); err != nil {
		return
	}

	if runtime.GOARCH != "wasm" && arguments.Arguments["--balance-decrypter-disable-init"] == false {
		tableSize := 0
		if arguments.Arguments["--balance-decrypter-table-size"] != nil {
			if tableSize, err = strconv.Atoi(arguments.Arguments["--balance-decrypter-table-size"].(string)); err != nil {
				return
			}
			tableSize = 1 << tableSize
		}
		go func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			gui.GUI.Info2Update("Decrypter", "Init... "+strconv.Itoa(int(math.Log2(float64(tableSize)))))
			balance_decrypter.BalanceDecrypter.SetTableSize(tableSize, ctx, func(string) {})
			gui.GUI.Info2Update("Decrypter", "Ready "+strconv.Itoa(int(math.Log2(float64(tableSize)))))
		}()
	}

	wallet.Wallet.InitializeWallet(blockchain.Blockchain.UpdateNewChainUpdate)
	if err = wallet.Wallet.StartWallet(); err != nil {
		return
	}

	if err = settings.Initialize(); err != nil {
		return
	}
	globals.MainEvents.BroadcastEvent("main", "settings initialized")

	if err = txs_builder.Initialize(); err != nil {
		return
	}
	globals.MainEvents.BroadcastEvent("main", "transactions builder initialized")

	forging.Forging.InitializeForging(txs_builder.TxsBuilder.CreateForgingTransactions, blockchain.Blockchain.NextBlockCreatedCn, blockchain.Blockchain.UpdateNewChainUpdate, blockchain.Blockchain.ForgingSolutionCn)

	if config_forging.FORGING_ENABLED {
		forging.Forging.StartForging()
	}

	blockchain.Blockchain.InitForging()

	if arguments.Arguments["--exit"] == true {
		os.Exit(1)
		return
	}

	if arguments.Arguments["--run-testnet-script"] == true {
		if err = testnet.Initialize(); err != nil {
			return
		}
	}

	if err = network.NewNetwork(); err != nil {
		return
	}
	globals.MainEvents.BroadcastEvent("main", "network initialized")

	chain_network.Initialize()

	gui.GUI.Log("Main Loop")
	globals.MainEvents.BroadcastEvent("main", "initialized")

	return
}

func InitMain(ready func()) {
	var err error

	argv := os.Args[1:]
	if err = arguments.InitArguments(argv); err != nil {
		saveError(err)
	}
	globals.MainEvents.BroadcastEvent("main", "arguments initialized")

	if err = config.InitConfig(); err != nil {
		saveError(err)
	}
	globals.MainEvents.BroadcastEvent("main", "config initialized")
	if err = network_config.InitConfig(); err != nil {
		return
	}

	startMain()

	if ready != nil {
		ready()
	}

	exitSignal := make(chan os.Signal, 10)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-exitSignal

	fmt.Println("Shutting down")
	app.Close()
}
