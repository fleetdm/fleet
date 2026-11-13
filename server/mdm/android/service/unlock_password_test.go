package service

import (
	"strings"
	"testing"
	"unicode"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

type unlockPasswordCounts struct {
	digits, lower, upper, symbols int
}

func countUnlockPasswordChars(t *testing.T, pw string) unlockPasswordCounts {
	t.Helper()
	var c unlockPasswordCounts
	for _, r := range pw {
		switch {
		case unicode.IsDigit(r):
			c.digits++
		case unicode.IsLower(r):
			c.lower++
		case unicode.IsUpper(r):
			c.upper++
		case strings.ContainsRune(unlockPasswordSymbols, r):
			c.symbols++
		default:
			t.Fatalf("unexpected character %q in %q", r, pw)
		}
	}
	return c
}

// longestDigitSequence returns the longest run of consecutive digits at a constant step, computed independently of
// hasUnlockPasswordSequence.
func longestDigitSequence(pw string) int {
	longest := 0
	for start := 0; start < len(pw); start++ {
		for end := start + 1; end <= len(pw); end++ {
			window := pw[start:end]
			if !isConstantStepDigits(window) {
				break
			}
			longest = max(longest, len(window))
		}
	}
	return longest
}

func isConstantStepDigits(s string) bool {
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
		if i >= 2 && int(s[i])-int(s[i-1]) != int(s[i-1])-int(s[i-2]) {
			return false
		}
	}
	return true
}

func TestGenerateUnlockPassword(t *testing.T) {
	cases := []struct {
		name     string
		policies []*androidmanagement.PasswordRequirements
		// minLength is the strictest length requirement across policies.
		minLength int
		// digitsOnly is true when the output must be a PIN.
		digitsOnly bool
		want       unlockPasswordCounts
		minLetters int
	}{
		{
			name:       "no policies",
			policies:   nil,
			minLength:  6,
			digitsOnly: true,
		},
		{
			name:       "nil policy is ignored",
			policies:   []*androidmanagement.PasswordRequirements{nil},
			minLength:  6,
			digitsOnly: true,
		},
		{
			name: "NUMERIC_COMPLEX length 6",
			policies: []*androidmanagement.PasswordRequirements{
				{PasswordQuality: "NUMERIC_COMPLEX", PasswordMinimumLength: 6},
			},
			minLength:  6,
			digitsOnly: true,
		},
		{
			name: "NUMERIC length 8",
			policies: []*androidmanagement.PasswordRequirements{
				{PasswordQuality: "NUMERIC", PasswordMinimumLength: 8},
			},
			minLength:  8,
			digitsOnly: true,
		},
		{
			name: "COMPLEXITY_LOW and COMPLEXITY_MEDIUM keep the default PIN",
			policies: []*androidmanagement.PasswordRequirements{
				{PasswordQuality: "COMPLEXITY_LOW"},
				{PasswordQuality: "COMPLEXITY_MEDIUM"},
			},
			minLength:  6,
			digitsOnly: true,
		},
		{
			name: "COMPLEXITY_HIGH",
			policies: []*androidmanagement.PasswordRequirements{
				{PasswordQuality: "COMPLEXITY_HIGH"},
			},
			minLength:  8,
			digitsOnly: true,
		},
		{
			name: "minimum length is ignored where Android doesn't enforce it",
			policies: []*androidmanagement.PasswordRequirements{
				{PasswordQuality: "SOMETHING", PasswordMinimumLength: 10},
				{PasswordQuality: "COMPLEXITY_MEDIUM", PasswordMinimumLength: 12},
				{PasswordQuality: "BIOMETRIC_WEAK", PasswordMinimumLength: 14},
			},
			minLength:  6,
			digitsOnly: true,
		},
		{
			name: "COMPLEXITY_HIGH ignores a minimum length over 32",
			policies: []*androidmanagement.PasswordRequirements{
				{PasswordQuality: "COMPLEXITY_HIGH", PasswordMinimumLength: 40},
			},
			minLength:  8,
			digitsOnly: true,
		},
		{
			name: "ALPHABETIC",
			policies: []*androidmanagement.PasswordRequirements{
				{PasswordQuality: "ALPHABETIC"},
			},
			minLength: 6,
			want:      unlockPasswordCounts{lower: 1},
		},
		{
			name: "ALPHANUMERIC length 10",
			policies: []*androidmanagement.PasswordRequirements{
				{PasswordQuality: "ALPHANUMERIC", PasswordMinimumLength: 10},
			},
			minLength: 10,
			want:      unlockPasswordCounts{digits: 1, lower: 1},
		},
		{
			name: "COMPLEX with no minimums uses Android's defaults",
			policies: []*androidmanagement.PasswordRequirements{
				{PasswordQuality: "COMPLEX"},
			},
			minLength:  6,
			want:       unlockPasswordCounts{digits: 1, symbols: 1},
			minLetters: 1,
		},
		{
			name: "COMPLEX with every minimum set",
			policies: []*androidmanagement.PasswordRequirements{
				{
					PasswordQuality:          "COMPLEX",
					PasswordMinimumLength:    12,
					PasswordMinimumLetters:   5,
					PasswordMinimumLowerCase: 2,
					PasswordMinimumUpperCase: 2,
					PasswordMinimumNonLetter: 6,
					PasswordMinimumNumeric:   2,
					PasswordMinimumSymbols:   3,
				},
			},
			minLength:  12,
			want:       unlockPasswordCounts{digits: 2, lower: 2, upper: 2, symbols: 3},
			minLetters: 5,
		},
		{
			name: "COMPLEX minimums add up past the minimum length",
			policies: []*androidmanagement.PasswordRequirements{
				{PasswordQuality: "COMPLEX", PasswordMinimumLength: 4, PasswordMinimumUpperCase: 4, PasswordMinimumSymbols: 4},
			},
			minLength:  9,
			want:       unlockPasswordCounts{digits: 1, upper: 4, symbols: 4},
			minLetters: 4,
		},
		{
			name: "two policies with different scopes use the strictest of each",
			policies: []*androidmanagement.PasswordRequirements{
				{PasswordQuality: "NUMERIC_COMPLEX", PasswordMinimumLength: 9, PasswordScope: "SCOPE_DEVICE"},
				{PasswordQuality: "ALPHANUMERIC", PasswordMinimumLength: 4, PasswordScope: "SCOPE_PROFILE"},
			},
			minLength: 9,
			want:      unlockPasswordCounts{digits: 1, lower: 1},
		},
		{
			name: "minimums on a non-COMPLEX policy aren't enforced by Android and are ignored",
			policies: []*androidmanagement.PasswordRequirements{
				{PasswordQuality: "NUMERIC", PasswordMinimumSymbols: 3, PasswordMinimumUpperCase: 3},
			},
			minLength:  6,
			digitsOnly: true,
		},
		{
			name: "unknown quality falls back to a complex password",
			policies: []*androidmanagement.PasswordRequirements{
				{PasswordQuality: "SOME_FUTURE_QUALITY"}, //nolint:gosec // G101: test data, not a credential
			},
			minLength:  8,
			want:       unlockPasswordCounts{digits: 1, symbols: 1},
			minLetters: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for range 200 {
				pw, err := generateUnlockPassword(tc.policies)
				require.NoError(t, err)
				require.GreaterOrEqual(t, len(pw), tc.minLength, pw)
				require.LessOrEqual(t, len(pw), unlockPasswordMaxLength, pw)
				require.False(t, hasUnlockPasswordSequence([]byte(pw)), pw)
				require.LessOrEqual(t, longestDigitSequence(pw), 3, pw)

				got := countUnlockPasswordChars(t, pw)
				if tc.digitsOnly {
					require.Equal(t, len(pw), got.digits, "expected a PIN, got %q", pw)
					require.Len(t, pw, tc.minLength, "a PIN should be exactly the required length")
					continue
				}
				assert.GreaterOrEqual(t, got.digits, tc.want.digits, pw)
				assert.GreaterOrEqual(t, got.lower, tc.want.lower, pw)
				assert.GreaterOrEqual(t, got.upper, tc.want.upper, pw)
				assert.GreaterOrEqual(t, got.symbols, tc.want.symbols, pw)
				assert.GreaterOrEqual(t, got.lower+got.upper, max(tc.minLetters, tc.want.lower, 1), pw)
			}
		})
	}
}

