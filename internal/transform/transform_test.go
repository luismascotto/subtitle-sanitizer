package transform

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/luismascotto/subtitle-sanitizer/internal/model"
	"github.com/luismascotto/subtitle-sanitizer/internal/rules"
)

func Test_removeUppercaseColonWords(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		s    string
		want string
	}{
		{
			name: "no uppercase words",
			s:    "Hello World",
			want: "Hello World",
		},
		{
			name: "remove a word:, but keep the rest",
			s:    "WORLD: Hello",
			want: "Hello",
		},
		{
			name: "remove uppercase words, but keep the rest (Upper with colon)",
			s:    "WORLD: HELLO",
			want: "HELLO",
		},
		{
			name: "remove uppercase words, empty line",
			s:    "WORLD 2:",
			want: "",
		},
		{
			name: "remove uppercase words with number, text after",
			s:    "GUARD 2: Hey!",
			want: "Hey!",
		},
		{
			name: "remove uppercase words extended, text after",
			s:    "GUARD AT BOOTH: Hey!",
			want: "Hey!",
		},
		{
			name: "remove uppercase words extended, text after",
			s:    "GUARD, ON PHONE: Hey!",
			want: "Hey!",
		},
		{
			name: "dont remove hours",
			s:    "7:00 p.m.",
			want: "7:00 p.m.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got := removeUppercaseColonWords(tt.s)

			if got != tt.want {
				t.Errorf("removeUppercaseColonWords() = [%v], want [%v]", got, tt.want)
			}
		})
	}
}

func Test_removeSingleLineColon(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		s    string
		want string
	}{
		{
			name: "no colon words single line",
			s:    "Hello World",
			want: "Hello World",
		},
		{
			name: "remove single line ending with colon",
			s:    "That woman said:",
			want: "",
		},
		{
			name: "dont remove single line ending with colon and have more than 3 words",
			s:    "This is a special release:",
			want: "This is a special release:",
		},
		{
			name: "dont remove line if not ending with colon",
			s:    "This release: hello",
			want: "This release: hello",
		},
		{
			name: "remove single line int text ending with colon",
			s:    "Hey\nThat woman said:",
			want: "Hey",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got := removeSingleLineColon(tt.s)

			if got != tt.want {
				t.Errorf("removeSingleLineColon() = [%v], want [%v]", got, tt.want)
			}
		})
	}
}

func Test_removeBetweenDelimitersParentheses(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{
			name: "remove between delimiters",
			s:    "Hello (World)",
			want: "Hello",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left := regexp.QuoteMeta("(")
			right := regexp.QuoteMeta(")")
			// Match shortest content between literal left/right (right is single-char here)
			//re := regexp.MustCompile(fmt.Sprintf(`%s[^%s]*%s`, left, right, right))
			re := regexp.MustCompile(fmt.Sprintf(`%s[^%s]*%s`, left, right, right))
			// Remove the text including the delimiters
			got := strings.TrimSpace(re.ReplaceAllString(tt.s, ""))
			if got != tt.want {
				t.Errorf("removeBetweenDelimiters() = [%v], want [%v]", got, tt.want)
			}
		})
	}
}

func Test_removeBetweenDelimitersBrackets(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{
			name: "remove between delimiters",
			s:    "Hello [World]",
			want: "Hello",
		}, {
			name: "remove between delimiters, with empty content",
			s:    "Hello []",
			want: "Hello",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left := regexp.QuoteMeta("[")
			right := regexp.QuoteMeta("]")
			// Match shortest content between literal left/right (right is single-char here)
			re := regexp.MustCompile(fmt.Sprintf(`%s[^%s]{0,}%s`, left, right, right))
			// Remove the text including the delimiters
			got := strings.TrimSpace(re.ReplaceAllString(tt.s, ""))
			if got != tt.want {
				t.Errorf("removeBetweenDelimiters() = [%v], want [%v]", got, tt.want)
			}
		})
	}
}

