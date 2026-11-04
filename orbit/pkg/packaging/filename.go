package packaging

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// maxFilenameBytes caps generated file names well below Windows' 260-character path limit, which also counts the
// folder: Intune stages MSIs about 120 characters deep, OneDrive folders add about 70, and WiX 3 fails on long paths.
// A UTF-8 byte count is never smaller than the UTF-16 count Windows uses, so this holds on every OS.
const maxFilenameBytes = 100

// outputFilename returns the default name of a generated package: fleet-<platform>[-<arch>][-<fleet>].<ext>.
func outputFilename(platform, arch, ext string, opt Options) string {
	name := "fleet-" + platform
	if arch != "" {
		name += "-" + arch
	}
	ext = "." + ext
	budget := maxFilenameBytes - len(name) - len("-") - len(ext)
	if fleet := fleetFilenamePart(opt.FleetName, opt.FleetID, opt.OtherFleetNames, budget); fleet != "" {
		name += "-" + fleet
	}
	return name + ext
}

// fleetFilenamePart returns the cleaned fleet name, cut to budget bytes. If another fleet's name gives the same part,
// it ends with the fleet ID instead, so their packages don't overwrite each other. Best effort: a fleet named "foo 3"
// can still match another "foo" with ID 3.
func fleetFilenamePart(name string, id uint, others []string, budget int) string {
	part := truncateUTF8(fleetNameForFilename(name), budget)
	if part == "" || id == 0 {
		return part
	}
	for _, other := range others {
		// Case-insensitive, like macOS and Windows file systems ("ſ" and "s" name the same file there).
		if strings.EqualFold(truncateUTF8(fleetNameForFilename(other), budget), part) {
			suffix := strconv.FormatUint(uint64(id), 10)
			if short := truncateUTF8(fleetNameForFilename(name), budget-len(suffix)-len("-")); short != "" {
				return short + "-" + suffix
			}
			return suffix
		}
	}
	return part
}

// fleetNameForFilename lowercases a fleet name and keeps only its letters and digits, in any script, turning every
// other run of characters (spaces, punctuation, path separators, emojis) into a single dash. It returns "" when no
// letter or digit is left.
func fleetNameForFilename(name string) string {
	// Invisible characters (zero-width joiners inside Persian or Indic words, soft hyphens, bidi controls, Hangul fillers,
	// the combining grapheme joiner) go first, so they neither split a word nor keep NFC from composing around them.
	name = norm.NFC.String(strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Other_Default_Ignorable_Code_Point, r) {
			return -1
		}
		return r
	}, name))
	var b strings.Builder
	dash := false
	for _, r := range name {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
		// Combining marks that NFC can't compose (e.g. Devanagari vowel signs) belong to the letter before them.
		// Variation selectors are marks too, but they only style the emoji before them.
		case unicode.In(r, unicode.Mn, unicode.Mc) && !unicode.Is(unicode.Variation_Selector, r) && b.Len() > 0 && !dash:
		default:
			dash = true
			continue
		}
		if dash && b.Len() > 0 {
			b.WriteByte('-')
		}
		dash = false
		b.WriteRune(unicode.ToLower(r))
	}
	// Lowercasing can leave a letter and a mark that compose.
	return norm.NFC.String(b.String())
}

// truncateUTF8 cuts s to at most n bytes without splitting a character, and trims the dash a cut can leave at the end.
func truncateUTF8(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return strings.TrimRight(s[:n], "-")
}
