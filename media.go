// Package aibot provides a Go SDK for WeChat Work (WeCom) AI Bot WebSocket API.
package aibot

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"
)

// DecryptFile decrypts an AES-256-CBC encrypted file buffer.
// The aesKeyB64 parameter is a Base64-encoded AES key from the WeCom message body
// (image.aeskey or file.aeskey fields). The IV is the first 16 bytes of the decoded key.
func DecryptFile(encrypted []byte, aesKeyB64 string) ([]byte, error) {
	if len(encrypted) == 0 {
		return nil, errors.New("decryptFile: encrypted buffer is empty or not provided")
	}
	if aesKeyB64 == "" {
		return nil, errors.New("decryptFile: aesKey must be a non-empty string")
	}

	key, err := base64.StdEncoding.DecodeString(aesKeyB64)
	if err != nil {
		return nil, fmt.Errorf("decryptFile: failed to decode aesKey: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("decryptFile: failed to create cipher: %w", err)
	}

	iv := key[:16]
	if len(encrypted)%block.BlockSize() != 0 {
		return nil, fmt.Errorf("decryptFile: ciphertext is not a multiple of block size (%d)", block.BlockSize())
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(encrypted))
	mode.CryptBlocks(decrypted, encrypted)

	padLen := int(decrypted[len(decrypted)-1])
	if padLen < 1 || padLen > 32 || padLen > len(decrypted) {
		return nil, fmt.Errorf("decryptFile: invalid PKCS#7 padding value: %d", padLen)
	}
	for i := len(decrypted) - padLen; i < len(decrypted); i++ {
		if decrypted[i] != byte(padLen) {
			return nil, errors.New("decryptFile: invalid PKCS#7 padding: padding bytes mismatch")
		}
	}

	result := make([]byte, len(decrypted)-padLen)
	copy(result, decrypted[:len(decrypted)-padLen])
	return result, nil
}
