package webassembly

import (
	"encoding/hex"
	"encoding/json"
	"pandora-pay/app"
	"pandora-pay/blockchain/data_storage"
	"pandora-pay/blockchain/data_storage/accounts"
	"pandora-pay/blockchain/data_storage/tokens"
	"pandora-pay/blockchain/data_storage/tokens/token"
	"pandora-pay/mempool"
	"pandora-pay/network/api/api_common/api_types"
	"pandora-pay/store"
	"pandora-pay/store/store_db/store_db_interface"
	"pandora-pay/webassembly/webassembly_utils"
	"syscall/js"
)

func storeAccount(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {

		var err error

		publicKey, err := hex.DecodeString(args[0].String())
		if err != nil {
			return nil, err
		}

		var apiAcc *api_types.APIAccount
		if !args[1].IsNull() {
			apiAcc = &api_types.APIAccount{}
			data := make([]byte, args[1].Get("byteLength").Int())
			js.CopyBytesToGo(data, args[1])
			if err = json.Unmarshal(data, apiAcc); err != nil {
				return nil, err
			}
		}

		mutex.Lock()
		defer mutex.Unlock()

		app.Mempool.SuspendProcessingCn <- struct{}{}
		defer app.Mempool.ContinueProcessing(mempool.CONTINUE_PROCESSING_NO_ERROR_RESET)

		if err = store.StoreBlockchain.DB.Update(func(writer store_db_interface.StoreDBTransactionInterface) (err error) {

			dataStorage := data_storage.CreateDataStorage(writer)

			var accs *accounts.Accounts

			var tokensList [][]byte
			if tokensList, err = dataStorage.AccsCollection.GetAccountTokens(publicKey); err != nil {
				return
			}

			for _, token := range tokensList {
				if accs, err = dataStorage.AccsCollection.GetMap(token); err != nil {
					return
				}
				accs.Delete(string(publicKey))
			}

			dataStorage.PlainAccs.Delete(string(publicKey))
			dataStorage.Regs.Delete(string(publicKey))

			if apiAcc != nil {

				for i, token := range apiAcc.Tokens {
					if accs, err = dataStorage.AccsCollection.GetMap(token); err != nil {
						return
					}
					apiAcc.Accs[i].PublicKey = publicKey
					apiAcc.Accs[i].Token = token
					if err = accs.Update(string(publicKey), apiAcc.Accs[i]); err != nil {
						return
					}
				}

				if apiAcc.PlainAcc != nil {
					apiAcc.PlainAcc.PublicKey = publicKey
					if err = dataStorage.PlainAccs.Update(string(publicKey), apiAcc.PlainAcc); err != nil {
						return
					}
				}

				if apiAcc.Reg != nil {
					apiAcc.Reg.PublicKey = publicKey
					if err = dataStorage.Regs.Update(string(publicKey), apiAcc.Reg); err != nil {
						return
					}
				}

			}

			return dataStorage.CommitChanges()
		}); err != nil {
			return nil, err
		}

		return true, nil
	})
}

func storeToken(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {

		var err error

		var tok *token.Token
		if !args[1].IsNull() {
			tok = &token.Token{}
			data := make([]byte, args[1].Get("byteLength").Int())
			js.CopyBytesToGo(data, args[1])
			if err = json.Unmarshal(data, &tok); err != nil {
				return nil, err
			}
		}

		hash, err := hex.DecodeString(args[0].String())
		if err != nil {
			return nil, err
		}

		mutex.Lock()
		defer mutex.Unlock()

		app.Mempool.SuspendProcessingCn <- struct{}{}
		defer app.Mempool.ContinueProcessing(mempool.CONTINUE_PROCESSING_NO_ERROR_RESET)

		if err = store.StoreBlockchain.DB.Update(func(writer store_db_interface.StoreDBTransactionInterface) (err error) {

			toks := tokens.NewTokens(writer)
			if tok == nil {
				toks.DeleteToken(hash)
			} else {
				if err = toks.UpdateToken(hash, tok); err != nil {
					return
				}
			}
			return toks.CommitChanges()
		}); err != nil {
			return nil, err
		}

		return true, nil
	})
}