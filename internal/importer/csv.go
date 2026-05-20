package importer

import (
	"encoding/csv"
	"errors"
	"io"
	"strings"

	"github.com/rogerio/passman/internal/vault"
)

func ParseCSV(r io.Reader) ([]vault.Entry, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		return nil, errors.New("empty or invalid CSV file")
	}

	nameIdx, valueIdx := detectColumns(header)
	if nameIdx < 0 || valueIdx < 0 {
		return nil, errors.New("unrecognized CSV format: expected 'name' and 'value' columns")
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

		name := field(record, nameIdx)
		value := field(record, valueIdx)
		if name == "" {
			continue
		}

		entries = append(entries, vault.Entry{
			Name:  name,
			Value: value,
		})
	}

	return entries, nil
}

func detectColumns(header []string) (nameIdx, valueIdx int) {
	nameIdx, valueIdx = -1, -1
	for i, h := range header {
		switch strings.ToLower(strings.TrimSpace(h)) {
		case "name":
			nameIdx = i
		case "value", "secret", "password":
			valueIdx = i
		}
	}
	return
}

func field(record []string, idx int) string {
	if idx < 0 || idx >= len(record) {
		return ""
	}
	return record[idx]
}
