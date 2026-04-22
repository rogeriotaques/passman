package agent

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func shortSocketPath(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "pm-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, "a.sock")
}

func testServer(t *testing.T) (*Server, *Client) {
	t.Helper()
	sock := shortSocketPath(t)
	srv := NewServer(sock)
	go srv.Start()

	client := NewClient(sock)
	for i := 0; i < 20; i++ {
		if client.Ping() {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Cleanup(func() { srv.Stop() })
	return srv, client
}

func TestServer_Ping(t *testing.T) {
	_, client := testServer(t)
	if !client.Ping() {
		t.Error("expected ping to succeed")
	}
}

func TestServer_StoreAndRetrieve(t *testing.T) {
	_, client := testServer(t)

	pw := []byte("supersecret")
	err := client.Store("/vault/test.enc", pw, 5*time.Minute)
	if err != nil {
		t.Fatalf("store: %v", err)
	}

	got, err := client.Retrieve("/vault/test.enc")
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if !bytes.Equal(got, pw) {
		t.Errorf("got %q, want %q", got, pw)
	}
}

func TestServer_RetrieveNotFound(t *testing.T) {
	_, client := testServer(t)

	_, err := client.Retrieve("/vault/nonexistent.enc")
	if err == nil {
		t.Error("expected error for missing entry")
	}
}

func TestServer_RetrieveExpired(t *testing.T) {
	_, client := testServer(t)

	err := client.Store("/vault/test.enc", []byte("secret"), 100*time.Millisecond)
	if err != nil {
		t.Fatalf("store: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	_, err = client.Retrieve("/vault/test.enc")
	if err == nil {
		t.Error("expected error for expired entry")
	}
}

func TestServer_Lock(t *testing.T) {
	_, client := testServer(t)

	_ = client.Store("/vault/a.enc", []byte("pw1"), 5*time.Minute)
	_ = client.Store("/vault/b.enc", []byte("pw2"), 5*time.Minute)

	err := client.Lock()
	if err != nil {
		t.Fatalf("lock: %v", err)
	}

	_, err = client.Retrieve("/vault/a.enc")
	if err == nil {
		t.Error("expected error after lock for entry a")
	}
	_, err = client.Retrieve("/vault/b.enc")
	if err == nil {
		t.Error("expected error after lock for entry b")
	}
}

func TestServer_StoreRefreshesTTL(t *testing.T) {
	_, client := testServer(t)

	_ = client.Store("/vault/test.enc", []byte("secret"), 300*time.Millisecond)
	time.Sleep(200 * time.Millisecond)

	_ = client.Store("/vault/test.enc", []byte("secret"), 300*time.Millisecond)
	time.Sleep(200 * time.Millisecond)

	got, err := client.Retrieve("/vault/test.enc")
	if err != nil {
		t.Fatalf("retrieve after refresh: %v", err)
	}
	if !bytes.Equal(got, []byte("secret")) {
		t.Errorf("got %q, want %q", got, "secret")
	}
}

func TestServer_MultipleVaults(t *testing.T) {
	_, client := testServer(t)

	_ = client.Store("/vault/a.enc", []byte("pw_a"), 5*time.Minute)
	_ = client.Store("/vault/b.enc", []byte("pw_b"), 5*time.Minute)

	gotA, _ := client.Retrieve("/vault/a.enc")
	gotB, _ := client.Retrieve("/vault/b.enc")

	if !bytes.Equal(gotA, []byte("pw_a")) {
		t.Errorf("vault a: got %q, want %q", gotA, "pw_a")
	}
	if !bytes.Equal(gotB, []byte("pw_b")) {
		t.Errorf("vault b: got %q, want %q", gotB, "pw_b")
	}
}

func TestServer_AutoShutdown(t *testing.T) {
	sock := shortSocketPath(t)
	srv := NewServer(sock)
	go srv.Start()

	client := NewClient(sock)
	for i := 0; i < 20; i++ {
		if client.Ping() {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	_ = client.Store("/vault/test.enc", []byte("secret"), 100*time.Millisecond)

	select {
	case <-srv.Done():
	case <-time.After(10 * time.Second):
		srv.Stop()
		t.Fatal("server did not auto-shutdown after all entries expired")
	}
}

func TestClient_PingNoServer(t *testing.T) {
	sock := shortSocketPath(t)
	client := NewClient(sock)
	if client.Ping() {
		t.Error("expected ping to fail when no server is running")
	}
}

func TestServer_ZerosPasswordOnLock(t *testing.T) {
	srv, client := testServer(t)

	_ = client.Store("/vault/test.enc", []byte("secret"), 5*time.Minute)

	_ = client.Lock()

	srv.mu.Lock()
	defer srv.mu.Unlock()
	if len(srv.entries) != 0 {
		t.Errorf("expected empty entries after lock, got %d", len(srv.entries))
	}
}
