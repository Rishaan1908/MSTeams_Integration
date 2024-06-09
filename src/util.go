package main

import (
	"crypto/rand"
	"encoding/base64"
)

// Ensure session data in the cookie is secure
func GenerateSecureKey(length int) (string, error) {
	key := make([]byte, length)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(key), nil
}
