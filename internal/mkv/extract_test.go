package mkv

import (
	"sort"
	"testing"
)

func trackWith(lang, title string, index int) subtitleTrack {
	tags := Tag{Language: lang, Title: title}
	return subtitleTrack{Index: index, Tags: tags}
}

func TestSubtitleExtension(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		track subtitleTrack
		want  string
	}{
		{name: "subrip", track: subtitleTrack{Codec: "subrip"}, want: ".srt"},
		{name: "ass", track: subtitleTrack{Codec: "ass"}, want: ".ass"},
		{name: "ssa", track: subtitleTrack{Codec: "ssa"}, want: ".ass"},
		{name: "vtt", track: subtitleTrack{Codec: "webvtt"}, want: ".vtt"},
		{name: "pgs", track: subtitleTrack{Codec: "hdmv_pgs_subtitle"}, want: ".sup"},
		{name: "dvdsub from tag", track: subtitleTrack{CodecTag: "DVDS"}, want: ".sub"},
		{name: "unknown", track: subtitleTrack{Codec: "unknown"}, want: ".sub"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := subtitleExtension(tc.track)
			if got != tc.want {
				t.Fatalf("subtitleExtension() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSubtitleTrackOrders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b subtitleTrack
		want bool // want a before b
	}{
		{
			name: "eng before non-eng",
			a:    trackWith("eng", "", 1),
			b:    trackWith("spa", "", 0),
			want: true,
		},
		{
			name: "eng non-SDH before eng SDH",
			a:    trackWith("eng", "English", 2),
			b:    trackWith("eng", "English SDH", 1),
			want: true,
		},
		{
			name: "case-insensitive SDH in title",
			a:    trackWith("eng", "english", 0),
			b:    trackWith("eng", "English (sdh)", 0),
			want: true,
		},
		{
			name: "same eng and SDH tie: lower index",
			a:    trackWith("eng", "Same", 1),
			b:    trackWith("eng", "Same", 3),
			want: true,
		},
		{
			name: "non-eng: non-SDH before SDH",
			a:    trackWith("spa", "Español", 5),
			b:    trackWith("spa", "Español SDH", 1),
			want: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := subtitleTrackOrders(tc.a, tc.b); got != tc.want {
				t.Fatalf("subtitleTrackOrders(a, b) = %v, want %v", got, tc.want)
			}
			if rev := subtitleTrackOrders(tc.b, tc.a); tc.want == rev && tc.want {
				t.Fatalf("subtitleTrackOrders(a,b) and (b,a) cannot both be true")
			}
		})
	}
}

func TestSubtitleTrackOrdersSortsSlice(t *testing.T) {
	t.Parallel()
	tracks := []subtitleTrack{
		trackWith("spa", "SDH", 0),
		trackWith("eng", "SDH", 1),
		trackWith("eng", "Plain", 2),
		trackWith("fre", "Plain", 3),
	}
	sort.Slice(tracks, func(i, j int) bool {
		return subtitleTrackOrders(tracks[i], tracks[j])
	})
	// eng first (Plain then SDH), then non-eng without SDH before SDH (fre then spa).
	if trackLanguage(tracks[0]) != "eng" || trackTitle(tracks[0]) != "Plain" {
		t.Fatalf("first track: lang=%q title=%q", trackLanguage(tracks[0]), trackTitle(tracks[0]))
	}
	if trackLanguage(tracks[1]) != "eng" || !titleContainsSDH(trackTitle(tracks[1])) {
		t.Fatalf("second track: want eng SDH")
	}
	if trackLanguage(tracks[2]) != "fre" || titleContainsSDH(trackTitle(tracks[2])) {
		t.Fatalf("third track: want fre non-SDH")
	}
	if trackLanguage(tracks[3]) != "spa" || !titleContainsSDH(trackTitle(tracks[3])) {
		t.Fatalf("fourth track: want spa SDH")
	}
}
