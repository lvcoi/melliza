package embed

import (
	"strings"
	"testing"
)

func TestGetPrompt(t *testing.T) {
	prdPath := "/path/to/prd.json"
	progressPath := "/path/to/progress.md"
	storyContext := `{"id":"US-001","title":"Test Story"}`
	prompt := GetPrompt(prdPath, progressPath, storyContext, "US-001", "Test Story")

	// Verify all placeholders were substituted
	if strings.Contains(prompt, "{{PRD_PATH}}") {
		t.Error("Expected {{PRD_PATH}} to be substituted")
	}
	if strings.Contains(prompt, "{{PROGRESS_PATH}}") {
		t.Error("Expected {{PROGRESS_PATH}} to be substituted")
	}
	if strings.Contains(prompt, "{{STORY_CONTEXT}}") {
		t.Error("Expected {{STORY_CONTEXT}} to be substituted")
	}
	if strings.Contains(prompt, "{{STORY_ID}}") {
		t.Error("Expected {{STORY_ID}} to be substituted")
	}
	if strings.Contains(prompt, "{{STORY_TITLE}}") {
		t.Error("Expected {{STORY_TITLE}} to be substituted")
	}

	// Verify the commit message contains the exact story ID and title
	if !strings.Contains(prompt, "feat: US-001 - Test Story") {
		t.Error("Expected prompt to contain exact commit message 'feat: US-001 - Test Story'")
	}

	// PRD path should NOT appear — agent must not modify prd.json
	if strings.Contains(prompt, prdPath) {
		t.Errorf("Expected prompt to NOT contain PRD path %q — agent must not touch prd.json", prdPath)
	}

	// Verify the progress path appears in the prompt
	if !strings.Contains(prompt, progressPath) {
		t.Errorf("Expected prompt to contain progress path %q", progressPath)
	}

	// Verify the story context is inlined in the prompt
	if !strings.Contains(prompt, storyContext) {
		t.Error("Expected prompt to contain inlined story context")
	}

	// Verify the prompt no longer contains self-certification instructions
	if strings.Contains(prompt, "melliza-complete") {
		t.Error("Expected prompt to NOT contain melliza-complete — loop controls completion")
	}

	if !strings.Contains(prompt, "ralph-status") {
		t.Error("Expected prompt to contain ralph-status instruction")
	}

	// The prompt must NOT tell the agent to set passes: true (self-certification removed)
	if strings.Contains(prompt, "passes: true") {
		t.Error("Expected prompt to NOT contain passes: true instruction — agent must not self-certify")
	}
}

func TestGetPrompt_NoSelfCertification(t *testing.T) {
	prompt := GetPrompt("/path/prd.json", "/path/progress.md", `{"id":"US-001"}`, "US-001", "Test Story")

	// The prompt must not tell the agent to update the PRD
	if strings.Contains(prompt, "Update the PRD") {
		t.Error("Expected prompt to NOT contain 'Update the PRD' — agent must not modify prd.json")
	}

	// The prompt must not contain melliza-complete stop condition
	if strings.Contains(prompt, "melliza-complete") {
		t.Error("Expected prompt to NOT contain melliza-complete — loop controls completion, not agent")
	}

	// The prompt must tell the agent to NOT modify prd.json
	if !strings.Contains(prompt, "Do NOT modify prd.json") {
		t.Error("Expected prompt to contain 'Do NOT modify prd.json' instruction")
	}
}

func TestGetPrompt_NoFileReadInstruction(t *testing.T) {
	prompt := GetPrompt("/path/prd.json", "/path/progress.md", `{"id":"US-001"}`, "US-001", "Test Story")

	// The prompt should NOT instruct Gemini to read the PRD file
	if strings.Contains(prompt, "Read the PRD") {
		t.Error("Expected prompt to NOT contain 'Read the PRD' file-read instruction")
	}
}

func TestPromptTemplateNotEmpty(t *testing.T) {
	if promptTemplate == "" {
		t.Error("Expected promptTemplate to be embedded and non-empty")
	}
}

func TestGetPrompt_MellizaExclusion(t *testing.T) {
	prompt := GetPrompt("/path/prd.json", "/path/progress.md", `{"id":"US-001"}`, "US-001", "Test Story")

	// The prompt must instruct Gemini to never stage or commit .melliza/ files
	if !strings.Contains(prompt, ".melliza/") {
		t.Error("Expected prompt to contain .melliza/ exclusion instruction")
	}
	if !strings.Contains(prompt, "NEVER stage or commit") {
		t.Error("Expected prompt to explicitly say NEVER stage or commit .melliza/ files")
	}
	// The commit step should not say "commit ALL changes" anymore
	if strings.Contains(prompt, "commit ALL changes") {
		t.Error("Expected prompt to NOT say 'commit ALL changes' — it should exclude .melliza/ files")
	}
}

