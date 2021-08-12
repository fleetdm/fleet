package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVulnerabilitiesScan(t *testing.T) {
	v := &VulnerabilitiesSlice{}
	var errorTests = []struct {
		name       string
		jsonString string
		errStr     string
	}{
		{
			"bad json input",
			`{"badjson"`,
			`src=[{"badjson"]: invalid character ']' after object key`,
		},
		{
			"json object instead of slice",
			`"something"`,
			`src="something": json: cannot unmarshal string into Go value of type fleet.VulnerabilitiesSlice`,
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Scan([]byte(tt.jsonString))
			require.Error(t, err)
			require.Equal(t, tt.errStr, err.Error())
		})
	}
}
