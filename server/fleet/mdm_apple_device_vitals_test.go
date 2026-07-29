package fleet

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMDMAppleCellularTechnologyJSON(t *testing.T) {
	// Apple reports CellularTechnology as an integer; the API returns its
	// display label. The integer stays the persisted representation.
	cases := []struct {
		raw  int64
		want string
	}{
		{0, `"None"`},
		{1, `"GSM"`},
		{2, `"CDMA"`},
		{3, `"GSM and CDMA"`},
		// A value outside Apple's documented set must not fail serialization
		// of the whole host response.
		{4, `"unknown"`},
	}
	for _, c := range cases {
		got, err := json.Marshal(MDMAppleCellularTechnology(c.raw))
		require.NoError(t, err)
		require.JSONEq(t, c.want, string(got))
	}

	// Round-trips from the display label, so a client (or Fleet's own test
	// suite) can unmarshal a host response it just received.
	for _, c := range cases[:4] {
		var ct MDMAppleCellularTechnology
		require.NoError(t, json.Unmarshal([]byte(c.want), &ct))
		require.EqualValues(t, c.raw, ct)
	}

	// The raw integer is also accepted, so a value straight from the ack or
	// the database decodes.
	var ct MDMAppleCellularTechnology
	require.NoError(t, json.Unmarshal([]byte("2"), &ct))
	require.Equal(t, MDMAppleCellularTechnologyCDMA, ct)

	// An unrecognized label decodes to the sentinel rather than erroring.
	require.NoError(t, json.Unmarshal([]byte(`"LTE"`), &ct))
	require.Equal(t, MDMAppleCellularTechnologyUnknown, ct)

	// Absent (nil pointer) is omitted entirely, not rendered as a label.
	b, err := json.Marshal(struct {
		CellularTechnology *MDMAppleCellularTechnology `json:"cellular_technology,omitempty"`
	}{})
	require.NoError(t, err)
	require.JSONEq(t, `{}`, string(b))
}
