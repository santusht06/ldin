// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	flagAPIBody    string
	flagAPIHeader  []string
	flagAPIMethod  string
	flagAPIVersion string
)

var apiCmd = &cobra.Command{
	Use:   "api [METHOD] <PATH>",
	Short: "Raw LinkedIn REST API client escape hatch",
	Long: `Execute raw authenticated HTTP requests directly against LinkedIn REST endpoints.
Allows instant access to new, preview, or unmapped LinkedIn API capabilities with automatic
authorization, rate limit handling, and version headers.

Examples:
  ldin api GET /rest/posts/urn%3Ali%3Ashare%3A123
  ldin api GET /v2/userinfo
  ldin api POST /rest/posts --body @post.json
  ldin api POST /rest/posts --body '{"author":"urn:li:person:123","commentary":"Hello"}'
  ldin api DELETE /rest/posts/urn%3Ali%3Ashare%3A123`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		method := "GET"
		endpoint := ""

		if len(args) == 1 {
			endpoint = args[0]
			if flagAPIMethod != "" {
				method = strings.ToUpper(flagAPIMethod)
			}
		} else {
			method = strings.ToUpper(args[0])
			endpoint = args[1]
		}

		var bodyData interface{}
		if flagAPIBody != "" {
			if strings.HasPrefix(flagAPIBody, "@") {
				filePath := flagAPIBody[1:]
				data, err := os.ReadFile(filePath)
				if err != nil {
					return fmt.Errorf("failed reading body file %s: %w", filePath, err)
				}
				bodyData = data
			} else {
				bodyData = flagAPIBody
			}
		}

		headers := make(map[string]string)
		for _, h := range flagAPIHeader {
			parts := strings.SplitN(h, ":", 2)
			if len(parts) == 2 {
				headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}

		if flagAPIVersion != "" {
			headers["Linkedin-Version"] = flagAPIVersion
		}

		ctx := context.Background()
		respBytes, err := LinkedInClient.Request(ctx, method, endpoint, nil, bodyData, headers)
		if err != nil {
			return err
		}

		// Pretty print JSON response if valid JSON
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, respBytes, "", "  "); err == nil {
			fmt.Println(prettyJSON.String())
			return nil
		}

		fmt.Println(string(respBytes))
		return nil
	},
}

func init() {
	apiCmd.Flags().StringVarP(&flagAPIBody, "body", "b", "", "Request body JSON string or @file path")
	apiCmd.Flags().StringSliceVarP(&flagAPIHeader, "header", "H", nil, "Custom HTTP headers (e.g. -H 'Key: Value')")
	apiCmd.Flags().StringVarP(&flagAPIMethod, "method", "X", "", "HTTP method (GET, POST, PUT, PATCH, DELETE)")
	apiCmd.Flags().StringVar(&flagAPIVersion, "version", "", "Custom Linkedin-Version header override")

	RootCmd.AddCommand(apiCmd)
}
