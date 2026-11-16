package service

import (
	"log/slog"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	nano_service "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"
)

// enrolledSerialCheckinService refuses an Authenticate whose serial belongs to a host enrolled under another UUID.
// Fleet's checkin service refuses the host update too, but it runs as a multi.New sub service whose errors never
// reach the device, so this wraps the whole chain like abOnlyEnrollmentCheckinService and refuses before nanomdm
// stores the enrollment.
type enrolledSerialCheckinService struct {
	nano_service.CheckinAndCommandService
	ds     fleet.Datastore
	logger *slog.Logger
}

func newEnrolledSerialCheckinService(next nano_service.CheckinAndCommandService, ds fleet.Datastore, logger *slog.Logger) nano_service.CheckinAndCommandService {
	return &enrolledSerialCheckinService{CheckinAndCommandService: next, ds: ds, logger: logger}
}

func (s *enrolledSerialCheckinService) Authenticate(r *mdm.Request, m *mdm.Authenticate) error {
	// An account-driven user enrollment has no UDID or serial here, and isn't matched by serial.
	if m.UDID != "" && m.SerialNumber != "" {
		enrolledUUID, err := s.ds.GetAppleSerialEnrolledUnderAnotherUUID(r.Context, m.SerialNumber, m.UDID)
		if err != nil {
			return ctxerr.Wrap(r.Context, err, "checking for a host enrolled with the serial")
		}
		if enrolledUUID != "" {
			s.logger.InfoContext(r.Context, "refusing MDM check-in: serial is enrolled under another UUID",
				"serial", m.SerialNumber, "claimed_uuid", m.UDID, "enrolled_uuid", enrolledUUID)
			return nano_service.NewHTTPStatusError(http.StatusConflict, &fleet.ConflictError{
				Message: "This device's serial number belongs to a host that's already enrolled. Delete the host or turn off MDM, then try again.",
			})
		}
	}
	return s.CheckinAndCommandService.Authenticate(r, m)
}
