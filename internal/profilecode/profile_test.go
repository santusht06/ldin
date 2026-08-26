// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package profilecode

import (
	"testing"
)

func TestProfileValidation(t *testing.T) {
	p := &ProfileAsCode{
		Name:     "Santusht Kotai",
		Headline: "Software Engineer | Backend Engineering | Distributed Systems",
		About:    "Backend engineer with deep experience building scalable microservices and developer tools in Go and Python.",
		Skills:   []string{"Go", "Python", "Docker", "PostgreSQL", "Redis"},
		Experience: []Experience{
			{
				Company:     "ShareXpress",
				Role:        "Software Engineer",
				StartDate:   "2024-01",
				Description: "Designing high-scale distributed backend systems.",
			},
		},
	}

	res := ValidateProfile(p)
	if !res.Valid {
		t.Fatalf("expected profile to be valid, got invalid with %d issues", len(res.Issues))
	}
	if res.Score < 80 {
		t.Fatalf("expected profile score >= 80, got %d", res.Score)
	}
}

func TestProfileDiff(t *testing.T) {
	base := &ProfileAsCode{
		Name:     "Santusht Kotai",
		Headline: "Python Developer",
		About:    "Building basic scripts.",
		Skills:   []string{"Python"},
	}

	target := &ProfileAsCode{
		Name:     "Santusht Kotai",
		Headline: "Software Engineer | Distributed Systems",
		About:    "Architecting high throughput distributed backends.",
		Skills:   []string{"Python", "Go", "Distributed Systems"},
	}

	diff := CompareProfiles(base, target)
	if !diff.HasDiff {
		t.Fatalf("expected diff to have differences")
	}

	if len(diff.Entries) == 0 {
		t.Fatalf("expected diff entries, got 0")
	}

	rendered := diff.RenderColoredDiff()
	if rendered == "" {
		t.Fatalf("rendered diff is empty")
	}
}

func TestProfileYAMLSerialization(t *testing.T) {
	p := &ProfileAsCode{
		Name:     "Santusht Kotai",
		Headline: "Software Engineer",
		Skills:   []string{"Go", "Docker"},
	}

	yamlStr, err := p.ToYAML()
	if err != nil {
		t.Fatalf("failed to convert to YAML: %v", err)
	}

	if yamlStr == "" {
		t.Fatalf("empty YAML output")
	}
}
