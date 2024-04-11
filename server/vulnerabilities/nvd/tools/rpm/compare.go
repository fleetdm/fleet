// Copyright (c) Facebook, Inc. and its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rpm

import (
	"strings"
	"unicode"
)

// LabelCompare returns 0 if the two packages have the same label, -1 if l1 < l2
// and 1 of l1 > l2.
func LabelCompare(l1, l2 Label) int {
	// 1. Set each epoch value to 0 if it’s null/None.
	if l1.Epoch == "" {
		l1.Epoch = "0"
	}
	if l2.Epoch == "" {
		l2.Epoch = "0"
	}

	// 2. Compare the epoch values using compare_values(). If they’re not equal,
	// return that result, else move on to the next portion (version). The logic
	// within compare_values() is that if one is empty/null and the other is not,
	// the non-empty one is greater, and that ends the comparison. If neither of
	// them is empty/not present, compare them using rpmvercmp() and follow the same
	// logic; if one is “greater” (newer) than the other, that’s the end result of
	// the comparison. Otherwise, move on to the next component (version).
	if c := versionCompare(l1.Epoch, l2.Epoch); c != 0 {
		return c
	}

	// 3. Compare the versions using the same logic.
	if c := versionCompare(l1.Version, l2.Version); c != 0 {
		return c
	}

	// 4. Compare the releases using the same logic.
	if c := versionCompare(l1.Release, l2.Release); c != 0 {
		return c
	}

	// 5. If all of the components are “equal”, the packages are the same.
	return 0
}

func versionCompare(v1, v2 string) int {
	// 1. If the strings are binary equal (a == b), they’re equal, return 0.
	if v1 == v2 {
		return 0
	}

	// 2. Loop over the strings, left-to-right.
	for {
		// 1. Trim anything that’s not [A-Za-z0-9] or tilde (~) from the front of both strings.
		v1 = strings.TrimLeftFunc(v1, isntAlnumOrTilde)
		v2 = strings.TrimLeftFunc(v2, isntAlnumOrTilde)

		v1StartsWithTilde := len(v1) > 0 && v1[0] == '~'
		v2StartsWithTilde := len(v2) > 0 && v2[0] == '~'

		if v1StartsWithTilde && v2StartsWithTilde {
			// 2. If both strings start with a tilde, discard it and move on to the next character.
			v1, v2 = v1[1:], v2[1:]
			continue
		} else if v1StartsWithTilde {
			// 3.a If string a starts with a tilde and string b does not, return -1 (string a is older);
			return -1
		} else if v2StartsWithTilde {
			// 3.b and the inverse if string b starts with a tilde and string a does not.
			return 1
		}

		// neither v1 nor v2 start with tilde
		// they start with a letter or digit, or empty

		if len(v1) == 0 || len(v2) == 0 {
			// 4. End the loop if either string has reached zero length.
			break
		}

		// 5. If the first character of a is a digit, pop the leading chunk of continuous digits from each
		// string (which may be ” for b if only one a starts with digits). If a begins with a letter, do
		// the same for leading letters.
		var isNumeric bool
		var segFunc func(rune) bool
		if unicode.IsDigit(rune(v1[0])) {
			isNumeric = true
			segFunc = unicode.IsDigit
		} else {
			isNumeric = false
			segFunc = unicode.IsLetter
		}

		var v1Seg, v2Seg string
		v1Seg, v1 = takeWhile(v1, segFunc) // seg will always have len > 0
		v2Seg, v2 = takeWhile(v2, segFunc)

		// 6. If the segement from b had 0 length, return 1 if the segment from a was numeric, or -1 if it
		// was alphabetic. The logical result of this is that if a begins with numbers and b does not, a is
		// newer (return 1). If a begins with letters and b does not, then a is older (return -1). If the
		// leading character(s) from a and b were both numbers or both letters, continue on.
		if len(v2Seg) == 0 {
			if isNumeric {
				return 1
			}
			return -1
		}

		// here we know that they both start with either numbers or letters
		if isNumeric {
			// 7. If the leading segments were both numeric, discard any leading zeros and whichever one is
			// longer wins. If a is longer than b (without leading zeroes), return 1, and vice-versa. If
			// they’re of the same length, continue on.
			v1Seg = strings.TrimLeftFunc(v1Seg, isZero)
			v2Seg = strings.TrimLeftFunc(v2Seg, isZero)
			if c := compareLen(v1Seg, v2Seg); c != 0 {
				return c
			}
		}

		// 8. Compare the leading segments with strcmp() (or <=> in Ruby). If that returns a non-zero value,
		// then return that value. Else continue to the next iteration of the loop.
		switch {
		case v1Seg < v2Seg:
			return -1
		case v1Seg > v2Seg:
			return 1
		}
	}

	// 3. If the loop ended (nothing has been returned yet, either both strings are
	// totally the same or they’re the same up to the end of one of them, like with
	// “1.2.3” and “1.2.3b”), then the longest wins - if what’s left of a is longer
	// than what’s left of b, return 1. Vice-versa for if what’s left of b is longer
	// than what’s left of a. And finally, if what’s left of them is the same length,
	// return 0.
	return compareLen(v1, v2)
}

func compareLen(s1, s2 string) int {
	diff := len(s1) - len(s2)
	switch {
	case diff < 0:
		return -1
	case diff > 0:
		return 1
	default:
		return 0
	}
}

func isntAlnumOrTilde(r rune) bool {
	return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '~')
}

func isZero(r rune) bool {
	return r == '0'
}

func takeWhile(s string, f func(rune) bool) (matched, rest string) {
	var i int
	for i < len(s) && f(rune(s[i])) {
		i++
	}
	return s[:i], s[i:]
}
