// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cdp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santusht/ldin/internal/profilecode"
)

// VoyagerProfile holds the real-time profile data fetched via CDP
type VoyagerProfile struct {
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
	Certifications []profilecode.Certification
	Languages      []string
	Connections    int
	Followers      int
}

// FetchProfile fetches a LinkedIn member's full profile via CDP browser injection.
// Chrome makes the request using its own TLS + real session cookies — bypassing JA3 detection.
func FetchProfile(ctx context.Context, bridge *Bridge, vanityName string) (*VoyagerProfile, error) {
	url := fmt.Sprintf("https://www.linkedin.com/voyager/api/identity/profiles/%s", vanityName)

	rawJSON, err := bridge.FetchURL(ctx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("CDP fetch of profile failed: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(rawJSON), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse profile JSON: %w\n\nRaw (first 200 chars): %.200s", err, rawJSON)
	}

	profile := &VoyagerProfile{VanityName: vanityName}
	parseProfileData(raw, profile)

	// Fetch skills separately if empty
	if len(profile.Skills) == 0 {
		skills, _ := FetchSkills(ctx, bridge, vanityName)
		profile.Skills = skills
	}

	return profile, nil
}

// FetchSkills fetches skills for a LinkedIn member via CDP
func FetchSkills(ctx context.Context, bridge *Bridge, vanityName string) ([]string, error) {
	url := fmt.Sprintf("https://www.linkedin.com/voyager/api/identity/profiles/%s/skills", vanityName)
	rawJSON, err := bridge.FetchURL(ctx, url, nil)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(rawJSON), &raw); err != nil {
		return nil, err
	}

	var skills []string
	if elements, ok := raw["elements"].([]interface{}); ok {
		for _, el := range elements {
			if m, ok := el.(map[string]interface{}); ok {
				if name, ok := m["name"].(string); ok && name != "" {
					skills = appendUnique(skills, name)
				}
			}
		}
	}
	return skills, nil
}

// FetchFullProfileView fetches the complete rich profile using the profileView endpoint via CDP
func FetchFullProfileView(ctx context.Context, bridge *Bridge, vanityName string) (*VoyagerProfile, error) {
	// This runs INSIDE Chrome, so LinkedIn sees it as a real browser request
	js := fmt.Sprintf(`
(async () => {
  try {
    const csrf = (document.cookie.match(/JSESSIONID="?([^";]+)"?/) || [])[1] || '';
    const endpoints = [
      '/voyager/api/identity/profiles/%s/profileView',
      '/voyager/api/identity/profiles/%s',
      '/voyager/api/identity/dash/profiles?q=vanityName&vanityName=%s',
    ];

    for (const ep of endpoints) {
      try {
        const resp = await fetch('https://www.linkedin.com' + ep, {
          credentials: 'include',
          headers: {
            'Accept': 'application/vnd.linkedin.normalized+json+2.1',
            'X-Li-Lang': 'en_US',
            'X-Requested-With': 'XMLHttpRequest',
            'Csrf-Token': csrf,
          }
        });
        if (resp.ok) {
          const data = await resp.json();
          return JSON.stringify({status: resp.status, endpoint: ep, data: data});
        }
      } catch(e) { continue; }
    }
    return JSON.stringify({error: 'all endpoints failed'});
  } catch(e) {
    return JSON.stringify({error: e.toString()});
  }
})()
`, vanityName, vanityName, vanityName)

	result, err := bridge.Eval(ctx, js)
	if err != nil {
		return nil, fmt.Errorf("CDP eval failed: %w", err)
	}

	var wrapper struct {
		Status   int                    `json:"status"`
		Endpoint string                 `json:"endpoint"`
		Data     map[string]interface{} `json:"data"`
		Error    string                 `json:"error"`
	}
	if err := json.Unmarshal([]byte(result), &wrapper); err != nil {
		return nil, fmt.Errorf("failed parsing CDP result: %w", err)
	}
	if wrapper.Error != "" {
		return nil, fmt.Errorf("LinkedIn returned error: %s", wrapper.Error)
	}

	profile := &VoyagerProfile{VanityName: vanityName}
	parseProfileData(wrapper.Data, profile)

	if len(profile.Skills) == 0 {
		skills, _ := FetchSkills(ctx, bridge, vanityName)
		profile.Skills = skills
	}

	return profile, nil
}

// parseProfileData walks the raw Voyager JSON and extracts profile fields
func parseProfileData(raw map[string]interface{}, profile *VoyagerProfile) {
	// Try top-level fields first (simple profile endpoint)
	if fn, ok := raw["firstName"].(string); ok && fn != "" {
		profile.FirstName = fn
	}
	if ln, ok := raw["lastName"].(string); ok && ln != "" {
		profile.LastName = ln
	}
	if h, ok := raw["headline"].(string); ok && h != "" {
		profile.Headline = h
	}
	if s, ok := raw["summary"].(string); ok && s != "" {
		profile.Summary = s
	}
	if vn, ok := raw["vanityName"].(string); ok && vn != "" {
		profile.VanityName = vn
	}

	// Parse geo location
	if geo, ok := raw["geoLocation"].(map[string]interface{}); ok {
		if geo2, ok := geo["geo"].(map[string]interface{}); ok {
			if name, ok := geo2["defaultLocalizedName"].(string); ok {
				profile.Location = name
			}
		}
	}
	if profile.Location == "" {
		if locName, ok := raw["locationName"].(string); ok {
			profile.Location = locName
		}
	}

	// Walk `included` array for richer profileView format
	included, _ := raw["included"].([]interface{})
	for _, item := range included {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		typeStr, _ := m["$type"].(string)

		if strings.Contains(typeStr, "MiniProfile") || strings.Contains(typeStr, "com.linkedin.voyager.identity.profile.Profile") {
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

		if strings.Contains(typeStr, "Skill") {
			if name, ok := m["name"].(string); ok && name != "" {
				profile.Skills = appendUnique(profile.Skills, name)
			}
		}

		if strings.Contains(typeStr, "Position") {
			exp := profilecode.Experience{}
			if title, ok := m["title"].(string); ok {
				exp.Role = title
			}
			if comp, ok := m["companyName"].(string); ok {
				exp.Company = comp
			}
			if desc, ok := m["description"].(string); ok {
				exp.Description = desc
			}
			parseDateRange(m, &exp)
			if exp.Role != "" || exp.Company != "" {
				profile.Experience = append(profile.Experience, exp)
			}
		}

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

		if strings.Contains(typeStr, "Language") {
			if name, ok := m["name"].(string); ok && name != "" {
				profile.Languages = appendUnique(profile.Languages, name)
			}
		}
	}
}

func parseDateRange(m map[string]interface{}, exp *profilecode.Experience) {
	dateRange, ok := m["dateRange"].(map[string]interface{})
	if !ok {
		return
	}
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

func appendUnique(slice []string, s string) []string {
	for _, existing := range slice {
		if existing == s {
			return slice
		}
	}
	return append(slice, s)
}
