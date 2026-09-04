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

const utf8BOM = "\ufeff"

// PosixQuote returns value as a single-quoted word. It is only safe inside the
// LC_ALL=C region Preamble builds: some libc decoders accept 0x27 as a multi-byte
// trail byte, letting a value ending on a lead byte consume its closing quote.
func PosixQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

// PowerShellCharArray returns an expression evaluating to value, built from its
// UTF-16 code units. Nothing from value reaches the script as source text, so the
// four Unicode code points PowerShell also treats as single quotes can't break
// out. It avoids method calls so it still evaluates under ConstrainedLanguage.
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
		pos, err := powerShellPreamblePos(contents)
		if err != nil {
			return "", err
		}
		if before := strings.TrimPrefix(contents[:pos], utf8BOM); before != "" && !strings.HasSuffix(before, "\n") {
			preamble = "\r\n" + preamble
		}
		return contents[:pos] + preamble + contents[pos:], nil
	}
	if !strings.HasPrefix(contents, "#!") {
		return preamble + contents, nil
	}
	if i := strings.IndexByte(contents, '\n'); i >= 0 {
		return contents[:i+1] + preamble + contents[i+1:], nil
	}
	return contents + "\n" + preamble, nil
}

// powerShellPreamblePos returns the offset to insert a preamble at: after a byte
// order mark, which breaks the script unless it stays on line 1, and after any
// using statements, which must precede every other statement. It fails on a
// leading param() block, which must come first and so leaves nowhere to insert.
func powerShellPreamblePos(contents string) (int, error) {
	pos := 0
	if strings.HasPrefix(contents, utf8BOM) {
		pos = len(utf8BOM)
	}
	insert := pos

	for {
		trimmed := strings.TrimLeft(contents[pos:], " \t\r\n")
		pos = len(contents) - len(trimmed)
		switch {
		case trimmed == "":
			return insert, nil

		case strings.HasPrefix(trimmed, "<#"):
			// block comments do not nest, and PowerShell rejects an unterminated one
			end := strings.Index(trimmed, "#>")
			if end < 0 {
				return insert, nil
			}
			pos += end + len("#>")

		case strings.HasPrefix(trimmed, "#"):
			nl := strings.IndexByte(trimmed, '\n')
			if nl < 0 {
				return insert, nil
			}
			pos += nl + 1

		case hasKeyword(trimmed, "using"):
			nl := strings.IndexByte(trimmed, '\n')
			if nl < 0 {
				return len(contents), nil
			}
			pos += nl + 1
			insert = pos

		case strings.HasPrefix(trimmed, "["):
			// [CmdletBinding()] may precede param(); [Type]::Member is ordinary code
			after, ok := skipBracketed(trimmed)
			if !ok {
				return insert, nil
			}
			pos = len(contents) - len(after)

		default:
			if hasKeyword(trimmed, "param") {
				return 0, ErrPowerShellLeadingParamBlock
			}
			return insert, nil
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

// hasKeyword reports whether s opens with kw as a complete word.
func hasKeyword(s, kw string) bool {
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
