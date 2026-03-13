package tui

import (
	"testing"
)

func TestSetupSpinner_Configure(t *testing.T) {
	s := NewSetupSpinner()
	s.Configure("my-prd")

	if s.prdName != "my-prd" {
		t.Errorf("Expected prdName %q, got %q", "my-prd", s.prdName)
	}

	if s.IsDone() {
		t.Error("Expected spinner to not be done immediately after configure")
	}

	if s.currentStep != SetupStepScan {
		t.Errorf("Expected initial step SetupStepScan, got %v", s.currentStep)
	}
}

func TestSetupSpinner_Advance(t *testing.T) {
	s := NewSetupSpinner()
	s.Configure("my-prd")

	// Should be at scan step
	if s.currentStep != SetupStepScan {
		t.Errorf("Expected SetupStepScan, got %v", s.currentStep)
	}

	// Advance to generate
	s.AdvanceStep()
	if s.currentStep != SetupStepGenerate {
		t.Errorf("Expected SetupStepGenerate, got %v", s.currentStep)
	}
	if s.IsDone() {
		t.Error("Expected not done after advancing to generate")
	}

	// Advance to done
	s.AdvanceStep()
	if s.currentStep != SetupStepDone {
		t.Errorf("Expected SetupStepDone, got %v", s.currentStep)
	}
	if !s.IsDone() {
		t.Error("Expected done after advancing to done step")
	}
}

func TestSetupSpinner_SetError(t *testing.T) {
	s := NewSetupSpinner()
	s.Configure("my-prd")

	s.SetError("detection failed")
	if !s.HasError() {
		t.Error("Expected HasError() to return true")
	}
	if s.errMsg != "detection failed" {
		t.Errorf("Expected errMsg %q, got %q", "detection failed", s.errMsg)
	}
}

func TestAppState_SetupString(t *testing.T) {
	if StateSetup.String() != "Setup" {
		t.Errorf("Expected StateSetup.String() = %q, got %q", "Setup", StateSetup.String())
	}
}

func TestSetupSpinner_Render(t *testing.T) {
	s := NewSetupSpinner()
	s.Configure("my-prd")
	s.SetSize(80, 24)

	output := s.Render()
	if output == "" {
		t.Error("Expected non-empty render output")
	}
}
