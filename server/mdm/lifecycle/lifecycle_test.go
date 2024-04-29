package mdmlifecycle

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mock"
	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestDoUnsupportedParams(t *testing.T) {
	ds := new(mock.Store)
	lc := New(ds, kitlog.NewNopLogger())

	err := lc.Do(context.Background(), HostOptions{})
	require.ErrorContains(t, err, "unsupported platform")

	err = lc.Do(context.Background(), HostOptions{Platform: "linux"})
	require.ErrorContains(t, err, "unsupported platform")

	err = lc.Do(context.Background(), HostOptions{Platform: "darwin", Action: "invalid"})
	require.ErrorContains(t, err, "unknown action")

	err = lc.Do(context.Background(), HostOptions{Platform: "windows", Action: "invalid"})
	require.ErrorContains(t, err, "unknown action")
}

func TestDoParamValidation(t *testing.T) {
	ds := new(mock.Store)
	lf := New(ds, kitlog.NewNopLogger())
	ctx := context.Background()

	cases := []struct {
		platform string
		action   HostAction
		wantErr  bool
	}{

		{"darwin", HostActionTurnOn, true},
		{"darwin", HostActionTurnOff, true},
		{"darwin", HostActionReset, true},
		{"darwin", HostActionDelete, true},
		{"windows", HostActionTurnOn, true},
		{"windows", HostActionTurnOff, true},
		{"windows", HostActionReset, true},
		{"windows", HostActionDelete, false},
	}

	for _, tc := range cases {
		err := lf.Do(ctx, HostOptions{
			Action:   tc.action,
			Platform: tc.platform,
		})
		if tc.wantErr {
			require.ErrorContains(t, err, "required")
		} else {
			require.NoError(t, err)
		}
	}
}
