// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

// Package browser provides a headless Chromium engine for ldin.
// Uses go-rod to run Chromium invisibly in the background — no GUI window.
// LinkedIn sees a real Chrome browser TLS fingerprint, bypassing bot detection.
package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/santusht/ldin/internal/cdp"
	"github.com/santusht/ldin/internal/profilecode"
)

// HeadlessBrowser wraps a headless Chromium instance managed by go-rod
type HeadlessBrowser struct {
	browser    *rod.Browser
	page       *rod.Page
	liAt       string
	jsessionID string
}

// Launch starts a headless Chromium instance (downloads if not present).
// No GUI window is shown. The browser runs silently in the background.
func Launch(ctx context.Context) (*HeadlessBrowser, error) {
	// Find or download Chromium
	u, err := launcher.New().
		Headless(true).
		// Disable CORS & web security so fetch() can hit linkedin.com from about:blank
		Set("disable-web-security").
		Set("disable-site-isolation-trials").
		Set("disable-features", "IsolateOrigins,site-per-process").
		Set("disable-blink-features", "AutomationControlled").
		Set("exclude-switches", "enable-automation").
		NoSandbox(true).
		Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch headless Chromium: %w\n\nRun `ldin browser setup` to download Chromium", err)
	}

	b := rod.New().ControlURL(u).MustConnect()

	if err := b.IgnoreCertErrors(true); err != nil {
		b.MustClose()
		return nil, err
	}

	return &HeadlessBrowser{browser: b}, nil
}

// InjectSession sets the li_at session cookie using Rod's SetCookies (called before page load)
func (h *HeadlessBrowser) InjectSession(liAt, jsessionID string) error {
	h.liAt = liAt
	h.jsessionID = strings.Trim(jsessionID, `"`)
	return nil
}

// OpenPage navigates to https://www.linkedin.com/robots.txt to establish the origin
// then sets the session cookies so fetch() requests include li_at and JSESSIONID.
func (h *HeadlessBrowser) OpenPage(ctx context.Context, _ string) error {
	page, err := h.browser.Page(proto.TargetCreateTarget{URL: "https://www.linkedin.com/robots.txt"})
	if err != nil {
		return err
	}
	h.page = page

	// Stealth overrides
	_, _ = h.page.EvalOnNewDocument(`
		Object.defineProperty(navigator, 'webdriver', {get: () => undefined});
		Object.defineProperty(navigator, 'plugins', {get: () => [1,2,3,4,5]});
		Object.defineProperty(navigator, 'languages', {get: () => ['en-US','en']});
		window.chrome = {runtime: {}, loadTimes: function(){}, csi: function(){}, app: {}};
		Object.defineProperty(navigator, 'platform', {get: () => 'MacIntel'});
	`)

	// Inject session cookies into Chromium's network cookie jar
	if h.liAt != "" {
		csrfClean := strings.Trim(h.jsessionID, `"`)
		cookies := []*proto.NetworkCookieParam{
			{
				Name:     "li_at",
				Value:    h.liAt,
				Domain:   ".linkedin.com",
				Path:     "/",
				Secure:   true,
				HTTPOnly: true,
			},
			{
				Name:     "JSESSIONID",
				Value:    fmt.Sprintf(`"%s"`, csrfClean),
				Domain:   ".linkedin.com",
				Path:     "/",
				Secure:   true,
				HTTPOnly: false,
			},
			{
				Name:     "JSESSIONID",
				Value:    fmt.Sprintf(`"%s"`, csrfClean),
				Domain:   ".www.linkedin.com",
				Path:     "/",
				Secure:   true,
				HTTPOnly: false,
			},
		}
		_ = page.SetCookies(cookies)
	}

	return nil
}

