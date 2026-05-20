package vault

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/rogerio/passman/internal/crypto"
)

const currentVersion = 2

var (
	ErrVaultExists        = errors.New("vault already exists")
	ErrVaultNotFound      = errors.New("vault not found")
	ErrUnsupportedVersion = errors.New("unsupported vault version")
)

type Store struct {
	Path      string
	KDFParams crypto.KDFParams
}

func (s *Store) Exists() bool {
	_, err := os.Stat(s.Path)
	return err == nil
}

func (s *Store) Init(password []byte) error {
	if s.Exists() {
		return ErrVaultExists
	}

	if err := os.MkdirAll(filepath.Dir(s.Path), 0700); err != nil {
		return fmt.Errorf("create vault directory: %w", err)
	}

	v := &Vault{}
	return s.Save(v, password)
}

func (s *Store) lock() (*os.File, error) {
	lockPath := s.Path + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0700); err != nil {
		return nil, fmt.Errorf("create lock directory: %w", err)
	}
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, fmt.Errorf("acquire lock: %w", err)
	}
	return f, nil
}

func unlock(f *os.File) {
	syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	f.Close()
}

func (s *Store) Save(v *Vault, password []byte) error {
	lf, err := s.lock()
	if err != nil {
		return err
	}
	defer unlock(lf)

	plaintext, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal vault: %w", err)
	}
	defer ZeroBytes(plaintext)

	params := s.kdfParams()
	key, err := crypto.DeriveKey(password, &params)
	if err != nil {
		return fmt.Errorf("derive key: %w", err)
	}
	defer ZeroBytes(key)

	blob, err := crypto.Encrypt(plaintext, key, params)
	if err != nil {
		return fmt.Errorf("encrypt vault: %w", err)
	}
	blob.Version = currentVersion

	data, err := json.Marshal(blob)
	if err != nil {
		return fmt.Errorf("marshal blob: %w", err)
	}

	tmpPath := s.Path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("write vault file: %w", err)
	}
	if err := os.Rename(tmpPath, s.Path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename vault file: %w", err)
	}

	return nil
}

func (s *Store) Load(password []byte) (*Vault, error) {
	lf, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer unlock(lf)

	data, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrVaultNotFound
		}
		return nil, fmt.Errorf("read vault file: %w", err)
	}

	var blob crypto.EncryptedBlob
	if err := json.Unmarshal(data, &blob); err != nil {
		return nil, fmt.Errorf("parse vault file: %w", err)
	}

	if blob.Version != currentVersion {
		return nil, fmt.Errorf("%w: got %d, expected %d", ErrUnsupportedVersion, blob.Version, currentVersion)
	}

	key, err := crypto.DeriveKey(password, &blob.KDF)
	if err != nil {
		return nil, fmt.Errorf("derive key: %w", err)
	}
	defer ZeroBytes(key)

	plaintext, err := crypto.Decrypt(&blob, key)
	if err != nil {
		return nil, err
	}
	defer ZeroBytes(plaintext)

	var v Vault
	if err := json.Unmarshal(plaintext, &v); err != nil {
		return nil, fmt.Errorf("parse vault data: %w", err)
	}

	return &v, nil
}

func (s *Store) kdfParams() crypto.KDFParams {
	if s.KDFParams.KeyLen > 0 {
		return s.KDFParams
	}
	return crypto.DefaultKDFParams()
}

func ZeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
