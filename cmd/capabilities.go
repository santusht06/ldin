// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/santusht/ldin/internal/capabilities"
	"github.com/santusht/ldin/internal/output"
)

var capabilitiesCmd = &cobra.Command{
	Use:     "capabilities",
	Aliases: []string{"caps"},
	Short:   "Audit LinkedIn API capabilities, supported scopes, and access tiers",
	Long: `Display the full LinkedIn API Capability Matrix against your active permissions.
Clearly highlights what endpoints are open for self-service vs those requiring Community Management approval.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var scopes []string
		if LinkedInClient != nil && LinkedInClient.Profile != nil {
			scopes = LinkedInClient.Profile.Scopes
		}

		eval := capabilities.EvaluateCapabilities(scopes)
		return Formatter.Print(eval, func() {
			fmt.Println(output.TitleStyle.Render(" LinkedIn API Capability Matrix "))
			fmt.Println(output.DimStyle.Render("Based on LinkedIn REST API (Version 202608 / Community Management Tiering)"))
			fmt.Println()

			var rows [][]string
			for _, item := range eval {
				status := output.SuccessBadge.Render("✓ Available")
				if !item.Available {
					status = output.WarningBadge.Render("⚠ Restricted")
				}
				rows = append(rows, []string{
					item.Capability.Category,
					item.Capability.Name,
					status,
					string(item.Capability.Tier),
					strings.Join(item.Capability.RequiredScopes, ", "),
				})
			}

			Formatter.PrintTable([]string{"Category", "Capability", "Status", "Access Tier", "Required Scopes"}, rows)
		})
	},
}

func init() {
	RootCmd.AddCommand(capabilitiesCmd)
}
