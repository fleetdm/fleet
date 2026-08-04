package fleet

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateManagedLocalAccountPassword(t *testing.T) {
	t.Parallel()

	const iterations = 50

	for _, tt := range []struct {
		name     string
		generate func() string
		classes  []string
	}{
		{
			"macOS, uppercase and digits only",
			GenerateManagedLocalAccountPasswordForMacOS,
			[]string{managedAccountDigits, managedAccountUppercase},
		},
		{
			"Windows, also lowercase and special characters",
			GenerateManagedLocalAccountPasswordForWindows,
			[]string{managedAccountDigits, managedAccountUppercase, managedAccountLowercase, managedAccountSpecial},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			alphabet := strings.Join(tt.classes, "")
			seen := make(map[string]struct{}, iterations)

			for range iterations {
				password := tt.generate()
				seen[password] = struct{}{}

				groups := strings.Split(password, managedAccountPasswordSeparator)
				require.Len(t, groups, managedAccountPasswordGroupCount, "password %q", password)
				for _, group := range groups {
					require.Len(t, group, managedAccountPasswordGroupLen, "password %q", password)
					// Membership in this variant's alphabet is what keeps the classes each platform gets honest,
					// and what catches an index that runs past the end of the alphabet.
					for i := range len(group) {
						require.Contains(t, alphabet, string(group[i]), "character outside the alphabet in %q", password)
					}
				}

				// Every class this variant draws from must appear, which is the guarantee the redraw loop exists for.
				for _, class := range tt.classes {
					require.True(t, strings.ContainsAny(password, class), "no character from %q in %q", class, password)
				}
			}

			// Repeats do not happen by chance; they mean the generator has stopped being random.
			assert.Len(t, seen, iterations, "generated a duplicate password")
		})
	}
}

// The classes are concatenated into one alphabet and indexed uniformly, so these two properties are what make the
// generator's output well formed. Neither is visible at a call site, which is why they are asserted directly.
func TestManagedAccountCharacterClasses(t *testing.T) {
	t.Parallel()

	classes := map[string]string{
		"digits":    managedAccountDigits,
		"uppercase": managedAccountUppercase,
		"lowercase": managedAccountLowercase,
		"special":   managedAccountSpecial,
	}

	// A class holding the separator would split into the wrong number of groups, and an admin reading the password off
	// a screen could not tell a separator from a password character.
	t.Run("no class contains the group separator", func(t *testing.T) {
		for name, class := range classes {
			assert.NotContains(t, class, managedAccountPasswordSeparator, "the %s class contains the group separator", name)
		}
	})

	// A character in two classes would be drawn twice as often as its neighbours, quietly biasing the password.
	t.Run("the classes do not overlap", func(t *testing.T) {
		owner := make(map[rune]string)
		for name, class := range classes {
			for _, char := range class {
				if previous, duplicated := owner[char]; duplicated {
					t.Errorf("%q appears in both the %s and %s classes", char, previous, name)
				}
				owner[char] = name
			}
		}
	})
}
