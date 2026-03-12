package tui

import "testing"

// ── Fix 4: ParseQuestions hardening tests ────────────────────────────────────

func TestParseQuestions_ValidQABlock(t *testing.T) {
	text := `1. What framework should we use?
   A. React
   B. Vue
   C. Angular

2. What database do you prefer?
   A. PostgreSQL
   B. MySQL
   C. SQLite`

	questions := ParseQuestions(text)
	if questions == nil {
		t.Fatal("expected questions to be parsed")
	}
	if len(questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(questions))
	}
	// Verify question text is extracted correctly (minus the numbering)
	if questions[0].Text != "What framework should we use?" {
		t.Errorf("unexpected q1 text: %q", questions[0].Text)
	}
	if len(questions[0].Options) != 3 {
		t.Errorf("expected 3 options for q1, got %d", len(questions[0].Options))
	}
	if len(questions[1].Options) != 3 {
		t.Errorf("expected 3 options for q2, got %d", len(questions[1].Options))
	}
}

func TestParseQuestions_SingleQuestion_ReturnsNil(t *testing.T) {
	text := `1. What framework should we use?
   A. React
   B. Vue`

	questions := ParseQuestions(text)
	if questions != nil {
		t.Errorf("expected nil for single question, got %d questions", len(questions))
	}
}

func TestParseQuestions_NumberedListWithoutQuestions(t *testing.T) {
	// This is a numbered implementation plan — no question marks
	text := `1. Set up the project structure
   a. Create the src directory
   b. Initialize package.json

2. Implement the API endpoints
   a. Create the user controller
   b. Add validation middleware

3. Write integration tests
   a. Set up test database
   b. Create test fixtures`

	questions := ParseQuestions(text)
	if questions != nil {
		t.Errorf("expected nil for numbered implementation steps (no question marks), got %d", len(questions))
	}
}

func TestParseQuestions_EmptyText(t *testing.T) {
	questions := ParseQuestions("")
	if questions != nil {
		t.Error("expected nil for empty text")
	}
}

func TestParseQuestions_PlainProse(t *testing.T) {
	text := "This is just regular text without any numbered questions or options."
	questions := ParseQuestions(text)
	if questions != nil {
		t.Error("expected nil for plain prose")
	}
}

func TestParseQuestions_QuestionsWithQuestionMarks(t *testing.T) {
	text := `1. What authentication method should we use?
   A. JWT tokens
   B. Session cookies

2. Should we implement rate limiting?
   A. Yes, with Redis
   B. No, defer to later`

	questions := ParseQuestions(text)
	if questions == nil {
		t.Fatal("expected questions to be parsed for valid Q&A with question marks")
	}
	if len(questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(questions))
	}
}

func TestParseQuestions_MixedContentWithSomeQuestionMarks(t *testing.T) {
	// Only 1 actual question, rest are statements — should return nil
	text := `1. What approach should we take?
   A. Option A
   B. Option B

2. Implement the solution
   a. First step
   b. Second step`

	// The second item has no question mark, so overall count of
	// "questions with ?" is only 1 — should return nil
	questions := ParseQuestions(text)
	if questions != nil {
		t.Errorf("expected nil when only 1 actual question, got %d", len(questions))
	}
}

func TestParseQuestions_MalformedInput(t *testing.T) {
	text := "1. \n   A. "
	questions := ParseQuestions(text)
	if questions != nil {
		t.Error("expected nil for malformed input")
	}
}
