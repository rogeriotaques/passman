package importer

import (
	"strings"
	"testing"
)

func TestParseCSV_NameValue(t *testing.T) {
	input := "name,value\ngithub,secret123\naws,awspass\n"
	entries, err := ParseCSV(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name != "github" || entries[0].Value != "secret123" {
		t.Errorf("unexpected first entry: %+v", entries[0])
	}
	if entries[1].Name != "aws" || entries[1].Value != "awspass" {
		t.Errorf("unexpected second entry: %+v", entries[1])
	}
}

func TestParseCSV_NamePassword(t *testing.T) {
	input := "name,password\ngithub,secret123\n"
	entries, err := ParseCSV(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "github" || entries[0].Value != "secret123" {
		t.Errorf("unexpected entry: %+v", entries[0])
	}
}

func TestParseCSV_NameSecret(t *testing.T) {
	input := "name,secret\ngithub,secret123\n"
	entries, err := ParseCSV(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Value != "secret123" {
		t.Errorf("expected value 'secret123', got %q", entries[0].Value)
	}
}

func TestParseCSV_CaseInsensitiveHeaders(t *testing.T) {
	input := "Name,Value\nGitHub,secret\n"
	entries, err := ParseCSV(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 || entries[0].Name != "GitHub" {
		t.Errorf("unexpected entry: %+v", entries)
	}
}

func TestParseCSV_SkipsEmptyName(t *testing.T) {
	input := "name,value\n,secret123\ngithub,pass\n"
	entries, err := ParseCSV(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (skip empty name), got %d", len(entries))
	}
	if entries[0].Name != "github" {
		t.Errorf("expected github, got %q", entries[0].Name)
	}
}

func TestParseCSV_UnknownFormat_Error(t *testing.T) {
	input := "foo,bar,baz\n1,2,3\n"
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

func TestParseCSV_EmptyValue(t *testing.T) {
	input := "name,value\ngithub,\n"
	entries, err := ParseCSV(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Value != "" {
		t.Errorf("expected empty value, got %q", entries[0].Value)
	}
}
