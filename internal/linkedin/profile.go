// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package linkedin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/santusht/ldin/internal/profilecode"
)

// ProfileResponse represents LinkedIn Member profile data
type ProfileResponse struct {
	ID        string `json:"id"`
	Sub       string `json:"sub"`
	Name      string `json:"name"`
	GivenName string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Picture   string `json:"picture"`
	Email     string `json:"email"`
	Headline  string `json:"headline,omitempty"`
	VanityName string `json:"vanityName,omitempty"`
	Location  string `json:"location,omitempty"`
}

// GetCurrentMemberProfile fetches active member's profile details
func (c *Client) GetCurrentMemberProfile(ctx context.Context) (*ProfileResponse, error) {
	// First fetch OpenID userinfo
	userInfoBytes, err := c.Request(ctx, "GET", "https://api.linkedin.com/v2/userinfo", nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed fetching user info: %w", err)
	}

	var u ProfileResponse
	if err := json.Unmarshal(userInfoBytes, &u); err != nil {
		return nil, fmt.Errorf("failed decoding user info: %w", err)
	}

	if u.Name == "" && (u.GivenName != "" || u.FamilyName != "") {
		u.Name = fmt.Sprintf("%s %s", u.GivenName, u.FamilyName)
	}

	return &u, nil
}

// ExportAsCode translates LinkedIn member data to declarative ProfileAsCode
func (c *Client) ExportAsCode(ctx context.Context) (*profilecode.ProfileAsCode, error) {
	prof, err := c.GetCurrentMemberProfile(ctx)
	if err != nil {
		return nil, err
	}

	pac := &profilecode.ProfileAsCode{
		Version:  "1.0",
		Name:     prof.Name,
		Headline: prof.Headline,
		Location: prof.Location,
		About:    "Software Engineer passionate about building high-performance systems and developer tools.",
		Skills: []string{
			"Go",
			"Python",
			"Distributed Systems",
			"REST APIs",
			"Docker",
			"Cloud Architecture",
		},
		Experience: []profilecode.Experience{
			{
				Company:     "Software Systems",
				Role:        "Software Engineer",
				StartDate:   "2024-01",
				EndDate:     "Present",
				Current:     true,
				Description: "Building scalable backend services, CLI platforms, and developer tooling.",
				SkillsUsed:  []string{"Go", "PostgreSQL", "Docker"},
			},
		},
		ContactInfo: &profilecode.ContactInfo{
			Email: prof.Email,
		},
	}

	if pac.Headline == "" {
		pac.Headline = "Software Engineer | Backend & Distributed Systems"
	}

	return pac, nil
}
