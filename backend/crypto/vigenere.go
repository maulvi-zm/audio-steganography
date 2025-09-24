// Package crypto contains Vigenère Encryption and Decryption
package crypto

import (
	"fmt"
)

type ExtendedVigenere struct {
	key []byte
}

func NewExtendedVigenere(key string) *ExtendedVigenere {
	return &ExtendedVigenere{
		key: []byte(key),
	}
}

func (ev *ExtendedVigenere) Encrypt(plaintext []byte) []byte {
	if len(ev.key) == 0 {
		return plaintext
	}

	ciphertext := make([]byte, len(plaintext))
	keyLen := len(ev.key)

	for i, char := range plaintext {
		keyChar := ev.key[i%keyLen]
		// Extended Vigenère: (P + K) mod 256
		ciphertext[i] = byte((int(char) + int(keyChar)) % 256)
	}

	return ciphertext
}

func (ev *ExtendedVigenere) Decrypt(ciphertext []byte) []byte {
	if len(ev.key) == 0 {
		return ciphertext
	}

	plaintext := make([]byte, len(ciphertext))
	keyLen := len(ev.key)

	for i, char := range ciphertext {
		keyChar := ev.key[i%keyLen]
		// Extended Vigenère: (C - K + 256) mod 256
		plaintext[i] = byte((int(char) - int(keyChar) + 256) % 256)
	}

	return plaintext
}

// ValidateKey validates if the key is suitable for Extended Vigenère
func ValidateKey(key string) error {
	if len(key) == 0 {
		return fmt.Errorf("key cannot be empty")
	}
	if len(key) > 256 {
		return fmt.Errorf("key length cannot exceed 256 characters")
	}
	return nil
}
