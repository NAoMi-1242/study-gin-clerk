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
)

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

// Encrypt encrypts plainText using AES-256-GCM and returns a Base64-encoded string (nonce + ciphertext).
func Encrypt(plainText, keyStr string) (string, error) {
	keyBytes, err := parseKey(keyStr)
	if err != nil {
		return "", err
	}

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
	cipherText := gcm.Seal(nonce, nonce, []byte(plainText), nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// Decrypt decrypts a Base64-encoded string (nonce + ciphertext) using AES-256-GCM.
func Decrypt(cipherTextBase64, keyStr string) (string, error) {
	keyBytes, err := parseKey(keyStr)
	if err != nil {
		return "", err
	}

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

// MaskAPIKey masks an API key for safe UI display (e.g., "sk-or-v1-...abcd").
func MaskAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 8 {
		return "****"
	}
	if strings.HasPrefix(key, "sk-or-v1-") && len(key) > 13 {
		return "sk-or-v1-..." + key[len(key)-4:]
	}
	if strings.HasPrefix(key, "sk-") && len(key) > 8 {
		return "sk-..." + key[len(key)-4:]
	}
	if len(key) > 12 {
		return key[:6] + "..." + key[len(key)-4:]
	}
	return key[:2] + "..." + key[len(key)-2:]
}

