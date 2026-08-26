// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package linkedin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// SocialActionsSummary details aggregated interactions on a post
type SocialActionsSummary struct {
	URN            string `json:"urn"`
	Target         string `json:"target"`
	LikesSummary   struct {
		TotalLikes  int  `json:"totalLikes"`
		LikedByMe   bool `json:"likedByCurrentUser"`
		SelectedType string `json:"selectedReactionType,omitempty"`
	} `json:"likesSummary"`
	CommentsSummary struct {
		TotalComments int `json:"totalComments"`
	} `json:"commentsSummary"`
	TotalShares    int `json:"totalShares,omitempty"`
}

// GetSocialSummary retrieves aggregated likes, comments, and current user interaction for a post URN
func (c *Client) GetSocialSummary(ctx context.Context, targetURN string) (*SocialActionsSummary, error) {
	endpoint := fmt.Sprintf("/rest/socialActions/%s", url.PathEscape(targetURN))

	respBytes, err := c.Request(ctx, "GET", endpoint, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	var summary SocialActionsSummary
	if err := json.Unmarshal(respBytes, &summary); err != nil {
		return nil, fmt.Errorf("failed parsing social summary: %w", err)
	}
	summary.Target = targetURN

	return &summary, nil
}
