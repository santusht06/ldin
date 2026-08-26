// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/santusht/ldin/internal/output"
)

var (
	flagOrgFile string
)

var orgCmd = &cobra.Command{
	Use:     "org",
	Aliases: []string{"organization"},
	Short:   "Manage LinkedIn Company Pages & Organizations",
	Long:    `List member organizations, inspect follower demographics, and publish posts as an organization administrator.`,
}

var orgListCmd = &cobra.Command{
	Use:   "list",
	Short: "List LinkedIn organizations and company pages you administer",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		orgs, err := LinkedInClient.ListOrganizations(ctx)
		if err != nil {
			return err
		}

		return Formatter.Print(orgs, func() {
			fmt.Println(output.TitleStyle.Render(" Managed Organizations "))
			var rows [][]string
			for _, o := range orgs {
				rows = append(rows, []string{
					o.URN,
					o.LocalizedName,
					o.Role,
					fmt.Sprintf("%d", o.FollowerCount),
					o.Website,
				})
			}
			Formatter.PrintTable([]string{"Organization URN", "Company Name", "Your Role", "Followers", "Website"}, rows)
		})
	},
}

var orgPostCmd = &cobra.Command{
	Use:   "post <org-urn> [commentary]",
	Args:  cobra.MinimumNArgs(1),
	Short: "Publish a post to LinkedIn on behalf of an organization",
	RunE: func(cmd *cobra.Command, args []string) error {
		orgURN := args[0]
		commentary := ""
		if len(args) > 1 {
			commentary = strings.Join(args[1:], " ")
		}

		if flagOrgFile != "" {
			data, err := os.ReadFile(flagOrgFile)
			if err != nil {
				return err
			}
			commentary = string(data)
		}

		if commentary == "" {
			return fmt.Errorf("post commentary is required")
		}

		ctx := context.Background()
		Formatter.Info("Publishing post for organization %s...", orgURN)
		resp, err := LinkedInClient.PostAsOrganization(ctx, orgURN, commentary)
		if err != nil {
			return fmt.Errorf("failed publishing as organization: %w", err)
		}

		return Formatter.Print(resp, func() {
			Formatter.Success("Organization post published successfully! (ID: %s)", resp.ID)
		})
	},
}

func init() {
	orgPostCmd.Flags().StringVarP(&flagOrgFile, "file", "f", "", "Read commentary from file")

	orgCmd.AddCommand(orgListCmd)
	orgCmd.AddCommand(orgPostCmd)

	RootCmd.AddCommand(orgCmd)
}
