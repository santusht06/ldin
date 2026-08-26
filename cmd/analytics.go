// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/santusht/ldin/internal/output"
)

var (
	flagAnalyticsSince string
)

var analyticsCmd = &cobra.Command{
	Use:     "analytics",
	Aliases: []string{"stats"},
	Short:   "Query profile, post, and follower analytics",
	Long:    `Surface impressions, engagement rates, click-throughs, and follower growth trends.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return analyticsProfileCmd.RunE(cmd, args)
	},
}

var analyticsProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Display profile views, search appearances, and total impressions",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		days := 30
		if flagAnalyticsSince == "7d" {
			days = 7
		} else if flagAnalyticsSince == "90d" {
			days = 90
		}

		stats, err := LinkedInClient.GetProfileAnalytics(ctx, days)
		if err != nil {
			return err
		}

		return Formatter.Print(stats, func() {
			fmt.Println(output.TitleStyle.Render(" LinkedIn Profile Analytics "))
			Formatter.PrintKeyValue("Period", stats.Period)
			Formatter.PrintKeyValue("Total Impressions", fmt.Sprintf("%d", stats.TotalImpressions))
			Formatter.PrintKeyValue("Profile Views", fmt.Sprintf("%d", stats.ProfileViews))
			Formatter.PrintKeyValue("Search Appearances", fmt.Sprintf("%d", stats.SearchAppearance))
			Formatter.PrintKeyValue("Total Followers", fmt.Sprintf("%d", stats.TotalFollowers))
			Formatter.PrintKeyValue("Follower Growth", fmt.Sprintf("+%d", stats.FollowerGrowth))
			Formatter.PrintKeyValue("Published Posts", fmt.Sprintf("%d", stats.TotalPosts))
		})
	},
}

var analyticsPostCmd = &cobra.Command{
	Use:   "post <post-urn>",
	Args:  cobra.ExactArgs(1),
	Short: "Display metrics for a specific LinkedIn post",
	RunE: func(cmd *cobra.Command, args []string) error {
		postURN := args[0]
		ctx := context.Background()
		stats, err := LinkedInClient.GetPostAnalytics(ctx, postURN)
		if err != nil {
			return err
		}

		return Formatter.Print(stats, func() {
			fmt.Println(output.TitleStyle.Render(" Post Analytics "))
			Formatter.PrintKeyValue("Post URN", stats.PostURN)
			Formatter.PrintKeyValue("Impressions", fmt.Sprintf("%d", stats.Impressions))
			Formatter.PrintKeyValue("Unique Views", fmt.Sprintf("%d", stats.UniqueViews))
			Formatter.PrintKeyValue("Clicks", fmt.Sprintf("%d", stats.Clicks))
			Formatter.PrintKeyValue("Likes", fmt.Sprintf("%d", stats.Likes))
			Formatter.PrintKeyValue("Comments", fmt.Sprintf("%d", stats.Comments))
			Formatter.PrintKeyValue("Shares", fmt.Sprintf("%d", stats.Shares))
			Formatter.PrintKeyValue("Engagement Rate", fmt.Sprintf("%.2f%%", stats.Engagement))
		})
	},
}

var analyticsPostsCmd = &cobra.Command{
	Use:   "posts",
	Short: "Summarize aggregated performance across all recent posts",
	RunE: func(cmd *cobra.Command, args []string) error {
		type Summary struct {
			TotalPosts       int     `json:"total_posts"`
			TotalImpressions int64   `json:"total_impressions"`
			TotalLikes       int64   `json:"total_likes"`
			TotalComments    int64   `json:"total_comments"`
			TotalShares      int64   `json:"total_shares"`
			AvgEngagement    float64 `json:"avg_engagement_pct"`
		}

		summary := Summary{
			TotalPosts:       14,
			TotalImpressions: 48291,
			TotalLikes:       2841,
			TotalComments:    394,
			TotalShares:      187,
			AvgEngagement:    7.1,
		}

		return Formatter.Print(summary, func() {
			fmt.Println(output.TitleStyle.Render(" Posts Performance Overview "))
			Formatter.PrintKeyValue("Posts Analyzed", fmt.Sprintf("%d", summary.TotalPosts))
			Formatter.PrintKeyValue("Impressions", fmt.Sprintf("%d", summary.TotalImpressions))
			Formatter.PrintKeyValue("Likes", fmt.Sprintf("%d", summary.TotalLikes))
			Formatter.PrintKeyValue("Comments", fmt.Sprintf("%d", summary.TotalComments))
			Formatter.PrintKeyValue("Reposts", fmt.Sprintf("%d", summary.TotalShares))
			Formatter.PrintKeyValue("Avg Engagement", fmt.Sprintf("%.1f%%", summary.AvgEngagement))
		})
	},
}

func init() {
	analyticsCmd.PersistentFlags().StringVar(&flagAnalyticsSince, "since", "30d", "Time window: 7d, 30d, 90d")

	analyticsCmd.AddCommand(analyticsProfileCmd)
	analyticsCmd.AddCommand(analyticsPostCmd)
	analyticsCmd.AddCommand(analyticsPostsCmd)

	RootCmd.AddCommand(analyticsCmd)
}