func TestGetReviewPrompt(t *testing.T) {
	diff := "diff --git a/main.go b/main.go\n+added line"
	criteria := "- User can log in\n- Password is validated"
	storyTitle := "User Authentication"

	prompt := GetReviewPrompt(diff, criteria, storyTitle)

	if prompt == "" {
		t.Error("Expected GetReviewPrompt() to return non-empty prompt")
	}

	// Verify placeholders were substituted
	if strings.Contains(prompt, "{{DIFF}}") {
		t.Error("Expected {{DIFF}} to be substituted")
	}
	if strings.Contains(prompt, "{{ACCEPTANCE_CRITERIA}}") {
		t.Error("Expected {{ACCEPTANCE_CRITERIA}} to be substituted")
	}
	if strings.Contains(prompt, "{{STORY_TITLE}}") {
		t.Error("Expected {{STORY_TITLE}} to be substituted")
	}

	// Verify content is present
	if !strings.Contains(prompt, diff) {
		t.Error("Expected prompt to contain the diff")
	}
	if !strings.Contains(prompt, criteria) {
		t.Error("Expected prompt to contain the acceptance criteria")
	}
	if !strings.Contains(prompt, storyTitle) {
		t.Error("Expected prompt to contain the story title")
	}

	// Verify it asks for a verdict
	if !strings.Contains(prompt, "verdict") {
		t.Error("Expected prompt to mention 'verdict'")
	}
}

func TestGetConvertPrompt(t *testing.T) {
	prdFilePath := "/path/to/prds/main/prd.md"
	prompt := GetConvertPrompt(prdFilePath, "US")

	// Verify the prompt is not empty
	if prompt == "" {
		t.Error("Expected GetConvertPrompt() to return non-empty prompt")
	}

	// Verify file path is substituted (not inlined content)
	if !strings.Contains(prompt, prdFilePath) {
		t.Error("Expected prompt to contain the PRD file path")
	}
	if strings.Contains(prompt, "{{PRD_FILE_PATH}}") {
		t.Error("Expected {{PRD_FILE_PATH}} to be substituted")
	}

	// Verify the old {{PRD_CONTENT}} placeholder is completely removed
	if strings.Contains(prompt, "{{PRD_CONTENT}}") {
		t.Error("Expected {{PRD_CONTENT}} placeholder to be completely removed")
	}

	// Verify ID prefix is substituted
	if strings.Contains(prompt, "{{ID_PREFIX}}") {
		t.Error("Expected {{ID_PREFIX}} to be substituted")
	}
	if !strings.Contains(prompt, "US-001") {
		t.Error("Expected prompt to contain US-001 when prefix is US")
	}

	// Verify key instructions are present
	if !strings.Contains(prompt, "JSON") {
		t.Error("Expected prompt to mention JSON")
	}

	if !strings.Contains(prompt, "userStories") {
		t.Error("Expected prompt to describe userStories structure")
	}

	if !strings.Contains(prompt, `"passes": false`) {
		t.Error("Expected prompt to specify passes: false default")
	}

	// Verify prompt instructs Gemini to read the file
	if !strings.Contains(prompt, "Read the PRD file") {
		t.Error("Expected prompt to instruct Gemini to read the PRD file")
	}
}

func TestGetConvertPrompt_CustomPrefix(t *testing.T) {
	prompt := GetConvertPrompt("/path/prd.md", "MFR")

	// Verify custom prefix is used, not hardcoded US
	if strings.Contains(prompt, "{{ID_PREFIX}}") {
		t.Error("Expected {{ID_PREFIX}} to be substituted")
	}
	if !strings.Contains(prompt, "MFR-001") {
		t.Error("Expected prompt to contain MFR-001 when prefix is MFR")
	}
	if !strings.Contains(prompt, "MFR-002") {
		t.Error("Expected prompt to contain MFR-002 when prefix is MFR")
	}
}

func TestGetInitPrompt(t *testing.T) {
	prdDir := "/path/to/.melliza/prds/main"

	// Test with no context
	prompt := GetInitPrompt(prdDir, "")
	if !strings.Contains(prompt, "No additional context provided") {
		t.Error("Expected default context message")
	}

	// Verify PRD directory is substituted
	if !strings.Contains(prompt, prdDir) {
		t.Errorf("Expected prompt to contain PRD directory %q", prdDir)
	}
	if strings.Contains(prompt, "{{PRD_DIR}}") {
		t.Error("Expected {{PRD_DIR}} to be substituted")
	}

	// Test with context
	context := "Build a todo app"
	promptWithContext := GetInitPrompt(prdDir, context)
	if !strings.Contains(promptWithContext, context) {
		t.Error("Expected context to be substituted in prompt")
	}
}

func TestGetEditPrompt(t *testing.T) {
	prompt := GetEditPrompt("/test/path/prds/main")
	if prompt == "" {
		t.Error("Expected GetEditPrompt() to return non-empty prompt")
	}
	if !strings.Contains(prompt, "/test/path/prds/main") {
		t.Error("Expected prompt to contain the PRD directory path")
	}
}
