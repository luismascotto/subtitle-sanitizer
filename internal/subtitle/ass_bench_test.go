package subtitle

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func loadExampleASSTimes(tb testing.TB) []string {
	tb.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "sub-example.ass"))
	if err != nil {
		tb.Fatalf("read sub-example.ass: %v", err)
	}
	var times []string
	for line := range strings.SplitSeq(string(data), "\n") {
		if !strings.HasPrefix(line, "Dialogue:") {
			continue
		}
		// Dialogue: Layer,Start,End,...
		parts := strings.SplitN(line, ",", 4)
		if len(parts) < 3 {
			tb.Fatalf("unexpected dialogue line: %q", line)
		}
		times = append(times, parts[1], parts[2])
	}
	if len(times) == 0 {
		tb.Fatal("no dialogue times found in sub-example.ass")
	}
	return times
}

// parseASSTimeSplit is the pre-Cut baseline kept only for alloc regression benches.
func parseASSTimeSplit(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	hmsMillis := strings.Split(s, ".")
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
	frac := strings.TrimSpace(hmsMillis[1])
	if len(frac) > 3 {
		frac = frac[:3]
	}
	msMultiplier := 1
	switch len(frac) {
	case 1:
		msMultiplier = 100
	case 2:
		msMultiplier = 10
	case 3:
		msMultiplier = 1
	default:
		frac = "0"
	}
	msVal, err := strconv.Atoi(frac)
	if err != nil {
		return 0, err
	}
	ms := msVal * msMultiplier
	return time.Duration(h)*time.Hour +
		time.Duration(m)*time.Minute +
		time.Duration(si)*time.Second +
		time.Duration(ms)*time.Millisecond, nil
}

func TestParseASSTime_MatchesSplitBaseline(t *testing.T) {
	times := loadExampleASSTimes(t)
	for _, in := range times {
		want, err := parseASSTimeSplit(in)
		if err != nil {
			t.Fatalf("parseASSTimeSplit(%q): %v", in, err)
		}
		got, err := parseASSTime(in)
		if err != nil {
			t.Fatalf("parseASSTime(%q): %v", in, err)
		}
		if got != want {
			t.Fatalf("parseASSTime(%q)=%v want %v", in, got, want)
		}
	}
}

func BenchmarkParseASSTime_subExample(b *testing.B) {
	times := loadExampleASSTimes(b)
	parsers := []struct {
		name string
		fn   func(string) (time.Duration, error)
	}{
		{"Split_baseline", parseASSTimeSplit},
		{"Cut", parseASSTime},
	}
	for _, p := range parsers {
		b.Run(p.name, func(b *testing.B) {
			b.ReportAllocs()
			var sink time.Duration
			for b.Loop() {
				for _, in := range times {
					d, err := p.fn(in)
					if err != nil {
						b.Fatal(err)
					}
					sink += d
				}
			}
			_ = sink
		})
	}
}

func BenchmarkParseASS_subExample(b *testing.B) {
	data, err := os.ReadFile(filepath.Join("..", "..", "sub-example.ass"))
	if err != nil {
		b.Fatalf("read sub-example.ass: %v", err)
	}
	b.ReportAllocs()
	for b.Loop() {
		doc, err := ParseASS(data)
		if err != nil {
			b.Fatal(err)
		}
		if len(doc.Cues) == 0 {
			b.Fatal("no cues")
		}
	}
}
