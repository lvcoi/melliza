package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	projectctx "github.com/lvcoi/melliza/internal/context"
)

// WizardStep represents a step in the setup wizard.
type WizardStep int

const (
	WizardStepBuilding  WizardStep = iota // "What are you building?"
	WizardStepProblem                     // "What problem does this solve?"
	WizardStepUsers                       // "Who does it solve it for?"
	WizardStepVision                      // "What's your vision?"
	WizardStepTechStack                   // Tech stack
	wizardStepCount                       // sentinel
)

func (s WizardStep) String() string {
	switch s {
	case WizardStepBuilding:
		return "Building"
	case WizardStepProblem:
		return "Problem"
	case WizardStepUsers:
		return "Users"
	case WizardStepVision:
		return "Vision"
	case WizardStepTechStack:
		return "TechStack"
	default:
		return "Unknown"
	}
}

// wizardCard holds config and input state for one wizard card.
type wizardCard struct {
	title       string
	prompt      string
	placeholder string
	input       textarea.Model
}

// WizardActivityItem represents a line in the left-panel activity feed.
type WizardActivityItem struct {
	Text   string
	Status string // "pending", "active", "done", "error"
}

// WizardActivityMsg updates an activity item in the left panel.
type WizardActivityMsg struct {
	Index  int
	Text   string
	Status string
}

// WizardScanDoneMsg is sent when background project scanning completes.
type WizardScanDoneMsg struct {
	TechSummary string
	IsEmptyDir  bool
}

// WizardCompleteMsg is sent when the wizard finishes and context files are written.
type WizardCompleteMsg struct{}

// Wizard is the project setup wizard component.
type Wizard struct {
	step    WizardStep
	cards   []wizardCard
	width   int
	height  int
	baseDir string

	// Left panel — activity feed
	activities []WizardActivityItem

	// Background scan results
	techSummary string
	isEmptyDir  bool
	scanDone    bool
}

// NewWizard creates a new setup wizard.
func NewWizard(baseDir string) *Wizard {
	cardDefs := []struct {
		title, prompt, placeholder string
	}{
		{
			"What are you building?",
			"Describe the product or feature you want to create.",
			"e.g. A Chrome extension that summarizes web pages using AI...",
		},
		{
			"What problem does this solve?",
			"Why does this need to exist? What pain point does it address?",
			"e.g. Users spend too much time reading long articles and miss key points...",
		},
		{
			"Who does it solve it for?",
			"Who are the primary users? What's their context?",
			"e.g. Busy professionals who need to quickly understand dense content...",
		},
		{
			"What's your vision of the end product?",
			"What does success look like? What's the main goal?",
			"e.g. A one-click summary that captures the 3 most important points...",
		},
		{
			"Tech stack",
			"Do you have a tech stack in mind, or leave blank for a recommendation.",
			"e.g. React, TypeScript, Node.js",
		},
	}

	cards := make([]wizardCard, len(cardDefs))
	for i, def := range cardDefs {
		ta := textarea.New()
		ta.Placeholder = def.placeholder
		ta.CharLimit = 2000
		ta.SetHeight(6)
		ta.ShowLineNumbers = false
		ta.Prompt = ""
		if i == 0 {
			ta.Focus()
		}
		cards[i] = wizardCard{
			title:       def.title,
			prompt:      def.prompt,
			placeholder: def.placeholder,
			input:       ta,
		}
	}

	return &Wizard{
		step:    WizardStepBuilding,
		cards:   cards,
		baseDir: baseDir,
		activities: []WizardActivityItem{
			{Text: "Scanning project structure", Status: "active"},
			{Text: "Detecting tech stack", Status: "pending"},
			{Text: "Reviewing your answers", Status: "pending"},
			{Text: "Writing context files", Status: "pending"},
			{Text: "Ready for PRD creation", Status: "pending"},
		},
	}
}

// SetSize sets the wizard dimensions.
func (w *Wizard) SetSize(width, height int) {
	w.width = width
	w.height = height

	// Update textarea widths for the right panel
	rightWidth := (width * detailsPanelPct / 100) - 8 // borders + padding
	if rightWidth < 20 {
		rightWidth = 20
	}
	for i := range w.cards {
		w.cards[i].input.SetWidth(rightWidth)
	}
}

// StartScan returns a command that scans the project directory in the background.
func (w *Wizard) StartScan() tea.Cmd {
	baseDir := w.baseDir
	return func() tea.Msg {
		stack := projectctx.DetectTechStack(baseDir)
		summary := projectctx.TechStackSummary(stack)
		isEmpty := projectctx.IsEmptyProject(baseDir)
		return WizardScanDoneMsg{TechSummary: summary, IsEmptyDir: isEmpty}
	}
}

// CurrentStep returns the current wizard step.
func (w *Wizard) CurrentStep() WizardStep {
	return w.step
}

// StepCount returns the total number of wizard steps.
func (w *Wizard) StepCount() int {
	return int(wizardStepCount)
}

