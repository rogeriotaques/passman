package importer

import (
	"bufio"
	"io"
	"strings"

	"github.com/rogerio/passman/internal/vault"
)

func ParseEnv(r io.Reader) ([]vault.Entry, error) {
	var entries []vault.Entry
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.TrimPrefix(line, "export ")

		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		value = unquote(value)

		if key == "" {
			continue
		}

		entries = append(entries, vault.Entry{
			Name:  key,
			Value: value,
		})
	}

	return entries, scanner.Err()
}

func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
