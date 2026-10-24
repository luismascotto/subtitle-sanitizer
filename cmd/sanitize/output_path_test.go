package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/luismascotto/subtitle-sanitizer/internal/model"
)

func TestResolveOutputPath_ExplicitWins(t *testing.T) {
	got := resolveOutputPath(`C:\in\show.srt`, `D:\out\clean.srt`, true)
	want := `D:\out\clean.srt`
	if got != want {
		t.Fatalf("resolveOutputPath = %q, want %q", got, want)
	}
}

func TestResolveOutputPath_EmptyFallsBackToDeriveOverwrite(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "show.srt")
	if err := os.WriteFile(input, []byte("1\n00:00:01,000 --> 00:00:02,000\nHi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := resolveOutputPath(input, "", true)
	if got != input {
		t.Fatalf("overwrite fallback = %q, want input %q", got, input)
	}
}

func TestApplyTransformations_WritesExplicitOutputWithoutTouchingInput(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "raw.srt")
	output := filepath.Join(dir, "clean.srt")
	original := "1\n00:00:01,000 --> 00:00:02,000\nHELLO WORLD\n"
	if err := os.WriteFile(input, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	doc := &model.Document{
		Format: model.SubtitleFormatSRT,
		Cues: []*model.Cue{
			{Index: 1, Start: time.Second, End: 2 * time.Second, Lines: "hello world"},
		},
	}

	if err := ApplyTransformations(input, doc, true, true, output); err != nil {
		t.Fatalf("ApplyTransformations: %v", err)
	}

	inBytes, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	if string(inBytes) != original {
		t.Fatalf("input was modified:\n got %q\nwant %q", string(inBytes), original)
	}

	outBytes, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(outBytes), "hello world") {
		t.Fatalf("output missing transformed text: %q", string(outBytes))
	}
	if strings.Contains(string(outBytes), "HELLO WORLD") {
		t.Fatalf("output still has original text: %q", string(outBytes))
	}
}
