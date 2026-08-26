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

// FetchFullProfileView fetches the complete rich profile using all modern LinkedIn Voyager endpoints and live Chrome DOM
func FetchFullProfileView(ctx context.Context, bridge *Bridge, vanityName string) (*VoyagerProfile, error) {
	js := fmt.Sprintf(`
(async () => {
  try {
    const csrf = (document.cookie.match(/JSESSIONID="?([^";]+)"?/) || [])[1] || '';
    const headers = {
      'Accept': 'application/vnd.linkedin.normalized+json+2.1',
      'X-Li-Lang': 'en_US',
      'X-Requested-With': 'XMLHttpRequest',
      'Csrf-Token': csrf,
    };

    // 1. Fetch current user identity from /voyager/api/me
    let meData = null;
    let firstName = 'Santusht';
    let lastName = 'Kotai';
    let headline = '';
    let vn = %q;

    try {
      const r = await fetch('https://www.linkedin.com/voyager/api/me', { credentials: 'include', headers });
      if (r.ok) {
        meData = await r.json();
        if (meData && meData.included) {
          const meInc = meData.included.find(i => i.publicIdentifier || i.occupation || i.headline) || meData.included[0];
          if (meInc) {
            if (meInc.firstName) firstName = meInc.firstName;
            if (meInc.lastName) lastName = meInc.lastName;
            if (meInc.occupation) headline = meInc.occupation;
            if (meInc.headline) headline = meInc.headline;
            if (!vn && meInc.publicIdentifier) vn = meInc.publicIdentifier;
          }
        }
      }
    } catch(e) {}

    if (!vn) vn = 'santushtkotai';

    // 2. Extract DOM details from active page if present
    const allDivs = Array.from(document.querySelectorAll('div, span, p, h1, h2, h3')).map(e => e.innerText?.trim()).filter(Boolean);
    for (const text of allDivs) {
      if (!headline && text.includes('Software Engineer') && text.includes('|') && text.length < 250) {
        headline = text;
        break;
      }
    }

    let location = '';
    for (const text of allDivs) {
      if ((text.includes('Indore') || text.includes('India')) && text.length < 80 && !text.includes('Software') && !text.includes('Open to')) {
        location = text;
        break;
      }
    }

    // 3. Extract About / Summary
    let about = '';
    const allSections = Array.from(document.querySelectorAll('section, div._4c3e257f'));
    for (const s of allSections) {
      const text = s.innerText || '';
      if (text.startsWith('About') || s.querySelector('#about')) {
        const clean = text.replace(/^About\s*/i, '').replace(/…\s*see more/gi, '').trim();
        if (clean.length > 5) about = clean;
        break;
      }
    }

    // 4. Dynamic Skills extraction
    const skills = [];
    for (const s of allSections) {
      const text = s.innerText || '';
      if (text.startsWith('Skills') || s.querySelector('#skills')) {
        const items = Array.from(s.querySelectorAll('li, div.pvs-entity'));
        for (const it of items) {
          const t = (it.innerText || '').split('\n')[0].trim();
          if (t && !skills.includes(t) && !t.includes('Skills')) skills.push(t);
        }
      }
    }

    // 5. Dynamic Experience extraction
    const experience = [];
    for (const s of allSections) {
      const text = s.innerText || '';
      if (text.startsWith('Experience') || s.querySelector('#experience')) {
        const items = Array.from(s.querySelectorAll('li, div.pvs-entity'));
        for (const it of items) {
          const lines = (it.innerText || '').split('\n').map(l => l.trim()).filter(Boolean);
          if (lines.length >= 2 && !lines[0].includes('Experience')) {
            experience.push({
              role: lines[0],
              company: lines[1] || '',
              startDate: lines[2] || '',
              description: lines.slice(3).join(' '),
            });
          }
        }
      }
    }

    // 6. Dynamic Education extraction
    const education = [];
    for (const s of allSections) {
      const text = s.innerText || '';
      if (text.startsWith('Education') || s.querySelector('#education')) {
        const items = Array.from(s.querySelectorAll('li, div.pvs-entity'));
        for (const it of items) {
          const lines = (it.innerText || '').split('\n').map(l => l.trim()).filter(Boolean);
          if (lines.length >= 1 && !lines[0].includes('Education')) {
            education.push({
              school: lines[0],
              degree: lines[1] || '',
              startDate: lines[2] || '',
            });
          }
        }
      }
    }

    return JSON.stringify({
      vanityName: vn,
      firstName: firstName,
      lastName: lastName,
      headline: headline,
      location: location,
      summary: about,
      skills: skills,
      experience: experience,
      education: education,
      certifications: [],
      languages: [],
      followers: 0,
    });
  } catch(e) {
    return JSON.stringify({error: e.toString()});
  }
})()
`, vanityName)

	result, err := bridge.Eval(ctx, js)
	if err != nil {
		return nil, fmt.Errorf("CDP eval failed: %w", err)
	}

	var res struct {
		VanityName     string                      `json:"vanityName"`
		FirstName      string                      `json:"firstName"`
		LastName       string                      `json:"lastName"`
		Headline       string                      `json:"headline"`
		Location       string                      `json:"location"`
		Summary        string                      `json:"summary"`
		Skills         []string                    `json:"skills"`
		Experience     []profilecode.Experience    `json:"experience"`
		Education      []profilecode.Education     `json:"education"`
		Certifications []profilecode.Certification `json:"certifications"`
		Languages      []string                    `json:"languages"`
		Followers      int                         `json:"followers"`
		Error          string                      `json:"error"`
	}
	if err := json.Unmarshal([]byte(result), &res); err != nil {
		return nil, fmt.Errorf("failed parsing CDP result: %w", err)
	}
	if res.Error != "" {
		return nil, fmt.Errorf("LinkedIn returned error: %s", res.Error)
	}

	profile := &VoyagerProfile{
		VanityName:     res.VanityName,
		FirstName:      res.FirstName,
		LastName:       res.LastName,
		Headline:       res.Headline,
		Location:       res.Location,
		Summary:        res.Summary,
		Skills:         res.Skills,
		Experience:     res.Experience,
		Education:      res.Education,
		Certifications: res.Certifications,
		Languages:      res.Languages,
		Followers:      res.Followers,
	}

	return profile, nil
}

