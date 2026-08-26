// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package profilecode

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// DiffEntry captures a single field difference
type DiffEntry struct {
	Field    string `json:"field"`
	OldValue string `json:"old_value"`
	NewValue string `json:"new_value"`
	Type     string `json:"type"` // "added", "removed", "modified"
}

// DiffResult holds all comparison differences
type DiffResult struct {
	Entries []DiffEntry `json:"entries"`
	HasDiff bool        `json:"has_diff"`
}

// CompareProfiles produces a structured diff between base and target profile
func CompareProfiles(base, target *ProfileAsCode) *DiffResult {
	res := &DiffResult{}

	if base == nil && target == nil {
		return res
	}
	if base == nil {
		base = &ProfileAsCode{}
	}
	if target == nil {
		target = &ProfileAsCode{}
	}

	// Compare basic string fields
	compareStringField(res, "Headline", base.Headline, target.Headline)
	compareStringField(res, "About", base.About, target.About)
	compareStringField(res, "Location", base.Location, target.Location)
	compareStringField(res, "Industry", base.Industry, target.Industry)
	compareStringField(res, "Name", base.Name, target.Name)

	// Compare Skills
	compareStringSlices(res, "Skills", base.Skills, target.Skills)

	// Compare Experience count and companies
	baseExpMap := make(map[string]Experience)
	for _, exp := range base.Experience {
		baseExpMap[exp.Company+"::"+exp.Role] = exp
	}
	targetExpMap := make(map[string]Experience)
	for _, exp := range target.Experience {
		targetExpMap[exp.Company+"::"+exp.Role] = exp
	}

	for key, exp := range targetExpMap {
		if _, exists := baseExpMap[key]; !exists {
			res.Entries = append(res.Entries, DiffEntry{
				Field:    fmt.Sprintf("Experience (%s - %s)", exp.Role, exp.Company),
				OldValue: "",
				NewValue: fmt.Sprintf("%s (%s to %s)", exp.Description, exp.StartDate, exp.EndDate),
				Type:     "added",
			})
			res.HasDiff = true
		}
	}
	for key, exp := range baseExpMap {
		if _, exists := targetExpMap[key]; !exists {
			res.Entries = append(res.Entries, DiffEntry{
				Field:    fmt.Sprintf("Experience (%s - %s)", exp.Role, exp.Company),
				OldValue: fmt.Sprintf("%s (%s to %s)", exp.Description, exp.StartDate, exp.EndDate),
				NewValue: "",
				Type:     "removed",
			})
			res.HasDiff = true
		}
	}

	// Compare Projects
	baseProjMap := make(map[string]Project)
	for _, p := range base.Projects {
		baseProjMap[p.Name] = p
	}
	targetProjMap := make(map[string]Project)
	for _, p := range target.Projects {
		targetProjMap[p.Name] = p
	}

	for name, p := range targetProjMap {
		if _, exists := baseProjMap[name]; !exists {
			res.Entries = append(res.Entries, DiffEntry{
				Field:    fmt.Sprintf("Project (%s)", name),
				OldValue: "",
				NewValue: p.Description,
				Type:     "added",
			})
			res.HasDiff = true
		}
	}
	for name, p := range baseProjMap {
		if _, exists := targetProjMap[name]; !exists {
			res.Entries = append(res.Entries, DiffEntry{
				Field:    fmt.Sprintf("Project (%s)", name),
				OldValue: p.Description,
				NewValue: "",
				Type:     "removed",
			})
			res.HasDiff = true
		}
	}

	return res
}

func compareStringField(res *DiffResult, field, oldVal, newVal string) {
	oldTrim := strings.TrimSpace(oldVal)
	newTrim := strings.TrimSpace(newVal)
	if oldTrim == newTrim {
		return
	}
	res.HasDiff = true
	if oldTrim == "" && newTrim != "" {
		res.Entries = append(res.Entries, DiffEntry{
			Field:    field,
			OldValue: "",
			NewValue: newTrim,
			Type:     "added",
		})
	} else if oldTrim != "" && newTrim == "" {
		res.Entries = append(res.Entries, DiffEntry{
			Field:    field,
			OldValue: oldTrim,
			NewValue: "",
			Type:     "removed",
		})
	} else {
		res.Entries = append(res.Entries, DiffEntry{
			Field:    field,
			OldValue: oldTrim,
			NewValue: newTrim,
			Type:     "modified",
		})
	}
}

func compareStringSlices(res *DiffResult, field string, oldSlice, newSlice []string) {
	oldMap := make(map[string]bool)
	for _, s := range oldSlice {
		oldMap[strings.TrimSpace(s)] = true
	}
	newMap := make(map[string]bool)
	for _, s := range newSlice {
		newMap[strings.TrimSpace(s)] = true
	}

	for s := range newMap {
		if !oldMap[s] {
			res.Entries = append(res.Entries, DiffEntry{
				Field:    field,
				OldValue: "",
				NewValue: s,
				Type:     "added",
			})
			res.HasDiff = true
		}
	}
	for s := range oldMap {
		if !newMap[s] {
			res.Entries = append(res.Entries, DiffEntry{
				Field:    field,
				OldValue: s,
				NewValue: "",
				Type:     "removed",
			})
			res.HasDiff = true
		}
	}
}

// RenderColoredDiff renders a git-style colored diff to terminal
func (res *DiffResult) RenderColoredDiff() string {
	if !res.HasDiff || len(res.Entries) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).Render("No differences detected. Profile is in sync.")
	}

	addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#05DF72"))
	remStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4D4D"))
	modStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00D2FF"))
	fieldStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F0F6FC"))

	var sb strings.Builder
	sb.WriteString("Profile Differences:\n\n")

	for _, entry := range res.Entries {
		sb.WriteString(fmt.Sprintf("%s:\n", fieldStyle.Render(entry.Field)))
		switch entry.Type {
		case "added":
			sb.WriteString(fmt.Sprintf("  %s\n", addStyle.Render("+ "+entry.NewValue)))
		case "removed":
			sb.WriteString(fmt.Sprintf("  %s\n", remStyle.Render("- "+entry.OldValue)))
		case "modified":
			sb.WriteString(fmt.Sprintf("  %s\n", remStyle.Render("- "+entry.OldValue)))
			sb.WriteString(fmt.Sprintf("  %s\n", addStyle.Render("+ "+entry.NewValue)))
		default:
			sb.WriteString(fmt.Sprintf("  %s\n", modStyle.Render("~ "+entry.NewValue)))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
