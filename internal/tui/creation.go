package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/lvcoi/melliza/internal/loop"
)

// ChatMode distinguishes between creating a new PRD and editing an existing one.
type ChatMode int

const (
	ChatModeCreate ChatMode = iota
	ChatModeEdit
)

// chatSpinnerTickMsg is sent to animate the waiting display in the creation chat.
type chatSpinnerTickMsg struct{}

// spinnerFrames are the braille spinner characters.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// chatRobotFrames are ASCII art frames for the waiting animation.
var chatRobotFrames = []string{
	"   ╭─────╮\n   │ ◉ ◉ │\n   │  ▽  │\n   ╰──┬──╯\n      │\n   ╭──┴──╮\n   │     │\n   ╰─────╯",
	"   ╭─────╮\n   │ ◉ ◉ │\n   │  ◇  │\n   ╰──┬──╯\n      │\n   ╭──┴──╮\n   │ ░░░ │\n   ╰─────╯",
	"   ╭─────╮\n   │ ◑ ◑ │\n   │  ▽  │\n   ╰──┬──╯\n      │\n   ╭──┴──╮\n   │ ▓▓▓ │\n   ╰─────╯",
	"   ╭─────╮\n   │ ◉ ◉ │\n   │  △  │\n   ╰──┬──╯\n      │\n   ╭──┴──╮\n   │ ███ │\n   ╰─────╯",
}

// chatWaitingJokes are shown while waiting for Gemini to respond.
var chatWaitingJokes = []string{
	"Why do programmers prefer dark mode? Because light attracts bugs.",
	"There are only 10 types of people: those who understand binary and those who don't.",
	"A SQL query walks into a bar, sees two tables and asks... 'Can I JOIN you?'",
	"!false — it's funny because it's true.",
	"Why do Java developers wear glasses? Because they can't C#.",
	"There's no place like 127.0.0.1.",
	"Algorithm: a word used by programmers when they don't want to explain what they did.",
	"It works on my machine. Ship it!",
	"99 little bugs in the code, 99 little bugs. Take one down, patch it around... 127 little bugs in the code.",
	"Debugging is like being the detective in a crime movie where you are also the murderer.",
	"I asked the AI to write a PRD. It wrote a PRD about writing PRDs.",
	"You're absolutely right. That's a great point. I completely agree.\n— Gemini, before doing what it was already going to do",
	"The AI said it was 95% confident. It was not.",
	"Prompt engineering: the art of saying 'no really, do what I said' in 47 different ways.",
	"The LLM hallucinated a library that doesn't exist.\nHonestly, the API looked pretty good though.",
	"AI will replace programmers any day now.\n— programmers, every year since 2022",
	"The code works and nobody knows why. The code breaks and nobody knows why.",
	"Homer: 'To start, press any key.' Where's the ANY key?!",
}

// Message role constants
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
)

// Message represents a single message in the chat history.
type Message struct {
	Role    string
	Content string
}

// PRDCreationChat represents the TUI component for interactive PRD creation.
type PRDCreationChat struct {
	prdName    string
	prdDir     string
	baseDir    string
	context    string
	mode       ChatMode
	messages   []Message
	sessionID  string
	input      textarea.Model
	viewport   viewport.Model
	width      int
	height     int
	loading    bool
	done       bool
	err        error

	// Track if Gemini has saved the PRD
	prdSaved bool

	// Question modal — shown when Gemini responds with structured clarifying questions
	questionModal        *QuestionModal
	showingQuestionModal bool

	// Waiting animation state
	spinnerFrame   int
	robotFrame     int
	jokeIndex      int
	lastJokeChange time.Time
	loadingStart   time.Time

	// Streaming state
	events           chan ChatEventMsg // Channel for streaming events from Gemini
	streamingContent string           // In-progress assistant text during streaming
	currentActivity  string           // Current tool being used (e.g. "Read", "Write")
}

