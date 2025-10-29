//go:build darwin
// +build darwin

package macos_user_profiles

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed testdata/profiles_output.xml
var testOutput []byte

func TestUnmarshalGenerate(t *testing.T) {
	t.Run("with valid username", func(t *testing.T) {
		profiles, err := unmarshalProfilesOutput(testOutput)
		require.NoError(t, err)

		rows := generateResults(profiles["martin"], "martin")
		require.Len(t, rows, 2)
		require.Equal(t, "Turn on automatic updates", rows[0]["display_name"])
		require.Equal(t, "Turn on set time and date automatically", rows[1]["display_name"])
	})

	t.Run("with invalid username", func(t *testing.T) {
		profiles, err := unmarshalProfilesOutput(testOutput)
		require.NoError(t, err)

		rows := generateResults(profiles["no-such-user"], "no-such-user")
		require.Len(t, rows, 0)
	})
}
