// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package gitcontext

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GitContributionContext encapsulates extracted developer context from git / github
type GitContributionContext struct {
	RepoName      string   `json:"repo_name" yaml:"repo_name"`
	Branch        string   `json:"branch" yaml:"branch"`
	RecentCommits []string `json:"recent_commits" yaml:"recent_commits"`
	LatestDiff    string   `json:"latest_diff,omitempty" yaml:"latest_diff,omitempty"`
	READMEExcerpt string   `json:"readme_excerpt,omitempty" yaml:"readme_excerpt,omitempty"`
	Languages     []string `json:"languages,omitempty" yaml:"languages,omitempty"`
	Tags          []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Summary       string   `json:"summary,omitempty" yaml:"summary,omitempty"`
}

// InspectLocalRepo gathers commit logs, branch, and diff from the current or specified path
func InspectLocalRepo(dirPath string, maxCommits int) (*GitContributionContext, error) {
	if dirPath == "" {
		dirPath = "."
	}
	if maxCommits <= 0 {
		maxCommits = 5
	}

	abs, _ := filepath.Abs(dirPath)
	repoName := filepath.Base(abs)
	branch := "main"
	var commits []string

	// 1. Get branch
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = dirPath
	if branchBytes, err := branchCmd.Output(); err == nil {
		branch = strings.TrimSpace(string(branchBytes))
	}

	// 2. Get repo name
	repoNameCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	repoNameCmd.Dir = dirPath
	if topBytes, err := repoNameCmd.Output(); err == nil {
		name := filepath.Base(strings.TrimSpace(string(topBytes)))
		if name != "" && name != "." {
			repoName = name
		}
	}

	// 3. Get recent commits
	logCmd := exec.Command("git", "log", fmt.Sprintf("-n%d", maxCommits), "--pretty=format:%h - %s (%cr) <%an>")
	logCmd.Dir = dirPath
	if logBytes, err := logCmd.Output(); err == nil {
		for _, line := range strings.Split(string(logBytes), "\n") {
			if strings.TrimSpace(line) != "" {
				commits = append(commits, strings.TrimSpace(line))
			}
		}
	}

	if len(commits) == 0 {
		commits = append(commits, "Initial release and architecture build")
	}

	// 4. Get short stat diff
	diffCmd := exec.Command("git", "diff", "--stat", "HEAD~1..HEAD")
	diffCmd.Dir = dirPath
	diffBytes, _ := diffCmd.Output()
	diffStat := strings.TrimSpace(string(diffBytes))

	// 5. Read README if present
	readmeExcerpt := ""
	readmePath := filepath.Join(dirPath, "README.md")
	if data, err := os.ReadFile(readmePath); err == nil {
		str := string(data)
		if len(str) > 500 {
			readmeExcerpt = str[:500] + "..."
		} else {
			readmeExcerpt = str
		}
	}

	return &GitContributionContext{
		RepoName:      repoName,
		Branch:        branch,
		RecentCommits: commits,
		LatestDiff:    diffStat,
		READMEExcerpt: readmeExcerpt,
	}, nil
}

// FetchGitHubRepo fetches public repository info and releases from GitHub API
func FetchGitHubRepo(ownerRepo string) (*GitContributionContext, error) {
	parts := strings.Split(ownerRepo, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid format '%s', expected 'owner/repo'", ownerRepo)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", parts[0], parts[1])
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "ldin-cli")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var data struct {
		Name        string `json:"name"`
		FullName    string `json:"full_name"`
		Description string `json:"description"`
		Language    string `json:"language"`
		Topics      []string `json:"topics"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&data)

	// Fetch latest commits
	commitsURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits?per_page=5", parts[0], parts[1])
	req2, _ := http.NewRequest("GET", commitsURL, nil)
	req2.Header.Set("User-Agent", "ldin-cli")
	resp2, err := client.Do(req2)
	var commits []string
	if err == nil {
		defer resp2.Body.Close()
		var commitData []struct {
			Sha    string `json:"sha"`
			Commit struct {
				Message string `json:"message"`
			} `json:"commit"`
		}
		body, _ := io.ReadAll(resp2.Body)
		_ = json.Unmarshal(body, &commitData)
		for _, c := range commitData {
			shaShort := c.Sha
			if len(shaShort) > 7 {
				shaShort = shaShort[:7]
			}
			msg := strings.Split(c.Commit.Message, "\n")[0]
			commits = append(commits, fmt.Sprintf("%s - %s", shaShort, msg))
		}
	}

	languages := []string{}
	if data.Language != "" {
		languages = append(languages, data.Language)
	}

	return &GitContributionContext{
		RepoName:      data.FullName,
		RecentCommits: commits,
		Summary:       data.Description,
		Languages:     languages,
		Tags:          data.Topics,
	}, nil
}
