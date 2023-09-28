package main

import (
	"os"
	"os/signal"
	"pandora-pay/address_balance_decrypter"
	"pandora-pay/config"
	"pandora-pay/config/arguments"
	"pandora-pay/gui"
	"syscall"
	"syscall/js"
)

func main() {
	var err error

	argv := os.Args[1:]
	if err = arguments.InitArguments(argv); err != nil {
		panic(err)
	}
	if err = config.InitConfig(); err != nil {
		panic(err)
	}

	if err = gui.InitGUI(); err != nil {
		panic(err)
	}

	if err = address_balance_decrypter.Initialize(false); err != nil {
		panic(err)
	}

	js.Global().Set("PandoraPayHelper", js.ValueOf(map[string]interface{}{
		"helloPandoraHelper": js.FuncOf(helloPandoraHelper),
		"wallet": js.ValueOf(map[string]interface{}{
			"initializeBalanceDecrypter": js.FuncOf(initializeBalanceDecrypter),
			"decryptBalance":             js.FuncOf(decryptBalance),
		}),
		"transactions": js.ValueOf(map[string]interface{}{
			"builder": js.ValueOf(map[string]interface{}{
				"createZetherTx": js.FuncOf(createZetherTx),
			}),
		}),
	}))

	js.Global().Call("WASMLoaded")

	exitSignal := make(chan os.Signal, 10)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-exitSignal

}
