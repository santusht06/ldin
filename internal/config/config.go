// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AppConfig represents global configuration stored in ~/.ldin/config.yaml
type AppConfig struct {
	Version             string      `yaml:"version" json:"version"`
	ActiveProfile       string      `yaml:"active_profile" json:"active_profile"`
	OutputFormat        string      `yaml:"output_format" json:"output_format"` // "human", "json", "yaml", "csv"
	LinkedInAPIVersion  string      `yaml:"linkedin_api_version" json:"linkedin_api_version"`
	DefaultOrganization string      `yaml:"default_organization,omitempty" json:"default_organization,omitempty"`
	Editor              string      `yaml:"editor,omitempty" json:"editor,omitempty"`
	AI                  AIConfig    `yaml:"ai" json:"ai"`
	Agent               AgentConfig `yaml:"agent" json:"agent"`
}

// AIConfig holds credentials and model configuration for AI operations
type AIConfig struct {
	Provider string `yaml:"provider" json:"provider"` // "gemini", "openai", "claude", "ollama"
	Model    string `yaml:"model" json:"model"`
	APIKey   string `yaml:"api_key,omitempty" json:"api_key,omitempty"`
	BaseURL  string `yaml:"base_url,omitempty" json:"base_url,omitempty"`
}

// AgentConfig controls autonomous agent behavior & permissions
type AgentConfig struct {
	AutoPublish   bool     `yaml:"auto_publish" json:"auto_publish"`
	AllowedScopes []string `yaml:"allowed_scopes" json:"allowed_scopes"` // "read", "draft", "publish", "delete"
	MaxIterations int      `yaml:"max_iterations" json:"max_iterations"`
}

// ProfileCredentials represents an authenticated identity stored in ~/.ldin/profiles/<name>.json
type ProfileCredentials struct {
	Name          string   `json:"name"`
	MemberID      string   `json:"member_id"`     // urn:li:person:xxxx
	MemberURN     string   `json:"member_urn"`    // urn:li:person:xxxx
	VanityName    string   `json:"vanity_name"`   // custom handle
	DisplayName   string   `json:"display_name"`
	Email         string   `json:"email"`
	AccessToken   string   `json:"access_token"`
	SessionCookie string   `json:"session_cookie,omitempty"` // li_at cookie
	CSRFToken     string   `json:"csrf_token,omitempty"`     // JSESSIONID / ajax token
	RefreshToken  string   `json:"refresh_token,omitempty"`
	ExpiresAt     int64    `json:"expires_at"` // Unix timestamp
	Scopes        []string `json:"scopes"`
	ClientID      string   `json:"client_id,omitempty"`
	ClientSecret  string   `json:"client_secret,omitempty"`
}

// ConfigManager handles loading and saving settings in ~/.ldin
type ConfigManager struct {
	BaseDir string
}

// NewConfigManager returns a ConfigManager initialized with ~/.ldin or LDIN_HOME
func NewConfigManager() (*ConfigManager, error) {
	home := os.Getenv("LDIN_HOME")
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to determine user home directory: %w", err)
		}
		home = filepath.Join(userHome, ".ldin")
	}

	if err := os.MkdirAll(home, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config dir %s: %w", home, err)
	}

	profilesDir := filepath.Join(home, "profiles")
	if err := os.MkdirAll(profilesDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create profiles dir: %w", err)
	}

	draftsDir := filepath.Join(home, "drafts")
	if err := os.MkdirAll(draftsDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create drafts dir: %w", err)
	}

	contextDir := filepath.Join(home, "context")
	if err := os.MkdirAll(contextDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create context dir: %w", err)
	}

	return &ConfigManager{BaseDir: home}, nil
}

// ConfigFilePath returns the absolute path to config.yaml
func (cm *ConfigManager) ConfigFilePath() string {
	return filepath.Join(cm.BaseDir, "config.yaml")
}

// ProfilesDir returns path to profiles directory
func (cm *ConfigManager) ProfilesDir() string {
	return filepath.Join(cm.BaseDir, "profiles")
}

// DraftsDir returns path to drafts directory
func (cm *ConfigManager) DraftsDir() string {
	return filepath.Join(cm.BaseDir, "drafts")
}

// ContextDir returns path to context directory
func (cm *ConfigManager) ContextDir() string {
	return filepath.Join(cm.BaseDir, "context")
}

// LoadConfig reads config.yaml or returns sensible defaults
func (cm *ConfigManager) LoadConfig() (*AppConfig, error) {
	cfgPath := cm.ConfigFilePath()
	cfg := &AppConfig{
		Version:            "1.0.0",
		ActiveProfile:      "default",
		OutputFormat:       "human",
		LinkedInAPIVersion: "202608",
		Editor:             os.Getenv("EDITOR"),
		AI: AIConfig{
			Provider: "gemini",
			Model:    "gemini-2.5-flash",
		},
		Agent: AgentConfig{
			AutoPublish:   false,
			AllowedScopes: []string{"read", "draft", "ai"},
			MaxIterations: 10,
		},
	}

	if cfg.Editor == "" {
		cfg.Editor = "nano"
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			_ = cm.SaveConfig(cfg)
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("error parsing config.yaml: %w", err)
	}

	return cfg, nil
}

// SaveConfig persists configuration to config.yaml
func (cm *ConfigManager) SaveConfig(cfg *AppConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("error encoding config to YAML: %w", err)
	}
	return os.WriteFile(cm.ConfigFilePath(), data, 0600)
}

// LoadProfile loads credentials for a named profile
func (cm *ConfigManager) LoadProfile(name string) (*ProfileCredentials, error) {
	if name == "" {
		cfg, _ := cm.LoadConfig()
		if cfg != nil && cfg.ActiveProfile != "" {
			name = cfg.ActiveProfile
		} else {
			name = "default"
		}
	}

	profilePath := filepath.Join(cm.ProfilesDir(), name+".json")
	data, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("profile '%s' not found: %w", name, err)
	}

	var creds ProfileCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("error reading profile '%s': %w", name, err)
	}
	return &creds, nil
}

// SaveProfile writes credentials for a named profile
func (cm *ConfigManager) SaveProfile(creds *ProfileCredentials) error {
	if creds.Name == "" {
		creds.Name = "default"
	}
	profilePath := filepath.Join(cm.ProfilesDir(), creds.Name+".json")
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding profile credentials: %w", err)
	}
	return os.WriteFile(profilePath, data, 0600)
}

// ListProfiles returns all available profile names
func (cm *ConfigManager) ListProfiles() ([]string, error) {
	entries, err := os.ReadDir(cm.ProfilesDir())
	if err != nil {
		return nil, err
	}

	var profiles []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			profiles = append(profiles, entry.Name()[:len(entry.Name())-5])
		}
	}
	return profiles, nil
}

// DeleteProfile removes a stored profile
func (cm *ConfigManager) DeleteProfile(name string) error {
	profilePath := filepath.Join(cm.ProfilesDir(), name+".json")
	return os.Remove(profilePath)
}
