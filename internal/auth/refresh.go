package auth

import (
	"crypto/rand"
	"encoding/hex"
)

func MakeRefreshToken() string {
	data := make([]byte, 32)

	if _, err := rand.Read(data); err != nil {
		panic(err)
	}

	return hex.EncodeToString(data)
}
