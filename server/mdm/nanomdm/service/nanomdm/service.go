// Pacakge nanomdm is an MDM service.
package nanomdm

import (
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

// Service is the main NanoMDM service which dispatches to storage.
type Service struct {
	logger     log.Logger
	normalizer func(e *mdm.Enrollment) *mdm.EnrollID
	store      storage.ServiceStore

	// Declarative Management
	dm service.DeclarativeManagement

	// UserAuthenticate processor
	ua service.UserAuthenticate

	// GetToken handler
	gt service.GetToken
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

// WithUserAuthenticate configures a UserAuthenticate check-in message handler.
func WithUserAuthenticate(ua service.UserAuthenticate) Option {
	return func(s *Service) {
		s.ua = ua
	}
}

// WithGetToken configures a GetToken check-in message handler.
func WithGetToken(gt service.GetToken) Option {
	return func(s *Service) {
		s.gt = gt
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

// UserAuthenticate Check-in message implementation
func (s *Service) UserAuthenticate(r *mdm.Request, message *mdm.UserAuthenticate) ([]byte, error) {
	if err := s.setupRequest(r, &message.Enrollment); err != nil {
		return nil, err
	}
	if s.ua == nil {
		return nil, errors.New("no UserAuthenticate handler")
	}
	return s.ua.UserAuthenticate(r, message)
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

// GetToken implements the GetToken Check-in message interface.
func (s *Service) GetToken(r *mdm.Request, message *mdm.GetToken) (*mdm.GetTokenResponse, error) {
	if err := s.setupRequest(r, &message.Enrollment); err != nil {
		return nil, err
	}
	ctxlog.Logger(r.Context, s.logger).Info(
		"msg", "GetToken",
		"token_service_type", message.TokenServiceType,
	)
	if s.gt == nil {
		return nil, errors.New("no GetToken handler")
	}
	return s.gt.GetToken(r, message)
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

		logger.Info(
			"msg", "host reported status with invalid command uuid",
			"command_uuid", results.CommandUUID,
			"status", results.Status,
			"error_chain", results.ErrorChain,
		)
	}
	if results.Status != "Idle" {
		// If the host is not idle, we use primary DB since we just wrote results of previous command.
		ctxdb.RequirePrimary(r.Context, true)
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
		// We expand secrets in the command before returning it to the caller so that we never store unencrypted secrets in the database.
		expanded, err := s.store.ExpandEmbeddedSecrets(r.Context, string(cmd.Raw))
		if err != nil {
			// This error is not expected since secrets should have been validated on profile upload.
			logger.Info("level", "error", "msg", "expanding embedded secrets", "err", err)
			// Since this error should not happen, we use the command as is, without expanding secrets.
		} else {
			cmd.Raw = []byte(expanded)
		}
		return cmd, nil
	}
	logger.Debug(
		"msg", "no command retrieved",
	)
	return nil, nil
}