// Answer returns the user's answer for a given step.
func (w *Wizard) Answer(step WizardStep) string {
	if int(step) >= len(w.cards) {
		return ""
	}
	return strings.TrimSpace(w.cards[step].input.Value())
}

// advance moves to the next card.
func (w *Wizard) advance() {
	if int(w.step) < len(w.cards)-1 {
		w.cards[w.step].input.Blur()
		w.step++
		w.cards[w.step].input.Focus()
	}
}

// retreat moves to the previous card.
func (w *Wizard) retreat() {
	if w.step > 0 {
		w.cards[w.step].input.Blur()
		w.step--
		w.cards[w.step].input.Focus()
	}
}

// isLastStep returns true if the current step is the last one.
func (w *Wizard) isLastStep() bool {
	return int(w.step) == len(w.cards)-1
}

// writeContextFiles writes the collected answers to .melliza/context/.
func (w *Wizard) writeContextFiles() error {
	building := w.Answer(WizardStepBuilding)
	problem := w.Answer(WizardStepProblem)
	users := w.Answer(WizardStepUsers)
	vision := w.Answer(WizardStepVision)
	techStack := w.Answer(WizardStepTechStack)

	// Product context
	var product strings.Builder
	product.WriteString("# Product Context\n\n")
	product.WriteString("## What We're Building\n\n")
	product.WriteString(building)
	product.WriteString("\n\n")
	product.WriteString("## Problem\n\n")
	product.WriteString(problem)
	product.WriteString("\n\n")
	product.WriteString("## Target Users\n\n")
	product.WriteString(users)
	product.WriteString("\n\n")
	product.WriteString("## Vision\n\n")
	product.WriteString(vision)
	product.WriteString("\n")

	if err := projectctx.WriteContextFile(w.baseDir, "product.md", product.String()); err != nil {
		return fmt.Errorf("failed to write product.md: %w", err)
	}

	// Tech stack
	if techStack == "" && w.techSummary != "" {
		techStack = w.techSummary + " (auto-detected)"
	}
	if techStack != "" {
		var ts strings.Builder
		ts.WriteString("# Tech Stack\n\n")
		ts.WriteString(techStack)
		ts.WriteString("\n")

		if err := projectctx.WriteContextFile(w.baseDir, "tech-stack.md", ts.String()); err != nil {
			return fmt.Errorf("failed to write tech-stack.md: %w", err)
		}
	}

	// Guidelines (minimal starter)
	var guidelines strings.Builder
	guidelines.WriteString("# Guidelines\n\n")
	guidelines.WriteString("## Coding Conventions\n\n")
	guidelines.WriteString("- Follow existing code patterns and style\n")
	guidelines.WriteString("- Write clear, descriptive variable and function names\n")
	guidelines.WriteString("- Keep functions focused and small\n")
	guidelines.WriteString("\n## Testing\n\n")
	guidelines.WriteString("- Write tests for new functionality\n")
	guidelines.WriteString("- Ensure all tests pass before completing a story\n")

	if err := projectctx.WriteContextFile(w.baseDir, "guidelines.md", guidelines.String()); err != nil {
		return fmt.Errorf("failed to write guidelines.md: %w", err)
	}

	return nil
}

// Update handles messages for the wizard.
func (w *Wizard) Update(msg tea.Msg) (*Wizard, tea.Cmd) {
	switch msg := msg.(type) {
	case WizardScanDoneMsg:
		w.scanDone = true
		w.techSummary = msg.TechSummary
		w.isEmptyDir = msg.IsEmptyDir

		// Update activity items
		w.activities[0].Status = "done"
		if w.techSummary != "" {
			w.activities[1].Text = "Detected: " + w.techSummary
			// Pre-populate the tech stack card
			w.cards[WizardStepTechStack].input.SetValue(w.techSummary)
		} else {
			w.activities[1].Text = "No tech stack detected"
		}
		w.activities[1].Status = "done"
		w.activities[2].Status = "active"
		return w, nil

	case WizardActivityMsg:
		if msg.Index >= 0 && msg.Index < len(w.activities) {
			w.activities[msg.Index].Text = msg.Text
			w.activities[msg.Index].Status = msg.Status
		}
		return w, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab":
			if w.isLastStep() {
				// Complete the wizard
				w.activities[3].Status = "active"
				if err := w.writeContextFiles(); err != nil {
					w.activities[3].Status = "error"
					w.activities[3].Text = "Error: " + err.Error()
					return w, nil
				}
				w.activities[3].Status = "done"
				w.activities[4].Status = "done"
				return w, func() tea.Msg { return WizardCompleteMsg{} }
			}
			w.advance()
			return w, nil

		case "shift+tab":
			w.retreat()
			return w, nil
		}

		// Forward all other keys (including Enter) to the textarea
		var cmd tea.Cmd
		w.cards[w.step].input, cmd = w.cards[w.step].input.Update(msg)
		return w, cmd
	}

	return w, nil
}

