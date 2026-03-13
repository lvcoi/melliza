package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// SetupSpinnerStep represents a step in the project setup process.
type SetupSpinnerStep int

const (
	SetupStepScan     SetupSpinnerStep = iota // Scanning project for tech stack
	SetupStepGenerate                         // Generating GEMINI.md
	SetupStepDone                             // Setup complete
)

// SetupSpinner manages the project setup spinner overlay state.
type SetupSpinner struct {
	width  int
	height int

	prdName      string
	currentStep  SetupSpinnerStep
	spinnerFrame int
	steps        []stepInfo
	errMsg       string
}

// NewSetupSpinner creates a new setup spinner.
func NewSetupSpinner() *SetupSpinner {
	return &SetupSpinner{}
}

// Configure sets up the spinner with the given PRD name.
func (s *SetupSpinner) Configure(prdName string) {
	s.prdName = prdName
	s.currentStep = SetupStepScan
	s.spinnerFrame = 0
	s.errMsg = ""

	s.steps = []stepInfo{
		{label: "Scanning project tech stack"},
		{label: "Generating GEMINI.md"},
	}

	if len(s.steps) > 0 {
		s.steps[0].active = true
	}
}

// SetSize sets the spinner dimensions.
func (s *SetupSpinner) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// AdvanceStep marks the current step as complete and moves to the next.
func (s *SetupSpinner) AdvanceStep() {
	idx := int(s.currentStep)
	if idx < len(s.steps) {
		s.steps[idx].complete = true
		s.steps[idx].active = false
	}

	s.currentStep++

	nextIdx := int(s.currentStep)
	if nextIdx < len(s.steps) {
		s.steps[nextIdx].active = true
	}
}

// SetError sets an error on the current step.
func (s *SetupSpinner) SetError(err string) {
	s.errMsg = err
	idx := int(s.currentStep)
	if idx < len(s.steps) {
		s.steps[idx].errMsg = err
		s.steps[idx].active = false
	}
}

// HasError returns true if there is an error.
func (s *SetupSpinner) HasError() bool {
	return s.errMsg != ""
}

// IsDone returns true if all steps are complete.
func (s *SetupSpinner) IsDone() bool {
	return s.currentStep >= SetupStepDone
}

// Tick advances the spinner animation frame.
func (s *SetupSpinner) Tick() {
	s.spinnerFrame++
}

// Render renders the setup spinner overlay.
func (s *SetupSpinner) Render() string {
	modalWidth := min(60, s.width-10)
	if modalWidth < 40 {
		modalWidth = 40
	}

	var content strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor)
	content.WriteString(titleStyle.Render(fmt.Sprintf("Setting up %s", s.prdName)))
	content.WriteString("\n")
	content.WriteString(DividerStyle.Render(strings.Repeat("─", modalWidth-4)))
	content.WriteString("\n\n")

	spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinnerStyle := lipgloss.NewStyle().Foreground(PrimaryColor)
	checkStyle := lipgloss.NewStyle().Foreground(SuccessColor)
	errorStyle := lipgloss.NewStyle().Foreground(ErrorColor)
	textStyle := lipgloss.NewStyle().Foreground(TextColor)
	mutedStyle := lipgloss.NewStyle().Foreground(MutedColor)

	for _, step := range s.steps {
		if step.complete {
			content.WriteString(checkStyle.Render("✓"))
			content.WriteString(" ")
			completedLabel := strings.Replace(step.label, "Scanning", "Scanned", 1)
			completedLabel = strings.Replace(completedLabel, "Generating", "Generated", 1)
			content.WriteString(textStyle.Render(completedLabel))
		} else if step.errMsg != "" {
			content.WriteString(errorStyle.Render("✗"))
			content.WriteString(" ")
			content.WriteString(errorStyle.Render(step.label))
			content.WriteString("\n")
			content.WriteString("  ")
			content.WriteString(errorStyle.Render(step.errMsg))
		} else if step.active {
			frame := spinnerFrames[s.spinnerFrame%len(spinnerFrames)]
			content.WriteString(spinnerStyle.Render(frame))
			content.WriteString(" ")
			content.WriteString(textStyle.Render(step.label))
		} else {
			content.WriteString(mutedStyle.Render("○"))
			content.WriteString(" ")
			content.WriteString(mutedStyle.Render(step.label))
		}
		content.WriteString("\n")
	}

	if s.IsDone() {
		content.WriteString("\n")
		content.WriteString(checkStyle.Render("Starting loop..."))
	}

	content.WriteString("\n")
	content.WriteString(DividerStyle.Render(strings.Repeat("─", modalWidth-4)))

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Padding(1, 2).
		Width(modalWidth)

	modal := modalStyle.Render(content.String())
	return CenterModal(modal, s.width, s.height)
}
