package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

type Cryptobox struct{ aead cipher.AEAD }

func New(key []byte) (*Cryptobox, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cryptobox{aead: aead}, nil
}
func (box *Cryptobox) Seal(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	nonce := make([]byte, box.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(append(nonce, box.aead.Seal(nil, nonce, []byte(plain), nil)...)), nil
}
func (box *Cryptobox) Open(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	blob, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil || len(blob) < box.aead.NonceSize() {
		return "", fmt.Errorf("encrypted secret is invalid")
	}
	plain, err := box.aead.Open(nil, blob[:box.aead.NonceSize()], blob[box.aead.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt secret: %w", err)
	}
	return string(plain), nil
}
