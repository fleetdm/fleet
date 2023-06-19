package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	windows_mdm "github.com/fleetdm/fleet/v4/server/mdm/windows"
	"github.com/go-kit/kit/log/level"
)

type setOrbitNodeKeyer interface {
	setOrbitNodeKey(nodeKey string)
}

type orbitError struct {
	message string
}

// EnrollOrbitRequest is the request Orbit instances use to enroll to Fleet.
type EnrollOrbitRequest struct {
	// EnrollSecret is the secret to authenticate the enroll request.
	EnrollSecret string `json:"enroll_secret"`
	// HardwareUUID is the device's hardware UUID.
	HardwareUUID string `json:"hardware_uuid"`
	// HardwareSerial is the device's serial number.
	HardwareSerial string `json:"hardware_serial"`
	// Hostname is the device's hostname.
	Hostname string `json:"hostname"`
	// Platform is the device's platform as defined by osquery.
	Platform string `json:"platform"`
}

type EnrollOrbitResponse struct {
	OrbitNodeKey string `json:"orbit_node_key,omitempty"`
	Err          error  `json:"error,omitempty"`
}

type orbitGetConfigRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
}

func (r *orbitGetConfigRequest) setOrbitNodeKey(nodeKey string) {
	r.OrbitNodeKey = nodeKey
}

func (r *orbitGetConfigRequest) orbitHostNodeKey() string {
	return r.OrbitNodeKey
}

type orbitGetConfigResponse struct {
	fleet.OrbitConfig
	Err error `json:"error,omitempty"`
}

func (r orbitGetConfigResponse) error() error { return r.Err }

func (e orbitError) Error() string {
	return e.message
}

func (r EnrollOrbitResponse) error() error { return r.Err }

// hijackRender so we can add a header with the server capabilities in the
// response, allowing Orbit to know what features are available without the
// need to enroll.
func (r EnrollOrbitResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	writeCapabilitiesHeader(w, fleet.ServerOrbitCapabilities)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	if err := enc.Encode(r); err != nil {
		encodeError(ctx, newOsqueryError(fmt.Sprintf("orbit enroll failed: %s", err)), w)
	}
}

func enrollOrbitEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*EnrollOrbitRequest)
	nodeKey, err := svc.EnrollOrbit(ctx, fleet.OrbitHostInfo{
		HardwareUUID:   req.HardwareUUID,
		HardwareSerial: req.HardwareSerial,
		Hostname:       req.Hostname,
		Platform:       req.Platform,
	}, req.EnrollSecret)
	if err != nil {
		return EnrollOrbitResponse{Err: err}, nil
	}
	return EnrollOrbitResponse{OrbitNodeKey: nodeKey}, nil
}

func (svc *Service) AuthenticateOrbitHost(ctx context.Context, orbitNodeKey string) (*fleet.Host, bool, error) {
	svc.authz.SkipAuthorization(ctx)

	if orbitNodeKey == "" {
		return nil, false, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("authentication error: missing orbit node key"))
	}

	host, err := svc.ds.LoadHostByOrbitNodeKey(ctx, orbitNodeKey)
	switch {
	case err == nil:
		// OK
	case fleet.IsNotFound(err):
		return nil, false, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("authentication error: invalid orbit node key"))
	default:
		return nil, false, ctxerr.Wrap(ctx, err, "authentication error orbit")
	}

	return host, svc.debugEnabledForHost(ctx, host.ID), nil
}

