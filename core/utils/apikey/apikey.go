package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

func Generate(prefix string) (string, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	encoded := base64.RawURLEncoding.EncodeToString(randomBytes)
	if strings.TrimSpace(prefix) == "" {
		return encoded, nil
	}
	return prefix + "_" + encoded, nil
}

func Hash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func Verify(hash string, raw string) bool {
	return Hash(raw) == hash
}
