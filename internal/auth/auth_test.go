// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package auth

import (
	"testing"
)

func TestGeneratePKCE(t *testing.T) {
	pkce, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("failed generating PKCE: %v", err)
	}

	if pkce.Verifier == "" {
		t.Errorf("verifier is empty")
	}
	if pkce.Challenge == "" {
		t.Errorf("challenge is empty")
	}
	if pkce.State == "" {
		t.Errorf("state is empty")
	}
	if pkce.Verifier == pkce.Challenge {
		t.Errorf("challenge should be SHA256 hashed from verifier")
	}
}

func TestDefaultScopes(t *testing.T) {
	if len(DefaultScopes) < 4 {
		t.Errorf("expected at least 4 default scopes, got %d", len(DefaultScopes))
	}
	scopeMap := make(map[string]bool)
	for _, s := range DefaultScopes {
		scopeMap[s] = true
	}

	for _, req := range []string{"openid", "profile", "email", "w_member_social"} {
		if !scopeMap[req] {
			t.Errorf("expected scope '%s' in default scopes", req)
		}
	}
}
