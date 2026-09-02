package update

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsActiveFleetEnrollment(t *testing.T) {
	const fleetGUID = "39771ECF-778A-41BD-AD7A-C6DA11E20FC8"

	testCases := []struct {
		name       string
		providerID string
		state      uint64
		subkeyName string
		want       bool
	}{
		{name: "enrolled state 1", providerID: "Fleet", state: 1, subkeyName: fleetGUID, want: true},
		// #48760: the previous code pinned to state == 1 and rejected 3, the value seen on affected devices, so on-demand syncs failed.
		{name: "enrolled state 3", providerID: "Fleet", state: 3, subkeyName: fleetGUID, want: true},
		{name: "state 0 rejected", providerID: "Fleet", state: 0, subkeyName: fleetGUID, want: false},
		{name: "non-fleet provider rejected", providerID: "MS DM Server", state: 3, subkeyName: fleetGUID, want: false},
		{name: "malformed subkey name rejected", providerID: "Fleet", state: 3, subkeyName: "not-a-guid", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isActiveFleetEnrollment(tc.providerID, tc.state, tc.subkeyName))
		})
	}
}
