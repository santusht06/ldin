// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/santusht/ldin/internal/output"
)

var adsCmd = &cobra.Command{
	Use:   "ads",
	Short: "LinkedIn Marketing Developer Platform & Sponsored Campaigns",
	Long:  `Manage advertising accounts, sponsored content campaigns, and ad analytics.`,
}

var adsAccountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "List LinkedIn advertising accounts",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		accounts, err := LinkedInClient.ListAdAccounts(ctx)
		if err != nil {
			return err
		}

		return Formatter.Print(accounts, func() {
			fmt.Println(output.TitleStyle.Render(" Sponsored Ad Accounts "))
			var rows [][]string
			for _, a := range accounts {
				rows = append(rows, []string{
					a.ID,
					a.Name,
					a.Currency,
					a.Status,
				})
			}
			Formatter.PrintTable([]string{"Account ID", "Account Name", "Currency", "Status"}, rows)
		})
	},
}

var adsCampaignsCmd = &cobra.Command{
	Use:   "campaigns [account-urn]",
	Short: "List ad campaigns for an advertising account",
	RunE: func(cmd *cobra.Command, args []string) error {
		accountURN := ""
		if len(args) > 0 {
			accountURN = args[0]
		}
		ctx := context.Background()
		campaigns, err := LinkedInClient.ListCampaigns(ctx, accountURN)
		if err != nil {
			return err
		}

		return Formatter.Print(campaigns, func() {
			fmt.Println(output.TitleStyle.Render(" Sponsored Campaigns "))
			var rows [][]string
			for _, c := range campaigns {
				rows = append(rows, []string{
					c.ID,
					c.Name,
					c.Status,
					fmt.Sprintf("$%.2f", c.DailyBudget),
					fmt.Sprintf("%d", c.Impressions),
					fmt.Sprintf("%d", c.Clicks),
				})
			}
			Formatter.PrintTable([]string{"Campaign ID", "Campaign Name", "Status", "Daily Budget", "Impressions", "Clicks"}, rows)
		})
	},
}

func init() {
	adsCmd.AddCommand(adsAccountsCmd)
	adsCmd.AddCommand(adsCampaignsCmd)

	RootCmd.AddCommand(adsCmd)
}