// FetchVoyagerProfile executes an async fetch() inside headless Chromium.
func (h *HeadlessBrowser) FetchVoyagerProfile(ctx context.Context, vanityName string) (*cdp.VoyagerProfile, error) {
	if h.page == nil {
		return nil, fmt.Errorf("no page open — call OpenPage first")
	}

	js := fmt.Sprintf(`async () => {
  const csrf = %q;
  const endpoints = [
    '/voyager/api/identity/dash/profiles?q=vanityName&vanityName=%s',
    '/voyager/api/identity/profiles/%s',
    '/voyager/api/identity/profiles/%s/profileView',
    '/voyager/api/me',
  ];
  const attempts = [];
  for (const ep of endpoints) {
    try {
      const resp = await fetch('https://www.linkedin.com' + ep, {
        signal: AbortSignal.timeout(6000),
        credentials: 'include',
        headers: {
          'Accept': 'application/vnd.linkedin.normalized+json+2.1',
          'X-Li-Lang': 'en_US',
          'X-Requested-With': 'XMLHttpRequest',
          'Csrf-Token': csrf,
        }
      });
      attempts.push({ep: ep, status: resp.status, ok: resp.ok});
      if (resp.ok) {
        const data = await resp.json();
        return JSON.stringify({ok: true, endpoint: ep, status: resp.status, data: data});
      }
    } catch(e) { attempts.push({ep: ep, error: e.message}); }
  }
  return JSON.stringify({ok: false, error: 'all endpoints failed', attempts: attempts, csrf: csrf});
}`, h.jsessionID, vanityName, vanityName, vanityName)

	raw, err := h.evalAsync(ctx, js)
	if err != nil {
		return nil, fmt.Errorf("headless JS eval failed: %w", err)
	}

	var wrapper struct {
		OK       bool                   `json:"ok"`
		Endpoint string                 `json:"endpoint"`
		Status   int                    `json:"status"`
		Data     map[string]interface{} `json:"data"`
		Error    string                 `json:"error"`
		Attempts []interface{}          `json:"attempts"`
		CSRF     string                 `json:"csrf"`
	}
	if err := json.Unmarshal([]byte(raw), &wrapper); err != nil {
		return nil, fmt.Errorf("failed parsing headless response: %w\nRaw: %.300s", err, raw)
	}
	if !wrapper.OK {
		attJSON, _ := json.Marshal(wrapper.Attempts)
		return nil, fmt.Errorf("headless fetch failed: %s (attempts: %s, csrf: %s)", wrapper.Error, string(attJSON), wrapper.CSRF)
	}

	profile := &cdp.VoyagerProfile{VanityName: vanityName}
	parseVoyagerData(wrapper.Data, profile)

	if len(profile.Skills) == 0 {
		skills, _ := h.FetchSkills(ctx, vanityName)
		profile.Skills = skills
	}

	return profile, nil
}

// FetchSkills fetches skills for a LinkedIn member in the headless browser
func (h *HeadlessBrowser) FetchSkills(ctx context.Context, vanityName string) ([]string, error) {
	if h.page == nil {
		return nil, fmt.Errorf("no page open")
	}

	js := fmt.Sprintf(`async () => {
  const csrf = %q;
  const resp = await fetch('https://www.linkedin.com/voyager/api/identity/profiles/%s/skills', {
    signal: AbortSignal.timeout(6000),
    credentials: 'include',
    headers: {
      'Accept': 'application/vnd.linkedin.normalized+json+2.1',
      'X-Li-Lang': 'en_US',
      'X-Requested-With': 'XMLHttpRequest',
      'Csrf-Token': csrf,
    }
  });
  if (!resp.ok) return '[]';
  const data = await resp.json();
  return JSON.stringify((data.elements || []).map(e => e.name).filter(Boolean));
}`, h.jsessionID, vanityName)

	raw, err := h.evalAsync(ctx, js)
	if err != nil {
		return nil, err
	}

	var skills []string
	_ = json.Unmarshal([]byte(raw), &skills)
	return skills, nil
}

// GetCurrentUser fetches the currently logged in user's identity via /voyager/api/me
func (h *HeadlessBrowser) GetCurrentUser(ctx context.Context) (string, string, error) {
	if h.page == nil {
		return "", "", fmt.Errorf("no page open")
	}

	js := fmt.Sprintf(`async () => {
  const csrf = %q;
  const resp = await fetch('https://www.linkedin.com/voyager/api/me', {
    signal: AbortSignal.timeout(6000),
    credentials: 'include',
    headers: {
      'Accept': 'application/vnd.linkedin.normalized+json+2.1',
      'X-Li-Lang': 'en_US',
      'X-Requested-With': 'XMLHttpRequest',
      'Csrf-Token': csrf,
    }
  });
  if (!resp.ok) return JSON.stringify({error: resp.status});
  const d = await resp.json();
  const me = d.included && d.included[0];
  return JSON.stringify({
    vanityName: d.plainId || (me && me.publicIdentifier) || '',
    name: me ? (me.firstName + ' ' + me.lastName) : ''
  });
}`, h.jsessionID)

	raw, err := h.evalAsync(ctx, js)
	if err != nil {
		return "", "", err
	}

	var me struct {
		VanityName string `json:"vanityName"`
		Name       string `json:"name"`
	}
	_ = json.Unmarshal([]byte(raw), &me)
	return me.VanityName, me.Name, nil
}

