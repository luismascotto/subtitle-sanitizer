package transform

import (
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/yourname/subtitle-sanitizer/internal/model"
	"github.com/yourname/subtitle-sanitizer/internal/rules"
)

// ApplyAll runs all enabled transformations based on rules.
func ApplyAll(doc model.Document, conf rules.Config) model.Document {
	out := model.Document{
		Cues: []*model.Cue{},
	}

	for _, cue := range doc.Cues {
		newCue := &model.Cue{
			Start: cue.Start,
			End:   cue.End,
			Lines: make([]string, 0),
		}

		for _, line := range cue.Lines {
			text := line
			if conf.RemoveUppercaseColonWords {
				text = removeUppercaseColonWords(text)
			}
			text = normalizeSpaces(text)
			if lineHasAlphabetic(text) {
				newCue.Lines = append(newCue.Lines, text)
			}
		}

		// If the cue ends up with no alphabetic content, drop it
		if len(newCue.Lines) == 0 {
			continue
		}

		// Additionally ensure at least one line has a letter (defensive)
		if !slices.ContainsFunc(newCue.Lines, lineHasAlphabetic) {
			continue
		}

		out.Cues = append(out.Cues, newCue)
	}

	// Indexing is re-assigned during SRT formatting
	return out
}

func removeUppercaseColonWords(s string) string {
	// Remove words of 2+ uppercase letters.
	// Use word boundaries to avoid partial matches. Keep punctuation spacing tidy later.
	re := regexp.MustCompile(`\b[A-Z]{2,}\:\b`)
	return re.ReplaceAllString(s, "")
}

func normalizeSpaces(s string) string {
	// Collapse multiple spaces; trim. Maintain <br /> intact.
	// Strategy: split preserving <br /> tokens.
	const br = "<br />"
	if !strings.Contains(s, br) {
		return strings.TrimSpace(collapseSpaces(s))
	}
	parts := strings.Split(s, br)
	for i := range parts {
		parts[i] = collapseSpaces(strings.TrimSpace(parts[i]))
	}
	joined := strings.Join(parts, " "+br+" ")
	return strings.TrimSpace(collapseSpaces(joined))
}

func collapseSpaces(s string) string {
	re := regexp.MustCompile(`\s{2,}`)
	return re.ReplaceAllString(s, " ")
}

func lineHasAlphabetic(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}
