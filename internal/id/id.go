package id

import (
	"crypto/rand"
	"encoding/base64"
)

func New() string {
	b := make([]byte, 18)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
