package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"io"
)

// Encryption allows encrypting/decrypting a sequence of bytes.
type Encryption interface {
	Encrypt([]byte) ([]byte, error)
	Decrypt([]byte) ([]byte, error)
}

type encryption struct {
	key   []byte
	block cipher.Block
}

// New creates a new encryption with the given passphrase.
func New(passphrase string) (Encryption, error) {
	key := keyFor(passphrase)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &encryption{
		key:   key,
		block: block,
	}, nil
}

func keyFor(passphrase string) []byte {
	h := sha256.New()
	h.Write([]byte(passphrase))
	return h.Sum(nil)
}

func (e *encryption) Encrypt(src []byte) ([]byte, error) {
	// +1 = need to account for the padding byte as well.
	padding := len(e.key) - (len(src)+1)%len(e.key)
	nsrc := append([]byte{byte(padding)}, make([]byte, padding)...)
	nsrc = append(nsrc, src...)
	// TODO(andreich): HMAC as well.
	size := e.block.BlockSize()
	dst := make([]byte, size+len(nsrc))
	if _, err := io.ReadFull(rand.Reader, dst[:size]); err != nil {
		return nil, err
	}
	mode := cipher.NewCBCEncrypter(e.block, dst[:size])
	mode.CryptBlocks(dst[size:], nsrc)
	return dst, nil
}

func (e *encryption) Decrypt(src []byte) ([]byte, error) {
	size := e.block.BlockSize()
	mode := cipher.NewCBCDecrypter(e.block, src[:size])
	mode.CryptBlocks(src[size:], src[size:])
	padding := int(src[size])
	var dst []byte
	// +1 = need to account for the padding byte as well.
	dst = append(dst, src[size+padding+1:]...)
	return dst, nil
}
