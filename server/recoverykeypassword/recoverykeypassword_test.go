package recoverykeypassword

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratePassword(t *testing.T) {
	// Pattern: 6 groups of 4 characters from the allowed charset, separated by dashes
	pattern := regexp.MustCompile(`^[23456789ABCDEFGHJKLMNPQRSTUVWXYZ]{4}(-[23456789ABCDEFGHJKLMNPQRSTUVWXYZ]{4}){5}$`)

	t.Run("format", func(t *testing.T) {
		password, err := GeneratePassword()
		require.NoError(t, err)
		assert.True(t, pattern.MatchString(password), "password %q does not match expected format", password)
		assert.Len(t, password, 29) // 24 chars + 5 dashes
	})

	t.Run("excludes confusing characters", func(t *testing.T) {
		// Generate multiple passwords and check none contain confusing chars
		confusingChars := regexp.MustCompile(`[01OIl]`)
		for range 100 {
			password, err := GeneratePassword()
			require.NoError(t, err)
			assert.False(t, confusingChars.MatchString(password), "password %q contains confusing characters", password)
		}
	})

	t.Run("uniqueness", func(t *testing.T) {
		// Generate multiple passwords and verify they're unique
		seen := make(map[string]bool)
		for range 100 {
			password, err := GeneratePassword()
			require.NoError(t, err)
			assert.False(t, seen[password], "duplicate password generated: %s", password)
			seen[password] = true
		}
	})
}
