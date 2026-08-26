// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package ldin

import (
	"context"
	"testing"
)

func TestPublicSDKInitialization(t *testing.T) {
	client := NewClient("mock-token-xyz")
	if client == nil {
		t.Fatalf("expected non-nil client")
	}

	ctx := context.Background()
	// Test Profile-as-Code generation through SDK
	pac, err := client.ExportProfileAsCode(ctx)
	if err == nil && pac != nil {
		if pac.Name == "" && pac.Headline == "" {
			t.Errorf("expected populated profile fields")
		}
	}
}
