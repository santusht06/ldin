// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/santusht/ldin/internal/output"
)

var eventCmd = &cobra.Command{
	Use:   "event",
	Short: "Manage LinkedIn Events (Live streams, audio events)",
	Long:  `Create and manage professional events on LinkedIn.`,
}

var eventListCmd = &cobra.Command{
	Use:   "list",
	Short: "List events you organize",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		events, err := LinkedInClient.ListEvents(ctx)
		if err != nil {
			return err
		}

		return Formatter.Print(events, func() {
			fmt.Println(output.TitleStyle.Render(" LinkedIn Events "))
			var rows [][]string
			for _, e := range events {
				rows = append(rows, []string{
					e.URN,
					e.Name,
					e.EventType,
					fmt.Sprintf("%d", e.AttendeeCnt),
				})
			}
			Formatter.PrintTable([]string{"Event URN", "Event Title", "Type", "Attendees"}, rows)
		})
	},
}

var eventCreateCmd = &cobra.Command{
	Use:   "create <name> [description]",
	Args:  cobra.MinimumNArgs(1),
	Short: "Register a new LinkedIn audio/video event",
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		desc := ""
		if len(args) > 1 {
			desc = strings.Join(args[1:], " ")
		}

		ctx := context.Background()
		Formatter.Info("Creating event '%s'...", name)
		ev, err := LinkedInClient.CreateEvent(ctx, name, desc, "ONLINE", 0, 0)
		if err != nil {
			return err
		}

		return Formatter.Print(ev, func() {
			Formatter.Success("Event created successfully!")
			Formatter.PrintKeyValue("Event URN", ev.URN)
			Formatter.PrintKeyValue("Name", ev.Name)
		})
	},
}

func init() {
	eventCmd.AddCommand(eventListCmd)
	eventCmd.AddCommand(eventCreateCmd)

	RootCmd.AddCommand(eventCmd)
}
