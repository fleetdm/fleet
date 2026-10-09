package markdown

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
	"pgregory.net/rapid"
)

// Documents are built from fragments that reach every kind of node the renderer handles, including raw HTML that
// must never come out as is.
var termsFragments = []string{
	"# Terms", "## Section", "Plain text.", "**bold** and *em* and ~~gone~~", "`code`", "```go\nx := 1\n```",
	"- item", "1. first", "> quoted", "- [x] done", "---", "| a | b |\n|:--|--:|\n| 1 | 2 |",
	"[link](https://example.com)", "![alt](https://example.com/x.png)", "<https://example.com>", "line<br>break",
	"<u>under</u>", "<div>\nblock\n</div>", "<!-- note -->", "<script>alert(1)</script>",
	`<img src=x onerror="alert(1)">`, `<a href="javascript:alert(1)">x</a>`, `<p style="color:red">styled</p>`,
	"&lt;not a tag&gt;", "&#x3C;script&#x3E;", "\u200b", "émoji 🎉", "*", "_", "[", "]", "(", ")", "<", ">", "\t", "    indented",
}

func genTerms(t *rapid.T) []byte {
	parts := rapid.SliceOfN(rapid.SampledFrom(termsFragments), 1, 30).Draw(t, "fragments")
	seps := rapid.SliceOfN(rapid.SampledFrom([]string{"\n", "\n\n", " ", ""}), len(parts), len(parts)).Draw(t, "separators")
	var b strings.Builder
	for i, p := range parts {
		b.WriteString(p)
		b.WriteString(seps[i])
	}
	return []byte(b.String())
}

// allowedAttr mirrors termsPolicy: the attributes each element may keep, and the values they may take.
var allowedAttr = map[string]map[string]*regexp.Regexp{
	"th":    {"align": regexp.MustCompile(`^(left|center|right)$`)},
	"td":    {"align": regexp.MustCompile(`^(left|center|right)$`)},
	"ol":    {"start": regexp.MustCompile(`^[0-9]+$`)},
	"code":  {"class": regexp.MustCompile(`^language-[a-zA-Z0-9_+-]+$`)},
	"input": {"type": regexp.MustCompile(`^checkbox$`), "checked": regexp.MustCompile(`^$`), "disabled": regexp.MustCompile(`^$`)},
}

var allowedElements = func() map[string]struct{} {
	set := map[string]struct{}{}
	for _, e := range []string{
		"p", "br", "hr", "h1", "h2", "h3", "h4", "h5", "h6", "ul", "ol", "li", "strong", "em", "b", "i", "del", "s",
		"code", "pre", "blockquote", "table", "thead", "tbody", "tr", "th", "td", "input",
	} {
		set[e] = struct{}{}
	}
	return set
}()

func TestRenderTermsProperties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		src := genTerms(t)
		out, renderErr := RenderTerms(src)

		// Only allowlisted elements and attributes come out, whatever the input.
		z := html.NewTokenizer(strings.NewReader(out))
		for tt := z.Next(); tt != html.ErrorToken; tt = z.Next() {
			if tt != html.StartTagToken && tt != html.SelfClosingTagToken {
				continue
			}
			tok := z.Token()
			_, allowed := allowedElements[tok.Data]
			require.True(t, allowed, "element %q in %q", tok.Data, out)
			for _, a := range tok.Attr {
				re, ok := allowedAttr[tok.Data][a.Key]
				require.True(t, ok, "attribute %q on %q in %q", a.Key, tok.Data, out)
				require.Regexp(t, re, a.Val)
			}
		}

		// An upload that passes validation renders to a page with something to read, so the device never gets an
		// error or an empty page for a file the admin was allowed to upload.
		if ValidateTerms(src) == nil {
			require.NoError(t, renderErr)
			require.NotEmpty(t, out)
		}
	})
}
