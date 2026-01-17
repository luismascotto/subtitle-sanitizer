package model

import "time"

type Cue struct {
	Index int
	Start time.Duration
	End   time.Duration
	Lines string
}

type Document struct {
	Cues []*Cue
}
