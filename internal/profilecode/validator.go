// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package profilecode

import (
	"fmt"
	"strings"
)

// IssueSeverity indicates error vs warning
type IssueSeverity string

const (
	SeverityError   IssueSeverity = "ERROR"
	SeverityWarning IssueSeverity = "WARNING"
	SeverityInfo    IssueSeverity = "INFO"
)

// ValidationIssue represents a single profile lint finding
type ValidationIssue struct {
	Field       string        `json:"field" yaml:"field"`
	Severity    IssueSeverity `json:"severity" yaml:"severity"`
	Message     string        `json:"message" yaml:"message"`
	Suggestion  string        `json:"suggestion,omitempty" yaml:"suggestion,omitempty"`
}

// ValidationResult aggregates issues
type ValidationResult struct {
	Valid    bool              `json:"valid" yaml:"valid"`
	Score    int               `json:"score" yaml:"score"` // 0 - 100
	Issues   []ValidationIssue `json:"issues" yaml:"issues"`
	Headline string            `json:"headline_status" yaml:"headline_status"`
	About    string            `json:"about_status" yaml:"about_status"`
}

// ValidateProfile audits a ProfileAsCode against LinkedIn best practices & constraints
func ValidateProfile(p *ProfileAsCode) *ValidationResult {
	res := &ValidationResult{
		Valid: true,
		Score: 100,
	}

	if p == nil {
		res.Valid = false
		res.Score = 0
		res.Issues = append(res.Issues, ValidationIssue{
			Field:    "Profile",
			Severity: SeverityError,
			Message:  "Profile is empty or uninitialized",
		})
		return res
	}

	// 1. Name validation
	if strings.TrimSpace(p.Name) == "" {
		res.Valid = false
		res.Score -= 20
		res.Issues = append(res.Issues, ValidationIssue{
			Field:      "Name",
			Severity:   SeverityError,
			Message:    "Name is required",
			Suggestion: "Add your full professional name.",
		})
	}

	// 2. Headline validation
	headlineLen := len([]rune(strings.TrimSpace(p.Headline)))
	if headlineLen == 0 {
		res.Valid = false
		res.Score -= 20
		res.Issues = append(res.Issues, ValidationIssue{
			Field:      "Headline",
			Severity:   SeverityError,
			Message:    "Headline is required",
			Suggestion: "Include your primary role, key technical stack, and specialty.",
		})
	} else if headlineLen > 220 {
		res.Score -= 10
		res.Issues = append(res.Issues, ValidationIssue{
			Field:      "Headline",
			Severity:   SeverityError,
			Message:    fmt.Sprintf("Headline exceeds LinkedIn maximum of 220 characters (current: %d)", headlineLen),
			Suggestion: "Shorten headline to fit within 220 characters.",
		})
	} else if headlineLen < 30 {
		res.Score -= 5
		res.Issues = append(res.Issues, ValidationIssue{
			Field:      "Headline",
			Severity:   SeverityWarning,
			Message:    "Headline is very short",
			Suggestion: "Add high-intent keywords (e.g. 'Software Engineer | Distributed Systems | Go & Rust').",
		})
	}

	// 3. About / Summary validation
	aboutLen := len([]rune(strings.TrimSpace(p.About)))
	if aboutLen == 0 {
		res.Score -= 15
		res.Issues = append(res.Issues, ValidationIssue{
			Field:      "About",
			Severity:   SeverityWarning,
			Message:    "About section is empty",
			Suggestion: "A compelling summary increases profile visibility by up to 4x.",
		})
	} else if aboutLen > 2600 {
		res.Score -= 10
		res.Issues = append(res.Issues, ValidationIssue{
			Field:      "About",
			Severity:   SeverityError,
			Message:    fmt.Sprintf("About section exceeds LinkedIn maximum of 2,600 characters (current: %d)", aboutLen),
			Suggestion: "Trim summary to fit within limits.",
		})
	} else if aboutLen < 150 {
		res.Score -= 5
		res.Issues = append(res.Issues, ValidationIssue{
			Field:      "About",
			Severity:   SeverityInfo,
			Message:    "About section could be more descriptive",
			Suggestion: "Highlight career highlights, key achievements, and technologies you love.",
		})
	}

	// 4. Skills validation
	if len(p.Skills) == 0 {
		res.Score -= 15
		res.Issues = append(res.Issues, ValidationIssue{
			Field:      "Skills",
			Severity:   SeverityWarning,
			Message:    "No skills specified",
			Suggestion: "Add at least 5-10 core technical and domain skills.",
		})
	} else {
		// Check duplicates
		seenSkills := make(map[string]bool)
		for _, s := range p.Skills {
			norm := strings.ToLower(strings.TrimSpace(s))
			if seenSkills[norm] {
				res.Issues = append(res.Issues, ValidationIssue{
					Field:      "Skills",
					Severity:   SeverityWarning,
					Message:    fmt.Sprintf("Duplicate skill detected: '%s'", s),
					Suggestion: "Remove duplicate entries.",
				})
			}
			seenSkills[norm] = true
		}
	}

	// 5. Experience validation
	if len(p.Experience) == 0 {
		res.Score -= 15
		res.Issues = append(res.Issues, ValidationIssue{
			Field:      "Experience",
			Severity:   SeverityWarning,
			Message:    "No work experience listed",
			Suggestion: "Add your current or past professional roles.",
		})
	} else {
		for i, exp := range p.Experience {
			if strings.TrimSpace(exp.Company) == "" || strings.TrimSpace(exp.Role) == "" {
				res.Score -= 5
				res.Issues = append(res.Issues, ValidationIssue{
					Field:      fmt.Sprintf("Experience[%d]", i+1),
					Severity:   SeverityError,
					Message:    "Experience entry is missing Company or Role",
					Suggestion: "Specify both company and job title.",
				})
			}
			if strings.TrimSpace(exp.Description) == "" {
				res.Score -= 3
				res.Issues = append(res.Issues, ValidationIssue{
					Field:      fmt.Sprintf("Experience (%s @ %s)", exp.Role, exp.Company),
					Severity:   SeverityWarning,
					Message:    "Experience entry lacks description/bullet points",
					Suggestion: "Use quantifiable achievements (e.g. 'Reduced latency by 40%...').",
				})
			}
		}
	}

	// 6. Projects validation
	for i, proj := range p.Projects {
		if strings.TrimSpace(proj.Name) == "" {
			res.Issues = append(res.Issues, ValidationIssue{
				Field:    fmt.Sprintf("Project[%d]", i+1),
				Severity: SeverityWarning,
				Message:  "Project missing a name",
			})
		}
		if strings.TrimSpace(proj.Description) == "" {
			res.Issues = append(res.Issues, ValidationIssue{
				Field:      fmt.Sprintf("Project (%s)", proj.Name),
				Severity:   SeverityInfo,
				Message:    "Project description is empty",
				Suggestion: "Explain problem solved and technologies used.",
			})
		}
	}

	if res.Score < 0 {
		res.Score = 0
	}
	if len(res.Issues) > 0 {
		for _, issue := range res.Issues {
			if issue.Severity == SeverityError {
				res.Valid = false
				break
			}
		}
	}

	return res
}
