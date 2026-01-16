package subtitle

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/yourname/subtitle-sanitizer/internal/model"
)

// ParseSRT parses minimal, common SRT. Best-effort if ignoreMinorErrors is true.
func ParseSRT(data []byte, ignoreMinorErrors bool) (*model.Document, error) {
	blocks := splitSRTBlocks(data)
	cues := make([]*model.Cue, 0, len(blocks))
	for _, blk := range blocks {
		if len(blk) == 0 {
			continue
		}
		cue, err := parseSRTBlock(blk)
		if err != nil {
			if ignoreMinorErrors {
				continue
			}
			return nil, err
		}
		cues = append(cues, cue)
	}
	// renumber indices from 1..N
	for i := range cues {
		cues[i].Index = i + 1
	}
	return &model.Document{Cues: cues}, nil
}

func splitSRTBlocks(data []byte) [][]string {
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

func parseSRTBlock(lines []string) (*model.Cue, error) {
	if len(lines) < 2 {
		return nil, errors.New("srt block too short")
	}
	// First line is (usually) index
	_, _ = strconv.Atoi(strings.TrimSpace(lines[0])) // ignore parsing errors; some files omit or duplicate
	// Second line is timing
	start, end, err := parseSRTTimingLine(lines[1])
	if err != nil {
		return nil, fmt.Errorf("parse timing: %w", err)
	}
	textLines := append([]string{}, lines[2:]...)
	return &model.Cue{
		Start: start,
		End:   end,
		Lines: textLines,
	}, nil
}

func parseSRTTimingLine(line string) (time.Duration, time.Duration, error) {
	// Example: 00:00:01,234 --> 00:00:04,567
	parts := strings.Split(line, "-->")
	if len(parts) != 2 {
		return 0, 0, errors.New("invalid timing separator")
	}
	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])
	start, err := parseSRTTime(startStr)
	if err != nil {
		return 0, 0, fmt.Errorf("start time: %w", err)
	}
	end, err := parseSRTTime(endStr)
	if err != nil {
		return 0, 0, fmt.Errorf("end time: %w", err)
	}
	return start, end, nil
}

func parseSRTTime(s string) (time.Duration, error) {
	// HH:MM:SS,mmm
	hmsMillis := strings.Split(s, ",")
	if len(hmsMillis) != 2 {
		return 0, errors.New("missing millis")
	}
	hms := strings.Split(hmsMillis[0], ":")
	if len(hms) != 3 {
		return 0, errors.New("invalid h:m:s")
	}
	h, err := strconv.Atoi(hms[0])
	if err != nil {
		return 0, err
	}
	m, err := strconv.Atoi(hms[1])
	if err != nil {
		return 0, err
	}
	si, err := strconv.Atoi(hms[2])
	if err != nil {
		return 0, err
	}
	ms, err := strconv.Atoi(hmsMillis[1])
	if err != nil {
		return 0, err
	}
	total := time.Duration(h)*time.Hour +
		time.Duration(m)*time.Minute +
		time.Duration(si)*time.Second +
		time.Duration(ms)*time.Millisecond
	return total, nil
}

func formatSRTTime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	h := int(d / time.Hour)
	d -= time.Duration(h) * time.Hour
	m := int(d / time.Minute)
	d -= time.Duration(m) * time.Minute
	s := int(d / time.Second)
	d -= time.Duration(s) * time.Second
	ms := int(d / time.Millisecond)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}

// FormatSRT renders a document to SRT and renumbers cues from 1..N.
func FormatSRT(doc model.Document) []byte {
	var buf bytes.Buffer
	index := 1
	for _, cue := range doc.Cues {
		hasText := false
		for _, line := range cue.Lines {
			if strings.TrimSpace(line) != "" {
				hasText = true
				break
			}
		}
		if !hasText {
			continue
		}
		if index > 1 {
			buf.WriteString("\n")
		}
		buf.WriteString(strconv.Itoa(index))
		buf.WriteString("\n")
		buf.WriteString(formatSRTTime(cue.Start))
		buf.WriteString(" --> ")
		buf.WriteString(formatSRTTime(cue.End))
		for _, line := range cue.Lines {
			buf.WriteString("\n")
			buf.WriteString(line)
		}
		buf.WriteString("\n")
		index++
	}
	return buf.Bytes()
}
