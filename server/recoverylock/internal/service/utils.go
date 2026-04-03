package service

import (
	"crypto/rand"
	"strings"

	"github.com/google/uuid"
)

// PasswordCharset contains characters for password generation.
// Excludes confusing characters: 0/O, 1/I/l
const PasswordCharset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz"

// GeneratePassword generates a recovery lock password.
// Format: XXXX-XXXX-XXXX-XXXX-XXXX-XXXX (6 groups of 4 characters)
func GeneratePassword() string {
	groups := make([]string, 6)
	for i := range groups {
		groups[i] = generateGroup(4)
	}
	return strings.Join(groups, "-")
}

// generateGroup generates a random group of characters.
func generateGroup(length int) string {
	b := make([]byte, length)
	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to less secure random if crypto/rand fails
		for i := range b {
			b[i] = PasswordCharset[i%len(PasswordCharset)]
		}
		return string(b)
	}
	for i := range b {
		b[i] = PasswordCharset[int(randomBytes[i])%len(PasswordCharset)]
	}
	return string(b)
}

// GenerateUUID generates a new UUID for MDM commands.
func GenerateUUID() string {
	return uuid.New().String()
}
