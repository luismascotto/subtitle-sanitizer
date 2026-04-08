// Package wasmbridge implements the JSON request/response contract used by cmd/wasm and cmd/tinywasm.
// JSON shapes are described under wasm/schema/ for OpenAPI-adjacent tooling.
package wasmbridge

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/luismascotto/subtitle-sanitizer/internal/model"
	"github.com/luismascotto/subtitle-sanitizer/internal/rules"
	"github.com/luismascotto/subtitle-sanitizer/internal/sanitize"
	"github.com/luismascotto/subtitle-sanitizer/internal/transform"
)

const marshallEror string = `{"ok":false,"error":"marshal response failed"}`

// Request is the JSON body consumed by [Process].
type Request struct {
	Subtitle    string `json:"subtitle"`
	SubtitleB64 string `json:"subtitleB64"`
	//Format      string          `json:"format"`
	Config json.RawMessage `json:"config"`
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
		return mustJSONErrStr(fmt.Sprintf("json: %v", err))
	}
	var (
		err    error
		raw    []byte
		format model.SubtitleFormat
		conf   rules.Config
		res    sanitize.Result
	)

	if raw, err = subtitleBytes(&req); err != nil {
		return mustJSONErr(err)
	}

	if format, err = parseFormat(req.Subtitle[:min(1024, len(raw))]); err != nil {
		return mustJSONErr(err)
	}

	if conf, err = configFromJSON(req.Config); err != nil {
		return mustJSONErr(err)
	}

	if res, err = sanitize.ParseAndApply(raw, format, conf); err != nil {
		return mustJSONErr(err)
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

func parseFormat(peekSubtitle string) (model.SubtitleFormat, error) {
	//Happy path first no allocation (almost)
	//check on body for ASS
	if strings.HasPrefix(peekSubtitle, "[Script Info]") {
		return model.SubtitleFormatASS, nil
	}

	ix := strings.Index(peekSubtitle, "[Events]")
	if ix > 13 && strings.Index(peekSubtitle, "Dialogue: ") > ix {
		return model.SubtitleFormatASS, nil
	}
	//Assume SRT?
	//SplitN 5 => up to 5 substrings from sep.
	// SRT is index \n 00:00:02,136 --> 00:00:04,238 \n some text \n NewLine \n index2....
	// Assuming a few possible nemlines empty on beginning and up to 3 lines,
	// I need 9 splitteds X,X,1,00:,text,text,text,empty,[the_rest]

	splittedN := strings.SplitN(peekSubtitle, "\n", 9)
	ix = 0
	for i := range splittedN {
		if len(splittedN[i]) > 0 {
			ix += 1
			// Two first columns to convert to int of find 00:..
			if ix <= 2 {
				if safeInt(splittedN[i]) > 0 {
					return model.SubtitleFormatSRT, nil
				}
				if strings.HasPrefix(strings.TrimSpace(splittedN[i]), "00:") {
					return model.SubtitleFormatSRT, nil
				}
			} else {
				break
			}
		}
	}
	return model.SubtitleFormatUnknown, nil
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
		return []byte(marshallEror)
	}
	return b
}

func mustJSONErrStr(s string) []byte {
	return mustJSON(Response{OK: false, Error: s})
}
func mustJSONErr(e error) []byte {
	return mustJSONErrStr(e.Error())
}

func safeInt(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}
