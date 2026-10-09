// Package markdown renders admin-supplied markdown into HTML that is safe to
// embed in pages Fleet serves to end users.
package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// RenderTerms converts markdown to sanitized HTML for a terms page shown in an
// OS-controlled web view during enrollment. That view is a captive page with
// no address bar or back button, so links are flattened to their text and
// images to their alt text: anything the user could navigate to or any
// resource fetched from a third party would strand the enrollment. Raw HTML is
// limited to what rawHTMLTransformer keeps.
//
// It returns an empty string when the result has no visible text, so callers
// fall back to their default page rather than show an empty one.
func RenderTerms(src []byte) (string, error) {
	src = trimBOM(src)
	if err := checkLimits(src); err != nil {
		return "", err
	}
	md := newTermsMarkdown()
	doc, err := parse(md, src)
	if err != nil {
		return "", err
	}
	return renderDoc(md, src, doc)
}

// MaxTermsSize is the largest document accepted. Real agreements are a few
// tens of KB; the limit bounds the cost of inputs goldmark handles slowly.
const MaxTermsSize = 512 << 10

// goldmark has no work budget. Emphasis and link matching are quadratic in the
// delimiters of a paragraph, an unclosed HTML comment or declaration scans the
// rest of its paragraph, the parser keeps a record per open block per line, and
// a table pads every row to the header's width. These limits keep the worst
// document cheap to parse and render.
const (
	maxLineLength   = 8 << 10
	maxLines        = 10_000
	maxNestedBlocks = 16 // quote and list markers opening one line
	maxIndentation  = 64 // leading whitespace, which keeps list items open
	maxRunMarkup    = 2_000
	maxHTMLOpeners  = 32
	maxTableCells   = 10_000
	maxRenderedSize = 2 << 20
)

// TermsError is a reason a document can't be used as terms. Error reads after
// the document's name, as in "terms.md is larger than 512 KB"; Message is
// written for the person who uploaded it.
type TermsError struct {
	reason  string
	Message string
}

func (e *TermsError) Error() string { return e.reason }

var (
	// ErrTooLarge means the document is larger than MaxTermsSize.
	ErrTooLarge = &TermsError{
		fmt.Sprintf("is larger than %d KB", MaxTermsSize>>10),
		fmt.Sprintf("The file must be %d KB or smaller.", MaxTermsSize>>10),
	}
	// ErrNotUTF8 means the document is not valid UTF-8 text.
	ErrNotUTF8 = &TermsError{"is not valid UTF-8 text", "The file must be UTF-8 text."}
	// ErrLineTooLong means a line is longer than the parser handles cheaply.
	ErrLineTooLong = &TermsError{
		fmt.Sprintf("has a line longer than %d KB", maxLineLength>>10),
		fmt.Sprintf("The file has a line longer than %d KB. Split long paragraphs into shorter lines and upload again.", maxLineLength>>10),
	}
	// ErrTooManyLines means the document has more lines than any agreement needs.
	ErrTooManyLines = &TermsError{
		fmt.Sprintf("has more than %s lines", thousands(maxLines)),
		fmt.Sprintf("The file has more than %s lines.", thousands(maxLines)),
	}
	// ErrTooComplex means a paragraph or list has more emphasis and link
	// characters than the parser handles cheaply.
	ErrTooComplex = &TermsError{
		"has too much formatting in one paragraph or list",
		"The file has too much formatting in one paragraph or list. Add blank lines between paragraphs and upload again.",
	}
	// ErrNestedTooDeep means lists, quotes or HTML are nested beyond any real use.
	ErrNestedTooDeep = &TermsError{"nests lists, quotes or HTML too deeply", "The file nests lists, quotes or HTML too deeply."}
	// ErrTableTooLarge means the tables have more cells than a page can show.
	ErrTableTooLarge = &TermsError{
		fmt.Sprintf("has tables with more than %s cells in total", thousands(maxTableCells)),
		fmt.Sprintf("The file's tables have more than %s cells in total.", thousands(maxTableCells)),
	}
	// ErrRenderedTooLarge means the rendered page is too large to serve.
	ErrRenderedTooLarge = &TermsError{
		fmt.Sprintf("renders to more than %d MB", maxRenderedSize>>20),
		"The file is too large to show. Make it shorter and upload again.",
	}
	// ErrContainsHTML means the document has HTML blocks with text in them that
	// aren't tables or text styles in the shape rewriteHTMLBlock accepts. Those
	// blocks are dropped from the rendered page, text included.
	ErrContainsHTML = &TermsError{
		"contains HTML other than tables and text styles, which isn't shown; convert it to markdown",
		"The file contains HTML other than tables and text styles. Convert it to markdown and upload again.",
	}
	// ErrNoVisibleText means the rendered page would have nothing to read.
	ErrNoVisibleText = &TermsError{"has no text to show", "The file has no text to show."}
)

