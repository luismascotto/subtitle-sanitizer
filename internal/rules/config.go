package rules

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config captures transformation rules.
// RemoveUppercaseColonWords: usually to refer to a specific person or thing that is not appearing in the scene. eg: "GUARD 2: Hey!", "KAREN: Hello!"
// RemoveSingleLineColon: remove any line that ends with ":" and has 3 or fewer words (case-insensitive). eg: "That woman said:", "This is a special release:"
// RemoveBetweenDelimiters: remove text between delimiters. eg: (tyres screeching), [bird chirping]
// RemoveLineIfContains: remove line if it contains the specified text. Used when some subtitles don't follow common rules or patterns. eg: "tense music * (should be [tense music])"
type Config struct {
	LoadedFromFile            bool        `json:"loadedFromFile"`
	RemoveUppercaseColonWords bool        `json:"removeUppercaseColonWords"`
	RemoveSingleLineColon     bool        `json:"removeSingleLineColon"`
	RemoveBetweenDelimiters   []Delimiter `json:"removeBetweenDelimiters"`
	RemoveLineIfContains      string      `json:"removeLineIfContains"`
}

type Delimiter struct {
	Left  string `json:"left"`
	Right string `json:"right"`
}

// LoadDefaultOrEmpty returns default config or loads from local config.json file
func LoadDefaultOrEmpty() Config {
	data, err := os.ReadFile("config.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading config:", err)
		return Config{}
	}
	var conf Config
	err = json.Unmarshal(data, &conf)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error unmarshalling config:", err)
		return Config{}
	}
	conf.LoadedFromFile = true
	return conf
}

func (c *Config) SaveToBackupFile(jsonData []byte) error {
	err := os.WriteFile("config.backup.json", jsonData, 0644)
	if err != nil {
		return fmt.Errorf("save config backup: %w", err)
	}
	return nil
}
