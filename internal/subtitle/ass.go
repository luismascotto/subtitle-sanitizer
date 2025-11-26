package subtitle

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/yourname/subtitle-sanitizer/internal/model"
)

// index for acessing splitted ass dialogue line
const (
	assDialogueLineLayerIndex = iota
	assDialogueLineStartIndex
	assDialogueLineEndIndex
	textIndex = 9
)

// ParseASS minimal, only Dialogue Event lines are parsed
func ParseASS(data []byte) (*model.Document, error) {

	blocks := splitASSBlocks(data)
	cues := []*model.Cue{}
	var err error
	for _, blk := range blocks {
		if len(blk) == 0 {
			continue
		}
		// For now, only parse [Events] block
		if strings.TrimSpace(blk[0]) != "[Events]" {
			continue
		}
		cues, err = parseASSEventsBlock(blk)
		if err != nil {
			return nil, err
		}
		break
	}
	if len(cues) == 0 {
		return nil, errors.New("no cues found")
	}
	// renumber indices from 1..N
	for i := range cues {
		cues[i].Index = i + 1
	}
	return &model.Document{Cues: cues}, nil
}

func splitASSBlocks(data []byte) [][]string {
	// Normalize newlines
	s := strings.ReplaceAll(string(data), "\r\n", "\n")
	parts := strings.Split(s, "\n\n")
	out := make([][]string, 0, len(parts))
	for _, p := range parts {
		lines := strings.Split(p, "\n")
		trimmed := make([]string, 0, len(lines))
		for _, l := range lines {
			trimmed = append(trimmed, strings.TrimRight(l, " \t"))
		}
		// Drop leading/trailing empty lines in each block
		for len(trimmed) > 0 && strings.TrimSpace(trimmed[0]) == "" {
			trimmed = trimmed[1:]
		}
		for len(trimmed) > 0 && strings.TrimSpace(trimmed[len(trimmed)-1]) == "" {
			trimmed = trimmed[:len(trimmed)-1]
		}
		if len(trimmed) > 0 {
			out = append(out, trimmed)
		}
	}
	return out
}

func parseASSEventsBlock(blk []string) ([]*model.Cue, error) {
	if len(blk) < 3 {
		return nil, errors.New("events block too short")
	}
	// First line is [Events]
	_, _ = strconv.Atoi(strings.TrimSpace(blk[0])) // ignore parsing errors; some files omit or duplicate
	dialoguesStartIndex := 1
	for dialoguesStartIndex < len(blk) && strings.TrimSpace(blk[dialoguesStartIndex]) != "Dialogue:" {
		dialoguesStartIndex++
	}
	if dialoguesStartIndex >= len(blk) {
		return nil, errors.New("no dialogue lines found")
	}
	cues := make([]*model.Cue, 0, len(blk)-dialoguesStartIndex)
	// Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
	// Dialogue: 0,0:00:04.87,0:00:06.00,Default,,0,0,0,,[Tommy]\NThe president of every bank,
	for _, dialogue := range blk[dialoguesStartIndex:] {
		if strings.TrimSpace(dialogue) == "" {
			continue
		}
		// Split into 10 parts. May exist commas in text, so use SplitN. to dont need to join the text later.
		parts := strings.SplitN(dialogue, ",", 10)
		if len(parts) < 10 {
			return nil, fmt.Errorf("invalid dialogue line: %s", dialogue)
		}
		// Parse timing
		start, err := parseASSTiming(parts[assDialogueLineStartIndex])
		if err != nil {
			return nil, fmt.Errorf("parse start timing: %w", err)
		}
		end, err := parseASSTiming(parts[assDialogueLineEndIndex])
		if err != nil {
			return nil, fmt.Errorf("parse timing: %w", err)
		}
		cueTextLines := strings.Split(parts[textIndex], "\\N")

		cues = append(cues, &model.Cue{
			Start: start,
			End:   end,
			Lines: cueTextLines,
		})
	}
	return cues, nil
}

func parseASSTiming(s string) (time.Duration, error) {
	// Example: 0:00:04.87
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0, errors.New("invalid timing")
	}
	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("parse hours: %w", err)
	}
	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("parse minutes: %w", err)
	}
	seconds, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, fmt.Errorf("parse seconds: %w", err)
	}
	milliseconds, err := strconv.Atoi(parts[3])
	if err != nil {
		return 0, fmt.Errorf("parse milliseconds: %w", err)
	}
	return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second + time.Duration(milliseconds)*time.Millisecond, nil
}
