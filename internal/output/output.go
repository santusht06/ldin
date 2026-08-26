// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

// OutputFormat defines supported output types
type OutputFormat string

const (
	FormatHuman OutputFormat = "human"
	FormatJSON  OutputFormat = "json"
	FormatYAML  OutputFormat = "yaml"
	FormatCSV   OutputFormat = "csv"
	FormatQuiet OutputFormat = "quiet"
)

// UI Theme & Lipgloss styles for rich terminal output
var (
	// Colors inspired by modern developer tooling & LinkedIn professional palette
	LinkedInBlue = lipgloss.Color("#0A66C2")
	AccentCyan   = lipgloss.Color("#00D2FF")
	SuccessGreen = lipgloss.Color("#05DF72")
	WarningAmber = lipgloss.Color("#FFB020")
	DangerRed    = lipgloss.Color("#FF4D4D")
	SubtleGray   = lipgloss.Color("#6E7681")
	BorderGray   = lipgloss.Color("#30363D")
	BgDark       = lipgloss.Color("#161B22")
	TextPrimary  = lipgloss.Color("#F0F6FC")
	TextMuted    = lipgloss.Color("#8B949E")

	// Pre-built styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(LinkedInBlue).
			Padding(0, 1).
			MarginBottom(1)

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(AccentCyan).
			MarginBottom(1)

	SectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#58A6FF")).
			MarginTop(1).
			MarginBottom(1)

	SuccessBadge = lipgloss.NewStyle().
			Foreground(SuccessGreen).
			Bold(true)

	WarningBadge = lipgloss.NewStyle().
			Foreground(WarningAmber).
			Bold(true)

	DangerBadge = lipgloss.NewStyle().
			Foreground(DangerRed).
			Bold(true)

	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderGray).
			Padding(1, 2).
			MarginBottom(1)

	PromptBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(LinkedInBlue).
			Padding(1, 2).
			MarginBottom(1)

	LabelStyle = lipgloss.NewStyle().
			Foreground(SubtleGray).
			Width(16)

	ValueStyle = lipgloss.NewStyle().
			Foreground(TextPrimary).
			Bold(true)

	DimStyle = lipgloss.NewStyle().
			Foreground(TextMuted)
)

// Formatter manages printing to stdout/stderr in the configured mode
type Formatter struct {
	Format  OutputFormat
	Writer  io.Writer
	Verbose bool
	Debug   bool
}

// NewFormatter creates an instance with the given format
func NewFormatter(format string, verbose, debug bool) *Formatter {
	f := OutputFormat(strings.ToLower(format))
	switch f {
	case FormatJSON, FormatYAML, FormatCSV, FormatQuiet:
		// valid
	default:
		f = FormatHuman
	}

	return &Formatter{
		Format:  f,
		Writer:  os.Stdout,
		Verbose: verbose,
		Debug:   debug,
	}
}

// Print renders data according to current format
func (f *Formatter) Print(data interface{}, humanRenderer func()) error {
	switch f.Format {
	case FormatJSON:
		enc := json.NewEncoder(f.Writer)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	case FormatYAML:
		enc := yaml.NewEncoder(f.Writer)
		return enc.Encode(data)
	case FormatQuiet:
		return nil
	default:
		if humanRenderer != nil {
			humanRenderer()
		}
		return nil
	}
}

// Success prints a styled green success message
func (f *Formatter) Success(msg string, args ...interface{}) {
	if f.Format == FormatQuiet || f.Format == FormatJSON || f.Format == FormatYAML {
		return
	}
	formatted := fmt.Sprintf(msg, args...)
	fmt.Fprintf(f.Writer, "%s %s\n", SuccessBadge.Render("✓"), formatted)
}

// Warning prints a styled amber warning message
func (f *Formatter) Warning(msg string, args ...interface{}) {
	if f.Format == FormatQuiet || f.Format == FormatJSON || f.Format == FormatYAML {
		return
	}
	formatted := fmt.Sprintf(msg, args...)
	fmt.Fprintf(f.Writer, "%s %s\n", WarningBadge.Render("⚠"), formatted)
}

// Error prints a styled red error message
func (f *Formatter) Error(msg string, args ...interface{}) {
	formatted := fmt.Sprintf(msg, args...)
	fmt.Fprintf(os.Stderr, "%s %s\n", DangerBadge.Render("✗"), formatted)
}

// Info prints a standard info message
func (f *Formatter) Info(msg string, args ...interface{}) {
	if f.Format == FormatQuiet || f.Format == FormatJSON || f.Format == FormatYAML {
		return
	}
	formatted := fmt.Sprintf(msg, args...)
	fmt.Fprintf(f.Writer, "%s %s\n", lipgloss.NewStyle().Foreground(AccentCyan).Render("ℹ"), formatted)
}

// PrintKeyValue renders a key-value row with aligned labels
func (f *Formatter) PrintKeyValue(key string, value string) {
	if f.Format != FormatHuman {
		return
	}
	fmt.Fprintf(f.Writer, "%s %s\n", LabelStyle.Render(key+":"), ValueStyle.Render(value))
}

// PrintDivider prints a subtle horizontal divider
func (f *Formatter) PrintDivider(width int) {
	if f.Format != FormatHuman {
		return
	}
	if width <= 0 {
		width = 60
	}
	fmt.Fprintln(f.Writer, DimStyle.Render(strings.Repeat("─", width)))
}

// PrintTable renders an aligned CLI table for human output or CSV for CSV output
func (f *Formatter) PrintTable(headers []string, rows [][]string) {
	if f.Format == FormatCSV {
		w := csv.NewWriter(f.Writer)
		_ = w.Write(headers)
		_ = w.WriteAll(rows)
		w.Flush()
		return
	}
	if f.Format != FormatHuman {
		return
	}

	if len(headers) == 0 {
		return
	}

	colWidths := make([]int, len(headers))
	for i, h := range headers {
		colWidths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Render header
	var headerParts []string
	for i, h := range headers {
		headerParts = append(headerParts, lipgloss.NewStyle().Bold(true).Foreground(AccentCyan).Width(colWidths[i]+2).Render(h))
	}
	fmt.Fprintln(f.Writer, strings.Join(headerParts, ""))

	// Render divider
	var divParts []string
	for i := range headers {
		divParts = append(divParts, DimStyle.Render(strings.Repeat("─", colWidths[i]))+"  ")
	}
	fmt.Fprintln(f.Writer, strings.Join(divParts, ""))

	// Render rows
	for _, row := range rows {
		var rowParts []string
		for i, cell := range row {
			if i < len(colWidths) {
				rowParts = append(rowParts, lipgloss.NewStyle().Width(colWidths[i]+2).Render(cell))
			}
		}
		fmt.Fprintln(f.Writer, strings.Join(rowParts, ""))
	}
	fmt.Fprintln(f.Writer)
}