// thousands formats a limit the way the messages print it: 10000 as "10,000".
func thousands(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// ValidateTerms reports what an admin should fix before a document is used as
// terms: anything that would silently lose text or leave the page empty.
func ValidateTerms(src []byte) error {
	src = trimBOM(src)
	if err := checkLimits(src); err != nil {
		return err
	}

	md := newTermsMarkdown()
	doc, err := parse(md, src)
	if err != nil {
		return err
	}

	// Inline tags that aren't kept are dropped but keep their text; an HTML
	// block that isn't kept is dropped whole. Comments and blocks without text
	// lose nothing, so they are fine.
	var htmlErr error
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if err, ok := n.AttributeString(rejectedHTMLAttr); entering && ok {
			htmlErr = err.(error)
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	if htmlErr != nil {
		return htmlErr
	}

	rendered, err := renderDoc(md, src, doc)
	if err != nil {
		return err
	}
	if rendered == "" {
		return ErrNoVisibleText
	}
	return nil
}

// checkLimits rejects documents that would be slow to parse or render, before
// parsing. Tables are budgeted inside the parser instead, where goldmark's own
// paragraphs are known. A run of non-blank lines contains whole paragraphs, so
// counting per run errs toward rejecting.
func checkLimits(src []byte) error {
	if len(src) > MaxTermsSize {
		return ErrTooLarge
	}
	if !utf8.Valid(src) {
		return ErrNotUTF8
	}
	lines := bytes.Count(src, []byte("\n"))
	if len(src) > 0 && src[len(src)-1] != '\n' {
		lines++
	}
	if lines > maxLines {
		return ErrTooManyLines
	}
	if bytes.Count(src, []byte("<!"))+bytes.Count(src, []byte("<?")) > maxHTMLOpeners {
		return ErrContainsHTML
	}

	runMarkup := 0
	for rest := src; len(rest) > 0; {
		var line []byte
		line, rest, _ = bytes.Cut(rest, []byte("\n"))
		if len(line) > maxLineLength {
			return ErrLineTooLong
		}
		if markers, indent := lineNesting(line); (markers > maxNestedBlocks || indent > maxIndentation) && !isThematicBreak(line) {
			return ErrNestedTooDeep
		}
		// Blank as goldmark defines it: only space, tab and CR.
		if util.IsBlank(line) {
			runMarkup = 0
			continue
		}
		for _, c := range line {
			if c == '*' || c == '_' || c == '~' || c == '[' || c == ']' {
				runMarkup++
			}
		}
		if runMarkup > maxRunMarkup {
			return ErrTooComplex
		}
	}
	return nil
}

// trimBOM drops a leading UTF-8 byte order mark, which editors on Windows add
// and which would otherwise turn a first-line heading into plain text.
func trimBOM(src []byte) []byte {
	return bytes.TrimPrefix(src, []byte("\xef\xbb\xbf"))
}

// lineNesting counts the quote and list markers that open a line, each of which
// opens a block, and its leading whitespace, which keeps list items open across
// lines. Text, digits and rules open nothing.
func lineNesting(line []byte) (markers, indent int) {
	for i := 0; i < len(line); {
		switch c := line[i]; {
		case c == ' ':
			indent++
			i++
		case c == '\t':
			indent += 4
			i++
		case c == '>':
			markers++
			i++
		case (c == '-' || c == '*' || c == '+') && endsListMarker(line, i+1):
			markers++
			i++
		case c >= '0' && c <= '9':
			j := i
			for j < len(line) && j-i < 9 && line[j] >= '0' && line[j] <= '9' {
				j++
			}
			if j == len(line) || (line[j] != '.' && line[j] != ')') || !endsListMarker(line, j+1) {
				return markers, indent
			}
			markers++
			i = j + 1
		default:
			return markers, indent
		}
	}
	return markers, indent
}

func endsListMarker(line []byte, i int) bool {
	return i == len(line) || line[i] == ' ' || line[i] == '\t'
}

// isThematicBreak reports a horizontal rule: three or more of the same '-', '*'
// or '_', with optional spaces. Mixed characters are nested lists to goldmark.
func isThematicBreak(line []byte) bool {
	t := bytes.Trim(line, " \t\r")
	if len(t) < 3 || (t[0] != '-' && t[0] != '*' && t[0] != '_') {
		return false
	}
	return len(bytes.Trim(t, string(t[0])+" \t")) == 0
}

// budgetedTables is goldmark's table extension with a cell budget. goldmark pads
// every row to the delimiter row's width, so a table costs columns times rows
// however short its rows are; the budget is checked on goldmark's own paragraph
// lines before the table is built, and covers all tables in the document.
type budgetedTables struct{}

func (budgetedTables) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithParagraphTransformers(util.Prioritized(tableBudget{extension.NewTableParagraphTransformer()}, 200)),
		parser.WithASTTransformers(util.Prioritized(extension.NewTableASTTransformer(), 0)),
	)
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		// The default puts alignment in a style attribute, which the sanitizer
		// drops, so use the align attribute it allows.
		util.Prioritized(extension.NewTableHTMLRenderer(extension.WithTableCellAlignMethod(extension.TableCellAlignAttribute)), 500),
	))
}

