package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestNewWizard(t *testing.T) {
	w := NewWizard("/tmp/test")

	if w.step != WizardStepBuilding {
		t.Errorf("Expected initial step WizardStepBuilding, got %v", w.step)
	}
	if len(w.cards) != int(wizardStepCount) {
		t.Errorf("Expected %d cards, got %d", wizardStepCount, len(w.cards))
	}
	if w.baseDir != "/tmp/test" {
		t.Errorf("Expected baseDir /tmp/test, got %q", w.baseDir)
	}
}

func TestWizard_StepNavigation(t *testing.T) {
	w := NewWizard("/tmp/test")
	w.SetSize(120, 40)

	// Start at step 0
	if w.CurrentStep() != WizardStepBuilding {
		t.Errorf("Expected WizardStepBuilding, got %v", w.CurrentStep())
	}

	// Tab advances
	w, _ = w.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if w.CurrentStep() != WizardStepProblem {
		t.Errorf("After Tab, expected WizardStepProblem, got %v", w.CurrentStep())
	}

	// Tab again
	w, _ = w.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if w.CurrentStep() != WizardStepUsers {
		t.Errorf("After 2nd Tab, expected WizardStepUsers, got %v", w.CurrentStep())
	}

	// Shift+Tab goes back
	w, _ = w.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if w.CurrentStep() != WizardStepProblem {
		t.Errorf("After Shift+Tab, expected WizardStepProblem, got %v", w.CurrentStep())
	}
}

func TestWizard_ShiftTabAtFirstStep(t *testing.T) {
	w := NewWizard("/tmp/test")
	w.SetSize(120, 40)

	// Shift+Tab at first step should not go below 0
	w, _ = w.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if w.CurrentStep() != WizardStepBuilding {
		t.Errorf("Shift+Tab at step 0 should stay at 0, got %v", w.CurrentStep())
	}
}

func TestWizard_StepCount(t *testing.T) {
	w := NewWizard("/tmp/test")
	if w.StepCount() != 5 {
		t.Errorf("Expected 5 steps, got %d", w.StepCount())
	}
}

func TestWizardStep_String(t *testing.T) {
	tests := []struct {
		step     WizardStep
		expected string
	}{
		{WizardStepBuilding, "Building"},
		{WizardStepProblem, "Problem"},
		{WizardStepUsers, "Users"},
		{WizardStepVision, "Vision"},
		{WizardStepTechStack, "TechStack"},
		{WizardStep(99), "Unknown"},
	}
	for _, tt := range tests {
		if got := tt.step.String(); got != tt.expected {
			t.Errorf("WizardStep(%d).String() = %q, want %q", tt.step, got, tt.expected)
		}
	}
}

func TestWizard_ScanDoneMsg(t *testing.T) {
	w := NewWizard("/tmp/test")
	w.SetSize(120, 40)

	w, _ = w.Update(WizardScanDoneMsg{TechSummary: "Go, React", IsEmptyDir: false})

	if !w.scanDone {
		t.Error("Expected scanDone to be true after WizardScanDoneMsg")
	}
	if w.techSummary != "Go, React" {
		t.Errorf("Expected techSummary %q, got %q", "Go, React", w.techSummary)
	}
	// Tech stack card should be pre-populated
	if w.Answer(WizardStepTechStack) != "Go, React" {
		t.Errorf("Expected tech stack card to be pre-populated, got %q", w.Answer(WizardStepTechStack))
	}
	// Activity items should be updated
	if w.activities[0].Status != "done" {
		t.Error("Expected first activity to be done")
	}
	if w.activities[1].Status != "done" {
		t.Error("Expected second activity to be done")
	}
}

