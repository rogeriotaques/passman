package vault

import (
	"testing"
)

func TestAdd_Success(t *testing.T) {
	v := &Vault{}
	err := v.Add(Entry{Name: "github", Value: "secret123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(v.Entries))
	}
	if v.Entries[0].Name != "github" || v.Entries[0].Value != "secret123" {
		t.Errorf("unexpected entry: %+v", v.Entries[0])
	}
}

func TestAdd_Duplicate_CaseInsensitive(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "github", Value: "pass1"})
	err := v.Add(Entry{Name: "GitHub", Value: "pass2"})
	if err != ErrEntryExists {
		t.Errorf("expected ErrEntryExists, got %v", err)
	}
}

func TestGet_Found(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "github", Value: "secret"})

	entry, err := v.Get("GitHub")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.Value != "secret" {
		t.Errorf("expected 'secret', got %q", entry.Value)
	}
}

func TestGet_NotFound(t *testing.T) {
	v := &Vault{}
	_, err := v.Get("nonexistent")
	if err != ErrEntryNotFound {
		t.Errorf("expected ErrEntryNotFound, got %v", err)
	}
}

func TestRemove_Success(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "github", Value: "secret"})

	err := v.Remove("GitHub")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(v.Entries))
	}
}

func TestRemove_NotFound(t *testing.T) {
	v := &Vault{}
	err := v.Remove("nonexistent")
	if err != ErrEntryNotFound {
		t.Errorf("expected ErrEntryNotFound, got %v", err)
	}
}

func TestList_Empty(t *testing.T) {
	v := &Vault{}
	names := v.List()
	if len(names) != 0 {
		t.Errorf("expected empty list, got %v", names)
	}
}

func TestList_Sorted(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "zebra", Value: "z"})
	_ = v.Add(Entry{Name: "alpha", Value: "a"})
	_ = v.Add(Entry{Name: "middle", Value: "m"})

	names := v.List()
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}
	if names[0] != "alpha" || names[1] != "middle" || names[2] != "zebra" {
		t.Errorf("expected sorted order, got %v", names)
	}
}

func TestSearch_SingleToken(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "github-work", Value: "w"})
	_ = v.Add(Entry{Name: "github-personal", Value: "p"})
	_ = v.Add(Entry{Name: "aws-prod", Value: "a"})

	results := v.Search("github")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestSearch_MultipleTokens(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "github-work", Value: "w"})
	_ = v.Add(Entry{Name: "github-personal", Value: "p"})

	results := v.Search("github work")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "github-work" {
		t.Errorf("expected github-work, got %s", results[0].Name)
	}
}

func TestSearch_CaseInsensitive(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "GitHub-Work", Value: "w"})

	results := v.Search("GITHUB")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestSearch_EmptyQuery_ReturnsAll(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "a", Value: "1"})
	_ = v.Add(Entry{Name: "b", Value: "2"})

	results := v.Search("")
	if len(results) != 2 {
		t.Errorf("empty query should return all, got %d", len(results))
	}
}

func TestSearch_NoMatch(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "github", Value: "g"})

	results := v.Search("nonexistent")
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearch_ResultsSorted(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "z-match", Value: "z"})
	_ = v.Add(Entry{Name: "a-match", Value: "a"})

	results := v.Search("match")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Name != "a-match" {
		t.Errorf("expected a-match first, got %s", results[0].Name)
	}
}

func TestUpsert_New(t *testing.T) {
	v := &Vault{}
	v.Upsert(Entry{Name: "github", Value: "new"})
	if len(v.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(v.Entries))
	}
	if v.Entries[0].Value != "new" {
		t.Errorf("expected 'new', got %q", v.Entries[0].Value)
	}
}

func TestUpsert_Replace(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "github", Value: "old"})
	v.Upsert(Entry{Name: "GitHub", Value: "updated"})
	if len(v.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(v.Entries))
	}
	if v.Entries[0].Value != "updated" {
		t.Errorf("expected 'updated', got %q", v.Entries[0].Value)
	}
}

func TestSearch_NeverMatchesValue(t *testing.T) {
	v := &Vault{}
	_ = v.Add(Entry{Name: "github", Value: "supersecret123"})

	results := v.Search("supersecret123")
	if len(results) != 0 {
		t.Errorf("search should not match values, got %d results", len(results))
	}
}
