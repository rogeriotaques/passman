package importer

import (
	"strings"
	"testing"
)

func TestParseEnv_BasicKeyValue(t *testing.T) {
	input := "DB_HOST=localhost\nDB_PORT=5432\n"
	entries, err := ParseEnv(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name != "DB_HOST" || entries[0].Password != "localhost" {
		t.Errorf("unexpected first entry: %+v", entries[0])
	}
	if entries[1].Name != "DB_PORT" || entries[1].Password != "5432" {
		t.Errorf("unexpected second entry: %+v", entries[1])
	}
}

func TestParseEnv_SkipsCommentsAndBlanks(t *testing.T) {
	input := "# comment\n\nKEY=value\n  \n# another comment\n"
	entries, err := ParseEnv(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestParseEnv_QuotedValues(t *testing.T) {
	input := `KEY1="hello world"
KEY2='single quoted'
KEY3="has \"escaped\" quotes"
`
	entries, err := ParseEnv(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Password != "hello world" {
		t.Errorf("expected unquoted value, got %q", entries[0].Password)
	}
	if entries[1].Password != "single quoted" {
		t.Errorf("expected unquoted value, got %q", entries[1].Password)
	}
}

func TestParseEnv_ValueWithEquals(t *testing.T) {
	input := "URL=postgres://user:pass@host/db?ssl=true\n"
	entries, err := ParseEnv(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entries[0].Password != "postgres://user:pass@host/db?ssl=true" {
		t.Errorf("unexpected value: %q", entries[0].Password)
	}
}

func TestParseEnv_ExportPrefix(t *testing.T) {
	input := "export KEY=value\n"
	entries, err := ParseEnv(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "KEY" {
		t.Errorf("expected name KEY, got %q", entries[0].Name)
	}
}

func TestParseEnv_EmptyValue(t *testing.T) {
	input := "EMPTY_KEY=\n"
	entries, err := ParseEnv(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Password != "" {
		t.Errorf("expected empty password, got %q", entries[0].Password)
	}
}

func TestParseEnv_Empty(t *testing.T) {
	entries, err := ParseEnv(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}
