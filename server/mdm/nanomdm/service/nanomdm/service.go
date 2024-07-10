// Pacakge nanomdm is an MDM service.
package nanomdm

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/log"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/log/ctxlog"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"
)

// Service is the main NanoMDM service which dispatches to storage.
type Service struct {
	logger     log.Logger
	normalizer func(e *mdm.Enrollment) *mdm.EnrollID
	store      storage.ServiceStore

	// By default the UserAuthenticate message will be rejected (410
	// response). If this is set true then a static zero-length
	// digest challenge will be supplied to the first UserAuthenticate
	// check-in message. See the Discussion section of
	// https://developer.apple.com/documentation/devicemanagement/userauthenticate
	sendEmptyDigestChallenge bool
	storeRejectedUserAuth    bool

	// Declarative Management
	dm service.DeclarativeManagement
}

// normalize generates enrollment IDs that are used by other
// services and the storage backend. Enrollment IDs need not
// necessarily be related to the UDID, UserIDs, or other identifiers
// sent in the request, but by convention that is what this normalizer
// uses.
//
// Device enrollments are identified by the UDID or EnrollmentID. User
// enrollments are then appended after a colon (":"). Note that the
// storage backends depend on the ParentID field matching a device
// enrollment so that the "parent" (device) enrollment can be
// referenced.
func normalize(e *mdm.Enrollment) *mdm.EnrollID {
	r := e.Resolved()
	if r == nil {
		return nil
	}
	eid := &mdm.EnrollID{
		Type: r.Type,
		ID:   r.DeviceChannelID,
	}
	if r.IsUserChannel {
		eid.ID += ":" + r.UserChannelID
		eid.ParentID = r.DeviceChannelID
	}
	return eid
}

type Option func(*Service)

func WithLogger(logger log.Logger) Option {
	return func(s *Service) {
		s.logger = logger
	}
}

func WithDeclarativeManagement(dm service.DeclarativeManagement) Option {
	return func(s *Service) {
		s.dm = dm
	}
}

// New returns a new NanoMDM main service.
func New(store storage.ServiceStore, opts ...Option) *Service {
	nanomdm := &Service{
		store:      store,
		logger:     log.NopLogger,
		normalizer: normalize,
	}
	for _, opt := range opts {
		opt(nanomdm)
	}
	return nanomdm
}

func (s *Service) setupRequest(r *mdm.Request, e *mdm.Enrollment) error {
	if r.EnrollID != nil && r.ID != "" {
		ctxlog.Logger(r.Context, s.logger).Debug(
			"msg", "overwriting enrollment id",
		)
	}
	r.EnrollID = s.normalizer(e)
	if err := r.EnrollID.Validate(); err != nil {
		return err
	}
	r.Context = newContextWithValues(r.Context, r)
	r.Context = ctxlog.AddFunc(r.Context, ctxKVs)
	return nil
}

// Authenticate Check-in message implementation.
func (s *Service) Authenticate(r *mdm.Request, message *mdm.Authenticate) error {
	if err := s.setupRequest(r, &message.Enrollment); err != nil {
		return err
	}
	logs := []interface{}{
		"msg", "Authenticate",
	}
	if message.SerialNumber != "" {
		logs = append(logs, "serial_number", message.SerialNumber)
	}
	ctxlog.Logger(r.Context, s.logger).Info(logs...)
	if err := s.store.StoreAuthenticate(r, message); err != nil {
		return err
	}
	// clear the command queue for any enrollment or sub-enrollment.
	// this prevents queued commands still being queued after device
	// unenrollment.
	if err := s.store.ClearQueue(r); err != nil {
		return err
	}
	// then, disable the enrollment or any sub-enrollment (because an
	// enrollment is only valid after a tokenupdate)
	return s.store.Disable(r)
}

// TokenUpdate Check-in message implementation.
func (s *Service) TokenUpdate(r *mdm.Request, message *mdm.TokenUpdate) error {
	if err := s.setupRequest(r, &message.Enrollment); err != nil {
		return err
	}
	ctxlog.Logger(r.Context, s.logger).Info("msg", "TokenUpdate")
	return s.store.StoreTokenUpdate(r, message)
}

