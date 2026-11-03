package sentinelone

import (
	"bufio"
	"strings"
)

// parseStatus flattens the indentation-structured output of `sentinelctl
// status` into a map keyed by the underscore-joined section path, e.g.
// "Daemons > Services > Agent Helper" becomes
// "daemons_services_agent_helper".
//
// A line without a colon is a section header at its own indentation level,
// except on Windows where sentinelctl also emits predicate lines such as
// "SentinelAgent is loaded" (see parsePredicateLine). The first value seen for
// a path wins.
func parseStatus(out string) map[string]string {
	type frame struct {
		indent int
		name   string
	}
	var stack []frame

	result := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(out))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	pathFor := func(key string) string {
		parts := make([]string, 0, len(stack)+1)
		for _, f := range stack {
			parts = append(parts, f.name)
		}
		parts = append(parts, key)
		return strings.Join(parts, "_")
	}

	for scanner.Scan() {
		raw := strings.TrimRight(scanner.Text(), "\r\n")
		if strings.TrimSpace(raw) == "" {
			continue
		}
		indent := leadingIndent(raw)
		trimmed := strings.TrimLeft(raw, " \t")

		// Leaving this indentation level closes every section at or below it.
		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}

		idx := strings.Index(trimmed, ":")
		if idx <= 0 {
			if key, val, ok := parsePredicateLine(trimmed); ok {
				if path := pathFor(key); result[path] == "" {
					result[path] = val
				}
				continue
			}
			if name := normalizeKey(trimmed); name != "" {
				stack = append(stack, frame{indent: indent, name: name})
			}
			continue
		}

		key := normalizeKey(trimmed[:idx])
		if key == "" {
			continue
		}
		// A key with no value opens a section, e.g. "Services:".
		val := strings.TrimSpace(trimmed[idx+1:])
		if val == "" {
			stack = append(stack, frame{indent: indent, name: key})
			continue
		}

		if path := pathFor(key); result[path] == "" {
			result[path] = val
		}
	}
	return result
}

// parsePredicateLine parses the "<subject> is <state>" lines that Windows
// sentinelctl emits, e.g. "SentinelAgent is running as PPL" becomes
// ("sentinelagent_is_running", "running as PPL"). The "_is_loaded" /
// "_is_running" suffix keeps the two states of one subject in separate keys.
func parsePredicateLine(line string) (key, value string, ok bool) {
	idx := strings.Index(strings.ToLower(line), " is ")
	if idx <= 0 {
		return "", "", false
	}
	subject := normalizeKey(line[:idx])
	value = strings.TrimSpace(line[idx+len(" is "):])
	if subject == "" || value == "" {
		return "", "", false
	}

	// A negated state ("is not loaded") keys off the same suffix as its
	// positive form, so an unhealthy host populates the same column.
	key = subject
	state := strings.TrimPrefix(normalizeKey(value), "not_")
	switch {
	case strings.HasPrefix(state, "loaded"):
		key = subject + "_is_loaded"
	case strings.HasPrefix(state, "running"):
		key = subject + "_is_running"
	}
	return key, value, true
}

// leadingIndent counts the leading spaces and tabs on a line.
func leadingIndent(s string) int {
	n := 0
	for _, r := range s {
		if r != ' ' && r != '\t' {
			break
		}
		n++
	}
	return n
}

// normalizeKey lower-snake-cases a sentinelctl label: "Console URL" becomes
// "console_url", "agent-helper" becomes "agent_helper".
func normalizeKey(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSep := false
	for _, r := range strings.TrimSpace(s) {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
			prevSep = false
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevSep = false
		default:
			if !prevSep && b.Len() > 0 {
				b.WriteRune('_')
				prevSep = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}
