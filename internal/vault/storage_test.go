package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rogerio/passman/internal/crypto"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	return &Store{
		Path:      filepath.Join(dir, "subdir", "vault.enc"),
		KDFParams: crypto.FastKDFParams(),
	}
}

func TestStore_Init_CreatesFile(t *testing.T) {
	s := testStore(t)
	err := s.Init([]byte("master"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.Exists() {
		t.Error("vault file should exist after init")
	}
}

func TestStore_Init_CreatesDirectory(t *testing.T) {
	s := testStore(t)
	_ = s.Init([]byte("master"))

	dir := filepath.Dir(s.Path)
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected a directory")
	}
}

func TestStore_Init_AlreadyExists_Error(t *testing.T) {
	s := testStore(t)
	_ = s.Init([]byte("master"))
	err := s.Init([]byte("master"))
	if err == nil {
		t.Error("expected error when vault already exists")
	}
}

func TestStore_SaveLoad_RoundTrip(t *testing.T) {
	s := testStore(t)
	password := []byte("master")
	_ = s.Init(password)

	v, err := s.Load(password)
	if err != nil {
		t.Fatalf("load after init: %v", err)
	}

	_ = v.Add(Entry{Name: "github", Value: "secret"})

	if err := s.Save(v, password); err != nil {
		t.Fatalf("save: %v", err)
	}

	v2, err := s.Load(password)
	if err != nil {
		t.Fatalf("load after save: %v", err)
	}

	if len(v2.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(v2.Entries))
	}
	if v2.Entries[0].Name != "github" {
		t.Errorf("expected name 'github', got %q", v2.Entries[0].Name)
	}
	if v2.Entries[0].Value != "secret" {
		t.Errorf("expected value 'secret', got %q", v2.Entries[0].Value)
	}
}

func TestStore_Load_WrongPassword(t *testing.T) {
	s := testStore(t)
	_ = s.Init([]byte("correct"))

	_, err := s.Load([]byte("wrong"))
	if err == nil {
		t.Error("expected error for wrong password")
	}
}

func TestStore_Load_CorruptedFile(t *testing.T) {
	s := testStore(t)
	_ = s.Init([]byte("master"))

	_ = os.WriteFile(s.Path, []byte("garbage data"), 0600)

	_, err := s.Load([]byte("master"))
	if err == nil {
		t.Error("expected error for corrupted file")
	}
}

func TestStore_Load_NonexistentFile(t *testing.T) {
	s := testStore(t)
	_, err := s.Load([]byte("master"))
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestStore_Exists_False(t *testing.T) {
	s := testStore(t)
	if s.Exists() {
		t.Error("should not exist before init")
	}
}

func TestStore_Load_UnsupportedVersion(t *testing.T) {
	s := testStore(t)
	_ = s.Init([]byte("master"))

	data, _ := os.ReadFile(s.Path)
	modified := strings.Replace(string(data), `"version":2`, `"version":99`, 1)
	os.WriteFile(s.Path, []byte(modified), 0600)

	_, err := s.Load([]byte("master"))
	if err == nil {
		t.Error("expected error for unsupported version")
	}
}

func TestStore_FilePermissions(t *testing.T) {
	s := testStore(t)
	_ = s.Init([]byte("master"))

	info, err := os.Stat(s.Path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("expected file permissions 0600, got %o", perm)
	}
}

func TestStore_AtomicWrite_NoTempFileLeftOver(t *testing.T) {
	s := testStore(t)
	_ = s.Init([]byte("master"))

	v, _ := s.Load([]byte("master"))
	_ = v.Add(Entry{Name: "test", Value: "secret"})
	_ = s.Save(v, []byte("master"))

	entries, err := os.ReadDir(filepath.Dir(s.Path))
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != "vault.enc" && e.Name() != "vault.enc.lock" {
			t.Errorf("unexpected file left over: %s", e.Name())
		}
	}
}

func TestStore_EmptyPassword(t *testing.T) {
	s := testStore(t)
	err := s.Init([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	v, err := s.Load([]byte(""))
	if err != nil {
		t.Fatalf("load with empty password: %v", err)
	}

	_ = v.Add(Entry{Name: "test", Value: "secret"})
	if err := s.Save(v, []byte("")); err != nil {
		t.Fatalf("save with empty password: %v", err)
	}

	v2, err := s.Load([]byte(""))
	if err != nil {
		t.Fatalf("reload with empty password: %v", err)
	}
	if len(v2.Entries) != 1 || v2.Entries[0].Value != "secret" {
		t.Errorf("unexpected vault state: %+v", v2)
	}
}
