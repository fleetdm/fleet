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
			{"ubuntu", "Ubuntu 18.4.0 LTS asdfasd", "ubuntu_1804"},
			{"rhel", "CentOS Linux 7.9.2009", "rhel_07"},
			{"amzn", "Amazon Linux 2.0.0", "amzn_02"},
			{"amzn", "Amazon Linux 2023.0.0", "amzn_2023"},
			{"rhel", "Fedora Linux 12.0.0", "rhel_06"},
			{"rhel", "Fedora Linux 13.0.0", "rhel_06"},
			{"rhel", "Fedora Linux 14.0.0", "rhel_06"},
			{"rhel", "Fedora Linux 15.0.0", "rhel_06"},
			{"rhel", "Fedora Linux 16.0.0", "rhel_06"},
			{"rhel", "Fedora Linux 17.0.0", "rhel_06"},
			{"rhel", "Fedora Linux 18.0.0", "rhel_06"},
			{"rhel", "Fedora Linux 19.0.0", "rhel_07"},
			{"rhel", "Fedora Linux 20.0.0", "rhel_07"},
			{"rhel", "Fedora Linux 21.0.0", "rhel_07"},
			{"rhel", "Fedora Linux 22.0.0", "rhel_07"},
			{"rhel", "Fedora Linux 23.0.0", "rhel_07"},
			{"rhel", "Fedora Linux 24.0.0", "rhel_07"},
			{"rhel", "Fedora Linux 25.0.0", "rhel_07"},
			{"rhel", "Fedora Linux 26.0.0", "rhel_07"},
			{"rhel", "Fedora Linux 27.0.0", "rhel_07"},
			{"rhel", "Fedora Linux 28.0.0", "rhel_08"},
			{"rhel", "Fedora Linux 29.0.0", "rhel_08"},
			{"rhel", "Fedora Linux 30.0.0", "rhel_08"},
			{"rhel", "Fedora Linux 31.0.0", "rhel_08"},
			{"rhel", "Fedora Linux 32.0.0", "rhel_08"},
			{"rhel", "Fedora Linux 33.0.0", "rhel_08"},
			{"rhel", "Fedora Linux 34.0.0", "rhel_09"},
			{"rhel", "Fedora Linux 35.0.0", "rhel_09"},
			{"rhel", "Fedora Linux 36.0.0", "rhel_09"},
			{"ubuntu", "Ubuntu 20.04.2 LTS", "ubuntu_2004"},
		}

		for _, c := range cases {
			require.Equal(t, c.expected, string(NewPlatform(c.platform, c.osVersion)), c)
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

	t.Run("ToGovalDictionaryFilename", func(t *testing.T) {
		cases := []struct {
			version  string
			expected string
		}{
			{"Amazon Linux 2.0.0", "fleet_goval_dictionary_amzn_02.sqlite3"},
			{"Amazon Linux 2023.0.0", "fleet_goval_dictionary_amzn_2023.sqlite3"},
		}
		for _, c := range cases {
			plat := NewPlatform("amzn", c.version)
			require.Equal(t, c.expected, plat.ToGovalDictionaryFilename())
		}
	})
}
