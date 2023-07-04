//go:build darwin

package profiles

import (
	"bytes"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestGetFleetdConfig(t *testing.T) {
	testErr := errors.New("test error")
	cases := []struct {
		cmdOut  *string
		cmdErr  error
		wantOut *fleet.MDMAppleFleetdConfig
		wantErr string
	}{
		{nil, testErr, nil, testErr.Error()},
		{ptr.String("invalid-json"), nil, nil, "unmarshaling configuration"},
		{ptr.String("{}"), nil, nil, ErrNotFound.Error()},
		{
			ptr.String(`{"EnrollSecret": "ENROLL_SECRET", "FleetURL": "https://test.example.com"}`),
			nil,
			&fleet.MDMAppleFleetdConfig{
				EnrollSecret: "ENROLL_SECRET",
				FleetURL:     "https://test.example.com",
			},
			"",
		},
		{
			ptr.String(`{"EnrollSecret": "ENROLL_SECRET", "FleetURL": ""}`),
			nil,
			nil,
			ErrNotFound.Error(),
		},
		{
			ptr.String(`{"EnrollSecret": "", "FleetURL": "https://test.example.com"}`),
			nil,
			nil,
			ErrNotFound.Error(),
		},
	}

	origExecScript := execScript
	t.Cleanup(func() { execScript = origExecScript })
	for _, c := range cases {
		execScript = func(script string) (*bytes.Buffer, error) {
			if c.cmdOut == nil {
				return nil, c.cmdErr
			}

			var buf bytes.Buffer
			buf.WriteString(*c.cmdOut)
			return &buf, nil
		}

		out, err := GetFleetdConfig()
		if c.wantErr != "" {
			require.ErrorContains(t, err, c.wantErr)
		} else {
			require.NoError(t, err)
		}
		require.Equal(t, c.wantOut, out)
	}

}
