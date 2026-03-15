package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasContextFiles_NoDir(t *testing.T) {
	dir := t.TempDir()
	if HasContextFiles(dir) {
		t.Error("Expected HasContextFiles to return false for empty dir")
	}
}

func TestHasContextFiles_WithProductMD(t *testing.T) {
	dir := t.TempDir()
	ctxDir := filepath.Join(dir, ".melliza", "context")
	os.MkdirAll(ctxDir, 0755)
	os.WriteFile(filepath.Join(ctxDir, "product.md"), []byte("test"), 0644)

	if !HasContextFiles(dir) {
		t.Error("Expected HasContextFiles to return true when product.md exists")
	}
}

func TestWriteContextFile(t *testing.T) {
	dir := t.TempDir()
	err := WriteContextFile(dir, "product.md", "# My Product")
	if err != nil {
		t.Fatalf("WriteContextFile() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".melliza", "context", "product.md"))
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}
	if string(data) != "# My Product" {
		t.Errorf("Expected '# My Product', got %q", string(data))
	}
}

func TestReadContextFile(t *testing.T) {
	dir := t.TempDir()
	WriteContextFile(dir, "tech-stack.md", "Go, React")

	content, err := ReadContextFile(dir, "tech-stack.md")
	if err != nil {
		t.Fatalf("ReadContextFile() error: %v", err)
	}
	if content != "Go, React" {
		t.Errorf("Expected 'Go, React', got %q", content)
	}
}

func TestReadContextFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadContextFile(dir, "nonexistent.md")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestIsEmptyProject(t *testing.T) {
	dir := t.TempDir()
	if !IsEmptyProject(dir) {
		t.Error("Expected empty temp dir to be detected as empty")
	}

	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)
	if IsEmptyProject(dir) {
		t.Error("Expected dir with main.go to not be empty")
	}
}

func TestIsEmptyProject_HiddenFilesOnly(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(""), 0644)
	if !IsEmptyProject(dir) {
		t.Error("Expected dir with only hidden files to be detected as empty")
	}
}

func TestHasMellizaDir(t *testing.T) {
	dir := t.TempDir()
	if HasMellizaDir(dir) {
		t.Error("Expected HasMellizaDir false for fresh dir")
	}

	os.MkdirAll(filepath.Join(dir, ".melliza"), 0755)
	if !HasMellizaDir(dir) {
		t.Error("Expected HasMellizaDir true after creating .melliza")
	}
}

func TestTechStackSummary(t *testing.T) {
	tests := []struct {
		stack    TechStack
		expected string
	}{
		{TechStack{}, ""},
		{TechStack{Go: true}, "Go"},
		{TechStack{Go: true, Node: true}, "Go, Node.js"},
		{TechStack{Python: true, Rust: true, Java: true}, "Python, Rust, Java"},
	}
	for _, tt := range tests {
		got := TechStackSummary(tt.stack)
		if got != tt.expected {
			t.Errorf("TechStackSummary(%+v) = %q, want %q", tt.stack, got, tt.expected)
		}
	}
}
