package model

import "time"

type Cue struct {
	Index int
	Start time.Duration
	End   time.Duration
	Lines string
}

type Document struct {
	Format SubtitleFormat
	Header string
	Cues   []*Cue
}

type SubtitleFormat int

const (
	SubtitleFormatUnknown SubtitleFormat = iota
	SubtitleFormatSRT
	SubtitleFormatASS
)
