// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package linkedin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/santusht/ldin/internal/config"
)

// PostVisibility defines audience
type PostVisibility string

const (
	VisibilityPublic      PostVisibility = "PUBLIC"
	VisibilityConnections PostVisibility = "CONNECTIONS"
	VisibilityLoggedUsers PostVisibility = "LOGGED_IN"
)

// FeedDistribution defines feed targeting
type FeedDistribution string

const (
	FeedDistributionMainFeed        FeedDistribution = "MAIN_FEED"
	FeedDistributionNone            FeedDistribution = "NONE"
	FeedDistributionTargeted        FeedDistribution = "TARGETED"
)

// PostContentType defines the media / attachment type
type PostContentType string

const (
	ContentTypeText       PostContentType = "text"
	ContentTypeImage      PostContentType = "image"
	ContentTypeMultiImage PostContentType = "multi_image"
	ContentTypeVideo      PostContentType = "video"
	ContentTypeDocument   PostContentType = "document"
	ContentTypeArticle    PostContentType = "article"
	ContentTypePoll       PostContentType = "poll"
)

// PostCreateRequest defines parameters for publishing a LinkedIn post
type PostCreateRequest struct {
	Author            string           `json:"author"`
	Commentary        string           `json:"commentary"`
	Visibility        PostVisibility   `json:"visibility"`
	Distribution      FeedDistribution `json:"distribution"`
	LifecycleState    string           `json:"lifecycleState"`
	IsReshareDisabled bool             `json:"isReshareDisabledByAuthor"`
	
	// Media attachments
	MediaURNs         []string         `json:"media_urns,omitempty"`
	DocumentURN       string           `json:"document_urn,omitempty"`
	DocumentTitle     string           `json:"document_title,omitempty"`
	ArticleURL        string           `json:"article_url,omitempty"`
	ArticleTitle      string           `json:"article_title,omitempty"`
	ArticleDesc       string           `json:"article_description,omitempty"`
	PollQuestion      string           `json:"poll_question,omitempty"`
	PollOptions       []string         `json:"poll_options,omitempty"`
	PollDurationDays  int              `json:"poll_duration_days,omitempty"`
}

// PostResponse represents a created or fetched LinkedIn Post
type PostResponse struct {
	ID                string `json:"id"`
	URN               string `json:"urn,omitempty"`
	Author            string `json:"author"`
	Commentary        string `json:"commentary"`
	Visibility        string `json:"visibility"`
	LifecycleState    string `json:"lifecycleState"`
	CreatedAt         int64  `json:"createdAt,omitempty"`
	LastModifiedAt    int64  `json:"lastModifiedAt,omitempty"`
	Content           *PostContentResponse `json:"content,omitempty"`
}

// PostContentResponse details attachments on a post
type PostContentResponse struct {
	Media    *MediaContent    `json:"media,omitempty"`
	MultiImage *MultiImageContent `json:"multiImage,omitempty"`
	Article  *ArticleContent  `json:"article,omitempty"`
	Poll     *PollContent     `json:"poll,omitempty"`
}

type MediaContent struct {
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
}

type MultiImageContent struct {
	Images []struct {
		ID  string `json:"id"`
		Alt string `json:"altText,omitempty"`
	} `json:"images"`
}

