// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// SessionCredentials holds extracted session cookies from username/password login
type SessionCredentials struct {
	LiAt       string `json:"li_at"`
	JSESSIONID string `json:"jsessionid"`
	UserURN    string `json:"user_urn"`
	UserEmail  string `json:"user_email"`
}

// LoginWithCredentials authenticates against LinkedIn using email & password
func LoginWithCredentials(ctx context.Context, email, password string) (*SessionCredentials, error) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 20 * time.Second,
	}

	// Step 1: GET LinkedIn login page to obtain initial CSRF token
	req1, err := http.NewRequestWithContext(ctx, "GET", "https://www.linkedin.com/login", nil)
	if err != nil {
		return nil, err
	}
	req1.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
	req1.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp1, err := client.Do(req1)
	if err != nil {
		return nil, fmt.Errorf("failed reaching LinkedIn login gateway: %w", err)
	}
	defer resp1.Body.Close()

	body1, _ := io.ReadAll(resp1.Body)
	bodyStr := string(body1)

	// Extract loginCsrfParam
	csrfParam := ""
	reCSRF := regexp.MustCompile(`name="loginCsrfParam"\s+value="([^"]+)"`)
	if match := reCSRF.FindStringSubmatch(bodyStr); len(match) > 1 {
		csrfParam = match[1]
	}

	// Step 2: POST email and password to authentication endpoint
	formData := url.Values{}
	formData.Set("session_key", email)
	formData.Set("session_password", password)
	if csrfParam != "" {
		formData.Set("loginCsrfParam", csrfParam)
	}

	req2, err := http.NewRequestWithContext(ctx, "POST", "https://www.linkedin.com/checkpoint/lg/login-submit", strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req2.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
	req2.Header.Set("Referer", "https://www.linkedin.com/login")

	resp2, err := client.Do(req2)
	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp2.Body.Close()

	// Extract li_at and JSESSIONID cookies from cookie jar
	targetURL, _ := url.Parse("https://www.linkedin.com")
	cookies := client.Jar.Cookies(targetURL)

	var liAt, jsessionId string
	for _, c := range cookies {
		if c.Name == "li_at" {
			liAt = c.Value
		}
		if c.Name == "JSESSIONID" {
			jsessionId = strings.Trim(c.Value, `"`)
		}
	}

	if liAt == "" {
		body2, _ := io.ReadAll(resp2.Body)
		if strings.Contains(string(body2), "challenge") || strings.Contains(string(body2), "checkpoint") {
			return nil, fmt.Errorf("account requires 2FA or CAPTCHA verification. Pass your session token directly: `ldin auth token <li_at_token>`")
		}
		return nil, fmt.Errorf("authentication failed: invalid email or password")
	}

	return &SessionCredentials{
		LiAt:       liAt,
		JSESSIONID: jsessionId,
		UserEmail:  email,
		UserURN:    "urn:li:person:authenticated",
	}, nil
}
