package txs_builder

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"math"
	"math/rand"
	"pandora-pay/addresses"
	"pandora-pay/blockchain/data_storage"
	"pandora-pay/blockchain/data_storage/accounts"
	"pandora-pay/blockchain/data_storage/accounts/account"
	"pandora-pay/blockchain/data_storage/plain_accounts/plain_account/asset_fee_liquidity"
	"pandora-pay/blockchain/data_storage/registrations/registration"
	"pandora-pay/blockchain/transactions/transaction"
	"pandora-pay/config/config_coins"
	"pandora-pay/config/globals"
	"pandora-pay/cryptography/bn256"
	"pandora-pay/cryptography/crypto"
	"pandora-pay/network/websocks/connection/advanced_connection_types"
	"pandora-pay/store"
	"pandora-pay/store/store_db/store_db_interface"
	"pandora-pay/txs_builder/wizard"
	"pandora-pay/wallet/wallet_address"
)

func (builder *TxsBuilder) getRandomAccount(accs *accounts.Accounts) (addr *addresses.Address, err error) {

	var acc *account.Account

	if acc, err = accs.GetRandomAccount(); err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, errors.New("Error getting any random account")
	}

	if addr, err = addresses.CreateAddr(acc.PublicKey, nil, nil, 0, nil); err != nil {
		return nil, err
	}

	return
}

func (builder *TxsBuilder) createZetherRing(sender string, recipient *string, assetId []byte, ringConfiguration *ZetherRingConfiguration, dataStorage *data_storage.DataStorage) ([]string, error) {

	var addr *addresses.Address
	var err error

	if ringConfiguration.RingSize == -1 {
		pow := rand.Intn(5) + 4
		ringConfiguration.RingSize = int(math.Pow(2, float64(pow)))
	}
	if ringConfiguration.NewAccounts == -1 {
		ringConfiguration.NewAccounts = rand.Intn(ringConfiguration.RingSize / 5)
	}

	if ringConfiguration.RingSize < 0 {
		return nil, errors.New("number is negative")
	}
	if !crypto.IsPowerOf2(ringConfiguration.RingSize) {
		return nil, errors.New("ring size is not a power of 2")
	}
	if ringConfiguration.NewAccounts < 0 || ringConfiguration.NewAccounts > ringConfiguration.RingSize-2 {
		return nil, errors.New("New accounts needs to be in the interval [0, ringSize-2] ")
	}

	var accs *accounts.Accounts
	if accs, err = dataStorage.AccsCollection.GetMap(assetId); err != nil {
		return nil, err
	}

	alreadyUsed := make(map[string]bool)

	if addr, err = addresses.DecodeAddr(sender); err != nil {
		return nil, err
	}
	alreadyUsed[string(addr.PublicKey)] = true

	if *recipient == "" {
		if addr, err = builder.getRandomAccount(accs); err != nil {
			return nil, err
		}
		*recipient = addr.EncodeAddr()
	}

	if addr, err = addresses.DecodeAddr(*recipient); err != nil {
		return nil, err
	}
	alreadyUsed[string(addr.PublicKey)] = true

	rings := make([]string, ringConfiguration.RingSize-2)

	if globals.Arguments["--new-devnet"] == true && accs.Count < 80000 {
		ringConfiguration.NewAccounts = ringConfiguration.RingSize - 2
	}

	for i := 0; i < ringConfiguration.RingSize-2; i++ {

		if i < ringConfiguration.NewAccounts || accs.Count-2+uint64(ringConfiguration.NewAccounts) <= uint64(i) {
			priv := addresses.GenerateNewPrivateKey()
			if addr, err = priv.GenerateAddress(true, nil, 0, nil); err != nil {
				return nil, err
			}
		} else {
			if addr, err = builder.getRandomAccount(accs); err != nil {
				return nil, err
			}
		}

		if alreadyUsed[string(addr.PublicKey)] {
			i--
			continue
		}
		alreadyUsed[string(addr.PublicKey)] = true
		rings[i] = addr.EncodeAddr()
	}

	return rings, nil
}

