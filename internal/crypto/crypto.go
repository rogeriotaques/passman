package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
)

var ErrDecryptionFailed = errors.New("decryption failed: wrong password or corrupted data")

const saltLen = 16
const nonceLen = 12

func DeriveKey(password []byte, params *KDFParams) ([]byte, error) {
	if params.Salt == nil {
		params.Salt = make([]byte, saltLen)
		if _, err := io.ReadFull(rand.Reader, params.Salt); err != nil {
			return nil, err
		}
	}
	key := argon2.IDKey(password, params.Salt, params.Time, params.Memory, params.Threads, params.KeyLen)
	return key, nil
}

func Encrypt(plaintext, key []byte, params KDFParams) (*EncryptedBlob, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, nonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return &EncryptedBlob{
		KDF:        params,
		Nonce:      nonce,
		Ciphertext: ciphertext,
	}, nil
}

func Decrypt(blob *EncryptedBlob, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	plaintext, err := gcm.Open(nil, blob.Nonce, blob.Ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}
