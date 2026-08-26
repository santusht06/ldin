// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/santusht/ldin/internal/agent"
	"github.com/santusht/ldin/internal/gitcontext"
	"github.com/santusht/ldin/internal/output"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Sync GitHub / local Git developer context with ldin memory",
	Long: `Extract commit history, pull request descriptions, diff statistics, and README details
so the AI agent understands your real-world engineering work.`,
}

var repoSyncCmd = &cobra.Command{
	Use:   "sync [owner/repo | local-dir]",
	Short: "Sync repository context for AI agent memory",
	RunE: func(cmd *cobra.Command, args []string) error {
		target := "."
		if len(args) > 0 {
			target = args[0]
		}

		var gitCtx *gitcontext.GitContributionContext
		var err error

		if strings.Contains(target, "/") && !strings.HasPrefix(target, ".") && !strings.HasPrefix(target, "/") {
			// GitHub remote repo
			Formatter.Info("Syncing GitHub repository context from %s...", target)
			gitCtx, err = gitcontext.FetchGitHubRepo(target)
		} else {
			// Local Git repo
			Formatter.Info("Inspecting local git repository at %s...", target)
			gitCtx, err = gitcontext.InspectLocalRepo(target, 5)
		}

		if err != nil {
			return fmt.Errorf("failed syncing repo context: %w", err)
		}

		// Store in agent memory
		mem := agent.LoadMemory(ConfigMgr)
		summary := fmt.Sprintf("Repository %s (%s). Recent commits: %s", gitCtx.RepoName, strings.Join(gitCtx.Languages, ", "), strings.Join(gitCtx.RecentCommits, "; "))
		mem.SyncedRepos[gitCtx.RepoName] = summary
		_ = agent.SaveMemory(ConfigMgr, mem)

		return Formatter.Print(gitCtx, func() {
			Formatter.Success("Synced repository '%s' to ldin context memory!", gitCtx.RepoName)
			Formatter.PrintKeyValue("Branch", gitCtx.Branch)
			if len(gitCtx.Languages) > 0 {
				Formatter.PrintKeyValue("Tech Stack", strings.Join(gitCtx.Languages, ", "))
			}
			fmt.Println()
			fmt.Println(output.HeaderStyle.Render("Recent Commits Ingested:"))
			for _, c := range gitCtx.RecentCommits {
				fmt.Printf("  • %s\n", c)
			}
			fmt.Println()
			Formatter.Info("Tip: Now run `ldin ai \"write a LinkedIn post about my recent work on %s\"`", gitCtx.RepoName)
		})
	},
}

func init() {
	repoCmd.AddCommand(repoSyncCmd)
	RootCmd.AddCommand(repoCmd)
}