func Test_removeBetweenDelimitersCurlyBraces(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{
			name: "remove between delimiters, avoid formatting delimiters",
			s:    "{\\b1}Hello {World}{\\b0}",
			want: "{\\b1}Hello {\\b0}",
		}, {
			name: "remove between delimiters, avoid formatting delimiters 2",
			s:    "{\\b1}Hello {World}{\\b0}",
			want: "{\\b1}Hello {\\b0}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left := regexp.QuoteMeta("{")
			right := regexp.QuoteMeta("}")
			// Match shortest content between literal left/right (right is single-char here)
			re := regexp.MustCompile(fmt.Sprintf(`%s[^%s\\]*%s`, left, right, right))
			// Remove the text including the delimiters
			got := strings.TrimSpace(re.ReplaceAllString(tt.s, ""))
			if got != tt.want {
				t.Errorf("removeBetweenDelimiters() = [%v], want [%v]", got, tt.want)
			}
		})
	}
}

// Between < and > delimiters
func Test_removeBetweenDelimitersAngleBrackets(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{
			name: "remove between delimiters",
			s:    "<Hello> World",
			want: "World",
		}, {
			name: "remove between delimiters, avoid formatting delimiters",
			s:    "<Hello> <i>World</i>",
			want: "<i>World</i>",
		}, {
			name: "remove between delimiters, avoid formatting delimiters 2",
			s:    "<Hello> <bold>World</bold>",
			want: "World</bold>",
		}, {
			name: "remove between delimiters, avoid formatting delimiters 3",
			s:    "<font color='red'>Hello</font> <wrong>World</bold>",
			want: "<font color='red'>Hello</font> World</bold>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left := regexp.QuoteMeta("<")
			right := regexp.QuoteMeta(">")
			// Match shortest content between literal left/right (right is single-char here)
			re := regexp.MustCompile(fmt.Sprintf(`%s[^%s\\/=]{3,}%s`, left, right, right))
			// Remove the text including the delimiters
			got := strings.TrimSpace(re.ReplaceAllString(tt.s, ""))
			if got != tt.want {
				t.Errorf("removeBetweenDelimiters() = [%v], want [%v]", got, tt.want)
			}
		})
	}
}

func Test_convertASSFormattingToSRT(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "italic on/off",
			in:   "{\\i1}Text{\\i0}",
			want: "<i>Text</i>",
		},
		{
			name: "bold + italic nested",
			in:   "{\\b1}{\\i1}Text{\\i0}{\\b0}",
			want: "<b><i>Text</i></b>",
		},
		{
			name: "underline and strike",
			in:   "{\\u1}Under{\\u0} and {\\s1}strike{\\s0}",
			want: "<u>Under</u> and <s>strike</s>",
		},
		{
			name: "opening within text",
			in:   "Hello {\\b1}World{\\b0}",
			want: "Hello <b>World</b>",
		},
		{
			name: "other style ASS tag removed",
			in:   "{\\pos(10,20)}Text",
			want: "Text",
		},
		{
			name: "centervalues other than 1 also open tag",
			in:   "{\\b7002}Text{\\b0}",
			want: "<b>Text</b>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertASSFormattingToSRT(tt.in)
			if got != tt.want {
				t.Fatalf("convertASSFormattingToSRT(%q) = %q; want %q", tt.in, got, tt.want)
			}
		})
	}
}

func Test_dontRemoveBetweenDelimitersAcrossLinesRegex(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{
			name: "don't remove between delimiters across lines",
			s:    "Hello [World is\nending]",
			want: "Hello [World is\nending]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left := regexp.QuoteMeta("[")
			right := regexp.QuoteMeta("]")
			// Match shortest content between literal left/right (right is single-char here)
			re := regexp.MustCompile(fmt.Sprintf(`%s[^%s\n]{0,}%s`, left, right, right))
			// Remove the text including the delimiters
			got := strings.TrimSpace(re.ReplaceAllString(tt.s, ""))
			if got != tt.want {
				t.Errorf("removeBetweenDelimiters() = [%v], want [%v]", got, tt.want)
			}
		})
	}
}

func Test_removeBetweenDelimitersAcrossLinesRegex(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{
			name: "remove between delimiters across lines",
			s:    "Hello [World is\nending]",
			want: "Hello",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left := regexp.QuoteMeta("[")
			right := regexp.QuoteMeta("]")
			// Match shortest content between literal left/right (right is single-char here)
			re := regexp.MustCompile(fmt.Sprintf(`%s[^%s]{0,}%s`, left, right, right))
			// Remove the text including the delimiters
			got := strings.TrimSpace(re.ReplaceAllString(tt.s, ""))
			if got != tt.want {
				t.Errorf("removeBetweenDelimiters() = [%v], want [%v]", got, tt.want)
			}
		})
	}
}

