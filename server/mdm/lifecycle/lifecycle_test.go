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
	lf := New(ds, kitlog.NewNopLogger())

	err := lf.Do(context.Background(), HostOptions{})
	require.ErrorContains(t, err, "unsupported platform")

	err = lf.Do(context.Background(), HostOptions{Platform: "linux"})
	require.ErrorContains(t, err, "unsupported platform")

	err = lf.Do(context.Background(), HostOptions{Platform: "darwin", Action: "invalid"})
	require.ErrorContains(t, err, "unknown action")

	err = lf.Do(context.Background(), HostOptions{Platform: "windows", Action: "invalid"})
	require.ErrorContains(t, err, "unknown action")
}

func TestDoParamValidation(t *testing.T) {
	ds := new(mock.Store)
	lf := New(ds, kitlog.NewNopLogger())
	ctx := context.Background()

	actions := []HostAction{
		HostActionTurnOn, HostActionTurnOff,
		HostActionReset, HostActionDelete,
	}

	platforms := []string{
		"darwin", "windows",
	}

	for _, platform := range platforms {
		for _, action := range actions {
			err := lf.Do(ctx, HostOptions{
				Action:   action,
				Platform: platform,
			})
			require.ErrorContains(t, err, "required")
		}
	}
}
