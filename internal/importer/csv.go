package importer

import (
	"encoding/csv"
	"errors"
	"io"
	"strings"

	"github.com/rogerio/passman/internal/vault"
)

type csvMapping struct {
	name     int
	username int
	password int
	notes    int
}

func ParseCSV(r io.Reader) ([]vault.Entry, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		return nil, errors.New("empty or invalid CSV file")
	}

	mapping, err := detectCSVFormat(header)
	if err != nil {
		return nil, err
	}

	var entries []vault.Entry
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		password := getField(record, mapping.password)
		if password == "" {
			continue
		}

		entries = append(entries, vault.Entry{
			Name:     getField(record, mapping.name),
			Username: getField(record, mapping.username),
			Password: password,
			Notes:    getField(record, mapping.notes),
		})
	}

	return entries, nil
}

func detectCSVFormat(header []string) (*csvMapping, error) {
	normalized := make([]string, len(header))
	for i, h := range header {
		normalized[i] = strings.ToLower(strings.TrimSpace(h))
	}

	colIndex := func(names ...string) int {
		for _, name := range names {
			for i, h := range normalized {
				if h == name {
					return i
				}
			}
		}
		return -1
	}

	nameIdx := colIndex("title", "name")
	usernameIdx := colIndex("username", "login_username")
	passwordIdx := colIndex("password", "login_password")
	notesIdx := colIndex("notes")

	if nameIdx < 0 || passwordIdx < 0 {
		return nil, errors.New("unrecognized CSV format: expected 'title'/'name' and 'password'/'login_password' columns")
	}

	return &csvMapping{
		name:     nameIdx,
		username: usernameIdx,
		password: passwordIdx,
		notes:    notesIdx,
	}, nil
}

func getField(record []string, idx int) string {
	if idx < 0 || idx >= len(record) {
		return ""
	}
	return record[idx]
}
