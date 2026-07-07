package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Table is one GFM pipe table: a header row and its data rows, verbatim.
type Table struct {
	Header []string   `json:"header"`
	Rows   [][]string `json:"rows"`
}

// ExtractTables parses every pipe table in the Markdown file at filePath,
// keeping those whose header contains headerText (all when empty) and, when
// tableIndex >= 0, only the table at that document-order position.
func ExtractTables(filePath string, headerText string, tableIndex int) ([]Table, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", filePath, err)
	}

	tables := parseTables(string(content))

	var result []Table
	for i, t := range tables {
		if tableIndex != -1 && i != tableIndex {
			continue
		}
		if headerText != "" {
			match := false
			for _, h := range t.Header {
				if strings.Contains(h, headerText) {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		result = append(result, t)
	}

	return result, nil
}

func parseTables(content string) []Table {
	var tables []Table
	lines := strings.Split(content, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || !strings.Contains(line, "|") {
			continue
		}

		if i+1 >= len(lines) {
			break
		}
		nextLine := strings.TrimSpace(lines[i+1])
		if !isSeparatorRow(nextLine) {
			continue
		}

		table := Table{}
		table.Header = parseRow(line)

		i += 2
		for i < len(lines) {
			rowLine := strings.TrimSpace(lines[i])
			if rowLine == "" || !strings.Contains(rowLine, "|") {
				break
			}
			table.Rows = append(table.Rows, parseRow(rowLine))
			i++
		}
		tables = append(tables, table)
	}

	return tables
}

func isSeparatorRow(line string) bool {
	if !strings.Contains(line, "|") {
		return false
	}
	parts := strings.Split(strings.Trim(line, "|"), "|")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		for _, r := range p {
			if r != '-' && r != ':' {
				return false
			}
		}
	}
	return true
}

func parseRow(line string) []string {
	trimmed := strings.Trim(line, "|")
	parts := strings.Split(trimmed, "|")
	var row []string
	for _, p := range parts {
		row = append(row, strings.TrimSpace(p))
	}
	return row
}

// EncodeTables writes the tables to w as indented JSON.
func EncodeTables(w io.Writer, tables []Table) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(tables)
}
