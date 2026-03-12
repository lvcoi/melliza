package tui

import (
	"fmt"
	"regexp"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ─── Domain types ────────────────────────────────────────────────────────────

// ParsedQuestion holds one clarifying question and its lettered options.
type ParsedQuestion struct {
	Text    string
	Options []string // e.g. ["A. Improve onboarding", "B. Increase retention"]
}

// questionOption implements list.DefaultItem for the fancy list inside the modal.
type questionOption struct {
	letter string // "A", "B", "C", …
	text   string
	isOther bool
}

func (o questionOption) Title() string       { return o.letter + ". " + o.text }
func (o questionOption) Description() string { return "" }
func (o questionOption) FilterValue() string { return o.letter }

// ─── Parsing ─────────────────────────────────────────────────────────────────

// numberedQ matches lines like "1." / "1)" at the start of a line.
var reQuestion = regexp.MustCompile(`(?m)^\s*\d+[.)]\s+(.+)`)

// letteredOpt matches lines like "A." / "a)" / "- A." at the start of a line.
var reOption = regexp.MustCompile(`(?m)^\s*[-*]?\s*([A-Ea-e])[.)]\s+(.+)`)

// ParseQuestions attempts to extract structured Q&A from Gemini's markdown text.
// Returns nil if fewer than 2 questions are found (don't trigger the modal for
// regular prose).
func ParseQuestions(text string) []ParsedQuestion {
	var questions []ParsedQuestion

	// Split on numbered question markers — each chunk is one question block.
	chunks := reQuestion.FindAllStringIndex(text, -1)
	if len(chunks) < 2 {
		return nil
	}

	// Build question blocks
	for i, loc := range chunks {
		end := len(text)
		if i+1 < len(chunks) {
			end = chunks[i+1][0]
		}
		block := text[loc[0]:end]

		// Extract question text (first line of block, stripped of numbering)
		firstLine := strings.TrimSpace(strings.SplitN(block, "\n", 2)[0])
		firstLine = regexp.MustCompile(`^\d+[.)]\s*`).ReplaceAllString(firstLine, "")

		// Extract lettered options from the block
		optMatches := reOption.FindAllStringSubmatch(block, -1)
		var opts []string
		seenLetters := map[string]bool{}
		for _, m := range optMatches {
			letter := strings.ToUpper(m[1])
			if seenLetters[letter] {
				continue
			}
			seenLetters[letter] = true
			opts = append(opts, letter+". "+strings.TrimSpace(m[2]))
		}

		if len(opts) >= 2 || len(questions) > 0 {
			// Allow a question with no options (free text) if we already have
			// some structured questions.
			questions = append(questions, ParsedQuestion{
				Text:    firstLine,
				Options: opts,
			})
		}
	}

	if len(questions) < 2 {
		return nil
	}
	return questions
}

// ─── QuestionModal ────────────────────────────────────────────────────────────

// QuestionModal presents structured clarifying questions one at a time using a
// list-fancy style delegate.  The user navigates options with ↑/↓, confirms
// with Space/Enter, and advances to the next question with Tab.
type QuestionModal struct {
	questions   []ParsedQuestion
	currentQ    int // index of the currently displayed question
	selections  []int    // selected option index per question (-1 = Other)
	otherInputs []textarea.Model

	list    list.Model // options for the current question
	listKeys questionListKeys
	width   int
	height  int
	done    bool
}

type questionListKeys struct {
	selectItem key.Binding
	next       key.Binding
	prev       key.Binding
}

func (k questionListKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.selectItem, k.next}
}
func (k questionListKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.selectItem, k.next, k.prev}}
}

func newQuestionListKeys() questionListKeys {
	return questionListKeys{
		selectItem: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter/space", "select"),
		),
		next: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next question"),
		),
		prev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev question"),
		),
	}
}

// newQuestionDelegate returns a styled list delegate for the question modal.
func newQuestionDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.
		Foreground(PrimaryColor).
		BorderLeftForeground(PrimaryColor)
	// Normal (unselected) items use muted color
	d.Styles.NormalTitle = d.Styles.NormalTitle.Foreground(TextColor)
	return d
}

