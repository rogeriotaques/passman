package generate

import (
	"strings"
	"testing"
)

const (
	lowerChars  = "abcdefghijklmnopqrstuvwxyz"
	upperChars  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digitChars  = "0123456789"
	symbolChars = "!@#$%^&*()-_=+[]{}|;:',.<>?/`~"
)

func containsAny(s, chars string) bool {
	for _, c := range chars {
		if strings.ContainsRune(s, c) {
			return true
		}
	}
	return false
}

func TestGenerate_DefaultLength(t *testing.T) {
	opts := DefaultOptions()
	pw, err := Generate(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pw) != 20 {
		t.Errorf("expected length 20, got %d", len(pw))
	}
}

func TestGenerate_RespectsLength(t *testing.T) {
	for _, length := range []int{4, 8, 32, 64, 128} {
		opts := DefaultOptions()
		opts.Length = length
		pw, err := Generate(opts)
		if err != nil {
			t.Fatalf("length=%d: unexpected error: %v", length, err)
		}
		if len(pw) != length {
			t.Errorf("length=%d: got %d", length, len(pw))
		}
	}
}

func TestGenerate_ContainsAllCharsets(t *testing.T) {
	opts := DefaultOptions()
	opts.Length = 100

	hasLower, hasUpper, hasDigit, hasSymbol := false, false, false, false
	for i := 0; i < 50; i++ {
		pw, err := Generate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if containsAny(pw, lowerChars) {
			hasLower = true
		}
		if containsAny(pw, upperChars) {
			hasUpper = true
		}
		if containsAny(pw, digitChars) {
			hasDigit = true
		}
		if containsAny(pw, symbolChars) {
			hasSymbol = true
		}
	}
	if !hasLower || !hasUpper || !hasDigit || !hasSymbol {
		t.Errorf("missing charsets: lower=%v upper=%v digit=%v symbol=%v", hasLower, hasUpper, hasDigit, hasSymbol)
	}
}

func TestGenerate_OnlyLower(t *testing.T) {
	opts := Options{
		Length:         50,
		IncludeLower:   true,
		IncludeUpper:   false,
		IncludeDigits:  false,
		IncludeSymbols: false,
	}
	pw, err := Generate(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range pw {
		if !strings.ContainsRune(lowerChars, c) {
			t.Errorf("unexpected character %q in lowercase-only password", c)
		}
	}
}

func TestGenerate_OnlyDigits(t *testing.T) {
	opts := Options{
		Length:         50,
		IncludeLower:   false,
		IncludeUpper:   false,
		IncludeDigits:  true,
		IncludeSymbols: false,
	}
	pw, err := Generate(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range pw {
		if !strings.ContainsRune(digitChars, c) {
			t.Errorf("unexpected character %q in digits-only password", c)
		}
	}
}

func TestGenerate_ZeroLength_Error(t *testing.T) {
	opts := DefaultOptions()
	opts.Length = 0
	_, err := Generate(opts)
	if err == nil {
		t.Error("expected error for zero length")
	}
}

func TestGenerate_NegativeLength_Error(t *testing.T) {
	opts := DefaultOptions()
	opts.Length = -5
	_, err := Generate(opts)
	if err == nil {
		t.Error("expected error for negative length")
	}
}

func TestGenerate_NoCharsetsEnabled_Error(t *testing.T) {
	opts := Options{Length: 20}
	_, err := Generate(opts)
	if err == nil {
		t.Error("expected error when no charsets enabled")
	}
}

func TestGenerate_GuaranteesCharClasses(t *testing.T) {
	opts := Options{
		Length:         8,
		IncludeLower:   true,
		IncludeUpper:   true,
		IncludeDigits:  true,
		IncludeSymbols: true,
	}
	for i := 0; i < 100; i++ {
		pw, err := Generate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !containsAny(pw, lowerChars) {
			t.Errorf("password %q missing lowercase", pw)
		}
		if !containsAny(pw, upperChars) {
			t.Errorf("password %q missing uppercase", pw)
		}
		if !containsAny(pw, digitChars) {
			t.Errorf("password %q missing digit", pw)
		}
		if !containsAny(pw, symbolChars) {
			t.Errorf("password %q missing symbol", pw)
		}
	}
}

func TestGenerate_LengthTooShortForAllClasses(t *testing.T) {
	opts := Options{
		Length:         2,
		IncludeLower:   true,
		IncludeUpper:   true,
		IncludeDigits:  true,
		IncludeSymbols: true,
	}
	_, err := Generate(opts)
	if err == nil {
		t.Error("expected error when length is too short for all enabled classes")
	}
}

func TestGenerate_Uniqueness(t *testing.T) {
	opts := DefaultOptions()
	pw1, err := Generate(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pw2, err := Generate(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pw1 == pw2 {
		t.Error("two consecutive Generate calls produced identical passwords")
	}
}
