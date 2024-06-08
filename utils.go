package main

import (
	"crypto/rand"
	"encoding/base64"
	"log"
)

func generateState() string {
	// Generate a random 32-byte array
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("Failed to generate random state: %v", err)
	}

	// Encode the bytes as a base64 URL-encoded string
	return base64.URLEncoding.EncodeToString(b)
}