// NewQuestionModal creates a new QuestionModal for the given questions.
func NewQuestionModal(questions []ParsedQuestion) *QuestionModal {
	keys := newQuestionListKeys()

	// Build "Other" textarea inputs for every question
	others := make([]textarea.Model, len(questions))
	for i := range others {
		ta := textarea.New()
		ta.Placeholder = "Type your own answer..."
		ta.CharLimit = 300
		ta.SetHeight(2)
		ta.ShowLineNumbers = false
		ta.Prompt = ""
		others[i] = ta
	}

	qm := &QuestionModal{
		questions:   questions,
		currentQ:    0,
		selections:  make([]int, len(questions)),
		otherInputs: others,
		listKeys:    keys,
	}
	// Default: no selection made yet — use -2 as sentinel
	for i := range qm.selections {
		qm.selections[i] = -2
	}

	qm.list = qm.buildList(0)
	return qm
}

// buildList constructs a list.Model for question index qi.
func (m *QuestionModal) buildList(qi int) list.Model {
	q := m.questions[qi]
	items := make([]list.Item, 0, len(q.Options)+1)
	for _, opt := range q.Options {
		letter := strings.SplitN(opt, ".", 2)[0]
		rest := strings.TrimSpace(strings.SplitN(opt, ".", 2)[1])
		items = append(items, questionOption{letter: letter, text: rest})
	}
	// Always add "Other" as last option
	items = append(items, questionOption{letter: "Other", text: "Type your own answer", isOther: true})

	d := newQuestionDelegate()
	l := list.New(items, d, m.listWidth(), m.listHeight())
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.KeyMap.Quit = key.NewBinding() // disable q inside modal

	// Restore prior selection if any
	sel := m.selections[qi]
	if sel >= 0 && sel < len(items) {
		l.Select(sel)
	}
	return l
}

func (m *QuestionModal) listWidth() int {
	w := min(60, m.width-14)
	if w < 20 {
		w = 20
	}
	return w
}

func (m *QuestionModal) listHeight() int {
	// Each item is 1 line + spacing; cap at 10 items visible
	q := m.questions[m.currentQ]
	n := len(q.Options) + 1 // +1 for Other
	if n > 10 {
		n = 10
	}
	return n * 2
}

// SetSize updates modal dimensions and rebuilds the list.
func (m *QuestionModal) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.list = m.buildList(m.currentQ)
	for i := range m.otherInputs {
		m.otherInputs[i].SetWidth(m.listWidth() - 2)
	}
}

// IsDone returns true once all questions are answered and submitted.
func (m *QuestionModal) IsDone() bool { return m.done }

// BuildAnswer formats all selections into a comma-separated string sent back
// to Gemini, e.g. "1A, 2C, 3B" or "1A, 2 Other: custom text".
func (m *QuestionModal) BuildAnswer() string {
	var parts []string
	for i, qi := range m.selections {
		q := m.questions[i]
		qNum := fmt.Sprintf("%d", i+1)
		if qi == -1 {
			// Other
			other := strings.TrimSpace(m.otherInputs[i].Value())
			if other == "" {
				other = "(no answer)"
			}
			parts = append(parts, qNum+" Other: "+other)
		} else if qi >= 0 && qi < len(q.Options) {
			letter := strings.SplitN(q.Options[qi], ".", 2)[0]
			parts = append(parts, qNum+letter)
		} else {
			// No selection — skip
			parts = append(parts, qNum+"?")
		}
	}
	return strings.Join(parts, ", ")
}

// currentOtherSelected returns true if the "Other" option is selected on the
// current question.
func (m *QuestionModal) currentOtherSelected() bool {
	qi := m.currentQ
	sel := m.selections[qi]
	return sel == -1
}