var tableCellsKey = parser.NewContextKey()

type tableBudget struct {
	inner parser.ParagraphTransformer
}

func (t tableBudget) Transform(p *ast.Paragraph, reader text.Reader, pc parser.Context) {
	lines := p.Lines()
	for i := 1; i < lines.Len(); i++ {
		seg := lines.At(i)
		cols := tableDelimiterColumns(seg.Value(reader.Source()))
		if cols == 0 {
			continue
		}
		// goldmark builds one table, from the first delimiter row: the line
		// before it is the header and every line after it a row.
		used, _ := pc.Get(tableCellsKey).(int)
		used += cols * (lines.Len() - i)
		pc.Set(tableCellsKey, used)
		if used > maxTableCells {
			return // leave it as text; parse reports the error
		}
		break
	}
	t.inner.Transform(p, reader, pc)
}

var tableDelimiterCell = regexp.MustCompile(`^\s*:?-+:?\s*$`)

// tableDelimiterColumns mirrors goldmark's table delimiter check
// (extension/table.go, isTableDelim and parseDelimiter) and returns the column
// count, or 0 when line is not a delimiter row.
func tableDelimiterColumns(line []byte) int {
	if w, _ := util.IndentWidth(line, 0); w > 3 {
		return 0
	}
	allDash := true
	for _, b := range line {
		if b != '-' {
			allDash = false
		}
		if !util.IsSpace(b) && b != '-' && b != '|' && b != ':' {
			return 0
		}
	}
	if allDash {
		return 0
	}
	cols := bytes.Split(line, []byte{'|'})
	if util.IsBlank(cols[0]) {
		cols = cols[1:]
	}
	if len(cols) > 0 && util.IsBlank(cols[len(cols)-1]) {
		cols = cols[:len(cols)-1]
	}
	for _, c := range cols {
		if !tableDelimiterCell.Match(c) {
			return 0
		}
	}
	return len(cols)
}

// GFM minus Linkify: a bare URL stays ordinary text instead of becoming a link.
func newTermsMarkdown() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(budgetedTables{}, extension.Strikethrough, extension.TaskList),
		goldmark.WithParserOptions(parser.WithASTTransformers(
			util.Prioritized(flattenTransformer{}, 100),
			util.Prioritized(rawHTMLTransformer{}, 90),
		)),
		goldmark.WithRendererOptions(renderer.WithNodeRenderers(
			util.Prioritized(rawHTMLRenderer{}, 500),
		)),
	)
}

// The terms page is on the enrollment critical path, where a panic would drop
// the connection, so parse and renderDoc report one as an error instead.

func parse(md goldmark.Markdown, src []byte) (doc ast.Node, err error) {
	defer func() {
		if r := recover(); r != nil {
			doc, err = nil, fmt.Errorf("parse markdown: panic: %v", r)
		}
	}()
	pc := parser.NewContext()
	doc = md.Parser().Parse(text.NewReader(src), parser.WithContext(pc))
	if used, _ := pc.Get(tableCellsKey).(int); used > maxTableCells {
		return nil, ErrTableTooLarge
	}
	return doc, nil
}

