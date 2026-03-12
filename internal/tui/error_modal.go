package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ErrorModalChoice represents what the user selected in the error modal.
type ErrorModalChoice int

const (
	ErrorChoiceNone      ErrorModalChoice = iota
	ErrorChoiceSaveQuit                   // Save progress and quit Melliza
	ErrorChoiceStopLoop                   // Stop just this loop, stay in TUI
	ErrorChoiceContinue                   // Dismiss and keep going
)

// errorItem implements list.DefaultItem for the fancy list in the error modal.
type errorItem struct {
	title       string
	description string
	choice      ErrorModalChoice
	recommended bool
}

func (i errorItem) Title() string {
	if i.recommended {
		return i.title + " (Recommended)"
	}
	return i.title
}
func (i errorItem) Description() string { return i.description }
func (i errorItem) FilterValue() string { return i.title }

// errorModalKeyMap holds keys specific to the error modal list.
type errorModalKeyMap struct {
	choose key.Binding
}

func (k errorModalKeyMap) ShortHelp() []key.Binding  { return []key.Binding{k.choose} }
func (k errorModalKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{{k.choose}} }

func newErrorModalKeyMap() errorModalKeyMap {
	return errorModalKeyMap{
		choose: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
	}
}

// ErrorModal is a centered overlay that appears after repeated failures without
// progress. It uses a list-fancy style delegate so each option has a title and
// description row with visual highlighting.
type ErrorModal struct {
	list    list.Model
	keys    errorModalKeyMap
	errText string // trimmed error message to display
	width   int
	height  int
	done    bool
	choice  ErrorModalChoice
}

// NewErrorModal creates an ErrorModal. Width/height are set later via SetSize.
func NewErrorModal() *ErrorModal {
	keys := newErrorModalKeyMap()

	items := []list.Item{
		errorItem{
			title:       "Save and quit",
			description: "Stop the loop, save your progress, and exit Melliza",
			choice:      ErrorChoiceSaveQuit,
			recommended: true,
		},
		errorItem{
			title:       "Stop loop",
			description: "Keep Melliza open but stop running this PRD",
			choice:      ErrorChoiceStopLoop,
		},
		errorItem{
			title:       "Continue anyway",
			description: "Dismiss this message and let the loop keep trying",
			choice:      ErrorChoiceContinue,
		},
	}

	delegate := newErrorModalDelegate(keys)
	l := list.New(items, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.KeyMap.Quit = key.NewBinding() // disable q-to-quit inside the modal

	return &ErrorModal{
		list: l,
		keys: keys,
	}
}

// newErrorModalDelegate returns a styled list delegate for the error modal.
func newErrorModalDelegate(keys errorModalKeyMap) list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	// Style: selected item uses error red for title
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.
		Foreground(ErrorColor).
		BorderLeftForeground(ErrorColor)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.
		Foreground(lipgloss.Color("#FF8C87")).
		BorderLeftForeground(ErrorColor)

	d.UpdateFunc = func(msg tea.Msg, m *list.Model) tea.Cmd {
		// enter is handled in ErrorModal.Update; nothing special here
		return nil
	}

	d.ShortHelpFunc = func() []key.Binding { return []key.Binding{keys.choose} }
	d.FullHelpFunc = func() [][]key.Binding { return [][]key.Binding{{keys.choose}} }

	return d
}

// SetSize updates the modal and inner list dimensions.
func (m *ErrorModal) SetSize(width, height int) {
	m.width = width
	m.height = height
	// Inner list width = modal content area (modal is ~60 chars wide, minus padding/border)
	modalWidth := min(64, width-10)
	if modalWidth < 44 {
		modalWidth = 44
	}
	// 3 items × 2 lines each + spacing
	listHeight := 3 * 3
	m.list.SetSize(modalWidth-4, listHeight)
}

// SetError stores the error text shown inside the modal header.
func (m *ErrorModal) SetError(errText string) {
	// Trim to keep the modal compact
	if len(errText) > 160 {
		errText = errText[:157] + "..."
	}
	m.errText = errText
	m.done = false
	m.choice = ErrorChoiceNone
	// Reset selection to first item (Save and quit)
	m.list.Select(0)
}

// Reset resets modal state so it can be reused for a new error.
func (m *ErrorModal) Reset() {
	m.done = false
	m.choice = ErrorChoiceNone
	m.list.Select(0)
}

// IsDone returns true once the user has confirmed a choice.
func (m *ErrorModal) IsDone() bool { return m.done }

// Choice returns the selected option (valid after IsDone() == true).
func (m *ErrorModal) Choice() ErrorModalChoice { return m.choice }

// Update handles key input for the error modal.
func (m *ErrorModal) Update(msg tea.Msg) (*ErrorModal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.choose):
			if sel, ok := m.list.SelectedItem().(errorItem); ok {
				m.choice = sel.choice
				m.done = true
			}
			return m, nil
		case msg.String() == "esc":
			// Esc = dismiss (same as Continue)
			m.choice = ErrorChoiceContinue
			m.done = true
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// Render draws the centered error modal overlay.
func (m *ErrorModal) Render() string {
	modalWidth := min(64, m.width-10)
	if modalWidth < 44 {
		modalWidth = 44
	}
	innerWidth := modalWidth - 4 // subtract border + padding

	var b strings.Builder

	// ── Title ──
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ErrorColor)
	b.WriteString(titleStyle.Render("Oops!"))
	b.WriteString("\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", innerWidth)))
	b.WriteString("\n\n")

	// ── Sub-heading ──
	subStyle := lipgloss.NewStyle().Foreground(WarningColor).Bold(true)
	b.WriteString(subStyle.Render("Looks like we hit a snag..."))
	b.WriteString("\n\n")

	// ── Error message ──
	if m.errText != "" {
		errStyle := lipgloss.NewStyle().
			Foreground(TextMutedColor).
			Width(innerWidth)
		// Word-wrap manually so long lines don't overflow
		b.WriteString(errStyle.Render(m.errText))
		b.WriteString("\n\n")
	}

	b.WriteString(DividerStyle.Render(strings.Repeat("─", innerWidth)))
	b.WriteString("\n")

	// ── Fancy list of options ──
	b.WriteString(m.list.View())
	b.WriteString("\n")

	// ── Footer hint ──
	b.WriteString(DividerStyle.Render(strings.Repeat("─", innerWidth)))
	b.WriteString("\n")
	footerStyle := lipgloss.NewStyle().Foreground(MutedColor)
	b.WriteString(footerStyle.Render("↑/↓ Navigate   Enter Select   Esc Dismiss"))

	// ── Modal box ──
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ErrorColor).
		Padding(1, 2).
		Width(modalWidth)

	modal := modalStyle.Render(b.String())
	return CenterModal(modal, m.width, m.height)
}
