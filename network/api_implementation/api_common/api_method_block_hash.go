package api_common

import (
	"net/http"
	"pandora-pay/blockchain"
)

type APIBlockHashRequest struct {
	Height uint64 `json:"height" msgpack:"height"`
}

type APIBlockHashReply struct {
	Hash []byte `json:"hash" msgpack:"hash"`
}

func (api *APICommon) GetBlockHash(r *http.Request, args *APIBlockHashRequest, reply *APIBlockHashReply) (err error) {
	reply.Hash, err = blockchain.Blockchain.OpenLoadBlockHash(args.Height)
	return
}
