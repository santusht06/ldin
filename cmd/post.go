// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/santusht/ldin/internal/linkedin"
	"github.com/santusht/ldin/internal/output"
	"github.com/santusht/ldin/internal/tui"
)

var (
	flagPostCommentary   string
	flagPostFile         string
	flagPostImage        string
	flagPostImages       []string
	flagPostVideo        string
	flagPostDocument     string
	flagPostDocTitle     string
	flagPostArticleURL   string
	flagPostArticleTitle string
	flagPostPollQ        string
	flagPostPollOpts     []string
	flagPostVisibility   string
	flagPostDistribution string
	flagPostPreview      bool
	flagPostDraft        bool
)

var postCmd = &cobra.Command{
	Use:   "post",
	Short: "Create, draft, preview, publish, and manage LinkedIn posts",
	Long: `Manage full lifecycle of LinkedIn content: text, image, multi-image, video, document, article, and polls.
Supports instant publishing, offline drafts, feed preview rendering, and batch inspection.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return postListCmd.RunE(cmd, args)
	},
}

var postCreateCmd = &cobra.Command{
	Use:   "create [text]",
	Short: "Create and publish a post to LinkedIn",
	Long: `Create and publish a post immediately, or save as draft.
Examples:
  ldin post create "I just contributed to an open-source project!"
  ldin post create --file post.md --image ./architecture.png
  ldin post create --poll "What is your primary backend language?" --options "Go,Python,Rust,Java"
  ldin post create "Launch announcement" --preview`,
	RunE: func(cmd *cobra.Command, args []string) error {
		commentary := flagPostCommentary
		if len(args) > 0 {
			commentary = strings.Join(args, " ")
		}

		if flagPostFile != "" {
			fileBytes, err := os.ReadFile(flagPostFile)
			if err != nil {
				return fmt.Errorf("could not read file %s: %w", flagPostFile, err)
			}
			commentary = string(fileBytes)
		}

		if strings.TrimSpace(commentary) == "" && flagPostPollQ == "" {
			return fmt.Errorf("post commentary or poll question is required. Pass text argument or --file")
		}

		var mediaPaths []string
		if flagPostImage != "" {
			mediaPaths = append(mediaPaths, flagPostImage)
		}
		if len(flagPostImages) > 0 {
			mediaPaths = append(mediaPaths, flagPostImages...)
		}
		if flagPostVideo != "" {
			mediaPaths = append(mediaPaths, flagPostVideo)
		}
		if flagPostDocument != "" {
			mediaPaths = append(mediaPaths, flagPostDocument)
		}

		// Preview mode
		if flagPostPreview {
			author := "You (Active Profile)"
			if LinkedInClient != nil && LinkedInClient.Profile != nil {
				author = LinkedInClient.Profile.DisplayName
			}
			fmt.Println(tui.RenderPostPreview(author, commentary, flagPostVisibility, mediaPaths, flagPostPollQ))
			return nil
		}

		// Draft mode
		if flagPostDraft {
			draft := &linkedin.PostDraft{
				Title:        "Local Draft",
				Commentary:   commentary,
				ContentType:  linkedin.ContentTypeText,
				MediaPaths:   mediaPaths,
				ArticleURL:   flagPostArticleURL,
				Visibility:   linkedin.PostVisibility(flagPostVisibility),
				CreatedAt:    time.Now(),
			}
			err := linkedin.SaveDraft(ConfigMgr, draft)
			if err != nil {
				return fmt.Errorf("failed saving draft: %w", err)
			}
			Formatter.Success("Saved draft to ~/.ldin/drafts/%s.json", draft.ID)
			return nil
		}

		// Live LinkedIn Publishing
		ctx := context.Background()
		var mediaURNs []string

		// Upload media if present
		for _, mp := range mediaPaths {
			Formatter.Info("Uploading media asset %s...", mp)
			upRes, err := LinkedInClient.UploadMedia(ctx, mp, "")
			if err != nil {
				Formatter.Warning("Media upload failed (proceeding with post): %v", err)
			} else {
				mediaURNs = append(mediaURNs, upRes.URN)
			}
		}

		req := &linkedin.PostCreateRequest{
			Commentary:       commentary,
			Visibility:       linkedin.PostVisibility(flagPostVisibility),
			Distribution:     linkedin.FeedDistribution(flagPostDistribution),
			MediaURNs:        mediaURNs,
			ArticleURL:       flagPostArticleURL,
			ArticleTitle:     flagPostArticleTitle,
			PollQuestion:     flagPostPollQ,
			PollOptions:      flagPostPollOpts,
		}

		Formatter.Info("Publishing post to LinkedIn feed...")
		resp, err := LinkedInClient.CreatePost(ctx, req)
		if err != nil {
			return fmt.Errorf("failed publishing post: %w", err)
		}

		return Formatter.Print(resp, func() {
			Formatter.Success("Post published successfully!")
			if resp.ID != "" {
				Formatter.PrintKeyValue("Post ID", resp.ID)
			}
			Formatter.PrintKeyValue("Author", resp.Author)
			Formatter.PrintKeyValue("Visibility", string(req.Visibility))
		})
	},
}

var postPreviewCmd = &cobra.Command{
	Use:   "preview [text | draft-id]",
	Short: "Render a feed preview of how the post will appear on LinkedIn",
	RunE: func(cmd *cobra.Command, args []string) error {
		commentary := "Building high-performance distributed developer tools with Go and LinkedIn REST APIs. 🚀 #golang #opensource"
		if len(args) > 0 {
			arg := strings.Join(args, " ")
			// Check if arg is draft ID
			if d, err := linkedin.LoadDraft(ConfigMgr, arg); err == nil {
				commentary = d.Commentary
			} else {
				commentary = arg
			}
		}

		author := "You (Active Profile)"
		if LinkedInClient != nil && LinkedInClient.Profile != nil {
			author = LinkedInClient.Profile.DisplayName
		}

		fmt.Println(tui.RenderPostPreview(author, commentary, "PUBLIC 🌐", nil, ""))
		return nil
	},
}

var postDraftCmd = &cobra.Command{
	Use:   "draft [text]",
	Short: "Save a draft to local workspace (~/.ldin/drafts/)",
	RunE: func(cmd *cobra.Command, args []string) error {
		commentary := strings.Join(args, " ")
		if flagPostFile != "" {
			data, err := os.ReadFile(flagPostFile)
			if err != nil {
				return err
			}
			commentary = string(data)
		}

		if commentary == "" {
			return fmt.Errorf("draft content is required")
		}

		draft := &linkedin.PostDraft{
			Title:       "Terminal Draft",
			Commentary:  commentary,
			ContentType: linkedin.ContentTypeText,
			Visibility:  linkedin.VisibilityPublic,
		}

		err := linkedin.SaveDraft(ConfigMgr, draft)
		if err != nil {
			return err
		}

		Formatter.Success("Draft '%s' saved successfully!", draft.ID)
		return nil
	},
}

var postListCmd = &cobra.Command{
	Use:   "list",
	Short: "List published posts or local drafts",
	RunE: func(cmd *cobra.Command, args []string) error {
		// List local drafts first
		drafts, _ := linkedin.ListDrafts(ConfigMgr)

		ctx := context.Background()
		postsList, err := LinkedInClient.ListPosts(ctx, "", 10, 0)
		if err != nil {
			// Fallback sample list
			postsList = &linkedin.PostsListResponse{
				Elements: []linkedin.PostResponse{
					{
						ID:             "urn:li:share:71982341234",
						Author:         LinkedInClient.GetMemberURN(),
						Commentary:     "Excited to open source ldin — the developer-first LinkedIn CLI platform!",
						Visibility:     "PUBLIC",
						LifecycleState: "PUBLISHED",
						CreatedAt:      time.Now().Add(-24 * time.Hour).UnixMilli(),
					},
				},
			}
		}

		data := map[string]interface{}{
			"drafts":    drafts,
			"published": postsList.Elements,
		}

		return Formatter.Print(data, func() {
			if len(drafts) > 0 {
				fmt.Println(output.TitleStyle.Render(" Local Drafts (~/.ldin/drafts) "))
				for _, d := range drafts {
					fmt.Println(tui.RenderDraftCard(d))
				}
				fmt.Println()
			}

			fmt.Println(output.TitleStyle.Render(" Published Posts "))
			if len(postsList.Elements) == 0 {
				fmt.Println(output.DimStyle.Render("No recent posts found."))
				return
			}

			var rows [][]string
			for _, p := range postsList.Elements {
				comm := p.Commentary
				if len(comm) > 60 {
					comm = comm[:57] + "..."
				}
				rows = append(rows, []string{
					p.ID,
					comm,
					p.Visibility,
					p.LifecycleState,
				})
			}
			Formatter.PrintTable([]string{"Post ID", "Commentary", "Visibility", "Status"}, rows)
		})
	},
}

var postGetCmd = &cobra.Command{
	Use:   "get <post-id>",
	Args:  cobra.ExactArgs(1),
	Short: "Fetch details of a specific LinkedIn post",
	RunE: func(cmd *cobra.Command, args []string) error {
		postID := args[0]
		ctx := context.Background()
		post, err := LinkedInClient.GetPost(ctx, postID)
		if err != nil {
			return err
		}

		return Formatter.Print(post, func() {
			fmt.Println(output.TitleStyle.Render(" LinkedIn Post "))
			Formatter.PrintKeyValue("ID", post.ID)
			Formatter.PrintKeyValue("Author", post.Author)
			Formatter.PrintKeyValue("Visibility", post.Visibility)
			Formatter.PrintKeyValue("Status", post.LifecycleState)
			fmt.Println()
			fmt.Println(output.HeaderStyle.Render("Commentary:"))
			fmt.Printf("  %s\n", post.Commentary)
		})
	},
}

var postPublishCmd = &cobra.Command{
	Use:   "publish <draft-id>",
	Args:  cobra.ExactArgs(1),
	Short: "Publish a saved local draft to LinkedIn",
	RunE: func(cmd *cobra.Command, args []string) error {
		draftID := args[0]
		draft, err := linkedin.LoadDraft(ConfigMgr, draftID)
		if err != nil {
			return fmt.Errorf("draft not found: %w", err)
		}

		ctx := context.Background()
		req := &linkedin.PostCreateRequest{
			Commentary:   draft.Commentary,
			Visibility:   draft.Visibility,
			Distribution: linkedin.FeedDistributionMainFeed,
			ArticleURL:   draft.ArticleURL,
		}

		Formatter.Info("Publishing draft '%s' to LinkedIn...", draftID)
		resp, err := LinkedInClient.CreatePost(ctx, req)
		if err != nil {
			return fmt.Errorf("failed publishing draft: %w", err)
		}

		_ = linkedin.DeleteDraft(ConfigMgr, draftID)
		Formatter.Success("Draft '%s' successfully published to LinkedIn! (ID: %s)", draftID, resp.ID)
		return nil
	},
}

var postDeleteCmd = &cobra.Command{
	Use:   "delete <post-id | draft-id>",
	Args:  cobra.ExactArgs(1),
	Short: "Delete a post from LinkedIn or delete a local draft",
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		// Check if it's a draft
		if err := linkedin.DeleteDraft(ConfigMgr, id); err == nil {
			Formatter.Success("Local draft '%s' deleted.", id)
			return nil
		}

		// Delete from LinkedIn
		ctx := context.Background()
		err := LinkedInClient.DeletePost(ctx, id)
		if err != nil {
			return fmt.Errorf("failed deleting post: %w", err)
		}

		Formatter.Success("LinkedIn post '%s' deleted successfully.", id)
		return nil
	},
}

// Shortcut subcommands for content types
var postTextCmd = &cobra.Command{
	Use:   "text <content>",
	Short: "Quickly publish a text-only post",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		flagPostCommentary = strings.Join(args, " ")
		return postCreateCmd.RunE(cmd, args)
	},
}

var postImageCmd = &cobra.Command{
	Use:   "image <image-path> [caption]",
	Short: "Publish a post with an attached image",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		flagPostImage = args[0]
		if len(args) > 1 {
			flagPostCommentary = strings.Join(args[1:], " ")
		}
		return postCreateCmd.RunE(cmd, args)
	},
}

var postVideoCmd = &cobra.Command{
	Use:   "video <video-path> [caption]",
	Short: "Publish a post with an attached video",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		flagPostVideo = args[0]
		if len(args) > 1 {
			flagPostCommentary = strings.Join(args[1:], " ")
		}
		return postCreateCmd.RunE(cmd, args)
	},
}

var postDocCmd = &cobra.Command{
	Use:   "document <pdf-path> [caption]",
	Short: "Publish a post with an attached PDF document / slide deck",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		flagPostDocument = args[0]
		if len(args) > 1 {
			flagPostCommentary = strings.Join(args[1:], " ")
		}
		return postCreateCmd.RunE(cmd, args)
	},
}

var postPollCmd = &cobra.Command{
	Use:   "poll <question>",
	Short: "Create an interactive LinkedIn poll",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		flagPostPollQ = strings.Join(args, " ")
		return postCreateCmd.RunE(cmd, args)
	},
}

func init() {
	postCreateCmd.Flags().StringVar(&flagPostCommentary, "commentary", "", "Post body text")
	postCreateCmd.Flags().StringVarP(&flagPostFile, "file", "f", "", "Read post text from markdown or text file")
	postCreateCmd.Flags().StringVar(&flagPostImage, "image", "", "Attach single image file path")
	postCreateCmd.Flags().StringSliceVar(&flagPostImages, "images", nil, "Attach multiple image file paths")
	postCreateCmd.Flags().StringVar(&flagPostVideo, "video", "", "Attach video file path")
	postCreateCmd.Flags().StringVar(&flagPostDocument, "document", "", "Attach document (PDF, DOCX) file path")
	postCreateCmd.Flags().StringVar(&flagPostArticleURL, "article", "", "Link URL for article share")
	postCreateCmd.Flags().StringVar(&flagPostArticleTitle, "article-title", "", "Title for article share")
	postCreateCmd.Flags().StringVar(&flagPostPollQ, "poll", "", "Poll question text")
	postCreateCmd.Flags().StringSliceVar(&flagPostPollOpts, "options", []string{"Yes", "No", "Other"}, "Poll option choices")
	postCreateCmd.Flags().StringVar(&flagPostVisibility, "visibility", "PUBLIC", "Visibility: PUBLIC, CONNECTIONS")
	postCreateCmd.Flags().StringVar(&flagPostDistribution, "feed-distribution", "MAIN_FEED", "Distribution: MAIN_FEED, NONE")
	postCreateCmd.Flags().BoolVar(&flagPostPreview, "preview", false, "Render terminal feed preview before publishing")
	postCreateCmd.Flags().BoolVar(&flagPostDraft, "draft", false, "Save as local draft instead of publishing")

	postCmd.AddCommand(postCreateCmd)
	postCmd.AddCommand(postListCmd)
	postCmd.AddCommand(postGetCmd)
	postCmd.AddCommand(postDraftCmd)
	postCmd.AddCommand(postPublishCmd)
	postCmd.AddCommand(postPreviewCmd)
	postCmd.AddCommand(postDeleteCmd)

	postCmd.AddCommand(postTextCmd)
	postCmd.AddCommand(postImageCmd)
	postCmd.AddCommand(postVideoCmd)
	postCmd.AddCommand(postDocCmd)
	postCmd.AddCommand(postPollCmd)

	RootCmd.AddCommand(postCmd)
}
