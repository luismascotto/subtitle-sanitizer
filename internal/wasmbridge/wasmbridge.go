package wasmbridge

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/luismascotto/subtitle-sanitizer/internal/model"
	"github.com/luismascotto/subtitle-sanitizer/internal/rules"
	"github.com/luismascotto/subtitle-sanitizer/internal/sanitize"
	"github.com/luismascotto/subtitle-sanitizer/internal/transform"
)

// Request is the JSON body consumed by [Process].
type Request struct {
	Subtitle    string          `json:"subtitle"`
	SubtitleB64 string          `json:"subtitleB64"`
	Format      string          `json:"format"`
	Config      json.RawMessage `json:"config"`
}

// Response is the JSON returned by [Process].
type Response struct {
	OK      bool                  `json:"ok"`
	Error   string                `json:"error,omitempty"`
	SRT     string                `json:"srt,omitempty"`
	Changes []transform.CueChange `json:"changes,omitempty"`
}

// Process runs parse + sanitize from JSON bytes and returns JSON (always valid on best effort).
func Process(body []byte) []byte {
	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		return mustJSON(Response{OK: false, Error: fmt.Sprintf("json: %v", err)})
	}

	raw, err := subtitleBytes(&req)
	if err != nil {
		return mustJSON(Response{OK: false, Error: err.Error()})
	}

	format, err := parseFormat(req.Format)
	if err != nil {
		return mustJSON(Response{OK: false, Error: err.Error()})
	}

	conf, err := configFromJSON(req.Config)
	if err != nil {
		return mustJSON(Response{OK: false, Error: err.Error()})
	}

	res, err := sanitize.ParseAndApply(raw, format, conf)
	if err != nil {
		return mustJSON(Response{OK: false, Error: err.Error()})
	}

	out := Response{
		OK:      true,
		SRT:     string(res.SRT),
		Changes: res.Changes,
	}
	return mustJSON(out)
}

func subtitleBytes(req *Request) ([]byte, error) {
	if req.SubtitleB64 != "" {
		return base64.StdEncoding.DecodeString(strings.TrimSpace(req.SubtitleB64))
	}
	if req.Subtitle != "" {
		return []byte(req.Subtitle), nil
	}
	return nil, fmt.Errorf("subtitle or subtitleB64 is required")
}

func parseFormat(s string) (model.SubtitleFormat, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "srt":
		return model.SubtitleFormatSRT, nil
	case "ass":
		return model.SubtitleFormatASS, nil
	case "":
		return 0, fmt.Errorf("format is required (srt or ass)")
	default:
		return 0, fmt.Errorf("format must be srt or ass, got %q", s)
	}
}

func configFromJSON(raw json.RawMessage) (rules.Config, error) {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" || s == "{}" {
		return rules.DefaultConfig(), nil
	}
	return rules.ParseConfig([]byte(s))
}

func mustJSON(v Response) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{"ok":false,"error":"marshal response failed"}`)
	}
	return b
}
