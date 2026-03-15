// Package context provides tech stack detection and GEMINI.md generation
// for configuring the agent's project awareness.
package context

import (
	"os"
	"path/filepath"
)

// TechStack holds the detected technology stack for a project.
type TechStack struct {
	Go     bool
	Node   bool
	Python bool
	Rust   bool
	Java   bool
}

// DetectTechStack scans the given directory for marker files and returns
// the detected technology stack.
func DetectTechStack(dir string) TechStack {
	var stack TechStack

	checks := []struct {
		file  string
		field *bool
	}{
		{"go.mod", &stack.Go},
		{"package.json", &stack.Node},
		{"requirements.txt", &stack.Python},
		{"pyproject.toml", &stack.Python},
		{"setup.py", &stack.Python},
		{"Cargo.toml", &stack.Rust},
		{"pom.xml", &stack.Java},
		{"build.gradle", &stack.Java},
	}

	for _, c := range checks {
		if _, err := os.Stat(filepath.Join(dir, c.file)); err == nil {
			*c.field = true
		}
	}

	return stack
}
