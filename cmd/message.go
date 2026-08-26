// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/santusht/ldin/internal/output"
)

var messageCmd = &cobra.Command{
	Use:     "message",
	Aliases: []string{"msg", "dm", "messages", "dms"},
	Short:   "Direct Messaging & InMail (Enterprise Partner Tier)",
	Long: `Inspect LinkedIn direct messaging (DMs) and InMail conversations.
Note: Direct 1-to-1 member messaging on LinkedIn requires Enterprise Partner Program approval.`,
}

var messageListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List recent direct messages / conversation threads",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(output.TitleStyle.Render(" LinkedIn Direct Messaging & DMs "))
		fmt.Println(output.DimStyle.Render("Capability Audit: Member-to-Member Messaging (1-to-1 DMs)"))
		fmt.Println()

		Formatter.PrintKeyValue("Status", output.WarningBadge.Render("✗ Restricted / Not Available"))
		Formatter.PrintKeyValue("Access Tier Required", "Enterprise / LinkedIn Talent Solutions Partner Program")
		Formatter.PrintKeyValue("Required Scopes", "r_messages, w_messages, r_compliance_messages")
		Formatter.PrintKeyValue("Active App Tier", "Self-Service / Standard (openid, profile, email, w_member_social)")

		fmt.Println()
		fmt.Println(output.HeaderStyle.Render("Why LinkedIn Restricts DMs via API"))
		fmt.Println("  LinkedIn does not provide self-service REST API access to private DMs")
		fmt.Println("  in order to protect user inboxes from automated spam bots.")
		fmt.Println("  Direct 1-to-1 messaging is restricted to approved Enterprise CRM and")
		fmt.Println("  Recruiter partners under strict compliance oversight.")
		fmt.Println()
		fmt.Println(output.DimStyle.Render("  Official Docs: https://learn.microsoft.com/en-us/linkedin/talent/integrations/recruiter/1-1-messaging"))
		fmt.Println()
		return nil
	},
}

func init() {
	messageCmd.AddCommand(messageListCmd)
	RootCmd.AddCommand(messageCmd)
}
