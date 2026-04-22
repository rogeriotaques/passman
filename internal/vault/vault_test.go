package vault

import (
	"testing"
	"time"
)

func TestVault_Add_Success(t *testing.T) {
	v := &Vault{}
	entry := Entry{Name: "github", Username: "user", Password: "pass123"}
	if err := v.Add(entry); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(v.Entries))
	}
	if v.Entries[0].Name != "github" {
		t.Errorf("expected name 'github', got %q", v.Entries[0].Name)
	}
}

func TestVault_Add_SetsTimestamps(t *testing.T) {
	v := &Vault{}
	before := time.Now()
	_ = v.Add(Entry{Name: "test", Username: "u", Password: "p"})
	after := time.Now()

	e := v.Entries[0]
	if e.CreatedAt.Before(before) || e.CreatedAt.After(after) {
		t.Error("CreatedAt not set correctly")
	}
	if e.UpdatedAt.Before(before) || e.UpdatedAt.After(after) {
		t.Error("UpdatedAt not set correctly")
	}
}

func TestVault_Add_DuplicateName_Error(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "github", Username: "u1", Password: "p1"})
	err := v.Add(Entry{Name: "github", Username: "u2", Password: "p2"})
	if err == nil {
		t.Error("expected error for duplicate name")
	}
}

func TestVault_Add_DuplicateName_CaseInsensitive(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "GitHub", Username: "u1", Password: "p1"})
	err := v.Add(Entry{Name: "github", Username: "u2", Password: "p2"})
	if err == nil {
		t.Error("expected error for case-insensitive duplicate name")
	}
}

func TestVault_Get_Found(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "github", Username: "user", Password: "pass"})

	e, err := v.Get("github")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Username != "user" {
		t.Errorf("expected username 'user', got %q", e.Username)
	}
}

func TestVault_Get_CaseInsensitive(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "GitHub", Username: "user", Password: "pass"})

	e, err := v.Get("github")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Username != "user" {
		t.Errorf("expected username 'user', got %q", e.Username)
	}
}

func TestVault_Get_NotFound(t *testing.T) {
	v := &Vault{}
	_, err := v.Get("nonexistent")
	if err == nil {
		t.Error("expected error for missing entry")
	}
}

func TestVault_List_Empty(t *testing.T) {
	v := &Vault{}
	names := v.List()
	if len(names) != 0 {
		t.Errorf("expected empty list, got %v", names)
	}
}

func TestVault_List_Sorted(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "charlie", Username: "u", Password: "p"})
	_ = v.Add(Entry{Name: "alpha", Username: "u", Password: "p"})
	_ = v.Add(Entry{Name: "bravo", Username: "u", Password: "p"})

	names := v.List()
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}
	if names[0] != "alpha" || names[1] != "bravo" || names[2] != "charlie" {
		t.Errorf("expected sorted order, got %v", names)
	}
}

func TestVault_Remove_Success(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "github", Username: "u", Password: "p"})

	err := v.Remove("github")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(v.Entries))
	}
}

func TestVault_Remove_CaseInsensitive(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "GitHub", Username: "u", Password: "p"})

	err := v.Remove("github")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(v.Entries))
	}
}

func TestVault_Remove_NotFound(t *testing.T) {
	v := &Vault{}
	err := v.Remove("nonexistent")
	if err == nil {
		t.Error("expected error for removing nonexistent entry")
	}
}

func TestVault_ListByTag_Found(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "aws-key", Username: "u", Password: "p", Tags: []string{"prod"}})
	_ = v.Add(Entry{Name: "db-pass", Username: "u", Password: "p", Tags: []string{"prod", "staging"}})
	_ = v.Add(Entry{Name: "dev-key", Username: "u", Password: "p", Tags: []string{"dev"}})

	entries := v.ListByTag("prod")
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestVault_ListByTag_CaseInsensitive(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "aws-key", Username: "u", Password: "p", Tags: []string{"Prod"}})

	entries := v.ListByTag("prod")
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestVault_ListByTag_NoMatch(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "aws-key", Username: "u", Password: "p", Tags: []string{"prod"}})

	entries := v.ListByTag("staging")
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestVault_ListByTag_EmptyVault(t *testing.T) {
	v := &Vault{}
	entries := v.ListByTag("prod")
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestVault_Search_SingleToken_MatchesName(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "github", Username: "user", Password: "p"})
	_ = v.Add(Entry{Name: "gitlab", Username: "user", Password: "p"})
	_ = v.Add(Entry{Name: "aws-prod", Username: "admin", Password: "p"})

	results := v.Search("git")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestVault_Search_SingleToken_MatchesUsername(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "github", Username: "octocat@example.com", Password: "p"})
	_ = v.Add(Entry{Name: "aws", Username: "admin", Password: "p"})

	results := v.Search("octocat")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "github" {
		t.Errorf("expected github, got %q", results[0].Name)
	}
}

func TestVault_Search_SingleToken_MatchesNotes(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "aws", Username: "u", Password: "p", Notes: "production account"})
	_ = v.Add(Entry{Name: "dev", Username: "u", Password: "p", Notes: "local testing"})

	results := v.Search("production")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "aws" {
		t.Errorf("expected aws, got %q", results[0].Name)
	}
}

func TestVault_Search_SingleToken_MatchesTags(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "aws-key", Username: "u", Password: "p", Tags: []string{"prod", "infra"}})
	_ = v.Add(Entry{Name: "dev-key", Username: "u", Password: "p", Tags: []string{"dev"}})

	results := v.Search("infra")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "aws-key" {
		t.Errorf("expected aws-key, got %q", results[0].Name)
	}
}

func TestVault_Search_MultipleTokens_AllMustMatch(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "github-work", Username: "alice@corp.com", Password: "p"})
	_ = v.Add(Entry{Name: "github-personal", Username: "alice@gmail.com", Password: "p"})
	_ = v.Add(Entry{Name: "gitlab-work", Username: "alice@corp.com", Password: "p"})

	results := v.Search("github corp")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "github-work" {
		t.Errorf("expected github-work, got %q", results[0].Name)
	}
}

func TestVault_Search_CaseInsensitive(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "GitHub", Username: "User", Password: "p"})

	results := v.Search("github")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	results = v.Search("GITHUB")
	if len(results) != 1 {
		t.Fatalf("expected 1 result for uppercase query, got %d", len(results))
	}
}

func TestVault_Search_EmptyQuery_ReturnsAll(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "a", Username: "u", Password: "p"})
	_ = v.Add(Entry{Name: "b", Username: "u", Password: "p"})

	results := v.Search("")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestVault_Search_NoMatch(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "github", Username: "u", Password: "p"})

	results := v.Search("nonexistent")
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestVault_Search_ResultsSorted(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "charlie", Username: "u", Password: "p"})
	_ = v.Add(Entry{Name: "alpha", Username: "u", Password: "p"})
	_ = v.Add(Entry{Name: "bravo", Username: "u", Password: "p"})

	results := v.Search("")
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Name != "alpha" || results[1].Name != "bravo" || results[2].Name != "charlie" {
		t.Errorf("expected sorted order, got %v", []string{results[0].Name, results[1].Name, results[2].Name})
	}
}

func TestVault_Search_TokenMatchesAcrossFields(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "myservice", Username: "admin@corp.com", Password: "p", Tags: []string{"prod"}})

	results := v.Search("myservice prod")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestVault_Search_NeverMatchesPassword(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "github", Username: "u", Password: "supersecret123"})

	results := v.Search("supersecret123")
	if len(results) != 0 {
		t.Errorf("search should not match passwords, got %d results", len(results))
	}
}
