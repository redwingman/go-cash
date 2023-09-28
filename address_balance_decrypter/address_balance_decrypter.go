package address_balance_decrypter

import (
	"context"
	"github.com/tevino/abool"
	"pandora-pay/config"
	"pandora-pay/cryptography/bn256"
	"pandora-pay/cryptography/crypto"
	"pandora-pay/cryptography/crypto/balance_decryptor"
	"pandora-pay/helpers/generics"
)

type AddressBalanceDecrypter struct {
	all                   *generics.Map[string, *addressBalanceDecrypterWork]
	previousValues        *generics.Map[string, uint64]
	previousValuesChanged *abool.AtomicBool
	workers               []*AddressBalanceDecrypterWorker
	newWorkCn             chan *addressBalanceDecrypterWork
}

func (decryptor *AddressBalanceDecrypter) DecryptBalance(decryptionName string, publicKey, privateKey, encryptedBalance, asset []byte, useNewPreviousValue bool, newPreviousValue uint64, storeNewPreviousValue bool, ctx context.Context, statusCallback func(string)) (uint64, error) {

	if len(encryptedBalance) == 0 {
		return 0, nil
	}

	previousValue := uint64(0)
	if useNewPreviousValue {
		previousValue = newPreviousValue
	} else {
		previousValue, _ = decryptor.previousValues.Load(string(publicKey) + "_" + string(asset) + "_" + decryptionName)
	}

	balance, err := new(crypto.ElGamal).Deserialize(encryptedBalance)
	if err != nil {
		return 0, err
	}

	balancePoint := new(bn256.G1).Add(balance.Left, new(bn256.G1).Neg(new(bn256.G1).ScalarMult(balance.Right, new(crypto.BNRed).SetBytes(privateKey).BigInt())))
	if balance_decryptor.BalanceDecrypter.TryDecryptBalance(balancePoint, previousValue) {
		return previousValue, nil
	}

	foundWork, loaded := decryptor.all.LoadOrStore(string(publicKey)+"_"+string(encryptedBalance), &addressBalanceDecrypterWork{balancePoint, previousValue, make(chan struct{}), ADDRESS_BALANCE_DECRYPTED_INIT, 0, nil, ctx, statusCallback})
	if !loaded {
		decryptor.newWorkCn <- foundWork
	}

	<-foundWork.wait
	if foundWork.result.err != nil {
		return 0, foundWork.result.err
	}

	if storeNewPreviousValue {
		decryptor.SaveDecryptedBalance(decryptionName, publicKey, asset, foundWork.result.decryptedBalance)
	}

	return foundWork.result.decryptedBalance, nil
}

func (decryptor *AddressBalanceDecrypter) SaveDecryptedBalance(decryptionName string, publicKey, asset []byte, value uint64) {
	decryptor.previousValues.Store(string(publicKey)+"_"+string(asset)+"_"+decryptionName, value)
	decryptor.previousValuesChanged.Set()
}

func newAddressBalanceDecrypter(useStore bool) (*AddressBalanceDecrypter, error) {

	threadsCount := config.CPU_THREADS
	if config.LIGHT_COMPUTATIONS {
		threadsCount = generics.Max(1, config.CPU_THREADS/2)
	}

	addressBalanceDecrypter := &AddressBalanceDecrypter{
		&generics.Map[string, *addressBalanceDecrypterWork]{},
		&generics.Map[string, uint64]{},
		abool.New(),
		make([]*AddressBalanceDecrypterWorker, threadsCount),
		make(chan *addressBalanceDecrypterWork, 1),
	}

	if useStore {
		if err := addressBalanceDecrypter.loadFromStore(); err != nil {
			return nil, err
		}
	}

	for i := range addressBalanceDecrypter.workers {
		addressBalanceDecrypter.workers[i] = newAddressBalanceDecrypterWorker(addressBalanceDecrypter.newWorkCn)
	}

	for _, worker := range addressBalanceDecrypter.workers {
		worker.start()
	}

	if useStore {
		go addressBalanceDecrypter.saveToStore()
	}

	return addressBalanceDecrypter, nil
}

var Decrypter *AddressBalanceDecrypter

func Initialize(useStore bool) (err error) {
	Decrypter, err = newAddressBalanceDecrypter(useStore)
	return
}
