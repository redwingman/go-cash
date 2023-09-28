package routes

import (
	"context"
	"pandora-pay/address_balance_decrypter"
	"pandora-pay/builds/builds_data"
)

func RouteWalletDecryptBalance(req *builds_data.WalletDecryptBalanceReq) (any, error) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	value, err := address_balance_decrypter.Decrypter.DecryptBalance("wallet", req.PublicKey, req.PrivateKey, req.Balance, req.Asset, true, req.PreviousValue, true, ctx, func(status string) {})
	if err != nil {
		return nil, err
	}

	return value, nil
}
