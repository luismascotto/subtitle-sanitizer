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
	reSpaces              = regexp.MustCompile(`\s{2,}`)
	reUppercaseColonWords = regexp.MustCompile(`\b[A-Z]{2,}:[ \t]*`)
)

// ApplyAll runs all enabled transformations based on rules.
func ApplyAll(doc model.Document, conf rules.Config, fromASS bool) model.Document {
	out := model.Document{
		Cues: []*model.Cue{},
	}

	for _, cue := range doc.Cues {
		rulesApplied := []string{}
		ruleTriggered := false

		newCue := &model.Cue{
			Index: cue.Index,
			Start: cue.Start,
			End:   cue.End,
			Lines: "",
		}

		text := cue.Lines

		if fromASS {
			text = convertASSFormattingToSRT(text)
		}

		if conf.RemoveLineIfContains != "" && strings.Contains(text, conf.RemoveLineIfContains) {
			ruleTriggered = true
			rulesApplied = append(rulesApplied, "removeLineIfContains")
			// Skip processing this line
			continue
		}

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

		if text != "" && len(conf.RemoveBetweenDelimiters) > 0 {
			for _, delimiter := range conf.RemoveBetweenDelimiters {
				controlEscape := ""
				if delimiter.Left == "{" {
					// ASS format uses curly braces for formatting (italic, bold, etc.), {\i1}Text{\i0}
					controlEscape = "\\"
				}
				minContentLen := 0
				if delimiter.Left == "<" {
					// SRT format uses angle brackets for formatting (italic, bold, etc.), <i>Text</i>
					// also <font xxx>Text</font>
					minContentLen = 3
					controlEscape = "/="
				}
				// Quote delimiter literals to avoid regex meta interpretation.
				left := regexp.QuoteMeta(delimiter.Left)
				right := regexp.QuoteMeta(delimiter.Right)
				// Use a negated character class against the right delimiter (assumed single rune)
				// to avoid greedy cross-boundary removal; replace all occurrences.
				re, err := regexp.Compile(fmt.Sprintf(`%s[^%s%s]{%d,}%s`, left, controlEscape, right, minContentLen, right))
				if err != nil {
					fmt.Println("Error compiling regex:", err, "for delimiter:", delimiter)
					continue
				}
				if re.MatchString(text) {
					ruleTriggered = true
					rulesApplied = append(rulesApplied, "removeBetweenDelimiters"+delimiter.Left+delimiter.Right)
					text = strings.TrimSpace(re.ReplaceAllString(text, ""))
					if text == "" {
						// Skip unnecessary RemoveBetweenDelimiters rule processing
						break
					}
				}
			}
		}

		if text != "" {
			text = strings.TrimSuffix(text, "\n")
			text = strings.TrimPrefix(text, "\n")
			text = strings.TrimSpace(collapseSpaces(text))
			if lineHasAlphabetic(text) {
				newCue.Lines = text
			}
		}

		if len(rulesApplied) > 0 {
			//Print cue index, original text and transformed text
			fmt.Printf("Cue %d: %s -> %s \t%v\n",
				cue.Index,
				strings.ReplaceAll(cue.Lines, "\n", " \\n "),
				strings.ReplaceAll(newCue.Lines, "\n", " \\n "),
				rulesApplied)
		}

		out.Cues = append(out.Cues, newCue)
	}

	// Indexing is re-assigned during SRT formatting
	return out
}

func removeUppercaseColonWords(s string) (bool, string) {
	// Remove words of 2+ uppercase letters.
	// Use word boundaries to avoid partial matches. Keep punctuation spacing tidy later.
	if len(s) > 0 && reUppercaseColonWords.MatchString(s) {
		return true, reUppercaseColonWords.ReplaceAllString(s, "")
	}
	return false, s
}

func removeSingleLineColon(s string) (bool, string) {
	// Remove any line that ends with ":" and has 3 or fewer words (case-insensitive)
	if len(s) == 0 {
		return false, s
	}
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	removed := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasSuffix(trimmed, ":") {
			withoutColon := strings.TrimSpace(strings.TrimSuffix(trimmed, ":"))
			// Count words using whitespace splitting
			words := strings.Fields(withoutColon)
			if len(words) > 0 && len(words) <= 3 {
				removed = true
				continue
			}
		}
		out = append(out, line)
	}
	return removed, strings.Join(out, "\n")
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

func convertASSFormattingToSRT(s string) string {
	// Convert ASS formatting to SRT formatting
	// ASS format uses curly braces for formatting (italic, bold, etc.), {\i1}Text{\i0}
	// SRT format uses angle brackets for formatting (italic, bold, etc.), <i>Text</i>
	// {\X1..N} -> <X>
	// {\X0} -> </X>
	// X -> b, i, u, s

	//Ignore other formatting

	// Opening tags: {\i1}, {\b2}, etc. (any non-zero digit(s))
	open := regexp.MustCompile(`{\\([bius])[1-9]\d*}`)
	formatted := open.ReplaceAllString(s, "<$1>")
	// Closing tags: {\i0}, {\b0}, etc.
	close := regexp.MustCompile(`{\\([bius])0}`)
	formatted = close.ReplaceAllString(formatted, "</$1>")

	ignore := regexp.MustCompile(`{\\[^bius][^}]*}`)
	formatted = ignore.ReplaceAllString(formatted, "")
	return formatted
}
