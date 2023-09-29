package forging

import (
	"math/big"
	"pandora-pay/addresses"
	"pandora-pay/blockchain/data_storage/accounts/account"
)

type forgingWalletAddress struct {
	privateKey              *addresses.PrivateKey
	privateKeyPoint         *big.Int
	publicKey               []byte
	publicKeyStr            string
	account                 *account.Account
	decryptedStakingBalance uint64
	workerIndex             int
	chainHash               []byte
}

func (self *forgingWalletAddress) clone() *forgingWalletAddress {
	return &forgingWalletAddress{
		self.privateKey,
		self.privateKeyPoint,
		self.publicKey,
		self.publicKeyStr,
		self.account,
		self.decryptedStakingBalance,
		self.workerIndex,
		self.chainHash,
	}
}
