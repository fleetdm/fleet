package oval

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOvalDownloadDefinitionsMatchingHostInfo(t *testing.T) {
	ovalSources := map[Platform]string{
		"ubuntu-14": "",
		"ubuntu-18": "",
	}
	cases := []struct {
		platform Platform
		expected string
	}{
		{"ubuntu-14", "oval_ubuntu-14"},
		{"ubuntu-18", "oval_ubuntu-18"},
	}

	dw := func(a string, b string) error { return nil }

	for _, c := range cases {
		r, err := downloadDefinitions(ovalSources, c.platform, dw)
		require.Nil(t, err)
		require.Contains(t, r, c.expected)
	}
}

func TestOvalDownloadDefinitionsPlatformNotFound(t *testing.T) {
	ovalSources := map[Platform]string{"ubuntu-14": ""}
	dw := func(a string, b string) error { return nil }
	_, err := downloadDefinitions(ovalSources, "rhel-8", dw)
	require.ErrorContains(t, err, "could not find platform")
}