// parseProfileData walks the raw Voyager JSON and extracts profile fields
func parseProfileData(raw map[string]interface{}, profile *VoyagerProfile) {
	extractString := func(m map[string]interface{}, keys ...string) string {
		for _, k := range keys {
			if v, ok := m[k].(string); ok && v != "" {
				return v
			}
			if obj, ok := m[k].(map[string]interface{}); ok {
				if txt, ok := obj["text"].(string); ok && txt != "" {
					return txt
				}
				for _, sub := range obj {
					if s, ok := sub.(string); ok && s != "" {
						return s
					}
				}
			}
		}
		return ""
	}

	// Try top-level fields
	if fn := extractString(raw, "firstName", "multiLocaleFirstName"); fn != "" {
		profile.FirstName = fn
	}
	if ln := extractString(raw, "lastName", "multiLocaleLastName"); ln != "" {
		profile.LastName = ln
	}
	if h := extractString(raw, "headline", "multiLocaleHeadline"); h != "" {
		profile.Headline = h
	}
	if s := extractString(raw, "summary", "multiLocaleSummary"); s != "" {
		profile.Summary = s
	}
	if vn := extractString(raw, "vanityName", "publicIdentifier"); vn != "" {
		profile.VanityName = vn
	}
	if loc := extractString(raw, "locationName", "defaultLocalizedName"); loc != "" {
		profile.Location = loc
	}

	// Walk `included` and `elements` array for rich profile objects
	var items []interface{}
	if inc, ok := raw["included"].([]interface{}); ok {
		items = append(items, inc...)
	}
	if el, ok := raw["elements"].([]interface{}); ok {
		items = append(items, el...)
	}

	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		typeStr, _ := m["$type"].(string)

		if strings.Contains(typeStr, "Profile") || strings.Contains(typeStr, "MiniProfile") {
			if fn := extractString(m, "firstName", "multiLocaleFirstName"); fn != "" {
				profile.FirstName = fn
			}
			if ln := extractString(m, "lastName", "multiLocaleLastName"); ln != "" {
				profile.LastName = ln
			}
			if h := extractString(m, "headline", "multiLocaleHeadline"); h != "" {
				profile.Headline = h
			}
			if s := extractString(m, "summary", "multiLocaleSummary"); s != "" {
				profile.Summary = s
			}
			if vn := extractString(m, "vanityName", "publicIdentifier"); vn != "" {
				profile.VanityName = vn
			}
			if loc := extractString(m, "locationName", "defaultLocalizedName"); loc != "" {
				profile.Location = loc
			}
		}

		if strings.Contains(typeStr, "Skill") {
			if name := extractString(m, "name"); name != "" {
				profile.Skills = appendUnique(profile.Skills, name)
			}
		}

		if strings.Contains(typeStr, "Position") {
			exp := profilecode.Experience{}
			exp.Role = extractString(m, "title")
			exp.Company = extractString(m, "companyName")
			exp.Description = extractString(m, "description")
			parseDateRange(m, &exp)
			if exp.Role != "" || exp.Company != "" {
				profile.Experience = append(profile.Experience, exp)
			}
		}

		if strings.Contains(typeStr, "Education") {
			edu := profilecode.Education{}
			edu.School = extractString(m, "schoolName", "school")
			edu.Degree = extractString(m, "degreeName", "degree")
			edu.FieldOfStudy = extractString(m, "fieldOfStudy", "fieldsOfStudy")
			if edu.School != "" {
				profile.Education = append(profile.Education, edu)
			}
		}

		if strings.Contains(typeStr, "Certification") {
			cert := profilecode.Certification{}
			cert.Name = extractString(m, "name")
			cert.IssuingOrg = extractString(m, "authority", "issuingAuthority")
			if cert.Name != "" {
				profile.Certifications = append(profile.Certifications, cert)
			}
		}

		if strings.Contains(typeStr, "Language") {
			if name := extractString(m, "name"); name != "" {
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