func TestApplyAll_removeLineIfContains_logsChange(t *testing.T) {
	original := "foo music * bar"
	doc := model.Document{
		Format: model.SubtitleFormatSRT,
		Cues: []*model.Cue{
			{Index: 1, Lines: original},
		},
	}
	conf := rules.Config{RemoveLineIfContains: " music *"}
	out, ch := ApplyAll(doc, conf)
	if len(out.Cues) != 0 {
		t.Fatalf("cue should be dropped, got %d cues", len(out.Cues))
	}
	if len(ch) != 1 || ch[0].Rules[0] != string(rules.RuleRemoveLineIfContains) || ch[0].Original != original {
		t.Fatalf("unexpected changes: %+v", ch)
	}
}

func TestApplyAll_removeLineColon_logsChange(t *testing.T) {
	original := "That woman said:"
	doc := model.Document{
		Format: model.SubtitleFormatSRT,
		Cues: []*model.Cue{
			{Index: 1, Lines: original},
		},
	}
	conf := rules.Config{RemoveSingleLineColon: true}
	out, ch := ApplyAll(doc, conf)
	if len(out.Cues) != 0 {
		t.Fatalf("cue should be dropped, got %d cues", len(out.Cues))
	}
	if len(ch) != 1 || ch[0].Rules[0] != string(rules.RuleRemoveSingleLineColon) || ch[0].Original != original {
		t.Fatalf("unexpected changes: %+v", ch)
	}
}

func TestApplyAll_removeTextBeforeColonIfUppercase_logsChange(t *testing.T) {
	original := "VOICE OVER:"
	doc := model.Document{
		Format: model.SubtitleFormatSRT,
		Cues: []*model.Cue{
			{Index: 1, Lines: original},
		},
	}
	conf := rules.Config{RemoveTextBeforeColonIfUppercase: true}
	out, ch := ApplyAll(doc, conf)
	if len(out.Cues) != 0 {
		t.Fatalf("cue should be dropped, got %d cues", len(out.Cues))
	}
	if len(ch) != 1 || ch[0].Rules[0] != string(rules.RuleRemoveTextBeforeColonIfUppercase) || ch[0].Original != original {
		t.Fatalf("unexpected changes: %+v", ch)
	}
}

func TestApplyAll_removeTextBeforeColon_logsChange(t *testing.T) {
	original := "Voice over:"
	doc := model.Document{
		Format: model.SubtitleFormatSRT,
		Cues: []*model.Cue{
			{Index: 1, Lines: original},
		},
	}
	conf := rules.Config{RemoveTextBeforeColon: true}
	out, ch := ApplyAll(doc, conf)
	if len(out.Cues) != 0 {
		t.Fatalf("cue should be dropped, got %d cues", len(out.Cues))
	}
	if len(ch) != 1 || ch[0].Rules[0] != string(rules.RuleRemoveTextBeforeColon) || ch[0].Original != original {
		t.Fatalf("unexpected changes: %+v", ch)
	}
}

func TestApplyAll_parentheses_emitsChange(t *testing.T) {
	doc := model.Document{
		Format: model.SubtitleFormatSRT,
		Cues: []*model.Cue{
			{Index: 1, Lines: "Hello (noise) world"},
		},
	}
	conf := rules.Config{
		RemoveBetweenDelimiters: []rules.Delimiter{{Left: "(", Right: ")"}},
	}
	_, ch := ApplyAll(doc, conf)
	if len(ch) != 1 {
		t.Fatalf("want 1 change, got %d: %+v", len(ch), ch)
	}
}

func TestMarkdownRows_empty(t *testing.T) {
	if s := MarkdownRows(nil); s != "" {
		t.Fatalf("got %q", s)
	}
	if s := MarkdownRows([]CueChange{}); s != "" {
		t.Fatalf("empty slice: got %q", s)
	}
}

func TestMarkdownRows_singleRow(t *testing.T) {
	s := MarkdownRows([]CueChange{{
		CueIndex:    3,
		Original:    "a\nb",
		Transformed: "c",
		Rules:       []string{string(rules.RuleRemoveBetweenDelimiters) + " ( )"},
	}})
	if !strings.Contains(s, "| 3 |") || !strings.Contains(s, " \\n ") || !strings.Contains(s, "\\ Delims /") {
		t.Fatalf("unexpected markdown row:\n%s", s)
	}
}

