// Package password provides password generation for recovery key passwords.
package password

import (
	"crypto/rand"
	"fmt"
	"strings"
)

// charset excludes confusing characters (0/O, 1/I/l)
const charset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

// Generate generates a password in format: 5ADZ-HTZ8-LJJ4-B2F8-JWH3-YPBT
// (6 groups of 4 alphanumeric characters separated by dashes)
func Generate() (string, error) {
	const (
		groupCount = 6
		groupLen   = 4
	)

	groups := make([]string, groupCount)
	charsetLen := len(charset)

	for i := range groupCount {
		randBytes := make([]byte, groupLen)
		if _, err := rand.Read(randBytes); err != nil {
			return "", fmt.Errorf("generating random bytes: %w", err)
		}

		group := make([]byte, groupLen)
		for j := range groupLen {
			group[j] = charset[int(randBytes[j])%charsetLen]
		}
		groups[i] = string(group)
	}

	return strings.Join(groups, "-"), nil
}
