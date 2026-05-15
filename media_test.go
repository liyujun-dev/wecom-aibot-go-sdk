package aibot_test

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"testing"

	aibot "github.com/liyujun/wecom-aibot-go-sdk"
)

func pkcs7Pad32(data []byte) []byte {
	blockSize := 32
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}

func encryptAES256CBC(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	iv := key[:16]
	padded := pkcs7Pad32(plaintext)
	ciphertext := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, padded)
	return ciphertext, nil
}

func TestDecryptFile(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}

	plaintext := []byte("hello world, this is a test file content for wecom decryption")

	ciphertext, err := encryptAES256CBC(plaintext, key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	aesKeyB64 := base64.StdEncoding.EncodeToString(key)

	result, err := aibot.DecryptFile(ciphertext, aesKeyB64)
	if err != nil {
		t.Fatalf("DecryptFile: %v", err)
	}

	if !bytes.Equal(result, plaintext) {
		t.Errorf("decrypted data mismatch: got %q, want %q", result, plaintext)
	}
}

func TestDecryptFileEmptyInput(t *testing.T) {
	_, err := aibot.DecryptFile(nil, "dGVzdA==")
	if err == nil {
		t.Error("expected error for nil input, got nil")
	}

	_, err = aibot.DecryptFile([]byte{}, "dGVzdA==")
	if err == nil {
		t.Error("expected error for empty input, got nil")
	}
}

func TestDecryptFileBadKey(t *testing.T) {
	_, err := aibot.DecryptFile([]byte("data"), "")
	if err == nil {
		t.Error("expected error for empty aesKey, got nil")
	}
}
