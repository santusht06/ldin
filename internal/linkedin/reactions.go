// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package linkedin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// ReactionType represents standard LinkedIn reaction types
type ReactionType string

const (
	ReactionLike        ReactionType = "LIKE"
	ReactionCelebrate   ReactionType = "PRAISE"       // LinkedIn API maps celebrate to PRAISE
	ReactionSupport     ReactionType = "APPRECIATION" // Support is APPRECIATION
	ReactionLove        ReactionType = "EMPATHY"      // Love is EMPATHY
	ReactionInsightful  ReactionType = "INTEREST"     // Insightful is INTEREST
	ReactionCurious     ReactionType = "MAYBE"        // Curious / Entertainment
)

// ReactionResponse represents a reaction result
type ReactionResponse struct {
	ReactionType string `json:"reactionType"`
	Actor        string `json:"actor"`
	Target       string `json:"target"`
}

// React sends or updates a reaction on a post or comment
func (c *Client) React(ctx context.Context, targetURN string, rType ReactionType) (*ReactionResponse, error) {
	if rType == "" {
		rType = ReactionLike
	}

	payload := map[string]interface{}{
		"actor":        c.GetMemberURN(),
		"reactionType": string(rType),
	}

	endpoint := fmt.Sprintf("/rest/reactions/%s", url.PathEscape(targetURN))
	headers := map[string]string{
		"Linkedin-Version": c.APIVersion,
	}

	respBytes, err := c.Request(ctx, "POST", endpoint, nil, payload, headers)
	if err != nil {
		return nil, err
	}

	var res ReactionResponse
	_ = json.Unmarshal(respBytes, &res)
	res.ReactionType = string(rType)
	res.Actor = c.GetMemberURN()
	res.Target = targetURN

	return &res, nil
}

// DeleteReaction removes the user's reaction from a post or comment
func (c *Client) DeleteReaction(ctx context.Context, targetURN string) error {
	actor := url.QueryEscape(c.GetMemberURN())
	endpoint := fmt.Sprintf("/rest/reactions/%s?actor=%s", url.PathEscape(targetURN), actor)
	_, err := c.Request(ctx, "DELETE", endpoint, nil, nil, nil)
	return err
}

// NormalizeReactionType parses user string input to standard LinkedIn enum
func NormalizeReactionType(input string) ReactionType {
	switch strings.ToUpper(strings.TrimSpace(input)) {
	case "LIKE":
		return ReactionLike
	case "CELEBRATE", "PRAISE":
		return ReactionCelebrate
	case "SUPPORT", "APPRECIATION":
		return ReactionSupport
	case "LOVE", "EMPATHY":
		return ReactionLove
	case "INSIGHTFUL", "INTEREST":
		return ReactionInsightful
	case "CURIOUS", "ENTERTAINMENT", "MAYBE":
		return ReactionCurious
	default:
		return ReactionLike
	}
}
