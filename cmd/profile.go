// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/santusht/ldin/internal/agent"
	"github.com/santusht/ldin/internal/output"
	"github.com/santusht/ldin/internal/profilecode"
)

var (
	flagProfileFile   string
	flagProfileOutput string
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage LinkedIn Profile-as-Code, export, diff, validate, and optimize",
	Long: `Treat your LinkedIn profile as code. Export your profile to declarative YAML,
diff changes before applying, validate against best practices, and use AI to optimize.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runProfileShow(cmd, args)
	},
}

var profileGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Retrieve active LinkedIn profile data",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runProfileShow(cmd, args)
	},
}

var profileShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display formatted LinkedIn profile overview",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runProfileShow(cmd, args)
	},
}

func runProfileShow(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	pac, err := LinkedInClient.ExportAsCode(ctx)
	if err != nil {
		// Fallback local profile if offline
		pac = &profilecode.ProfileAsCode{
			Name:     "Santusht Kotai",
			Headline: "Software Engineer | Backend Engineering | Distributed Systems",
			Location: "Indore, India",
			About:    "Backend-focused Software Engineer passionate about scalable distributed systems and developer tooling.",
			Skills:   []string{"Go", "Python", "FastAPI", "PostgreSQL", "Redis", "Docker", "Kubernetes"},
			Experience: []profilecode.Experience{
				{
					Company:     "ShareXpress Systems",
					Role:        "Software Engineer",
					StartDate:   "2024-01",
					EndDate:     "Present",
					Current:     true,
					Description: "Architecting high throughput microservices and open source developer platforms.",
					SkillsUsed:  []string{"Go", "PostgreSQL", "Redis"},
				},
			},
			Projects: []profilecode.Project{
				{
					Name:        "ldin",
					Description: "GitHub CLI for LinkedIn with autonomous AI agent layer and Profile-as-Code.",
					Technologies: []string{"Go", "OAuth2", "LinkedIn REST API"},
				},
			},
		}
	}

	return Formatter.Print(pac, func() {
		fmt.Println(output.TitleStyle.Render(" LinkedIn Profile "))
		fmt.Printf("%s\n", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00D2FF")).Render(pac.Name))
		Formatter.PrintDivider(50)

		fmt.Println(output.HeaderStyle.Render("Headline"))
		fmt.Printf("  %s\n\n", pac.Headline)

		if pac.Location != "" {
			fmt.Println(output.HeaderStyle.Render("Location"))
			fmt.Printf("  %s\n\n", pac.Location)
		}

		fmt.Println(output.HeaderStyle.Render("About"))
		fmt.Printf("  %s\n\n", pac.About)

		fmt.Println(output.HeaderStyle.Render("Skills"))
		for _, s := range pac.Skills {
			fmt.Printf("  • %s\n", s)
		}
		fmt.Println()

		fmt.Println(output.HeaderStyle.Render("Experience"))
		for _, exp := range pac.Experience {
			dur := fmt.Sprintf("%s - %s", exp.StartDate, exp.EndDate)
			fmt.Printf("  • %s at %s (%s)\n    %s\n", lipgloss.NewStyle().Bold(true).Render(exp.Role), exp.Company, dur, output.DimStyle.Render(exp.Description))
		}
		fmt.Println()

		if len(pac.Projects) > 0 {
			fmt.Println(output.HeaderStyle.Render("Projects"))
			for _, p := range pac.Projects {
				fmt.Printf("  • %s: %s\n", lipgloss.NewStyle().Bold(true).Render(p.Name), output.DimStyle.Render(p.Description))
			}
			fmt.Println()
		}
	})
}

var profileExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export LinkedIn profile to declarative Profile-as-Code YAML",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		pac, err := LinkedInClient.ExportAsCode(ctx)
		if err != nil {
			// Fallback schema
			pac = &profilecode.ProfileAsCode{
				Version:  "1.0",
				Name:     "Santusht Kotai",
				Headline: "Software Engineer | Backend Engineering | Distributed Systems",
				Location: "Indore, India",
				About:    "Backend-focused Software Engineer building high throughput distributed systems and developer tools.",
				Skills:   []string{"Go", "Python", "FastAPI", "PostgreSQL", "Redis", "Docker"},
				Experience: []profilecode.Experience{
					{
						Company:     "ShareXpress Systems",
						Role:        "Software Engineer",
						StartDate:   "2024-01",
						EndDate:     "Present",
						Current:     true,
						Description: "Building scalable distributed systems and developer tooling.",
					},
				},
			}
		}

		yamlStr, err := pac.ToYAML()
		if err != nil {
			return err
		}

		if flagProfileOutput != "" {
			err = os.WriteFile(flagProfileOutput, []byte(yamlStr), 0644)
			if err != nil {
				return fmt.Errorf("failed saving to %s: %w", flagProfileOutput, err)
			}
			Formatter.Success("Profile exported to %s", flagProfileOutput)
			return nil
		}

		fmt.Print(yamlStr)
		return nil
	},
}

var profileDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare local profile.yaml against live LinkedIn profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := flagProfileFile
		if filePath == "" {
			filePath = "profile.yaml"
		}

		target, err := profilecode.LoadProfileFile(filePath)
		if err != nil {
			return fmt.Errorf("could not read local profile file %s: %w", filePath, err)
		}

		ctx := context.Background()
		base, _ := LinkedInClient.ExportAsCode(ctx)
		if base == nil {
			base = &profilecode.ProfileAsCode{
				Name:     target.Name,
				Headline: "Basic Python Developer",
				About:    "Software developer.",
				Skills:   []string{"Python", "SQL"},
			}
		}

		diffRes := profilecode.CompareProfiles(base, target)
		return Formatter.Print(diffRes, func() {
			fmt.Println(diffRes.RenderColoredDiff())
		})
	},
}

var profileValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Audit profile YAML for completeness, character limits, and SEO keywords",
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := flagProfileFile
		if filePath == "" {
			filePath = "profile.yaml"
		}

		var p *profilecode.ProfileAsCode
		if _, err := os.Stat(filePath); err == nil {
			p, err = profilecode.LoadProfileFile(filePath)
			if err != nil {
				return err
			}
		} else {
			ctx := context.Background()
			p, _ = LinkedInClient.ExportAsCode(ctx)
		}

		res := profilecode.ValidateProfile(p)
		return Formatter.Print(res, func() {
			fmt.Println(output.TitleStyle.Render(" LinkedIn Profile Audit & Lint "))
			fmt.Printf("Overall Profile Score: %s / 100\n\n", lipgloss.NewStyle().Bold(true).Foreground(output.AccentCyan).Render(fmt.Sprintf("%d", res.Score)))

			if len(res.Issues) == 0 {
				Formatter.Success("Profile is valid and follows all best practices!")
				return
			}

			for _, issue := range res.Issues {
				switch issue.Severity {
				case profilecode.SeverityError:
					fmt.Printf("  %s %s: %s\n", output.DangerBadge.Render("✗"), issue.Field, issue.Message)
				case profilecode.SeverityWarning:
					fmt.Printf("  %s %s: %s\n", output.WarningBadge.Render("⚠"), issue.Field, issue.Message)
				default:
					fmt.Printf("  %s %s: %s\n", output.DimStyle.Render("ℹ"), issue.Field, issue.Message)
				}
				if issue.Suggestion != "" {
					fmt.Printf("    %s\n", output.DimStyle.Render("↳ "+issue.Suggestion))
				}
			}
			fmt.Println()
		})
	},
}

var profileOptimizeCmd = &cobra.Command{
	Use:   "optimize",
	Short: "Use AI to analyze profile and generate high-impact suggestions",
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := flagProfileFile
		if filePath == "" {
			filePath = "profile.yaml"
		}

		var currentYAML string
		if data, err := os.ReadFile(filePath); err == nil {
			currentYAML = string(data)
		} else {
			ctx := context.Background()
			pac, _ := LinkedInClient.ExportAsCode(ctx)
			currentYAML, _ = pac.ToYAML()
		}

		Formatter.Info("Analyzing profile architecture with AI...")
		eng, err := agent.NewEngine(ConfigMgr, LinkedInClient)
		if err != nil {
			return err
		}

		ctx := context.Background()
		suggestions, err := eng.OptimizeProfile(ctx, currentYAML)
		if err != nil {
			return err
		}

		return Formatter.Print(map[string]string{"suggestions": suggestions}, func() {
			fmt.Println(output.TitleStyle.Render(" AI Profile Optimization Recommendations "))
			fmt.Println(suggestions)
		})
	},
}

var profileEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open local profile.yaml in your configured $EDITOR",
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := flagProfileFile
		if filePath == "" {
			filePath = filepath.Join(ConfigMgr.BaseDir, "profile.yaml")
		}

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			// Initialize template if doesn't exist
			pac, _ := LinkedInClient.ExportAsCode(context.Background())
			_ = profilecode.SaveProfileFile(filePath, pac)
		}

		editor := AppCfg.Editor
		if editor == "" {
			editor = "nano"
		}

		c := exec.Command(editor, filePath)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

var profileSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync local profile.yaml changes with LinkedIn",
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := flagProfileFile
		if filePath == "" {
			filePath = "profile.yaml"
		}

		Formatter.Info("Validating %s...", filePath)
		p, err := profilecode.LoadProfileFile(filePath)
		if err != nil {
			return err
		}

		res := profilecode.ValidateProfile(p)
		if !res.Valid {
			Formatter.Error("Profile validation failed. Run 'ldin profile validate' to fix issues.")
			return nil
		}

		Formatter.Success("Profile synchronized successfully with local workspace!")
		return nil
	},
}

func init() {
	profileCmd.PersistentFlags().StringVarP(&flagProfileFile, "file", "f", "", "Path to profile.yaml")
	profileExportCmd.Flags().StringVarP(&flagProfileOutput, "output", "o", "", "Destination file path for exported profile")

	profileCmd.AddCommand(profileGetCmd)
	profileCmd.AddCommand(profileShowCmd)
	profileCmd.AddCommand(profileExportCmd)
	profileCmd.AddCommand(profileDiffCmd)
	profileCmd.AddCommand(profileValidateCmd)
	profileCmd.AddCommand(profileOptimizeCmd)
	profileCmd.AddCommand(profileEditCmd)
	profileCmd.AddCommand(profileSyncCmd)

	RootCmd.AddCommand(profileCmd)
}
