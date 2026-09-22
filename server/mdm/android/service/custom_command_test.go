package service

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/androidmanagement/v1"
)

func TestAndroidCustomCommandType(t *testing.T) {
	for _, tc := range []struct {
		name string
		cmd  androidmanagement.Command
		want string
	}{
		{
			name: "explicit type wins",
			cmd:  androidmanagement.Command{Type: "REBOOT"},
			want: "REBOOT",
		},
		{
			name: "explicit WIPE",
			cmd:  androidmanagement.Command{Type: "WIPE", WipeParams: &androidmanagement.WipeParams{}},
			want: string(android.MDMAndroidCommandTypeWipe),
		},
		{
			// AMAPI sets the type to WIPE itself for this shape, and ProcessPubSubPush keys the
			// unenroll bookkeeping on the stored type, so it must not be persisted as CUSTOM.
			name: "wipeParams with no type is a wipe",
			cmd:  androidmanagement.Command{WipeParams: &androidmanagement.WipeParams{}},
			want: string(android.MDMAndroidCommandTypeWipe),
		},
		{
			name: "wipeParams carrying a reason is still a wipe",
			cmd: androidmanagement.Command{WipeParams: &androidmanagement.WipeParams{
				WipeReason: &androidmanagement.UserFacingMessage{DefaultMessage: "bye"},
			}},
			want: string(android.MDMAndroidCommandTypeWipe),
		},
		{
			// Other param-inferred types have no side effect in Fleet, so they stay CUSTOM.
			name: "other inferable params stay custom",
			cmd:  androidmanagement.Command{ClearAppsDataParams: &androidmanagement.ClearAppsDataParams{}},
			want: "CUSTOM",
		},
		{
			name: "empty command",
			cmd:  androidmanagement.Command{},
			want: "CUSTOM",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, androidCustomCommandType(&tc.cmd))
		})
	}
}
