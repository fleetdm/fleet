package str

import "strings"

func SplitAndTrim(s string, delimiter string, removeEmpty bool) []string {
	parts := strings.Split(s, delimiter)
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if !removeEmpty || part != "" {
			cleaned = append(cleaned, part)
		}
	}
	return cleaned
}