func TestWizard_WriteContextFiles(t *testing.T) {
	dir := t.TempDir()
	w := NewWizard(dir)
	w.SetSize(120, 40)

	// Set answers
	w.cards[WizardStepBuilding].input.SetValue("A task manager")
	w.cards[WizardStepProblem].input.SetValue("People forget things")
	w.cards[WizardStepUsers].input.SetValue("Everyone")
	w.cards[WizardStepVision].input.SetValue("Simple and fast")
	w.cards[WizardStepTechStack].input.SetValue("Go, React")

	err := w.writeContextFiles()
	if err != nil {
		t.Fatalf("writeContextFiles() error: %v", err)
	}

	// Verify product.md
	productPath := filepath.Join(dir, ".melliza", "context", "product.md")
	data, err := os.ReadFile(productPath)
	if err != nil {
		t.Fatalf("Failed to read product.md: %v", err)
	}
	content := string(data)
	if !containsStr(content, "A task manager") {
		t.Error("product.md should contain the building answer")
	}
	if !containsStr(content, "People forget things") {
		t.Error("product.md should contain the problem answer")
	}

	// Verify tech-stack.md
	tsPath := filepath.Join(dir, ".melliza", "context", "tech-stack.md")
	data, err = os.ReadFile(tsPath)
	if err != nil {
		t.Fatalf("Failed to read tech-stack.md: %v", err)
	}
	if !containsStr(string(data), "Go, React") {
		t.Error("tech-stack.md should contain the tech stack")
	}

	// Verify guidelines.md
	guidelinesPath := filepath.Join(dir, ".melliza", "context", "guidelines.md")
	if _, err := os.Stat(guidelinesPath); err != nil {
		t.Error("Expected guidelines.md to exist")
	}
}

func TestWizard_CompleteOnLastTab(t *testing.T) {
	dir := t.TempDir()
	w := NewWizard(dir)
	w.SetSize(120, 40)

	// Navigate to the last step
	for i := 0; i < int(wizardStepCount)-1; i++ {
		w, _ = w.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	}

	if !w.isLastStep() {
		t.Fatalf("Expected to be on last step, on %v", w.CurrentStep())
	}

	// Set at least one answer so files have content
	w.cards[WizardStepBuilding].input.SetValue("test")

	// Tab on last step should trigger completion
	var cmd tea.Cmd
	w, cmd = w.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	if cmd == nil {
		t.Fatal("Expected a command from completing the wizard")
	}

	// Execute the command to verify it returns WizardCompleteMsg
	msg := cmd()
	if _, ok := msg.(WizardCompleteMsg); !ok {
		t.Errorf("Expected WizardCompleteMsg, got %T", msg)
	}

	// Context files should be written
	productPath := filepath.Join(dir, ".melliza", "context", "product.md")
	if _, err := os.Stat(productPath); err != nil {
		t.Error("Expected product.md to be created on completion")
	}
}

func TestWizard_Render(t *testing.T) {
	w := NewWizard("/tmp/test")
	w.SetSize(120, 40)

	output := w.View()
	if output == "" {
		t.Error("Expected non-empty render output")
	}
	// Should contain the brand
	if !containsStr(output, "melliza") {
		t.Error("Expected render to contain 'melliza'")
	}
	// Should show step indicator
	if !containsStr(output, "Step 1") {
		t.Error("Expected render to contain 'Step 1'")
	}
}

func TestWizard_RenderTooSmall(t *testing.T) {
	w := NewWizard("/tmp/test")
	w.SetSize(30, 5)
	output := w.View()
	if !containsStr(output, "too small") {
		t.Error("Expected 'too small' message for tiny terminal")
	}
}

func TestWizard_Answer(t *testing.T) {
	w := NewWizard("/tmp/test")
	w.cards[WizardStepBuilding].input.SetValue("  my product  ")

	ans := w.Answer(WizardStepBuilding)
	if ans != "my product" {
		t.Errorf("Expected trimmed answer %q, got %q", "my product", ans)
	}

	// Out of bounds returns empty
	if w.Answer(WizardStep(99)) != "" {
		t.Error("Expected empty string for out-of-bounds step")
	}
}

// containsStr is a simple substring check for tests.
func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && len(sub) > 0 && stringContains(s, sub))
}

func stringContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
