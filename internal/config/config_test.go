// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigManagerLifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ldin-test-*")
	if err != nil {
		t.Fatalf("failed creating temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	os.Setenv("LDIN_HOME", tempDir)
	defer os.Unsetenv("LDIN_HOME")

	cm, err := NewConfigManager()
	if err != nil {
		t.Fatalf("failed initializing config manager: %v", err)
	}

	// Test default config loading
	cfg, err := cm.LoadConfig()
	if err != nil {
		t.Fatalf("failed loading default config: %v", err)
	}
	if cfg.ActiveProfile != "default" {
		t.Errorf("expected active profile 'default', got '%s'", cfg.ActiveProfile)
	}

	// Test modifying & saving config
	cfg.OutputFormat = "json"
	cfg.AI.Provider = "gemini"
	cfg.AI.Model = "gemini-2.5-flash"
	if err := cm.SaveConfig(cfg); err != nil {
		t.Fatalf("failed saving config: %v", err)
	}

	reloaded, err := cm.LoadConfig()
	if err != nil {
		t.Fatalf("failed reloading config: %v", err)
	}
	if reloaded.OutputFormat != "json" {
		t.Errorf("expected output format 'json', got '%s'", reloaded.OutputFormat)
	}

	// Test profile storage
	creds := &ProfileCredentials{
		Name:        "work",
		DisplayName: "Santusht Kotai",
		Email:       "test@example.com",
		AccessToken: "test-token-123",
		Scopes:      []string{"openid", "profile", "w_member_social"},
	}

	if err := cm.SaveProfile(creds); err != nil {
		t.Fatalf("failed saving profile: %v", err)
	}

	loadedCreds, err := cm.LoadProfile("work")
	if err != nil {
		t.Fatalf("failed loading profile: %v", err)
	}
	if loadedCreds.DisplayName != "Santusht Kotai" {
		t.Errorf("expected display name 'Santusht Kotai', got '%s'", loadedCreds.DisplayName)
	}

	// Test listing profiles
	profiles, err := cm.ListProfiles()
	if err != nil {
		t.Fatalf("failed listing profiles: %v", err)
	}
	if len(profiles) != 1 || profiles[0] != "work" {
		t.Errorf("expected ['work'], got %v", profiles)
	}

	// Test deleting profile
	if err := cm.DeleteProfile("work"); err != nil {
		t.Fatalf("failed deleting profile: %v", err)
	}

	if _, err := cm.LoadProfile("work"); err == nil {
		t.Errorf("expected error loading deleted profile, got nil")
	}
}

func TestDirectoryPaths(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "ldin-paths-test")
	cm := &ConfigManager{BaseDir: tempDir}

	if cm.ConfigFilePath() != filepath.Join(tempDir, "config.yaml") {
		t.Errorf("unexpected config file path: %s", cm.ConfigFilePath())
	}
	if cm.ProfilesDir() != filepath.Join(tempDir, "profiles") {
		t.Errorf("unexpected profiles dir: %s", cm.ProfilesDir())
	}
	if cm.DraftsDir() != filepath.Join(tempDir, "drafts") {
		t.Errorf("unexpected drafts dir: %s", cm.DraftsDir())
	}
	if cm.ContextDir() != filepath.Join(tempDir, "context") {
		t.Errorf("unexpected context dir: %s", cm.ContextDir())
	}
}
