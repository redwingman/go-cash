package wallet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"pandora-pay/blockchain/data_storage"
	"pandora-pay/blockchain/data_storage/accounts"
	"pandora-pay/blockchain/data_storage/accounts/account"
	"pandora-pay/blockchain/data_storage/registrations/registration"
	"pandora-pay/blockchain/forging"
	"pandora-pay/config/config_coins"
	"pandora-pay/config/config_forging"
	"pandora-pay/config/globals"
	"pandora-pay/gui"
	"pandora-pay/helpers"
	"pandora-pay/helpers/generics"
	"pandora-pay/helpers/msgpack"
	"pandora-pay/store"
	"pandora-pay/store/store_db/store_db_interface"
	"pandora-pay/wallet/wallet_address"
	"strconv"
)

func (self *wallet) saveWalletAddress(adr *wallet_address.WalletAddress, lock bool) error {

	if lock {
		self.Lock.RLock()
		defer self.Lock.RUnlock()
	}

	for i, adr2 := range self.Addresses {
		if adr2 == adr {
			return self.saveWallet(i, i+1, -1, false)
		}
	}

	return nil
}

func (self *wallet) saveWalletEntire(lock bool) error {
	if lock {
		self.Lock.RLock()
		defer self.Lock.RUnlock()
	}
	return self.saveWallet(0, self.Count, -1, false)
}

func (self *wallet) saveWallet(start, end, deleteIndex int, lock bool) error {

	if lock {
		self.Lock.RLock()
		defer self.Lock.RUnlock()
	}

	start = generics.Max(0, start)
	end = generics.Min(end, len(self.Addresses))

	if !self.Loaded {
		return errors.New("Can't save your wallet because your stored wallet on the drive was not successfully loaded")
	}

	return store.StoreWallet.DB.Update(func(writer store_db_interface.StoreDBTransactionInterface) (err error) {

		var marshal []byte

		writer.Put("saved", []byte{0})

		if marshal, err = helpers.GetMarshalledDataExcept(self.Encryption); err != nil {
			return
		}
		writer.Put("encryption", marshal)

		if marshal, err = helpers.GetMarshalledDataExcept(self, "addresses", "encryption"); err != nil {
			return
		}
		if marshal, err = self.Encryption.encryptData(marshal); err != nil {
			return
		}

		writer.Put("wallet", marshal)

		for i := start; i < end; i++ {
			if marshal, err = msgpack.Marshal(self.Addresses[i]); err != nil {
				return
			}
			if marshal, err = self.Encryption.encryptData(marshal); err != nil {
				return
			}
			writer.Put("wallet-address-"+strconv.Itoa(i), marshal)
		}
		if deleteIndex != -1 {
			writer.Delete("wallet-address-" + strconv.Itoa(deleteIndex))
		}

		writer.Put("saved", []byte{1})
		return
	})
}

func (self *wallet) loadWallet(password string, firstTime bool) error {
	self.Lock.Lock()
	defer self.Lock.Unlock()

	if self.Loaded {
		return errors.New("Wallet was already loaded!")
	}

	self.clearWallet()

	return store.StoreWallet.DB.View(func(reader store_db_interface.StoreDBTransactionInterface) (err error) {

		saved := reader.Get("saved") //safe only internal
		if saved == nil {
			return errors.New("Wallet doesn't exist")
		}

		if bytes.Equal(saved, []byte{1}) {

			gui.GUI.Log("Wallet Loading... ")

			var unmarshal []byte

			unmarshal = reader.Get("encryption")
			if unmarshal == nil {
				return errors.New("encryption data was not found")
			}
			if err = msgpack.Unmarshal(unmarshal, self.Encryption); err != nil {
				return
			}

			if self.Encryption.Encrypted != ENCRYPTED_VERSION_PLAIN_TEXT {
				if password == "" {
					return nil
				}
				self.Encryption.password = password
				if err = self.Encryption.createEncryptionCipher(); err != nil {
					return
				}
			}

			if unmarshal, err = self.Encryption.decryptData(reader.Get("wallet")); err != nil {
				return
			}
			if err = msgpack.Unmarshal(unmarshal, self); err != nil {
				return
			}

			self.Addresses = make([]*wallet_address.WalletAddress, 0)
			self.addressesMap = make(map[string]*wallet_address.WalletAddress)

			for i := 0; i < self.Count; i++ {

				if unmarshal, err = self.Encryption.decryptData(reader.Get("wallet-address-" + strconv.Itoa(i))); err != nil {
					return
				}

				newWalletAddress := &wallet_address.WalletAddress{}
				if err = msgpack.Unmarshal(unmarshal, newWalletAddress); err != nil {
					return
				}

				if newWalletAddress.PrivateKey != nil {
					if !bytes.Equal(newWalletAddress.PrivateKey.GeneratePublicKey(), newWalletAddress.PublicKey) {
						return errors.New("Public Keys are not matching!")
					}
				}

				self.Addresses = append(self.Addresses, newWalletAddress)
				self.addressesMap[string(newWalletAddress.PublicKey)] = newWalletAddress

			}

			self.setLoaded(true)
			if !firstTime {
				if err = self.walletLoaded(firstTime); err != nil {
					return
				}
			}

		} else {
			return errors.New("Error loading wallet ?")
		}
		return
	})
}

func (self *wallet) walletLoaded(firstTime bool) error {

	if !firstTime {
		if err := self.InitForgingWallet(); err != nil {
			return err
		}
	}

	self.updateWallet()
	globals.MainEvents.BroadcastEvent("wallet/loaded", self.Count)
	gui.GUI.Log("Wallet Loaded! " + strconv.Itoa(self.Count))

	return nil
}

func (self *wallet) StartWallet() error {

	self.Lock.Lock()
	defer self.Lock.Unlock()

	return self.walletLoaded(true)
}

func (self *wallet) InitForgingWallet() (err error) {

	if !config_forging.FORGING_ENABLED {
		return nil
	}

	for _, addr := range self.Addresses {
		if err = forging.Forging.Wallet.AddWallet(addr.PublicKey, addr.SharedStaked, false, nil, nil, 0); err != nil {
			return
		}
	}

	return store.StoreBlockchain.DB.View(func(reader store_db_interface.StoreDBTransactionInterface) (err error) {

		chainHeight, _ := binary.Uvarint(reader.Get("chainHeight"))
		dataStorage := data_storage.NewDataStorage(reader)
		var accs *accounts.Accounts
		if accs, err = dataStorage.AccsCollection.GetMap(config_coins.NATIVE_ASSET_FULL); err != nil {
			return
		}

		for _, addr := range self.Addresses {

			var acc *account.Account
			var reg *registration.Registration

			if acc, err = accs.Get(string(addr.PublicKey)); err != nil {
				return
			}
			if reg, err = dataStorage.Regs.Get(string(addr.PublicKey)); err != nil {
				return
			}

			if err = self.refreshWalletAccount(acc, reg, chainHeight, addr); err != nil {
				return
			}
		}

		return
	})
}
