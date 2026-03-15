package loop

import (
	"testing"

	"github.com/lvcoi/melliza/internal/prd"
)

func TestParseReviewVerdict_Pass(t *testing.T) {
	input := `{"verdict":"pass","reasons":[]}`
	v, err := ParseReviewVerdict(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !v.Pass {
		t.Error("expected Pass to be true")
	}
	if len(v.Reasons) != 0 {
		t.Errorf("expected no reasons, got %v", v.Reasons)
	}
}

func TestParseReviewVerdict_Fail(t *testing.T) {
	input := `{"verdict":"fail","reasons":["tests are broken","missing error handling"]}`
	v, err := ParseReviewVerdict(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Pass {
		t.Error("expected Pass to be false")
	}
	if len(v.Reasons) != 2 {
		t.Fatalf("expected 2 reasons, got %d", len(v.Reasons))
	}
	if v.Reasons[0] != "tests are broken" {
		t.Errorf("expected first reason 'tests are broken', got %q", v.Reasons[0])
	}
}

func TestParseReviewVerdict_Invalid(t *testing.T) {
	inputs := []string{
		"not json",
		`{"verdict":"maybe"}`,
		"",
	}
	for _, input := range inputs {
		_, err := ParseReviewVerdict(input)
		if err == nil {
			t.Errorf("expected error for input %q, got nil", input)
		}
	}
}

func TestLoop_FinalizeWithReview_PassSetsPassesTrue(t *testing.T) {
	tmpDir := t.TempDir()
	prdPath := createTestPRD(t, tmpDir, false)

	l := NewLoop(prdPath, "test", 5)
	l.currentStoryID = "US-001"
	// Set a review function that always passes
	l.reviewFunc = func(prdPath, storyID string) (ReviewVerdict, error) {
		return ReviewVerdict{Pass: true}, nil
	}

	err := l.finalizeIteration()
	if err != nil {
		t.Fatalf("finalizeIteration() error: %v", err)
	}

	// Verify story was marked passed
	p, err := prd.LoadPRD(prdPath)
	if err != nil {
		t.Fatalf("LoadPRD() error: %v", err)
	}
	if !p.UserStories[0].Passes {
		t.Error("Expected story to be marked passed after review pass")
	}

	// Verify review lifecycle events were emitted (drain without closing)
	var events []Event
	for len(l.events) > 0 {
		events = append(events, <-l.events)
	}
	if len(events) < 2 {
		t.Fatalf("Expected at least 2 review events, got %d", len(events))
	}
	if events[0].Type != EventReviewStart {
		t.Errorf("Expected first event EventReviewStart, got %v", events[0].Type)
	}
	if events[1].Type != EventReviewPass {
		t.Errorf("Expected second event EventReviewPass, got %v", events[1].Type)
	}
}

func TestLoop_FinalizeWithReview_FailCapturesReasons(t *testing.T) {
	tmpDir := t.TempDir()
	prdPath := createTestPRD(t, tmpDir, false)

	l := NewLoop(prdPath, "test", 5)
	l.currentStoryID = "US-001"
	// Set a review function that rejects
	l.reviewFunc = func(prdPath, storyID string) (ReviewVerdict, error) {
		return ReviewVerdict{
			Pass:    false,
			Reasons: []string{"tests are broken", "missing validation"},
		}, nil
	}

	err := l.finalizeIteration()
	if err != nil {
		t.Fatalf("finalizeIteration() error: %v", err)
	}

	// Story should NOT be marked passed
	p, err := prd.LoadPRD(prdPath)
	if err != nil {
		t.Fatalf("LoadPRD() error: %v", err)
	}
	if p.UserStories[0].Passes {
		t.Error("Expected story to NOT be marked passed after review fail")
	}

	// Rejection reasons should be captured
	reasons := l.LastRejectionReasons()
	if len(reasons) != 2 {
		t.Fatalf("Expected 2 rejection reasons, got %d", len(reasons))
	}
	if reasons[0] != "tests are broken" {
		t.Errorf("Expected first reason 'tests are broken', got %q", reasons[0])
	}

	// Verify review lifecycle events were emitted (drain without closing)
	var events []Event
	for len(l.events) > 0 {
		events = append(events, <-l.events)
	}
	if len(events) < 2 {
		t.Fatalf("Expected at least 2 review events, got %d", len(events))
	}
	if events[0].Type != EventReviewStart {
		t.Errorf("Expected first event EventReviewStart, got %v", events[0].Type)
	}
	if events[1].Type != EventReviewFail {
		t.Errorf("Expected second event EventReviewFail, got %v", events[1].Type)
	}
	if events[1].Text != "tests are broken; missing validation" {
		t.Errorf("Expected fail text 'tests are broken; missing validation', got %q", events[1].Text)
	}
}

func TestParseReviewVerdict_Markdown(t *testing.T) {
	// Review agent may wrap JSON in markdown code fences
	input := "```json\n{\"verdict\":\"pass\",\"reasons\":[]}\n```"
	v, err := ParseReviewVerdict(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !v.Pass {
		t.Error("expected Pass to be true")
	}
}
