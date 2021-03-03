package transaction_simple

import (
	"errors"
	"pandora-pay/blockchain/transactions/transaction/transaction-simple/transaction_simple_unstake"
	transaction_type "pandora-pay/blockchain/transactions/transaction/transaction-type"
	"pandora-pay/crypto/ecdsa"
	"pandora-pay/helpers"
)

type TransactionSimple struct {
	Nonce uint64
	Vin   []TransactionSimpleInput
	Vout  []TransactionSimpleOutput
	Extra interface{}
}

func (tx *TransactionSimple) VerifySignature(hash helpers.Hash) bool {
	if len(tx.Vin) == 0 {
		return false
	}

	for _, vin := range tx.Vin {
		if ecdsa.VerifySignature(vin.PublicKey[:], hash[:], vin.Signature[0:64]) == false {
			return false
		}
	}
	return true
}

func (tx *TransactionSimple) Serialize(writer *helpers.BufferWriter, inclSignature bool, txType transaction_type.TransactionType) {
	writer.WriteUvarint(tx.Nonce)

	writer.WriteUvarint(uint64(len(tx.Vin)))
	for _, vin := range tx.Vin {
		vin.Serialize(writer, inclSignature)
	}

	//vout only TransactionTypeSimple
	if txType == transaction_type.TransactionTypeSimple {
		writer.WriteUvarint(uint64(len(tx.Vout)))
		for _, vout := range tx.Vout {
			vout.Serialize(writer)
		}
	}

	switch txType {
	case transaction_type.TransactionTypeSimpleUnstake:
		extra := tx.Extra.(transaction_simple_unstake.TransactionSimpleUnstake)
		extra.Serialize(writer)
	}
}

func (tx *TransactionSimple) Deserialize(reader *helpers.BufferReader, txType transaction_type.TransactionType) (err error) {

	var n uint64

	if tx.Nonce, err = reader.ReadUvarint(); err != nil {
		return
	}

	if n, err = reader.ReadUvarint(); err != nil {
		return
	}
	for i := 0; i < int(n); i++ {
		vin := TransactionSimpleInput{}
		if err = vin.Deserialize(reader); err != nil {
			return
		}
		tx.Vin = append(tx.Vin, vin)
	}

	//vout only TransactionTypeSimple
	if txType == transaction_type.TransactionTypeSimple {
		if n, err = reader.ReadUvarint(); err != nil {
			return
		}
		for i := 0; i < int(n); i++ {
			vout := TransactionSimpleOutput{}
			if err = vout.Deserialize(reader); err != nil {
				return
			}
			tx.Vout = append(tx.Vout, vout)
		}
	}

	switch txType {
	case transaction_type.TransactionTypeSimple:
	case transaction_type.TransactionTypeSimpleUnstake:
		extra := transaction_simple_unstake.TransactionSimpleUnstake{}
		err = extra.Deserialize(reader)
		tx.Extra = extra
	default:
		err = errors.New("Invalid txType")
	}

	return
}