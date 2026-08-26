// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package agent

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/santusht/ldin/internal/config"
)

// AgentMemory stores persistent developer context, tone notes, and synced repos
type AgentMemory struct {
	DeveloperBio      string            `json:"developer_bio,omitempty"`
	WritingStyleNotes string            `json:"writing_style_notes,omitempty"`
	TargetAudience    string            `json:"target_audience,omitempty"`
	PinnedSkills      []string          `json:"pinned_skills,omitempty"`
	SyncedRepos       map[string]string `json:"synced_repos,omitempty"` // repo -> summary
}

// LoadMemory loads context memory from ~/.ldin/context/memory.json
func LoadMemory(cm *config.ConfigManager) *AgentMemory {
	memPath := filepath.Join(cm.ContextDir(), "memory.json")
	data, err := os.ReadFile(memPath)
	if err != nil {
		return &AgentMemory{
			WritingStyleNotes: "Concise, authentic engineering voice. Focus on technical architecture, quantifiable metrics, and lessons learned.",
			TargetAudience:    "Software Engineers, Engineering Managers, Tech Recruiters",
			SyncedRepos:       make(map[string]string),
		}
	}
	var mem AgentMemory
	if err := json.Unmarshal(data, &mem); err != nil {
		return &AgentMemory{SyncedRepos: make(map[string]string)}
	}
	if mem.SyncedRepos == nil {
		mem.SyncedRepos = make(map[string]string)
	}
	return &mem
}

// SaveMemory persists context memory
func SaveMemory(cm *config.ConfigManager, mem *AgentMemory) error {
	memPath := filepath.Join(cm.ContextDir(), "memory.json")
	data, err := json.MarshalIndent(mem, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(memPath, data, 0600)
}
