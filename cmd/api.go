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
	"github.com/santusht/ldin/internal/output"
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

var apiVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the active LinkedIn API version",
	Long: `Display the active LinkedIn-Version header sent with all REST requests.
LinkedIn uses monthly versioning (YYYYMM).

To change the default API version:
  ldin config set linkedin.version 202608`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ver := AppCfg.LinkedInAPIVersion
		if ver == "" {
			ver = "202608"
		}
		fmt.Println(output.TitleStyle.Render(" LinkedIn REST API Version "))
		Formatter.PrintKeyValue("Active API Version", ver)
		Formatter.PrintKeyValue("Header Sent", fmt.Sprintf("Linkedin-Version: %s", ver))
		Formatter.PrintKeyValue("Protocol Version", "X-Restli-Protocol-Version: 2.0.0")
		Formatter.PrintKeyValue("Base URL", "https://api.linkedin.com")
		fmt.Printf("\n  To override for a single request:  ldin api GET /rest/... --version %s\n", ver)
		fmt.Printf("  To change permanently:             ldin config set linkedin.version <YYYYMM>\n\n")
		return nil
	},
}

var apiVersionsCmd = &cobra.Command{
	Use:   "versions",
	Short: "List LinkedIn REST API versions and lifecycle status",
	Long: `LinkedIn releases new API versions monthly using the YYYYMM format.
Each API version is supported for 12 months from its release date.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(output.TitleStyle.Render(" LinkedIn API Release Schedule "))
		fmt.Println(output.DimStyle.Render("LinkedIn Versioning Policy: Monthly releases (YYYYMM), 12-month support window"))
		fmt.Println()

		rows := [][]string{
			{"202608", "Active (Latest / Default)", "2026-08", "2027-08"},
			{"202607", "Active", "2026-07", "2027-07"},
			{"202606", "Active", "2026-06", "2027-06"},
			{"202605", "Active", "2026-05", "2027-05"},
			{"202604", "Active", "2026-04", "2027-04"},
			{"202603", "Active", "2026-03", "2027-03"},
			{"202602", "Active", "2026-02", "2027-02"},
			{"202601", "Active", "2026-01", "2027-01"},
			{"202512", "Maintenance", "2025-12", "2026-12"},
			{"202511", "Maintenance", "2025-11", "2026-11"},
		}

		Formatter.PrintTable([]string{"Version (YYYYMM)", "Status", "Released", "End of Support"}, rows)
		fmt.Printf("\n  Current default: 202608 (configured in ~/.ldin/config.yaml)\n\n")
		return nil
	},
}

func init() {
	apiCmd.Flags().StringVarP(&flagAPIBody, "body", "b", "", "Request body JSON string or @file path")
	apiCmd.Flags().StringSliceVarP(&flagAPIHeader, "header", "H", nil, "Custom HTTP headers (e.g. -H 'Key: Value')")
	apiCmd.Flags().StringVarP(&flagAPIMethod, "method", "X", "", "HTTP method (GET, POST, PUT, PATCH, DELETE)")
	apiCmd.Flags().StringVar(&flagAPIVersion, "version", "", "Custom Linkedin-Version header override")

	apiCmd.AddCommand(apiVersionCmd)
	apiCmd.AddCommand(apiVersionsCmd)

	RootCmd.AddCommand(apiCmd)
}
