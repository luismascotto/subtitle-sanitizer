package sanitize

import (
	"github.com/luismascotto/subtitle-sanitizer/internal/model"
	"github.com/luismascotto/subtitle-sanitizer/internal/rules"
	"github.com/luismascotto/subtitle-sanitizer/internal/subtitle"
	"github.com/luismascotto/subtitle-sanitizer/internal/transform"
)

// Result is the outcome of parsing and applying rules to a subtitle document.
type Result struct {
	Document model.Document
	Changes  []transform.CueChange
	SRT      []byte
}

// Apply runs transforms on an already-parsed document and returns SRT bytes.
func Apply(doc model.Document, conf rules.Config) Result {
	out, changes := transform.ApplyAll(doc, conf)
	return Result{
		Document: out,
		Changes:  changes,
		SRT:      subtitle.FormatSRT(out),
	}
}

// ParseAndApply parses raw subtitle bytes then applies conf.
func ParseAndApply(raw []byte, format model.SubtitleFormat, conf rules.Config) (Result, error) {
	doc, err := subtitle.Parse(raw, format)
	if err != nil {
		return Result{}, err
	}
	return Apply(*doc, conf), nil
}
