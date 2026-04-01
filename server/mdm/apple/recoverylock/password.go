package recoverylock

import (
	"crypto/rand"
	"strings"
)

// PasswordCharset excludes confusing characters (0/O, 1/I/l)
const PasswordCharset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

// GeneratePassword generates a password in format: 5ADZ-HTZ8-LJJ4-B2F8-JWH3-YPBT
// (6 groups of 4 alphanumeric characters separated by dashes)
func GeneratePassword() string {
	const (
		groupCount = 6
		groupLen   = 4
	)

	groups := make([]string, groupCount)
	charsetLen := len(PasswordCharset)

	for i := range groupCount {
		randBytes := make([]byte, groupLen)
		_, _ = rand.Read(randBytes) // rand.Read never returns an error; it panics on failure

		group := make([]byte, groupLen)
		for j := range groupLen {
			group[j] = PasswordCharset[int(randBytes[j])%charsetLen]
		}
		groups[i] = string(group)
	}

	return strings.Join(groups, "-")
}