// NewPRDCreationChat creates a new PRDCreationChat component.
func NewPRDCreationChat(baseDir, prdName, context string) *PRDCreationChat {
	ta := textarea.New()
	ta.Placeholder = "Type your response..."
	ta.Focus()
	ta.CharLimit = 1000
	ta.SetWidth(50)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	km := ta.KeyMap
	km.InsertNewline = key.NewBinding(key.WithKeys("shift+enter"))
	ta.KeyMap = km

	vp := viewport.New()

	return &PRDCreationChat{
		prdName:   prdName,
		prdDir:    filepath.Join(baseDir, ".melliza", "prds", prdName),
		baseDir:   baseDir,
		context:   context,
		messages:  make([]Message, 0),
		input:     ta,
		viewport:  vp,
		loading:   false,
		done:      false,
		jokeIndex: rand.Intn(len(chatWaitingJokes)),
	}
}

// SetMode sets the chat mode (create or edit).
func (c *PRDCreationChat) SetMode(mode ChatMode) {
	c.mode = mode
}

// SetSize sets the dimensions for the chat component.
func (c *PRDCreationChat) SetSize(width, height int) {
	c.width = width
	c.height = height
	
	// Subtract header, footer, and borders
	vpWidth := width - 4
	vpHeight := height - 15 // Account for header, textarea input, footer, and borders

	if vpWidth < 1 {
		vpWidth = 1
	}
	if vpHeight < 1 {
		vpHeight = 1
	}

	c.viewport.SetWidth(vpWidth)
	c.viewport.SetHeight(vpHeight)
	c.input.SetWidth(vpWidth - 6)
	c.input.SetHeight(3)

	if c.questionModal != nil {
		c.questionModal.SetSize(width, height)
	}

	c.renderViewport()
}

// ChatEventMsg is sent when a chat event occurs.
type ChatEventMsg struct {
	Type      string // "init", "delta", "message", "done", "error"
	Content   string
	SessionID string
}

func (c *PRDCreationChat) Init() tea.Cmd {
	return nil // Actual initialization is triggered via a specific command
}

// StartSession initiates the chat session with Gemini.
func (c *PRDCreationChat) StartSession(prompt string) tea.Cmd {
	c.loading = true
	c.loadingStart = time.Now()
	c.lastJokeChange = time.Now()
	c.messages = append(c.messages, Message{Role: RoleSystem, Content: "Initializing PRD creation session..."})
	c.renderViewport()
	c.viewport.GotoBottom()

	return c.startGeminiStream(prompt, "")
}

// SendMessage sends a user message to Gemini.
func (c *PRDCreationChat) SendMessage() tea.Cmd {
	if c.loading || c.done {
		return nil
	}

	content := c.input.Value()
	if strings.TrimSpace(content) == "" {
		return nil
	}

	if content == "/exit" {
		c.done = true
		c.messages = append(c.messages, Message{Role: RoleUser, Content: content})
		c.input.Reset()
		c.renderViewport()
		c.viewport.GotoBottom()
		return nil
	}

	c.messages = append(c.messages, Message{Role: RoleUser, Content: content})
	c.input.Reset()
	c.loading = true
	c.loadingStart = time.Now()
	c.lastJokeChange = time.Now()
	c.renderViewport()
	c.viewport.GotoBottom()

	return c.startGeminiStream(content, c.sessionID)
}

// geminiStreamMsg is the JSON structure of a Gemini stream-json line.
type geminiStreamMsg struct {
	Type    string          `json:"type"`
	Subtype string          `json:"subtype,omitempty"`
	Message json.RawMessage `json:"message,omitempty"`
}

// geminiAssistantMsg represents the assistant message body.
type geminiAssistantMsg struct {
	Content []geminiContentBlock `json:"content"`
}

// geminiContentBlock represents a content block in an assistant message.
type geminiContentBlock struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Name  string `json:"name,omitempty"`
}