func TestApplyAll_noCues(t *testing.T) {
	out, ch := ApplyAll(model.Document{Format: model.SubtitleFormatSRT}, rules.Config{})
	if len(out.Cues) != 0 || len(ch) != 0 {
		t.Fatalf("want empty out and changes, got %d cues, %d changes", len(out.Cues), len(ch))
	}
}

func TestApplyAll_emptyCueLines(t *testing.T) {
	doc := model.Document{
		Format: model.SubtitleFormatSRT,
		Cues:   []*model.Cue{{Index: 1, Lines: ""}},
	}
	out, ch := ApplyAll(doc, rules.DefaultConfig())
	if len(out.Cues) != 0 || len(ch) != 0 {
		t.Fatalf("empty cue text should produce no output and no changes, got cues=%d changes=%d", len(out.Cues), len(ch))
	}
}

func TestApplyAll_removeLineIfContains_emptyConfigDoesNothing(t *testing.T) {
	doc := model.Document{
		Format: model.SubtitleFormatSRT,
		Cues:   []*model.Cue{{Index: 1, Lines: "foo music * bar"}},
	}
	conf := rules.Config{RemoveLineIfContains: ""}
	out, ch := ApplyAll(doc, conf)
	if len(ch) != 0 {
		t.Fatalf("expected no changes, got %+v", ch)
	}
	if len(out.Cues) != 1 || out.Cues[0].Lines != "foo music * bar" {
		t.Fatalf("cue should be unchanged, got %+v", out.Cues)
	}
}

func TestApplyAll_removeLineIfContains_multilineConfigSecondPatternMatches(t *testing.T) {
	doc := model.Document{
		Format: model.SubtitleFormatSRT,
		Cues:   []*model.Cue{{Index: 1, Lines: "alpha beta gamma"}},
	}
	conf := rules.Config{RemoveLineIfContains: "noSuchSubstring\nbeta"}
	out, ch := ApplyAll(doc, conf)
	if len(out.Cues) != 0 || len(ch) != 1 {
		t.Fatalf("want cue dropped and one change, got cues=%d changes=%d", len(out.Cues), len(ch))
	}
	if len(ch[0].Rules) != 1 || ch[0].Rules[0] != string(rules.RuleRemoveLineIfContains) {
		t.Fatalf("unexpected rules: %+v", ch[0].Rules)
	}
}

func TestApplyAll_removeLineIfContains_emptyFragmentMatchesEverything(t *testing.T) {
	// strings.Contains(s, "") is always true; a leading newline in the config yields an empty first fragment.
	doc := model.Document{
		Format: model.SubtitleFormatSRT,
		Cues:   []*model.Cue{{Index: 1, Lines: "untouched"}},
	}
	conf := rules.Config{RemoveLineIfContains: "\nneedle"}
	out, ch := ApplyAll(doc, conf)
	if len(out.Cues) != 0 || len(ch) != 1 {
		t.Fatalf("want cue cleared via empty fragment, got cues=%d changes=%v", len(out.Cues), ch)
	}
}

func TestApplyAll_removeBetweenDelimiters_nilAndEmptySlice(t *testing.T) {
	doc := model.Document{
		Format: model.SubtitleFormatSRT,
		Cues:   []*model.Cue{{Index: 1, Lines: "Hello (noise) world"}},
	}
	for _, delims := range [][]rules.Delimiter{nil, {}} {
		conf := rules.Config{RemoveBetweenDelimiters: delims}
		out, ch := ApplyAll(doc, conf)
		if len(ch) != 0 || len(out.Cues) != 1 || out.Cues[0].Lines != "Hello (noise) world" {
			t.Fatalf("delims=%v: want unchanged, got ch=%+v cues=%+v", delims, ch, out.Cues)
		}
	}
}

