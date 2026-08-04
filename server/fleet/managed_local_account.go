package fleet

import (
	"crypto/rand"
	"encoding/binary"
	"strings"
)

// ManagedLocalAccountUsername is the short name of the local admin account Fleet provisions when the managed local
// account feature is enabled. macOS creates it via the AccountConfiguration MDM command, Windows fleetd creates it directly.
const ManagedLocalAccountUsername = "_fleetadmin"

const (
	// managedAccountPasswordGroupCount is the number of character groups in a managed account password.
	managedAccountPasswordGroupCount = 6
	// managedAccountPasswordGroupLen is the number of characters per group.
	managedAccountPasswordGroupLen = 4
	// managedAccountPasswordSeparator joins the groups. Grouping matters because this password is read
	// off a screen and typed at a login prompt, not pasted.
	managedAccountPasswordSeparator = "-"
)

// Character classes for the managed account password. These deliberately omit characters that are
// easily confused when transcribed by hand: 0/O/o and 1/I/l.
const (
	managedAccountDigits    = "23456789"
	managedAccountUppercase = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	managedAccountLowercase = "abcdefghijkmnpqrstuvwxyz"
	// managedAccountSpecial stays deliberately narrow. This password is read off a screen and typed at a sign-in prompt
	// on a machine whose keyboard layout Fleet does not control, so it omits anything that is a dead key on a common
	// non-US layout (^ ~ ` ' "), the characters that are confusable with one another (. , : ; and the bracket families),
	// the underscore, which disappears against an underline, the backslash and pipe, and the hyphen, which already
	// separates the groups. What remains is reachable on US, UK, German, and French layouts, some of it through AltGr
	// but none of it as a dead key.
	managedAccountSpecial = "!#$%*+=?@"
)

// GenerateManagedLocalAccountPasswordForMacOS returns a password for the macOS managed local admin account. It draws
// from uppercase letters and digits only, matching the recovery lock password format admins already read off a screen.
func GenerateManagedLocalAccountPasswordForMacOS() string {
	return generateManagedLocalAccountPassword(managedAccountDigits, managedAccountUppercase)
}

// GenerateManagedLocalAccountPasswordForWindows returns a password for the Windows managed local admin account. On top
// of the macOS classes it draws from lowercase and special characters, so the password satisfies a policy that demands
// all four Windows character categories. fleetd cannot read that policy before it calls NetUserAdd, and Windows reports
// every rejection as the same opaque code after the fact, so the password covers the strictest case up front.
func GenerateManagedLocalAccountPasswordForWindows() string {
	return generateManagedLocalAccountPassword(
		managedAccountDigits, managedAccountUppercase, managedAccountLowercase, managedAccountSpecial)
}

// generateManagedLocalAccountPassword returns a cryptographically random password drawn from the given character
// classes, formatted in hyphen-separated groups so it can be read aloud and typed at a login prompt without error.
//
// The password is guaranteed to contain at least one character from each class, so the category count never depends on
// chance. That guarantee is applied by discarding and redrawing a password that misses a class, rather than by seeding
// one character per class and shuffling: seeding skews the result towards balanced class counts, while redrawing stays
// exactly uniform over the passwords that satisfy the guarantee. About one Windows draw in fourteen is discarded, and
// one macOS draw in a thousand.
func generateManagedLocalAccountPassword(classes ...string) string {
	all := strings.Join(classes, "")
	total := managedAccountPasswordGroupCount * managedAccountPasswordGroupLen

	chars := make([]byte, total)
	for {
		for i := range chars {
			chars[i] = all[randomIndex(len(all))]
		}
		if containsEachClass(chars, classes) {
			break
		}
	}

	groups := make([]string, 0, managedAccountPasswordGroupCount)
	for i := 0; i < total; i += managedAccountPasswordGroupLen {
		groups = append(groups, string(chars[i:i+managedAccountPasswordGroupLen]))
	}
	return strings.Join(groups, managedAccountPasswordSeparator)
}

func containsEachClass(chars []byte, classes []string) bool {
	for _, class := range classes {
		if !strings.ContainsAny(string(chars), class) {
			return false
		}
	}
	return true
}

// randomIndex returns a uniformly random value in [0, n), for any n that fits in an int32.
//
// It rejects draws in the final, short bucket rather than taking a plain modulo, which would bias
// the result towards early characters for any alphabet size that does not divide the draw range.
func randomIndex(n int) int {
	const span = int64(1) << 32
	// limit rounds span down to the largest exact multiple of n, discarding that remainder. Note it is
	// the size of the draw range, not of the alphabet: for n=65 it is 4294967235, only 61 short of span.
	limit := span - span%int64(n)
	var b [4]byte
	for {
		// crypto/rand.Read never returns an error; it crashes the program if the system entropy source
		// fails, so there is no failure mode for a caller to handle.
		_, _ = rand.Read(b[:])
		// Read the four bytes as one number uniform over [0, span). The redraw (v < limit) is unlikely since span and
		// limit are very close together.
		if v := int64(binary.BigEndian.Uint32(b[:])); v < limit {
			return int(v % int64(n))
		}
	}
}