// EnrollOrbit enrolls an Orbit instance to Fleet and returns the orbit node key.
func (svc *Service) EnrollOrbit(ctx context.Context, hostInfo fleet.OrbitHostInfo, enrollSecret string) (string, error) {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	logging.WithLevel(
		logging.WithExtras(ctx,
			"hardware_uuid", hostInfo.HardwareUUID,
			"hardware_serial", hostInfo.HardwareSerial,
			"hostname", hostInfo.Hostname,
			"platform", hostInfo.Platform,
		),
		level.Info,
	)

	secret, err := svc.ds.VerifyEnrollSecret(ctx, enrollSecret)
	if err != nil {
		if fleet.IsNotFound(err) {
			// OK - This can happen if the following sequence of events take place:
			// 	1. User deletes global/team enroll secret.
			// 	2. User deletes the host in Fleet.
			// 	3. Orbit tries to re-enroll using old secret.
			return "", fleet.NewAuthFailedError("invalid secret")
		}
		return "", orbitError{message: err.Error()}
	}

	orbitNodeKey, err := server.GenerateRandomText(svc.config.Osquery.NodeKeySize)
	if err != nil {
		return "", orbitError{message: "failed to generate orbit node key: " + err.Error()}
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return "", orbitError{message: "app config load failed: " + err.Error()}
	}

	_, err = svc.ds.EnrollOrbit(ctx, appConfig.MDM.EnabledAndConfigured, hostInfo, orbitNodeKey, secret.TeamID)
	if err != nil {
		return "", orbitError{message: "failed to enroll " + err.Error()}
	}

	return orbitNodeKey, nil
}

func getOrbitConfigEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	cfg, err := svc.GetOrbitConfig(ctx)
	if err != nil {
		return orbitGetConfigResponse{Err: err}, nil
	}
	return orbitGetConfigResponse{OrbitConfig: cfg}, nil
}

func (svc *Service) GetOrbitConfig(ctx context.Context) (fleet.OrbitConfig, error) {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	var notifs fleet.OrbitConfigNotifications

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return fleet.OrbitConfig{Notifications: notifs}, orbitError{message: "internal error: missing host from request context"}
	}

	config, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return fleet.OrbitConfig{Notifications: notifs}, err
	}

	// set the host's orbit notifications for macOS MDM
	if config.MDM.EnabledAndConfigured && host.IsOsqueryEnrolled() {
		if host.NeedsDEPEnrollment() {
			notifs.RenewEnrollmentProfile = true
		}

		if config.MDM.MacOSMigration.Enable &&
			host.IsElegibleForDEPMigration() {
			notifs.NeedsMDMMigration = true
		}

		if host.DiskEncryptionResetRequested != nil && *host.DiskEncryptionResetRequested {
			notifs.RotateDiskEncryptionKey = true

			// Since this is an user initiated action, we disable
			// the flag when we deliver the notification to Orbit
			if err := svc.ds.SetDiskEncryptionResetStatus(ctx, host.ID, false); err != nil {
				return fleet.OrbitConfig{Notifications: notifs}, err
			}
		}
	}

	// set the host's orbit notifications for Windows MDM
	if config.MDM.WindowsEnabledAndConfigured {
		if host.IsElegibleForWindowsMDMEnrollment() {
			discoURL, err := windows_mdm.ResolveWindowsMDMDiscovery(config.ServerSettings.ServerURL)
			if err != nil {
				return fleet.OrbitConfig{Notifications: notifs}, err
			}
			notifs.WindowsMDMDiscoveryEndpoint = discoURL
			notifs.NeedsProgrammaticWindowsMDMEnrollment = true
		}
	}

	// team ID is not nil, get team specific flags and options
	if host.TeamID != nil {
		teamAgentOptions, err := svc.ds.TeamAgentOptions(ctx, *host.TeamID)
		if err != nil {
			return fleet.OrbitConfig{Notifications: notifs}, err
		}

		var opts fleet.AgentOptions
		if teamAgentOptions != nil && len(*teamAgentOptions) > 0 {
			if err := json.Unmarshal(*teamAgentOptions, &opts); err != nil {
				return fleet.OrbitConfig{Notifications: notifs}, err
			}
		}

		mdmConfig, err := svc.ds.TeamMDMConfig(ctx, *host.TeamID)
		if err != nil {
			return fleet.OrbitConfig{Notifications: notifs}, err
		}

		var nudgeConfig *fleet.NudgeConfig
		if mdmConfig != nil &&
			mdmConfig.MacOSUpdates.Deadline.Value != "" &&
			mdmConfig.MacOSUpdates.MinimumVersion.Value != "" {
			nudgeConfig, err = fleet.NewNudgeConfig(mdmConfig.MacOSUpdates)
			if err != nil {
				return fleet.OrbitConfig{Notifications: notifs}, err
			}
		}

		return fleet.OrbitConfig{
			Flags:         opts.CommandLineStartUpFlags,
			Extensions:    opts.Extensions,
			Notifications: notifs,
			NudgeConfig:   nudgeConfig,
		}, nil
	}

	// team ID is nil, get global flags and options
	var opts fleet.AgentOptions
	if config.AgentOptions != nil {
		if err := json.Unmarshal(*config.AgentOptions, &opts); err != nil {
			return fleet.OrbitConfig{Notifications: notifs}, err
		}
	}

	var nudgeConfig *fleet.NudgeConfig
	if config.MDM.MacOSUpdates.Deadline.Value != "" &&
		config.MDM.MacOSUpdates.MinimumVersion.Value != "" {
		nudgeConfig, err = fleet.NewNudgeConfig(config.MDM.MacOSUpdates)
		if err != nil {
			return fleet.OrbitConfig{Notifications: notifs}, err
		}
	}

	return fleet.OrbitConfig{
		Flags:         opts.CommandLineStartUpFlags,
		Extensions:    opts.Extensions,
		Notifications: notifs,
		NudgeConfig:   nudgeConfig,
	}, nil
}

