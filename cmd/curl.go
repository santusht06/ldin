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
	flagCurlData    string
	flagCurlHeaders []string
	flagCurlMethod  string
	flagCurlDryRun  bool
)

var curlCmd = &cobra.Command{
	Use:     "curl [METHOD] <PATH_OR_URL>",
	Aliases: []string{"http", "req"},
	Short:   "Execute authenticated curl-like requests directly against LinkedIn REST API",
	Long: `ldin curl is your direct terminal bridge between cURL and LinkedIn.
Executes raw REST requests with automatic Bearer token injection, LinkedIn version headers,
and JSON indentation.

Examples:
  ldin curl GET /v2/userinfo
  ldin curl /v2/userinfo
  ldin curl POST /rest/posts -d '{"author":"urn:li:person:me","commentary":"Hello from terminal!"}'
  ldin curl GET /rest/posts/urn%3Ali%3Ashare%3A71982341234
  ldin curl GET /v2/userinfo --dry-run`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		method := "GET"
		endpoint := ""

		if len(args) == 1 {
			endpoint = args[0]
			if flagCurlMethod != "" {
				method = strings.ToUpper(flagCurlMethod)
			}
		} else {
			method = strings.ToUpper(args[0])
			endpoint = args[1]
		}

		headers := make(map[string]string)
		for _, h := range flagCurlHeaders {
			parts := strings.SplitN(h, ":", 2)
			if len(parts) == 2 {
				headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}

		var body interface{}
		if flagCurlData != "" {
			if strings.HasPrefix(flagCurlData, "@") {
				fileBytes, err := os.ReadFile(flagCurlData[1:])
				if err != nil {
					return fmt.Errorf("could not read data file %s: %w", flagCurlData[1:], err)
				}
				body = fileBytes
			} else {
				body = flagCurlData
			}
		}

		fullURL := endpoint
		if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			if !strings.HasPrefix(endpoint, "/") {
				endpoint = "/" + endpoint
			}
			fullURL = LinkedInClient.BaseURL + endpoint
		}

		if flagCurlDryRun {
			curlStr := LinkedInClient.BuildCurl(method, fullURL, body, headers)
			fmt.Println(curlStr)
			return nil
		}

		ctx := context.Background()
		respBytes, err := LinkedInClient.Request(ctx, method, endpoint, nil, body, headers)
		if err != nil {
			return err
		}

		// Pretty print JSON response
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, respBytes, "", "  "); err == nil {
			fmt.Println(prettyJSON.String())
			return nil
		}

		fmt.Println(string(respBytes))
		return nil
	},
}

var importCurlCmd = &cobra.Command{
	Use:   "import-curl <curl_command_string>",
	Short: "Import and execute any raw cURL command using active ldin tokens",
	Long: `Paste any raw cURL command (from documentation or DevTools).
ldin automatically injects active Bearer tokens and executes the request.

Example:
  ldin import-curl "curl 'https://api.linkedin.com/v2/userinfo' -H 'Authorization: Bearer ...'"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rawCurl := strings.Join(args, " ")

		// Parse method
		method := "GET"
		if strings.Contains(rawCurl, "-X POST") || strings.Contains(rawCurl, "--request POST") {
			method = "POST"
		} else if strings.Contains(rawCurl, "-X DELETE") || strings.Contains(rawCurl, "--request DELETE") {
			method = "DELETE"
		} else if strings.Contains(rawCurl, "-X PUT") || strings.Contains(rawCurl, "--request PUT") {
			method = "PUT"
		} else if strings.Contains(rawCurl, "-d ") || strings.Contains(rawCurl, "--data ") {
			method = "POST"
		}

		// Extract URL
		tokens := strings.Fields(rawCurl)
		targetURL := ""
		for _, t := range tokens {
			clean := strings.Trim(t, `"'`)
			if strings.HasPrefix(clean, "http://") || strings.HasPrefix(clean, "https://") {
				targetURL = clean
				break
			}
		}

		if targetURL == "" {
			return fmt.Errorf("could not extract URL from curl command")
		}

		Formatter.Info("Executing bridged cURL %s %s...", method, targetURL)
		ctx := context.Background()
		respBytes, err := LinkedInClient.Request(ctx, method, targetURL, nil, nil, nil)
		if err != nil {
			return err
		}

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
	curlCmd.Flags().StringVarP(&flagCurlData, "data", "d", "", "HTTP POST/PUT data string or @filename")
	curlCmd.Flags().StringSliceVarP(&flagCurlHeaders, "header", "H", nil, "Custom HTTP header (Key: Value)")
	curlCmd.Flags().StringVarP(&flagCurlMethod, "request", "X", "", "HTTP method (GET, POST, PUT, DELETE, PATCH)")
	curlCmd.Flags().BoolVar(&flagCurlDryRun, "dry-run", false, "Print equivalent curl command without executing")

	RootCmd.AddCommand(curlCmd)
	RootCmd.AddCommand(importCurlCmd)
}