func (builder *TxsBuilder) prebuild(extraPayloads []wizard.WizardZetherPayloadExtra, senders []string, sendAssets [][]byte, amounts []uint64, recipients []string, burns []uint64, ringsConfiguration []*ZetherRingConfiguration, data []*wizard.WizardTransactionData, fees []*wizard.WizardZetherTransactionFee, ctx context.Context, statusCallback func(string)) ([]*wizard.WizardZetherTransfer, map[string]map[string][]byte, [][]*bn256.G1, map[string]*wizard.WizardZetherPublicKeyIndex, uint64, []byte, error) {

	if len(senders) != len(sendAssets) || len(sendAssets) != len(amounts) || len(amounts) != len(recipients) || len(recipients) != len(burns) || len(burns) != len(data) || len(data) != len(fees) {
		return nil, nil, nil, nil, 0, nil, errors.New("Length of senders and transfers are not matching")
	}

	sendersPrivateKeys := make([]*addresses.PrivateKey, len(senders))
	sendersWalletAddresses := make([]*wallet_address.WalletAddress, len(senders))

	for t := range senders {
		if senders[t] == "" {

			sendersPrivateKeys[t] = addresses.GenerateNewPrivateKey()
			addr, err := sendersPrivateKeys[t].GenerateAddress(true, nil, 0, nil)
			if err != nil {
				return nil, nil, nil, nil, 0, nil, err
			}
			senders[t] = addr.EncodeAddr()

		} else {

			addr, err := builder.wallet.GetWalletAddressByEncodedAddress(senders[t], true)
			if err != nil {
				return nil, nil, nil, nil, 0, nil, err
			}

			if addr.PrivateKey == nil {
				return nil, nil, nil, nil, 0, nil, errors.New("Can't be used for transactions as the private key is missing")
			}

			sendersPrivateKeys[t] = &addresses.PrivateKey{Key: addr.PrivateKey.Key[:]}
			senders[t] = addr.AddressRegistrationEncoded
			sendersWalletAddresses[t] = addr
		}

	}

	ringMembers := make([][]string, len(senders))

	var chainHeight uint64
	var chainHash []byte

	transfers := make([]*wizard.WizardZetherTransfer, len(senders))
	emap := wizard.InitializeEmap(sendAssets)
	rings := make([][]*bn256.G1, len(senders))
	publicKeyIndexes := make(map[string]*wizard.WizardZetherPublicKeyIndex)

	sendersEncryptedBalances := make([][]byte, len(senders))

	if err := store.StoreBlockchain.DB.View(func(reader store_db_interface.StoreDBTransactionInterface) (err error) {

		dataStorage := data_storage.NewDataStorage(reader)

		for t := range senders {
			if ringMembers[t], err = builder.createZetherRing(senders[t], &recipients[t], sendAssets[t], ringsConfiguration[t], dataStorage); err != nil {
				return
			}
		}

		chainHeight, _ = binary.Uvarint(reader.Get("chainHeight"))
		chainHash = reader.Get("chainHash")

		for t, ast := range sendAssets {

			var accs *accounts.Accounts
			if accs, err = dataStorage.AccsCollection.GetMap(ast); err != nil {
				return
			}

			if !bytes.Equal(ast, config_coins.NATIVE_ASSET_FULL) && fees[t].Auto {
				var assetFeeLiquidity *asset_fee_liquidity.AssetFeeLiquidity
				if assetFeeLiquidity, err = dataStorage.GetAssetFeeLiquidityTop(ast, chainHeight); err != nil {
					return
				}
				if assetFeeLiquidity == nil {
					return errors.New("There is no Asset Fee Liquidity for this asset")
				}
				fees[t].Rate = assetFeeLiquidity.Rate
				fees[t].LeadingZeros = assetFeeLiquidity.LeadingZeros
			}

			transfers[t] = &wizard.WizardZetherTransfer{
				Asset:           ast,
				Sender:          sendersPrivateKeys[t].Key[:],
				Recipient:       recipients[t],
				Amount:          amounts[t],
				Burn:            burns[t],
				Data:            data[t],
				FeeRate:         fees[t].Rate,
				FeeLeadingZeros: fees[t].LeadingZeros,
				PayloadExtra:    extraPayloads[t],
			}

			var ring []*bn256.G1
			uniqueMap := make(map[string]bool)

			addPoint := func(address string) (err error) {
				var addr *addresses.Address
				var p *crypto.Point

				if addr, err = addresses.DecodeAddr(address); err != nil {
					return
				}
				if uniqueMap[string(addr.PublicKey)] {
					return
				}
				uniqueMap[string(addr.PublicKey)] = true

				if p, err = addr.GetPoint(); err != nil {
					return
				}

				if emap[string(ast)][p.G1().String()] == nil {

					var acc *account.Account
					if acc, err = accs.GetAccount(addr.PublicKey); err != nil {
						return
					}

					var balance []byte = nil
					if acc != nil {
						balance = acc.Balance.Amount.Serialize()
					}

					if balance, err = builder.mempool.GetZetherBalance(addr.PublicKey, balance, ast); err != nil {
						return
					}

					if senders[t] == address { //sender
						sendersEncryptedBalances[t] = balance
					}

					emap[string(ast)][p.G1().String()] = balance
				}
				ring = append(ring, p.G1())

				if publicKeyIndexes[string(addr.PublicKey)] == nil {
					var reg *registration.Registration
					if reg, err = dataStorage.Regs.GetRegistration(addr.PublicKey); err != nil {
						return
					}

					publicKeyIndex := &wizard.WizardZetherPublicKeyIndex{}
					publicKeyIndexes[string(addr.PublicKey)] = publicKeyIndex

					if reg != nil {
						publicKeyIndex.Registered = true
						publicKeyIndex.RegisteredIndex = reg.Index
					} else {
						publicKeyIndex.RegistrationSignature = addr.Registration
					}
				}

				return
			}

			if err = addPoint(senders[t]); err != nil {
				return
			}
			if err = addPoint(recipients[t]); err != nil {
				return
			}
			for _, ringMember := range ringMembers[t] {
				if err = addPoint(ringMember); err != nil {
					return
				}
			}

			rings[t] = ring
		}
		statusCallback("Wallet Addresses Found")

		return
	}); err != nil {
		return nil, nil, nil, nil, 0, nil, err
	}
	statusCallback("Balances checked")

	for t := range transfers {
		if sendersWalletAddresses[t] == nil {
			transfers[t].SenderDecryptedBalance = transfers[t].Amount
		} else {

			var err error
			if transfers[t].SenderDecryptedBalance, err = builder.wallet.DecryptBalanceByPublicKey(sendersWalletAddresses[t].PublicKey, sendersEncryptedBalances[t], transfers[t].Asset, false, 0, true, true, ctx, statusCallback); err != nil {
				return nil, nil, nil, nil, 0, nil, err
			}

		}
		if transfers[t].SenderDecryptedBalance == 0 {
			return nil, nil, nil, nil, 0, nil, errors.New("You have no funds")
		}
		if transfers[t].SenderDecryptedBalance < amounts[t] {
			return nil, nil, nil, nil, 0, nil, errors.New("Not enough funds")
		}
	}

	statusCallback("Balances decoded")

	return transfers, emap, rings, publicKeyIndexes, chainHeight, chainHash, nil
}

