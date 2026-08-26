// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/santusht/ldin/internal/agent/providers"
	"github.com/santusht/ldin/internal/agent/tools"
	"github.com/santusht/ldin/internal/config"
	"github.com/santusht/ldin/internal/linkedin"
)

// AgentPlanStep represents a single reasoning/action step by the agent
type AgentPlanStep struct {
	Thought     string `json:"thought"`
	Action      string `json:"action"`
	ActionInput string `json:"action_input,omitempty"`
	Result      string `json:"result,omitempty"`
}

// AgentExecutionResult holds the final outcome
type AgentExecutionResult struct {
	Response     string          `json:"response"`
	Steps        []AgentPlanStep `json:"steps"`
	GeneratedPost string          `json:"generated_post,omitempty"`
	DraftID      string          `json:"draft_id,omitempty"`
	ActionTaken  string          `json:"action_taken,omitempty"`
}

// Engine coordinates AI agent execution
type Engine struct {
	Config     *config.AppConfig
	ConfigMgr  *config.ConfigManager
	Provider   providers.Provider
	Tools      *tools.ToolRegistry
	Guard      *PermissionGuard
	Memory     *AgentMemory
	LinkedIn   *linkedin.Client
}

// NewEngine instantiates the AI Agent Engine
func NewEngine(cm *config.ConfigManager, client *linkedin.Client) (*Engine, error) {
	cfg, err := cm.LoadConfig()
	if err != nil {
		return nil, err
	}

	provider, err := providers.GetProvider(&cfg.AI)
	if err != nil {
		return nil, err
	}

	toolReg := tools.NewToolRegistry(cm, client)
	guard := NewPermissionGuard(&cfg.Agent)
	mem := LoadMemory(cm)

	return &Engine{
		Config:    cfg,
		ConfigMgr: cm,
		Provider:  provider,
		Tools:     toolReg,
		Guard:     guard,
		Memory:    mem,
		LinkedIn:  client,
	}, nil
}

// Run executes a natural language instruction through the agent
func (e *Engine) Run(ctx context.Context, userInstruction string) (*AgentExecutionResult, error) {
	systemPrompt := `You are ldin AI — an expert LinkedIn strategist and developer workspace agent.
Your mission is to manage professional identity, optimize profile-as-code, and create authentic, high-impact technical LinkedIn content from developer workflows.

Core Guidelines:
1. Speak in a genuine, developer-first voice. Avoid cringey AI buzzwords, hype, or generic corporate filler.
2. Focus on architecture, technical insights, engineering trade-offs, and measurable outcomes.
3. Structure LinkedIn posts with strong hooks, readable bullet points, clean spacing, and 3-5 relevant hashtags.
4. When optimizing profiles or headlines, emphasize core stack, distributed systems, and backend mastery.
5. NEVER publish directly without explicit user approval. Always save as draft first.`

	// Step 1: Context augmentation
	var contextNotes []string
	if e.LinkedIn != nil && e.LinkedIn.Profile != nil {
		contextNotes = append(contextNotes, fmt.Sprintf("Active User: %s (%s)", e.LinkedIn.Profile.DisplayName, e.LinkedIn.Profile.Email))
	}
	if e.Memory.WritingStyleNotes != "" {
		contextNotes = append(contextNotes, fmt.Sprintf("User Writing Style: %s", e.Memory.WritingStyleNotes))
	}

	promptWithContext := userInstruction
	if len(contextNotes) > 0 {
		promptWithContext = fmt.Sprintf("Context:\n%s\n\nInstruction: %s", strings.Join(contextNotes, "\n"), userInstruction)
	}

	// Step 2: Call AI Provider
	completion, err := e.Provider.GenerateCompletion(ctx, systemPrompt, promptWithContext)
	if err != nil {
		return nil, fmt.Errorf("agent generation failed: %w", err)
	}

	// Step 3: Automatically persist draft if user asked for a post
	result := &AgentExecutionResult{
		Response: completion,
		Steps: []AgentPlanStep{
			{
				Thought: "Analyzed developer context and generated optimized content.",
				Action:  "generate_content",
				Result:  "Content synthesized successfully.",
			},
		},
	}

	cleanInstruction := strings.ToLower(userInstruction)
	if strings.Contains(cleanInstruction, "post") || strings.Contains(cleanInstruction, "publish") || strings.Contains(cleanInstruction, "draft") {
		draft := &linkedin.PostDraft{
			Title:       "AI Generated Post",
			Commentary:  completion,
			ContentType: linkedin.ContentTypeText,
			Visibility:  linkedin.VisibilityPublic,
		}
		if err := linkedin.SaveDraft(e.ConfigMgr, draft); err == nil {
			result.DraftID = draft.ID
			result.GeneratedPost = completion
			result.ActionTaken = fmt.Sprintf("Saved draft to ~/.ldin/drafts/%s.json", draft.ID)
		}
	}

	return result, nil
}

// OptimizeProfile analyzes profile text and returns recommendations & improved YAML
func (e *Engine) OptimizeProfile(ctx context.Context, currentProfileYAML string) (string, error) {
	systemPrompt := `You are a Senior Principal Engineer and Technical Career Strategist.
Analyze the provided LinkedIn Profile-as-Code YAML. 
Provide:
1. Specific strategic critiques (Headline strength, keyword indexing, impact metrics in experience).
2. An optimized, high-converting YAML profile version suitable for senior developer roles.`

	userPrompt := fmt.Sprintf("Profile YAML:\n```yaml\n%s\n```\n\nOptimize this profile for high technical impact.", currentProfileYAML)

	return e.Provider.GenerateCompletion(ctx, systemPrompt, userPrompt)
}

// GeneratePostFromGit inspects local git context and creates a LinkedIn post
func (e *Engine) GeneratePostFromGit(ctx context.Context, repoPath, customPrompt string) (string, error) {
	gitCtx, err := e.Tools.ExecuteTool(ctx, "git_inspect", map[string]interface{}{"dir": repoPath})
	if err != nil {
		return "", fmt.Errorf("failed inspecting git repository: %w", err)
	}

	systemPrompt := `You are an expert developer-advocate. Turn raw git commits, diff stats, and project context into an engaging, technical LinkedIn post.
Focus on:
- What problem was solved
- Interesting engineering decisions & trade-offs
- Code patterns or tools used
- Call to action asking other developers for their approach`

	userPrompt := fmt.Sprintf("Git Repository Context:\n%+v\n\nUser Notes: %s", gitCtx, customPrompt)

	return e.Provider.GenerateCompletion(ctx, systemPrompt, userPrompt)
}