// Update handles input for the question modal.
func (m *QuestionModal) Update(msg tea.Msg) (*QuestionModal, tea.Cmd) {
	// If "Other" is selected on current question, route typing into the textarea
	if m.currentOtherSelected() {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			switch msg.String() {
			case "tab":
				return m.advance()
			case "shift+tab":
				return m.retreat()
			case "esc":
				// Deselect Other, go back to list
				m.selections[m.currentQ] = -2
				return m, nil
			}
			var cmd tea.Cmd
			m.otherInputs[m.currentQ], cmd = m.otherInputs[m.currentQ].Update(msg)
			return m, cmd
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.listKeys.next):
			return m.advance()
		case key.Matches(msg, m.listKeys.prev):
			return m.retreat()
		case key.Matches(msg, m.listKeys.selectItem):
			if sel, ok := m.list.SelectedItem().(questionOption); ok {
				if sel.isOther {
					m.selections[m.currentQ] = -1
					m.otherInputs[m.currentQ].Focus()
				} else {
					// Find option index in the current question's Options slice
					qOpts := m.questions[m.currentQ].Options
					for idx, opt := range qOpts {
						letter := strings.SplitN(opt, ".", 2)[0]
						if letter == sel.letter {
							m.selections[m.currentQ] = idx
							break
						}
					}
				}
			}
			return m.advance()
		case msg.String() == "esc":
			// Esc closes modal with whatever has been answered so far
			m.done = true
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// advance moves to next question or marks done if on the last question.
func (m *QuestionModal) advance() (*QuestionModal, tea.Cmd) {
	if m.currentQ < len(m.questions)-1 {
		m.currentQ++
		m.list = m.buildList(m.currentQ)
	} else {
		m.done = true
	}
	return m, nil
}

// retreat moves back to the previous question.
func (m *QuestionModal) retreat() (*QuestionModal, tea.Cmd) {
	if m.currentQ > 0 {
		m.currentQ--
		m.list = m.buildList(m.currentQ)
	}
	return m, nil
}

// Render draws the centered question modal overlay.
func (m *QuestionModal) Render() string {
	innerW := m.listWidth()
	q := m.questions[m.currentQ]

	var b strings.Builder

	// ── Title + progress ─────────────────────────────────────────────────────
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor)
	b.WriteString(titleStyle.Render("Clarifying Questions"))
	b.WriteString("\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", innerW)))
	b.WriteString("\n")
	progressStyle := lipgloss.NewStyle().Foreground(MutedColor)
	b.WriteString(progressStyle.Render(fmt.Sprintf(
		"Question %d of %d", m.currentQ+1, len(m.questions),
	)))
	b.WriteString("\n\n")

	// ── Question text ─────────────────────────────────────────────────────────
	qStyle := lipgloss.NewStyle().Foreground(TextColor).Bold(true).Width(innerW)
	b.WriteString(qStyle.Render(q.Text))
	b.WriteString("\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", innerW)))
	b.WriteString("\n")

	// ── Options list ──────────────────────────────────────────────────────────
	if m.currentOtherSelected() {
		// Show the textarea for "Other"
		otherLabel := lipgloss.NewStyle().Foreground(PrimaryColor).Render("Other: ")
		b.WriteString(otherLabel)
		b.WriteString("\n")
		b.WriteString(m.otherInputs[m.currentQ].View())
		b.WriteString("\n")
	} else {
		b.WriteString(m.list.View())
	}

	// ── Current selection indicator ───────────────────────────────────────────
	sel := m.selections[m.currentQ]
	if sel >= 0 && sel < len(q.Options) {
		selStyle := lipgloss.NewStyle().Foreground(SuccessColor)
		letter := strings.SplitN(q.Options[sel], ".", 2)[0]
		b.WriteString("\n")
		b.WriteString(selStyle.Render("✓ Selected: " + letter))
		b.WriteString("\n")
	} else if sel == -1 {
		selStyle := lipgloss.NewStyle().Foreground(SuccessColor)
		b.WriteString("\n")
		b.WriteString(selStyle.Render("✓ Selected: Other"))
		b.WriteString("\n")
	}

	// ── Footer ────────────────────────────────────────────────────────────────
	b.WriteString("\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", innerW)))
	b.WriteString("\n")
	footStyle := lipgloss.NewStyle().Foreground(MutedColor)
	var footText string
	if m.currentQ == len(m.questions)-1 {
		footText = "↑/↓ Navigate   Enter/Space Select+Submit   Shift+Tab Back   Esc Dismiss"
	} else {
		footText = "↑/↓ Navigate   Enter/Space Select+Next   Shift+Tab Back   Esc Dismiss"
	}
	b.WriteString(footStyle.Render(footText))

	// ── Modal box ─────────────────────────────────────────────────────────────
	modalWidth := innerW + 8 // account for padding + border
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Padding(1, 3).
		Width(modalWidth)

	modal := modalStyle.Render(b.String())
	return m.centerModal(modal)
}

func (m *QuestionModal) centerModal(modal string) string {
	lines := strings.Split(modal, "\n")
	mh := len(lines)
	mw := 0
	for _, l := range lines {
		if w := lipgloss.Width(l); w > mw {
			mw = w
		}
	}
	top := (m.height - mh) / 2
	left := (m.width - mw) / 2
	if top < 0 {
		top = 0
	}
	if left < 0 {
		left = 0
	}
	var out strings.Builder
	pad := strings.Repeat(" ", left)
	for i := 0; i < top; i++ {
		out.WriteString("\n")
	}
	for _, line := range lines {
		out.WriteString(pad)
		out.WriteString(line)
		out.WriteString("\n")
	}
	return out.String()
}
