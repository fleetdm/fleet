package dnf

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed test-data/dnf_upgradeable.txt
var dnf_upgradeable []byte

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
			input: []byte("\n tester..wow\n\n Last\n*^$\npackage.          1.2.3 source\n\nfoo.bar 111\n   \n"),
			expected: []map[string]string{
				{
					"package": "package",
					"source":  "source",
					"version": "1.2.3",
				},
			},
		},
		{
			name:  "dnf_upgradeable",
			input: dnf_upgradeable,
			expected: []map[string]string{
				{
					"package": "apr-util",
					"source":  "updates",
					"version": "1.5.2-6.el7_9.1",
				},
				{
					"package": "autofs",
					"source":  "updates",
					"version": "1:5.0.7-116.el7_9.1",
				},
				{
					"package": "bind-libs",
					"source":  "updates",
					"version": "32:9.11.4-26.P2.el7_9.13",
				},
				{
					"package": "brave-browser",
					"source":  "brave-browser-rpm-release.s3.brave.com_x86_64_",
					"version": "1.56.14-1",
				},
				{
					"package": "brave-keyring",
					"source":  "brave-browser-rpm-release.s3.brave.com_x86_64_",
					"version": "1.14-1",
				},
				{
					"package": "firefox",
					"source":  "updates",
					"version": "102.12.0-1.el7.centos",
				},
				{
					"package": "java-1.8.0-openjdk",
					"source":  "updates",
					"version": "1:1.8.0.372.b07-1.el7_9",
				},
				{
					"package": "java-1.8.0-openjdk-headless",
					"source":  "updates",
					"version": "1:1.8.0.372.b07-1.el7_9",
				},
				{
					"package": "openssl",
					"source":  "updates",
					"version": "1:1.0.2k-26.el7_9",
				},
				{
					"package": "openssl-libs",
					"source":  "updates",
					"version": "1:1.0.2k-26.el7_9",
				},
				{
					"package": "osquery",
					"source":  "osquery-s3-rpm-repo",
					"version": "5.9.1-1.linux",
				},
				{
					"package": "perf",
					"source":  "updates",
					"version": "3.10.0-1160.92.1.el7",
				},
				{
					"package": "python",
					"source":  "updates",
					"version": "2.7.5-93.el7_9",
				},
				{
					"package": "sudo",
					"source":  "updates",
					"version": "1.8.23-10.el7_9.3",
				},
				{
					"package": "zlib",
					"source":  "updates",
					"version": "1.2.7-21.el7_9",
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