// startGeminiStream starts Gemini and streams events through a channel.
// Each event is delivered as a ChatEventMsg via listenForChatEvent().
func (c *PRDCreationChat) startGeminiStream(prompt string, sessionID string) tea.Cmd {
	c.events = make(chan ChatEventMsg, 50)
	c.streamingContent = ""
	c.currentActivity = ""

	go func() {
		defer close(c.events)

		args := []string{"--yolo", "--output-format", "stream-json", "-e", "none"}
		if sessionID != "" {
			args = append(args, "-r", sessionID, "-p", prompt)
		} else {
			args = append(args, "-p", prompt)
		}

		cmd := exec.Command("gemini", args...)
		cmd.Dir = c.baseDir
		cmd.Stdin = nil

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			c.events <- ChatEventMsg{Type: "error", Content: err.Error()}
			return
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			c.events <- ChatEventMsg{Type: "error", Content: err.Error()}
			return
		}

		if err := cmd.Start(); err != nil {
			c.events <- ChatEventMsg{Type: "error", Content: err.Error()}
			return
		}

		// Capture stderr in background
		var stderrLines []string
		var stderrMu sync.Mutex
		stderrDone := make(chan struct{})
		go func() {
			defer close(stderrDone)
			prdDir := filepath.Join(c.baseDir, ".melliza", "prds", c.prdName)
			logPath := filepath.Join(prdDir, "gemini.log")
			logFile, _ := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if logFile != nil {
				defer logFile.Close()
			}
			sc := bufio.NewScanner(stderr)
			for sc.Scan() {
				line := sc.Text()
				if logFile != nil {
					logFile.WriteString("[stderr] " + line + "\n")
				}
				stderrMu.Lock()
				stderrLines = append(stderrLines, line)
				if len(stderrLines) > 20 {
					stderrLines = stderrLines[len(stderrLines)-20:]
				}
				stderrMu.Unlock()
			}
		}()

		// Parse stdout using correct Gemini stream-json format
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()

			var msg geminiStreamMsg
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				continue
			}

			switch msg.Type {
			case "system":
				if msg.Subtype == "init" {
					var initData struct {
						SessionID string `json:"session_id"`
					}
					json.Unmarshal([]byte(line), &initData)
					c.events <- ChatEventMsg{Type: "init", SessionID: initData.SessionID}
				}

			case "assistant":
				if msg.Message == nil {
					continue
				}
				var aMsg geminiAssistantMsg
				if err := json.Unmarshal(msg.Message, &aMsg); err != nil {
					continue
				}
				for _, block := range aMsg.Content {
					switch block.Type {
					case "text":
						c.events <- ChatEventMsg{Type: "delta", Content: block.Text}
					case "tool_use":
						c.events <- ChatEventMsg{Type: "tool", Content: block.Name}
					}
				}

			case "user":
				// Tool result returned — clear activity indicator
				c.events <- ChatEventMsg{Type: "tool_done"}

			case "result":
				var resultData struct {
					Status string `json:"status"`
					Error  struct {
						Message string `json:"message"`
					} `json:"error"`
				}
				json.Unmarshal([]byte(line), &resultData)
				if resultData.Status == "error" && resultData.Error.Message != "" {
					c.events <- ChatEventMsg{Type: "error", Content: fmt.Sprintf("Gemini API error: %s", resultData.Error.Message)}
				}
			}
		}

		waitErr := cmd.Wait()
		<-stderrDone

		if waitErr != nil {
			stderrMu.Lock()
			errContext := loop.FilterStderrForError(stderrLines)
			stderrMu.Unlock()
			if errContext != "" {
				c.events <- ChatEventMsg{Type: "error", Content: fmt.Sprintf("Gemini failed: %s", errContext)}
			}
		}
	}()

	return c.listenForChatEvent()
}

// listenForChatEvent returns a Bubble Tea command that reads one event from the
// streaming channel. When the channel closes, it emits a "done" event.
func (c *PRDCreationChat) listenForChatEvent() tea.Cmd {
	ch := c.events
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return ChatEventMsg{Type: "done"}
		}
		return event
	}
}

