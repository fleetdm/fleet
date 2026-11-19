package file

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func FuzzDecodeMSITables(f *testing.F) {
	encode := func(values ...uint16) []byte {
		var data []byte
		for _, value := range values {
			data = binary.LittleEndian.AppendUint16(data, value)
		}
		return data
	}
	data := []byte("PropertyNameValueProductNameApp")
	pool := encode(0, 0, 8, 1, 4, 1, 5, 1, 11, 1, 3, 1)
	columns := encode(1, 1, 1, 2, 2, 3, 0xD00, 0xF00)
	f.Add(data, pool, columns, encode(4, 5))
	f.Add(data, pool, columns, encode(0, 65535))
	f.Add(data, pool, encode(0, 65535, 1, 2, 2, 3, 0xD00, 0xF00), encode(4, 5))
	f.Add([]byte{}, encode(0, 0, 0, 1, 65535, 65535), []byte{}, []byte{})
	f.Add([]byte{}, []byte{}, []byte{}, []byte{})

	f.Fuzz(func(t *testing.T, data, pool, columns, properties []byte) {
		if len(data)+len(pool)+len(columns)+len(properties) > 64<<10 {
			t.Skip("MSI tables exceed fuzz input budget")
		}
		stringTable, err := decodeStrings(bytes.NewReader(data), bytes.NewReader(pool))
		if err != nil {
			return
		}
		require.True(t, bytes.HasPrefix(data, []byte(strings.Join(stringTable, ""))),
			"decoded strings must consume a prefix of the string data")

		table, err := decodePropertyTableColumns(bytes.NewReader(columns), stringTable)
		if err != nil {
			return
		}
		props, err := decodePropertyTable(bytes.NewReader(properties), table, stringTable)
		if err != nil {
			return
		}
		knownStrings := make(map[string]struct{}, len(stringTable))
		for _, value := range stringTable {
			knownStrings[value] = struct{}{}
		}
		for key, value := range props {
			_, knownKey := knownStrings[key]
			_, knownValue := knownStrings[value]
			require.True(t, knownKey, "property name must come from the string table")
			require.True(t, knownValue, "property value must come from the string table")
		}
	})
}
