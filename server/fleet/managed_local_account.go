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
)

// GenerateManagedLocalAccountPassword returns a cryptographically random password for the managed local admin account,
// formatted in hyphen-separated groups so it can be read aloud and typed at a login prompt without error.
//
// includeLowercase controls whether lowercase letters appear.
//
// The password is guaranteed to contain at least one character from each enabled class, so the category count never
// depends on chance. That guarantee is applied by discarding and redrawing a password that misses a class, rather than
// by seeding one character per class and shuffling: seeding skews the result towards balanced class counts, while
// redrawing stays exactly uniform over the passwords that satisfy the guarantee. Roughly one draw in 40 is discarded.
func GenerateManagedLocalAccountPassword(includeLowercase bool) string {
	classes := []string{managedAccountDigits, managedAccountUppercase}
	if includeLowercase {
		classes = append(classes, managedAccountLowercase)
	}
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
	// the size of the draw range, not of the alphabet: for n=56 it is 4294967264, only 32 short of span.
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
