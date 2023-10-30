package apt

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed test-data/apt_upgradeable.txt
var apt_upgradeable []byte

func TestParse(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		name     string
		input    []byte
		expected []map[string]string
	}{
		{
			name:     "empty input",
			expected: make([]map[string]string, 0),
		},
		{
			name:  "malformed input",
			input: []byte("Listing...\naccounts\\jammy-updates, jammy-security  1 amd64 [%& . 22.07.5-2ubuntu1.3)\n\n\nfoobarservice/jammy-updates,jammy-security,security 22.05ubun1.2vv aarch [upgradable from: 22.05ubun1.3vv]\ntestshort/ubuntu-security 22.05 [upgradeable from: 22.05.1]\n\n"),
			expected: []map[string]string{
				{
					"package":         "foobarservice",
					"sources":         "jammy-updates,jammy-security,security",
					"update_version":  "22.05ubun1.2vv",
					"current_version": "22.05ubun1.3vv",
				},
			},
		},
		{
			name:  "apt_upgradeable",
			input: apt_upgradeable,
			expected: []map[string]string{
				{
					"package":         "accountsservice",
					"sources":         "jammy-updates,jammy-security",
					"update_version":  "22.07.5-2ubuntu1.4",
					"current_version": "22.07.5-2ubuntu1.3",
				},
				{
					"package":         "apt-utils",
					"sources":         "jammy-updates",
					"update_version":  "2.4.9",
					"current_version": "2.4.8",
				},
				{
					"package":         "apt",
					"sources":         "jammy-updates",
					"update_version":  "2.4.9",
					"current_version": "2.4.8",
				},
				{
					"package":         "base-files",
					"sources":         "jammy-updates",
					"update_version":  "12ubuntu4.3",
					"current_version": "12ubuntu4.2",
				},
				{
					"package":         "binutils-common",
					"sources":         "jammy-updates,jammy-security",
					"update_version":  "2.38-4ubuntu2.2",
					"current_version": "2.38-4ubuntu2.1",
				},
				{
					"package":         "binutils-x86-64-linux-gnu",
					"sources":         "jammy-updates,jammy-security",
					"update_version":  "2.38-4ubuntu2.2",
					"current_version": "2.38-4ubuntu2.1",
				},
				{
					"package":         "dpkg",
					"sources":         "jammy-updates",
					"update_version":  "1.21.1ubuntu2.2",
					"current_version": "1.21.1ubuntu2.1",
				},
				{
					"package":         "libkrb5-3",
					"sources":         "jammy-updates",
					"update_version":  "1.19.2-2ubuntu0.2",
					"current_version": "1.19.2-2ubuntu0.1",
				},
				{
					"package":         "libldap-common",
					"sources":         "jammy-updates,jammy-updates",
					"update_version":  "2.5.14+dfsg-0ubuntu0.22.04.2",
					"current_version": "2.5.13+dfsg-0ubuntu0.22.04.1",
				},
				{
					"package":         "openssl",
					"sources":         "jammy-updates,jammy-security",
					"update_version":  "3.0.2-0ubuntu1.10",
					"current_version": "3.0.2-0ubuntu1.8",
				},
				{
					"package":         "perl-base",
					"sources":         "jammy-updates,jammy-security",
					"update_version":  "5.34.0-3ubuntu1.2",
					"current_version": "5.34.0-3ubuntu1.1",
				},
				{
					"package":         "perl-modules-5.34",
					"sources":         "jammy-updates,jammy-updates,jammy-security,jammy-security",
					"update_version":  "5.34.0-3ubuntu1.2",
					"current_version": "5.34.0-3ubuntu1.1",
				},
				{
					"package":         "sudo",
					"sources":         "jammy-updates,jammy-security",
					"update_version":  "1.9.9-1ubuntu2.4",
					"current_version": "1.9.9-1ubuntu2.3",
				},
				{
					"package":         "vim",
					"sources":         "jammy-updates,jammy-security",
					"update_version":  "2:8.2.3995-1ubuntu2.9",
					"current_version": "2:8.2.3995-1ubuntu2.3",
				},
				{
					"package":         "xxd",
					"sources":         "jammy-updates,jammy-security",
					"update_version":  "2:8.2.3995-1ubuntu2.9",
					"current_version": "2:8.2.3995-1ubuntu2.3",
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := New()
			result, err := p.Parse(bytes.NewReader(tt.input))
			require.NoError(t, err, "unexpected error parsing input")

			require.ElementsMatch(t, tt.expected, result)
		})
	}
}
