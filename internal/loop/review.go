package loop

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ReviewVerdict holds the result of a review agent's assessment.
type ReviewVerdict struct {
	Pass    bool     // Whether the iteration's changes are accepted
	Reasons []string // Rejection reasons (empty when Pass is true)
}

// reviewVerdictJSON is the JSON wire format for the review verdict.
type reviewVerdictJSON struct {
	Verdict string   `json:"verdict"`
	Reasons []string `json:"reasons"`
}

// ParseReviewVerdict parses a review verdict from JSON output.
// Accepts raw JSON or markdown-fenced JSON (```json ... ```).
func ParseReviewVerdict(input string) (ReviewVerdict, error) {
	input = strings.TrimSpace(input)

	// Strip markdown code fences if present
	if strings.HasPrefix(input, "```") {
		lines := strings.Split(input, "\n")
		// Remove first and last lines (fences)
		if len(lines) >= 3 {
			input = strings.Join(lines[1:len(lines)-1], "\n")
			input = strings.TrimSpace(input)
		}
	}

	if input == "" {
		return ReviewVerdict{}, fmt.Errorf("empty review verdict")
	}

	var raw reviewVerdictJSON
	if err := json.Unmarshal([]byte(input), &raw); err != nil {
		return ReviewVerdict{}, fmt.Errorf("failed to parse review verdict: %w", err)
	}

	switch raw.Verdict {
	case "pass":
		return ReviewVerdict{Pass: true, Reasons: raw.Reasons}, nil
	case "fail":
		return ReviewVerdict{Pass: false, Reasons: raw.Reasons}, nil
	default:
		return ReviewVerdict{}, fmt.Errorf("invalid verdict %q, expected 'pass' or 'fail'", raw.Verdict)
	}
}

// ReviewFunc is a function that reviews an iteration's changes and returns a verdict.
// The loop calls this function during finalizeIteration when it is set.
type ReviewFunc func(prdPath, storyID string) (ReviewVerdict, error)
