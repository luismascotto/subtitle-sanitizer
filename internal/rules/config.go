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
	LoadedFromFile                   bool        `json:"loadedFromFile"`
	RemoveTextBeforeColonIfUppercase bool        `json:"removeTextBeforeColonIfUppercase"`
	RemoveTextBeforeColon            bool        `json:"removeTextBeforeColon"`
	RemoveSingleLineColon            bool        `json:"removeSingleLineColon"`
	RemoveBetweenDelimiters          []Delimiter `json:"removeBetweenDelimiters"`
	RemoveLineIfContains             string      `json:"removeLineIfContains"`
}

type Delimiter struct {
	Left  string `json:"left"`
	Right string `json:"right"`
}

// DefaultConfig returns built-in rule defaults when no config file is used.
func DefaultConfig() Config {
	return Config{
		LoadedFromFile:                   false,
		RemoveTextBeforeColonIfUppercase: true,
		RemoveTextBeforeColon:            true,
		RemoveSingleLineColon:            false,
		RemoveBetweenDelimiters: []Delimiter{
			{Left: "(", Right: ")"},
			{Left: "[", Right: "]"},
			{Left: "{", Right: "}"},
			{Left: "*", Right: "*"},
		},
		RemoveLineIfContains: " music *",
	}
}

// ParseConfig unmarshals JSON rule config. On success, LoadedFromFile is set true
// (caller supplied explicit JSON; any loadedFromFile field in the payload is ignored).
func ParseConfig(data []byte) (Config, error) {
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, err
	}
	c.LoadedFromFile = true
	return c, nil
}

// LoadDefaultOrEmpty loads config.json from the current working directory, or returns
// DefaultConfig after logging read/unmarshal errors to stderr.
func LoadDefaultOrEmpty() Config {
	data, err := os.ReadFile("config.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading config:", err)
		return DefaultConfig()
	}
	conf, err := ParseConfig(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error unmarshalling config:", err)
		return DefaultConfig()
	}
	return conf
}

func (c *Config) SaveToBackupFile(jsonData []byte) error {
	err := os.WriteFile("config.backup.json", jsonData, 0644)
	if err != nil {
		return fmt.Errorf("save config backup: %w", err)
	}
	return nil
}
