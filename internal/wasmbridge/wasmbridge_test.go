package wasmbridge

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProcess_minimalSRT(t *testing.T) {
	req := `{
		"format": "srt",
		"subtitle": "1\n00:00:01,000 --> 00:00:02,000\nHello (x) world\n\n"
	}`
	out := Process([]byte(req))
	var resp Response
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.OK {
		t.Fatalf("ok=false: %s", resp.Error)
	}
	if !strings.Contains(resp.SRT, "Hello") || strings.Contains(resp.SRT, "(x)") {
		t.Fatalf("unexpected srt: %q", resp.SRT)
	}
	if len(resp.Changes) < 1 {
		t.Fatalf("expected changes, got %+v", resp.Changes)
	}
}

func TestProcess_invalidJSON(t *testing.T) {
	out := Process([]byte(`{`))
	var resp Response
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.OK || resp.Error == "" {
		t.Fatalf("expected error: %+v", resp)
	}
}
