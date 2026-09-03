package service

import (
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	nano_service "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"
)

// abOnlyEnrollmentCheckinService wraps the full MDM checkin/command service chain
// (nanomdm's core service plus Fleet's own, dispatched together via multi.New) so that
// rejecting an Authenticate actually reaches the device.
//
// multi.New treats every service after the first as a "sub service": their errors are only
// logged (see MultiService.runOthers), never returned to the HTTP caller — only the first
// service's return value determines the response. Fleet's own checkin service is registered
// as a sub service on purpose (its many side effects — host creation, activity logging —
// must not fail the live MDM protocol handshake if e.g. a DB write has a transient error), so
// a business rule that needs to actually block enrollment cannot live there. This wrapper is
// placed around the whole chain instead (same spot certauth.New wraps it), so its Authenticate
// return value is what the HTTP handler sees.
type abOnlyEnrollmentCheckinService struct {
	nano_service.CheckinAndCommandService
	ds     fleet.Datastore
	logger *slog.Logger
}

// newABOnlyEnrollmentCheckinService wraps next with the Apple Business-only enrollment check.
func newABOnlyEnrollmentCheckinService(next nano_service.CheckinAndCommandService, ds fleet.Datastore, logger *slog.Logger) nano_service.CheckinAndCommandService {
	return &abOnlyEnrollmentCheckinService{CheckinAndCommandService: next, ds: ds, logger: logger}
}

func (s *abOnlyEnrollmentCheckinService) Authenticate(r *mdm.Request, m *mdm.Authenticate) error {
	appCfg, err := s.ds.AppConfig(r.Context)
	if err != nil {
		return ctxerr.Wrap(r.Context, err, "loading app config for AB-only enrollment check")
	}
	if appCfg.MDM.OnlyAllowAppleBusinessEnrollment {
		assignments, err := s.ds.GetHostDEPAssignmentsBySerial(r.Context, m.SerialNumber)
		if err != nil {
			return ctxerr.Wrap(r.Context, err, "checking DEP assignment for AB-only enrollment")
		}
		if len(assignments) == 0 {
			// r.EnrollID is nil here: this wrapper runs before nanomdm's core service, which
			// is what populates it (see the "critical" ordering note on multi.New's call
			// site in handler.go). Log the identifiers available directly on the message
			// instead of the usual r.ID.
			s.logger.InfoContext(r.Context, "rejecting enrollment: serial not assigned in Apple Business",
				"udid", m.UDID, "serial", m.SerialNumber,
			)
			returnErr := fleet.ABOnlyEnrollmentForbiddenError{}
			return nano_service.NewHTTPStatusError(returnErr.StatusCode(), &returnErr)
		}
	}
	return s.CheckinAndCommandService.Authenticate(r, m)
}
