package loop

import (
	"encoding/json"
	"fmt"
	"strings"
)

// EventType represents the type of event parsed from Gemini's stream-json output.
type EventType int

const (
	// EventUnknown represents an unrecognized event type.
	EventUnknown EventType = iota
	// EventIterationStart is emitted at the start of a Gemini iteration (system init).
	EventIterationStart
	// EventAssistantText is emitted when Gemini outputs text.
	EventAssistantText
	// EventToolStart is emitted when Gemini invokes a tool.
	EventToolStart
	// EventToolResult is emitted when a tool returns a result.
	EventToolResult
	// EventStoryStarted is emitted when Gemini indicates a story is being worked on.
	EventStoryStarted
	// EventStoryCompleted is emitted when Gemini completes a story.
	EventStoryCompleted
	// EventComplete is emitted when <melliza-complete/> is detected.
	EventComplete
	// EventMaxIterationsReached is emitted when max iterations are reached.
	EventMaxIterationsReached
	// EventError is emitted when an error occurs.
	EventError
	// EventRetrying is emitted when retrying after a crash.
	EventRetrying
	// EventWatchdogTimeout is emitted when the watchdog kills a hung process.
	EventWatchdogTimeout
	// EventStdout is emitted for raw stdout lines that aren't parsed as other events.
	EventStdout
	// EventStderr is emitted for raw stderr lines.
	EventStderr
	// EventRateLimit is emitted when Gemini hits a rate-limit / quota error in stderr.
	// The loop stops automatically; the TUI should offer the user options.
	EventRateLimit
	// EventReviewStart is emitted when a review agent begins reviewing an iteration.
	EventReviewStart
	// EventReviewPass is emitted when the review agent accepts the iteration's changes.
	EventReviewPass
	// EventReviewFail is emitted when the review agent rejects the iteration's changes.
	EventReviewFail
)

// String returns the string representation of an EventType.
func (e EventType) String() string {
	switch e {
	case EventIterationStart:
		return "IterationStart"
	case EventAssistantText:
		return "AssistantText"
	case EventToolStart:
		return "ToolStart"
	case EventToolResult:
		return "ToolResult"
	case EventStoryStarted:
		return "StoryStarted"
	case EventStoryCompleted:
		return "StoryCompleted"
	case EventComplete:
		return "Complete"
	case EventMaxIterationsReached:
		return "MaxIterationsReached"
	case EventError:
		return "Error"
	case EventRetrying:
		return "Retrying"
	case EventWatchdogTimeout:
		return "WatchdogTimeout"
	case EventStdout:
		return "Stdout"
	case EventStderr:
		return "Stderr"
	case EventRateLimit:
		return "RateLimit"
	case EventReviewStart:
		return "ReviewStart"
	case EventReviewPass:
		return "ReviewPass"
	case EventReviewFail:
		return "ReviewFail"
	default:
		return "Unknown"
	}
}

// Event represents a parsed event from Gemini's stream-json output.
type Event struct {
	Type       EventType
	Iteration  int
	Text       string
	Tool       string
	ToolInput  map[string]interface{}
	StoryID    string
	Err        error
	RetryCount int // Current retry attempt (1-based)
	RetryMax   int // Maximum retries allowed
}

// streamMessage represents the top-level structure of a stream-json line.
type streamMessage struct {
	Type    string          `json:"type"`
	Subtype string          `json:"subtype,omitempty"`
	Message json.RawMessage `json:"message,omitempty"`
}

// assistantMessage represents the structure of an assistant message.
type assistantMessage struct {
	Content []contentBlock `json:"content"`
}

// contentBlock represents a block of content in an assistant message.
type contentBlock struct {
	Type  string                 `json:"type"`
	Text  string                 `json:"text,omitempty"`
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
}

// userMessage represents a tool result message.
type userMessage struct {
	Content []toolResultBlock `json:"content"`
}

// toolResultBlock represents a tool result in a user message.
type toolResultBlock struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
}

// ParseLine parses a single line of stream-json output and returns an Event.
// If the line cannot be parsed or is not relevant, it returns nil.
func ParseLine(line string) *Event {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	var msg streamMessage
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return nil
	}

	switch msg.Type {
	case "system":
		if msg.Subtype == "init" {
			return &Event{Type: EventIterationStart}
		}
		return nil

	case "assistant":
		return parseAssistantMessage(msg.Message)

	case "user":
		return parseUserMessage(msg.Message)

	case "result":
		return parseResultMessage(line)

	default:
		return nil
	}
}

// parseAssistantMessage parses an assistant message and returns appropriate events.
func parseAssistantMessage(raw json.RawMessage) *Event {
	if raw == nil {
		return nil
	}

	var msg assistantMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil
	}

	// Process content blocks - return the first meaningful event
	// In practice, we might want to return multiple events, but for simplicity
	// we return the first one found
	for _, block := range msg.Content {
		switch block.Type {
		case "text":
			text := block.Text
			// Check for <melliza-complete/> tag
			if strings.Contains(text, "<melliza-complete/>") {
				return &Event{
					Type: EventComplete,
					Text: text,
				}
			}
			// Check for story markers using ralph-status tags
			if storyID := extractStoryID(text, "<ralph-status>", "</ralph-status>"); storyID != "" {
				return &Event{
					Type:    EventStoryStarted,
					Text:    text,
					StoryID: storyID,
				}
			}
			return &Event{
				Type: EventAssistantText,
				Text: text,
			}

		case "tool_use":
			return &Event{
				Type:      EventToolStart,
				Tool:      block.Name,
				ToolInput: block.Input,
			}
		}
	}

	return nil
}

// parseUserMessage parses a user message (typically tool results).
func parseUserMessage(raw json.RawMessage) *Event {
	if raw == nil {
		return nil
	}

	var msg userMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil
	}

	for _, block := range msg.Content {
		if block.Type == "tool_result" {
			return &Event{
				Type: EventToolResult,
				Text: block.Content,
			}
		}
	}

	return nil
}

// parseResultMessage parses a result event. Gemini emits these at the end of a
// turn — when status is "error" we surface the API error message.
func parseResultMessage(line string) *Event {
	var raw struct {
		Status string `json:"status"`
		Error  struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return nil
	}
	if raw.Status == "error" && raw.Error.Message != "" {
		return &Event{
			Type: EventError,
			Text: raw.Error.Message,
			Err:  fmt.Errorf("Gemini API error: %s", raw.Error.Message),
		}
	}
	return nil
}

// extractStoryID extracts a story ID from text between start and end tags.
func extractStoryID(text, startTag, endTag string) string {
	startIdx := strings.Index(text, startTag)
	if startIdx == -1 {
		return ""
	}
	startIdx += len(startTag)

	endIdx := strings.Index(text[startIdx:], endTag)
	if endIdx == -1 {
		return ""
	}

	return strings.TrimSpace(text[startIdx : startIdx+endIdx])
}
