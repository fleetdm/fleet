package oval

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOvalPlatform(t *testing.T) {
	t.Run("getMajorRelease", func(t *testing.T) {
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
	})

	t.Run("ToFilename", func(t *testing.T) {
		cases := []struct {
			date     time.Time
			expected string
		}{
			{time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), "fleet_oval_ubuntu-20_2020-01-01.json"},
			{time.Date(2020, 10, 10, 0, 0, 0, 0, time.UTC), "fleet_oval_ubuntu-20_2020-10-10.json"},
		}
		for _, c := range cases {
			plat := NewPlatform("ubuntu", "Ubuntu 20.4.0")
			require.Equal(t, c.expected, plat.ToFilename(c.date, "json"))
		}
	})
}
