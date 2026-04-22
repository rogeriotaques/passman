package importer

import (
	"strings"
	"testing"
)

func TestParseCSV_1Password(t *testing.T) {
	input := `Title,Username,Password,Notes,URL
GitHub,octocat,secret123,my notes,https://github.com
AWS Console,admin,awspass,,https://aws.amazon.com
`
	entries, err := ParseCSV(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name != "GitHub" || entries[0].Username != "octocat" || entries[0].Password != "secret123" {
		t.Errorf("unexpected first entry: %+v", entries[0])
	}
	if entries[0].Notes != "my notes" {
		t.Errorf("expected notes 'my notes', got %q", entries[0].Notes)
	}
	if entries[1].Name != "AWS Console" || entries[1].Password != "awspass" {
		t.Errorf("unexpected second entry: %+v", entries[1])
	}
}

func TestParseCSV_Bitwarden(t *testing.T) {
	input := `name,login_uri,login_username,login_password,notes
GitHub,https://github.com,octocat,secret123,my notes
`
	entries, err := ParseCSV(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "GitHub" || entries[0].Username != "octocat" || entries[0].Password != "secret123" {
		t.Errorf("unexpected entry: %+v", entries[0])
	}
}

func TestParseCSV_SkipsEmptyPassword(t *testing.T) {
	input := `Title,Username,Password,Notes,URL
NoPass,user,,some notes,https://example.com
HasPass,user,secret,,
`
	entries, err := ParseCSV(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (skip empty password), got %d", len(entries))
	}
	if entries[0].Name != "HasPass" {
		t.Errorf("expected HasPass, got %q", entries[0].Name)
	}
}

func TestParseCSV_UnknownFormat_Error(t *testing.T) {
	input := `foo,bar,baz
1,2,3
`
	_, err := ParseCSV(strings.NewReader(input))
	if err == nil {
		t.Error("expected error for unrecognized CSV format")
	}
}

func TestParseCSV_Empty(t *testing.T) {
	_, err := ParseCSV(strings.NewReader(""))
	if err == nil {
		t.Error("expected error for empty input")
	}
}
