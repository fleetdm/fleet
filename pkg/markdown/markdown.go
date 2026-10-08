// Package markdown renders admin-supplied markdown into HTML that is safe to
// embed in pages Fleet serves to end users.
package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"regexp"
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
)

// RenderTerms converts markdown to sanitized HTML for a terms page shown in an
// OS-controlled web view during enrollment. That view is a captive page with
// no address bar or back button, so links are flattened to their text and
// images to their alt text: anything the user could navigate to or any
// resource fetched from a third party would strand the enrollment.
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

var (
	// ErrTooLarge means the document is larger than MaxTermsSize.
	ErrTooLarge = errors.New("is larger than 512 KB")
	// ErrNotUTF8 means the document is not valid UTF-8 text.
	ErrNotUTF8 = errors.New("is not valid UTF-8 text")
	// ErrLineTooLong means a line is longer than the parser handles cheaply.
	ErrLineTooLong = errors.New("has a line longer than 8 KB")
	// ErrTooManyLines means the document has more lines than any agreement needs.
	ErrTooManyLines = errors.New("has more than 10,000 lines")
	// ErrTooComplex means a paragraph or list has more emphasis and link
	// characters than the parser handles cheaply.
	ErrTooComplex = errors.New("has too much formatting in one paragraph or list")
	// ErrNestedTooDeep means lists or quotes are nested beyond any real use.
	ErrNestedTooDeep = errors.New("nests lists or quotes too deeply")
	// ErrTableTooLarge means the tables have more cells than a page can show.
	ErrTableTooLarge = errors.New("has tables with more than 10,000 cells in total")
	// ErrRenderedTooLarge means the rendered page is too large to serve.
	ErrRenderedTooLarge = errors.New("renders to more than 2 MB")
	// ErrContainsHTML means the document has HTML blocks with text in them.
	// Those blocks are dropped from the rendered page, text included.
	ErrContainsHTML = errors.New("contains HTML, which isn't shown; convert it to markdown")
	// ErrNoVisibleText means the rendered page would have nothing to read.
	ErrNoVisibleText = errors.New("has no text to show")
)

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

	// Inline tags are dropped but keep their text; a block of HTML is dropped
	// whole. Comments and blocks without text lose nothing, so they are fine.
	var htmlWithText bool
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if b, ok := n.(*ast.HTMLBlock); entering && ok && hasVisibleText(htmlBlockSource(b, src)) {
			htmlWithText = true
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	if htmlWithText {
		return ErrContainsHTML
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

	rendered = termsPolicy.Sanitize(buf.String())
	if !hasVisibleText(rendered) {
		return "", nil
	}
	return rendered, nil
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
// break; other inline HTML is dropped by the renderer.
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

// termsPolicy is the structural subset of HTML the terms page needs. goldmark
// already drops raw HTML from the source; this is defense in depth, and it is
// also what strips anything a future goldmark extension might add.
var termsPolicy = func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements(
		"p", "br", "hr",
		"h1", "h2", "h3", "h4", "h5", "h6",
		"ul", "ol", "li",
		"strong", "em", "b", "i", "del", "s", "code", "pre", "blockquote",
		"table", "thead", "tbody", "tr", "th", "td",
	)
	p.AllowAttrs("align").Matching(regexp.MustCompile(`^(left|center|right)$`)).OnElements("th", "td")
	p.AllowAttrs("start").Matching(bluemonday.Integer).OnElements("ol")
	p.AllowAttrs("class").Matching(regexp.MustCompile(`^language-[a-zA-Z0-9_+-]+$`)).OnElements("code")
	// GFM task list checkboxes.
	p.AllowAttrs("type").Matching(regexp.MustCompile(`^checkbox$`)).OnElements("input")
	p.AllowAttrs("checked", "disabled").OnElements("input")
	return p
}()
