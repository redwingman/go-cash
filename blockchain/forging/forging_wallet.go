package forging

import (
	"context"
	"encoding/binary"
	"errors"
	"github.com/tevino/abool"
	"pandora-pay/address_balance_decrypter"
	"pandora-pay/blockchain/blockchain_types"
	"pandora-pay/blockchain/data_storage"
	"pandora-pay/blockchain/data_storage/accounts"
	"pandora-pay/blockchain/data_storage/accounts/account"
	"pandora-pay/blockchain/data_storage/registrations/registration"
	"pandora-pay/config/config_coins"
	"pandora-pay/config/config_forging"
	"pandora-pay/cryptography/crypto"
	"pandora-pay/gui"
	"pandora-pay/helpers/generics"
	"pandora-pay/helpers/multicast"
	"pandora-pay/store"
	"pandora-pay/store/store_db/store_db_interface"
	"pandora-pay/wallet/wallet_address/shared_staked"
	"time"
)

type forgingWallet struct {
	addressesMap           map[string]*forgingWalletAddress
	workersAddresses       []int
	workers                []*ForgingWorkerThread
	updateNewChainUpdate   *multicast.MulticastChannel[*blockchain_types.BlockchainUpdates]
	updateWalletAddressCn  chan *forgingWalletAddressUpdate
	workersCreatedCn       <-chan []*ForgingWorkerThread
	workersDestroyedCn     <-chan struct{}
	decryptBalancesUpdates *generics.Map[string, *forgingWalletAddress]
	forging                *forging
	initialized            *abool.AtomicBool
}

type forgingWalletAddressUpdate struct {
	chainHeight  uint64
	publicKey    []byte
	sharedStaked *shared_staked.WalletAddressSharedStaked
	account      *account.Account
	registration *registration.Registration
}

func (self *forgingWallet) AddWallet(publicKey []byte, sharedStaked *shared_staked.WalletAddressSharedStaked, hasAccount bool, account *account.Account, reg *registration.Registration, chainHeight uint64) (err error) {

	if !config_forging.FORGING_ENABLED || self.initialized.IsNotSet() {
		return
	}

	if !hasAccount {

		//let's read the balance
		if err = store.StoreBlockchain.DB.View(func(reader store_db_interface.StoreDBTransactionInterface) (err error) {

			chainHeight, _ = binary.Uvarint(reader.Get("chainHeight"))
			dataStorage := data_storage.NewDataStorage(reader)

			var accs *accounts.Accounts
			if accs, err = dataStorage.AccsCollection.GetMap(config_coins.NATIVE_ASSET_FULL); err != nil {
				return
			}

			if account, err = accs.Get(string(publicKey)); err != nil {
				return
			}
			if reg, err = dataStorage.Regs.Get(string(publicKey)); err != nil {
				return
			}

			return
		}); err != nil {
			return
		}

	}

	self.updateWalletAddressCn <- &forgingWalletAddressUpdate{
		chainHeight,
		publicKey,
		sharedStaked,
		account,
		reg,
	}
	return
}

func (self *forgingWallet) RemoveWallet(publicKey []byte, hasAccount bool, acc *account.Account, reg *registration.Registration, chainHeight uint64) { //20 byte
	self.AddWallet(publicKey, nil, hasAccount, acc, reg, chainHeight)
}

func (self *forgingWallet) runDecryptBalanceAndNotifyWorkers() {

	var addr *forgingWalletAddress
	for {

		found := false
		self.decryptBalancesUpdates.Range(func(publicKey string, _ *forgingWalletAddress) bool {
			addr, _ = self.decryptBalancesUpdates.LoadAndDelete(publicKey)
			found = true
			return false
		})

		if !found {
			time.Sleep(10 * time.Millisecond)
			continue
		} else {
			stakingAmountEncryptedBalanceSerialized := addr.account.Balance.Amount.Serialize()
			addr.decryptedStakingBalance, _ = address_balance_decrypter.Decrypter.DecryptBalance("wallet", addr.publicKey, addr.privateKey.Key, stakingAmountEncryptedBalanceSerialized, config_coins.NATIVE_ASSET_FULL, false, 0, true, context.Background(), func(string) {})

			self.workers[addr.workerIndex].addWalletAddressCn <- addr
		}
	}

}

