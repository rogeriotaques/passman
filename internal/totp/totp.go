package totp

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

const (
	period = 30
	digits = 6
)

func GenerateAt(secret string, t time.Time) (string, error) {
	key, err := decodeSecret(secret)
	if err != nil {
		return "", fmt.Errorf("invalid TOTP secret: %w", err)
	}

	counter := uint64(t.Unix()) / period

	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	hash := mac.Sum(nil)

	offset := hash[len(hash)-1] & 0x0F
	code := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7FFFFFFF

	mod := uint32(1)
	for i := 0; i < digits; i++ {
		mod *= 10
	}

	return fmt.Sprintf("%0*d", digits, code%mod), nil
}

func Generate(secret string) (string, error) {
	return GenerateAt(secret, time.Now())
}

func TimeRemaining(t time.Time) int {
	elapsed := t.Unix() % period
	if elapsed == 0 {
		return period
	}
	return int(period - elapsed)
}

func decodeSecret(secret string) ([]byte, error) {
	s := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(secret), " ", ""))
	for len(s)%8 != 0 {
		s += "="
	}
	return base32.StdEncoding.DecodeString(s)
}
