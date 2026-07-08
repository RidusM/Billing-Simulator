package entity

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func GeneratePublicID(prefix string) (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate public id: %w", err)
	}
	return prefix + "_" + hex.EncodeToString(b), nil
}
