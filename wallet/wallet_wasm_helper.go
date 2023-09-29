package wallet

func (self *wallet) GetPrivateKeys(publicKey, asset []byte) (privateKey, spendPrivateKey []byte, previousValue uint64) {

	self.Lock.RLock()
	defer self.Lock.RUnlock()

	addr := self.addressesMap[string(publicKey)]

	if addr.PrivateKey != nil {
		privateKey = addr.PrivateKey.Key
	}

	if addr.SpendPrivateKey != nil {
		spendPrivateKey = addr.SpendPrivateKey.Key
	}

	return
}
