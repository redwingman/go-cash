package address_balance_decrypter

import (
	"context"
	"pandora-pay/cryptography/bn256"
)

type addressBalanceDecrypterWorkResult struct {
	decryptedBalance uint64
	err              error
}

type addressBalanceDecrypterWork struct {
	encryptedBalance *bn256.G1
	previousValue    uint64
	wait             chan struct{}
	status           int32 //use atomic
	time             int64
	result           *addressBalanceDecrypterWorkResult
	ctx              context.Context
	statusCallback   func(string)
}

const (
	ADDRESS_BALANCE_DECRYPTED_INIT int32 = iota
	ADDRESS_BALANCE_DECRYPTED_PROCESSED
)
