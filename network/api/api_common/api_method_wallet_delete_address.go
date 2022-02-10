package api_common

import (
	"errors"
	"github.com/vmihailenco/msgpack/v5"
	"net/http"
	"net/url"
	"pandora-pay/helpers/urldecoder"
	"pandora-pay/network/api/api_common/api_types"
	"pandora-pay/network/websocks/connection"
)

type APIWalletDeleteAddress struct {
	api_types.APIAuthenticateBaseRequest
	APIWalletDeleteAddressBase
}

type APIWalletDeleteAddressBase struct {
	api_types.APIAccountBaseRequest
}

type APIWalletDeleteAddressReply struct {
	Status bool `json:"status" msgpack:"status"`
}

func (api *APICommon) WalletDeleteAddress(r *http.Request, args *APIWalletDeleteAddressBase, reply *APIWalletDeleteAddressReply, authenticated bool) error {
	if !authenticated {
		return errors.New("Invalid User or Password")
	}

	publicKey, err := args.GetPublicKey(true)
	if err != nil {
		return err
	}

	reply.Status, err = api.wallet.RemoveAddressByPublicKey(publicKey, true)
	return err
}

func (api *APICommon) WalletDeleteAddress_http(values url.Values) (interface{}, error) {
	args := &APIWalletDeleteAddress{}
	if err := urldecoder.Decoder.Decode(args, values); err != nil {
		return nil, err
	}
	reply := &APIWalletDeleteAddressReply{}
	return reply, api.WalletDeleteAddress(nil, &args.APIWalletDeleteAddressBase, reply, args.CheckAuthenticated())
}

func (api *APICommon) WalletDeleteAddress_websockets(conn *connection.AdvancedConnection, values []byte) (interface{}, error) {
	args := &APIWalletDeleteAddressBase{}
	if err := msgpack.Unmarshal(values, args); err != nil {
		return nil, err
	}
	reply := &APIWalletDeleteAddressReply{}
	return reply, api.WalletDeleteAddress(nil, args, reply, conn.Authenticated.IsSet())
}
