package websocket

import (
	"strings"
	"testing"
)

func TestSanitizeText_Basic(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"  hello  ", "hello"},
		{"hello\nworld", "hello\nworld"},
	}
	for _, tt := range tests {
		got := sanitizeText(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeText(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSanitizeText_StripsControlChars(t *testing.T) {
	input := "he\x00ll\x01o"
	got := sanitizeText(input)
	if strings.ContainsAny(got, "\x00\x01") {
		t.Errorf("sanitizeText should strip control chars, got %q", got)
	}
}

func TestSanitizeText_TruncatesLong(t *testing.T) {
	long := strings.Repeat("a", 500)
	got := sanitizeText(long)
	if len([]rune(got)) > maxTextLen {
		t.Errorf("sanitizeText should cap at %d runes, got %d", maxTextLen, len([]rune(got)))
	}
}

func TestSanitizeText_Empty(t *testing.T) {
	got := sanitizeText("   ")
	if got != "" {
		t.Errorf("sanitizeText(whitespace) = %q, want empty", got)
	}
}
