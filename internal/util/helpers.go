package util

import (
	"crypto/rand"
	"crypto/sha256"
	"log"
	"os"
	"strings"
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

func Env(envName, defaultValue string) string {
	var val, exists = os.LookupEnv(envName)

	if !exists {
		log.Printf("WARNING: '%s' is not set. Using default value ('%s').", envName, defaultValue)
		return defaultValue
	}

	return strings.TrimSpace(val)
}
