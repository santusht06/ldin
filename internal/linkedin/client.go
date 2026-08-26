// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package linkedin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/santusht/ldin/internal/capabilities"
	"github.com/santusht/ldin/internal/config"
)

const (
	DefaultBaseURL    = "https://api.linkedin.com"
	DefaultAPIVersion = "202608"
)

// APIError details errors returned by LinkedIn REST API
type APIError struct {
	StatusCode int    `json:"status_code"`
	Code       string `json:"code,omitempty"`
	Message    string `json:"message"`
	ErrorDetail string `json:"error_detail,omitempty"`
	Hint       string `json:"hint,omitempty"`
}

func (e *APIError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("LinkedIn API Error (HTTP %d): %s\n  Hint: %s", e.StatusCode, e.Message, e.Hint)
	}
	return fmt.Sprintf("LinkedIn API Error (HTTP %d): %s", e.StatusCode, e.Message)
}

// Client is the primary HTTP client communicating with LinkedIn REST API
type Client struct {
	BaseURL     string
	AccessToken string
	APIVersion  string
	HTTPClient  *http.Client
	Profile     *config.ProfileCredentials
}

// NewClient initializes a LinkedIn REST API client for the active profile
func NewClient(creds *config.ProfileCredentials, apiVersion string) *Client {
	if apiVersion == "" {
		apiVersion = DefaultAPIVersion
	}
	tok := ""
	if creds != nil {
		tok = creds.AccessToken
	}

	return &Client{
		BaseURL:     DefaultBaseURL,
		AccessToken: tok,
		APIVersion:  apiVersion,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		Profile: creds,
	}
}

// Request executes an HTTP request against the LinkedIn REST API
func (c *Client) Request(ctx context.Context, method, endpoint string, query url.Values, body interface{}, headers map[string]string) ([]byte, error) {
	fullURL := endpoint
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		if !strings.HasPrefix(endpoint, "/") {
			endpoint = "/" + endpoint
		}
		fullURL = c.BaseURL + endpoint
	}

	if len(query) > 0 {
		if strings.Contains(fullURL, "?") {
			fullURL += "&" + query.Encode()
		} else {
			fullURL += "?" + query.Encode()
		}
	}

	var reqBody io.Reader
	if body != nil {
		switch v := body.(type) {
		case []byte:
			reqBody = bytes.NewReader(v)
		case string:
			reqBody = strings.NewReader(v)
		case io.Reader:
			reqBody = v
		default:
			jsonData, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to serialize request body: %w", err)
			}
			reqBody = bytes.NewReader(jsonData)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to construct request: %w", err)
	}

	// Set standard LinkedIn REST API headers
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Linkedin-Version", c.APIVersion)
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Custom header overrides
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, c.handleErrorResponse(resp.StatusCode, respBody, endpoint)
	}

	return respBody, nil
}

func (c *Client) handleErrorResponse(statusCode int, body []byte, endpoint string) error {
	var parsed struct {
		Message string `json:"message"`
		ServiceErrorCode int `json:"serviceErrorCode"`
		Status int `json:"status"`
	}
	_ = json.Unmarshal(body, &parsed)

	msg := parsed.Message
	if msg == "" {
		msg = string(body)
	}
	if strings.TrimSpace(msg) == "" {
		msg = http.StatusText(statusCode)
	}

	apiErr := &APIError{
		StatusCode: statusCode,
		Message:    msg,
	}

	// Generate actionable hints based on common status codes & capabilities
	switch statusCode {
	case 401:
		apiErr.Hint = "Your access token has expired or is invalid. Run 'ldin auth login' or 'ldin auth refresh'."
	case 403:
		apiErr.Hint = fmt.Sprintf("Missing required LinkedIn API permission. Check 'ldin capabilities' or 'ldin auth scopes' to verify access for endpoint %s.", endpoint)
	case 404:
		apiErr.Hint = "The requested resource, post, or URN was not found or has been deleted."
	case 429:
		apiErr.Hint = "LinkedIn API rate limit reached. Please wait before retrying."
	}

	return apiErr
}

// GetMemberURN returns the current authenticated member's URN
func (c *Client) GetMemberURN() string {
	if c.Profile != nil && c.Profile.MemberURN != "" {
		return c.Profile.MemberURN
	}
	return "urn:li:person:me"
}

// GetGrantedScopes returns scopes on the active profile
func (c *Client) GetGrantedScopes() []string {
	if c.Profile != nil {
		return c.Profile.Scopes
	}
	return nil
}

// CheckFeatureAvailability helper to test if a feature can be called
func (c *Client) CheckFeatureAvailability(featureID string) (bool, string) {
	scopes := c.GetGrantedScopes()
	avail, cap, missing := capabilities.CheckCapability(featureID, scopes)
	if avail {
		return true, ""
	}
	if cap == nil {
		return false, "Unknown feature ID"
	}
	return false, fmt.Sprintf("Feature '%s' requires scopes: %s (Tier: %s)", cap.Name, strings.Join(missing, ", "), cap.Tier)
}
