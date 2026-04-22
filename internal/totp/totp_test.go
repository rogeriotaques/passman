package totp

import (
	"testing"
	"time"
)

func TestGenerate_RFC6238_Vectors(t *testing.T) {
	// RFC 6238 test vector: secret = "12345678901234567890" (ASCII), SHA1
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ" // base32 of "12345678901234567890"

	tests := []struct {
		time     int64
		expected string
	}{
		{59, "287082"},
		{1111111109, "081804"},
		{1111111111, "050471"},
		{1234567890, "005924"},
		{2000000000, "279037"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			ts := time.Unix(tc.time, 0)
			code, err := GenerateAt(secret, ts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if code != tc.expected {
				t.Errorf("at time %d: expected %s, got %s", tc.time, tc.expected, code)
			}
		})
	}
}

func TestGenerate_InvalidBase32(t *testing.T) {
	_, err := GenerateAt("not-valid-base32!!!", time.Now())
	if err == nil {
		t.Error("expected error for invalid base32 secret")
	}
}

func TestGenerate_RetursSixDigits(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	code, err := GenerateAt(secret, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(code) != 6 {
		t.Errorf("expected 6-digit code, got %d digits: %s", len(code), code)
	}
}

func TestTimeRemaining(t *testing.T) {
	ts := time.Unix(15, 0) // 15 seconds into a 30s period
	remaining := TimeRemaining(ts)
	if remaining != 15 {
		t.Errorf("expected 15 seconds remaining, got %d", remaining)
	}
}

func TestTimeRemaining_AtBoundary(t *testing.T) {
	ts := time.Unix(30, 0) // exactly at period boundary
	remaining := TimeRemaining(ts)
	if remaining != 30 {
		t.Errorf("expected 30 seconds remaining, got %d", remaining)
	}
}
