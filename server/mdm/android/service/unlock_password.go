package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"google.golang.org/api/androidmanagement/v1"
)

const (
	unlockPasswordDefaultLength = 6
	unlockPasswordMaxLength     = 32
	// Android's NUMERIC_COMPLEX and complexity bands reject a sequence longer than 3 (MAX_ALLOWED_SEQUENCE).
	unlockPasswordMaxSequence = 3

	// Look-alike characters (l, o, I, O) are left out because admins read the password out to the end user.
	unlockPasswordDigits  = "0123456789"
	unlockPasswordLower   = "abcdefghijkmnpqrstuvwxyz"
	unlockPasswordUpper   = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	unlockPasswordSymbols = "!@#$%&*+-=?"
)

type unlockPasswordKind int

const (
	unlockPasswordNumeric unlockPasswordKind = iota
	unlockPasswordAlphanumeric
	unlockPasswordComplex
)

// unlockPasswordSpec is the strictest requirement across all applied password policies.
type unlockPasswordSpec struct {
	kind      unlockPasswordKind
	length    int
	letters   int
	lower     int
	upper     int
	nonLetter int
	numeric   int
	symbols   int
}

// generateUnlockPassword returns a password that satisfies every applied password policy. With no
// policies it returns a 6-digit PIN with no repeating or ordered sequence.
func generateUnlockPassword(policies []*androidmanagement.PasswordRequirements) (string, error) {
	spec := mergeUnlockPasswordPolicies(policies)

	var classes []string
	var counts []int
	fill := unlockPasswordDigits
	switch spec.kind {
	case unlockPasswordNumeric:
		classes, counts = []string{unlockPasswordDigits}, []int{spec.length}
	case unlockPasswordAlphanumeric:
		classes, counts = []string{unlockPasswordLower, unlockPasswordDigits}, []int{1, 1}
		fill = unlockPasswordLower + unlockPasswordDigits
	case unlockPasswordComplex:
		// Unset minimums fall back to Android's COMPLEX defaults: one letter, one digit, one symbol.
		letters := max(spec.letters, spec.lower+spec.upper, 1)
		numeric := max(spec.numeric, 1)
		symbols := max(spec.symbols, 1)
		numeric += max(0, spec.nonLetter-numeric-symbols)
		classes = []string{unlockPasswordUpper, unlockPasswordLower, unlockPasswordDigits, unlockPasswordSymbols}
		counts = []int{spec.upper, letters - spec.upper, numeric, symbols}
		fill = unlockPasswordLower + unlockPasswordDigits
	}

	required := 0
	for _, c := range counts {
		required += c
	}
	length := max(spec.length, required)
	if length > unlockPasswordMaxLength {
		return "", &fleet.BadRequestError{Message: fmt.Sprintf(
			"Couldn't lock host. The host's password policy requires a password longer than %d characters.", unlockPasswordMaxLength)}
	}

	// Rejection sampling: a uniformly random candidate rarely contains a 4-character sequence, so this
	// almost always succeeds on the first few tries.
	for range 1000 {
		candidate := make([]byte, 0, length)
		for i, class := range classes {
			for range counts[i] {
				c, err := randomChar(class)
				if err != nil {
					return "", err
				}
				candidate = append(candidate, c)
			}
		}
		for len(candidate) < length {
			c, err := randomChar(fill)
			if err != nil {
				return "", err
			}
			candidate = append(candidate, c)
		}
		if err := shuffleBytes(candidate); err != nil {
			return "", err
		}
		if !hasUnlockPasswordSequence(candidate) {
			return string(candidate), nil
		}
	}
	return "", errors.New("couldn't generate an unlock password without an ordered or repeating sequence")
}

func mergeUnlockPasswordPolicies(policies []*androidmanagement.PasswordRequirements) unlockPasswordSpec {
	spec := unlockPasswordSpec{kind: unlockPasswordNumeric, length: unlockPasswordDefaultLength}
	for _, p := range policies {
		if p == nil {
			continue
		}
		spec.length = max(spec.length, int(p.PasswordMinimumLength))

		switch p.PasswordQuality {
		case "", "PASSWORD_QUALITY_UNSPECIFIED", "BIOMETRIC_WEAK", "SOMETHING", "NUMERIC", "NUMERIC_COMPLEX",
			"COMPLEXITY_LOW", "COMPLEXITY_MEDIUM":
			// A PIN with no sequence satisfies all of these.
		case "COMPLEXITY_HIGH":
			// On Android 12+, a PIN in the high band needs at least 8 digits.
			spec.length = max(spec.length, 8)
		case "ALPHABETIC", "ALPHANUMERIC":
			spec.kind = max(spec.kind, unlockPasswordAlphanumeric)
		default:
			// COMPLEX, or a quality added after this was written. A complex password with a letter, a digit, and a
			// symbol satisfies every other band, so it's the safest fallback.
			spec.kind = unlockPasswordComplex
			if p.PasswordQuality != "COMPLEX" {
				spec.length = max(spec.length, 8)
			}
			// The passwordMinimum* fields are only enforced for COMPLEX.
			spec.letters = max(spec.letters, int(p.PasswordMinimumLetters))
			spec.lower = max(spec.lower, int(p.PasswordMinimumLowerCase))
			spec.upper = max(spec.upper, int(p.PasswordMinimumUpperCase))
			spec.nonLetter = max(spec.nonLetter, int(p.PasswordMinimumNonLetter))
			spec.numeric = max(spec.numeric, int(p.PasswordMinimumNumeric))
			spec.symbols = max(spec.symbols, int(p.PasswordMinimumSymbols))
		}
	}
	return spec
}

// hasUnlockPasswordSequence reports whether s has more than unlockPasswordMaxSequence consecutive characters of
// the same class at a constant step (e.g. 4444, 1234, 8642, abcd). This is stricter than Android, which only allows
// a step of 1 for letters and 0 for other non-digits.
func hasUnlockPasswordSequence(s []byte) bool {
	run := 1
	for i := 1; i < len(s); i++ {
		sameClass := unlockPasswordCharClass(s[i]) == unlockPasswordCharClass(s[i-1])
		switch {
		case !sameClass:
			run = 1
		case run == 1 || int(s[i])-int(s[i-1]) == int(s[i-1])-int(s[i-2]):
			run++
		default:
			run = 2
		}
		if run > unlockPasswordMaxSequence {
			return true
		}
	}
	return false
}

func unlockPasswordCharClass(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return 0
	case c >= 'a' && c <= 'z':
		return 1
	case c >= 'A' && c <= 'Z':
		return 2
	default:
		return 3
	}
}

func randomChar(set string) (byte, error) {
	i, err := randomInt(len(set))
	if err != nil {
		return 0, err
	}
	return set[i], nil
}

func shuffleBytes(b []byte) error {
	for i := len(b) - 1; i > 0; i-- {
		j, err := randomInt(i + 1)
		if err != nil {
			return err
		}
		b[i], b[j] = b[j], b[i]
	}
	return nil
}

func randomInt(n int) (int, error) {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, fmt.Errorf("generating random number: %w", err)
	}
	return int(v.Int64()), nil
}