// Update handles messages and updates the component state.
func (c *PRDCreationChat) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case ChatEventMsg:
		switch msg.Type {
		case "init":
			// Session started — capture session ID for follow-up messages
			if msg.SessionID != "" {
				c.sessionID = msg.SessionID
			}
			return c, c.listenForChatEvent()

		case "delta":
			// Streaming text chunk from Gemini
			c.streamingContent += msg.Content
			c.currentActivity = "" // Clear tool activity when text arrives
			c.renderViewport()
			c.viewport.GotoBottom()
			return c, c.listenForChatEvent()

		case "tool":
			// Tool started — show activity
			c.currentActivity = msg.Content
			c.renderViewport()
			c.viewport.GotoBottom()
			return c, c.listenForChatEvent()

		case "tool_done":
			// Tool finished
			c.currentActivity = ""
			c.renderViewport()
			return c, c.listenForChatEvent()

		case "done":
			// Gemini process finished — finalize streaming content
			c.loading = false
			c.currentActivity = ""
			content := c.streamingContent
			c.streamingContent = ""

			if content != "" {
				c.messages = append(c.messages, Message{Role: RoleAssistant, Content: content})

				// Check if PRD was saved
				if strings.Contains(content, "prd.md") || strings.Contains(content, "saved") {
					c.prdSaved = true
				}
				// Check if Gemini is finished
				if strings.Contains(content, "/exit") || strings.Contains(content, "<melliza-complete/>") {
					c.done = true
				}
				// Detect structured clarifying questions
				if !c.done && !c.showingQuestionModal {
					if qs := ParseQuestions(content); len(qs) >= 2 {
						c.questionModal = NewQuestionModal(qs)
						c.questionModal.SetSize(c.width, c.height)
						c.showingQuestionModal = true
					}
				}
			}
			c.renderViewport()
			c.viewport.GotoBottom()

		case "message":
			// Legacy: single-shot message (backward compat)
			c.loading = false
			if msg.SessionID != "" {
				c.sessionID = msg.SessionID
			}
			c.messages = append(c.messages, Message{Role: RoleAssistant, Content: msg.Content})
			if strings.Contains(msg.Content, "prd.md") || strings.Contains(msg.Content, "saved") {
				c.prdSaved = true
			}
			if strings.Contains(msg.Content, "/exit") || strings.Contains(msg.Content, "<melliza-complete/>") {
				c.done = true
			}
			if !c.done && !c.showingQuestionModal {
				if qs := ParseQuestions(msg.Content); len(qs) >= 2 {
					c.questionModal = NewQuestionModal(qs)
					c.questionModal.SetSize(c.width, c.height)
					c.showingQuestionModal = true
				}
			}
			c.renderViewport()
			c.viewport.GotoBottom()

		case "error":
			c.loading = false
			c.currentActivity = ""
			c.streamingContent = ""
			c.err = fmt.Errorf("%s", msg.Content)
			c.messages = append(c.messages, Message{Role: RoleSystem, Content: "Error: " + msg.Content})
			c.renderViewport()
			c.viewport.GotoBottom()
		}

	case tea.KeyPressMsg:
		// Question modal intercepts all keys when active
		if c.showingQuestionModal && c.questionModal != nil {
			updated, modalCmd := c.questionModal.Update(msg)
			c.questionModal = updated
			if c.questionModal.IsDone() {
				cancelled := c.questionModal.Cancelled()
				answer := c.questionModal.BuildAnswer()
				c.showingQuestionModal = false
				c.questionModal = nil
				if !cancelled && answer != "" {
					// Inject the answer as a user message and send to Gemini
					c.input.SetValue(answer)
					return c, c.SendMessage()
				}
			}
			return c, modalCmd
		}

		// Viewport scrolling — always available
		switch msg.String() {
		case "pgup", "ctrl+u":
			c.viewport.HalfPageUp()
			return c, nil
		case "pgdown", "ctrl+d":
			c.viewport.HalfPageDown()
			return c, nil
		case "up":
			c.viewport.ScrollUp(1)
			return c, nil
		case "down":
			c.viewport.ScrollDown(1)
			return c, nil
		}

		if c.loading || c.done {
			return c, nil
		}

		switch msg.String() {
		case "enter":
			return c, c.SendMessage()
		}

		c.input, cmd = c.input.Update(msg)
		return c, cmd
	}

	return c, nil
}

// advanceAnimation advances the waiting animation frames.
func (c *PRDCreationChat) advanceAnimation() {
	c.spinnerFrame = (c.spinnerFrame + 1) % len(spinnerFrames)
	// Robot animates slower — every 3rd tick
	if c.spinnerFrame%3 == 0 {
		c.robotFrame = (c.robotFrame + 1) % len(chatRobotFrames)
	}
	// Rotate joke every 8 seconds
	if time.Since(c.lastJokeChange) > 8*time.Second {
		c.jokeIndex = (c.jokeIndex + 1) % len(chatWaitingJokes)
		c.lastJokeChange = time.Now()
	}
}

