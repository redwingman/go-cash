package main

import (
	"os"
	"os/signal"
	"pandora-pay/address_balance_decrypter"
	"pandora-pay/builds/electron_helper/server"
	"pandora-pay/config"
	"pandora-pay/config/arguments"
	"pandora-pay/gui"
	"syscall"
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

	if err := address_balance_decrypter.Initialize(false); err != nil {
		panic(err)
	}

	if err = server.CreateServer(); err != nil {
		panic(err)
	}

	exitSignal := make(chan os.Signal, 10)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-exitSignal

}
