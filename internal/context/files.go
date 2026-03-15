package context

import (
	"os"
	"path/filepath"
	"strings"
)

const contextSubdir = ".melliza/context"

// ContextDir returns the path to the context files directory.
func ContextDir(baseDir string) string {
	return filepath.Join(baseDir, contextSubdir)
}

// HasContextFiles returns true if the project has context files (product.md exists).
func HasContextFiles(baseDir string) bool {
	productPath := filepath.Join(ContextDir(baseDir), "product.md")
	_, err := os.Stat(productPath)
	return err == nil
}

// WriteContextFile writes a context file to .melliza/context/{name}.
// Creates the directory if it doesn't exist.
func WriteContextFile(baseDir, name, content string) error {
	dir := ContextDir(baseDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
}

// ReadContextFile reads a context file from .melliza/context/{name}.
func ReadContextFile(baseDir, name string) (string, error) {
	data, err := os.ReadFile(filepath.Join(ContextDir(baseDir), name))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// HasMellizaDir returns true if the .melliza directory exists.
func HasMellizaDir(baseDir string) bool {
	_, err := os.Stat(filepath.Join(baseDir, ".melliza"))
	return err == nil
}

// IsEmptyProject returns true if the directory has no non-hidden files.
func IsEmptyProject(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return true
	}
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") {
			return false
		}
	}
	return true
}

// TechStackSummary returns a human-readable summary of the detected tech stack.
func TechStackSummary(stack TechStack) string {
	var items []string
	if stack.Go {
		items = append(items, "Go")
	}
	if stack.Node {
		items = append(items, "Node.js")
	}
	if stack.Python {
		items = append(items, "Python")
	}
	if stack.Rust {
		items = append(items, "Rust")
	}
	if stack.Java {
		items = append(items, "Java")
	}
	if len(items) == 0 {
		return ""
	}
	return strings.Join(items, ", ")
}
