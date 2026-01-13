package rules

// Config captures transformation rules. Extendable via JSON in future.
type Config struct {
	RemoveUppercaseColonWords bool        `json:"removeUppercaseColonWords"`
	RemoveSingleLineColon     bool        `json:"removeSingleLineColon"`
	RemoveBetweenDelimiters   []Delimiter `json:"removeBetweenDelimiters"`
	RemoveLineIfContains      string      `json:"removeLineIfContains"`
}

type Delimiter struct {
	Left  string `json:"left"`
	Right string `json:"right"`
}

// LoadDefaultOrEmpty returns default config or loads from a future path (stub).
func LoadDefaultOrEmpty() Config {
	return Config{}
}
