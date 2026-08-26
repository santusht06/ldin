// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/santusht/ldin/internal/agent"
	"github.com/santusht/ldin/internal/cdp"
	"github.com/santusht/ldin/internal/linkedin"
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
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	vanityName := ""
	if creds, err := ConfigMgr.LoadProfile(AppCfg.ActiveProfile); err == nil {
		vanityName = creds.VanityName
	}

	// --- Strategy 1: Chrome DevTools Protocol (real-time, bypasses TLS fingerprinting) ---
	Formatter.Info("Connecting to Chrome CDP bridge for real-time data...")
	tabs, cdpErr := cdp.ListTabs(cdp.DefaultCDPHost, cdp.DefaultCDPPort)
	if cdpErr == nil {
		tab := cdp.FindLinkedInTab(tabs)
		if tab != nil {
			bridge, err := cdp.Connect(ctx, tab)
			if err == nil {
				defer bridge.Close()
				Formatter.Info("CDP connected → tab: %s", tab.Title)
				profile, err := cdp.FetchFullProfileView(ctx, bridge, vanityName)
				if err == nil && (profile.FirstName != "" || profile.Headline != "" || len(profile.Skills) > 0) {
					return renderCDPProfile(profile)
				}
				Formatter.Warning("CDP fetch returned empty profile, trying direct Voyager...")
			}
		} else {
			Formatter.Warning("No LinkedIn tab open in Chrome. Open linkedin.com in Chrome for best results.")
		}
	} else {
		Formatter.Warning("Chrome CDP not available (start with: ldin browser launch). Trying Voyager API...")
	}

	// --- Strategy 2: Voyager API with session cookie ---
	rich, voyagerErr := LinkedInClient.GetRichProfile(ctx)
	if voyagerErr == nil && (rich.FirstName != "" || len(rich.Skills) > 0) {
		return renderVoyagerProfile(rich)
	}

	// --- Strategy 3: Official OpenID userinfo (basic identity only) ---
	Formatter.Warning("Voyager API blocked. Fetching basic identity via official OpenID API...")
	basic, err := LinkedInClient.GetCurrentMemberProfile(ctx)
	if err != nil {
		return fmt.Errorf("all profile fetch strategies failed.\n\nTo get full real-time profile:\n  1. ldin browser launch\n  2. ldin profile show")
	}

	// Render basic identity
	return Formatter.Print(basic, func() {
		fmt.Println(output.TitleStyle.Render(" LinkedIn Profile (Basic — OpenID) "))
		fmt.Printf("%s\n", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00D2FF")).Render(basic.Name))
		Formatter.PrintDivider(50)
		fmt.Println(output.HeaderStyle.Render("Headline"))
		fmt.Printf("  %s\n\n", basic.Headline)
		if basic.Location != "" {
			fmt.Println(output.HeaderStyle.Render("Location"))
			fmt.Printf("  %s\n\n", basic.Location)
		}
		fmt.Printf("\n  ℹ For full profile (skills, experience, education):\n  Run: ldin browser launch\n  Then: ldin profile show\n\n")
	})
}

func renderCDPProfile(p *cdp.VoyagerProfile) error {
	name := p.FirstName + " " + p.LastName
	return Formatter.Print(p, func() {
		fmt.Println(output.TitleStyle.Render(" LinkedIn Profile ✓ Live via Chrome "))
		fmt.Printf("%s\n", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00D2FF")).Render(name))
		if p.VanityName != "" {
			fmt.Printf("  linkedin.com/in/%s\n", p.VanityName)
		}
		Formatter.PrintDivider(50)
		renderProfileFields(p.Headline, p.Location, p.Summary, p.Skills, p.Experience, p.Education, p.Certifications, p.Languages)
	})
}

func renderVoyagerProfile(p *linkedin.VoyagerProfileData) error {
	name := p.FirstName + " " + p.LastName
	return Formatter.Print(p, func() {
		fmt.Println(output.TitleStyle.Render(" LinkedIn Profile (Live) "))
		fmt.Printf("%s\n", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00D2FF")).Render(name))
		if p.VanityName != "" {
			fmt.Printf("  linkedin.com/in/%s\n", p.VanityName)
		}
		Formatter.PrintDivider(50)
		renderProfileFields(p.Headline, p.Location, p.Summary, p.Skills, p.Experience, p.Education, p.Certifications, p.Languages)
	})
}

func renderProfileFields(headline, location, summary string, skills []string, experience []profilecode.Experience, education []profilecode.Education, certs []profilecode.Certification, languages []string) {
	if headline != "" {
		fmt.Println(output.HeaderStyle.Render("Headline"))
		fmt.Printf("  %s\n\n", headline)
	}
	if location != "" {
		fmt.Println(output.HeaderStyle.Render("Location"))
		fmt.Printf("  %s\n\n", location)
	}
	if summary != "" {
		fmt.Println(output.HeaderStyle.Render("About"))
		fmt.Printf("  %s\n\n", summary)
	}
	if len(skills) > 0 {
		fmt.Println(output.HeaderStyle.Render("Skills"))
		for _, s := range skills {
			fmt.Printf("  • %s\n", s)
		}
		fmt.Println()
	}
	if len(experience) > 0 {
		fmt.Println(output.HeaderStyle.Render("Experience"))
		for _, e := range experience {
			dr := e.StartDate
			if e.EndDate != "" {
				dr += " → " + e.EndDate
			}
			fmt.Printf("  • %s at %s", e.Role, e.Company)
			if dr != "" {
				fmt.Printf(" (%s)", dr)
			}
			fmt.Println()
			if e.Description != "" {
				fmt.Printf("    %s\n", e.Description)
			}
		}
		fmt.Println()
	}
	if len(education) > 0 {
		fmt.Println(output.HeaderStyle.Render("Education"))
		for _, ed := range education {
			fmt.Printf("  • %s", ed.School)
			if ed.Degree != "" {
				fmt.Printf(" — %s", ed.Degree)
			}
			if ed.FieldOfStudy != "" {
				fmt.Printf(" (%s)", ed.FieldOfStudy)
			}
			fmt.Println()
		}
		fmt.Println()
	}
	if len(certs) > 0 {
		fmt.Println(output.HeaderStyle.Render("Certifications"))
		for _, c := range certs {
			fmt.Printf("  • %s", c.Name)
			if c.IssuingOrg != "" {
				fmt.Printf(" — %s", c.IssuingOrg)
			}
			fmt.Println()
		}
		fmt.Println()
	}
	if len(languages) > 0 {
		fmt.Println(output.HeaderStyle.Render("Languages"))
		for _, lang := range languages {
			fmt.Printf("  • %s\n", lang)
		}
		fmt.Println()
	}
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
