// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package agent

import (
	"fmt"
	"strings"

	"github.com/santusht/ldin/internal/config"
)

// AgentPermission defines permitted action scope for the autonomous agent
type AgentPermission string

const (
	PermRead    AgentPermission = "read"
	PermDraft   AgentPermission = "draft"
	PermAI      AgentPermission = "ai"
	PermPublish AgentPermission = "publish"
	PermDelete  AgentPermission = "delete"
)

// PermissionGuard validates actions before agent executes them
type PermissionGuard struct {
	Allowed map[AgentPermission]bool
}

// NewPermissionGuard initializes guard from config
func NewPermissionGuard(cfg *config.AgentConfig) *PermissionGuard {
	pg := &PermissionGuard{
		Allowed: make(map[AgentPermission]bool),
	}

	// Always allow read, draft, and ai by default
	pg.Allowed[PermRead] = true
	pg.Allowed[PermDraft] = true
	pg.Allowed[PermAI] = true

	if cfg != nil {
		for _, s := range cfg.AllowedScopes {
			pg.Allowed[AgentPermission(strings.ToLower(strings.TrimSpace(s)))] = true
		}
		if cfg.AutoPublish {
			pg.Allowed[PermPublish] = true
		}
	}

	return pg
}

// CheckPermission returns nil if action is permitted or descriptive error
func (pg *PermissionGuard) CheckPermission(perm AgentPermission) error {
	if pg.Allowed[perm] {
		return nil
	}
	return fmt.Errorf("permission '%s' is not granted to ldin agent. Run 'ldin agent allow %s' to authorize", perm, perm)
}

// SetPermission dynamically enables or disables an agent scope
func (pg *PermissionGuard) SetPermission(perm AgentPermission, allow bool) {
	pg.Allowed[perm] = allow
}

// GetStatus returns the current permission breakdown
func (pg *PermissionGuard) GetStatus() map[string]bool {
	return map[string]bool{
		"read (Inspect profile, posts, repos, analytics)":    pg.Allowed[PermRead],
		"draft (Generate post drafts, optimize profile text)": pg.Allowed[PermDraft],
		"ai (Run reasoning models & content analyzers)":       pg.Allowed[PermAI],
		"publish (Directly publish to LinkedIn live feed)":   pg.Allowed[PermPublish],
		"delete (Delete posts or comments)":                  pg.Allowed[PermDelete],
	}
}
