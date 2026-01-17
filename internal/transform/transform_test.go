package transform

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
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
		}, {
			name: "remove uppercase word, empty line",
			s:    "WORLD:",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got := removeUppercaseColonWords(tt.s)

			if !strings.Contains(got, tt.want) {
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
			name: "remove  single line with colon",
			s:    "That woman said:",
			want: "",
		},
		{
			name: "line with colon but have more than 3 words",
			s:    "This is a special release:",
			want: "This is a special release:",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got := removeUppercaseColonWords(tt.s)

			if !strings.Contains(got, tt.want) {
				t.Errorf("removeUppercaseColonWords() = [%v], want [%v]", got, tt.want)
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
