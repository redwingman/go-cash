package wallet_address

import (
	"errors"
	"pandora-pay/addresses"
	"pandora-pay/wallet/wallet_address/shared_staked"
)

type WalletAddress struct {
	Version                    Version                                  `json:"version" msgpack:"version"`
	Name                       string                                   `json:"name" msgpack:"name"`
	SeedIndex                  uint32                                   `json:"seedIndex" msgpack:"seedIndex"`
	IsMine                     bool                                     `json:"isMine" msgpack:"isMine"`
	IsImported                 bool                                     `json:"isImported" msgpack:"isImported"`
	SecretKey                  []byte                                   `json:"secretKey" msgpack:"secretKey"`
	PrivateKey                 *addresses.PrivateKey                    `json:"privateKey" msgpack:"privateKey"`
	SpendPrivateKey            *addresses.PrivateKey                    `json:"spendPrivateKey" msgpack:"spendPrivateKey"`
	Registration               []byte                                   `json:"registration" msgpack:"registration"`
	PublicKey                  []byte                                   `json:"publicKey" msgpack:"publicKey"`
	Staked                     bool                                     `json:"staked" msgpack:"staked"`
	SpendRequired              bool                                     `json:"spendRequired" msgpack:"spendRequired"`
	SpendPublicKey             []byte                                   `json:"spendPublicKey" msgpack:"spendPublicKey"`
	IsSharedStaked             bool                                     `json:"isSharedStaked,omitempty" msgpack:"isSharedStaked,omitempty"`
	SharedStaked               *shared_staked.WalletAddressSharedStaked `json:"sharedStaked,omitempty" msgpack:"sharedStaked,omitempty"`
	AddressEncoded             string                                   `json:"addressEncoded" msgpack:"addressEncoded"`
	AddressRegistrationEncoded string                                   `json:"addressRegistrationEncoded" msgpack:"addressRegistrationEncoded"`
}

func (self *WalletAddress) DeriveSharedStaked() (*shared_staked.WalletAddressSharedStaked, error) {

	if self.PrivateKey == nil {
		return nil, errors.New("Private Key is missing")
	}

	return &shared_staked.WalletAddressSharedStaked{
		PrivateKey: self.PrivateKey,
		PublicKey:  self.PublicKey,
	}, nil

}

func (self *WalletAddress) GetAddress(registered bool) string {
	if registered {
		return self.AddressEncoded
	}
	return self.AddressRegistrationEncoded
}

func (self *WalletAddress) DecryptMessage(message []byte) ([]byte, error) {
	if self.PrivateKey == nil {
		return nil, errors.New("Private Key is missing")
	}
	return self.PrivateKey.Decrypt(message)
}

func (self *WalletAddress) SignMessage(message []byte) ([]byte, error) {
	if self.PrivateKey == nil {
		return nil, errors.New("Private Key is missing")
	}
	return self.PrivateKey.Sign(message)
}

func (self *WalletAddress) VerifySignedMessage(message, signature []byte) (bool, error) {
	address, err := addresses.DecodeAddr(self.GetAddress(false))
	if err != nil {
		return false, err
	}
	return address.VerifySignedMessage(message, signature), nil
}

func (self *WalletAddress) Clone() *WalletAddress {

	if self == nil {
		return nil
	}

	var sharedStaked *shared_staked.WalletAddressSharedStaked
	if self.SharedStaked != nil {
		sharedStaked = &shared_staked.WalletAddressSharedStaked{self.SharedStaked.PrivateKey, self.SharedStaked.PublicKey}
	}

	return &WalletAddress{
		self.Version,
		self.Name,
		self.SeedIndex,
		self.IsMine,
		self.IsImported,
		self.SecretKey,
		self.PrivateKey,
		self.SpendPrivateKey,
		self.Registration,
		self.PublicKey,
		self.Staked,
		self.SpendRequired,
		self.SpendPublicKey,
		self.IsSharedStaked,
		sharedStaked,
		self.AddressEncoded,
		self.AddressRegistrationEncoded,
	}
}
