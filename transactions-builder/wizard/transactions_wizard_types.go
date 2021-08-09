package wizard

import (
	"crypto/rand"
	transaction_type "pandora-pay/blockchain/transactions/transaction/transaction-type"
	"pandora-pay/cryptography/ecdsa"
	"pandora-pay/cryptography/ecies"
	"pandora-pay/helpers"
)

type TransactionsWizardFee struct {
	Fixed, PerByte uint64
	PerByteAuto    bool
	Token          []byte
}

type TransactionsWizardFeeExtra struct {
	TransactionsWizardFee
	PayInExtra bool
}

type TransactionsWizardData struct {
	Data               helpers.HexBytes `json:"data,omitempty"`
	Encrypt            bool             `json:"encrypt,omitempty"`
	PublicKeyToEncrypt helpers.HexBytes `json:"publicKeyToEncrypt,omitempty"`
}

func (data *TransactionsWizardData) getDataVersion() transaction_type.TransactionDataVersion {
	if len(data.Data) == 0 {
		return transaction_type.TX_DATA_NONE
	}
	if data.Encrypt {
		return transaction_type.TX_DATA_ENCRYPTED
	}
	return transaction_type.TX_DATA_PLAIN_TEXT
}

func (data *TransactionsWizardData) getData() ([]byte, error) {

	if len(data.Data) == 0 {
		return nil, nil
	}

	if !data.Encrypt {
		return data.Data, nil
	} else {

		pub, err := ecdsa.UnmarshalPubkey(data.PublicKeyToEncrypt)
		if err != nil {
			return nil, err
		}

		return ecies.Encrypt(rand.Reader, ecies.ImportECDSAPublic(pub), data.Data, nil, nil)
	}

}
