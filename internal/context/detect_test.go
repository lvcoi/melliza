package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectTechStack_GoProject(t *testing.T) {
	dir := t.TempDir()
	// Create go.mod to simulate a Go project
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)

	stack := DetectTechStack(dir)
	if !stack.Go {
		t.Error("Expected Go to be detected")
	}
	if stack.Node {
		t.Error("Expected Node to not be detected")
	}
}

func TestDetectTechStack_NodeProject(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)

	stack := DetectTechStack(dir)
	if !stack.Node {
		t.Error("Expected Node to be detected")
	}
	if stack.Go {
		t.Error("Expected Go to not be detected")
	}
}

func TestDetectTechStack_MixedProject(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]"), 0644)

	stack := DetectTechStack(dir)
	if !stack.Go {
		t.Error("Expected Go to be detected")
	}
	if !stack.Node {
		t.Error("Expected Node to be detected")
	}
	if !stack.Rust {
		t.Error("Expected Rust to be detected")
	}
}

func TestDetectTechStack_EmptyProject(t *testing.T) {
	dir := t.TempDir()

	stack := DetectTechStack(dir)
	if stack.Go || stack.Node || stack.Python || stack.Rust || stack.Java {
		t.Error("Expected no tech stack to be detected in empty directory")
	}
}
