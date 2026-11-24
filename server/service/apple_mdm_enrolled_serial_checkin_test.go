package service

import (
	"context"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	nano_service "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

type authenticateRecorder struct {
	nano_service.CheckinAndCommandService
	called bool
}

func (a *authenticateRecorder) Authenticate(*mdm.Request, *mdm.Authenticate) error {
	a.called = true
	return nil
}

func TestEnrolledSerialCheckinServiceAccountDriven(t *testing.T) {
	ds := new(mock.Store)
	ds.GetAppleAccountDrivenEnrollmentIDTakenFunc = func(ctx context.Context, enrollmentID string) (bool, error) {
		return enrollmentID == "victim-uuid", nil
	}

	for _, c := range []struct {
		id, serial  string
		wantRefused bool
	}{
		{"victim-uuid", "", true},
		// a claimed serial doesn't change which ID nanomdm stores the enrollment under
		{"victim-uuid", "some-serial", true},
		{"own-enrollment-id", "", false},
	} {
		next := &authenticateRecorder{}
		svc := newEnrolledSerialCheckinService(next, ds, slog.New(slog.DiscardHandler))
		m := &mdm.Authenticate{SerialNumber: c.serial}
		m.EnrollmentID = c.id
		err := svc.Authenticate(&mdm.Request{Context: t.Context()}, m)
		if c.wantRefused {
			require.Error(t, err, c)
		} else {
			require.NoError(t, err, c)
		}
		require.Equal(t, !c.wantRefused, next.called, c)
	}
}
