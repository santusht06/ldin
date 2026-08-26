// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestJSONFormatting(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter("json", false, false)
	f.Writer = &buf

	data := map[string]string{
		"message": "Hello ldin",
		"status":  "success",
	}

	err := f.Print(data, nil)
	if err != nil {
		t.Fatalf("f.Print failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"message": "Hello ldin"`) {
		t.Errorf("expected JSON to contain message, got: %s", out)
	}
}

func TestYAMLFormatting(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter("yaml", false, false)
	f.Writer = &buf

	data := map[string]string{
		"tool": "ldin",
	}

	err := f.Print(data, nil)
	if err != nil {
		t.Fatalf("f.Print failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "tool: ldin") {
		t.Errorf("expected YAML output, got: %s", out)
	}
}

func TestCSVTableFormatting(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter("csv", false, false)
	f.Writer = &buf

	headers := []string{"ID", "Name", "Role"}
	rows := [][]string{
		{"1", "Santusht", "Engineer"},
	}

	f.PrintTable(headers, rows)
	out := buf.String()
	if !strings.Contains(out, "ID,Name,Role") {
		t.Errorf("expected CSV header row, got: %s", out)
	}
	if !strings.Contains(out, "1,Santusht,Engineer") {
		t.Errorf("expected CSV data row, got: %s", out)
	}
}
