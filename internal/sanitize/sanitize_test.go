package sanitize

import (
	"testing"

	"github.com/luismascotto/subtitle-sanitizer/internal/model"
	"github.com/luismascotto/subtitle-sanitizer/internal/rules"
)

func TestParseAndApply_srt(t *testing.T) {
	raw := []byte(`1
00:00:01,000 --> 00:00:02,000
Hello (x) world

`)
	conf := rules.Config{
		RemoveBetweenDelimiters: []rules.Delimiter{{Left: "(", Right: ")"}},
	}
	res, err := ParseAndApply(raw, model.SubtitleFormatSRT, conf)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Changes) < 1 {
		t.Fatal("expected at least one cue change")
	}
	if len(res.SRT) == 0 {
		t.Fatal("expected SRT output")
	}
}

func TestApply_roundTripCue(t *testing.T) {
	doc := model.Document{
		Format: model.SubtitleFormatSRT,
		Cues: []*model.Cue{
			{Index: 1, Start: 0, End: 1e9, Lines: "plain"},
		},
	}
	res := Apply(doc, rules.Config{})
	if len(res.Document.Cues) != 1 {
		t.Fatalf("cues: %d", len(res.Document.Cues))
	}
}
