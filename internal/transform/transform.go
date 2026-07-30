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

const (
	minCapsInWord = 2
)

var (
	// reBr                  = regexp.MustCompile(`<br />`)
	reSpaces = regexp.MustCompile(`\s{2,}`)
	//reUppercaseColonWords    = regexp.MustCompile(`\b[A-Z]{1,}\s*[A-Z0-9]{1,}:[ \t]*`)
	reTextWithColon          = regexp.MustCompile(`^[^:]+:[ \t]*`)
	reUppercaseTextWithColon = regexp.MustCompile(`^[^:a-z]*[A-Z][^:a-z]*:[ \t]*`)
	// ASS override tags → SRT (compiled once; convertASSFormattingToSRT used per cue).
	reASSOpenTag   = regexp.MustCompile(`{\\([bius])[1-9]\d*}`)
	reASSCloseTag  = regexp.MustCompile(`{\\([bius])0}`)
	reASSOtherTags = regexp.MustCompile(`{\\[^bius][^}]*}`)
)

// CueChange records one cue that had at least one rule applied (including full-line removal).
type CueChange struct {
	CueIndex    int      `json:"cueIndex"`
	Original    string   `json:"original"`
	Transformed string   `json:"transformed"`
	Rules       []string `json:"rules"`
}

// ApplyFn transforms a document using prepared Rules (compiled delimiters included).
type ApplyFn func(doc model.Document, r Rules) (model.Document, []CueChange)

// cueOutcome is the result of transforming one cue. Kept is nil when the cue is dropped.
type cueOutcome struct {
	kept   *model.Cue
	change *CueChange // nil when no rule fired
}

// Rules is a Config with delimiter regexes compiled once. Reuse across documents/files
// that share the same rules; safe for concurrent Apply calls.
type Rules struct {
	conf   rules.Config
	delims []compiledDelimiter
}

// NewRules compiles delimiter regexes from conf. Call once per config, not per file.
func NewRules(conf rules.Config) Rules {
	return Rules{
		conf:   conf,
		delims: compileDelimiters(conf.RemoveBetweenDelimiters),
	}
}

// Config returns the underlying rule config.
func (r Rules) Config() rules.Config { return r.conf }

// ApplyAll runs all enabled transformations in parallel (GOMAXPROCS workers).
// Compiles delimiters once for this call; prefer NewRules + ApplyAllWithRules when
// applying the same config to many files.
// Use ApplyAllSequential when targeting WASM/TinyGo if goroutine pools misbehave.
func ApplyAll(doc model.Document, conf rules.Config) (model.Document, []CueChange) {
	return applyAllParallel(doc, NewRules(conf))
}

// ApplyAllWithRules applies prepared Rules (no delimiter recompilation).
func ApplyAllWithRules(doc model.Document, r Rules) (model.Document, []CueChange) {
	return applyAllParallel(doc, r)
}

// ApplyAllSequential is the explicit sequential ApplyFn.
var ApplyAllSequential ApplyFn = applyAllSequential

// ApplyAllParallel is the explicit parallel ApplyFn.
var ApplyAllParallel ApplyFn = applyAllParallel

func applyAllSequential(doc model.Document, r Rules) (model.Document, []CueChange) {
	outcomes := make([]cueOutcome, len(doc.Cues))
	for i, cue := range doc.Cues {
		outcomes[i] = applyCue(cue, doc.Format, r)
	}
	return assembleDocument(doc, outcomes)
}

// assembleDocument merges per-cue outcomes in input order into a document and change log.
func assembleDocument(doc model.Document, outcomes []cueOutcome) (model.Document, []CueChange) {
	out := model.Document{
		Format: doc.Format,
		Header: doc.Header,
		Cues:   make([]*model.Cue, 0, len(outcomes)),
	}
	var changes []CueChange
	for _, res := range outcomes {
		if res.change != nil {
			changes = append(changes, *res.change)
		}
		if res.kept != nil {
			out.Cues = append(out.Cues, res.kept)
		}
	}
	// Indexing is re-assigned during SRT formatting
	return out, changes
}

