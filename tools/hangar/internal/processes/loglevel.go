package processes

import (
	"regexp"
	"strings"
)

// icontains reports whether needle appears in hay, ASCII case-insensitively,
// without allocating a lowercased copy (this fires thousands of times a
// second under fleet serve debug logging).
func icontains(hay, needle string) bool {
	if len(needle) > len(hay) {
		return false
	}
	if needle == "" {
		return true
	}
outer:
	for i := 0; i+len(needle) <= len(hay); i++ {
		for j := 0; j < len(needle); j++ {
			if !eqIgnoreCaseByte(hay[i+j], needle[j]) {
				continue outer
			}
		}
		return true
	}
	return false
}

func eqIgnoreCaseByte(a, b byte) bool {
	if 'A' <= a && a <= 'Z' {
		a += 'a' - 'A'
	}
	if 'A' <= b && b <= 'Z' {
		b += 'a' - 'A'
	}
	return a == b
}

// detectLevel classifies a log line as debug/info/warn/error, or "" (nil)
// when no signal is found. Checks logrus/slog "level=" first, then common
// token forms in the first 64 bytes (where timestamp+level live).
func detectLevel(msg string) string {
	if icontains(msg, "level=error") || icontains(msg, "level=err") {
		return "error"
	}
	if icontains(msg, "level=warn") {
		return "warn"
	}
	if icontains(msg, "level=debug") {
		return "debug"
	}
	if icontains(msg, "level=info") {
		return "info"
	}

	headLen := len(msg)
	if headLen > 64 {
		headLen = 64
	}
	head := msg[:headLen] // byte slice — safe (no UTF-8 boundary panic, unlike Rust's str slice)

	startsWithError := len(head) >= 5 && eqIgnoreCasePrefix(head[:5], "error")
	if icontains(head, " error ") || icontains(head, "] error ") || startsWithError {
		return "error"
	}
	if icontains(head, " warn ") || icontains(head, "] warn ") || icontains(head, " warning") {
		return "warn"
	}
	if icontains(head, " debug ") || icontains(head, "] debug ") {
		return "debug"
	}
	if icontains(head, " info ") || icontains(head, "] info ") {
		return "info"
	}
	return ""
}

func eqIgnoreCasePrefix(s, prefix string) bool {
	if len(s) != len(prefix) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !eqIgnoreCaseByte(s[i], prefix[i]) {
			return false
		}
	}
	return true
}

// secretRe matches bearer tokens and token/password=VALUE pairs so we can
// redact them before persisting logs to ~/Library/Logs. Best-effort, not a
// comprehensive PII filter.
var secretRe = regexp.MustCompile(`(?i)(Bearer\s+[^\s'"\r\n]+)|((?:token|password|authtoken|authorization)=[^\s'"&\r\n]+)`)

// scrubSecrets redacts the value while keeping the key/prefix.
func scrubSecrets(line string) string {
	return secretRe.ReplaceAllStringFunc(line, func(m string) string {
		if cutPrefixCI(m, "Bearer ") {
			return "Bearer [redacted]"
		}
		if i := strings.IndexByte(m, '='); i >= 0 {
			return m[:i] + "=[redacted]"
		}
		return m
	})
}

func cutPrefixCI(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return eqIgnoreCasePrefix(s[:len(prefix)], prefix)
}

// signalName maps a signal number to its short name (the ones we're likely
// to see); anything else is "?".
func signalName(sig int) string {
	switch sig {
	case 1:
		return "SIGHUP"
	case 2:
		return "SIGINT"
	case 3:
		return "SIGQUIT"
	case 6:
		return "SIGABRT"
	case 9:
		return "SIGKILL"
	case 11:
		return "SIGSEGV"
	case 13:
		return "SIGPIPE"
	case 15:
		return "SIGTERM"
	default:
		return "?"
	}
}
