package variables

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
)

// Dialect is the interpreter a preamble is written for.
type Dialect int

const (
	DialectPOSIX Dialect = iota
	DialectPowerShell
)

// ErrPowerShellLeadingParamBlock reports that the script opens with a param()
// block, which PowerShell requires to be the first statement.
var ErrPowerShellLeadingParamBlock = errors.New("PowerShell script starts with a param() block")

// PosixQuote returns value as a single-quoted word. It is only safe inside the
// LC_ALL=C region Preamble builds: some libc decoders accept 0x27 as a
// multi-byte trail byte, letting a value ending on a lead byte consume its own
// closing quote.
func PosixQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

// PowerShellCharArray returns an expression evaluating to value, built from its
// UTF-16 code units. Nothing from value reaches the script as source text, so no
// quoting rule applies to it; PowerShell treats four Unicode code points besides
// ' as single-quote delimiters. It avoids method calls so it still evaluates
// under ConstrainedLanguage mode.
func PowerShellCharArray(value string) string {
	if value == "" {
		return "''"
	}
	units := utf16.Encode([]rune(value))
	var b strings.Builder
	b.WriteString("([char[]](")
	for i, u := range units {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatUint(uint64(u), 10))
	}
	b.WriteString(") -join '')")
	return b.String()
}

// Preamble builds the assignments defining vars, keyed by name without the
// FLEET_VAR_ prefix. Output is ordered by name.
func Preamble(vars map[string]string, dialect Dialect) string {
	if len(vars) == 0 {
		return ""
	}
	names := make([]string, 0, len(vars))
	for name := range vars {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	if dialect == DialectPowerShell {
		for _, name := range names {
			b.WriteString("$FLEET_VAR_")
			b.WriteString(name)
			b.WriteString(" = ")
			b.WriteString(PowerShellCharArray(vars[name]))
			b.WriteString("\r\n")
		}
		return b.String()
	}

	// see PosixQuote: the assignments must parse in a single-byte locale
	b.WriteString("__fleet_lc=${LC_ALL-}; __fleet_lg=${LANG-}; LC_ALL=C; LANG=C\n")
	b.WriteString("export")
	for _, name := range names {
		b.WriteString(" FLEET_VAR_")
		b.WriteString(name)
		b.WriteByte('=')
		b.WriteString(PosixQuote(vars[name]))
	}
	b.WriteString("\nLC_ALL=${__fleet_lc}; LANG=${__fleet_lg}; unset __fleet_lc __fleet_lg\n")
	return b.String()
}

// InsertPreamble places preamble ahead of the body, keeping any shebang on line
// 1 so the interpreter is still chosen the same way.
func InsertPreamble(contents, preamble string, dialect Dialect) (string, error) {
	if preamble == "" {
		return contents, nil
	}
	if dialect == DialectPowerShell {
		if hasLeadingParamBlock(contents) {
			return "", ErrPowerShellLeadingParamBlock
		}
		return preamble + contents, nil
	}
	if !strings.HasPrefix(contents, "#!") {
		return preamble + contents, nil
	}
	if i := strings.IndexByte(contents, '\n'); i >= 0 {
		return contents[:i+1] + preamble + contents[i+1:], nil
	}
	return contents + "\n" + preamble, nil
}

// hasLeadingParamBlock reports whether the first statement is a param() block.
// Only comments, #Requires and attributes may precede one. A block comment
// counts as a param block: finding its end needs a real tokenizer.
func hasLeadingParamBlock(contents string) bool {
	rest := contents
	for {
		rest = strings.TrimLeft(rest, " \t\r\n")
		switch {
		case rest == "":
			return false
		case strings.HasPrefix(rest, "<#"):
			return true
		case strings.HasPrefix(rest, "#"):
			i := strings.IndexByte(rest, '\n')
			if i < 0 {
				return false
			}
			rest = rest[i+1:]
		case strings.HasPrefix(rest, "["):
			// [CmdletBinding()] may precede param(); [Type]::Member is ordinary code
			after, ok := skipBracketed(rest)
			if !ok {
				return true
			}
			rest = after
		default:
			return isParamKeyword(rest)
		}
	}
}

func skipBracketed(s string) (string, bool) {
	depth := 0
	for i, c := range s {
		switch c {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return s[i+1:], true
			}
		}
	}
	return "", false
}

func isParamKeyword(s string) bool {
	const kw = "param"
	if len(s) < len(kw) || !strings.EqualFold(s[:len(kw)], kw) {
		return false
	}
	after := s[len(kw):]
	if after == "" {
		return true
	}
	c := after[0]
	return !(c == '_' || c == '-' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z'))
}
