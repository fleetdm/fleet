package ddmpredicate

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ParseError describes why a predicate failed to parse. Pos is a byte
// offset into the input.
type ParseError struct {
	Pos int
	Msg string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("invalid predicate at offset %d: %s", e.Pos, e.Msg)
}

type tokKind int

const (
	tokEOF tokKind = iota
	tokIdent
	tokNumber
	tokString
	tokVariable // $name; text holds name
	tokAt       // @name; text holds name
	tokFormat   // %@, %K, %d, ...; text holds the raw sequence
	tokPunct
)

type token struct {
	kind    tokKind
	text    string
	escaped bool // tokIdent written as #name
	pos     int
}

func (t token) isPunct(s string) bool { return t.kind == tokPunct && t.text == s }

// keyword returns the uppercased text for unescaped identifiers, so keyword
// matching is case-insensitive; escaped identifiers (#name) never match.
func (t token) keyword() string {
	if t.kind != tokIdent || t.escaped {
		return ""
	}
	return strings.ToUpper(t.text)
}

func (t token) describe() string {
	switch t.kind {
	case tokEOF:
		return "end of predicate"
	case tokString:
		return fmt.Sprintf("string %s", (&StringLiteral{Value: t.text}).String())
	case tokVariable:
		return fmt.Sprintf("'$%s'", t.text)
	case tokAt:
		return fmt.Sprintf("'@%s'", t.text)
	default:
		return fmt.Sprintf("'%s'", t.text)
	}
}

var twoCharPuncts = []string{"**", "==", "!=", "<>", "<=", ">=", "=<", "=>", ":=", "&&", "||"}

const oneCharPuncts = "()[]{},.+-*/=<>!"

type lexer struct {
	input string
	pos   int
}

func (l *lexer) errorf(pos int, format string, args ...any) error {
	return &ParseError{Pos: pos, Msg: fmt.Sprintf(format, args...)}
}

