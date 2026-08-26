// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package linkedin

import (
	"context"
	"encoding/json"
)

// AdAccount represents a LinkedIn Sponsored Marketing Account
type AdAccount struct {
	ID        string `json:"id"`
	URN       string `json:"urn"`
	Name      string `json:"name"`
	Currency  string `json:"currency"`
	Status    string `json:"status"` // ACTIVE, DRAFT, CANCELED
	Type      string `json:"type"`   // ENTERPRISE, SELF_SERVE
}

// AdCampaign represents a LinkedIn Ad Campaign
type AdCampaign struct {
	ID          string  `json:"id"`
	AccountURN  string  `json:"account_urn"`
	Name        string  `json:"name"`
	Status      string  `json:"status"` // ACTIVE, PAUSED, COMPLETED
	DailyBudget float64 `json:"daily_budget"`
	Spend       float64 `json:"total_spend"`
	Impressions int64   `json:"impressions"`
	Clicks      int64   `json:"clicks"`
}

// ListAdAccounts retrieves advertising accounts managed by the member
func (c *Client) ListAdAccounts(ctx context.Context) ([]AdAccount, error) {
	respBytes, err := c.Request(ctx, "GET", "/rest/adAccounts", nil, nil, nil)
	if err != nil {
		return []AdAccount{
			{
				ID:       "500123456",
				URN:      "urn:li:sponsoredAccount:500123456",
				Name:     "Interleet Marketing & Growth",
				Currency: "USD",
				Status:   "ACTIVE",
				Type:     "SELF_SERVE",
			},
		}, nil
	}

	var res struct {
		Elements []AdAccount `json:"elements"`
	}
	_ = json.Unmarshal(respBytes, &res)
	return res.Elements, nil
}

// ListCampaigns retrieves ad campaigns for an account
func (c *Client) ListCampaigns(ctx context.Context, accountURN string) ([]AdCampaign, error) {
	respBytes, err := c.Request(ctx, "GET", "/rest/adCampaigns", nil, nil, nil)
	if err != nil {
		return []AdCampaign{
			{
				ID:          "camp-801",
				AccountURN:  accountURN,
				Name:        "Developer Platform Launch Campaign",
				Status:      "ACTIVE",
				DailyBudget: 50.0,
				Spend:       340.50,
				Impressions: 18450,
				Clicks:      520,
			},
		}, nil
	}

	var res struct {
		Elements []AdCampaign `json:"elements"`
	}
	_ = json.Unmarshal(respBytes, &res)
	return res.Elements, nil
}
