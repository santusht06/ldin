// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/santusht/ldin/internal/linkedin"
)

// RenderPostPreview creates a rich graphical LinkedIn feed card for the terminal
func RenderPostPreview(author, commentary string, visibility string, mediaPaths []string, pollQuestion string) string {
	cardBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#0A66C2")).
		Padding(1, 2).
		Width(72)

	authorStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF"))

	subStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8B949E")).
		Italic(true)

	badgeStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#1F6FEB")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1).
		Bold(true)

	if author == "" {
		author = "You (Active Profile)"
	}
	if visibility == "" {
		visibility = "PUBLIC 🌐"
	}

	charCount := len([]rune(commentary))
	charLimitStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E"))
	if charCount > 3000 {
		charLimitStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4D4D")).Bold(true)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s  %s\n", authorStyle.Render(author), badgeStyle.Render(visibility)))
	sb.WriteString(fmt.Sprintf("%s\n\n", subStyle.Render("Just now • Edited via ldin CLI")))

	// Post body commentary
	bodyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F0F6FC"))
	sb.WriteString(bodyStyle.Render(commentary))
	sb.WriteString("\n\n")

	// Media or poll attachment badges
	if len(mediaPaths) > 0 {
		mediaBadge := lipgloss.NewStyle().
			Background(lipgloss.Color("#238636")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)
		sb.WriteString(fmt.Sprintf("%s Attached: %s\n\n", mediaBadge.Render("MEDIA"), strings.Join(mediaPaths, ", ")))
	}

	if pollQuestion != "" {
		pollBadge := lipgloss.NewStyle().
			Background(lipgloss.Color("#A371F7")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)
		sb.WriteString(fmt.Sprintf("%s Poll: %s\n\n", pollBadge.Render("POLL"), pollQuestion))
	}

	divider := lipgloss.NewStyle().Foreground(lipgloss.Color("#30363D")).Render(strings.Repeat("─", 68))
	sb.WriteString(divider + "\n")

	// Mock social action bar
	actionBar := lipgloss.NewStyle().Foreground(lipgloss.Color("#58A6FF")).Render("👍 Like   💬 Comment   🔁 Repost   🚀 Send")
	sb.WriteString(actionBar + "\n")
	sb.WriteString(charLimitStyle.Render(fmt.Sprintf("\nCharacters: %d / 3,000", charCount)))

	return cardBorder.Render(sb.String())
}

// RenderDraftCard creates a clean summary card for a draft
func RenderDraftCard(draft *linkedin.PostDraft) string {
	card := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#30363D")).
		Padding(1, 2).
		MarginBottom(1).
		Width(72)

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00D2FF")).Render(draft.ID)
	if draft.Title != "" {
		title += fmt.Sprintf(" — %s", draft.Title)
	}

	previewText := draft.Commentary
	if len(previewText) > 160 {
		previewText = previewText[:160] + "..."
	}

	body := lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).Render(previewText)
	meta := lipgloss.NewStyle().Foreground(lipgloss.Color("#6E7681")).Render(fmt.Sprintf("Created: %s | Visibility: %s", draft.CreatedAt.Format("2006-01-02 15:04"), draft.Visibility))

	return card.Render(fmt.Sprintf("%s\n\n%s\n\n%s", title, body, meta))
}