// View renders the wizard.
func (w *Wizard) View() string {
	if w.width < 40 || w.height < 10 {
		return "Terminal too small"
	}

	// Header
	brand := headerStyle.Render("melliza")
	setupLabel := lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true).
		Render("[Setup]")
	stepInfo := SubtitleStyle.Render(fmt.Sprintf("Step %d of %d", int(w.step)+1, len(w.cards)))

	leftHeader := lipgloss.JoinHorizontal(lipgloss.Center, brand, "  ", setupLabel)
	headerSpacing := strings.Repeat(" ", max(0, w.width-lipgloss.Width(leftHeader)-lipgloss.Width(stepInfo)-2))
	headerLine := lipgloss.JoinHorizontal(lipgloss.Center, leftHeader, headerSpacing, stepInfo)
	border := DividerStyle.Render(strings.Repeat("─", w.width))
	header := lipgloss.JoinVertical(lipgloss.Left, headerLine, border)

	// Footer
	var shortcuts string
	if w.isLastStep() {
		shortcuts = "Tab: finish  │  Shift+Tab: back  │  Enter: newline  │  Esc: quit"
	} else {
		shortcuts = "Tab: next  │  Shift+Tab: back  │  Enter: newline  │  Esc: quit"
	}
	footerLine := footerStyle.Render(shortcuts)
	footer := lipgloss.JoinVertical(lipgloss.Left, border, footerLine)

	// Content height
	headerH := 2
	footerH := 2
	contentHeight := w.height - headerH - footerH

	// Split panels
	leftWidth := (w.width * storiesPanelPct / 100) - 2
	rightWidth := w.width - leftWidth - 4

	leftPanel := w.renderActivityPanel(leftWidth, contentHeight)
	rightPanel := w.renderCardPanel(rightWidth, contentHeight)

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	content = clampHeight(content, contentHeight)

	return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}

// renderActivityPanel renders the left-side activity feed.
func (w *Wizard) renderActivityPanel(width, height int) string {
	var b strings.Builder

	// Welcome message
	welcomeStyle := lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor)
	b.WriteString(welcomeStyle.Render("Welcome to melliza"))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(MutedColor).Render("Let's set up your project."))
	b.WriteString("\n\n")

	// Setup progress
	b.WriteString(PanelTitleStyle.Render("Setup Progress"))
	b.WriteString("\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", width-4)))
	b.WriteString("\n\n")

	checkStyle := lipgloss.NewStyle().Foreground(SuccessColor)
	activeStyle := lipgloss.NewStyle().Foreground(PrimaryColor)
	pendingStyle := lipgloss.NewStyle().Foreground(MutedColor)
	errorStyle := lipgloss.NewStyle().Foreground(ErrorColor)

	for _, item := range w.activities {
		var icon string
		var textStyle lipgloss.Style

		switch item.Status {
		case "done":
			icon = checkStyle.Render("✓")
			textStyle = lipgloss.NewStyle().Foreground(TextColor)
		case "active":
			icon = activeStyle.Render("⠋")
			textStyle = lipgloss.NewStyle().Foreground(TextColor)
		case "error":
			icon = errorStyle.Render("✗")
			textStyle = errorStyle
		default: // pending
			icon = pendingStyle.Render("○")
			textStyle = pendingStyle
		}

		b.WriteString(fmt.Sprintf("  %s %s\n", icon, textStyle.Render(item.Text)))
	}

	// Card completion indicators
	b.WriteString("\n")
	b.WriteString(PanelTitleStyle.Render("Your Answers"))
	b.WriteString("\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", width-4)))
	b.WriteString("\n\n")

	for i, card := range w.cards {
		answered := strings.TrimSpace(card.input.Value()) != ""
		var icon string
		var label string

		if WizardStep(i) == w.step {
			icon = activeStyle.Render("▸")
			label = lipgloss.NewStyle().Bold(true).Foreground(TextColor).Render(card.title)
		} else if answered {
			icon = checkStyle.Render("✓")
			label = lipgloss.NewStyle().Foreground(TextColor).Render(card.title)
		} else {
			icon = pendingStyle.Render("○")
			label = pendingStyle.Render(card.title)
		}
		b.WriteString(fmt.Sprintf("  %s %s\n", icon, label))
	}

	return panelStyle.Width(width).Height(height).Render(b.String())
}

// renderCardPanel renders the right-side card with the current question.
func (w *Wizard) renderCardPanel(width, height int) string {
	card := &w.cards[w.step]

	var b strings.Builder

	// Card title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor)
	b.WriteString(titleStyle.Render(card.title))
	b.WriteString("\n\n")

	// Prompt/subtitle
	promptStyle := lipgloss.NewStyle().Foreground(TextColor)
	b.WriteString(promptStyle.Render(card.prompt))
	b.WriteString("\n\n")

	// Textarea
	b.WriteString(card.input.View())
	b.WriteString("\n")

	// Navigation hint at the bottom
	b.WriteString("\n")
	hintStyle := lipgloss.NewStyle().Foreground(MutedColor).Italic(true)
	if w.isLastStep() {
		b.WriteString(hintStyle.Render("Press Tab to complete setup"))
	} else {
		b.WriteString(hintStyle.Render("Press Tab to continue"))
	}

	return panelStyle.Width(width).Height(height).Render(b.String())
}
