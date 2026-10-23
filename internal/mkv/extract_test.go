package mkv

import (
	"sort"
	"testing"
)

func trackWith(lang, title string, index int) subtitleTrack {
	tags := Tag{Language: lang, Title: title}
	return subtitleTrack{Index: index, Tags: tags}
}

func trackWithDisp(lang, title string, index int, forced, hearingImpaired int) subtitleTrack {
	t := trackWith(lang, title, index)
	t.Dispositions = Disposition{Forced: forced, HearingImpaired: hearingImpaired}
	return t
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
		{name: "unknown", track: subtitleTrack{Codec: "unknown"}, want: ""},
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
			name: "non-eng same rank: lower index",
			a:    trackWith("spa", "Español SDH", 1),
			b:    trackWith("spa", "Español", 5),
			want: true,
		},
		{
			name: "eng SDH before eng Forced (title)",
			a:    trackWith("eng", "English SDH", 2),
			b:    trackWith("eng", "English Forced", 1),
			want: true,
		},
		{
			name: "eng SDH before eng Forced (disposition)",
			a:    trackWithDisp("eng", "English SDH", 2, 0, 0),
			b:    trackWithDisp("eng", "English", 1, 1, 0),
			want: true,
		},
		{
			name: "eng puro before eng SDH",
			a:    trackWith("eng", "English", 1),
			b:    trackWith("eng", "English SDH", 0),
			want: true,
		},
		{
			name: "non-eng non-forced before any Forced",
			a:    trackWith("spa", "Español", 5),
			b:    trackWith("eng", "English Forced", 1),
			want: true,
		},
		{
			name: "Forced+SDH still ranks as Forced (after eng SDH)",
			a:    trackWith("eng", "English SDH", 2),
			b:    trackWith("eng", "English SDH Forced", 1),
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
		trackWith("eng", "English Forced", 1),
		trackWith("eng", "English SDH", 2),
		trackWith("spa", "Español", 3),
		trackWith("eng", "English", 4),
	}
	sort.Slice(tracks, func(i, j int) bool {
		return subtitleTrackOrders(tracks[i], tracks[j])
	})
	// 0 eng puro, 1 eng SDH, 2 other non-forced, 3 any Forced
	want := []struct{ lang, title string }{
		{"eng", "English"},
		{"eng", "English SDH"},
		{"spa", "Español"},
		{"eng", "English Forced"},
	}
	for i, w := range want {
		if trackLanguage(tracks[i]) != w.lang || trackTitle(tracks[i]) != w.title {
			t.Fatalf("tracks[%d]: lang=%q title=%q, want lang=%q title=%q",
				i, trackLanguage(tracks[i]), trackTitle(tracks[i]), w.lang, w.title)
		}
	}
}
