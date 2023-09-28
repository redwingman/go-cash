package address_balance_decrypter

import (
	"pandora-pay/config"
	"pandora-pay/cryptography/crypto/balance_decrypter"
	"sync/atomic"
	"time"
)

type AddressBalanceDecrypterWorker struct {
	newWorkCn chan *addressBalanceDecrypterWork
}

func (worker *AddressBalanceDecrypterWorker) processWork(work *addressBalanceDecrypterWork) (uint64, error) {
	return balance_decrypter.BalanceDecrypter.DecryptBalance(work.encryptedBalance, false, 0, work.ctx, work.statusCallback)
}

func (worker *AddressBalanceDecrypterWorker) run() {

	for {
		foundWork, _ := <-worker.newWorkCn

		foundWork.result = &addressBalanceDecrypterWorkResult{}

		foundWork.result.decryptedBalance, foundWork.result.err = worker.processWork(foundWork)

		foundWork.time = time.Now().Unix()
		atomic.StoreInt32(&foundWork.status, ADDRESS_BALANCE_DECRYPTED_PROCESSED)

		close(foundWork.wait)

		if config.LIGHT_COMPUTATIONS {
			time.Sleep(50 * time.Millisecond)
		}

	}
}

func (worker *AddressBalanceDecrypterWorker) start() {
	go worker.run()
}

func newAddressBalanceDecrypterWorker(newWorkCn chan *addressBalanceDecrypterWork) *AddressBalanceDecrypterWorker {
	worker := &AddressBalanceDecrypterWorker{
		newWorkCn,
	}
	return worker
}