func (self *forgingWallet) updateAccountToForgingWorkers(addr *forgingWalletAddress) {

	if len(self.workers) == 0 { //in case it was not started yet
		return
	}

	if addr.workerIndex == -1 {
		min := 0
		index := -1
		for i := 0; i < len(self.workersAddresses); i++ {
			if i == 0 || min > self.workersAddresses[i] {
				min = self.workersAddresses[i]
				index = i
			}
		}

		addr.workerIndex = index
		self.workersAddresses[index]++

	}

	self.decryptBalancesUpdates.Store(addr.publicKeyStr, addr.clone())
}

func (self *forgingWallet) removeAccountFromForgingWorkers(publicKey string) {

	addr := self.addressesMap[publicKey]

	if addr != nil && addr.workerIndex != -1 {
		self.workers[addr.workerIndex].removeWalletAddressCn <- addr.publicKeyStr
		self.workersAddresses[addr.workerIndex]--
		addr.workerIndex = -1
	}
}

func (self *forgingWallet) deleteAccount(publicKey string) {
	if addr := self.addressesMap[publicKey]; addr != nil {
		self.removeAccountFromForgingWorkers(publicKey)
	}
}

func (self *forgingWallet) runProcessUpdates() {

	var err error

	updateNewChainCn := self.updateNewChainUpdate.AddListener()
	defer self.updateNewChainUpdate.RemoveChannel(updateNewChainCn)

	var chainHash []byte

	for {
		select {
		case workers := <-self.workersCreatedCn:

			self.workers = workers
			self.workersAddresses = make([]int, len(workers))
			for _, addr := range self.addressesMap {
				self.updateAccountToForgingWorkers(addr)
			}
		case <-self.workersDestroyedCn:

			self.workers = []*ForgingWorkerThread{}
			self.workersAddresses = []int{}
			for _, addr := range self.addressesMap {
				addr.workerIndex = -1
			}
		case update := <-self.updateWalletAddressCn:

			key := string(update.publicKey)

			//let's delete it
			if update.sharedStaked == nil || update.sharedStaked.PrivateKey == nil {
				self.removeAccountFromForgingWorkers(key)
			} else {

				if err = func() (err error) {

					if update.account == nil {
						return errors.New("Account was not found")
					}
					if update.registration == nil {
						return errors.New("Registration was not found")
					}

					if !update.registration.Staked {
						return errors.New("It is no longer staked")
					}

					address := self.addressesMap[key]
					if address == nil {

						keyPoint := new(crypto.BNRed).SetBytes(update.sharedStaked.PrivateKey.Key)

						address = &forgingWalletAddress{
							update.sharedStaked.PrivateKey,
							keyPoint.BigInt(),
							update.publicKey,
							string(update.publicKey),
							update.account,
							0,
							-1,
							chainHash,
						}
						self.addressesMap[key] = address
						self.updateAccountToForgingWorkers(address)
					}

					return
				}(); err != nil {
					self.deleteAccount(key)
					gui.GUI.Error(err)
				}

			}
		case update := <-updateNewChainCn:

			accs, _ := update.AccsCollection.GetMapIfExists(config_coins.NATIVE_ASSET_FULL)
			if accs == nil {
				continue
			}

			chainHash = update.BlockHash

			for k, v := range update.Registrations.Committed {
				if self.addressesMap[k] != nil {
					if v.Stored == "update" {
						if !v.Element.Staked {
							self.deleteAccount(k)
						}
					} else if v.Stored == "delete" {
						self.deleteAccount(k)
					}
				}
			}

			for k, v := range accs.HashMap.Committed {
				if self.addressesMap[k] != nil {
					if v.Stored == "update" {

						self.addressesMap[k].account = v.Element
						self.addressesMap[k].chainHash = chainHash
						self.updateAccountToForgingWorkers(self.addressesMap[k])

					} else if v.Stored == "delete" {
						self.deleteAccount(k)
						gui.GUI.Error("Account was deleted from Forging")
					}

				}
			}

		}

	}

}
