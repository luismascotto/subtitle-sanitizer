package wasmbridge

import (
	"encoding/json"
	"os"
	"path/filepath"
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

// TestProcess_fullAssGolden matches sub-example.ass + config.test.json → *-his-expected.srt (WASM JSON contract).
func TestProcess_fullAssGolden(t *testing.T) {
	dir := filepath.Join("testdata", "full_ass")
	ass, err := os.ReadFile(filepath.Join(dir, "sub-example.ass"))
	if err != nil {
		t.Fatal(err)
	}
	cfgBytes, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(dir, "expected.srt"))
	if err != nil {
		t.Fatal(err)
	}

	body, err := json.Marshal(struct {
		Format   string          `json:"format"`
		Subtitle string          `json:"subtitle"`
		Config   json.RawMessage `json:"config"`
	}{
		Format:   "ass",
		Subtitle: string(ass),
		Config:   json.RawMessage(strings.TrimSpace(string(cfgBytes))),
	})
	if err != nil {
		t.Fatal(err)
	}

	out := Process(body)
	var resp Response
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.OK {
		t.Fatalf("ok=false: %s", resp.Error)
	}
	if normalizeEOL([]byte(resp.SRT)) != normalizeEOL(want) {
		t.Fatalf("SRT mismatch (first 500 chars got/want):\n--- got ---\n%s\n--- want ---\n%s",
			trunc(resp.SRT, 500), trunc(string(want), 500))
	}
}

func normalizeEOL(b []byte) string {
	s := strings.ReplaceAll(string(b), "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
