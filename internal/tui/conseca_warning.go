package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// ConsecaWarningOption represents the user's choice in the Conseca warning dialog.
type ConsecaWarningOption int

const (
	ConsecaOptionContinue ConsecaWarningOption = iota
	ConsecaOptionQuit
)

// ConsecaWarning manages the Conseca safety warning dialog state.
type ConsecaWarning struct {
	width         int
	height        int
	selectedIndex int
}

// NewConsecaWarning creates a new Conseca warning dialog.
func NewConsecaWarning() *ConsecaWarning {
	return &ConsecaWarning{
		selectedIndex: 0,
	}
}

// SetSize sets the dialog dimensions.
func (c *ConsecaWarning) SetSize(width, height int) {
	c.width = width
	c.height = height
}

// MoveUp moves selection up.
func (c *ConsecaWarning) MoveUp() {
	if c.selectedIndex > 0 {
		c.selectedIndex--
	}
}

// MoveDown moves selection down.
func (c *ConsecaWarning) MoveDown() {
	if c.selectedIndex < 1 {
		c.selectedIndex++
	}
}

// GetSelectedOption returns the currently selected option.
func (c *ConsecaWarning) GetSelectedOption() ConsecaWarningOption {
	if c.selectedIndex == 0 {
		return ConsecaOptionContinue
	}
	return ConsecaOptionQuit
}

// Reset resets the dialog to its default state.
func (c *ConsecaWarning) Reset() {
	c.selectedIndex = 0
}

// Render renders the Conseca warning dialog.
func (c *ConsecaWarning) Render() string {
	modalWidth := min(65, c.width-10)
	if modalWidth < 40 {
		modalWidth = 40
	}

	contentWidth := modalWidth - 4

	var content strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(WarningColor)
	content.WriteString(titleStyle.Render("Gemini Conseca Safety Checker"))
	content.WriteString("\n")
	content.WriteString(DividerStyle.Render(strings.Repeat("─", contentWidth)))
	content.WriteString("\n\n")

	// Warning message
	messageStyle := lipgloss.NewStyle().Foreground(TextColor)
	content.WriteString(messageStyle.Render("Conseca is enabled in your Gemini settings."))
	content.WriteString("\n")
	content.WriteString(messageStyle.Render("It may block Gemini from writing files,"))
	content.WriteString("\n")
	content.WriteString(messageStyle.Render("which will prevent PRD creation and"))
	content.WriteString("\n")
	content.WriteString(messageStyle.Render("story implementation from working."))
	content.WriteString("\n\n")

	// Fix hint
	hintStyle := lipgloss.NewStyle().Foreground(MutedColor)
	content.WriteString(hintStyle.Render("To fix: set enableConseca to false in"))
	content.WriteString("\n")
	content.WriteString(hintStyle.Render("~/.gemini/settings.json"))
	content.WriteString("\n\n")

	// Options
	options := []struct {
		label string
		hint  string
	}{
		{"Continue anyway", "Gemini may fail to write files"},
		{"Quit", "Fix settings first"},
	}

	selectedStyle := lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true)
	optionStyle := lipgloss.NewStyle().Foreground(TextColor)
	optHintStyle := lipgloss.NewStyle().Foreground(MutedColor)

	for i, opt := range options {
		if i == c.selectedIndex {
			content.WriteString(selectedStyle.Render(fmt.Sprintf("▶ %s", opt.label)))
		} else {
			content.WriteString(optionStyle.Render(fmt.Sprintf("  %s", opt.label)))
		}
		content.WriteString("\n")
		content.WriteString(optHintStyle.Render(fmt.Sprintf("    %s", opt.hint)))
		content.WriteString("\n")
	}

	// Footer
	content.WriteString("\n")
	content.WriteString(DividerStyle.Render(strings.Repeat("─", contentWidth)))
	content.WriteString("\n")
	footerStyle := lipgloss.NewStyle().Foreground(MutedColor)
	content.WriteString(footerStyle.Render("↑/↓: Navigate  Enter: Select"))

	// Modal box
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(WarningColor).
		Padding(1, 2).
		Width(modalWidth)

	modal := modalStyle.Render(content.String())
	return CenterModal(modal, c.width, c.height)
}
