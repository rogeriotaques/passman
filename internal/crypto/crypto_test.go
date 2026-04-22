package crypto

import (
	"bytes"
	"testing"
)

func testParams() *KDFParams {
	p := FastKDFParams()
	return &p
}

func TestDeriveKey_GeneratesSaltWhenNil(t *testing.T) {
	params := testParams()
	params.Salt = nil
	key, err := DeriveKey([]byte("password"), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(params.Salt) != 16 {
		t.Errorf("expected 16-byte salt, got %d", len(params.Salt))
	}
	if len(key) != 32 {
		t.Errorf("expected 32-byte key, got %d", len(key))
	}
}

func TestDeriveKey_Deterministic(t *testing.T) {
	params := testParams()
	params.Salt = bytes.Repeat([]byte{0xAB}, 16)

	key1, err := DeriveKey([]byte("password"), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	params2 := testParams()
	params2.Salt = bytes.Repeat([]byte{0xAB}, 16)

	key2, err := DeriveKey([]byte("password"), params2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(key1, key2) {
		t.Error("same password + salt should produce same key")
	}
}

func TestDeriveKey_DifferentPasswords(t *testing.T) {
	salt := bytes.Repeat([]byte{0xAB}, 16)

	p1 := testParams()
	p1.Salt = salt
	key1, _ := DeriveKey([]byte("password1"), p1)

	p2 := testParams()
	p2.Salt = make([]byte, 16)
	copy(p2.Salt, salt)
	key2, _ := DeriveKey([]byte("password2"), p2)

	if bytes.Equal(key1, key2) {
		t.Error("different passwords should produce different keys")
	}
}

func TestDeriveKey_DifferentSalts(t *testing.T) {
	p1 := testParams()
	p1.Salt = bytes.Repeat([]byte{0x01}, 16)
	key1, _ := DeriveKey([]byte("password"), p1)

	p2 := testParams()
	p2.Salt = bytes.Repeat([]byte{0x02}, 16)
	key2, _ := DeriveKey([]byte("password"), p2)

	if bytes.Equal(key1, key2) {
		t.Error("different salts should produce different keys")
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	plaintext := []byte("my secret data")
	key := bytes.Repeat([]byte{0xAA}, 32)
	params := FastKDFParams()
	params.Salt = bytes.Repeat([]byte{0xBB}, 16)

	blob, err := Encrypt(plaintext, key, params)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	got, err := Decrypt(blob, key)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if !bytes.Equal(got, plaintext) {
		t.Errorf("expected %q, got %q", plaintext, got)
	}
}

func TestEncryptDecrypt_EmptyPlaintext(t *testing.T) {
	key := bytes.Repeat([]byte{0xAA}, 32)
	params := FastKDFParams()
	params.Salt = bytes.Repeat([]byte{0xBB}, 16)

	blob, err := Encrypt([]byte{}, key, params)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	got, err := Decrypt(blob, key)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("expected empty plaintext, got %q", got)
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	plaintext := []byte("secret")
	key := bytes.Repeat([]byte{0xAA}, 32)
	wrongKey := bytes.Repeat([]byte{0xBB}, 32)
	params := FastKDFParams()
	params.Salt = bytes.Repeat([]byte{0xCC}, 16)

	blob, err := Encrypt(plaintext, key, params)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	_, err = Decrypt(blob, wrongKey)
	if err == nil {
		t.Error("expected error when decrypting with wrong key")
	}
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	plaintext := []byte("secret")
	key := bytes.Repeat([]byte{0xAA}, 32)
	params := FastKDFParams()
	params.Salt = bytes.Repeat([]byte{0xCC}, 16)

	blob, err := Encrypt(plaintext, key, params)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	blob.Ciphertext[0] ^= 0xFF

	_, err = Decrypt(blob, key)
	if err == nil {
		t.Error("expected error when ciphertext is tampered")
	}
}

func TestDecrypt_TamperedNonce(t *testing.T) {
	plaintext := []byte("secret")
	key := bytes.Repeat([]byte{0xAA}, 32)
	params := FastKDFParams()
	params.Salt = bytes.Repeat([]byte{0xCC}, 16)

	blob, err := Encrypt(plaintext, key, params)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	blob.Nonce[0] ^= 0xFF

	_, err = Decrypt(blob, key)
	if err == nil {
		t.Error("expected error when nonce is tampered")
	}
}

func TestEncrypt_UniqueNonces(t *testing.T) {
	key := bytes.Repeat([]byte{0xAA}, 32)
	params := FastKDFParams()
	params.Salt = bytes.Repeat([]byte{0xBB}, 16)

	blob1, _ := Encrypt([]byte("same"), key, params)
	blob2, _ := Encrypt([]byte("same"), key, params)

	if bytes.Equal(blob1.Nonce, blob2.Nonce) {
		t.Error("two encryptions should produce different nonces")
	}
}

func TestEncryptedBlob_Version(t *testing.T) {
	key := bytes.Repeat([]byte{0xAA}, 32)
	params := FastKDFParams()
	params.Salt = bytes.Repeat([]byte{0xBB}, 16)

	blob, _ := Encrypt([]byte("data"), key, params)
	if blob.Version != 1 {
		t.Errorf("expected version 1, got %d", blob.Version)
	}
}
