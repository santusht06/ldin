// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package linkedin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/santusht/ldin/internal/profilecode"
)

// ProfileResponse represents LinkedIn Member profile data
type ProfileResponse struct {
	ID         string `json:"id"`
	Sub        string `json:"sub"`
	Name       string `json:"name"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Picture    string `json:"picture"`
	Email      string `json:"email"`
	Headline   string `json:"headline,omitempty"`
	VanityName string `json:"vanityName,omitempty"`
	Location   string `json:"location,omitempty"`
}

// VoyagerProfileData holds the rich real-time profile from LinkedIn's Voyager API
type VoyagerProfileData struct {
	VanityName     string
	FirstName      string
	LastName       string
	Headline       string
	Summary        string
	Location       string
	ProfilePicture string
	Skills         []string
	Experience     []profilecode.Experience
	Education      []profilecode.Education
	Languages      []string
	Certifications []profilecode.Certification
}

// GetCurrentMemberProfile fetches active member's profile details via OpenID
func (c *Client) GetCurrentMemberProfile(ctx context.Context) (*ProfileResponse, error) {
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

// GetVoyagerProfile uses LinkedIn's internal Voyager API to fetch full real-time profile
// This is the same API LinkedIn's own website uses — no enterprise approval required.
func (c *Client) GetVoyagerProfile(ctx context.Context, vanityName string) (*VoyagerProfileData, error) {
	if vanityName == "" && c.Profile != nil {
		vanityName = c.Profile.VanityName
	}
	if vanityName == "" {
		vanityName = "me"
	}

	sessionCookie := ""
	csrfToken := ""
	if c.Profile != nil {
		sessionCookie = c.Profile.SessionCookie
		csrfToken = c.Profile.CSRFToken
	}
	if sessionCookie == "" {
		sessionCookie = c.AccessToken
	}

	buildReq := func(endpoint string) (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "application/vnd.linkedin.normalized+json+2.1")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		req.Header.Set("X-Li-Lang", "en_US")
		req.Header.Set("X-Li-Track", `{"clientVersion":"1.13.14694.2","mpVersion":"1.13.14694.2","osName":"web","timezoneOffset":5.5}`)
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		if sessionCookie != "" {
			cookieStr := fmt.Sprintf("li_at=%s", sessionCookie)
			if csrfToken != "" {
				cookieStr += fmt.Sprintf("; JSESSIONID=\"%s\"", csrfToken)
				req.Header.Set("Csrf-Token", csrfToken)
			}
			req.Header.Set("Cookie", cookieStr)
		}
		return req, nil
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}

	// Try multiple Voyager endpoints in priority order
	endpoints := []string{
		// Primary: identity dash profiles (current)
		fmt.Sprintf("https://www.linkedin.com/voyager/api/identity/dash/profiles?q=memberIdentityUrn&memberIdentityUrn=urn%%3Ali%%3Amember%%3A(%%3Cli%%3AvanityName%%3A%s)&decorationId=com.linkedin.voyager.dash.deco.identity.profile.FullProfileWithEntities-93", vanityName),
		// Secondary: skills, positions, education via memberIdentityCards
		fmt.Sprintf("https://www.linkedin.com/voyager/api/identity/profiles/%s/profileContactInfo", vanityName),
		// Tertiary: simple profile by vanity URL (older format still active)
		fmt.Sprintf("https://www.linkedin.com/voyager/api/identity/profiles?q=vanityName&vanityName=%s", vanityName),
	}

	var lastErr error
	for _, ep := range endpoints {
		req, err := buildReq(ep)
		if err != nil {
			lastErr = err
			continue
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			resp.Body.Close()
			return nil, fmt.Errorf("authentication expired — run `ldin auth token <new_token>`")
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
			lastErr = fmt.Errorf("voyager API returned HTTP %d", resp.StatusCode)
			continue
		}

		var raw map[string]interface{}
		decodeErr := json.NewDecoder(resp.Body).Decode(&raw)
		resp.Body.Close()
		if decodeErr != nil {
			lastErr = decodeErr
			continue
		}

		profile := parseVoyagerProfile(raw, vanityName)
		// Consider it a success if we got at least a name or skills
		if profile.FirstName != "" || profile.Headline != "" || len(profile.Skills) > 0 {
			return profile, nil
		}
		lastErr = fmt.Errorf("profile parsed but returned empty data")
	}

	return nil, fmt.Errorf("all Voyager profile endpoints failed: %w", lastErr)
}

// GetVoyagerSkills fetches skills from Voyager API
func (c *Client) GetVoyagerSkills(ctx context.Context, vanityName string) ([]string, error) {
	endpoint := fmt.Sprintf("https://www.linkedin.com/voyager/api/identity/profiles/%s/skills", vanityName)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/vnd.linkedin.normalized+json+2.1")

	if c.Profile != nil && c.Profile.SessionCookie != "" {
		cookieStr := fmt.Sprintf("li_at=%s", c.Profile.SessionCookie)
		if c.Profile.CSRFToken != "" {
			cookieStr += fmt.Sprintf("; JSESSIONID=\"%s\"", c.Profile.CSRFToken)
			req.Header.Set("Csrf-Token", c.Profile.CSRFToken)
		}
		req.Header.Set("Cookie", cookieStr)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("skills API returned HTTP %d", resp.StatusCode)
	}

	var raw map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	skills := []string{}
	if elements, ok := raw["elements"].([]interface{}); ok {
		for _, el := range elements {
			if m, ok := el.(map[string]interface{}); ok {
				if name, ok := m["name"].(string); ok && name != "" {
					skills = append(skills, name)
				}
			}
		}
	}
	return skills, nil
}

// GetRichProfile returns real-time profile — tries Voyager first, falls back to OpenID
func (c *Client) GetRichProfile(ctx context.Context) (*VoyagerProfileData, error) {
	vanityName := ""
	if c.Profile != nil {
		vanityName = c.Profile.VanityName
	}

	// Try Voyager API first for real-time full profile
	profile, err := c.GetVoyagerProfile(ctx, vanityName)
	if err != nil {
		// Fall back to OpenID for at least basic info
		basic, basicErr := c.GetCurrentMemberProfile(ctx)
		if basicErr != nil {
			return nil, err // return original error
		}
		return &VoyagerProfileData{
			VanityName: vanityName,
			FirstName:  basic.GivenName,
			LastName:   basic.FamilyName,
			Headline:   basic.Headline,
			Location:   basic.Location,
		}, nil
	}

	// Also try to pull skills if not already populated
	if len(profile.Skills) == 0 && vanityName != "" && vanityName != "me" {
		skills, skillErr := c.GetVoyagerSkills(ctx, vanityName)
		if skillErr == nil && len(skills) > 0 {
			profile.Skills = skills
		}
	}

	return profile, nil
}

// parseVoyagerProfile extracts profile fields from raw LinkedIn Voyager API response
func parseVoyagerProfile(raw map[string]interface{}, vanityName string) *VoyagerProfileData {
	profile := &VoyagerProfileData{VanityName: vanityName}

	included, _ := raw["included"].([]interface{})
	for _, item := range included {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		typeStr, _ := m["$type"].(string)

		// Extract main profile entity
		if strings.Contains(typeStr, "MiniProfile") || strings.Contains(typeStr, "Profile") {
			if fn, ok := m["firstName"].(string); ok && fn != "" {
				profile.FirstName = fn
			}
			if ln, ok := m["lastName"].(string); ok && ln != "" {
				profile.LastName = ln
			}
			if h, ok := m["headline"].(string); ok && h != "" {
				profile.Headline = h
			}
			if s, ok := m["summary"].(string); ok && s != "" {
				profile.Summary = s
			}
			if vn, ok := m["vanityName"].(string); ok && vn != "" {
				profile.VanityName = vn
			}
		}

		// Extract geo location
		if strings.Contains(typeStr, "ProfileLocation") || strings.Contains(typeStr, "GeoLocation") {
			if loc, ok := m["countryName"].(string); ok && loc != "" {
				profile.Location = loc
			}
			if city, ok := m["city"].(string); ok && city != "" {
				profile.Location = city
			}
			if defaultLocalizedName, ok := m["defaultLocalizedName"].(string); ok && defaultLocalizedName != "" {
				profile.Location = defaultLocalizedName
			}
		}

		// Extract skills
		if strings.Contains(typeStr, "Skill") {
			if name, ok := m["name"].(string); ok && name != "" {
				profile.Skills = appendUnique(profile.Skills, name)
			}
		}

		// Extract positions (experience)
		if strings.Contains(typeStr, "Position") {
			exp := profilecode.Experience{}
			if title, ok := m["title"].(string); ok {
				exp.Role = title
			}
			if desc, ok := m["description"].(string); ok {
				exp.Description = desc
			}
			if comp, ok := m["companyName"].(string); ok {
				exp.Company = comp
			}
			if dateRange, ok := m["dateRange"].(map[string]interface{}); ok {
				if start, ok := dateRange["start"].(map[string]interface{}); ok {
					year, _ := start["year"].(float64)
					month, _ := start["month"].(float64)
					if year > 0 {
						exp.StartDate = fmt.Sprintf("%.0f-%02.0f", year, month)
					}
				}
				if end, ok := dateRange["end"].(map[string]interface{}); ok {
					year, _ := end["year"].(float64)
					month, _ := end["month"].(float64)
					if year > 0 {
						exp.EndDate = fmt.Sprintf("%.0f-%02.0f", year, month)
					}
				} else {
					exp.EndDate = "Present"
					exp.Current = true
				}
			}
			if exp.Role != "" || exp.Company != "" {
				profile.Experience = append(profile.Experience, exp)
			}
		}

		// Extract education
		if strings.Contains(typeStr, "Education") {
			edu := profilecode.Education{}
			if school, ok := m["schoolName"].(string); ok {
				edu.School = school
			}
			if degree, ok := m["degreeName"].(string); ok {
				edu.Degree = degree
			}
			if field, ok := m["fieldOfStudy"].(string); ok {
				edu.FieldOfStudy = field
			}
			if edu.School != "" {
				profile.Education = append(profile.Education, edu)
			}
		}

		// Extract certifications
		if strings.Contains(typeStr, "Certification") {
			cert := profilecode.Certification{}
			if name, ok := m["name"].(string); ok {
				cert.Name = name
			}
			if issuer, ok := m["authority"].(string); ok {
				cert.IssuingOrg = issuer
			}
			if cert.Name != "" {
				profile.Certifications = append(profile.Certifications, cert)
			}
		}

		// Extract languages
		if strings.Contains(typeStr, "Language") {
			if name, ok := m["name"].(string); ok && name != "" {
				profile.Languages = appendUnique(profile.Languages, name)
			}
		}
	}

	return profile
}

func appendUnique(slice []string, s string) []string {
	for _, existing := range slice {
		if existing == s {
			return slice
		}
	}
	return append(slice, s)
}

func newCookieJar() (http.CookieJar, error) {
	type simpleJar struct {
		cookies map[string][]*http.Cookie
	}
	// Use net/http/cookiejar via url
	return nil, nil // returns nil, http.Client uses its own jar
}

// ExportAsCode translates real-time profile to declarative ProfileAsCode
func (c *Client) ExportAsCode(ctx context.Context) (*profilecode.ProfileAsCode, error) {
	rich, err := c.GetRichProfile(ctx)
	if err != nil {
		// last resort: OpenID basic
		basic, basicErr := c.GetCurrentMemberProfile(ctx)
		if basicErr != nil {
			return nil, err
		}
		return &profilecode.ProfileAsCode{
			Version:  "1.0",
			Name:     basic.Name,
			Headline: basic.Headline,
			Location: basic.Location,
			ContactInfo: &profilecode.ContactInfo{
				Email: basic.Email,
			},
		}, nil
	}

	pac := &profilecode.ProfileAsCode{
		Version:        "1.0",
		Name:           strings.TrimSpace(rich.FirstName + " " + rich.LastName),
		Headline:       rich.Headline,
		Location:       rich.Location,
		About:          rich.Summary,
		Skills:         rich.Skills,
		Experience:     rich.Experience,
		Education:      rich.Education,
		Languages:      rich.Languages,
		Certifications: rich.Certifications,
	}

	if pac.Headline == "" {
		pac.Headline = "Software Engineer"
	}

	return pac, nil
}

// GetProfileURL returns public profile URL from vanity name
func (c *Client) GetProfileURL(vanityName string) string {
	return fmt.Sprintf("https://www.linkedin.com/in/%s/", url.PathEscape(vanityName))
}
