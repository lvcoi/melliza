package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lvcoi/melliza/internal/loop"
)

func TestGetToolIcon(t *testing.T) {
	tests := []struct {
		toolName string
		expected string
	}{
		{"Read", "📖"},
		{"Edit", "✏️"},
		{"Write", "📝"},
		{"Bash", "🔨"},
		{"Glob", "🔍"},
		{"Grep", "🔎"},
		{"Task", "🤖"},
		{"WebFetch", "🌐"},
		{"WebSearch", "🌐"},
		{"Unknown", "⚙️"},
		{"", "⚙️"},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			result := getToolIcon(tt.toolName)
			if result != tt.expected {
				t.Errorf("getToolIcon(%q) = %q, want %q", tt.toolName, result, tt.expected)
			}
		})
	}
}

func TestGetToolArgument(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		input    map[string]interface{}
		expected string
	}{
		{
			name:     "Read with file_path",
			toolName: "Read",
			input:    map[string]interface{}{"file_path": "/path/to/file.go"},
			expected: "/path/to/file.go",
		},
		{
			name:     "Edit with file_path",
			toolName: "Edit",
			input:    map[string]interface{}{"file_path": "/test.go", "old_string": "foo"},
			expected: "/test.go",
		},
		{
			name:     "Bash with command",
			toolName: "Bash",
			input:    map[string]interface{}{"command": "go test ./..."},
			expected: "go test ./...",
		},
		{
			name:     "Bash with long command",
			toolName: "Bash",
			input:    map[string]interface{}{"command": "very long command that exceeds sixty characters and should be truncated"},
			expected: "very long command that exceeds sixty characters and shoul...",
		},
		{
			name:     "Glob with pattern",
			toolName: "Glob",
			input:    map[string]interface{}{"pattern": "**/*.go"},
			expected: "**/*.go",
		},
		{
			name:     "Grep with pattern",
			toolName: "Grep",
			input:    map[string]interface{}{"pattern": "func Test"},
			expected: "func Test",
		},
		{
			name:     "WebFetch with url",
			toolName: "WebFetch",
			input:    map[string]interface{}{"url": "https://example.com"},
			expected: "https://example.com",
		},
		{
			name:     "WebSearch with query",
			toolName: "WebSearch",
			input:    map[string]interface{}{"query": "golang testing"},
			expected: "golang testing",
		},
		{
			name:     "Task with description",
			toolName: "Task",
			input:    map[string]interface{}{"description": "run tests"},
			expected: "run tests",
		},
		{
			name:     "nil input",
			toolName: "Read",
			input:    nil,
			expected: "",
		},
		{
			name:     "missing key",
			toolName: "Read",
			input:    map[string]interface{}{"other": "value"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getToolArgument(tt.toolName, tt.input)
			if result != tt.expected {
				t.Errorf("getToolArgument(%q, %v) = %q, want %q", tt.toolName, tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewLogViewer(t *testing.T) {
	lv := NewLogViewer()
	if lv == nil {
		t.Fatal("NewLogViewer returned nil")
	}
	if !lv.autoScroll {
		t.Error("Expected autoScroll to be true by default")
	}
	if len(lv.entries) != 0 {
		t.Error("Expected entries to be empty")
	}
}

func TestLogViewer_Clear(t *testing.T) {
	lv := NewLogViewer()
	lv.entries = []LogEntry{{Text: "test"}}
	lv.autoScroll = false

	lv.Clear()

	if len(lv.entries) != 0 {
		t.Error("Expected entries to be empty after Clear")
	}
	if !lv.autoScroll {
		t.Error("Expected autoScroll to be true after Clear")
	}
}

func TestLogViewer_SetSize(t *testing.T) {
	lv := NewLogViewer()
	lv.SetSize(100, 50)

	if lv.width != 100 {
		t.Errorf("Expected width 100, got %d", lv.width)
	}
}

func TestLogViewer_IsAutoScrolling(t *testing.T) {
	lv := NewLogViewer()
	if !lv.IsAutoScrolling() {
		t.Error("Expected IsAutoScrolling to be true by default")
	}

	lv.ScrollUp()
	// autoScroll should still be true if at top (nothing to scroll up from)
	if !lv.IsAutoScrolling() {
		t.Error("Expected IsAutoScrolling to remain true when at top")
	}
}

// ── Fix 1: Duplicate-on-reload log bug ───────────────────────────────────────

func TestLogViewer_SaveLoadNoDuplicates(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.jsonl")

	lv := NewLogViewer()
	lv.SetSize(80, 40)

	// Write 3 entries via AddEvent + AppendEntry (simulates live events)
	for i, text := range []string{"event1", "event2", "event3"} {
		evt := loop.Event{Type: loop.EventAssistantText, Iteration: i + 1, Text: text}
		lv.AddEvent(evt)
		if len(lv.entries) > 0 {
			_ = lv.AppendEntry(logPath, lv.entries[len(lv.entries)-1])
		}
	}
	if len(lv.entries) != 3 {
		t.Fatalf("expected 3 entries after add, got %d", len(lv.entries))
	}

	// Simulate switch: save → clear → load
	_ = lv.SaveEntries(logPath)
	lv.Clear()
	_ = lv.LoadEntries(logPath)

	if len(lv.entries) != 3 {
		t.Fatalf("expected 3 entries after load, got %d", len(lv.entries))
	}

	// Rewrite file to match in-memory state (the fix)
	_ = lv.SaveEntries(logPath)

	// Add a new live event + append
	newEvt := loop.Event{Type: loop.EventAssistantText, Iteration: 4, Text: "event4"}
	lv.AddEvent(newEvt)
	if len(lv.entries) > 0 {
		_ = lv.AppendEntry(logPath, lv.entries[len(lv.entries)-1])
	}

	// Simulate another switch cycle: save → clear → load
	_ = lv.SaveEntries(logPath)
	lv.Clear()
	_ = lv.LoadEntries(logPath)

	if len(lv.entries) != 4 {
		t.Errorf("expected 4 entries after second reload, got %d", len(lv.entries))
	}

	// Verify no duplicates by checking entry text
	texts := make(map[string]int)
	for _, e := range lv.entries {
		texts[e.Text]++
	}
	for text, count := range texts {
		if count > 1 {
			t.Errorf("duplicate entry %q appeared %d times", text, count)
		}
	}
}

func TestLogViewer_LoadEntries_NonExistentFile(t *testing.T) {
	lv := NewLogViewer()
	err := lv.LoadEntries("/nonexistent/path/log.jsonl")
	if err != nil {
		t.Errorf("expected nil error for nonexistent file, got %v", err)
	}
	if len(lv.entries) != 0 {
		t.Error("expected 0 entries for nonexistent file")
	}
}

func TestLogViewer_SaveEntries_EmptyIsNoop(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.jsonl")

	lv := NewLogViewer()
	err := lv.SaveEntries(logPath)
	if err != nil {
		t.Errorf("expected nil error for empty save, got %v", err)
	}
	// File should NOT be created for empty entries
	if _, err := os.Stat(logPath); err == nil {
		t.Error("expected no file to be created for empty save")
	}
}

// ── Fix 2: LastEntry accessor ────────────────────────────────────────────────

func TestLogViewer_LastEntry_Empty(t *testing.T) {
	lv := NewLogViewer()
	_, ok := lv.LastEntry()
	if ok {
		t.Error("expected LastEntry() to return false for empty log")
	}
}

func TestLogViewer_LastEntry_WithEntries(t *testing.T) {
	lv := NewLogViewer()
	lv.SetSize(80, 40)

	lv.AddEvent(loop.Event{Type: loop.EventAssistantText, Iteration: 1, Text: "first"})
	lv.AddEvent(loop.Event{Type: loop.EventAssistantText, Iteration: 2, Text: "second"})

	entry, ok := lv.LastEntry()
	if !ok {
		t.Fatal("expected LastEntry() to return true")
	}
	if entry.Text != "second" {
		t.Errorf("expected last entry text 'second', got %q", entry.Text)
	}
	if entry.Iteration != 2 {
		t.Errorf("expected last entry iteration 2, got %d", entry.Iteration)
	}
}

func TestStripLineNumbers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "arrow format",
			input:    "   1→<?php\n   2→\n   3→use App\\Models;",
			expected: "<?php\n\nuse App\\Models;",
		},
		{
			name:     "tab format",
			input:    "   1\t<?php\n   2\t\n   3\tuse App\\Models;",
			expected: "<?php\n\nuse App\\Models;",
		},
		{
			name:     "double digit line numbers",
			input:    "  10→function test() {\n  11→    return true;\n  12→}",
			expected: "function test() {\n    return true;\n}",
		},
		{
			name:     "no line numbers",
			input:    "<?php\nuse App\\Models;",
			expected: "<?php\nuse App\\Models;",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripLineNumbers(tt.input)
			if result != tt.expected {
				t.Errorf("stripLineNumbers() =\n%q\nwant:\n%q", result, tt.expected)
			}
		})
	}
}