type ArticleContent struct {
	Source      string `json:"source"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

type PollContent struct {
	Question string `json:"question"`
	Options  []struct {
		Text string `json:"text"`
	} `json:"options"`
}

// PostDraft represents a locally saved draft post
type PostDraft struct {
	ID           string           `json:"id" yaml:"id"`
	Title        string           `json:"title,omitempty" yaml:"title,omitempty"`
	Commentary   string           `json:"commentary" yaml:"commentary"`
	ContentType  PostContentType  `json:"content_type" yaml:"content_type"`
	MediaPaths   []string         `json:"media_paths,omitempty" yaml:"media_paths,omitempty"`
	ArticleURL   string           `json:"article_url,omitempty" yaml:"article_url,omitempty"`
	Visibility   PostVisibility   `json:"visibility" yaml:"visibility"`
	CreatedAt    time.Time        `json:"created_at" yaml:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at" yaml:"updated_at"`
	Tags         []string         `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// PostsListResponse represents response from LinkedIn posts collection
type PostsListResponse struct {
	Elements []PostResponse `json:"elements"`
	Paging   struct {
		Count int `json:"count"`
		Start int `json:"start"`
		Total int `json:"total,omitempty"`
	} `json:"paging"`
}

// CreatePost publishes a new post via LinkedIn /rest/posts API
func (c *Client) CreatePost(ctx context.Context, req *PostCreateRequest) (*PostResponse, error) {
	if req.Author == "" {
		req.Author = c.GetMemberURN()
	}
	if req.Visibility == "" {
		req.Visibility = VisibilityPublic
	}
	if req.Distribution == "" {
		req.Distribution = FeedDistributionMainFeed
	}
	if req.LifecycleState == "" {
		req.LifecycleState = "PUBLISHED"
	}

	payload := map[string]interface{}{
		"author":                        req.Author,
		"commentary":                    req.Commentary,
		"visibility":                    req.Visibility,
		"distribution": map[string]interface{}{
			"feedDistribution": req.Distribution,
			"targetEntities":   []interface{}{},
			"thirdPartyDistributionChannels": []interface{}{},
		},
		"lifecycleState":                req.LifecycleState,
		"isReshareDisabledByAuthor":     req.IsReshareDisabled,
	}

	// Attach media if provided
	if len(req.MediaURNs) == 1 {
		payload["content"] = map[string]interface{}{
			"media": map[string]interface{}{
				"id": req.MediaURNs[0],
			},
		}
	} else if len(req.MediaURNs) > 1 {
		var images []map[string]interface{}
		for _, urn := range req.MediaURNs {
			images = append(images, map[string]interface{}{
				"id": urn,
			})
		}
		payload["content"] = map[string]interface{}{
			"multiImage": map[string]interface{}{
				"images": images,
			},
		}
	} else if req.ArticleURL != "" {
		payload["content"] = map[string]interface{}{
			"article": map[string]interface{}{
				"source":      req.ArticleURL,
				"title":       req.ArticleTitle,
				"description": req.ArticleDesc,
			},
		}
	} else if req.PollQuestion != "" && len(req.PollOptions) >= 2 {
		var options []map[string]interface{}
		for _, opt := range req.PollOptions {
			options = append(options, map[string]interface{}{
				"text": opt,
			})
		}
		dur := req.PollDurationDays
		if dur <= 0 {
			dur = 7
		}
		payload["content"] = map[string]interface{}{
			"poll": map[string]interface{}{
				"question": req.PollQuestion,
				"options":  options,
				"settings": map[string]interface{}{
					"duration": fmt.Sprintf("DAYS_%d", dur),
				},
			},
		}
	}

	headers := map[string]string{
		"Linkedin-Version": c.APIVersion,
	}

	respBytes, err := c.Request(ctx, "POST", "/rest/posts", nil, payload, headers)
	if err != nil {
		return nil, err
	}

	var res PostResponse
	_ = json.Unmarshal(respBytes, &res)
	if res.Commentary == "" {
		res.Commentary = req.Commentary
		res.Author = req.Author
		res.Visibility = string(req.Visibility)
	}

	return &res, nil
}

// GetPost retrieves a post by its ID or URN
func (c *Client) GetPost(ctx context.Context, postID string) (*PostResponse, error) {
	encodedID := url.PathEscape(postID)
	endpoint := fmt.Sprintf("/rest/posts/%s", encodedID)

	respBytes, err := c.Request(ctx, "GET", endpoint, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	var post PostResponse
	if err := json.Unmarshal(respBytes, &post); err != nil {
		return nil, fmt.Errorf("failed to parse post response: %w", err)
	}
	return &post, nil
}

// ListPosts fetches posts by author
func (c *Client) ListPosts(ctx context.Context, authorURN string, count, start int) (*PostsListResponse, error) {
	if authorURN == "" {
		authorURN = c.GetMemberURN()
	}
	if count <= 0 {
		count = 10
	}

	q := url.Values{}
	q.Set("author", authorURN)
	q.Set("q", "author")
	q.Set("count", fmt.Sprintf("%d", count))
	q.Set("start", fmt.Sprintf("%d", start))

	respBytes, err := c.Request(ctx, "GET", "/rest/posts", q, nil, nil)
	if err != nil {
		return nil, err
	}

	var list PostsListResponse
	if err := json.Unmarshal(respBytes, &list); err != nil {
		return nil, fmt.Errorf("failed to parse posts list: %w", err)
	}
	return &list, nil
}

// DeletePost deletes a post by ID
func (c *Client) DeletePost(ctx context.Context, postID string) error {
	encodedID := url.PathEscape(postID)
	endpoint := fmt.Sprintf("/rest/posts/%s", encodedID)

	_, err := c.Request(ctx, "DELETE", endpoint, nil, nil, nil)
	return err
}

// SaveDraft saves a draft to ~/.ldin/drafts/<id>.json
func SaveDraft(cm *config.ConfigManager, draft *PostDraft) error {
	if draft.ID == "" {
		draft.ID = fmt.Sprintf("draft-%d", time.Now().Unix())
	}
	if draft.CreatedAt.IsZero() {
		draft.CreatedAt = time.Now()
	}
	draft.UpdatedAt = time.Now()

	filePath := filepath.Join(cm.DraftsDir(), draft.ID+".json")
	data, err := json.MarshalIndent(draft, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0600)
}

// LoadDraft retrieves a draft by ID
func LoadDraft(cm *config.ConfigManager, draftID string) (*PostDraft, error) {
	filePath := filepath.Join(cm.DraftsDir(), draftID+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("draft '%s' not found: %w", draftID, err)
	}
	var draft PostDraft
	if err := json.Unmarshal(data, &draft); err != nil {
		return nil, err
	}
	return &draft, nil
}

// ListDrafts returns all local post drafts
func ListDrafts(cm *config.ConfigManager) ([]*PostDraft, error) {
	entries, err := os.ReadDir(cm.DraftsDir())
	if err != nil {
		return nil, err
	}

	var drafts []*PostDraft
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			id := strings.TrimSuffix(e.Name(), ".json")
			d, err := LoadDraft(cm, id)
			if err == nil {
				drafts = append(drafts, d)
			}
		}
	}
	return drafts, nil
}

// DeleteDraft removes a draft by ID
func DeleteDraft(cm *config.ConfigManager, draftID string) error {
	filePath := filepath.Join(cm.DraftsDir(), draftID+".json")
	return os.Remove(filePath)
}
