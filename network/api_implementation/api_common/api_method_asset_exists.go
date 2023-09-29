package api_common

import (
	"net/http"
	"pandora-pay/blockchain"
	"pandora-pay/helpers"
)

type APIAssetExistsRequest struct {
	Hash helpers.Base64 `json:"hash"  msgpack:"hash"`
}

type APIAssetExistsReply struct {
	Exists bool `json:"exists" msgpack:"exists"`
}

func (api *APICommon) GetAssetExists(r *http.Request, args *APIAssetExistsRequest, reply *APIAssetExistsReply) (err error) {
	reply.Exists, err = blockchain.Blockchain.OpenExistsBlock(args.Hash)
	return
}
