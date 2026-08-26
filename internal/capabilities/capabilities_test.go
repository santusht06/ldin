// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package capabilities

import (
	"testing"
)

func TestEvaluateCapabilities(t *testing.T) {
	// Member with basic self-service scopes
	scopes := []string{"openid", "profile", "email", "w_member_social"}

	eval := EvaluateCapabilities(scopes)
	if len(eval) == 0 {
		t.Fatalf("expected evaluated capabilities list, got empty")
	}

	var foundPostCreate bool
	var foundAds bool
	for _, st := range eval {
		if st.Capability.ID == "posts.create" {
			foundPostCreate = true
			if !st.Available {
				t.Fatalf("expected posts.create to be available with w_member_social")
			}
		}
		if st.Capability.ID == "ads.campaigns" {
			foundAds = true
			if st.Available {
				t.Fatalf("expected ads.campaigns to be restricted without rw_ads")
			}
		}
	}

	if !foundPostCreate {
		t.Fatalf("posts.create capability not found in registry")
	}
	if !foundAds {
		t.Fatalf("ads.campaigns capability not found in registry")
	}
}