// CheckOut Check-in message implementation.
func (s *Service) CheckOut(r *mdm.Request, message *mdm.CheckOut) error {
	if err := s.setupRequest(r, &message.Enrollment); err != nil {
		return err
	}
	ctxlog.Logger(r.Context, s.logger).Info("msg", "CheckOut")
	return s.store.Disable(r)
}

const emptyDigestChallenge = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>DigestChallenge</key>
	<string></string>
</dict>
</plist>`

var emptyDigestChallengeBytes = []byte(emptyDigestChallenge)

func (s *Service) UserAuthenticate(r *mdm.Request, message *mdm.UserAuthenticate) ([]byte, error) {
	if err := s.setupRequest(r, &message.Enrollment); err != nil {
		return nil, err
	}
	logger := ctxlog.Logger(r.Context, s.logger)
	if s.sendEmptyDigestChallenge || s.storeRejectedUserAuth {
		if err := s.store.StoreUserAuthenticate(r, message); err != nil {
			return nil, err
		}
	}
	// if the DigestResponse is empty then this is the first (of two)
	// UserAuthenticate messages depending on our response
	if message.DigestResponse == "" {
		if s.sendEmptyDigestChallenge {
			logger.Info(
				"msg", "sending empty DigestChallenge response to UserAuthenticate",
			)
			return emptyDigestChallengeBytes, nil
		}
		return nil, service.NewHTTPStatusError(
			http.StatusGone,
			fmt.Errorf("declining management of user: %s", r.ID),
		)
	}
	logger.Debug(
		"msg", "sending empty response to second UserAuthenticate",
	)
	return nil, nil
}

func (s *Service) SetBootstrapToken(r *mdm.Request, message *mdm.SetBootstrapToken) error {
	if err := s.setupRequest(r, &message.Enrollment); err != nil {
		return err
	}
	ctxlog.Logger(r.Context, s.logger).Info("msg", "SetBootstrapToken")
	return s.store.StoreBootstrapToken(r, message)
}

func (s *Service) GetBootstrapToken(r *mdm.Request, message *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	if err := s.setupRequest(r, &message.Enrollment); err != nil {
		return nil, err
	}
	ctxlog.Logger(r.Context, s.logger).Info("msg", "GetBootstrapToken")
	return s.store.RetrieveBootstrapToken(r, message)
}

// DeclarativeManagement Check-in message implementation. Calls out to
// the service's DM handler (if configured).
func (s *Service) DeclarativeManagement(r *mdm.Request, message *mdm.DeclarativeManagement) ([]byte, error) {
	if err := s.setupRequest(r, &message.Enrollment); err != nil {
		return nil, err
	}
	ctxlog.Logger(r.Context, s.logger).Info(
		"msg", "DeclarativeManagement",
		"endpoint", message.Endpoint,
	)
	if s.dm == nil {
		return nil, errors.New("no Declarative Management handler")
	}
	return s.dm.DeclarativeManagement(r, message)
}

// CommandAndReportResults command report and next-command request implementation.
func (s *Service) CommandAndReportResults(r *mdm.Request, results *mdm.CommandResults) (*mdm.Command, error) {
	if err := s.setupRequest(r, &results.Enrollment); err != nil {
		return nil, err
	}
	logger := ctxlog.Logger(r.Context, s.logger)
	logs := []interface{}{
		"status", results.Status,
	}
	if results.CommandUUID != "" {
		logs = append(logs, "command_uuid", results.CommandUUID)
	}
	logger.Info(logs...)
	err := s.store.StoreCommandReport(r, results)
	if err != nil {
		// allow not found commands, this is an edge case only
		// valid for migrations, other response codes confuse the
		// mdmclient, and this gives us the opportunity to answer
		// with more commands
		if !service.IsNotFound(err) {
			return nil, fmt.Errorf("storing command report: %w", err)
		}

		logger.Debug(
			"msg", "host reported status with invalid command uuid",
			"command_uuid", results.CommandUUID,
			"request_type", results.RequestType,
		)
	}
	cmd, err := s.store.RetrieveNextCommand(r, results.Status == "NotNow")
	if err != nil {
		return nil, fmt.Errorf("retrieving next command: %w", err)
	}
	if cmd != nil {
		logger.Debug(
			"msg", "command retrieved",
			"command_uuid", cmd.CommandUUID,
			"request_type", cmd.Command.RequestType,
		)
		return cmd, nil
	}
	logger.Debug(
		"msg", "no command retrieved",
	)
	return nil, nil
}
