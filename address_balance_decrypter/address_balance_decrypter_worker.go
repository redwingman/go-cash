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

func (self *AddressBalanceDecrypterWorker) processWork(work *addressBalanceDecrypterWork) (uint64, error) {
	return balance_decrypter.BalanceDecrypter.DecryptBalance(work.encryptedBalance, false, 0, work.ctx, work.statusCallback)
}

func (self *AddressBalanceDecrypterWorker) run() {

	for {
		foundWork, _ := <-self.newWorkCn

		foundWork.result = &addressBalanceDecrypterWorkResult{}

		foundWork.result.decryptedBalance, foundWork.result.err = self.processWork(foundWork)

		foundWork.time = time.Now().Unix()
		atomic.StoreInt32(&foundWork.status, ADDRESS_BALANCE_DECRYPTED_PROCESSED)

		close(foundWork.wait)

		if config.LIGHT_COMPUTATIONS {
			time.Sleep(50 * time.Millisecond)
		}

	}
}

func (self *AddressBalanceDecrypterWorker) start() {
	go self.run()
}

func newAddressBalanceDecrypterWorker(newWorkCn chan *addressBalanceDecrypterWork) *AddressBalanceDecrypterWorker {
	worker := &AddressBalanceDecrypterWorker{
		newWorkCn,
	}
	return worker
}
