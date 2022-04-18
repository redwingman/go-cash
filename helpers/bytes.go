package helpers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
)

func RandomBytes(size int) []byte {
	a := make([]byte, size)
	_, err := rand.Read(a)
	for err != nil {
		_, err = rand.Read(a)
	}
	return a
}

func EmptyBytes(size int) []byte {
	return make([]byte, size)
}

func CloneBytes(a []byte) []byte {
	if a == nil {
		return nil
	}
	out := make([]byte, len(a))
	copy(out, a)
	return out
}

func DecodeHex(a string) []byte {
	out, err := hex.DecodeString(a)
	if err != nil {
		panic(err)
	}
	return out
}

func DecodeBase64(a string) []byte {
	out, err := base64.StdEncoding.DecodeString(a)
	if err != nil {
		panic(err)
	}
	return out
}