func TestApplyAll_emptyParenthesesRemoved(t *testing.T) {
	doc := model.Document{
		Format: model.SubtitleFormatSRT,
		Cues:   []*model.Cue{{Index: 1, Lines: "Hello () world"}},
	}
	conf := rules.Config{RemoveBetweenDelimiters: []rules.Delimiter{{Left: "(", Right: ")"}}}
	out, ch := ApplyAll(doc, conf)
	if len(ch) != 1 {
		t.Fatalf("want 1 change, got %+v", ch)
	}
	if len(out.Cues) != 1 || out.Cues[0].Lines != "Hello world" {
		t.Fatalf("got lines %q, want %q", out.Cues[0].Lines, "Hello world")
	}
}

func TestApplyAll_postProcess_dropsLinesWithoutLetters(t *testing.T) {
	doc := model.Document{
		Format: model.SubtitleFormatSRT,
		Cues:   []*model.Cue{{Index: 1, Lines: "Note: 42"}},
	}
	conf := rules.Config{RemoveTextBeforeColon: true}
	out, ch := ApplyAll(doc, conf)
	if len(ch) != 1 {
		t.Fatalf("want one change logged, got %+v", ch)
	}
	if len(out.Cues) != 0 {
		t.Fatalf("numeric-only remainder should drop cue, got %+v", out.Cues)
	}
	if ch[0].Transformed != "" {
		t.Fatalf("transformed should be empty after stripping non-alphabetic lines, got %q", ch[0].Transformed)
	}
}

func TestApplyAll_removeTextBeforeColon_lowercaseSpeakerUnchangedWhenUppercaseRuleOnly(t *testing.T) {
	doc := model.Document{
		Format: model.SubtitleFormatSRT,
		Cues:   []*model.Cue{{Index: 1, Lines: "Father: Hello"}},
	}
	conf := rules.Config{RemoveTextBeforeColonIfUppercase: true, RemoveTextBeforeColon: false}
	out, ch := ApplyAll(doc, conf)
	if len(ch) != 0 || len(out.Cues) != 1 || out.Cues[0].Lines != "Father: Hello" {
		t.Fatalf("uppercase-only rule must not strip mixed-case speaker; ch=%+v cues=%+v", ch, out.Cues)
	}
}

func TestApplyAll_ASS_convertThenStripUppercaseSpeaker(t *testing.T) {
	// Speaker label must not be inside {\i1}…{\i0} or the line no longer matches ^…[A-Z]…: at the start after "<".
	doc := model.Document{
		Format: model.SubtitleFormatASS,
		Cues:   []*model.Cue{{Index: 1, Lines: `NARRATOR: {\i1}Hello{\i0}`}},
	}
	conf := rules.Config{RemoveTextBeforeColonIfUppercase: true}
	out, ch := ApplyAll(doc, conf)
	if len(ch) != 1 {
		t.Fatalf("want 1 change, got %+v", ch)
	}
	if len(out.Cues) != 1 || out.Cues[0].Lines != "<i>Hello</i>" {
		t.Fatalf("got %q, want %q", out.Cues[0].Lines, "<i>Hello</i>")
	}
}

func Test_removeSingleLineColon_edgeCases(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		removed, got := removeSingleLineColon("")
		if removed || got != "" {
			t.Fatalf("removed=%v got=%q", removed, got)
		}
	})
	t.Run("line is only colon not removed", func(t *testing.T) {
		removed, got := removeSingleLineColon(":")
		if removed || got != ":" {
			t.Fatalf("removed=%v got=%q", removed, got)
		}
	})
	t.Run("whitespace only before colon", func(t *testing.T) {
		removed, got := removeSingleLineColon("   :")
		if removed || got != "   :" {
			t.Fatalf("removed=%v got=%q", removed, got)
		}
	})
}

func Test_removeTextBeforeColon_emptyString(t *testing.T) {
	removed, got := removeTextBeforeColon("")
	if removed || got != "" {
		t.Fatalf("removed=%v got=%q", removed, got)
	}
}

func Test_removeUppercaseTextWithColon_emptyString(t *testing.T) {
	removed, got := removeUppercaseTextWithColon("")
	if removed || got != "" {
		t.Fatalf("removed=%v got=%q", removed, got)
	}
}

func Test_collapseSpaces(t *testing.T) {
	if got := collapseSpaces("a    b\t\tc"); got != "a b c" {
		t.Fatalf("got %q", got)
	}
	if got := collapseSpaces("no-extra"); got != "no-extra" {
		t.Fatalf("got %q", got)
	}
}