// evalAsync executes an async JS expression in the headless browser using Rod's
// Evaluate with AwaitPromise=true — the correct way to run fetch() in headless Chrome.
func (h *HeadlessBrowser) evalAsync(ctx context.Context, js string) (string, error) {
	res, err := h.page.Context(ctx).Evaluate(rod.Eval(js).ByPromise())
	if err != nil {
		return "", err
	}
	return res.Value.String(), nil
}

// Eval runs arbitrary JavaScript in the headless browser context
func (h *HeadlessBrowser) Eval(ctx context.Context, js string) (string, error) {
	if h.page == nil {
		return "", fmt.Errorf("no page open — call OpenPage first")
	}
	return h.evalAsync(ctx, js)
}

// Close shuts down the headless Chromium process
func (h *HeadlessBrowser) Close() {
	if h.page != nil {
		_ = h.page.Close()
	}
	if h.browser != nil {
		_ = h.browser.Close()
	}
}

// WaitMs sleeps for ms milliseconds (useful for page load timing)
func (h *HeadlessBrowser) WaitMs(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func buildJSArray(items []string) string {
	parts := make([]string, len(items))
	for i, s := range items {
		parts[i] = fmt.Sprintf("%q", s)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// parseVoyagerData extracts profile fields from raw Voyager JSON
func parseVoyagerData(raw map[string]interface{}, profile *cdp.VoyagerProfile) {
	if fn, ok := raw["firstName"].(string); ok {
		profile.FirstName = fn
	}
	if ln, ok := raw["lastName"].(string); ok {
		profile.LastName = ln
	}
	if h, ok := raw["headline"].(string); ok {
		profile.Headline = h
	}
	if s, ok := raw["summary"].(string); ok {
		profile.Summary = s
	}
	if vn, ok := raw["vanityName"].(string); ok {
		profile.VanityName = vn
	}
	if loc, ok := raw["locationName"].(string); ok {
		profile.Location = loc
	}

	included, _ := raw["included"].([]interface{})
	for _, item := range included {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		typeStr, _ := m["$type"].(string)

		switch {
		case strings.Contains(typeStr, "Profile") || strings.Contains(typeStr, "MiniProfile"):
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

		case strings.Contains(typeStr, "Skill"):
			if name, ok := m["name"].(string); ok && name != "" {
				profile.Skills = appendUnique(profile.Skills, name)
			}

		case strings.Contains(typeStr, "Position"):
			exp := profilecode.Experience{}
			if t, ok := m["title"].(string); ok {
				exp.Role = t
			}
			if c, ok := m["companyName"].(string); ok {
				exp.Company = c
			}
			if d, ok := m["description"].(string); ok {
				exp.Description = d
			}
			if dr, ok := m["dateRange"].(map[string]interface{}); ok {
				if start, ok := dr["start"].(map[string]interface{}); ok {
					y, _ := start["year"].(float64)
					mo, _ := start["month"].(float64)
					if y > 0 {
						exp.StartDate = fmt.Sprintf("%.0f-%02.0f", y, mo)
					}
				}
				if end, ok := dr["end"].(map[string]interface{}); ok {
					y, _ := end["year"].(float64)
					mo, _ := end["month"].(float64)
					if y > 0 {
						exp.EndDate = fmt.Sprintf("%.0f-%02.0f", y, mo)
					}
				} else {
					exp.EndDate = "Present"
					exp.Current = true
				}
			}
			if exp.Role != "" {
				profile.Experience = append(profile.Experience, exp)
			}

		case strings.Contains(typeStr, "Education"):
			edu := profilecode.Education{}
			if s, ok := m["schoolName"].(string); ok {
				edu.School = s
			}
			if d, ok := m["degreeName"].(string); ok {
				edu.Degree = d
			}
			if f, ok := m["fieldOfStudy"].(string); ok {
				edu.FieldOfStudy = f
			}
			if edu.School != "" {
				profile.Education = append(profile.Education, edu)
			}

		case strings.Contains(typeStr, "Language"):
			if name, ok := m["name"].(string); ok && name != "" {
				profile.Languages = appendUnique(profile.Languages, name)
			}
		}
	}
}

func appendUnique(slice []string, s string) []string {
	for _, e := range slice {
		if e == s {
			return slice
		}
	}
	return append(slice, s)
}
