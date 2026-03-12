package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// CenterModal centers a rendered modal string on the screen.
func CenterModal(modal string, screenWidth, screenHeight int) string {
	lines := strings.Split(modal, "\n")
	modalHeight := len(lines)
	modalWidth := 0
	for _, line := range lines {
		if w := lipgloss.Width(line); w > modalWidth {
			modalWidth = w
		}
	}

	topPadding := (screenHeight - modalHeight) / 2
	leftPadding := (screenWidth - modalWidth) / 2

	if topPadding < 0 {
		topPadding = 0
	}
	if leftPadding < 0 {
		leftPadding = 0
	}

	var result strings.Builder

	for i := 0; i < topPadding; i++ {
		result.WriteString("\n")
	}

	leftPad := strings.Repeat(" ", leftPadding)
	for _, line := range lines {
		result.WriteString(leftPad)
		result.WriteString(line)
		result.WriteString("\n")
	}
	return result.String()
}