/////////////////////////////////////////////////////////////////////////////////
// Ping orbit endpoint
/////////////////////////////////////////////////////////////////////////////////

type orbitPingRequest struct{}

type orbitPingResponse struct{}

func (r orbitPingResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	writeCapabilitiesHeader(w, fleet.ServerOrbitCapabilities)
}

func (r orbitPingResponse) error() error { return nil }

// NOTE: we're intentionally not reading the capabilities header in this
// endpoint as is unauthenticated and we don't want to trust whatever comes in
// there.
func orbitPingEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	svc.DisableAuthForPing(ctx)
	return orbitPingResponse{}, nil
}

/////////////////////////////////////////////////////////////////////////////////
// SetOrUpdateDeviceToken endpoint
/////////////////////////////////////////////////////////////////////////////////

type setOrUpdateDeviceTokenRequest struct {
	OrbitNodeKey    string `json:"orbit_node_key"`
	DeviceAuthToken string `json:"device_auth_token"`
}

func (r *setOrUpdateDeviceTokenRequest) setOrbitNodeKey(nodeKey string) {
	r.OrbitNodeKey = nodeKey
}

func (r *setOrUpdateDeviceTokenRequest) orbitHostNodeKey() string {
	return r.OrbitNodeKey
}

type setOrUpdateDeviceTokenResponse struct {
	Err error `json:"error,omitempty"`
}

func (r setOrUpdateDeviceTokenResponse) error() error { return r.Err }

func setOrUpdateDeviceTokenEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*setOrUpdateDeviceTokenRequest)
	if err := svc.SetOrUpdateDeviceAuthToken(ctx, req.DeviceAuthToken); err != nil {
		return setOrUpdateDeviceTokenResponse{Err: err}, nil
	}
	return setOrUpdateDeviceTokenResponse{}, nil
}

func (svc *Service) SetOrUpdateDeviceAuthToken(ctx context.Context, deviceAuthToken string) error {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return newOsqueryError("internal error: missing host from request context")
	}

	if err := svc.ds.SetOrUpdateDeviceAuthToken(ctx, host.ID, deviceAuthToken); err != nil {
		return newOsqueryError(fmt.Sprintf("internal error: failed to set or update device auth token: %e", err))
	}

	return nil
}
