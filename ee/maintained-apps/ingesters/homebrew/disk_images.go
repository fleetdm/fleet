package homebrew

import (
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/patch_policy"
)

// excludeDiskImages adds patch_policy.MacOSNotOnDiskImage to every WHERE clause that reads the apps table.
func excludeDiskImages(query string) (string, error) {
	var out strings.Builder
	i := 0
	for i < len(query) {
		if query[i] == '\'' {
			end := skipQuoted(query, i)
			out.WriteString(query[i:end])
			i = end
			continue
		}
		qualifier, afterTable, ok := matchFromApps(query, i)
		if !ok {
			out.WriteByte(query[i])
			i++
			continue
		}

		end := clauseEnd(query, afterTable)
		where := topLevelKeyword(query, afterTable, end, "WHERE")
		if where < 0 {
			return "", fmt.Errorf("query reads apps without a WHERE clause: %s", query)
		}
		bodyStart := where + len("WHERE")
		body, err := excludeDiskImages(strings.TrimSpace(query[bodyStart:end]))
		if err != nil {
			return "", err
		}
		for _, kw := range []string{"GROUP", "ORDER", "LIMIT", "UNION", "EXCEPT", "INTERSECT"} {
			if topLevelKeyword(body, 0, len(body), kw) >= 0 {
				return "", fmt.Errorf("query reads apps with %s after its WHERE clause: %s", kw, query)
			}
		}
		if topLevelKeyword(body, 0, len(body), "OR") >= 0 {
			body = "(" + body + ")"
		}

		out.WriteString(query[i:bodyStart])
		out.WriteString(" " + body + " AND " + patch_policy.MacOSNotOnDiskImage(qualifier))
		i = end
	}
	return out.String(), nil
}

func matchFromApps(s string, i int) (qualifier string, afterTable int, ok bool) {
	if i > 0 && isIdentChar(s[i-1]) {
		return "", 0, false
	}
	j, ok := matchWord(s, i, "FROM")
	if !ok || j >= len(s) || !isSpace(s[j]) {
		return "", 0, false
	}
	j = skipSpaces(s, j)
	j, ok = matchWord(s, j, "apps")
	if !ok {
		return "", 0, false
	}

	afterTable = j
	alias, k := readIdent(s, skipSpaces(s, j))
	if strings.EqualFold(alias, "AS") {
		alias, k = readIdent(s, skipSpaces(s, k))
	}
	switch strings.ToUpper(alias) {
	case "", "WHERE", "JOIN", "CROSS", "INNER", "LEFT", "NATURAL", "ON", "USING", "GROUP", "ORDER", "LIMIT", "UNION":
		return "apps", afterTable, true
	}
	return alias, k, true
}

func clauseEnd(s string, i int) int {
	depth := 0
	for i < len(s) {
		switch s[i] {
		case '\'':
			i = skipQuoted(s, i)
			continue
		case '(':
			depth++
		case ')':
			if depth == 0 {
				return i
			}
			depth--
		case ';':
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return i
}

func topLevelKeyword(s string, start, end int, kw string) int {
	depth := 0
	for i := start; i < end; {
		switch c := s[i]; {
		case c == '\'':
			i = skipQuoted(s, i)
			continue
		case c == '(':
			depth++
		case c == ')':
			depth--
		case depth == 0 && (i == 0 || !isIdentChar(s[i-1])):
			if j, ok := matchWord(s, i, kw); ok && j <= end {
				return i
			}
		}
		i++
	}
	return -1
}

func matchWord(s string, i int, w string) (int, bool) {
	j := i + len(w)
	if j > len(s) || !strings.EqualFold(s[i:j], w) || (j < len(s) && isIdentChar(s[j])) {
		return 0, false
	}
	return j, true
}

func readIdent(s string, i int) (string, int) {
	j := i
	for j < len(s) && isIdentChar(s[j]) {
		j++
	}
	return s[i:j], j
}

func skipQuoted(s string, i int) int {
	for j := i + 1; j < len(s); j++ {
		if s[j] != '\'' {
			continue
		}
		if j+1 < len(s) && s[j+1] == '\'' {
			j++
			continue
		}
		return j + 1
	}
	return len(s)
}

func skipSpaces(s string, i int) int {
	for i < len(s) && isSpace(s[i]) {
		i++
	}
	return i
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func isIdentChar(c byte) bool {
	return c == '_' || ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || ('0' <= c && c <= '9')
}
