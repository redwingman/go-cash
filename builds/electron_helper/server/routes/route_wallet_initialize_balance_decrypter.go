package routes

import (
	"context"
	"pandora-pay/builds/builds_data"
	"pandora-pay/cryptography/crypto/balance_decrypter"
)

func RouteWalletInitializeBalanceDecrypter(req *builds_data.WalletInitializeBalanceDecrypterReq) (any, error) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	balance_decrypter.BalanceDecrypter.SetTableSize(req.TableSize, ctx, func(status string) {})

	return true, nil
}