func Test_lineHasAlphabetic(t *testing.T) {
	if !lineHasAlphabetic("123a") {
		t.Fatal("expected true for letter")
	}
	if lineHasAlphabetic("123") {
		t.Fatal("expected false for digits only")
	}
	if lineHasAlphabetic("…") {
		t.Fatal("expected false for punctuation only")
	}
}

func Test_normalizeRepeatedSingleDelimitersAlgorithm(t *testing.T) {
	tests := []struct {
		name string
		sep  string
		s    string
		want string
	}{
		{
			name: "normalize repeated single delimiters rune",
			sep:  "♪",
			s:    "♪♪ text ♪♪",
			want: "♪ text ♪",
		}, {
			name: "normalize repeated single delimiters string/char",
			sep:  "*",
			s:    "** text **",
			want: "* text *",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left := fmt.Sprintf("%s", tt.sep)
			right := left
			//	t.Logf("left: [%v]=%d, right: [%v]=%d", left, utf8.RuneCountInString(left), right, utf8.RuneCountInString(right))
			if utf8.RuneCountInString(left) == 1 && left == right {
				got := strings.ReplaceAll(tt.s, left+left, left)
				if got != tt.want {
					t.Errorf("normalizeRepeatedSingleDelimiters() = [%v], want [%v]", got, tt.want)
				}
			} else {
				t.Errorf("normalizeRepeatedSingleDelimiters() = not single delimiter: [%v], [%v]", left, right)
			}
		})
	}
}

func TestApplyAll_removeBetweenDoubledEqualDelimiters_logsChange(t *testing.T) {
	original := "** Hello some Text **"
	doc := model.Document{
		Format: model.SubtitleFormatSRT,
		Cues:   []*model.Cue{{Index: 1, Lines: original}},
	}
	conf := rules.Config{RemoveBetweenDelimiters: []rules.Delimiter{{Left: "*", Right: "*"}}}
	out, ch := ApplyAll(doc, conf)
	if len(ch) != 1 {
		t.Fatalf("want 1 change, got %+v", ch)
	}
	if len(out.Cues) != 0 {
		t.Fatalf("text between doubled equal delimiters should be dropped, got %+v", out.Cues)
	}
	if ch[0].Rules[0] != string(rules.RuleRemoveBetweenDelimiters)+" * *" {
		t.Fatalf("unexpected rules: %+v", ch[0].Rules)
	}
	if ch[0].Original != original {
		t.Fatalf("original text should be unchanged, got %q", ch[0].Original)
	}
	if ch[0].Transformed != "" {
		t.Fatalf("transformed text should be empty, got %q", ch[0].Transformed)
	}
}

func TestApplyAll_Guard_Lt_Gt_between_Delimiters_logsChange(t *testing.T) {
	// Guard label must not be inside <i>…</i> or the line no longer matches ^…[A-Z]…: at the start after "<".
	original := "<i>Hello</i>"
	doc := model.Document{
		Format: model.SubtitleFormatSRT,
		Cues:   []*model.Cue{{Index: 1, Lines: original}},
	}
	conf := rules.Config{RemoveBetweenDelimiters: []rules.Delimiter{{Left: "<", Right: ">"}}}
	out, ch := ApplyAll(doc, conf)
	if len(ch) != 0 {
		t.Fatalf("want no changes, got %+v", ch)
	}
	if len(out.Cues) != 1 || out.Cues[0].Lines != original {
		t.Fatalf("got %q, want %q", out.Cues[0].Lines, original)
	}
}

func TestApplyAll_invalidDelimiterRegex_compileErrorContinues(t *testing.T) {
	invalid := string([]byte{0xff}) // invalid UTF-8 triggers regexp.Compile error
	original := "keep (noise) this"
	doc := model.Document{
		Format: model.SubtitleFormatSRT,
		Cues:   []*model.Cue{{Index: 1, Lines: original}},
	}
	conf := rules.Config{
		RemoveBetweenDelimiters: []rules.Delimiter{
			{Left: invalid, Right: ")"},
			{Left: "(", Right: ")"},
		},
	}
	out, ch := ApplyAll(doc, conf)
	if len(ch) != 1 {
		t.Fatalf("want one successful change after invalid delimiter, got %+v", ch)
	}
	if len(out.Cues) != 1 || out.Cues[0].Lines != "keep this" {
		t.Fatalf("unexpected output cues: %+v", out.Cues)
	}
}
