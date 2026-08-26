// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

// Package ldin provides programmatic access to LinkedIn REST APIs, Profile-as-Code,
// and developer workspace automation.
package ldin

import (
	"context"

	"github.com/santusht/ldin/internal/config"
	"github.com/santusht/ldin/internal/linkedin"
	"github.com/santusht/ldin/internal/profilecode"
)

// Client is the public SDK interface
type Client struct {
	li *linkedin.Client
}

// NewClient initializes SDK with access token
func NewClient(accessToken string) *Client {
	creds := &config.ProfileCredentials{
		AccessToken: accessToken,
	}
	return &Client{
		li: linkedin.NewClient(creds, linkedin.DefaultAPIVersion),
	}
}

// GetProfile retrieves member profile
func (c *Client) GetProfile(ctx context.Context) (*linkedin.ProfileResponse, error) {
	return c.li.GetCurrentMemberProfile(ctx)
}

// CreatePost creates a post on LinkedIn
func (c *Client) CreatePost(ctx context.Context, commentary string) (*linkedin.PostResponse, error) {
	req := &linkedin.PostCreateRequest{
		Commentary:   commentary,
		Visibility:   linkedin.VisibilityPublic,
		Distribution: linkedin.FeedDistributionMainFeed,
	}
	return c.li.CreatePost(ctx, req)
}

// ExportProfileAsCode generates declarative ProfileAsCode struct
func (c *Client) ExportProfileAsCode(ctx context.Context) (*profilecode.ProfileAsCode, error) {
	return c.li.ExportAsCode(ctx)
}
