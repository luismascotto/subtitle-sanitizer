package subtitle

import (
	"testing"
	"time"
)

func TestParseASSTime_Variants(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
	}{
		{"0:00:01.87", 1*time.Second + 870*time.Millisecond},  // centiseconds
		{"0:00:01.4", 1*time.Second + 400*time.Millisecond},   // tenths
		{"0:00:01.456", 1*time.Second + 456*time.Millisecond}, // milliseconds (non-standard but robust)
		{" 0:00:00.00 ", 0}, // trim spaces
	}
	for _, tt := range tests {
		got, err := parseASSTime(tt.in)
		if err != nil {
			t.Fatalf("parseASSTime(%q) unexpected error: %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("parseASSTime(%q) = %v; want %v", tt.in, got, tt.want)
		}
	}
}
