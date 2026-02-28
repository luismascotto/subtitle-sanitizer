package subtitle

import (
	"errors"

	"github.com/luismascotto/subtitle-sanitizer/internal/model"
)

// Parse wrapper for SRT and ASS parsers.
func Parse(data []byte, format model.SubtitleFormat) (*model.Document, error) {
	switch format {
	case model.SubtitleFormatSRT:
		return ParseSRT(data, true)
	case model.SubtitleFormatASS:
		return ParseASS(data)
	default:
		return nil, errors.New("unsupported subtitle format")
	}
}