func (builder *TxsBuilder) CreateZetherTx(extraPayloads []wizard.WizardZetherPayloadExtra, senders []string, sendAssets [][]byte, amounts []uint64, recipients []string, burns []uint64, ringsConfiguration []*ZetherRingConfiguration, data []*wizard.WizardTransactionData, fees []*wizard.WizardZetherTransactionFee, propagateTx, awaitAnswer, awaitBroadcast bool, validateTx bool, ctx context.Context, statusCallback func(string)) (*transaction.Transaction, error) {

	builder.lock.Lock()
	defer builder.lock.Unlock()

	transfers, emap, ringMembers, publicKeyIndexes, chainHeight, chainHash, err := builder.prebuild(extraPayloads, senders, sendAssets, amounts, recipients, burns, ringsConfiguration, data, fees, ctx, statusCallback)
	if err != nil {
		return nil, err
	}

	feesFinal := make([]*wizard.WizardTransactionFee, len(fees))
	for t, fee := range fees {
		feesFinal[t] = fee.WizardTransactionFee
	}

	var tx *transaction.Transaction
	if tx, err = wizard.CreateZetherTx(transfers, emap, ringMembers, chainHeight, chainHash, publicKeyIndexes, feesFinal, validateTx, ctx, statusCallback); err != nil {
		return nil, err
	}

	if propagateTx {
		if err := builder.mempool.AddTxToMempool(tx, chainHeight, true, awaitAnswer, awaitBroadcast, advanced_connection_types.UUID_ALL, ctx); err != nil {
			return nil, err
		}
	}

	return tx, nil
}
