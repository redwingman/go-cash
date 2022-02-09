package webassembly

import (
	"encoding/hex"
	"pandora-pay/app"
	"pandora-pay/helpers"
	"pandora-pay/webassembly/webassembly_utils"
	"strconv"
	"syscall/js"
)

func getWallet(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {
		app.Wallet.Lock.RLock()
		defer app.Wallet.Lock.RUnlock()

		data, err := helpers.GetJSONDataExcept(app.Wallet, "mnemonic")
		if err != nil {
			return nil, err
		}

		return webassembly_utils.ConvertBytes(data), nil
	})
}

func exportWalletJSON(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {

		if err := app.Wallet.Encryption.CheckPassword(args[0].String(), false); err != nil {
			return nil, err
		}

		app.Wallet.Lock.RLock()
		defer app.Wallet.Lock.RUnlock()

		data, err := helpers.GetJSONDataExcept(app.Wallet)
		if err != nil {
			return nil, err
		}

		return webassembly_utils.ConvertBytes(data), nil
	})
}

func getWalletMnemonic(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {
		if err := app.Wallet.Encryption.CheckPassword(args[0].String(), false); err != nil {
			return nil, err
		}
		app.Wallet.Lock.RLock()
		defer app.Wallet.Lock.RUnlock()
		return app.Wallet.Mnemonic, nil
	})
}

func getWalletAddressPrivateKey(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {

		if err := app.Wallet.Encryption.CheckPassword(args[1].String(), false); err != nil {
			return nil, err
		}

		addr, err := app.Wallet.GetWalletAddressByPublicKeyHex(args[0].String(), true)
		if err != nil {
			return nil, err
		}

		return hex.EncodeToString(addr.PrivateKey.Key), nil
	})
}

func getWalletAddress(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {

		if err := app.Wallet.Encryption.CheckPassword(args[1].String(), false); err != nil {
			return nil, err
		}

		adr, err := app.Wallet.GetWalletAddressByPublicKeyHex(args[0].String(), true)
		if err != nil {
			return nil, err
		}

		return webassembly_utils.ConvertJSONBytes(adr)
	})
}

func addNewWalletAddress(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {

		if err := app.Wallet.Encryption.CheckPassword(args[0].String(), false); err != nil {
			return nil, err
		}

		adr, err := app.Wallet.AddNewAddress(false)
		if err != nil {
			return nil, err
		}
		return webassembly_utils.ConvertJSONBytes(adr)
	})
}

func removeWalletAddress(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {
		if err := app.Wallet.Encryption.CheckPassword(args[0].String(), false); err != nil {
			return nil, err
		}

		publicKey, err := hex.DecodeString(args[1].String())
		if err != nil {
			return nil, err
		}

		return app.Wallet.RemoveAddressByPublicKey(publicKey, true)
	})
}

func renameWalletAddress(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {
		if err := app.Wallet.Encryption.CheckPassword(args[0].String(), false); err != nil {
			return nil, err
		}

		publicKey, err := hex.DecodeString(args[1].String())
		if err != nil {
			return nil, err
		}

		return app.Wallet.RenameAddressByPublicKey(publicKey, args[2].String(), true)
	})
}

func importWalletPrivateKey(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {

		if err := app.Wallet.Encryption.CheckPassword(args[0].String(), false); err != nil {
			return nil, err
		}

		key, err := hex.DecodeString(args[1].String())
		if err != nil {
			return nil, err
		}
		adr, err := app.Wallet.ImportPrivateKey(args[2].String(), key)
		if err != nil {
			return nil, err
		}
		return webassembly_utils.ConvertJSONBytes(adr)
	})
}

func importWalletJSON(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {
		if err := app.Wallet.Encryption.CheckPassword(args[1].String(), false); err != nil {
			return nil, err
		}
		return true, app.Wallet.ImportWalletJSON([]byte(args[0].String()))
	})
}

func importWalletAddressJSON(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {
		if err := app.Wallet.Encryption.CheckPassword(args[2].String(), false); err != nil {
			return nil, err
		}
		adr, err := app.Wallet.ImportWalletAddressJSON([]byte(args[1].String()))
		if err != nil {
			return nil, err
		}
		return webassembly_utils.ConvertJSONBytes(adr)
	})
}

func checkPasswordWallet(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {
		if err := app.Wallet.Encryption.CheckPassword(args[0].String(), false); err != nil {
			return false, err
		}
		return true, nil
	})
}

func encryptWallet(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {
		if err := app.Wallet.Encryption.Encrypt(args[0].String(), args[1].Int()); err != nil {
			return nil, err
		}
		return true, nil
	})
}

func decryptWallet(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {
		if err := app.Wallet.Encryption.Decrypt(args[0].String()); err != nil {
			return nil, err
		}
		return true, nil
	})
}

