package gemini

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// geminiSettings represents the relevant fields of ~/.gemini/settings.json.
type geminiSettings struct {
	Security struct {
		EnableConseca bool `json:"enableConseca"`
	} `json:"security"`
}

// IsConsecaEnabled reads ~/.gemini/settings.json and checks if Conseca is enabled.
// Returns false if the file doesn't exist, can't be read, or can't be parsed.
func IsConsecaEnabled() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	data, err := os.ReadFile(filepath.Join(home, ".gemini", "settings.json"))
	if err != nil {
		return false
	}

	var settings geminiSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return false
	}

	return settings.Security.EnableConseca
}