// applyCue transforms a single cue. Safe for concurrent calls: no shared mutable state
// (delimiter regexes on Rules are read-only).
func applyCue(cue *model.Cue, format model.SubtitleFormat, r Rules) cueOutcome {
	var rulesApplied []string
	ruleTriggered := false
	conf := r.conf

	text := cue.Lines

	if format == model.SubtitleFormatASS {
		text = convertASSFormattingToSRT(text)
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
		if conf.RemoveLineIfAllCapsAction {
			ruleTriggered, text = removeLineIfAllCapsAction(text)
			if ruleTriggered {
				rulesApplied = append(rulesApplied, string(rules.RuleRemoveLineIfAllCapsAction))
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

	if text != "" && conf.RemoveOnlySymbolsLine {
		if !lineHasAlphanumeric(text) {
			text = ""
			rulesApplied = append(rulesApplied, string(rules.RuleRemoveOnlySymbolsLine))
		}
	}

	if text != "" && len(r.delims) > 0 {
		text, rulesApplied = removeTextBetweenCompiledDelimiters(text, r.delims, rulesApplied)
	}

	if text != "" && len(rulesApplied) > 0 {
		var finalTextLines []string
		for line := range strings.SplitSeq(text, "\n") {
			if lineHasAlphanumeric(line) {
				sanitizedLine := strings.TrimSpace(collapseSpaces(line))
				if sanitizedLine != "" {
					finalTextLines = append(finalTextLines, sanitizedLine)
				}
			}
		}
		text = strings.Join(finalTextLines, "\n")
	}

	var change *CueChange
	if len(rulesApplied) > 0 {
		change = &CueChange{
			CueIndex:    cue.Index,
			Original:    cue.Lines,
			Transformed: text,
			Rules:       rulesApplied, // ownership transfer; not reused after this
		}
	}

	if text == "" {
		return cueOutcome{change: change}
	}
	// Reuse input cue when unchanged; allocate only when text diverged (rules or ASS convert).
	if text == cue.Lines {
		return cueOutcome{kept: cue, change: change}
	}
	return cueOutcome{
		kept: &model.Cue{
			Index: cue.Index,
			Start: cue.Start,
			End:   cue.End,
			Lines: text,
		},
		change: change,
	}
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

// compiledDelimiter holds a precompiled regex and metadata for one delimiter pair.
type compiledDelimiter struct {
	left     string
	right    string
	label    string // rule log label, e.g. `\ Delims / ( )`
	collapse string // Left+Left when Left==Right and single rune; empty otherwise
	re       *regexp.Regexp
}

// compileDelimiters builds reusable regexes from raw delimiter config.
// Invalid patterns are skipped (same as removeTextBetweenDelimiters).
func compileDelimiters(delimiters []rules.Delimiter) []compiledDelimiter {
	out := make([]compiledDelimiter, 0, len(delimiters))
	for _, d := range delimiters {
		if cd, ok := compileDelimiter(d); ok {
			out = append(out, cd)
		}
	}
	return out
}

func compileDelimiter(delimiter rules.Delimiter) (compiledDelimiter, bool) {
	controlEscape := ""
	minContentLen := 0
	if delimiter.Left == "<" {
		// SRT format uses angle brackets for formatting (italic, bold, etc.), <i>Text</i>
		minContentLen = 3
		controlEscape = "/="
	}
	left := regexp.QuoteMeta(delimiter.Left)
	right := regexp.QuoteMeta(delimiter.Right)
	re, err := regexp.Compile(fmt.Sprintf(`%s[^%s%s]{%d,}%s`, left, controlEscape, right, minContentLen, right))
	if err != nil {
		fmt.Println("Error compiling regex:", err, "for delimiter:", delimiter)
		return compiledDelimiter{}, false
	}
	cd := compiledDelimiter{
		left:  delimiter.Left,
		right: delimiter.Right,
		label: string(rules.RuleRemoveBetweenDelimiters) + " " + delimiter.Left + " " + delimiter.Right,
		re:    re,
	}
	if utf8.RuneCountInString(delimiter.Left) == 1 && delimiter.Left == delimiter.Right {
		cd.collapse = delimiter.Left + delimiter.Left
	}
	return cd, true
}

// removeTextBetweenCompiledDelimiters is the precompiled-regex counterpart of
// removeTextBetweenDelimiters (same recursive scan / rule labels).
func removeTextBetweenCompiledDelimiters(text string, delimiters []compiledDelimiter, rulesApplied []string) (string, []string) {
	for {
		ruleTriggered := false

		for _, d := range delimiters {
			if d.collapse != "" {
				text = strings.ReplaceAll(text, d.collapse, d.left)
			}
			if d.re.MatchString(text) {
				ruleTriggered = true
				rulesApplied = append(rulesApplied, d.label)
				text = strings.TrimSpace(d.re.ReplaceAllString(text, ""))
				if text == "" {
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

func RemoveTextBetweenOpenCloseMatchingDelimiter(text string, delimiter rules.Delimiter, rulesApplied []string) (string, []string) {
	// Remove text between open and close matching delimiter
	for {
		lastLeftIndex := strings.LastIndex(text, delimiter.Left)
		if lastLeftIndex == -1 {
			break
		}
		firstRightAfterLeft := strings.Index(text[lastLeftIndex:], delimiter.Right)
		if firstRightAfterLeft == -1 {
			break
		}
		text = strings.TrimSpace(text[:lastLeftIndex] + text[lastLeftIndex+firstRightAfterLeft+len(delimiter.Right):])
		rulesApplied = append(rulesApplied, string(rules.RuleRemoveBetweenDelimiters)+" "+delimiter.Left+" "+delimiter.Right)
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
	linesSequence := strings.SplitSeq(s, "\n")
	out := make([]string, 0)
	removed := false
	for line := range linesSequence {
		trimmed := strings.TrimSpace(line)
		if before, ok := strings.CutSuffix(trimmed, ":"); ok && before != "" {
			withoutColon := strings.TrimSpace(before)
			// Count words using whitespace splitting
			wordCount := 0
			for range strings.FieldsSeq(withoutColon) {
				wordCount++
				if wordCount > 3 {
					break
				}
			}
			if wordCount > 0 && wordCount <= 3 {
				removed = true
				continue
			}
		}
		out = append(out, line)
	}
	if !removed {
		return false, s
	}
	return true, strings.Join(out, "\n")
}

func collapseSpaces(s string) string {
	return reSpaces.ReplaceAllString(s, " ")
}

func isAllCapsLine(s string, min int) bool {
	words := 0
	for w := range strings.FieldsFuncSeq(s, func(r rune) bool {
		return !unicode.IsLetter(r)
	}) {
		for _, r := range w {
			if !unicode.IsUpper(r) {
				return false
			}
		}
		if len(w) > minCapsInWord {
			words++
		}
	}

	return words >= min
}

func removeLineIfAllCapsAction(s string) (bool, string) {
	if len(s) == 0 {
		return false, s
	}

	out := make([]string, 0)
	min := min(2, strings.Count(s, "\n")+1)

	removed := false
	for line := range strings.SplitSeq(s, "\n") {
		if !strings.HasSuffix(line, "?") && isAllCapsLine(strings.TrimSpace(line), min) {
			removed = true
			continue
		}
		out = append(out, line)
	}
	if !removed {
		return false, s
	}
	return true, strings.Join(out, "\n")
}

func lineHasAlphanumeric(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
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
	if !strings.Contains(s, `{\`) {
		return s
	}
	formatted := reASSOpenTag.ReplaceAllString(s, "<$1>")
	formatted = reASSCloseTag.ReplaceAllString(formatted, "</$1>")
	return reASSOtherTags.ReplaceAllString(formatted, "")
}

func removeUppercaseTextWithColon(s string) (bool, string) {
	// Remove all text before the colon and the colon itself
	if len(s) == 0 {
		return false, s
	}
	linesSequence := strings.SplitSeq(s, "\n")
	out := make([]string, 0)
	removed := false
	for line := range linesSequence {
		if reUppercaseTextWithColon.MatchString(line) {
			removed = true
			line = reUppercaseTextWithColon.ReplaceAllString(line, "")
		}
		out = append(out, line)
	}
	if removed {
		return true, strings.Join(out, "\n")
	}
	// Avoid inner allocations by returning the original string if no removal occurred
	return false, s
}

func removeTextBeforeColon(s string) (bool, string) {
	// Remove all text before the colon and the colon itself
	if len(s) == 0 {
		return false, s
	}
	linesSequence := strings.SplitSeq(s, "\n")
	out := make([]string, 0)
	removed := false
	for line := range linesSequence {
		if reTextWithColon.MatchString(line) {
			removed = true
		}
		line = reTextWithColon.ReplaceAllString(line, "")
		out = append(out, line)
	}
	if removed {
		return true, strings.Join(out, "\n")
	}
	// Avoid inner allocations by returning the original string if no removal occurred
	return false, s
}
