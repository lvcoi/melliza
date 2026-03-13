package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lvcoi/melliza/embed"
)

// GenerateGeminiMD generates a GEMINI.md file in the given directory based on
// the detected tech stack. It does NOT overwrite an existing GEMINI.md.
func GenerateGeminiMD(dir string, stack TechStack) error {
	geminiPath := filepath.Join(dir, "GEMINI.md")

	// Don't overwrite existing GEMINI.md
	if _, err := os.Stat(geminiPath); err == nil {
		return nil
	}

	techLines := formatTechStack(stack)
	content := embed.GetGeminiMDTemplate(techLines)

	return os.WriteFile(geminiPath, []byte(content), 0644)
}

// EnsureGeminiContext detects the tech stack and generates GEMINI.md if needed.
func EnsureGeminiContext(dir string) error {
	stack := DetectTechStack(dir)
	return GenerateGeminiMD(dir, stack)
}

// formatTechStack returns a markdown-formatted description of the tech stack.
func formatTechStack(stack TechStack) string {
	var items []string

	if stack.Go {
		items = append(items, "- Go")
	}
	if stack.Node {
		items = append(items, "- Node.js / JavaScript / TypeScript")
	}
	if stack.Python {
		items = append(items, "- Python")
	}
	if stack.Rust {
		items = append(items, "- Rust")
	}
	if stack.Java {
		items = append(items, "- Java")
	}

	if len(items) == 0 {
		return fmt.Sprintf("- (no tech stack detected)")
	}

	return strings.Join(items, "\n")
}
