package packaging

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// maxFilenameBytes keeps generated file names under the 255-byte limit of common file systems, with room for the
// " (1)" browsers add to repeated downloads and for the folder path under Windows' 260-character MAX_PATH.
const maxFilenameBytes = 200

// outputFilename returns the default name of a generated package: fleet-<platform>[-<arch>][-<fleet>].<ext>.
func outputFilename(platform, arch, fleetName, ext string) string {
	name := "fleet-" + platform
	if arch != "" {
		name += "-" + arch
	}
	ext = "." + ext
	if fleet := truncateUTF8(fleetNameForFilename(fleetName), maxFilenameBytes-len(name)-len("-")-len(ext)); fleet != "" {
		name += "-" + fleet
	}
	return name + ext
}

// fleetNameForFilename lowercases a fleet name and keeps only its letters and digits, in any script, turning every
// other run of characters (spaces, punctuation, path separators, emojis) into a single dash. It returns "" when no
// letter or digit is left.
func fleetNameForFilename(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range norm.NFC.String(name) {
		switch {
		// Invisible format characters (zero-width joiners inside Persian or Indic words, soft hyphens, bidi controls)
		// are dropped rather than split the word they sit in.
		case unicode.Is(unicode.Cf, r):
			continue
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
	return b.String()
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