func renderDoc(md goldmark.Markdown, src []byte, doc ast.Node) (rendered string, err error) {
	defer func() {
		if r := recover(); r != nil {
			rendered, err = "", fmt.Errorf("render markdown: panic: %v", r)
		}
	}()

	// Writes past the limit fail, so rendering stops growing the buffer there.
	var buf cappedBuffer
	if err := md.Renderer().Render(&buf, src, doc); err != nil {
		if errors.Is(err, ErrRenderedTooLarge) {
			return "", ErrRenderedTooLarge
		}
		return "", fmt.Errorf("render markdown: %w", err)
	}

	rendered, err = balance(termsPolicy.Sanitize(buf.String()))
	if err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}
	// Escaping can make the page larger than what goldmark wrote.
	if len(rendered) > maxRenderedSize {
		return "", ErrRenderedTooLarge
	}
	if !hasVisibleText(rendered) {
		return "", nil
	}
	return rendered, nil
}

// balance closes anything left open, so the document can't take in the page
// around it, such as the button that accepts the terms. Raw HTML is rebuilt
// well formed, so this is defense in depth.
func balance(s string) (string, error) {
	nodes, err := xhtml.ParseFragment(strings.NewReader(s), &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div})
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, n := range nodes {
		if err := xhtml.Render(&b, n); err != nil {
			return "", err
		}
	}
	return b.String(), nil
}

type cappedBuffer struct {
	bytes.Buffer
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > maxRenderedSize {
		return 0, ErrRenderedTooLarge
	}
	return b.Buffer.Write(p)
}

// textPolicy keeps only text, dropping every tag and the contents of script
// and style, which is what a reader of the page would see.
var textPolicy = bluemonday.StrictPolicy()

// hasVisibleText ignores invisible format characters such as a zero-width
// space, which TrimSpace keeps.
func hasVisibleText(s string) bool {
	return strings.IndexFunc(html.UnescapeString(textPolicy.Sanitize(s)), func(r rune) bool {
		return !unicode.IsSpace(r) && !unicode.Is(unicode.Cf, r)
	}) >= 0
}

func htmlBlockSource(b *ast.HTMLBlock, src []byte) string {
	var buf bytes.Buffer
	lines := b.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		buf.Write(seg.Value(src))
	}
	if b.HasClosure() {
		buf.Write(b.ClosureLine.Value(src))
	}
	return buf.String()
}

// flattenTransformer replaces links and images with their inline content, so
// the rendered page has no navigation and no remote fetches. It also keeps an
// inline <br>, the only way to break a line inside a table cell, as a line
// break; rawHTMLTransformer decides what other inline HTML is kept.
type flattenTransformer struct{}

var inlineBreak = regexp.MustCompile(`(?i)^<br\s*/?>$`)

func (flattenTransformer) Transform(doc *ast.Document, reader text.Reader, _ parser.Context) {
	source := reader.Source()
	var flatten, breaks []ast.Node
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *ast.Link, *ast.Image, *ast.AutoLink:
			flatten = append(flatten, n)
		case *ast.RawHTML:
			if inlineBreak.Match(n.Segments.Value(source)) {
				breaks = append(breaks, n)
			}
		}
		return ast.WalkContinue, nil
	})

	for _, n := range breaks {
		if parent := n.Parent(); parent != nil {
			br := ast.NewText()
			br.SetHardLineBreak(true)
			parent.ReplaceChild(parent, n, br)
		}
	}

	for _, n := range flatten {
		parent := n.Parent()
		if parent == nil {
			continue
		}
		if al, ok := n.(*ast.AutoLink); ok {
			// An autolink has no children; keep the URL as plain text.
			parent.ReplaceChild(parent, n, ast.NewString(al.URL(source)))
			continue
		}
		for child := n.FirstChild(); child != nil; {
			next := child.NextSibling()
			parent.InsertBefore(parent, n, child)
			child = next
		}
		parent.RemoveChild(parent, n)
	}
}

