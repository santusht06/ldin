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
	flagMediaType string
)

var mediaCmd = &cobra.Command{
	Use:   "media",
	Short: "Upload and manage media assets (images, videos, documents)",
	Long:  `Upload digital assets directly to LinkedIn REST media stores and receive Asset URNs for attaching to posts.`,
}

var mediaUploadCmd = &cobra.Command{
	Use:   "upload <file-path>",
	Args:  cobra.ExactArgs(1),
	Short: "Upload an image, video, or PDF document to LinkedIn",
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		mType := linkedin.MediaType(flagMediaType)

		ctx := context.Background()
		Formatter.Info("Uploading %s to LinkedIn Media Store...", filePath)
		res, err := LinkedInClient.UploadMedia(ctx, filePath, mType)
		if err != nil {
			return fmt.Errorf("media upload failed: %w", err)
		}

		return Formatter.Print(res, func() {
			Formatter.Success("Media uploaded successfully!")
			Formatter.PrintKeyValue("Asset URN", res.URN)
			Formatter.PrintKeyValue("Type", string(res.Type))
			Formatter.PrintKeyValue("File", res.Filename)
			Formatter.PrintKeyValue("Size (Bytes)", fmt.Sprintf("%d", res.SizeBytes))
		})
	},
}

func init() {
	mediaUploadCmd.Flags().StringVarP(&flagMediaType, "type", "t", "", "Media type: image, video, document")

	mediaCmd.AddCommand(mediaUploadCmd)
	RootCmd.AddCommand(mediaCmd)
}
