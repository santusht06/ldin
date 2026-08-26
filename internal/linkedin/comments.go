// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package linkedin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// Comment represents a LinkedIn post comment
type Comment struct {
	ID         string `json:"id"`
	URN        string `json:"urn,omitempty"`
	Actor      string `json:"actor"`
	Message    struct {
		Text string `json:"text"`
	} `json:"message"`
	CreatedAt  int64  `json:"createdAt,omitempty"`
	LikesCount int    `json:"likesCount,omitempty"`
	Parent     string `json:"parentComment,omitempty"`
}

// CommentsListResponse represents list of comments
type CommentsListResponse struct {
	Elements []Comment `json:"elements"`
	Paging   struct {
		Count int `json:"count"`
		Start int `json:"start"`
		Total int `json:"total,omitempty"`
	} `json:"paging"`
}

// CreateComment adds a comment or reply to a post or comment
func (c *Client) CreateComment(ctx context.Context, targetURN, text, parentCommentURN string) (*Comment, error) {
	payload := map[string]interface{}{
		"actor": c.GetMemberURN(),
		"message": map[string]string{
			"text": text,
		},
		"object": targetURN,
	}

	if parentCommentURN != "" {
		payload["parentComment"] = parentCommentURN
	}

	headers := map[string]string{
		"Linkedin-Version": c.APIVersion,
	}

	respBytes, err := c.Request(ctx, "POST", "/rest/socialActions/"+url.PathEscape(targetURN)+"/comments", nil, payload, headers)
	if err != nil {
		return nil, err
	}

	var comment Comment
	_ = json.Unmarshal(respBytes, &comment)
	if comment.Message.Text == "" {
		comment.Message.Text = text
		comment.Actor = c.GetMemberURN()
	}
	return &comment, nil
}

// ListComments retrieves comments on a post
func (c *Client) ListComments(ctx context.Context, postURN string, count, start int) (*CommentsListResponse, error) {
	if count <= 0 {
		count = 20
	}
	endpoint := fmt.Sprintf("/rest/socialActions/%s/comments", url.PathEscape(postURN))

	q := url.Values{}
	q.Set("count", fmt.Sprintf("%d", count))
	q.Set("start", fmt.Sprintf("%d", start))

	respBytes, err := c.Request(ctx, "GET", endpoint, q, nil, nil)
	if err != nil {
		return nil, err
	}

	var res CommentsListResponse
	if err := json.Unmarshal(respBytes, &res); err != nil {
		return nil, fmt.Errorf("failed parsing comments list: %w", err)
	}
	return &res, nil
}

// DeleteComment removes a comment by ID/URN
func (c *Client) DeleteComment(ctx context.Context, targetURN, commentURN string) error {
	endpoint := fmt.Sprintf("/rest/socialActions/%s/comments/%s", url.PathEscape(targetURN), url.PathEscape(commentURN))
	_, err := c.Request(ctx, "DELETE", endpoint, nil, nil, nil)
	return err
}
