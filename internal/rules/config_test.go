package rules

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig()
	if c.LoadedFromFile {
		t.Fatal("LoadedFromFile should be false")
	}
	if !c.RemoveTextBeforeColonIfUppercase || !c.RemoveTextBeforeColon {
		t.Fatal("colon rules should be enabled")
	}
	if got, want := c.RemoveLineIfContains, " music *"; got != want {
		t.Fatalf("RemoveLineIfContains = %q, want %q", got, want)
	}
	if len(c.RemoveBetweenDelimiters) != 4 {
		t.Fatalf("delimiters: got %d, want 4", len(c.RemoveBetweenDelimiters))
	}
}

func TestParseConfig(t *testing.T) {
	raw := `{
		"removeTextBeforeColonIfUppercase": false,
		"removeTextBeforeColon": true,
		"removeSingleLineColon": true,
		"removeBetweenDelimiters": [{"left":"(","right":")"}],
		"removeLineIfContains": "x",
		"loadedFromFile": false
	}`
	c, err := ParseConfig([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if !c.LoadedFromFile {
		t.Fatal("ParseConfig should set LoadedFromFile true")
	}
	if c.RemoveTextBeforeColonIfUppercase {
		t.Fatal("expected false from JSON")
	}
	if !c.RemoveSingleLineColon {
		t.Fatal("expected true from JSON")
	}
	if len(c.RemoveBetweenDelimiters) != 1 || c.RemoveBetweenDelimiters[0].Left != "(" {
		t.Fatalf("delimiters: %+v", c.RemoveBetweenDelimiters)
	}
}

func TestParseConfig_invalidJSON(t *testing.T) {
	_, err := ParseConfig([]byte(`{`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseConfig_roundTripDefault(t *testing.T) {
	b, err := json.Marshal(DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	c, err := ParseConfig(b)
	if err != nil {
		t.Fatal(err)
	}
	if !c.LoadedFromFile {
		t.Fatal("LoadedFromFile after ParseConfig")
	}
	// Clear for compare with default semantic (defaults are not "from file" until parsed)
	c.LoadedFromFile = false
	d := DefaultConfig()
	if !reflect.DeepEqual(c, d) {
		t.Fatalf("round trip mismatch:\n%+v\nvs\n%+v", c, d)
	}
}
