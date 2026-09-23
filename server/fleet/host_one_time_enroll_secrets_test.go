package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHostOneTimeEnrollSecretMatchesHost(t *testing.T) {
	enrollmentID := uint(7)
	hostID := uint(9)

	t.Run("enrollment bound", func(t *testing.T) {
		for _, tc := range []struct {
			name                                   string
			storedPlatform, storedUUID, storedSers string
			platform, uuid, serial                 string
			want                                   bool
		}{
			{
				// The case this whole branch exists for: minted before the device reported anything.
				name: "nothing captured", storedPlatform: "windows",
				platform: "windows", uuid: "UUID-1", serial: "",
				want: true,
			},
			{
				// A DevDetail response landed before the secret was minted, so the serial is on the row. Orbit still
				// presents none, because OrbitHostInfo.HardwareSerial is macOS and Linux only. Rejecting here would lock
				// out exactly the hosts furthest along in enrollment.
				name: "serial captured, agent presents none", storedPlatform: "windows", storedSers: "SERIAL-1",
				platform: "windows", uuid: "UUID-1", serial: "",
				want: true,
			},
			{
				name: "serial captured and agreed", storedPlatform: "windows", storedSers: "SERIAL-1",
				platform: "windows", uuid: "UUID-1", serial: "serial-1",
				want: true,
			},
			{
				// Both sides have one and they differ, which is the only case the serial can actually speak to.
				name: "serial captured and contradicted", storedPlatform: "windows", storedSers: "SERIAL-1",
				platform: "windows", uuid: "UUID-1", serial: "SERIAL-2",
				want: false,
			},
			{
				name: "platform contradicted", storedPlatform: "windows",
				platform: "darwin", uuid: "UUID-1", serial: "",
				want: false,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				s := &HostOneTimeEnrollSecret{
					MDMWindowsEnrollmentID: &enrollmentID,
					Platform:               tc.storedPlatform,
					HardwareUUID:           tc.storedUUID,
					HardwareSerial:         tc.storedSers,
				}
				require.Equal(t, tc.want, s.MatchesHost(tc.platform, tc.uuid, tc.serial))
			})
		}
	})

	t.Run("host bound stays exact", func(t *testing.T) {
		// The Apple path binds to a host that has already reported everything, so a missing identifier there is a
		// mismatch rather than a gap. The relaxation above must not reach it.
		s := &HostOneTimeEnrollSecret{
			HostID:         &hostID,
			Platform:       "darwin",
			HardwareUUID:   "UUID-1",
			HardwareSerial: "SERIAL-1",
		}
		require.True(t, s.MatchesHost("darwin", "uuid-1", "serial-1"))
		require.False(t, s.MatchesHost("darwin", "UUID-1", ""))
		require.False(t, s.MatchesHost("darwin", "", "SERIAL-1"))
	})
}