func removeEncryptionWallet(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {
		if err := app.Wallet.Encryption.CheckPassword(args[0].String(), true); err != nil {
			return nil, err
		}
		if err := app.Wallet.Encryption.RemoveEncryption(); err != nil {
			return nil, err
		}
		return true, nil
	})
}

func logoutWallet(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {
		if err := app.Wallet.Encryption.Logout(); err != nil {
			return nil, err
		}
		return true, nil
	})
}

//signing not encrypting
func signMessageWalletAddress(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {
		if err := app.Wallet.Encryption.CheckPassword(args[2].String(), false); err != nil {
			return false, err
		}

		message, err := hex.DecodeString(args[0].String())
		if err != nil {
			return nil, err
		}

		addr, err := app.Wallet.GetWalletAddressByEncodedAddress(args[1].String(), true)
		if err != nil {
			return nil, err
		}

		out, err := addr.SignMessage(message)
		if err != nil {
			return nil, err
		}

		return hex.EncodeToString(out), nil
	})
}

func decryptMessageWalletAddress(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {
		if err := app.Wallet.Encryption.CheckPassword(args[2].String(), false); err != nil {
			return false, err
		}

		data, err := hex.DecodeString(args[0].String())
		if err != nil {
			return nil, err
		}

		addr, err := app.Wallet.GetWalletAddressByEncodedAddress(args[1].String(), true)
		if err != nil {
			return nil, err
		}

		out, err := addr.DecryptMessage(data)
		if err != nil {
			return nil, err
		}

		return hex.EncodeToString(out), nil
	})
}

func deriveDelegatedStakeWalletAddress(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {

		if err := app.Wallet.Encryption.CheckPassword(args[2].String(), false); err != nil {
			return false, err
		}

		nonce, err := strconv.ParseUint(args[0].String(), 10, 64)
		if err != nil {
			return nil, err
		}

		addr, err := app.Wallet.GetWalletAddressByEncodedAddress(args[1].String(), true)
		if err != nil {
			return nil, err
		}

		delegatedStake, err := addr.DeriveDelegatedStake(uint32(nonce))
		if err != nil {
			return nil, err
		}

		return webassembly_utils.ConvertJSONBytes(delegatedStake)

	})
}

func updatePreviousDecryptedBalanceValueWalletAddress(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {

		if err := app.Wallet.Encryption.CheckPassword(args[1].String(), false); err != nil {
			return false, err
		}

		parameters := &struct {
			PublicKey helpers.HexBytes `json:"publicKey"`
			Asset     helpers.HexBytes `json:"asset"`
			Amount    uint64           `json:"amount"`
			Balance   helpers.HexBytes `json:"balance"`
		}{}

		if err := webassembly_utils.UnmarshalBytes(args[0], parameters); err != nil {
			return nil, err
		}

		if err := app.Wallet.UpdatePreviousDecryptedBalanceValueByPublicKey(parameters.PublicKey, parameters.Amount, parameters.Balance, parameters.Asset); err != nil {
			return nil, err
		}

		return true, nil
	})
}

func tryDecryptBalance(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {

		if err := app.Wallet.Encryption.CheckPassword(args[1].String(), false); err != nil {
			return false, err
		}

		parameters := &struct {
			PublicKey helpers.HexBytes `json:"publicKey"`
			Asset     helpers.HexBytes `json:"asset"`
			Balance   helpers.HexBytes `json:"balance"`
		}{}

		if err := webassembly_utils.UnmarshalBytes(args[0], parameters); err != nil {
			return nil, err
		}

		value, decoded, err := app.Wallet.TryDecryptBalance(parameters.PublicKey, parameters.Asset, parameters.Balance)
		if err != nil {
			return nil, err
		}

		return webassembly_utils.ConvertJSONBytes(struct {
			Value   uint64 `json:"value"`
			Decoded bool   `json:"decoded"`
		}{value, decoded})
	})
}

func getPrivateDataForDecryptingBalanceWalletAddress(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {

		if err := app.Wallet.Encryption.CheckPassword(args[1].String(), false); err != nil {
			return false, err
		}

		parameters := &struct {
			PublicKey helpers.HexBytes `json:"publicKey"`
			Asset     helpers.HexBytes `json:"asset"`
		}{}

		if err := webassembly_utils.UnmarshalBytes(args[0], parameters); err != nil {
			return nil, err
		}

		privateKey, previousValue := app.Wallet.GetDataForDecryptingBalance(parameters.PublicKey, parameters.Asset)

		return webassembly_utils.ConvertJSONBytes(struct {
			PrivateKey    helpers.HexBytes `json:"privateKey"`
			PreviousValue uint64           `json:"previousValue"`
		}{privateKey, previousValue})

	})
}
