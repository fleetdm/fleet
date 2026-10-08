package markdown

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

func TestRenderTerms(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		src      string
		contains []string
		excludes []string
	}{
		{
			name: "structure renders",
			src: "# Terms\n\nSome **bold** and _em_ text.\n\n1. First\n2. Second\n\n- a\n- b\n\n" +
				"| Col | Val |\n|---|---|\n| x | 1 |\n\n> quoted\n\n`code`\n\n---\n",
			contains: []string{"<h1>Terms</h1>", "<strong>bold</strong>", "<em>em</em>", "<ol>", "<ul>",
				"<table>", "<th>Col</th>", "<td>1</td>", "<blockquote>", "<code>code</code>", "<hr>"},
		},
		{
			// Inline raw HTML loses its tags and keeps the text between them as
			// escaped text, which is inert.
			name:     "inline raw html loses its tags",
			src:      "before <script>alert(1)</script> after\n\n<b onmouseover=\"x()\">hi</b>\n",
			contains: []string{"before", "after", "hi"},
			excludes: []string{"<script", "</script", "onmouseover", "<b", "<!--"},
		},
		{
			// A raw HTML block is dropped whole, content included.
			name:     "block raw html is dropped whole",
			src:      "intro\n\n<script>\nalert(1)\n</script>\n\n<iframe src=\"https://evil.example\"></iframe>\n\noutro\n",
			contains: []string{"intro", "outro"},
			excludes: []string{"<script", "alert(1)", "<iframe", "evil.example", "<!--"},
		},
		{
			name:     "links become their text",
			src:      "See [the policy](https://example.com/policy) and [bad](javascript:alert(1)).\n",
			contains: []string{"the policy", "bad"},
			excludes: []string{"<a ", "href", "example.com", "javascript:"},
		},
		{
			name:     "bare urls stay plain text",
			src:      "Visit https://example.com today.\n",
			contains: []string{"https://example.com"},
			excludes: []string{"<a ", "href"},
		},
		{
			name:     "autolinks become plain text",
			src:      "See <https://example.com/x?a=1&b=2> or <legal@acme.example>.\n",
			contains: []string{"https://example.com/x?a=1&amp;b=2", "legal@acme.example"},
			excludes: []string{"<a ", "href", "mailto"},
		},
		{
			name:     "inline line breaks are kept",
			src:      "Line one<br>Line two\n\n| Address |\n| --- |\n| 1 Main St<br/>Springfield |\n",
			contains: []string{"Line one<br", "Line two", "1 Main St<br", "Springfield</td>"},
			excludes: []string{"Line oneLine two", "StSpringfield"},
		},
		{
			name:     "a byte order mark doesn't break the first heading",
			src:      "\xef\xbb\xbf# Terms\n",
			contains: []string{"<h1>Terms</h1>"},
			excludes: []string{"\ufeff"},
		},
		{
			name:     "images become their alt text",
			src:      "Logo: ![Company logo](https://cdn.example.com/logo.png)\n",
			contains: []string{"Company logo"},
			excludes: []string{"<img", "cdn.example.com"},
		},
		{
			name:     "task lists keep their checkboxes",
			src:      "- [x] done\n- [ ] todo\n",
			contains: []string{`type="checkbox"`, "done", "todo"},
		},
		{
			name:     "non-ascii survives",
			src:      "Términos y condiciones: 利用規約\n",
			contains: []string{"Términos y condiciones: 利用規約"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := RenderTerms([]byte(tc.src))
			require.NoError(t, err)
			for _, s := range tc.contains {
				require.Contains(t, got, s)
			}
			for _, s := range tc.excludes {
				require.NotContains(t, got, s)
			}
		})
	}
}

func TestRenderTermsNoVisibleText(t *testing.T) {
	t.Parallel()

	for _, src := range []string{
		"",
		"<div>\nOnly raw HTML\n</div>\n",
		"<!-- a comment -->\n",
		"[](https://example.com)\n",
		"   \n\t\n",
	} {
		got, err := RenderTerms([]byte(src))
		require.NoError(t, err, src)
		require.Empty(t, got, src)
	}
}

// termsTable is a table with the given columns and body rows, which costs
// columns times (rows + 1) cells counting the header.
func termsTable(cols, rows int) string {
	var b strings.Builder
	b.WriteString(strings.Repeat("| h ", cols) + "|\n")
	b.WriteString(strings.Repeat("| - ", cols) + "|\n")
	for range rows {
		b.WriteString("| x |\n")
	}
	return b.String()
}

func TestValidateTermsTableBudgetBypasses(t *testing.T) {
	t.Parallel()

	header := strings.Repeat("| h ", 100) + "|\n" + strings.Repeat("| - ", 100) + "|\n"
	for name, src := range map[string]string{
		// A heading ends goldmark's paragraph without a blank line.
		"table after a heading": "a\n|-|\n# h\n" + header + strings.Repeat("x\n", 101),
		// Not blank to goldmark, and a 4-space indent can't open a quote.
		"rows that look like quotes": header + strings.Repeat("    >\n", 101),
		// goldmark's blank is only space, tab and CR.
		"rows of vertical tabs": header + strings.Repeat("\v\n", 101),
	} {
		require.ErrorIs(t, ValidateTerms([]byte(src)), ErrTableTooLarge, name)
	}
}

