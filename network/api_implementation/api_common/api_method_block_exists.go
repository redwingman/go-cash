package api_common

import (
	"net/http"
	"pandora-pay/blockchain"
	"pandora-pay/helpers"
)

type APIBlockExistsRequest struct {
	Hash helpers.Base64 `json:"hash"  msgpack:"hash"`
}

type APIBlockExistsReply struct {
	Exists bool `json:"exists" msgpack:"exists"`
}

func (api *APICommon) GetBlockExists(r *http.Request, args *APIBlockExistsRequest, reply *APIBlockExistsReply) (err error) {
	reply.Exists, err = blockchain.Blockchain.OpenExistsBlock(args.Hash)
	return
}
