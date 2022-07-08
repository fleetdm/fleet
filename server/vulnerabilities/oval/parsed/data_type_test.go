package oval_parsed

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewDataType(t *testing.T) {
	t.Run("NewDataType", func(t *testing.T) {
		cases := []struct {
			input    string
			expected DataType
		}{
			{"binary", Binary},
			{"boolean", Boolean},
			{"evr_string", EvrString},
			{"fileset_revision", FilesetRevision},
			{"float", Float},
			{"ios_version", IosVersion},
			{"int", Int},
			{"ipv4_address", Ipv4Address},
			{"ipv6_address", Ipv6Address},
			{"string", String},
			{"version", Version},
			{"asdafasdf", String},
			{"", String},
		}

		for _, c := range cases {
			require.Equal(t, c.expected, NewDataType(c.input))
		}
	})
}
