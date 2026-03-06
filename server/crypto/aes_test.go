package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptAESGCM(t *testing.T) {
	// 32-byte key for AES-256 (simple test key to avoid false positive secret scanning warnings)
	key := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{"empty", []byte{}},
		{"short", []byte("hello")},
		{"medium", []byte("this is a longer test message for encryption")},
		{"with special chars", []byte("hello\x00world\ntest\ttab")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := EncryptAESGCM(tt.plaintext, key)
			if err != nil {
				t.Fatalf("EncryptAESGCM() error = %v", err)
			}

			// Encrypted should be longer than plaintext (nonce + auth tag)
			if len(encrypted) <= len(tt.plaintext) {
				t.Errorf("encrypted length %d should be > plaintext length %d", len(encrypted), len(tt.plaintext))
			}

			decrypted, err := DecryptAESGCM(encrypted, key)
			if err != nil {
				t.Fatalf("DecryptAESGCM() error = %v", err)
			}

			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("DecryptAESGCM() = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestDecryptAESGCM_InvalidKey(t *testing.T) {
	key := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	wrongKey := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	plaintext := []byte("secret message")
	encrypted, err := EncryptAESGCM(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptAESGCM() error = %v", err)
	}

	_, err = DecryptAESGCM(encrypted, wrongKey)
	if err == nil {
		t.Error("DecryptAESGCM() with wrong key should fail")
	}
}

func TestDecryptAESGCM_MalformedCiphertext(t *testing.T) {
	key := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	// Too short to contain nonce
	_, err := DecryptAESGCM([]byte("short"), key)
	if err == nil {
		t.Error("DecryptAESGCM() with short ciphertext should fail")
	}
}

func TestEncryptAESGCM_InvalidKeyLength(t *testing.T) {
	// AES requires 16, 24, or 32 byte keys
	_, err := EncryptAESGCM([]byte("test"), "short")
	if err == nil {
		t.Error("EncryptAESGCM() with invalid key length should fail")
	}
}

func TestEncryptAESGCM_UniqueNonces(t *testing.T) {
	key := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	plaintext := []byte("same message")

	encrypted1, _ := EncryptAESGCM(plaintext, key)
	encrypted2, _ := EncryptAESGCM(plaintext, key)

	// Same plaintext should produce different ciphertext due to random nonce
	if bytes.Equal(encrypted1, encrypted2) {
		t.Error("encrypting same plaintext twice should produce different ciphertext")
	}
}
