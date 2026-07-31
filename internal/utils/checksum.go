package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

func CalculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()

	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	checksum := hasher.Sum(nil)
	return hex.EncodeToString(checksum), nil
}