func TestGenerateUnlockPasswordNoDigitSequences(t *testing.T) {
	seen := make(map[string]struct{})
	for range 10_000 {
		pw, err := generateUnlockPassword(nil)
		require.NoError(t, err)
		require.Len(t, pw, 6)
		require.LessOrEqual(t, longestDigitSequence(pw), 3, pw)
		seen[pw] = struct{}{}
	}
	// 10,000 draws from ~10^6 PINs should almost never collide; a low count means the randomness is broken.
	require.Greater(t, len(seen), 9_800)

	// Digit runs inside an alphanumeric password are checked too.
	policies := []*androidmanagement.PasswordRequirements{{PasswordQuality: "ALPHANUMERIC", PasswordMinimumLength: 16}}
	for range 10_000 {
		pw, err := generateUnlockPassword(policies)
		require.NoError(t, err)
		require.LessOrEqual(t, longestDigitSequence(pw), 3, pw)
	}
}

func TestGenerateUnlockPasswordTooLong(t *testing.T) {
	cases := []struct {
		name     string
		policies []*androidmanagement.PasswordRequirements
	}{
		{"minimum length", []*androidmanagement.PasswordRequirements{{PasswordQuality: "NUMERIC", PasswordMinimumLength: 33}}},
		{"complex minimums", []*androidmanagement.PasswordRequirements{
			{PasswordQuality: "COMPLEX", PasswordMinimumUpperCase: 20, PasswordMinimumSymbols: 20},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := generateUnlockPassword(tc.policies)
			var badRequest *fleet.BadRequestError
			require.ErrorAs(t, err, &badRequest)
		})
	}

	pw, err := generateUnlockPassword([]*androidmanagement.PasswordRequirements{{PasswordQuality: "NUMERIC", PasswordMinimumLength: 32}})
	require.NoError(t, err)
	require.Len(t, pw, 32)
}

func TestHasUnlockPasswordSequence(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"4444", true},
		{"1234", true},
		{"4321", true},
		{"2468", true},
		{"9630", true},
		{"x1357", true},
		{"abcd", true},
		{"AAAA", true},
		{"!!!!", true},
		{"123", false},
		{"1235", false},
		{"12a34", false},
		{"112233", false},
		{"a1b2c3d4", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, hasUnlockPasswordSequence([]byte(tc.in)))
		})
	}
}
