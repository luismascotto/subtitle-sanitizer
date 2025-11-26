package rules

// Config captures transformation rules. Extendable via JSON in future.
type Config struct {
	RemoveUppercaseWords bool `json:"removeUppercaseWords"`
}

// LoadDefaultOrEmpty returns default config or loads from a future path (stub).
func LoadDefaultOrEmpty() Config {
	return Config{}
}
