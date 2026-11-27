package utils

import (
	"strconv"
	"strings"
)

// Dpkgvercmp compares two Debian package versions ([epoch:]upstream[-revision]) the way
// dpkg --compare-versions does. Returns -1 if a < b, 0 if a == b and 1 if a > b.
//
// A prefix before the first colon that is not a non-negative integer is not treated as an
// epoch, so malformed versions still compare consistently instead of failing.
func Dpkgvercmp(a, b string) int {
	epochA, upstreamA, revisionA := splitDebVersion(a)
	epochB, upstreamB, revisionB := splitDebVersion(b)
	if epochA != epochB {
		if epochA < epochB {
			return -1
		}
		return 1
	}
	if r := debVerRevCmp(upstreamA, upstreamB); r != 0 {
		return r
	}
	return debVerRevCmp(revisionA, revisionB)
}

func splitDebVersion(v string) (epoch uint64, upstream, revision string) {
	if before, after, ok := strings.Cut(v, ":"); ok {
		if e, err := strconv.ParseUint(before, 10, 64); err == nil {
			epoch, v = e, after
		}
	}
	if i := strings.LastIndexByte(v, '-'); i >= 0 {
		return epoch, v[:i], v[i+1:]
	}
	return epoch, v, ""
}

// debVerRevCmp is dpkg's verrevcmp: alternating non-digit and digit runs, where in the non-digit
// runs '~' sorts before everything (even the end of the string) and letters sort before other
// characters, and digit runs compare numerically.
func debVerRevCmp(a, b string) int {
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		for (i < len(a) && !isDigitAt(a, i)) || (j < len(b) && !isDigitAt(b, j)) {
			ac, bc := debCharOrder(a, i), debCharOrder(b, j)
			if ac != bc {
				return compareInts(ac, bc)
			}
			i++
			j++
		}
		for i < len(a) && a[i] == '0' {
			i++
		}
		for j < len(b) && b[j] == '0' {
			j++
		}
		firstDiff := 0
		for isDigitAt(a, i) && isDigitAt(b, j) {
			if firstDiff == 0 {
				firstDiff = int(a[i]) - int(b[j])
			}
			i++
			j++
		}
		if isDigitAt(a, i) {
			return 1
		}
		if isDigitAt(b, j) {
			return -1
		}
		if firstDiff != 0 {
			return compareInts(firstDiff, 0)
		}
	}
	return 0
}

func debCharOrder(s string, i int) int {
	if i >= len(s) {
		return 0
	}
	c := s[i]
	switch {
	case c >= '0' && c <= '9':
		return 0
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		return int(c)
	case c == '~':
		return -1
	default:
		return int(c) + 256
	}
}

func isDigitAt(s string, i int) bool {
	return i < len(s) && s[i] >= '0' && s[i] <= '9'
}

func compareInts(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}