func TestRenderTermsTableAlignment(t *testing.T) {
	t.Parallel()

	got, err := RenderTerms([]byte("| Left | Right |\n| :-- | --: |\n| a | b |\n"))
	require.NoError(t, err)
	require.Contains(t, got, `<th align="left">Left</th>`)
	require.Contains(t, got, `<td align="right">b</td>`)
}

func TestRenderTermsAppliesLimits(t *testing.T) {
	t.Parallel()

	// The terms page renders stored documents, so it enforces the same limits
	// as upload in case a document reached the database another way.
	got, err := RenderTerms([]byte(strings.Repeat("a", maxLineLength+1)))
	require.ErrorIs(t, err, ErrLineTooLong)
	require.Empty(t, got)
}

type panicTransformer struct{}

func (panicTransformer) Transform(*ast.Document, text.Reader, parser.Context) {
	panic("boom")
}

func TestRenderRecoversPanic(t *testing.T) {
	t.Parallel()

	md := goldmark.New(goldmark.WithParserOptions(parser.WithASTTransformers(
		util.Prioritized(panicTransformer{}, 1),
	)))
	doc, err := parse(md, []byte("# Terms\n"))
	require.ErrorContains(t, err, "panic: boom")
	require.Nil(t, doc)
}

func TestValidateTerms(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		src  string
		want error
	}{
		{"plain markdown", "# Terms\n\nAccept to continue.\n", nil},
		{"inline html keeps its text", "Accept <u>all</u> terms.\n", nil},
		{"html block without text", "Terms.\n\n<br>\n\nMore terms.\n", nil},
		{"comment", "<!-- toc -->\n\n# Terms\n", nil},
		{"script block has nothing to read", "# Terms\n\n<script>\nalert(1)\n</script>\n", nil},
		{"html block with text", "# Terms\n\n<div>\nClause 4 applies.\n</div>\n", ErrContainsHTML},
		{"html table", "<table><tr><td>Clause</td></tr></table>\n", ErrContainsHTML},
		{"nothing to read", "[](https://example.com)\n", ErrNoVisibleText},
		{"only a comment", "<!-- draft -->\n", ErrNoVisibleText},
		{"only invisible characters", "\u200b\u2060\n", ErrNoVisibleText},
		{"not utf-8", "\xff\xfeA", ErrNotUTF8},
		{"too large", strings.Repeat("a\n", MaxTermsSize/2+1), ErrTooLarge},
		{"longest allowed line", strings.Repeat("a", maxLineLength), nil},
		{"line too long", strings.Repeat("a", maxLineLength+1), ErrLineTooLong},
		{"nested quotes", strings.Repeat("> ", 40) + "Clause\n", ErrNestedTooDeep},
		{"nested lists", strings.Repeat("- ", 40) + "Clause\n", ErrNestedTooDeep},
		{"long horizontal rule", strings.Repeat("-", 200) + "\n\nTerms.\n", nil},
		{"indented plain text", strings.Repeat(" ", 60) + "Terms of use\n", nil},
		{"text table border", "+" + strings.Repeat("-", 70) + "+\n", nil},
		{"long number", strings.Repeat("1", 70) + " units\n", nil},
		{"dashes then text", strings.Repeat("-", 65) + "Terms\n", nil},
		{"indentation too deep", strings.Repeat(" ", 300) + "Clause\n", ErrNestedTooDeep},
		{"tables at the cell budget", termsTable(100, 99), nil},
		{"tables over the cell budget", termsTable(100, 100), ErrTableTooLarge},
		{"cell budget spans tables", termsTable(100, 49) + "\n" + termsTable(100, 50), ErrTableTooLarge},
		{"setext heading is not a table", "Terms\n---\n" + strings.Repeat("Clause.\n", 5_000), nil},
		{"rendered too large", strings.Repeat(strings.Repeat("`<` ", 2_000)+"\n", 64), ErrRenderedTooLarge},
		{"exactly the line limit", strings.Repeat("a\n", maxLines), nil},
		{"too many lines", strings.Repeat("a\n", maxLines+1), ErrTooManyLines},
		{"too many lines, no final newline", strings.Repeat("a\n", maxLines) + "a", ErrTooManyLines},
		{"too much formatting in one paragraph", strings.Repeat("**a** ", 600) + "\n", ErrTooComplex},
		{"a blank line ends the formatting count", strings.Repeat("**a** ", 300) + "\n\n" + strings.Repeat("**a** ", 300) + "\n", nil},
		{"too many html comments", strings.Repeat("<!-- c -->\n\n", 33) + "Terms.\n", ErrContainsHTML},
		{"mixed rule characters nest lists", strings.Repeat("- * ", 20) + "\n", ErrNestedTooDeep},
		{"spaced horizontal rule", strings.Repeat("- ", 40) + "\n\nTerms.\n", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateTerms([]byte(tc.src))
			if tc.want == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.want)
		})
	}
}
