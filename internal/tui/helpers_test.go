package tui

import (
	"strings"
	"testing"
)

func TestCenterModal_BasicCentering(t *testing.T) {
	modal := "hello"
	result := CenterModal(modal, 80, 40)

	lines := strings.Split(result, "\n")
	// Should have top padding (empty lines before content)
	hasTopPadding := false
	for _, line := range lines {
		if line == "" {
			hasTopPadding = true
			break
		}
	}
	if !hasTopPadding {
		t.Error("expected top padding")
	}

	// Find the content line and verify left padding
	for _, line := range lines {
		if strings.Contains(line, "hello") {
			leftPad := len(line) - len(strings.TrimLeft(line, " "))
			if leftPad == 0 {
				t.Error("expected left padding")
			}
			break
		}
	}
}

func TestCenterModal_SmallScreen(t *testing.T) {
	modal := strings.Repeat("x", 100)
	result := CenterModal(modal, 50, 10)

	// Should not panic and should clamp padding to 0
	if !strings.Contains(result, strings.Repeat("x", 100)) {
		t.Error("expected modal content to be preserved")
	}
}

func TestCenterModal_ExactFit(t *testing.T) {
	modal := "test"
	result := CenterModal(modal, 4, 1)

	// Width matches exactly, so left padding should be 0
	lines := strings.Split(result, "\n")
	found := false
	for _, line := range lines {
		if strings.Contains(line, "test") {
			leftPad := len(line) - len(strings.TrimLeft(line, " "))
			if leftPad != 0 {
				t.Errorf("expected 0 left padding for exact fit, got %d", leftPad)
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("modal content not found in output")
	}
}

func TestCenterModal_MultilineModal(t *testing.T) {
	modal := "line1\nline2\nline3"
	result := CenterModal(modal, 80, 40)

	if !strings.Contains(result, "line1") {
		t.Error("expected line1 in output")
	}
	if !strings.Contains(result, "line2") {
		t.Error("expected line2 in output")
	}
	if !strings.Contains(result, "line3") {
		t.Error("expected line3 in output")
	}
}
