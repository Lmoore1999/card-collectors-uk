package database

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
)

const (
	Key = "CardCollectorsUK"
)

func DecryptSecret(cipherData []byte) (string, error) {
	block, err := aes.NewCipher([]byte(Key))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(cipherData) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce := cipherData[:nonceSize]
	ciphertext := cipherData[nonceSize:]

	plainText, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plainText), nil
}
