package crypto

import (
	"testing"
)

func TestAESCipher_EncryptDecrypt(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		plainText string
	}{
		{
			name:      "32-byte raw key",
			key:       "12345678901234567890123456789012",
			plainText: "sk-ant-api03-test-secret-key",
		},
		{
			name:      "64-char hex key",
			key:       "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			plainText: "AIzaSyTestSecretKeyGoogleGemini",
		},
		{
			name:      "empty plain text",
			key:       "12345678901234567890123456789012",
			plainText: "",
		},
		{
			name:      "unicode plain text",
			key:       "12345678901234567890123456789012",
			plainText: "秘密のAPIキー🔑✨",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cipher, err := NewAESCipher(tt.key)
			if err != nil {
				t.Fatalf("failed to create AESCipher: %v", err)
			}

			encrypted, err := cipher.Encrypt(tt.plainText)
			if err != nil {
				t.Fatalf("Encrypt failed: %v", err)
			}
			if encrypted == "" && tt.plainText != "" {
				t.Fatalf("Encrypt returned empty string")
			}

			decrypted, err := cipher.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decrypt failed: %v", err)
			}

			if decrypted != tt.plainText {
				t.Errorf("got %q, want %q", decrypted, tt.plainText)
			}
		})
	}
}

func TestAESCipher_InvalidKey(t *testing.T) {
	invalidKeys := []string{
		"",
		"too-short",
		"1234567890123456789012345678901",   // 31 bytes
		"123456789012345678901234567890123", // 33 bytes
	}

	for _, key := range invalidKeys {
		_, err := NewAESCipher(key)
		if err == nil {
			t.Errorf("expected error for invalid key %q, got nil", key)
		}
	}
}

func TestAESCipher_DecryptCorrupted(t *testing.T) {
	cipher, err := NewAESCipher("12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("failed to create AESCipher: %v", err)
	}

	// Invalid base64
	_, err = cipher.Decrypt("not-base64!@#$")
	if err == nil {
		t.Errorf("expected error for invalid base64, got nil")
	}

	// Base64 too short (less than nonce size)
	_, err = cipher.Decrypt("AAAA")
	if err == nil {
		t.Errorf("expected error for short cipher text, got nil")
	}
}

