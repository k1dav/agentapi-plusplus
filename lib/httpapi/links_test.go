package httpapi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractURLs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "plain",
			text: "see https://example.com/docs for details",
			want: []string{"https://example.com/docs"},
		},
		{
			name: "trailing punctuation",
			text: "check https://example.com/a. Also (https://example.com/b), and https://example.com/c!",
			want: []string{"https://example.com/a", "https://example.com/b", "https://example.com/c"},
		},
		{
			name: "balanced parens kept",
			text: "https://en.wikipedia.org/wiki/Go_(programming_language) is neat",
			want: []string{"https://en.wikipedia.org/wiki/Go_(programming_language)"},
		},
		{
			name: "www without scheme",
			text: "visit www.example.com/path today",
			want: []string{"www.example.com/path"},
		},
		{
			name: "multiple per line and json",
			text: `{"url":"https://api.example.com/v1?key=abc&x=1","other":"http://foo.bar/baz"}`,
			want: []string{"https://api.example.com/v1?key=abc&x=1", "http://foo.bar/baz"},
		},
		{
			name: "none",
			text: "no links here",
			want: []string{},
		},
		{
			name: "tui decoration glyphs not glued",
			text: "  ⎿ read https://support.claude.com/en/articles/15363606⎿ done",
			want: []string{"https://support.claude.com/en/articles/15363606"},
		},
		{
			name: "cjk text adjacent",
			text: "第二個 https://example.com/path 結束",
			want: []string{"https://example.com/path"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, extractURLs(tt.text))
		})
	}
}

func TestExtractURLsFromScreen(t *testing.T) {
	t.Parallel()

	t.Run("wrapped url is rejoined", func(t *testing.T) {
		t.Parallel()
		// A URL hard-wrapped at the terminal edge: the first line is full
		// width and ends mid-URL; the continuation line is indented.
		text := "The binary is at https://github.com/k1dav/agentapi-plusplus/releases/download/" + "\n" +
			"  v1.2.3/agentapi-linux-amd64 and it is ready."
		got := extractURLsFromScreen(text)
		assert.Equal(t,
			[]string{"https://github.com/k1dav/agentapi-plusplus/releases/download/v1.2.3/agentapi-linux-amd64"},
			got)
	})

	t.Run("three-line wrap", func(t *testing.T) {
		t.Parallel()
		u1 := "https://example.com/" + strings.Repeat("p", 58) // 78 chars total
		text := u1 + "\n" + strings.Repeat("q", 78) + "\n" + "rrr and done"
		got := extractURLsFromScreen(text)
		assert.Equal(t,
			[]string{"https://example.com/" + strings.Repeat("p", 58) + strings.Repeat("q", 78) + "rrr"},
			got)
	})

	t.Run("url ending mid-line is not joined", func(t *testing.T) {
		t.Parallel()
		text := "see https://example.com/docs for more\nand another line"
		got := extractURLsFromScreen(text)
		assert.Equal(t, []string{"https://example.com/docs"}, got)
	})

	t.Run("url at line end of short line is not joined", func(t *testing.T) {
		t.Parallel()
		// Line ends with the URL but is well short of the terminal width:
		// no wrap happened, don't join the next line.
		text := "see https://example.com/docs\nnextword here"
		got := extractURLsFromScreen(text)
		assert.Equal(t, []string{"https://example.com/docs"}, got)
	})
}
