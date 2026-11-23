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

	for id, wantRefused := range map[string]bool{"victim-uuid": true, "own-enrollment-id": false} {
		next := &authenticateRecorder{}
		svc := newEnrolledSerialCheckinService(next, ds, slog.New(slog.DiscardHandler))
		err := svc.Authenticate(&mdm.Request{Context: t.Context()}, &mdm.Authenticate{EnrollmentID: id})
		if wantRefused {
			require.Error(t, err, id)
		} else {
			require.NoError(t, err, id)
		}
		require.Equal(t, !wantRefused, next.called, id)
	}
}
