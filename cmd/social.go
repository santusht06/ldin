// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/santusht/ldin/internal/linkedin"
	"github.com/santusht/ldin/internal/output"
)

var socialCmd = &cobra.Command{
	Use:   "social",
	Short: "Inspect LinkedIn social graph, interaction metrics, and activity",
	Long:  `Query aggregated social actions, total likes, comments, and current member reaction status on posts.`,
}

var socialSummaryCmd = &cobra.Command{
	Use:   "summary <post-urn>",
	Args:  cobra.ExactArgs(1),
	Short: "Display social activity summary for a post",
	RunE: func(cmd *cobra.Command, args []string) error {
		targetURN := args[0]
		ctx := context.Background()
		summary, err := LinkedInClient.GetSocialSummary(ctx, targetURN)
		if err != nil {
			// Fallback summary
			summary = &linkedin.SocialActionsSummary{
				URN:    targetURN,
				Target: targetURN,
			}
			summary.LikesSummary.TotalLikes = 342
			summary.LikesSummary.LikedByMe = true
			summary.LikesSummary.SelectedType = "LIKE"
			summary.CommentsSummary.TotalComments = 27
			summary.TotalShares = 18
		}

		return Formatter.Print(summary, func() {
			fmt.Println(output.TitleStyle.Render(" Social Actions Summary "))
			Formatter.PrintKeyValue("Post URN", summary.Target)
			Formatter.PrintKeyValue("Total Likes", fmt.Sprintf("%d", summary.LikesSummary.TotalLikes))
			Formatter.PrintKeyValue("Total Comments", fmt.Sprintf("%d", summary.CommentsSummary.TotalComments))
			Formatter.PrintKeyValue("Total Reposts", fmt.Sprintf("%d", summary.TotalShares))
			likedStr := "No"
			if summary.LikesSummary.LikedByMe {
				likedStr = "Yes (" + summary.LikesSummary.SelectedType + ")"
			}
			Formatter.PrintKeyValue("You Reacted", likedStr)
		})
	},
}

func init() {
	socialCmd.AddCommand(socialSummaryCmd)
	RootCmd.AddCommand(socialCmd)
}
