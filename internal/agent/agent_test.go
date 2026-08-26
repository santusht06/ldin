// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package agent

import (
	"context"
	"testing"

	"github.com/santusht/ldin/internal/config"
)

func TestPermissionGuard(t *testing.T) {
	cfg := &config.AgentConfig{
		AutoPublish:   false,
		AllowedScopes: []string{"read", "draft", "ai"},
	}

	guard := NewPermissionGuard(cfg)

	// Read & draft must succeed
	if err := guard.CheckPermission(PermRead); err != nil {
		t.Fatalf("expected PermRead to succeed, got %v", err)
	}
	if err := guard.CheckPermission(PermDraft); err != nil {
		t.Fatalf("expected PermDraft to succeed, got %v", err)
	}

	// Publish should fail by default
	if err := guard.CheckPermission(PermPublish); err == nil {
		t.Fatalf("expected PermPublish to fail when AutoPublish is false")
	}

	// Enable publish
	guard.SetPermission(PermPublish, true)
	if err := guard.CheckPermission(PermPublish); err != nil {
		t.Fatalf("expected PermPublish to succeed after enabling, got %v", err)
	}
}

func TestEngineHeuristicExecution(t *testing.T) {
	cm, err := config.NewConfigManager()
	if err != nil {
		t.Fatalf("failed initializing config manager: %v", err)
	}

	eng, err := NewEngine(cm, nil)
	if err != nil {
		t.Fatalf("failed initializing engine: %v", err)
	}

	ctx := context.Background()
	result, err := eng.Run(ctx, "Create a post about our distributed cache")
	if err != nil {
		t.Fatalf("engine run failed: %v", err)
	}

	if result.Response == "" {
		t.Fatalf("expected non-empty response from engine")
	}
	if result.DraftID == "" {
		t.Fatalf("expected draft ID to be generated for post instruction")
	}
}
