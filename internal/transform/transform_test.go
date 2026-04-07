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

func Test_dontRemoveBetweenDelimitersAcrossLines(t *testing.T) {
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

func Test_removeBetweenDelimitersAcrossLines(t *testing.T) {
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
	doc := model.Document{
		Format: model.SubtitleFormatSRT,
		Cues: []*model.Cue{
			{Index: 1, Lines: "foo music * bar"},
		},
	}
	conf := rules.Config{RemoveLineIfContains: " music *"}
	out, ch := ApplyAll(doc, conf)
	if len(out.Cues) != 0 {
		t.Fatalf("cue should be dropped, got %d cues", len(out.Cues))
	}
	if len(ch) != 1 || ch[0].Rules[0] != "removeIfContains" || ch[0].Original != "foo music * bar" {
		t.Fatalf("unexpected changes: %+v", ch)
	}
}

func TestApplyAll_removeLineColon_logsChange(t *testing.T) {
	doc := model.Document{
		Format: model.SubtitleFormatSRT,
		Cues: []*model.Cue{
			{Index: 1, Lines: "That woman said:"},
		},
	}
	conf := rules.Config{RemoveSingleLineColon: true}
	out, ch := ApplyAll(doc, conf)
	if len(out.Cues) != 0 {
		t.Fatalf("cue should be dropped, got %d cues", len(out.Cues))
	}
	if len(ch) != 1 || ch[0].Rules[0] != "removeLineColon" || ch[0].Original != "That woman said:" {
		t.Fatalf("unexpected changes: %+v", ch)
	}
}

func TestApplyAll_removeTextBeforeColonIfUppercase_logsChange(t *testing.T) {
	doc := model.Document{
		Format: model.SubtitleFormatSRT,
		Cues: []*model.Cue{
			{Index: 1, Lines: "VOICE OVER:"},
		},
	}
	conf := rules.Config{RemoveTextBeforeColonIfUppercase: true}
	out, ch := ApplyAll(doc, conf)
	if len(out.Cues) != 0 {
		t.Fatalf("cue should be dropped, got %d cues", len(out.Cues))
	}
	if len(ch) != 1 || ch[0].Rules[0] != "TEXT:" || ch[0].Original != "VOICE OVER:" {
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
}

func Test_normalizeRepeatedSingleDelimiters(t *testing.T) {
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
