package generate

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
)

const (
	lowercase = "abcdefghijklmnopqrstuvwxyz"
	uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits    = "0123456789"
	symbols   = "!@#$%^&*()-_=+[]{}|;:',.<>?/`~"
)

type Options struct {
	Length         int
	IncludeLower   bool
	IncludeUpper   bool
	IncludeDigits  bool
	IncludeSymbols bool
}

func DefaultOptions() Options {
	return Options{
		Length:         20,
		IncludeLower:   true,
		IncludeUpper:   true,
		IncludeDigits:  true,
		IncludeSymbols: true,
	}
}

func Generate(opts Options) (string, error) {
	if opts.Length <= 0 {
		return "", errors.New("password length must be positive")
	}

	required := requiredCharsets(opts)
	if len(required) > opts.Length {
		return "", fmt.Errorf("password length %d is too short for %d required character classes", opts.Length, len(required))
	}

	charset := buildCharset(opts)
	if len(charset) == 0 {
		return "", errors.New("at least one charset must be enabled")
	}

	result := make([]byte, opts.Length)

	for i, class := range required {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(class))))
		if err != nil {
			return "", err
		}
		result[i] = class[idx.Int64()]
	}

	max := big.NewInt(int64(len(charset)))
	for i := len(required); i < opts.Length; i++ {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		result[i] = charset[idx.Int64()]
	}

	for i := len(result) - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return "", err
		}
		result[i], result[j.Int64()] = result[j.Int64()], result[i]
	}

	return string(result), nil
}

func requiredCharsets(opts Options) []string {
	var classes []string
	if opts.IncludeLower {
		classes = append(classes, lowercase)
	}
	if opts.IncludeUpper {
		classes = append(classes, uppercase)
	}
	if opts.IncludeDigits {
		classes = append(classes, digits)
	}
	if opts.IncludeSymbols {
		classes = append(classes, symbols)
	}
	return classes
}

func buildCharset(opts Options) string {
	var charset string
	if opts.IncludeLower {
		charset += lowercase
	}
	if opts.IncludeUpper {
		charset += uppercase
	}
	if opts.IncludeDigits {
		charset += digits
	}
	if opts.IncludeSymbols {
		charset += symbols
	}
	return charset
}
