// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

const (
	Version   = "1.0.0"
	BuildDate = "2026-08-26"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of ldin",
	RunE: func(cmd *cobra.Command, args []string) error {
		data := map[string]string{
			"version":    Version,
			"build_date": BuildDate,
			"go_version": runtime.Version(),
			"platform":   fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		}

		return Formatter.Print(data, func() {
			fmt.Fprintf(Formatter.Writer, "ldin version %s (%s) %s/%s\n", Version, BuildDate, runtime.GOOS, runtime.GOARCH)
		})
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
