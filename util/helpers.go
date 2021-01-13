package util

import (
	"crypto/rand"
	"crypto/sha256"
)

func Make32ByteSecret(input string) ([]byte, error) {
	var key = make([]byte, 32)

	if input != "" {
		var hash = sha256.Sum256([]byte(input))
		return hash[:], nil
	}

	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}

	return key, nil
}
