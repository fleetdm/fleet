package cryptsetup

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStatusErrors(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		input string
	}{
		{
			input: "",
		},
		{
			input: "\n\n\n\n",
		},
		{
			input: "type:    LUKS2",
		},
		{
			input: "Hello world",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			t.Parallel()

			data, err := parseStatus([]byte(tt.input))
			assert.Error(t, err, "parseStatus")
			assert.Nil(t, data, "data is nil")

		})
	}

}
func TestParseStatus(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		infile       string
		len          int
		status       string
		mounted      bool
		ctype        string
		keysize      string
		key_location string
	}{
		{
			infile:       "status-active-luks1.txt",
			status:       "active",
			mounted:      true,
			ctype:        "LUKS1",
			keysize:      "512 bits",
			key_location: "dm-crypt",
		},
		{
			infile:       "status-active-luks2.txt",
			status:       "active",
			mounted:      true,
			ctype:        "LUKS2",
			keysize:      "512 bits",
			key_location: "keyring",
		},
		{
			infile:       "status-active-mounted.txt",
			status:       "active",
			mounted:      true,
			ctype:        "PLAIN",
			keysize:      "256 bits",
			key_location: "dm-crypt",
		},
		{
			infile:       "status-active-umounted.txt",
			status:       "active",
			ctype:        "PLAIN",
			keysize:      "256 bits",
			key_location: "dm-crypt",
		},
		{
			infile:       "status-active.txt",
			status:       "active",
			mounted:      true,
			ctype:        "PLAIN",
			keysize:      "256 bits",
			key_location: "dm-crypt",
		},
		{
			infile: "status-error.txt",
			status: "not_found",
		},
		{
			infile: "status-inactive.txt",
			status: "inactive",
		},
		{
			infile: "status-unactive.txt",
			status: "inactive",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.infile, func(t *testing.T) {
			t.Parallel()

			input, err := os.ReadFile(filepath.Join("testdata", tt.infile))
			require.NoError(t, err, "read input file")

			data, err := parseStatus(input)
			require.NoError(t, err, "parseStatus")

			assert.Equal(t, tt.status, data["status"], "status")
			assert.Equal(t, strconv.FormatBool(tt.mounted), data["mounted"], "mounted")

			// These values aren't populated in the map,
			// so only check them if the test case lists
			// them
			if tt.ctype != "" {
				assert.Equal(t, tt.ctype, data["type"], "type")
				assert.Equal(t, tt.keysize, data["keysize"], "keysize")
				assert.Equal(t, tt.key_location, data["key_location"], "key_location")
			}
		})
	}
}
