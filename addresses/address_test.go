package addresses

import (
	"github.com/stretchr/testify/assert"
	"pandora-pay/config"
	"pandora-pay/helpers"
	"pandora-pay/helpers/base58"
	"testing"
)

func TestAddress_EncodeAddr(t *testing.T) {

	//WIF
	//1+20+1+4

	privateKey := GenerateNewPrivateKey()
	address, err := privateKey.GenerateAddress(true, 0, helpers.EmptyBytes(0))
	assert.Nil(t, err, "Address Generation raised an error")
	assert.Equal(t, len(address.PublicKey), 20, "Address Generated is invalid")
	assert.Equal(t, len(address.PaymentID), 0, "Address Generated is invalid")

	encoded, err := address.EncodeAddr()
	assert.Nil(t, err, "Address EncodeAddr raised an error")

	decoded, err := base58.Decode(encoded[config.NETWORK_BYTE_PREFIX_LENGTH:])
	assert.Nil(t, err, "Address Decoding raised an error")
	assert.Equal(t, len(decoded), 1+20+1+4, "AddressEncoded length is invalid")

	address, err = privateKey.GenerateAddress(true, 20, helpers.EmptyBytes(0))
	assert.Nil(t, err, "Address Generating Address raised an error")
	assert.Equal(t, len(address.PublicKey), 20, "Address Generated is invalid")
	assert.Equal(t, len(address.PaymentID), 0, "Address Generated is invalid")
	assert.Equal(t, address.Amount, uint64(20), "Address Generated Amount is invalid")

	encodedAmount, err := address.EncodeAddr()
	assert.Nil(t, err, "Encoding Address raised an error")
	assert.NotEqual(t, len(encoded), len(encodedAmount), "Encoded Amounts are invalid")
	assert.NotEqual(t, encoded, encodedAmount, "Encoded Amounts are invalid")

	address, err = privateKey.GenerateAddress(true, 20, helpers.EmptyBytes(8))
	assert.Nil(t, err, "Encoding Address raised an error")
	assert.Equal(t, len(address.PublicKey), 20, "Address Generated is invalid")
	assert.Equal(t, len(address.PaymentID), 8, "Address Generated is invalid")
	assert.Equal(t, address.Amount, uint64(20), "Address Generated Amount is invalid")

	encodedAmountPaymentId, err := address.EncodeAddr()
	assert.Nil(t, err, "Encoding Address raised an error")
	assert.NotEqual(t, len(encodedAmount), len(encodedAmountPaymentId), "Encoded Amounts are invalid")
	assert.NotEqual(t, len(encoded), len(encodedAmountPaymentId), "Encoded Amounts are invalid")
	assert.NotEqual(t, encodedAmount, encodedAmountPaymentId, "Encoded Amounts are invalid")
	assert.NotEqual(t, encoded, encodedAmountPaymentId, "Encoded Amounts are invalid")

}

func TestDecodeAddr(t *testing.T) {

	privateKey := GenerateNewPrivateKey()
	address, err := privateKey.GenerateAddress(true, 0, helpers.EmptyBytes(0))
	assert.Nil(t, err)

	encoded, err := address.EncodeAddr()
	assert.Nil(t, err)

	decodedAddress, err := DecodeAddrSilent(encoded)
	assert.Nil(t, err, "Invalid Decoded Address")

	assert.Equal(t, decodedAddress.PublicKey, address.PublicKey, "Decoded Address is not identical")
	assert.Equal(t, decodedAddress.Amount, address.Amount, "Decoded Address is not identical")
	assert.Equal(t, decodedAddress.PaymentID, address.PaymentID, "Decoded Address is not identical")

	address, err = privateKey.GenerateAddress(false, 40, helpers.EmptyBytes(8))
	assert.Nil(t, err)

	encoded, err = address.EncodeAddr()
	assert.Nil(t, err)

	decodedAddress, err = DecodeAddrSilent(encoded)
	assert.Nil(t, err)

	assert.Equal(t, decodedAddress.PublicKey, address.PublicKey, "Decoded Address is not identical")
	assert.Equal(t, decodedAddress.Amount, address.Amount, "Decoded Address is not identical")
	assert.Equal(t, decodedAddress.PaymentID, address.PaymentID, "Decoded Address is not identical")

}
