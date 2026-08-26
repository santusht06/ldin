// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package linkedin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/santusht/ldin/internal/config"
)

func TestReactionNormalization(t *testing.T) {
	cases := []struct {
		input    string
		expected ReactionType
	}{
		{"LIKE", ReactionLike},
		{"like", ReactionLike},
		{"CELEBRATE", ReactionCelebrate},
		{"praise", ReactionCelebrate},
		{"SUPPORT", ReactionSupport},
		{"appreciation", ReactionSupport},
		{"LOVE", ReactionLove},
		{"empathy", ReactionLove},
		{"INSIGHTFUL", ReactionInsightful},
		{"interest", ReactionInsightful},
		{"CURIOUS", ReactionCurious},
		{"maybe", ReactionCurious},
		{"UNKNOWN", ReactionLike},
	}

	for _, tc := range cases {
		got := NormalizeReactionType(tc.input)
		if got != tc.expected {
			t.Errorf("NormalizeReactionType(%s) = %s, expected %s", tc.input, got, tc.expected)
		}
	}
}

func TestDraftLifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ldin-draft-test-*")
	if err != nil {
		t.Fatalf("failed creating temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	os.Setenv("LDIN_HOME", tempDir)
	defer os.Unsetenv("LDIN_HOME")

	cm, err := config.NewConfigManager()
	if err != nil {
		t.Fatalf("failed initializing config manager: %v", err)
	}

	draft := &PostDraft{
		Title:       "Test Launch Post",
		Commentary:  "Excited to launch ldin CLI for developers!",
		ContentType: ContentTypeText,
		Visibility:  VisibilityPublic,
		CreatedAt:   time.Now(),
	}

	if err := SaveDraft(cm, draft); err != nil {
		t.Fatalf("failed saving draft: %v", err)
	}

	loaded, err := LoadDraft(cm, draft.ID)
	if err != nil {
		t.Fatalf("failed loading draft: %v", err)
	}
	if loaded.Commentary != draft.Commentary {
		t.Errorf("commentary mismatch: got '%s', expected '%s'", loaded.Commentary, draft.Commentary)
	}

	drafts, err := ListDrafts(cm)
	if err != nil {
		t.Fatalf("failed listing drafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Errorf("expected 1 draft, got %d", len(drafts))
	}

	if err := DeleteDraft(cm, draft.ID); err != nil {
		t.Fatalf("failed deleting draft: %v", err)
	}

	if _, err := LoadDraft(cm, draft.ID); err == nil {
		t.Errorf("expected error loading deleted draft, got nil")
	}
}

func TestClientRESTRequest(t *testing.T) {
	// Mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-access-token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("Linkedin-Version") != "202608" {
			http.Error(w, "Invalid version", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sub":"urn:li:person:mock123","name":"Test Developer"}`))
	}))
	defer server.Close()

	creds := &config.ProfileCredentials{
		AccessToken: "test-access-token",
		MemberURN:   "urn:li:person:mock123",
	}

	client := NewClient(creds, "202608")
	client.BaseURL = server.URL

	ctx := context.Background()
	body, err := client.Request(ctx, "GET", "/v2/userinfo", nil, nil, nil)
	if err != nil {
		t.Fatalf("client request failed: %v", err)
	}

	if len(body) == 0 {
		t.Errorf("expected non-empty response body")
	}
}
