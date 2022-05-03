package oval

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOvalPlatformGetMajorRelease(t *testing.T) {
	cases := []struct {
		osVersion string
		expected  string
	}{
		{"CentOS Linux 8.3.2011", "8"},
		{"Ubuntu 20.4.0", "20"},
		{"CentOS 6.10.0", "6"},
		{"Debian GNU/Linux 9.0.0", "9"},
		{"Debian GNU/Linux 10.0.0", "10"},
		{"CentOS Linux 7.9.2009", "7"},
		{"Ubuntu 16.4.0", "16"},
		{"Ubuntu 18.4.0", "18"},
		{"Ubuntu 18.4", "18"},
		{"Ubuntu 18", "18"},
		{"", ""},
	}

	for _, c := range cases {
		require.Equal(t, c.expected, getMajorRelease(c.osVersion))
	}
}
