package ai_agent_sessions

import (
	"encoding/json"
	"slices"
	"strings"
	"unicode/utf8"
)

// fitToolInput cuts a tool call's arguments to limit bytes, and to
// maxNamedBlockBytes once JSON-escaped. Arguments that are a JSON object or
// array stay valid JSON, so consumers can still parse a cut call; anything
// else is cut like any other content.
func fitToolInput(s string, limit int) string {
	target := limit
	// Shortening by bytes can still leave escape-heavy arguments over the
	// escaped limit; aim lower in proportion and retry.
	for range 4 {
		out, ok := shortenJSON(s, target)
		if !ok {
			break
		}
		w := jsonLen(out)
		if w <= maxNamedBlockBytes {
			return out
		}
		target = int(int64(target) * maxNamedBlockBytes / int64(w))
	}
	s = truncateUTF8(s, limit)
	if jsonLen(s) > maxNamedBlockBytes {
		s = truncateJSON(s, maxNamedBlockBytes)
	}
	return s
}

// segment is a run of compacted JSON, or the body of a string value that may
// be cut.
type segment struct {
	text  string
	value bool
}

// shortenJSON fits a JSON object or array into limit bytes by cutting its
// longest string values first, keeping the result valid JSON. Keys, numbers
// and structure are kept and whitespace is dropped. ok is false when s isn't
// a JSON object or array, or cutting every string value isn't enough.
func shortenJSON(s string, limit int) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" || (s[0] != '{' && s[0] != '[') || !json.Valid([]byte(s)) {
		return "", false
	}

	var (
		segs   []segment
		fixed  strings.Builder
		lens   []int
		static int
	)
	flush := func() {
		if fixed.Len() > 0 {
			segs = append(segs, segment{text: fixed.String()})
			static += fixed.Len()
			fixed.Reset()
		}
	}
	for i := 0; i < len(s); {
		c := s[i]
		switch c {
		case ' ', '\t', '\n', '\r':
			i++
			continue
		case '"':
		default:
			fixed.WriteByte(c)
			i++
			continue
		}
		end := i + 1
		for s[end] != '"' {
			if s[end] == '\\' {
				end++
			}
			end++
		}
		next := end + 1
		for next < len(s) && strings.IndexByte(" \t\n\r", s[next]) >= 0 {
			next++
		}
		if next < len(s) && s[next] == ':' {
			fixed.WriteString(s[i : end+1])
		} else {
			fixed.WriteByte('"')
			flush()
			segs = append(segs, segment{text: s[i+1 : end], value: true})
			lens = append(lens, end-i-1)
			fixed.WriteByte('"')
		}
		i = end + 1
	}
	flush()

	budget := limit - static
	if budget < 0 {
		return "", false
	}
	// Find the largest per-string length that fits: strings shorter than it
	// are kept whole, the rest share what's left equally.
	slices.Sort(lens)
	maxLen := -1
	for k, l := range lens {
		if l*(len(lens)-k) > budget {
			maxLen = budget / (len(lens) - k)
			break
		}
		budget -= l
	}

	var b strings.Builder
	for _, sg := range segs {
		if sg.value && maxLen >= 0 {
			b.WriteString(cutStringBody(sg.text, maxLen))
		} else {
			b.WriteString(sg.text)
		}
	}
	return b.String(), true
}

// cutStringBody cuts the body of a JSON string literal to at most limit bytes
// without splitting an escape sequence or a rune.
func cutStringBody(body string, limit int) string {
	i := 0
	for i < len(body) {
		n := 1
		switch {
		case body[i] == '\\' && i+1 < len(body) && body[i+1] == 'u':
			n = 6
		case body[i] == '\\':
			n = 2
		case body[i] >= utf8.RuneSelf:
			_, n = utf8.DecodeRuneInString(body[i:])
		}
		if i+n > limit {
			break
		}
		i += n
	}
	return body[:i]
}
