package api_types

import (
	"encoding/hex"
	"errors"
	"net/url"
	"pandora-pay/addresses"
	"pandora-pay/cryptography"
	"pandora-pay/helpers"
	"strconv"
	"strings"
)

type SubscriptionType uint8

const (
	SUBSCRIPTION_ACCOUNT SubscriptionType = iota
	SUBSCRIPTION_PLAIN_ACCOUNT
	SUBSCRIPTION_ACCOUNT_TRANSACTIONS
	SUBSCRIPTION_TOKEN
	SUBSCRIPTION_REGISTRATION
	SUBSCRIPTION_TRANSACTION
)

type APIReturnType uint8

const (
	RETURN_JSON APIReturnType = iota
	RETURN_SERIALIZED
)

func GetReturnType(s string, defaultValue APIReturnType) APIReturnType {
	switch s {
	case "0":
		return RETURN_JSON
	case "1":
		return RETURN_SERIALIZED
	default:
		return defaultValue
	}
}

type APIBlockRequest struct {
	Height     uint64           `json:"height,omitempty"`
	Hash       helpers.HexBytes `json:"hash,omitempty"`
	ReturnType APIReturnType    `json:"returnType,omitempty"`
}

type APIBlockInfoRequest struct {
	Height uint64           `json:"height,omitempty"`
	Hash   helpers.HexBytes `json:"hash,omitempty"`
}

type APIBlockCompleteMissingTxsRequest struct {
	Hash       helpers.HexBytes `json:"hash,omitempty"`
	MissingTxs []int            `json:"missingTxs,omitempty"`
}

type APIBlockCompleteRequest struct {
	Height     uint64           `json:"height,omitempty"`
	Hash       helpers.HexBytes `json:"hash,omitempty"`
	ReturnType APIReturnType    `json:"returnType,omitempty"`
}

type APITransactionRequest struct {
	Height     uint64           `json:"height,omitempty"`
	Hash       helpers.HexBytes `json:"hash,omitempty"`
	ReturnType APIReturnType    `json:"returnType,omitempty"`
}

type APITransactionInfoRequest struct {
	Height uint64           `json:"height,omitempty"`
	Hash   helpers.HexBytes `json:"hash,omitempty"`
}

type APIAccountBaseRequest struct {
	Address   string           `json:"address,omitempty"`
	PublicKey helpers.HexBytes `json:"publicKey,omitempty"`
}

type APIAccountRequest struct {
	APIAccountBaseRequest
	ReturnType APIReturnType `json:"returnType,omitempty"`
}

type APIAccountTxsRequest struct {
	APIAccountBaseRequest
	Next uint64 `json:"next,omitempty"`
}

type APIAccountsKeysByIndexRequest struct {
	Indexes         []uint64         `json:"indexes"`
	Token           helpers.HexBytes `json:"token"`
	EncodeAddresses bool             `json:"encodeAddresses"`
}

type APIAccountsByKeysRequest struct {
	Keys           []*APIAccountBaseRequest `json:"keys,omitempty"`
	Token          helpers.HexBytes         `json:"token,omitempty"`
	IncludeMempool bool                     `json:"includeMempool,omitempty"`
	ReturnType     APIReturnType            `json:"returnType,omitempty"`
}

func (request *APIAccountBaseRequest) GetPublicKey() ([]byte, error) {
	var publicKey []byte
	if request.Address != "" {
		address, err := addresses.DecodeAddr(request.Address)
		if err != nil {
			return nil, errors.New("Invalid address")
		}
		publicKey = address.PublicKey
	} else if request.PublicKey != nil && len(request.PublicKey) == cryptography.PublicKeySize {
		publicKey = request.PublicKey
	} else {
		return nil, errors.New("Invalid address")
	}

	return publicKey, nil
}

type APITokenInfoRequest struct {
	Hash helpers.HexBytes `json:"hash"`
}

type APITokenRequest struct {
	Hash       helpers.HexBytes `json:"hash"`
	ReturnType APIReturnType    `json:"returnType"`
}

type APISubscriptionRequest struct {
	Key        []byte           `json:"key,omitempty"`
	Type       SubscriptionType `json:"type,omitempty"`
	ReturnType APIReturnType    `json:"returnType,omitempty"`
}

type APIUnsubscriptionRequest struct {
	Key  []byte           `json:"Key,omitempty"`
	Type SubscriptionType `json:"type,omitempty"`
}

type APIMempoolRequest struct {
	ChainHash helpers.HexBytes `json:"chainHash,omitempty"`
	Page      int              `json:"page,omitempty"`
	Count     int              `json:"count,omitempty"`
}

func (self *APIAccountBaseRequest) ImportFromValues(values *url.Values) (err error) {

	if values.Get("address") != "" {
		self.Address = strings.ReplaceAll(values.Get("address"), " ", "+")
	} else if values.Get("publicKey") != "" {
		self.PublicKey, err = hex.DecodeString(values.Get("publicKey"))
	} else {
		err = errors.New("parameter 'address' or 'hash' was not specified")
	}

	return
}

func (self *APIAccountsKeysByIndexRequest) ImportFromValues(values *url.Values) (err error) {

	if values.Get("indexes") != "" {
		v := strings.Split(values.Get("indexes"), ",")
		self.Indexes = make([]uint64, len(v))
		for i := 0; i < len(v); i++ {
			if self.Indexes[i], err = strconv.ParseUint(v[i], 10, 64); err != nil {
				return err
			}
		}
	} else {
		return errors.New("parameter `indexes` is missing")
	}

	if values.Get("token") != "" {
		if self.Token, err = hex.DecodeString(values.Get("token")); err != nil {
			return err
		}
	}

	return
}