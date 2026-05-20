package vault

import (
	"errors"
	"sort"
	"strings"
)

var (
	ErrEntryNotFound = errors.New("entry not found")
	ErrEntryExists   = errors.New("entry already exists")
)

type Entry struct {
	Name  string `json:"name"`
	Value string `json:"value"`
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

func (v *Vault) Remove(name string) error {
	for i, e := range v.Entries {
		if strings.EqualFold(e.Name, name) {
			v.Entries = append(v.Entries[:i], v.Entries[i+1:]...)
			return nil
		}
	}
	return ErrEntryNotFound
}

func (v *Vault) Upsert(entry Entry) {
	for i, e := range v.Entries {
		if strings.EqualFold(e.Name, entry.Name) {
			v.Entries[i] = entry
			return
		}
	}
	v.Entries = append(v.Entries, entry)
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
		nameLower := strings.ToLower(e.Name)
		match := true
		for _, tok := range tokens {
			if !strings.Contains(nameLower, tok) {
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
