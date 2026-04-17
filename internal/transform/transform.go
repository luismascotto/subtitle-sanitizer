package transform

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/luismascotto/subtitle-sanitizer/internal/model"
	"github.com/luismascotto/subtitle-sanitizer/internal/rules"
)

var (
	// reBr                  = regexp.MustCompile(`<br />`)
	reSpaces = regexp.MustCompile(`\s{2,}`)
	//reUppercaseColonWords    = regexp.MustCompile(`\b[A-Z]{1,}\s*[A-Z0-9]{1,}:[ \t]*`)
	reTextWithColon          = regexp.MustCompile(`^[^:]+:[ \t]*`)
	reUppercaseTextWithColon = regexp.MustCompile(`^[^:a-z]*[A-Z][^:a-z]*:[ \t]*`)
)

// CueChange records one cue that had at least one rule applied (including full-line removal).
type CueChange struct {
	CueIndex    int      `json:"cueIndex"`
	Original    string   `json:"original"`
	Transformed string   `json:"transformed"`
	Rules       []string `json:"rules"`
}

// ApplyAll runs all enabled transformations based on rules.
func ApplyAll(doc model.Document, conf rules.Config) (model.Document, []CueChange) {
	out := model.Document{
		Format: doc.Format,
		Header: doc.Header,
		Cues:   []*model.Cue{},
	}
	var changes []CueChange
	for _, cue := range doc.Cues {
		rulesApplied := []string{}
		ruleTriggered := false

		text := cue.Lines

		if doc.Format == model.SubtitleFormatASS {
			text = convertASSFormattingToSRT(text)
		}

		newCue := &model.Cue{
			Index: cue.Index,
			Start: cue.Start,
			End:   cue.End,
			Lines: text,
		}

		if conf.RemoveLineIfContains != "" {
			lstRemoveLineIfContains := strings.SplitSeq(conf.RemoveLineIfContains, "\n")
			for removeLineIfContains := range lstRemoveLineIfContains {
				if strings.Contains(text, removeLineIfContains) {
					text = ""
					rulesApplied = append(rulesApplied, string(rules.RuleRemoveLineIfContains))
					break
				}
			}
		}
		if text != "" {
			if conf.RemoveSingleLineColon {
				ruleTriggered, text = removeSingleLineColon(text)
				if ruleTriggered {
					rulesApplied = append(rulesApplied, string(rules.RuleRemoveSingleLineColon))
				}
			}

			if conf.RemoveTextBeforeColonIfUppercase {
				ruleTriggered, text = removeUppercaseTextWithColon(text)
				if ruleTriggered {
					rulesApplied = append(rulesApplied, string(rules.RuleRemoveTextBeforeColonIfUppercase))
				}
			} else if conf.RemoveTextBeforeColon {
				ruleTriggered, text = removeTextBeforeColon(text)
				if ruleTriggered {
					rulesApplied = append(rulesApplied, string(rules.RuleRemoveTextBeforeColon))
				}
			}
		}

		if text != "" && len(conf.RemoveBetweenDelimiters) > 0 {
			text, rulesApplied = removeTextBetweenDelimiters(text, conf.RemoveBetweenDelimiters, rulesApplied)
		}

		if text != "" && len(rulesApplied) > 0 {
			textLines := strings.Split(text, "\n")
			finalTextLines := []string{}
			for i := range textLines {
				if lineHasAlphabetic(textLines[i]) {
					sanitizedLine := strings.TrimSpace(collapseSpaces(textLines[i]))
					if sanitizedLine != "" {
						finalTextLines = append(finalTextLines, sanitizedLine)
					}
				}
			}
			text = strings.Join(finalTextLines, "\n")
		}

		//Simplify new cue text update. Initialize newCue.Lines with text and add to changes if rules were applied.
		if len(rulesApplied) > 0 {

			newCue.Lines = text

			changes = append(changes, CueChange{
				CueIndex:    cue.Index,
				Original:    cue.Lines,
				Transformed: newCue.Lines,
				Rules:       append([]string(nil), rulesApplied...),
			})
		}

		// Only add newCue to out.Cues if text is not empty
		if text != "" {
			out.Cues = append(out.Cues, newCue)
		}
		//out.Cues = append(out.Cues, newCue)
	}

	// Indexing is re-assigned during SRT formatting
	return out, changes
}

func removeTextBetweenDelimiters(text string, delimiters []rules.Delimiter, rulesApplied []string) (string, []string) {
	// Rerun delimiter scan if any rule was triggered for recursive processing.
	for {
		ruleTriggered := false

		for _, delimiter := range delimiters {
			// If delimiters are equal, try to normalize repetitions of the same delimites (ex: ♪♪ text ♪♪ -> ♪ text ♪)
			if utf8.RuneCountInString(delimiter.Left) == 1 && delimiter.Left == delimiter.Right {
				text = strings.ReplaceAll(text, delimiter.Left+delimiter.Left, delimiter.Left)
			}
			controlEscape := ""
			// ASS format is transformed to SRT format, so we don't need to guard for '{'
			// if delimiter.Left == "{" {
			// 	// ASS format uses curly braces for formatting (italic, bold, etc.), {\i1}Text{\i0}
			// 	controlEscape = "\\"
			// }
			minContentLen := 0
			if delimiter.Left == "<" {
				// SRT format uses angle brackets for formatting (italic, bold, etc.), <i>Text</i>
				// also <font=xxx>Text</font>
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
				rulesApplied = append(rulesApplied, string(rules.RuleRemoveBetweenDelimiters)+" "+delimiter.Left+" "+delimiter.Right)
				text = strings.TrimSpace(re.ReplaceAllString(text, ""))
				if text == "" {
					// Skip unnecessary RemoveBetweenDelimiters rule processing
					break
				}
			}
		}

		if !ruleTriggered || text == "" {
			break
		}
	}
	return text, rulesApplied
}

// MarkdownRows renders cue changes as markdown table body rows (no header).
func MarkdownRows(entries []CueChange) string {
	if len(entries) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&sb, "| %d | %-30s | %-30s | %s |\n",
			e.CueIndex,
			strings.ReplaceAll(e.Original, "\n", " \\n "),
			strings.ReplaceAll(e.Transformed, "\n", " \\n "),
			strings.Join(e.Rules, ", "))
	}
	return sb.String()
}

func removeUppercaseColonWords(s string) (bool, string) {
	// Remove words of 2+ uppercase letters.
	// Use word boundaries to avoid partial matches. Keep punctuation spacing tidy later.
	if len(s) > 0 && reUppercaseTextWithColon.MatchString(s) {
		return true, reUppercaseTextWithColon.ReplaceAllString(s, "")
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
		if before, ok := strings.CutSuffix(trimmed, ":"); ok && before != "" {
			withoutColon := strings.TrimSpace(before)
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

func removeUppercaseTextWithColon(s string) (bool, string) {
	// Remove all text before the colon and the colon itself
	if len(s) == 0 {
		return false, s
	}
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	removed := false
	for _, line := range lines {
		if reUppercaseTextWithColon.MatchString(line) {
			removed = true
		}
		line = reUppercaseTextWithColon.ReplaceAllString(line, "")
		out = append(out, line)
	}
	return removed, strings.Join(out, "\n")
}

func removeTextBeforeColon(s string) (bool, string) {
	// Remove all text before the colon and the colon itself
	if len(s) == 0 {
		return false, s
	}
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	removed := false
	for _, line := range lines {
		if reTextWithColon.MatchString(line) {
			removed = true
		}
		line = reTextWithColon.ReplaceAllString(line, "")
		out = append(out, line)
	}
	return removed, strings.Join(out, "\n")
}