// termsPolicy is the structural subset of HTML the terms page needs. Raw HTML
// is already rebuilt from allowed tags; this is defense in depth, and it is also
// what strips anything a future goldmark extension might add.
var termsPolicy = func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements(
		"p", "br", "hr",
		"h1", "h2", "h3", "h4", "h5", "h6",
		"ul", "ol", "li",
		"strong", "em", "b", "i", "u", "del", "s", "sub", "sup", "code", "pre", "blockquote",
		"table", "caption", "colgroup", "col", "thead", "tbody", "tfoot", "tr", "th", "td",
	)
	p.AllowAttrs("align").Matching(regexp.MustCompile(`^(left|center|right)$`)).OnElements("th", "td")
	p.AllowAttrs("colspan", "rowspan").Matching(spanValue).OnElements("th", "td")
	p.AllowAttrs("start").Matching(bluemonday.Integer).OnElements("ol")
	p.AllowAttrs("class").Matching(regexp.MustCompile(`^language-[a-zA-Z0-9_+-]+$`)).OnElements("code")
	// GFM task list checkboxes.
	p.AllowAttrs("type").Matching(regexp.MustCompile(`^checkbox$`)).OnElements("input")
	p.AllowAttrs("checked", "disabled").OnElements("input")
	return p
}()

// Raw HTML is kept only in the shapes word processors export, through pandoc's
// GitHub flavor, for what markdown can't express: tables with merged cells, and
// underline, subscript and superscript. It is rebuilt from its tags rather than
// passed on, so nothing the browser's parser would repair, such as an unclosed
// element or text loose in a table, reaches the page: repairs can reorder text,
// hide it, or multiply the page's size.

// phrasingTags are the inline tags raw HTML can use.
var phrasingTags = map[string]struct{}{
	"b": {}, "i": {}, "u": {}, "s": {}, "em": {}, "strong": {}, "del": {}, "code": {}, "sub": {}, "sup": {},
}

const (
	maxHTMLNesting   = 32
	maxInlineNesting = 16
)

// allowedChild is the HTML content model, narrowed to what the terms page
// shows. a and span are dropped keeping their text and img becomes its alt
// text, as markdown links and images do.
func allowedChild(parent, child string) bool {
	_, phrasing := phrasingTags[child]
	phrasing = phrasing || child == "br" || child == "a" || child == "span" || child == "img"
	switch parent {
	case "table":
		return child == "caption" || child == "colgroup" || child == "thead" || child == "tbody" || child == "tfoot" || child == "tr"
	case "colgroup":
		return child == "col"
	case "thead", "tbody", "tfoot":
		return child == "tr"
	case "tr":
		return child == "th" || child == "td"
	case "ul", "ol":
		return child == "li"
	case "", "li", "th", "td":
		return phrasing || child == "p" || child == "ul" || child == "ol" || child == "table"
	default: // p, caption and phrasing elements
		return phrasing
	}
}

var spanValue = regexp.MustCompile(`^([1-9][0-9]?|100)$`)

// rewriteHTMLBlock rebuilds an HTML block from its tokens, or reports why it
// can't be shown. Every element must be allowed where it is and closed in
// order; only th and td keep attributes. area is the grid cells its table cells
// cover, charged to the same budget as markdown tables.
func rewriteHTMLBlock(block string) (out string, area int, err error) {
	var b strings.Builder
	open := make([]string, 0, maxHTMLNesting)
	z := xhtml.NewTokenizer(strings.NewReader(block))
	for {
		tt := z.Next()
		parent := ""
		if len(open) > 0 {
			parent = open[len(open)-1]
		}
		switch tt {
		case xhtml.ErrorToken:
			// Raw bytes left over are a tag cut off by the end of the block.
			if !errors.Is(z.Err(), io.EOF) || len(z.Raw()) > 0 || len(open) > 0 {
				return "", area, ErrContainsHTML
			}
			return b.String(), area, nil
		case xhtml.TextToken:
			text := strings.ReplaceAll(string(z.Text()), "\x00", "\ufffd")
			// Text goes where text-level tags do; loose in a table, the browser moves it.
			if strings.TrimSpace(text) != "" && !allowedChild(parent, "b") {
				return "", area, ErrContainsHTML
			}
			b.WriteString(xhtml.EscapeString(text))
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			name, hasAttr := z.TagName()
			tag := string(name)
			void := tag == "br" || tag == "col" || tag == "img"
			if !allowedChild(parent, tag) || (tt == xhtml.SelfClosingTagToken && !void) {
				return "", area, ErrContainsHTML
			}
			attrs := map[string]string{}
			for hasAttr {
				var key, val []byte
				key, val, hasAttr = z.TagAttr()
				attrs[string(key)] = string(val)
			}
			switch tag {
			case "a", "span":
			case "img":
				b.WriteString(xhtml.EscapeString(attrs["alt"]))
			case "th", "td":
				colspan, rowspan := 1, 1
				b.WriteString("<" + tag)
				if a := attrs["align"]; a == "left" || a == "center" || a == "right" {
					b.WriteString(` align="` + a + `"`)
				}
				if v := attrs["colspan"]; spanValue.MatchString(v) {
					colspan, _ = strconv.Atoi(v)
					b.WriteString(` colspan="` + v + `"`)
				}
				if v := attrs["rowspan"]; spanValue.MatchString(v) {
					rowspan, _ = strconv.Atoi(v)
					b.WriteString(` rowspan="` + v + `"`)
				}
				b.WriteString(">")
				area += colspan * rowspan
			default:
				b.WriteString("<" + tag + ">")
			}
			if !void {
				if len(open) == maxHTMLNesting {
					return "", area, ErrNestedTooDeep
				}
				open = append(open, tag)
			}
		case xhtml.EndTagToken:
			name, _ := z.TagName()
			tag := string(name)
			if parent != tag {
				return "", area, ErrContainsHTML
			}
			open = open[:len(open)-1]
			if tag != "a" && tag != "span" {
				b.WriteString("</" + tag + ">")
			}
		default: // comments, CDATA, processing instructions, doctypes
			return "", area, ErrContainsHTML
		}
	}
}

