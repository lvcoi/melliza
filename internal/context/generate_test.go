package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateGeminiMD_CreatesFiles(t *testing.T) {
	dir := t.TempDir()
	// Create go.mod so stack is detected
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)

	stack := DetectTechStack(dir)
	err := GenerateGeminiMD(dir, stack)
	if err != nil {
		t.Fatalf("GenerateGeminiMD() error: %v", err)
	}

	// Verify GEMINI.md was created
	geminiPath := filepath.Join(dir, "GEMINI.md")
	data, err := os.ReadFile(geminiPath)
	if err != nil {
		t.Fatalf("Failed to read GEMINI.md: %v", err)
	}
	content := string(data)

	if content == "" {
		t.Error("Expected GEMINI.md to have content")
	}

	// Should reference Go since it was detected
	if !containsAny(content, "Go", "go") {
		t.Error("Expected GEMINI.md to mention Go")
	}
}

func TestGenerateGeminiMD_DoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()

	// Create an existing GEMINI.md
	geminiPath := filepath.Join(dir, "GEMINI.md")
	os.WriteFile(geminiPath, []byte("# Custom GEMINI.md"), 0644)

	stack := DetectTechStack(dir)
	err := GenerateGeminiMD(dir, stack)
	if err != nil {
		t.Fatalf("GenerateGeminiMD() error: %v", err)
	}

	// Verify original content was preserved
	data, err := os.ReadFile(geminiPath)
	if err != nil {
		t.Fatalf("Failed to read GEMINI.md: %v", err)
	}

	if string(data) != "# Custom GEMINI.md" {
		t.Errorf("Expected GEMINI.md to not be overwritten, got %q", string(data))
	}
}

func TestEnsureGeminiContext_CreatesOnFirstRun(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)

	err := EnsureGeminiContext(dir)
	if err != nil {
		t.Fatalf("EnsureGeminiContext() error: %v", err)
	}

	// Verify GEMINI.md was created
	geminiPath := filepath.Join(dir, "GEMINI.md")
	if _, err := os.Stat(geminiPath); err != nil {
		t.Error("Expected GEMINI.md to exist after EnsureGeminiContext")
	}
}

func TestEnsureGeminiContext_PreservesExisting(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)

	// Create existing GEMINI.md
	geminiPath := filepath.Join(dir, "GEMINI.md")
	os.WriteFile(geminiPath, []byte("custom content"), 0644)

	err := EnsureGeminiContext(dir)
	if err != nil {
		t.Fatalf("EnsureGeminiContext() error: %v", err)
	}

	data, _ := os.ReadFile(geminiPath)
	if string(data) != "custom content" {
		t.Error("Expected existing GEMINI.md to be preserved")
	}
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(s) > 0 && len(sub) > 0 {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
