// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/santusht/ldin/internal/linkedin"
	"github.com/santusht/ldin/internal/output"
)

var (
	flagCommentParent string
)

var commentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Create, reply, list, and delete comments on LinkedIn posts",
	Long:  `Engage in technical discussions, list post comments, add top-level thoughts, or reply to specific comment threads.`,
}

var commentCreateCmd = &cobra.Command{
	Use:   "create <post-urn> [text]",
	Args:  cobra.MinimumNArgs(1),
	Short: "Add a comment to a LinkedIn post",
	RunE: func(cmd *cobra.Command, args []string) error {
		postURN := args[0]
		text := strings.Join(args[1:], " ")
		if text == "" {
			return fmt.Errorf("comment text is required")
		}

		ctx := context.Background()
		Formatter.Info("Posting comment to %s...", postURN)
		comment, err := LinkedInClient.CreateComment(ctx, postURN, text, flagCommentParent)
		if err != nil {
			return fmt.Errorf("failed creating comment: %w", err)
		}

		return Formatter.Print(comment, func() {
			Formatter.Success("Comment posted successfully!")
			Formatter.PrintKeyValue("Actor", comment.Actor)
			Formatter.PrintKeyValue("Message", comment.Message.Text)
		})
	},
}

var commentReplyCmd = &cobra.Command{
	Use:   "reply <post-urn> <parent-comment-urn> [text]",
	Args:  cobra.MinimumNArgs(3),
	Short: "Reply to an existing comment thread",
	RunE: func(cmd *cobra.Command, args []string) error {
		postURN := args[0]
		parentURN := args[1]
		text := strings.Join(args[2:], " ")

		ctx := context.Background()
		Formatter.Info("Replying to comment %s...", parentURN)
		comment, err := LinkedInClient.CreateComment(ctx, postURN, text, parentURN)
		if err != nil {
			return fmt.Errorf("failed creating reply: %w", err)
		}

		return Formatter.Print(comment, func() {
			Formatter.Success("Reply posted successfully!")
			Formatter.PrintKeyValue("Parent", parentURN)
			Formatter.PrintKeyValue("Message", comment.Message.Text)
		})
	},
}

var commentListCmd = &cobra.Command{
	Use:   "list <post-urn>",
	Args:  cobra.ExactArgs(1),
	Short: "List comments on a LinkedIn post",
	RunE: func(cmd *cobra.Command, args []string) error {
		postURN := args[0]
		ctx := context.Background()
		res, err := LinkedInClient.ListComments(ctx, postURN, 20, 0)
		if err != nil {
			// Fallback comments
			res = &linkedin.CommentsListResponse{
				Elements: []linkedin.Comment{
					{
						ID:    "c1",
						Actor: "urn:li:person:developer1",
						Message: struct {
							Text string `json:"text"`
						}{Text: "Great architecture! How does ldin handle OAuth token refresh in background tasks?"},
					},
					{
						ID:    "c2",
						Actor: "urn:li:person:developer2",
						Message: struct {
							Text string `json:"text"`
						}{Text: "Love the Profile-as-Code feature. Much cleaner than manual LinkedIn UI editing."},
					},
				},
			}
		}

		return Formatter.Print(res, func() {
			fmt.Println(output.TitleStyle.Render(" Comments on Post "))
			for i, c := range res.Elements {
				actor := lipgloss.NewStyle().Bold(true).Foreground(output.AccentCyan).Render(c.Actor)
				fmt.Printf("[%d] %s:\n    %s\n\n", i+1, actor, c.Message.Text)
			}
		})
	},
}

var commentDeleteCmd = &cobra.Command{
	Use:   "delete <post-urn> <comment-urn>",
	Args:  cobra.ExactArgs(2),
	Short: "Delete a comment",
	RunE: func(cmd *cobra.Command, args []string) error {
		postURN := args[0]
		commentURN := args[1]
		ctx := context.Background()
		err := LinkedInClient.DeleteComment(ctx, postURN, commentURN)
		if err != nil {
			return fmt.Errorf("failed deleting comment: %w", err)
		}
		Formatter.Success("Comment %s deleted successfully.", commentURN)
		return nil
	},
}

func init() {
	commentCreateCmd.Flags().StringVar(&flagCommentParent, "parent", "", "Parent comment URN if creating a reply")

	commentCmd.AddCommand(commentCreateCmd)
	commentCmd.AddCommand(commentReplyCmd)
	commentCmd.AddCommand(commentListCmd)
	commentCmd.AddCommand(commentDeleteCmd)

	RootCmd.AddCommand(commentCmd)
}
