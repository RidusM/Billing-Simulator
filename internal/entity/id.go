package entity

import (
	"crypto/rand"
	"encoding/hex"
)

func GeneratePublicID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}