// inlineTag matches an inline open or closing tag and its name.
var inlineTag = regexp.MustCompile(`^<(/?)([A-Za-z][A-Za-z0-9]*)[\s/>]`)

// pairInlineTags keeps raw inline tags that open and close among one parent's
// children, rebuilt without attributes. Pairs can't straddle markdown's own
// elements, and an unclosed tag can't style the rest of the page; the others
// are dropped, keeping the text between them.
func pairInlineTags(parent ast.Node, source []byte) {
	type openTag struct {
		node *ast.RawHTML
		name string
	}
	open := make([]openTag, 0, maxInlineNesting)
	for c := parent.FirstChild(); c != nil; c = c.NextSibling() {
		raw, ok := c.(*ast.RawHTML)
		if !ok {
			continue
		}
		m := inlineTag.FindSubmatch(raw.Segments.Value(source))
		if m == nil {
			continue
		}
		name := strings.ToLower(string(m[2]))
		if _, ok := phrasingTags[name]; !ok {
			continue
		}
		if len(m[1]) == 0 {
			if len(open) < maxInlineNesting {
				open = append(open, openTag{raw, name})
			}
			continue
		}
		if n := len(open); n > 0 && open[n-1].name == name {
			open[n-1].node.SetAttributeString(keptHTMLAttr, "<"+name+">")
			raw.SetAttributeString(keptHTMLAttr, "</"+name+">")
			open = open[:n-1]
		}
	}
}

const (
	// keptHTMLAttr holds the HTML a raw HTML node renders as.
	keptHTMLAttr = "fleet-kept-html"
	// rejectedHTMLAttr holds why an HTML block with text can't be shown.
	rejectedHTMLAttr = "fleet-rejected-html"
)

// rawHTMLTransformer decides, while parsing, what raw HTML is kept, and charges
// HTML table cells to the tables' cell budget.
type rawHTMLTransformer struct{}

func (rawHTMLTransformer) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	source := reader.Source()
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		b, ok := n.(*ast.HTMLBlock)
		if !ok {
			pairInlineTags(n, source)
			return ast.WalkContinue, nil
		}
		block := htmlBlockSource(b, source)
		out, area, err := rewriteHTMLBlock(block)
		used, _ := pc.Get(tableCellsKey).(int)
		pc.Set(tableCellsKey, used+area)
		switch {
		case err == nil:
			b.SetAttributeString(keptHTMLAttr, out)
		case hasVisibleText(block):
			b.SetAttributeString(rejectedHTMLAttr, err)
		}
		return ast.WalkSkipChildren, nil
	})
}

// rawHTMLRenderer writes the HTML rawHTMLTransformer kept, and nothing for the
// rest, where goldmark would drop all raw HTML.
type rawHTMLRenderer struct{}

func (rawHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindHTMLBlock, renderKeptHTML)
	reg.Register(ast.KindRawHTML, renderKeptHTML)
}

func renderKeptHTML(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		if kept, ok := node.AttributeString(keptHTMLAttr); ok {
			_, _ = w.WriteString(kept.(string))
		}
	}
	return ast.WalkSkipChildren, nil
}
