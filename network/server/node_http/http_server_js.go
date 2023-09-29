//go:build wasm
// +build wasm

package node_http

import (
	"pandora-pay/network/api_implementation/api_common"
	"pandora-pay/network/api_implementation/api_websockets"
	"pandora-pay/network/websocks"
)

type httpServerType struct {
	ApiWebsockets *api_websockets.APIWebsockets
}

var HttpServer *httpServerType

func NewHttpServer() error {

	apiStore := api_common.NewAPIStore()
	apiCommon, err := api_common.NewAPICommon(apiStore)
	if err != nil {
		return err
	}

	apiWebsockets := api_websockets.NewWebsocketsAPI(apiStore, apiCommon)
	websocks.NewWebsockets(apiWebsockets.GetMap)

	HttpServer = &httpServerType{
		apiWebsockets,
	}

	return nil
}
