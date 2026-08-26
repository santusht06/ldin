// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/santusht/ldin/internal/linkedin"
)

var (
	flagReactionType string
)

var reactionCmd = &cobra.Command{
	Use:   "reaction",
	Short: "React to LinkedIn posts (Like, Celebrate, Support, Love, Insightful, Curious)",
	Long:  `Send or remove emotional reactions on LinkedIn posts and comments.`,
}

var reactionLikeCmd = &cobra.Command{
	Use:   "like <target-urn>",
	Args:  cobra.ExactArgs(1),
	Short: "Like a LinkedIn post or comment",
	RunE: func(cmd *cobra.Command, args []string) error {
		targetURN := args[0]
		ctx := context.Background()
		Formatter.Info("Liking %s...", targetURN)
		res, err := LinkedInClient.React(ctx, targetURN, linkedin.ReactionLike)
		if err != nil {
			return fmt.Errorf("failed liking post: %w", err)
		}
		return Formatter.Print(res, func() {
			Formatter.Success("Liked %s 👍", targetURN)
		})
	},
}

var reactionReactCmd = &cobra.Command{
	Use:   "react <target-urn>",
	Args:  cobra.ExactArgs(1),
	Short: "Add a specific reaction type (LIKE, CELEBRATE, SUPPORT, LOVE, INSIGHTFUL, CURIOUS)",
	RunE: func(cmd *cobra.Command, args []string) error {
		targetURN := args[0]
		rType := linkedin.NormalizeReactionType(flagReactionType)

		ctx := context.Background()
		Formatter.Info("Sending reaction %s to %s...", rType, targetURN)
		res, err := LinkedInClient.React(ctx, targetURN, rType)
		if err != nil {
			return fmt.Errorf("failed sending reaction: %w", err)
		}
		return Formatter.Print(res, func() {
			Formatter.Success("Reacted %s to %s", string(rType), targetURN)
		})
	},
}

var reactionUnlikeCmd = &cobra.Command{
	Use:   "unlike <target-urn>",
	Args:  cobra.ExactArgs(1),
	Short: "Remove reaction from a LinkedIn post or comment",
	RunE: func(cmd *cobra.Command, args []string) error {
		targetURN := args[0]
		ctx := context.Background()
		err := LinkedInClient.DeleteReaction(ctx, targetURN)
		if err != nil {
			return fmt.Errorf("failed unliking: %w", err)
		}
		Formatter.Success("Removed reaction from %s", targetURN)
		return nil
	},
}

func init() {
	reactionReactCmd.Flags().StringVarP(&flagReactionType, "type", "t", "LIKE", "Reaction type: LIKE, CELEBRATE, SUPPORT, LOVE, INSIGHTFUL, CURIOUS")

	reactionCmd.AddCommand(reactionLikeCmd)
	reactionCmd.AddCommand(reactionReactCmd)
	reactionCmd.AddCommand(reactionUnlikeCmd)

	RootCmd.AddCommand(reactionCmd)
}
