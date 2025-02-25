package utils

import (
	"strconv"
	"strings"
	"unicode"
)

// Rpmvercmp Compares two evr strings (EPOCH:VERSION-RELEASE) by looking at each part in order:
//   - EPOCHs are compared based on their numeric values, if missing then '0' is assumed,
//     if equal then VERSIONs are compared.
//   - VERSIONS are compared according to librpm's rpmvercmp algo
//     (see http://ftp.rpm.org/api/4.4.2.2/rpmvercmp_8c-source.html), if equal RELEASEs are
//     compared.
//   - RELEASEs are compared using the rpmvercmp algo, if equal then both are equal.
//
// Returns:
//
//	-1 if a < b
//	0 if a == b
//	1 if a > b
func Rpmvercmp(a, b string) int {
	epoch1 := epoch(a)
	epoch2 := epoch(b)

	if epoch1 < epoch2 {
		return -1
	} else if epoch1 > epoch2 {
		return 1
	}

	if r := rpmCmp(version(a), version(b)); r != 0 {
		return r
	}

	return rpmCmp(Release(a), Release(b))
}

type segment struct {
	number  *int
	letters string
	offset  int
}

func (s segment) isEmpty() bool {
	//  We need to ignore tildes because they are meant to be 'modifiers'.
	return s.offset == 0 && s.number == nil && s.letters == "" || s.letters == "~"
}

func (s segment) compare(b segment) int {
	// Tildes 'inside' versions 'lower' the precedent seg.
	if s.letters == "~" {
		if b.letters == "~" {
			return 0
		}
		return -1
	}
	if b.letters == "~" {
		if s.letters == "~" {
			return 0
		}
		return 1
	}

	// Both are numeric, in which case we compare
	// their numeric values
	if s.number != nil && b.number != nil { //nolint:gocritic // ignore ifElseChain
		if *s.number == *b.number {
			return 0
		} else if *s.number < *b.number {
			return -1
		}

		return 1
		// 'a' is a number seg, 'b' is a letter seg,
		// numbers are always greater than letters
	} else if s.number != nil && b.number == nil {
		return 1
		// 'a' is a letter seg, 'b' is a number seg
	} else if s.number == nil && b.number != nil {
		return -1
		// Both segs are letters, then we just
		// compare them
	}

	if s.letters == b.letters {
		return 0
	}
	return strings.Compare(s.letters, b.letters)
}

// Returns the next maximal alphabetic or numeric segment,
// with separators (non-alphanumeric characters) ignored.
func nextSeg(ver string) segment {
	var end int

	var wasNum bool
	var wasLetter bool

	var foundTilde bool
	var foundNonAlphaNum bool

	for i, c := range ver {
		// Non-alpha num chars are either separators or a tilde
		foundNonAlphaNum = !unicode.IsNumber(c) && !unicode.IsLetter(c)
		foundTilde = c == '~'

		// Check whether we arrived at a different segment.
		if (unicode.IsNumber(c) && wasLetter) ||
			(unicode.IsLetter(c) && wasNum) ||
			foundNonAlphaNum {
			end = i
			break
		}

		wasNum = unicode.IsNumber(c)
		wasLetter = unicode.IsLetter(c)
	}

	if end == 0 {
		// Edge case: 'ver' starts
		// with non-alphanumeric char
		// like a separator or tilde.
		if foundNonAlphaNum {
			end = 1
		} else {
			end = len(ver)
		}
	}

	r := segment{
		offset: end,
	}

	// Skip over non-alphanumeric chars
	// (except tildes)
	if foundNonAlphaNum && !foundTilde {
		r.offset++
	}

	if wasNum {
		val, err := strconv.Atoi(ver[:end])
		if err == nil {
			r.number = &val
		}
	} else {
		r.letters = ver[:end]
	}

	return r
}

// How to compare two version strings according to the rpmvercmp algorithm:
// Each label is separated into a list of maximal alphabetic or numeric segments,
// with separators (non-alphanumeric characters) ignored. So, '2.0.1' becomes ('2', '0', '1'),
// while ('2xFg33.+f.5') becomes ('2', 'xFg', '33', 'f', '5').
// All numbers are converted to their numeric value. So '10' becomes 10, '000230'
// becomes 230, and '00000' becomes 0.
// The elements in the list are compared one by one using the following algorithm:
// - If two elements are decided to be different, the label with the newer element wins
// as the newer label.
// - If the elements are decided to be equal, the next elements are
// compared until we either reach different elements or one of the lists runs out.
// - In case one of the lists run out, the other label wins as the newer label.
// So, for example, (1, 2) is newer than (1, 1), and (1, 2, 0) is newer than (1, 2).
//
// The algorithm for comparing list elements is as follows:
// - If one of the elements is a number, while the other is alphabetic, the numeric
// elements is considered newer. So 10 is newer than 'abc', and 0 is newer than 'Z'.
// - If both the elements are numbers, the larger number is considered newer.
// So 5 is newer than 4 and 10 is newer than 2. If the numbers are equal, the elements are decided equal.
// -If both the elements are alphabetic, they are compared lexicographically, with the greater string
// resulting in a newer element. So 'b' is newer than 'a', 'add' is newer than 'ZULU'
// (because lowercase characters win in str comparisons), and 'aba' is newer than 'ab'.
// - If the strings are identical, the elements are decided equal.
func rpmCmp(a, b string) int {
	var offsetA int
	var offsetB int

	if a == "" && b != "" {
		return -1
	}
	if a != "" && b == "" {
		return 1
	}

	for {
		if a[offsetA:] == b[offsetB:] {
			return 0
		}

		segA := nextSeg(a[offsetA:])
		segB := nextSeg(b[offsetB:])

		offsetA += segA.offset
		if offsetA > len(a) {
			break
		}
		offsetB += segB.offset
		if offsetB > len(b) {
			break
		}

		// Check whether 'a' or 'b' ran out of segments.
		if segA.isEmpty() && !segB.isEmpty() {
			return -1
		}
		if !segA.isEmpty() && segB.isEmpty() {
			return 1
		}

		r := segA.compare(segB)
		if r == 0 {
			continue
		}
		return r
	}

	return -1
}

func Release(v string) string {
	var s int
	e := len(v)
	var seen bool

	for i, c := range v {
		if c == '-' && !seen && i > 0 {
			s = i + 1
			seen = true
		} else if c == ' ' && seen {
			e = i
			break
		}
	}

	if !seen {
		return ""
	}

	return v[s:e]
}

func version(v string) string {
	var s int
	e := len(v)
	var seenEpoch bool

	for i, c := range v {
		if c == ' ' && !seenEpoch {
			s = i + 1
		}

		if c == ':' && !seenEpoch {
			seenEpoch = true
			s = i + 1
		}

		if c == '-' && i != 0 {
			e = i
			break
		}
	}

	return v[s:e]
}

func epoch(v string) int {
	var s int
	var e int

	for i, c := range v {
		if c == ' ' {
			s++
		}
		if c == ':' {
			e = i
			break
		}
	}

	if e == 0 {
		return 0
	}

	r, err := strconv.Atoi(v[s:e])
	if err != nil {
		return 0
	}
	return r
}
