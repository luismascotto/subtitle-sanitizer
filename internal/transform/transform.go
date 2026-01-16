package transform

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/yourname/subtitle-sanitizer/internal/model"
	"github.com/yourname/subtitle-sanitizer/internal/rules"
)

var (
	// reBr                  = regexp.MustCompile(`<br />`)
	reSpaces               = regexp.MustCompile(`\s{2,}`)
	reUppercaseColonWords  = regexp.MustCompile(`\b[A-Z]{2,}\:\b`)
	reColonWordsSingleLine = regexp.MustCompile(`(?i)\b[A-Z]{2,}\:\b(?:[A-Za-z]+\s+){0,2}$`)
)

// ApplyAll runs all enabled transformations based on rules.
func ApplyAll(doc model.Document, conf rules.Config) model.Document {
	out := model.Document{
		Cues: []*model.Cue{},
	}
	rulesApplied := []string{}
	ruleTriggered := false

	for _, cue := range doc.Cues {
		newCue := &model.Cue{
			Start: cue.Start,
			End:   cue.End,
			Lines: make([]string, 0),
		}

		for _, line := range cue.Lines {
			text := line
			if conf.RemoveUppercaseColonWords {
				ruleTriggered, text = removeUppercaseColonWords(text)
				if ruleTriggered {
					rulesApplied = append(rulesApplied, "removeUppercaseColonWords")
				}
			}

			if conf.RemoveSingleLineColon {
				ruleTriggered, text = removeSingleLineColon(text)
				if ruleTriggered {
					rulesApplied = append(rulesApplied, "removeSingleLineColon")
				}
			}

			if text != "" && conf.RemoveLineIfContains != "" && strings.Contains(text, conf.RemoveLineIfContains) {
				ruleTriggered = true
				rulesApplied = append(rulesApplied, "removeLineIfContains")
				text = ""
			}

			if text != "" && len(conf.RemoveBetweenDelimiters) > 0 {
				for _, delimiter := range conf.RemoveBetweenDelimiters {
					// Quote delimiter literals to avoid regex meta interpretation.
					left := regexp.QuoteMeta(delimiter.Left)
					right := regexp.QuoteMeta(delimiter.Right)
					// Use a negated character class against the right delimiter (assumed single rune)
					// to avoid greedy cross-boundary removal; replace all occurrences.
					re, err := regexp.Compile(fmt.Sprintf(`%s[^%s]*%s`, left, right, right))
					if err != nil {
						fmt.Println("Error compiling regex:", err, "for delimiter:", delimiter)
						continue
					}
					if re.MatchString(text) {
						ruleTriggered = true
						rulesApplied = append(rulesApplied, "removeBetweenDelimiters"+delimiter.Left+delimiter.Right)
						text = strings.TrimSpace(re.ReplaceAllString(text, ""))
						if text == "" {
							// Skip unnecessary processing
							break
						}
					}
				}
			}

			if text != "" {
				text = normalizeSpaces(text)
				if lineHasAlphabetic(text) {
					newCue.Lines = append(newCue.Lines, text)
				}
			}
		}

		// Add newCue to out.Cues (even with no text) for comparison
		// before accept final result. Empty cues will be dropped on writing to file.

		// // If the cue ends up with no alphabetic content, drop it
		// if len(newCue.Lines) == 0 {
		// 	continue
		// }

		// // Additionally ensure at least one line has a letter (defensive)
		// if !slices.ContainsFunc(newCue.Lines, lineHasAlphabetic) {
		// 	continue
		// }

		out.Cues = append(out.Cues, newCue)
	}

	// Indexing is re-assigned during SRT formatting
	return out
}

func removeUppercaseColonWords(s string) (bool, string) {
	// Remove words of 2+ uppercase letters.
	// Use word boundaries to avoid partial matches. Keep punctuation spacing tidy later.
	if reUppercaseColonWords.MatchString(s) {
		return true, reUppercaseColonWords.ReplaceAllString(s, "")
	}
	return false, s
}

func removeSingleLineColon(s string) (bool, string) {
	// Remove line if ends with a colon, with 3 words os less, case insensitive
	// Use word boundaries to avoid partial matches. Keep punctuation spacing tidy later.
	if reColonWordsSingleLine.MatchString(s) {
		return true, reColonWordsSingleLine.ReplaceAllString(s, "")
	}
	return false, s
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
	return reSpaces.ReplaceAllString(s, " ")
}

func lineHasAlphabetic(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}
