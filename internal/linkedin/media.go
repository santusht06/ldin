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
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MediaType represents upload category
type MediaType string

const (
	MediaTypeImage    MediaType = "image"
	MediaTypeVideo    MediaType = "video"
	MediaTypeDocument MediaType = "document"
)

// UploadRegistrationResponse represents LinkedIn media upload registration
type UploadRegistrationResponse struct {
	Value struct {
		UploadMechanism struct {
			MediaUploadHttpRequest struct {
				UploadURL string            `json:"uploadUrl"`
				Headers   map[string]string `json:"headers"`
			} `json:"com.linkedin.digitalmedia.uploading.MediaUploadHttpRequest"`
		} `json:"uploadMechanism"`
		MediaArtifact string `json:"mediaArtifact"`
		Asset         string `json:"asset"`
		Image         string `json:"image"`
		Video         string `json:"video"`
		Document      string `json:"document"`
	} `json:"value"`
}

// MediaUploadResult wraps successful media upload information
type MediaUploadResult struct {
	URN       string    `json:"urn"`
	Type      MediaType `json:"type"`
	Filename  string    `json:"filename"`
	SizeBytes int64     `json:"size_bytes"`
}

// UploadMedia executes the 3-step LinkedIn media upload protocol
func (c *Client) UploadMedia(ctx context.Context, filePath string, mType MediaType) (*MediaUploadResult, error) {
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed reading file %s: %w", filePath, err)
	}

	owner := c.GetMemberURN()
	var endpoint string
	var payload map[string]interface{}

	switch mType {
	case MediaTypeImage:
		endpoint = "/rest/images?action=initializeUpload"
		payload = map[string]interface{}{
			"initializeUploadRequest": map[string]interface{}{
				"owner": owner,
			},
		}
	case MediaTypeVideo:
		endpoint = "/rest/videos?action=initializeUpload"
		payload = map[string]interface{}{
			"initializeUploadRequest": map[string]interface{}{
				"owner":         owner,
				"fileSizeBytes": len(fileData),
				"uploadCaptions": false,
				"uploadThumbnail": false,
			},
		}
	case MediaTypeDocument:
		endpoint = "/rest/documents?action=initializeUpload"
		payload = map[string]interface{}{
			"initializeUploadRequest": map[string]interface{}{
				"owner": owner,
			},
		}
	default:
		// Detect by extension
		ext := strings.ToLower(filepath.Ext(filePath))
		if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".webp" {
			return c.UploadMedia(ctx, filePath, MediaTypeImage)
		} else if ext == ".mp4" || ext == ".mov" || ext == ".avi" {
			return c.UploadMedia(ctx, filePath, MediaTypeVideo)
		} else if ext == ".pdf" || ext == ".doc" || ext == ".docx" || ext == ".ppt" || ext == ".pptx" {
			return c.UploadMedia(ctx, filePath, MediaTypeDocument)
		}
		return nil, fmt.Errorf("unsupported media format for file %s", filePath)
	}

	// Step 1: Initialize Upload
	headers := map[string]string{
		"Linkedin-Version": c.APIVersion,
	}
	respBytes, err := c.Request(ctx, "POST", endpoint, nil, payload, headers)
	if err != nil {
		return nil, fmt.Errorf("media initialization failed: %w", err)
	}

	var reg UploadRegistrationResponse
	if err := json.Unmarshal(respBytes, &reg); err != nil {
		return nil, fmt.Errorf("invalid upload registration response: %w", err)
	}

	uploadURL := reg.Value.UploadMechanism.MediaUploadHttpRequest.UploadURL
	if uploadURL == "" {
		return nil, fmt.Errorf("no uploadUrl returned by LinkedIn for media")
	}

	// Determine Asset URN
	assetURN := reg.Value.Image
	if assetURN == "" {
		assetURN = reg.Value.Video
	}
	if assetURN == "" {
		assetURN = reg.Value.Document
	}
	if assetURN == "" {
		assetURN = reg.Value.Asset
	}

	// Step 2: Upload binary bytes to the authorized pre-signed URL
	putReq, err := http.NewRequestWithContext(ctx, "PUT", uploadURL, bytes.NewReader(fileData))
	if err != nil {
		return nil, fmt.Errorf("failed constructing binary upload request: %w", err)
	}

	for k, v := range reg.Value.UploadMechanism.MediaUploadHttpRequest.Headers {
		putReq.Header.Set(k, v)
	}
	if putReq.Header.Get("Content-Type") == "" {
		putReq.Header.Set("Content-Type", "application/octet-stream")
	}

	uploadClient := &http.Client{Timeout: 120 * time.Second}
	putResp, err := uploadClient.Do(putReq)
	if err != nil {
		return nil, fmt.Errorf("binary upload failed: %w", err)
	}
	defer putResp.Body.Close()

	if putResp.StatusCode < 200 || putResp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(putResp.Body)
		return nil, fmt.Errorf("binary upload failed (HTTP %d): %s", putResp.StatusCode, string(respBody))
	}

	return &MediaUploadResult{
		URN:       assetURN,
		Type:      mType,
		Filename:  filepath.Base(filePath),
		SizeBytes: int64(len(fileData)),
	}, nil
}
