package oval

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOvalPlatform(t *testing.T) {
	t.Run("NewPlatform", func(t *testing.T) {
		cases := []struct {
			platform  string
			osVersion string
			expected  string
		}{
			{"centos", "CentOS Linux 8.3.2011", "centos_08"},
			{"ubuntu", "Ubuntu 20.4.0", "ubuntu_2004"},
			{"centos", "CentOS 6.10.0", "centos_06"},
			{"debian", "Debian GNU/Linux 9.0.0", "debian_09"},
			{"debian", "Debian GNU/Linux 10.0.0", "debian_10"},
			{"centos", "CentOS Linux 7.9.2009", "centos_07"},
			{"ubuntu", "Ubuntu 16.4.0", "ubuntu_1604"},
			{"ubuntu", "Ubuntu 18.4.0", "ubuntu_1804"},
			{"ubuntu", "Ubuntu 18.4", "ubuntu_1804"},
			{"ubuntu", "Ubuntu 18.4.0 ", "ubuntu_1804"},
			{"rhel", "CentOS Linux 7.9.2009", "rhel_07"},
		}

		for _, c := range cases {
			require.Equal(t, c.expected, string(NewPlatform(c.platform, c.osVersion)))
		}
	})

	t.Run("ToFilename", func(t *testing.T) {
		cases := []struct {
			date     time.Time
			expected string
		}{
			{time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), "fleet_oval_ubuntu_2004-2020_01_01.json"},
			{time.Date(2020, 10, 10, 0, 0, 0, 0, time.UTC), "fleet_oval_ubuntu_2004-2020_10_10.json"},
		}
		for _, c := range cases {
			plat := NewPlatform("ubuntu", "Ubuntu 20.4.0")
			require.Equal(t, c.expected, plat.ToFilename(c.date, "json"))
		}
	})
}
