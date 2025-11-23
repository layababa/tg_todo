package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"strings"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	// Generate a valid 32-byte key
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		t.Fatal(err)
	}
	keyBase64 := base64.StdEncoding.EncodeToString(key)

	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "simple text",
			plaintext: "Hello, World!",
		},
		{
			name:      "notion token",
			plaintext: "secret_1234567890abcdefghijklmnopqrstuvwxyz",
		},
		{
			name:      "long text",
			plaintext: strings.Repeat("a", 1000),
		},
		{
			name:      "unicode text",
			plaintext: "你好世界 🌍",
		},
		{
			name:      "empty string",
			plaintext: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := Encrypt(tt.plaintext, keyBase64)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// Empty plaintext should return empty ciphertext
			if tt.plaintext == "" {
				if ciphertext != "" {
					t.Errorf("Encrypt('') = %v, want empty string", ciphertext)
				}
				return
			}

			// Ciphertext should be different from plaintext
			if ciphertext == tt.plaintext {
				t.Error("Ciphertext should not equal plaintext")
			}

			// Ciphertext should be base64-encoded
			if _, err := base64.StdEncoding.DecodeString(ciphertext); err != nil {
				t.Errorf("Ciphertext is not valid base64: %v", err)
			}

			// Decrypt
			decrypted, err := Decrypt(ciphertext, keyBase64)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			// Decrypted should match original plaintext
			if decrypted != tt.plaintext {
				t.Errorf("Decrypt() = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncrypt_DifferentKeys(t *testing.T) {
	plaintext := "secret_data"

	// Generate two different keys
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	io.ReadFull(rand.Reader, key1)
	io.ReadFull(rand.Reader, key2)

	key1Base64 := base64.StdEncoding.EncodeToString(key1)
	key2Base64 := base64.StdEncoding.EncodeToString(key2)

	// Encrypt with key1
	ciphertext, err := Encrypt(plaintext, key1Base64)
	if err != nil {
		t.Fatal(err)
	}

	// Try to decrypt with key2 (should fail)
	_, err = Decrypt(ciphertext, key2Base64)
	if err != ErrDecryptionFailed {
		t.Errorf("Decrypt with wrong key should return ErrDecryptionFailed, got %v", err)
	}
}

func TestEncrypt_InvalidKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr error
	}{
		{
			name:    "too short key",
			key:     base64.StdEncoding.EncodeToString([]byte("short")),
			wantErr: ErrInvalidKey,
		},
		{
			name:    "too long key",
			key:     base64.StdEncoding.EncodeToString(make([]byte, 64)),
			wantErr: ErrInvalidKey,
		},
		{
			name:    "invalid base64",
			key:     "not-valid-base64!!!",
			wantErr: nil, // Will error on base64 decode, not ErrInvalidKey
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Encrypt("test", tt.key)
			if err == nil {
				t.Error("Encrypt() should return error for invalid key")
			}
		})
	}
}

func TestDecrypt_InvalidCiphertext(t *testing.T) {
	key := make([]byte, 32)
	io.ReadFull(rand.Reader, key)
	keyBase64 := base64.StdEncoding.EncodeToString(key)

	tests := []struct {
		name       string
		ciphertext string
		wantErr    error
	}{
		{
			name:       "invalid base64",
			ciphertext: "not-valid-base64!!!",
			wantErr:    nil, // Will error on base64 decode
		},
		{
			name:       "too short ciphertext",
			ciphertext: base64.StdEncoding.EncodeToString([]byte("short")),
			wantErr:    ErrInvalidCiphertext,
		},
		{
			name:       "corrupted ciphertext",
			ciphertext: base64.StdEncoding.EncodeToString(make([]byte, 50)),
			wantErr:    ErrDecryptionFailed,
		},
		{
			name:       "empty string",
			ciphertext: "",
			wantErr:    nil, // Should return empty string without error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Decrypt(tt.ciphertext, keyBase64)

			if tt.ciphertext == "" {
				if err != nil {
					t.Errorf("Decrypt('') should not return error, got %v", err)
				}
				if result != "" {
					t.Errorf("Decrypt('') = %v, want empty string", result)
				}
				return
			}

			if err == nil {
				t.Error("Decrypt() should return error for invalid ciphertext")
			}
			if tt.wantErr != nil && err != tt.wantErr {
				t.Errorf("Decrypt() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncrypt_Determinism(t *testing.T) {
	key := make([]byte, 32)
	io.ReadFull(rand.Reader, key)
	keyBase64 := base64.StdEncoding.EncodeToString(key)

	plaintext := "test data"

	// Encrypt twice
	ciphertext1, err := Encrypt(plaintext, keyBase64)
	if err != nil {
		t.Fatal(err)
	}

	ciphertext2, err := Encrypt(plaintext, keyBase64)
	if err != nil {
		t.Fatal(err)
	}

	// Ciphertexts should be different (because of random nonce)
	if ciphertext1 == ciphertext2 {
		t.Error("Two encryptions of same plaintext should produce different ciphertexts (due to random nonce)")
	}

	// But both should decrypt to the same plaintext
	decrypted1, _ := Decrypt(ciphertext1, keyBase64)
	decrypted2, _ := Decrypt(ciphertext2, keyBase64)

	if decrypted1 != plaintext || decrypted2 != plaintext {
		t.Error("Both ciphertexts should decrypt to the same plaintext")
	}
}

func TestGenerateKey(t *testing.T) {
	// Generate a key
	keyBase64, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	// Decode to check length
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		t.Fatalf("Generated key is not valid base64: %v", err)
	}

	// Should be 32 bytes
	if len(key) != 32 {
		t.Errorf("GenerateKey() produced key of length %v, want 32", len(key))
	}

	// Generate another key
	keyBase64_2, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	// Two generated keys should be different
	if keyBase64 == keyBase64_2 {
		t.Error("Two generated keys should be different")
	}

	// Generated key should work for encryption
	plaintext := "test"
	ciphertext, err := Encrypt(plaintext, keyBase64)
	if err != nil {
		t.Errorf("Generated key should work for encryption: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, keyBase64)
	if err != nil {
		t.Errorf("Generated key should work for decryption: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Encrypt/Decrypt with generated key failed")
	}
}

func BenchmarkEncrypt(b *testing.B) {
	key := make([]byte, 32)
	io.ReadFull(rand.Reader, key)
	keyBase64 := base64.StdEncoding.EncodeToString(key)
	plaintext := "secret_token_1234567890"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Encrypt(plaintext, keyBase64)
	}
}

func BenchmarkDecrypt(b *testing.B) {
	key := make([]byte, 32)
	io.ReadFull(rand.Reader, key)
	keyBase64 := base64.StdEncoding.EncodeToString(key)
	plaintext := "secret_token_1234567890"
	ciphertext, _ := Encrypt(plaintext, keyBase64)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Decrypt(ciphertext, keyBase64)
	}
}
