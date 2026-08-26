// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package gitcontext

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInspectLocalRepo(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ldin-git-test-*")
	if err != nil {
		t.Fatalf("failed creating temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	readmeContent := "# Test Project\nBuilding distributed microservices in Go."
	_ = os.WriteFile(filepath.Join(tempDir, "README.md"), []byte(readmeContent), 0644)

	ctx, err := InspectLocalRepo(tempDir, 5)
	if err != nil {
		t.Fatalf("InspectLocalRepo failed: %v", err)
	}

	if ctx.RepoName == "" {
		t.Errorf("expected non-empty repo name")
	}
	if ctx.READMEExcerpt == "" {
		t.Errorf("expected README excerpt, got empty")
	}
	if len(ctx.RecentCommits) == 0 {
		t.Errorf("expected fallback or real commits, got empty")
	}
}