// renderViewport prepares the content for the viewport.
func (c *PRDCreationChat) renderViewport() {
	var b strings.Builder

	for _, m := range c.messages {
		switch m.Role {
		case RoleUser:
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Render("You: "))
			b.WriteString("\n")
			b.WriteString(m.Content)
		case RoleAssistant:
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(SuccessColor).Render("Gemini: "))
			b.WriteString("\n")
			b.WriteString(renderGlamour(m.Content, c.viewport.Width()-2))
		case RoleSystem:
			b.WriteString(lipgloss.NewStyle().Italic(true).Foreground(MutedColor).Render(m.Content))
		}
		b.WriteString("\n\n")
	}

	// Show in-progress streaming content
	if c.streamingContent != "" {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(SuccessColor).Render("Gemini: "))
		b.WriteString("\n")
		// Use plain text during streaming to avoid rendering issues with partial markdown
		b.WriteString(lipgloss.NewStyle().Foreground(TextColor).Render(c.streamingContent))
		b.WriteString("\n\n")
	}

	if c.loading {
		elapsed := time.Since(c.loadingStart).Truncate(time.Second)
		spinner := spinnerFrames[c.spinnerFrame]

		// Show tool activity if a tool is running
		if c.currentActivity != "" {
			activityLine := fmt.Sprintf("%s Using %s... (%s)", spinner, c.currentActivity, elapsed)
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(WarningColor).Render(activityLine))
			b.WriteString("\n")
		} else if c.streamingContent == "" {
			// Only show full waiting animation if no content has arrived yet
			robot := chatRobotFrames[c.robotFrame]
			joke := chatWaitingJokes[c.jokeIndex]

			statusLine := fmt.Sprintf("%s Gemini is thinking... (%s)", spinner, elapsed)
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Render(statusLine))
			b.WriteString("\n\n")

			b.WriteString(lipgloss.NewStyle().Foreground(MutedColor).Render(robot))
			b.WriteString("\n\n")

			jokeStyle := lipgloss.NewStyle().
				Italic(true).
				Foreground(WarningColor).
				PaddingLeft(3)
			b.WriteString(jokeStyle.Render("  " + joke))
		} else {
			// Content is streaming but no tool active — just show spinner
			statusLine := fmt.Sprintf("%s Gemini is working... (%s)", spinner, elapsed)
			b.WriteString(lipgloss.NewStyle().Foreground(MutedColor).Render(statusLine))
		}
	}

	c.viewport.SetContent(b.String())
}

// View renders the component.
func (c *PRDCreationChat) View() tea.View {
	if c.width < 5 {
		return tea.NewView("Initializing...")
	}

	var b strings.Builder

	// Header
	headerText := "PRD Creation Chat"
	if c.mode == ChatModeEdit {
		headerText = "PRD Edit Chat"
	}
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Render(headerText))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(BorderColor).Render(strings.Repeat("─", c.width-4)))
	b.WriteString("\n\n")

	// Chat history
	b.WriteString(c.viewport.View())
	b.WriteString("\n\n")

	// Input field
	if !c.done {
		b.WriteString(lipgloss.NewStyle().Foreground(BorderColor).Render(strings.Repeat("─", c.width-4)))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(PrimaryColor).Render(" > "))
		b.WriteString(c.input.View())
	} else {
		doneText := "PRD completed! Press Enter to start implementation."
		if c.mode == ChatModeEdit {
			doneText = "PRD updated! Press Enter to convert and return to dashboard."
		}
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(SuccessColor).Render(doneText))
	}

	// Footer with shortcuts
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(BorderColor).Render(strings.Repeat("─", c.width-4)))
	b.WriteString("\n")
	var shortcuts string
	if c.done {
		shortcuts = "Enter: convert  │  Esc: back  │  Ctrl+C: quit"
	} else if c.loading {
		shortcuts = "Esc: back  │  Ctrl+C: quit  │  pgup/pgdn: scroll"
	} else {
		shortcuts = "Enter: send  │  Shift+Enter: newline  │  /exit: finish  │  Esc: back  │  pgup/pgdn: scroll"
	}
	b.WriteString(lipgloss.NewStyle().Foreground(MutedColor).Padding(0, 1).Render(shortcuts))

	base := lipgloss.NewStyle().Padding(1, 2).Render(b.String())

	// Overlay the question modal if active
	if c.showingQuestionModal && c.questionModal != nil {
		return tea.NewView(c.questionModal.Render())
	}

	return tea.NewView(base)
}
