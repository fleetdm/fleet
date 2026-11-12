package fleet

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateManagedLocalAccountPassword(t *testing.T) {
	t.Parallel()
	macOSAlphabet := managedAccountDigits + managedAccountUppercase
	windowsAlphabet := macOSAlphabet + managedAccountLowercase

	const iterations = 50

	for _, tt := range []struct {
		name             string
		includeLowercase bool
		alphabet         string
	}{
		{"macOS, single case", false, macOSAlphabet},
		{"Windows, with lowercase", true, windowsAlphabet},
	} {
		t.Run(tt.name, func(t *testing.T) {
			seen := make(map[string]struct{}, iterations)

			for range iterations {
				password := GenerateManagedLocalAccountPassword(tt.includeLowercase)
				seen[password] = struct{}{}

				groups := strings.Split(password, managedAccountPasswordSeparator)
				require.Len(t, groups, managedAccountPasswordGroupCount, "password %q", password)
				for _, group := range groups {
					require.Len(t, group, managedAccountPasswordGroupLen, "password %q", password)
					// Membership in this variant's alphabet is what holds the lowercase switch honest, and
					// what catches an index that runs past the end of the alphabet.
					for i := range len(group) {
						require.Contains(t, tt.alphabet, string(group[i]), "character outside the alphabet in %q", password)
					}
				}

				require.True(t, strings.ContainsAny(password, managedAccountDigits), "no digit in %q", password)
				require.True(t, strings.ContainsAny(password, managedAccountUppercase), "no uppercase in %q", password)
				if tt.includeLowercase {
					require.True(t, strings.ContainsAny(password, managedAccountLowercase), "no lowercase in %q", password)
				}
			}

			// Repeats do not happen by chance; they mean the generator has stopped being random.
			assert.Len(t, seen, iterations, "generated a duplicate password")
		})
	}
}
