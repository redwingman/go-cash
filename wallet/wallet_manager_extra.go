package wallet

import (
	"context"
	"errors"
	"fmt"
	"github.com/tyler-smith/go-bip39"
	"pandora-pay/addresses"
	"pandora-pay/blockchain/data_storage/accounts/account"
	"pandora-pay/blockchain/data_storage/registrations/registration"
	"pandora-pay/blockchain/forging"
	"pandora-pay/config/config_coins"
	"pandora-pay/config/config_stake"
	"pandora-pay/gui"
	"pandora-pay/wallet/wallet_address"
)

func (self *wallet) createSeed(lock bool) error {

	if lock {
		self.Lock.Lock()
		defer self.Lock.Unlock()
	}

	if !self.Loaded {
		return errors.New("Wallet was not loaded!")
	}

	for {
		entropy, err := bip39.NewEntropy(256)
		if err != nil {
			continue
		}

		mnemonic, err := bip39.NewMnemonic(entropy)
		if err != nil {
			continue
		}

		self.Mnemonic = mnemonic

		// Generate a Bip32 HD wallet for the mnemonic and a user supplied password
		seed, err := bip39.NewSeedWithErrorChecking(mnemonic, "SEED Secret Passphrase")
		if err != nil {
			continue
		}

		var seedExtended *addresses.SeedExtended
		if seedExtended, err = addresses.NewSeedExtended(seed); err != nil {
			continue
		}

		self.Seed = seedExtended.Serialize()
		return nil
	}
}

func (self *wallet) CreateEmptyWallet() (err error) {

	self.Lock.Lock()
	defer self.Lock.Unlock()

	self.clearWallet()
	self.setLoaded(true)

	if err = self.createSeed(false); err != nil {
		return
	}
	if _, err = self.AddNewAddress(false, "", false, false, true); err != nil {
		return
	}

	return
}

func (self *wallet) ImportMnemonic(mnemonic string) (err error) {

	self.Lock.Lock()
	defer self.Lock.Unlock()

	if self.Mnemonic == mnemonic {
		return
	}

	self.clearWallet()
	self.setLoaded(true)

	self.Mnemonic = mnemonic

	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, "SEED Secret Passphrase")
	if err != nil {
		return
	}

	seedExtended, err := addresses.NewSeedExtended(seed)
	if err != nil {
		return
	}

	self.Seed = seedExtended.Serialize()

	if _, err = self.AddNewAddress(false, "", false, false, true); err != nil {
		return
	}

	return
}

func (self *wallet) ImportEntropy(entropy []byte) (err error) {

	self.Lock.Lock()
	defer self.Lock.Unlock()

	var mnemonic string
	if mnemonic, err = bip39.NewMnemonic(entropy); err != nil {
		return
	}

	if mnemonic == self.Mnemonic {
		return
	}

	self.clearWallet()
	self.setLoaded(true)

	self.Mnemonic = mnemonic

	seed, err := bip39.NewSeedWithErrorChecking(self.Mnemonic, "SEED Secret Passphrase")
	if err != nil {
		return err
	}

	seedExtended, err := addresses.NewSeedExtended(seed)
	if err != nil {
		return
	}

	self.Seed = seedExtended.Serialize()

	if _, err = self.AddNewAddress(false, "", false, false, true); err != nil {
		return
	}

	return
}

func (self *wallet) updateWallet() {
	gui.GUI.InfoUpdate("Wallet Addrs", fmt.Sprintf("%d  %s", self.Count, self.Encryption.Encrypted))
}

// it must be locked and use original walletAddresses, not cloned ones
func (self *wallet) refreshWalletAccount(acc *account.Account, reg *registration.Registration, chainHeight uint64, addr *wallet_address.WalletAddress) (err error) {

	deleted := false

	if acc == nil || reg == nil || !reg.Staked || addr.SharedStaked == nil {
		deleted = true
	} else {

		stakingAmountBalance := acc.Balance.Amount.Serialize()

		var stakingAmount uint64
		if stakingAmountBalance != nil {
			stakingAmount, _ = self.DecryptBalance(addr, stakingAmountBalance, config_coins.NATIVE_ASSET_FULL, false, 0, true, context.Background(), func(string) {})
		}

		if stakingAmount < config_stake.GetRequiredStake(chainHeight) {
			deleted = true
		}

	}

	if deleted {

		forging.Forging.Wallet.RemoveWallet(addr.PublicKey, true, acc, reg, chainHeight)

		if addr.IsSharedStaked {
			_, err = self.RemoveAddressByPublicKey(addr.PublicKey, true)
			return
		}

	} else {
		forging.Forging.Wallet.AddWallet(addr.PublicKey, addr.SharedStaked, true, acc, reg, chainHeight)
	}

	return
}
