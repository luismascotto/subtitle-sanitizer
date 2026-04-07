package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Config captures transformation rules.
// RemoveSingleLineColon: remove any line that ends with ":" and has 3 or fewer words (case-insensitive). eg: "That woman said:", "This is a special release:"
// RemoveTextBeforeColon(if uppercase explicitly specified): usually to refer to a specific person or thing that is not appearing in the scene. eg: "Father: Hi son!", "GUARD 2: Hey!", "KAREN: Hello!"
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

// DescribeEffective returns a readable summary of the active rules (for CLI preview).
func (c Config) DescribeEffective() string {
	var b strings.Builder
	fmt.Fprintf(&b, "removeTextBeforeColonIfUppercase: %t\n", c.RemoveTextBeforeColonIfUppercase)
	fmt.Fprintf(&b, "removeTextBeforeColon: %t\n", c.RemoveTextBeforeColon)
	fmt.Fprintf(&b, "removeSingleLineColon: %t\n", c.RemoveSingleLineColon)
	b.WriteString("removeBetweenDelimiters:\n")
	if len(c.RemoveBetweenDelimiters) == 0 {
		b.WriteString("  (none)\n")
	} else {
		for _, d := range c.RemoveBetweenDelimiters {
			fmt.Fprintf(&b, "  - left=%q right=%q\n", d.Left, d.Right)
		}
	}
	if c.RemoveLineIfContains != "" {
		fmt.Fprintf(&b, "removeLineIfContains: %q\n", c.RemoveLineIfContains)
	} else {
		b.WriteString("removeLineIfContains: (empty; disabled)\n")
	}
	if c.LoadedFromFile {
		b.WriteString("\nsource: config.json\n")
	} else {
		b.WriteString("\nsource: built-in defaults\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (c *Config) SaveToBackupFile(jsonData []byte) error {
	err := os.WriteFile("config.backup.json", jsonData, 0644)
	if err != nil {
		return fmt.Errorf("save config backup: %w", err)
	}
	return nil
}

// Rules description/enumerator
type AbbreviatedRuleDescription string

const (
	RuleRemoveTextBeforeColonIfUppercase AbbreviatedRuleDescription = "TEXT:"
	RuleRemoveTextBeforeColon            AbbreviatedRuleDescription = "Text:"
	RuleRemoveSingleLineColon            AbbreviatedRuleDescription = "[Line]:"
	RuleRemoveBetweenDelimiters          AbbreviatedRuleDescription = "\\ Delims /"
	RuleRemoveLineIfContains             AbbreviatedRuleDescription = "%Contains%"
)
