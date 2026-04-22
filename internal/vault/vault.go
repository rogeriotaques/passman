package vault

import (
	"errors"
	"sort"
	"strings"
	"time"
)

var (
	ErrEntryNotFound = errors.New("entry not found")
	ErrEntryExists   = errors.New("entry already exists")
)

type Entry struct {
	Name       string    `json:"name"`
	Username   string    `json:"username"`
	Password   string    `json:"password"`
	Notes      string    `json:"notes"`
	Tags       []string  `json:"tags"`
	TotpSecret string    `json:"totp_secret,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Vault struct {
	Entries []Entry `json:"entries"`
}

func (v *Vault) Add(entry Entry) error {
	for _, e := range v.Entries {
		if strings.EqualFold(e.Name, entry.Name) {
			return ErrEntryExists
		}
	}
	now := time.Now()
	entry.CreatedAt = now
	entry.UpdatedAt = now
	v.Entries = append(v.Entries, entry)
	return nil
}

func (v *Vault) Get(name string) (*Entry, error) {
	for i := range v.Entries {
		if strings.EqualFold(v.Entries[i].Name, name) {
			return &v.Entries[i], nil
		}
	}
	return nil, ErrEntryNotFound
}

func (v *Vault) List() []string {
	names := make([]string, len(v.Entries))
	for i, e := range v.Entries {
		names[i] = e.Name
	}
	sort.Strings(names)
	return names
}

func (v *Vault) Search(query string) []Entry {
	tokens := strings.Fields(strings.ToLower(query))

	var results []Entry
	for _, e := range v.Entries {
		searchable := strings.ToLower(
			e.Name + " " + e.Username + " " + e.Notes + " " + strings.Join(e.Tags, " "),
		)

		match := true
		for _, tok := range tokens {
			if !strings.Contains(searchable, tok) {
				match = false
				break
			}
		}
		if match {
			results = append(results, e)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
	})
	return results
}

func (v *Vault) ListByTag(tag string) []Entry {
	var result []Entry
	for _, e := range v.Entries {
		for _, t := range e.Tags {
			if strings.EqualFold(t, tag) {
				result = append(result, e)
				break
			}
		}
	}
	return result
}

func (v *Vault) Remove(name string) error {
	for i, e := range v.Entries {
		if strings.EqualFold(e.Name, name) {
			v.Entries = append(v.Entries[:i], v.Entries[i+1:]...)
			return nil
		}
	}
	return ErrEntryNotFound
}
