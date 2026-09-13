package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"study-gin-clerk/internal/config"
)

// AESCipher provides AES-256-GCM encryption and decryption with a pre-parsed key.
type AESCipher struct {
	keyBytes []byte
}

// NewAESCipher creates a new AESCipher from configuration, validating and caching the 32-byte key.
func NewAESCipher(cfg config.Config) (*AESCipher, error) {
	keyBytes, err := parseKey(cfg.EncryptionKey)
	if err != nil {
		return nil, err
	}
	return &AESCipher{keyBytes: keyBytes}, nil
}

// Encrypt encrypts plainText using AES-256-GCM and returns a Base64-encoded string (nonce + ciphertext).
func (c *AESCipher) Encrypt(plainText string) (string, error) {
	return encryptWithBytes([]byte(plainText), c.keyBytes)
}

// Decrypt decrypts a Base64-encoded string (nonce + ciphertext) using AES-256-GCM.
func (c *AESCipher) Decrypt(cipherTextBase64 string) (string, error) {
	return decryptWithBytes(cipherTextBase64, c.keyBytes)
}

// parseKey parses a 32-byte key from either a 64-char hex string or a 32-byte string.
func parseKey(keyStr string) ([]byte, error) {
	keyStr = strings.TrimSpace(keyStr)
	if len(keyStr) == 64 {
		decoded, err := hex.DecodeString(keyStr)
		if err == nil && len(decoded) == 32 {
			return decoded, nil
		}
	}
	if len(keyStr) == 32 {
		return []byte(keyStr), nil
	}
	return nil, fmt.Errorf("encryption key must be 32 bytes (or 64 hex characters)")
}

func encryptWithBytes(plainText, keyBytes []byte) (string, error) {
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// gcm.Seal appends the ciphertext to nonce (so nonce is at the beginning)
	cipherText := gcm.Seal(nonce, nonce, plainText, nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

func decryptWithBytes(cipherTextBase64 string, keyBytes []byte) (string, error) {
	data, err := base64.StdEncoding.DecodeString(cipherTextBase64)
	if err != nil {
		return "", fmt.Errorf("invalid base64 cipher text: %w", err)
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("cipher text is too short")
	}

	nonce, cipherText := data[:nonceSize], data[nonceSize:]
	plainTextBytes, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed (invalid key or corrupted data): %w", err)
	}

	return string(plainTextBytes), nil
}

// Encrypt encrypts plainText using AES-256-GCM and returns a Base64-encoded string.
func Encrypt(plainText, keyStr string) (string, error) {
	keyBytes, err := parseKey(keyStr)
	if err != nil {
		return "", err
	}
	return encryptWithBytes([]byte(plainText), keyBytes)
}

// Decrypt decrypts a Base64-encoded string using AES-256-GCM.
func Decrypt(cipherTextBase64, keyStr string) (string, error) {
	keyBytes, err := parseKey(keyStr)
	if err != nil {
		return "", err
	}
	return decryptWithBytes(cipherTextBase64, keyBytes)
}
