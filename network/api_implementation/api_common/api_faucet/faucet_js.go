//go:build wasm
// +build wasm

package api_faucet

type Faucet struct {
}

func NewFaucet() (*Faucet, error) {
	return &Faucet{}, nil
}
