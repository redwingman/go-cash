package api_common

import (
	"errors"
	"net/http"
	"pandora-pay/helpers/generics"
	"pandora-pay/wallet"
	"pandora-pay/wallet/wallet_address"
)

type APIWalletGetAccountsReply struct {
	Version   wallet.Version                  `json:"version" msgpack:"version"`
	Encrypted wallet.EncryptedVersion         `json:"encrypted" msgpack:"encrypted"`
	Addresses []*wallet_address.WalletAddress `json:"addresses" msgpack:"addresses"`
}

func (api *APICommon) GetWalletAddresses(r *http.Request, args *struct{}, reply *APIWalletGetAccountsReply, authenticated bool) (err error) {

	if !authenticated {
		return errors.New("Invalid User or Password")
	}

	wallet.Wallet.Lock.RLock()
	defer wallet.Wallet.Lock.RUnlock()

	reply.Version = wallet.Wallet.Version
	reply.Encrypted = wallet.Wallet.Encryption.Encrypted

	reply.Addresses = make([]*wallet_address.WalletAddress, len(wallet.Wallet.Addresses))
	for i, addr := range wallet.Wallet.Addresses {
		if reply.Addresses[i], err = generics.Clone[*wallet_address.WalletAddress](addr, new(wallet_address.WalletAddress)); err != nil {
			return
		}
	}

	return
}
