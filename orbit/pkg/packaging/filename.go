package packaging

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// maxFilenameBytes caps generated file names well below Windows' 260-character path limit, which also counts the
// folder: Intune stages MSIs about 120 characters deep, OneDrive folders add about 70, and WiX 3 fails on long paths.
// A UTF-8 byte count is never smaller than the UTF-16 count Windows uses, so this holds on every OS.
const maxFilenameBytes = 100

// Fleet is a fleet as the package file name needs it.
type Fleet struct {
	ID   uint
	Name string
}

// outputFilename returns the default name of a generated package: fleet-<platform>[-<arch>][-<fleet>].<ext>.
func outputFilename(platform, arch, ext string, opt Options) string {
	name := "fleet-" + platform
	if arch != "" {
		name += "-" + arch
	}
	ext = "." + ext
	budget := maxFilenameBytes - len(name) - len("-") - len(ext)
	if fleet := fleetFilenamePart(Fleet{ID: opt.FleetID, Name: opt.FleetName}, opt.OtherFleets, budget); fleet != "" {
		name += "-" + fleet
	}
	return name + ext
}

// fleetFilenamePart returns the fleet's cleaned name, cut to budget bytes. Fleets whose parts match get their ID at the
// end, in rounds until no two match, so their packages don't overwrite each other. Parts that end in different IDs
// never match, and each round is decided from the same parts, so a fleet gets the same part whichever package is built.
func fleetFilenamePart(fleet Fleet, others []Fleet, budget int) string {
	all := append([]Fleet{fleet}, others...)
	parts, keys := make([]string, len(all)), make([]string, len(all))
	for i, f := range all {
		parts[i] = truncateUTF8(fleetNameForFilename(f.Name), budget)
		keys[i] = sameFileKey(parts[i])
	}
	withID := make([]bool, len(all))
	for {
		count := make(map[string]int, len(all))
		for _, key := range keys {
			count[key]++
		}
		var round []int
		for i := range all {
			if !withID[i] && parts[i] != "" && all[i].ID != 0 && count[keys[i]] > 1 {
				round = append(round, i)
			}
		}
		if len(round) == 0 {
			return parts[0]
		}
		for _, i := range round {
			parts[i], withID[i] = nameWithID(all[i], budget), true
			keys[i] = sameFileKey(parts[i])
		}
	}
}

// sameFileKey is the same for parts that name the same file on a case-insensitive file system. macOS folds case fully
// ("ß" and "ss", "ﬁ" and "fi" are one file), so lowercasing isn't enough.
func sameFileKey(part string) string {
	return norm.NFC.String(cases.Fold().String(part))
}

// nameWithID returns the fleet's cleaned name followed by its ID, cut to fit in budget bytes when the ID does.
func nameWithID(f Fleet, budget int) string {
	suffix := strconv.FormatUint(uint64(f.ID), 10)
	if short := truncateUTF8(fleetNameForFilename(f.Name), budget-len(suffix)-len("-")); short != "" {
		return short + "-" + suffix
	}
	return suffix
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
