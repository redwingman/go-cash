package wallet

import (
	"pandora-pay/blockchain/blockchain_types"
	"pandora-pay/config"
	"pandora-pay/helpers/multicast"
	"pandora-pay/wallet/wallet_address"
	"sync"
)

type wallet struct {
	Encryption           *walletEncryption               `json:"encryption" msgpack:"encryption"`
	Version              Version                         `json:"version" msgpack:"version"`
	Mnemonic             string                          `json:"mnemonic" msgpack:"mnemonic"`
	Seed                 []byte                          `json:"seed" msgpack:"seed"` //32 byte
	SeedIndex            uint32                          `json:"seedIndex" msgpack:"seedIndex"`
	Count                int                             `json:"count" msgpack:"count"`
	CountImportedIndex   int                             `json:"countIndex" msgpack:"countIndex"`
	Addresses            []*wallet_address.WalletAddress `json:"addresses" msgpack:"addresses"`
	Loaded               bool                            `json:"loaded" msgpack:"loaded"`
	DelegatesCount       int                             `json:"delegatesCount" msgpack:"delegatesCount"`
	addressesMap         map[string]*wallet_address.WalletAddress
	updateNewChainUpdate *multicast.MulticastChannel[*blockchain_types.BlockchainUpdates]
	nonHardening         bool         `json:"nonHardening" msgpack:"nonHardening"`
	Lock                 sync.RWMutex `json:"-" msgpack:"-"`
}

var Wallet *wallet

// must be locked before
func (self *wallet) clearWallet() {
	self.Version = VERSION_SIMPLE
	self.Mnemonic = ""
	self.Seed = nil
	self.SeedIndex = 0
	self.Count = 0
	self.CountImportedIndex = 0
	self.Addresses = make([]*wallet_address.WalletAddress, 0)
	self.addressesMap = make(map[string]*wallet_address.WalletAddress)
	self.Encryption = createEncryption(self)
	self.nonHardening = false
	self.setLoaded(false)
}

// must be locked before
func (self *wallet) setLoaded(newValue bool) {
	self.Loaded = newValue
	self.initWalletCLI()
}

func createWalletInstance(updateNewChainUpdate *multicast.MulticastChannel[*blockchain_types.BlockchainUpdates]) *wallet {
	w := &wallet{
		updateNewChainUpdate: updateNewChainUpdate,
	}
	w.clearWallet()
	return w
}

func createWallet() (*wallet, error) {

	w := createWalletInstance(nil)

	if err := w.loadWallet("", true); err != nil {
		if err.Error() == "cipher: message authentication failed" {
			return w, nil
		}
		if err.Error() != "Wallet doesn't exist" {
			return nil, err
		}
		if err = w.CreateEmptyWallet(); err != nil {
			return nil, err
		}
	}

	return w, nil
}

func (self *wallet) InitializeWallet(updateNewChainUpdate *multicast.MulticastChannel[*blockchain_types.BlockchainUpdates]) {

	self.Lock.Lock()
	self.updateNewChainUpdate = updateNewChainUpdate
	self.Lock.Unlock()

	if config.NODE_CONSENSUS == config.NODE_CONSENSUS_TYPE_FULL {
		self.processRefreshWallets()
	}
}

func Initialize() (err error) {
	Wallet, err = createWallet()
	return err
}
