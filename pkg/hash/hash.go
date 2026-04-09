package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func ComputeHMAC(data []byte, key string) string {
	if key == "" {
		return ""
	}
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func VerifyHMAC(data []byte, hash, key string) bool {
	if key == "" {
		return true
	}
	expectedHash := ComputeHMAC(data, key)
	return hmac.Equal([]byte(hash), []byte(expectedHash))
}
