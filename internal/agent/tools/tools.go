// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/santusht/ldin/internal/config"
	"github.com/santusht/ldin/internal/gitcontext"
	"github.com/santusht/ldin/internal/linkedin"
)

// AgentTool represents a callable capability for the AI agent
type AgentTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Execute     func(ctx context.Context, args map[string]interface{}) (interface{}, error) `json:"-"`
}

// ToolRegistry holds available tools for the agent loop
type ToolRegistry struct {
	tools  map[string]AgentTool
	client *linkedin.Client
	cm     *config.ConfigManager
}

// NewToolRegistry initializes the tool registry with LinkedIn client & context
func NewToolRegistry(cm *config.ConfigManager, client *linkedin.Client) *ToolRegistry {
	tr := &ToolRegistry{
		tools:  make(map[string]AgentTool),
		client: client,
		cm:     cm,
	}

	tr.registerDefaults()
	return tr
}

func (tr *ToolRegistry) registerDefaults() {
	// 1. Get Profile Tool
	tr.Register(AgentTool{
		Name:        "get_profile",
		Description: "Fetches current LinkedIn member profile details and headline",
		Execute: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			if tr.client == nil {
				return nil, fmt.Errorf("linkedin client not initialized")
			}
			return tr.client.GetCurrentMemberProfile(ctx)
		},
	})

	// 2. Export Profile Tool
	tr.Register(AgentTool{
		Name:        "export_profile",
		Description: "Exports member profile into structured ProfileAsCode YAML representation",
		Execute: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			if tr.client == nil {
				return nil, fmt.Errorf("linkedin client not initialized")
			}
			return tr.client.ExportAsCode(ctx)
		},
	})

	// 3. Get Posts Tool
	tr.Register(AgentTool{
		Name:        "get_posts",
		Description: "Lists recent posts by author to analyze writing style and tone",
		Execute: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			if tr.client == nil {
				return nil, fmt.Errorf("linkedin client not initialized")
			}
			count := 5
			if c, ok := args["count"].(float64); ok {
				count = int(c)
			}
			return tr.client.ListPosts(ctx, "", count, 0)
		},
	})

	// 4. Create Draft Post Tool
	tr.Register(AgentTool{
		Name:        "create_draft_post",
		Description: "Saves a generated post draft into ~/.ldin/drafts for preview and review",
		Execute: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			commentary, _ := args["commentary"].(string)
			if commentary == "" {
				return nil, fmt.Errorf("commentary text is required")
			}
			title, _ := args["title"].(string)

			draft := &linkedin.PostDraft{
				Title:       title,
				Commentary:  commentary,
				ContentType: linkedin.ContentTypeText,
				Visibility:  linkedin.VisibilityPublic,
			}
			err := linkedin.SaveDraft(tr.cm, draft)
			if err != nil {
				return nil, err
			}
			return map[string]string{
				"status":   "success",
				"draft_id": draft.ID,
				"message":  "Draft saved successfully",
			}, nil
		},
	})

	// 5. Get Analytics Tool
	tr.Register(AgentTool{
		Name:        "get_analytics",
		Description: "Retrieves recent post or profile engagement metrics",
		Execute: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			if tr.client == nil {
				return nil, fmt.Errorf("linkedin client not initialized")
			}
			return tr.client.GetProfileAnalytics(ctx, 30)
		},
	})

	// 6. Inspect Local Git Repo Tool
	tr.Register(AgentTool{
		Name:        "git_inspect",
		Description: "Inspects local git repository to extract recent commits, diff, and README context",
		Execute: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			dir, _ := args["dir"].(string)
			return gitcontext.InspectLocalRepo(dir, 5)
		},
	})

	// 7. Sync GitHub Repo Tool
	tr.Register(AgentTool{
		Name:        "github_sync",
		Description: "Fetches public repository commits, stars, topics, and description from GitHub",
		Execute: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			repo, _ := args["repo"].(string)
			if repo == "" {
				return nil, fmt.Errorf("repo argument 'owner/repo' is required")
			}
			return gitcontext.FetchGitHubRepo(repo)
		},
	})

	// 8. Read Local File Tool
	tr.Register(AgentTool{
		Name:        "read_file",
		Description: "Reads contents of a local markdown, code, or documentation file",
		Execute: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			path, _ := args["path"].(string)
			if path == "" {
				return nil, fmt.Errorf("file path is required")
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			str := string(data)
			if len(str) > 4000 {
				str = str[:4000] + "... [truncated]"
			}
			return str, nil
		},
	})
}

// Register adds a tool to the registry
func (tr *ToolRegistry) Register(tool AgentTool) {
	tr.tools[tool.Name] = tool
}

// Get finds a tool by name
func (tr *ToolRegistry) Get(name string) (AgentTool, bool) {
	t, ok := tr.tools[name]
	return t, ok
}

// List returns summary of registered tools
func (tr *ToolRegistry) List() []map[string]string {
	var list []map[string]string
	for _, t := range tr.tools {
		list = append(list, map[string]string{
			"name":        t.Name,
			"description": t.Description,
		})
	}
	return list
}

// ExecuteTool runs the requested tool
func (tr *ToolRegistry) ExecuteTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	tool, ok := tr.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool '%s' not recognized", name)
	}
	return tool.Execute(ctx, args)
}

// FormatToolsAsPrompt transforms tool definitions for LLM system prompt
func (tr *ToolRegistry) FormatToolsAsPrompt() string {
	b, _ := json.MarshalIndent(tr.List(), "", "  ")
	return string(b)
}