func (l *lexer) skipSpace() {
	for l.pos < len(l.input) {
		r, size := utf8.DecodeRuneInString(l.input[l.pos:])
		if !unicode.IsSpace(r) {
			return
		}
		l.pos += size
	}
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentChar(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func (l *lexer) next() (token, error) {
	l.skipSpace()
	start := l.pos
	if l.pos >= len(l.input) {
		return token{kind: tokEOF, pos: start}, nil
	}

	c := l.input[l.pos]
	switch {
	case isIdentStart(c):
		return token{kind: tokIdent, text: l.scanIdent(), pos: start}, nil
	case isDigit(c) || (c == '.' && l.pos+1 < len(l.input) && isDigit(l.input[l.pos+1])):
		return token{kind: tokNumber, text: l.scanNumber(), pos: start}, nil
	case c == '\'' || c == '"':
		text, err := l.scanString()
		if err != nil {
			return token{}, err
		}
		return token{kind: tokString, text: text, pos: start}, nil
	case c == '#':
		l.pos++
		if l.pos >= len(l.input) || !isIdentStart(l.input[l.pos]) {
			return token{}, l.errorf(start, "expected an identifier after '#'")
		}
		return token{kind: tokIdent, text: l.scanIdent(), escaped: true, pos: start}, nil
	case c == '$':
		l.pos++
		if l.pos >= len(l.input) || !isIdentStart(l.input[l.pos]) {
			return token{}, l.errorf(start, "expected a variable name after '$'")
		}
		return token{kind: tokVariable, text: l.scanIdent(), pos: start}, nil
	case c == '@':
		l.pos++
		if l.pos >= len(l.input) || !isIdentStart(l.input[l.pos]) {
			return token{}, l.errorf(start, "expected a key path function after '@', such as @status(...), @key(...), or @property(...)")
		}
		return token{kind: tokAt, text: l.scanIdent(), pos: start}, nil
	case c == '%':
		l.pos++
		for l.pos < len(l.input) && l.pos-start < 5 {
			c := l.input[l.pos]
			if c == '@' || c == 'K' || c == '%' {
				l.pos++
				break
			}
			if !isIdentStart(c) {
				break
			}
			l.pos++
		}
		return token{kind: tokFormat, text: l.input[start:l.pos], pos: start}, nil
	}

	for _, p := range twoCharPuncts {
		if strings.HasPrefix(l.input[l.pos:], p) {
			l.pos += len(p)
			return token{kind: tokPunct, text: p, pos: start}, nil
		}
	}
	if strings.IndexByte(oneCharPuncts, c) >= 0 {
		l.pos++
		return token{kind: tokPunct, text: string(c), pos: start}, nil
	}

	r, _ := utf8.DecodeRuneInString(l.input[l.pos:])
	return token{}, l.errorf(start, "unexpected character %q", r)
}

func (l *lexer) scanIdent() string {
	start := l.pos
	for l.pos < len(l.input) && isIdentChar(l.input[l.pos]) {
		l.pos++
	}
	return l.input[start:l.pos]
}

func (l *lexer) scanNumber() string {
	start := l.pos
	if strings.HasPrefix(l.input[l.pos:], "0x") || strings.HasPrefix(l.input[l.pos:], "0X") {
		l.pos += 2
		for l.pos < len(l.input) && isHexDigit(l.input[l.pos]) {
			l.pos++
		}
		return l.input[start:l.pos]
	}
	for l.pos < len(l.input) && isDigit(l.input[l.pos]) {
		l.pos++
	}
	// Only take a '.' as a decimal point when a digit follows, so `0.A`
	// stays a key path (0 . A) rather than the number "0." plus an ident.
	if l.pos+1 < len(l.input) && l.input[l.pos] == '.' && isDigit(l.input[l.pos+1]) {
		l.pos++
		for l.pos < len(l.input) && isDigit(l.input[l.pos]) {
			l.pos++
		}
	}
	if l.pos < len(l.input) && (l.input[l.pos] == 'e' || l.input[l.pos] == 'E') {
		mark := l.pos
		l.pos++
		if l.pos < len(l.input) && (l.input[l.pos] == '+' || l.input[l.pos] == '-') {
			l.pos++
		}
		if l.pos < len(l.input) && isDigit(l.input[l.pos]) {
			for l.pos < len(l.input) && isDigit(l.input[l.pos]) {
				l.pos++
			}
		} else {
			l.pos = mark // the e was not an exponent; leave it for the next token
		}
	}
	return l.input[start:l.pos]
}

func isHexDigit(c byte) bool {
	return isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func (l *lexer) scanString() (string, error) {
	start := l.pos
	quote := l.input[l.pos]
	l.pos++
	var b strings.Builder
	for l.pos < len(l.input) {
		c := l.input[l.pos]
		switch c {
		case quote:
			l.pos++
			return b.String(), nil
		case '\\':
			l.pos++
			if l.pos >= len(l.input) {
				return "", l.errorf(start, "unterminated string")
			}
			esc := l.input[l.pos]
			l.pos++
			switch esc {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case 'u', 'U':
				n := 4
				if esc == 'U' {
					n = 8
				}
				if l.pos+n > len(l.input) {
					return "", l.errorf(l.pos-2, "incomplete \\%c escape", esc)
				}
				var v rune
				for i := 0; i < n; i++ {
					c := l.input[l.pos]
					if !isHexDigit(c) {
						return "", l.errorf(l.pos, "invalid hex digit %q in \\%c escape", c, esc)
					}
					v = v<<4 | hexVal(c)
					l.pos++
				}
				b.WriteRune(v)
			default:
				b.WriteByte(esc)
			}
		default:
			b.WriteByte(c)
			l.pos++
		}
	}
	return "", l.errorf(start, "unterminated string")
}

func hexVal(c byte) rune {
	switch {
	case c >= '0' && c <= '9':
		return rune(c - '0')
	case c >= 'a' && c <= 'f':
		return rune(c-'a') + 10
	default:
		return rune(c-'A') + 10
	}
}

// scanKeyPathArg reads the raw key path between the parentheses of
// @status(...), @key(...), or @property(...). DDM item names contain
// dashes (device.operating-system.version), which the normal tokenizer
// would split, so the parser switches to this scanner for the argument.
// It returns the key path and its starting offset.
func (l *lexer) scanKeyPathArg() (string, int) {
	l.skipSpace()
	start := l.pos
	for l.pos < len(l.input) {
		c := l.input[l.pos]
		if isIdentChar(c) || c == '.' || c == '-' {
			l.pos++
			continue
		}
		break
	}
	return l.input[start:l.pos], start
}
