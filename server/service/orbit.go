package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log/level"

	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
)

type setOrbitNodeKeyer interface {
	setOrbitNodeKey(nodeKey string)
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

func (r EnrollOrbitResponse) error() error { return r.Err }

// hijackRender so we can add a header with the server capabilities in the
// response, allowing Orbit to know what features are available without the
// need to enroll.
func (r EnrollOrbitResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	writeCapabilitiesHeader(w, fleet.GetServerOrbitCapabilities())
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
		return "", fleet.OrbitError{Message: err.Error()}
	}

	orbitNodeKey, err := server.GenerateRandomText(svc.config.Osquery.NodeKeySize)
	if err != nil {
		return "", fleet.OrbitError{Message: "failed to generate orbit node key: " + err.Error()}
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return "", fleet.OrbitError{Message: "app config load failed: " + err.Error()}
	}

	_, err = svc.ds.EnrollOrbit(ctx, appConfig.MDM.EnabledAndConfigured, hostInfo, orbitNodeKey, secret.TeamID)
	if err != nil {
		return "", fleet.OrbitError{Message: "failed to enroll " + err.Error()}
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
	const pendingScriptMaxAge = time.Minute

	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return fleet.OrbitConfig{}, fleet.OrbitError{Message: "internal error: missing host from request context"}
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return fleet.OrbitConfig{}, err
	}

	// set the host's orbit notifications for macOS MDM
	var notifs fleet.OrbitConfigNotifications
	if appConfig.MDM.EnabledAndConfigured && host.IsOsqueryEnrolled() {
		// TODO(mna): all those notifications implied a macos hosts, but none of
		// the checks enforce that (only indirectly in some cases, like
		// IsDEPAssignedToFleet), should we add such a platform check?

		if host.NeedsDEPEnrollment() {
			notifs.RenewEnrollmentProfile = true
		}

		if appConfig.MDM.MacOSMigration.Enable &&
			host.IsEligibleForDEPMigration() {
			notifs.NeedsMDMMigration = true
		}

		if host.DiskEncryptionResetRequested != nil && *host.DiskEncryptionResetRequested {
			notifs.RotateDiskEncryptionKey = true

			// Since this is an user initiated action, we disable
			// the flag when we deliver the notification to Orbit
			if err := svc.ds.SetDiskEncryptionResetStatus(ctx, host.ID, false); err != nil {
				return fleet.OrbitConfig{}, err
			}
		}
	}

	// set the host's orbit notifications for Windows MDM
	if appConfig.MDM.WindowsEnabledAndConfigured {
		if host.IsEligibleForWindowsMDMEnrollment() {
			discoURL, err := microsoft_mdm.ResolveWindowsMDMDiscovery(appConfig.ServerSettings.ServerURL)
			if err != nil {
				return fleet.OrbitConfig{}, err
			}
			notifs.WindowsMDMDiscoveryEndpoint = discoURL
			notifs.NeedsProgrammaticWindowsMDMEnrollment = true
		}
	}
	if config.IsMDMFeatureFlagEnabled() && !appConfig.MDM.WindowsEnabledAndConfigured {
		if host.IsEligibleForWindowsMDMUnenrollment() {
			notifs.NeedsProgrammaticWindowsMDMUnenrollment = true
		}
	}

	// load the pending script executions for that host
	pending, err := svc.ds.ListPendingHostScriptExecutions(ctx, host.ID, pendingScriptMaxAge)
	if err != nil {
		return fleet.OrbitConfig{}, err
	}
	if len(pending) > 0 {
		execIDs := make([]string, 0, len(pending))
		for _, p := range pending {
			execIDs = append(execIDs, p.ExecutionID)
		}
		notifs.PendingScriptExecutionIDs = execIDs
	}

	// team ID is not nil, get team specific flags and options
	if host.TeamID != nil {
		teamAgentOptions, err := svc.ds.TeamAgentOptions(ctx, *host.TeamID)
		if err != nil {
			return fleet.OrbitConfig{}, err
		}

		var opts fleet.AgentOptions
		if teamAgentOptions != nil && len(*teamAgentOptions) > 0 {
			if err := json.Unmarshal(*teamAgentOptions, &opts); err != nil {
				return fleet.OrbitConfig{}, err
			}
		}

		extensionsFiltered, err := svc.filterExtensionsForHost(ctx, opts.Extensions, host)
		if err != nil {
			return fleet.OrbitConfig{}, err
		}

		mdmConfig, err := svc.ds.TeamMDMConfig(ctx, *host.TeamID)
		if err != nil {
			return fleet.OrbitConfig{}, err
		}

		var nudgeConfig *fleet.NudgeConfig
		if appConfig.MDM.EnabledAndConfigured &&
			mdmConfig != nil &&
			mdmConfig.MacOSUpdates.EnabledForHost(host) {
			nudgeConfig, err = fleet.NewNudgeConfig(mdmConfig.MacOSUpdates)
			if err != nil {
				return fleet.OrbitConfig{}, err
			}
		}

		return fleet.OrbitConfig{
			Flags:         opts.CommandLineStartUpFlags,
			Extensions:    extensionsFiltered,
			Notifications: notifs,
			NudgeConfig:   nudgeConfig,
		}, nil
	}

	// team ID is nil, get global flags and options
	var opts fleet.AgentOptions
	if appConfig.AgentOptions != nil {
		if err := json.Unmarshal(*appConfig.AgentOptions, &opts); err != nil {
			return fleet.OrbitConfig{}, err
		}
	}

	extensionsFiltered, err := svc.filterExtensionsForHost(ctx, opts.Extensions, host)
	if err != nil {
		return fleet.OrbitConfig{}, err
	}

	var nudgeConfig *fleet.NudgeConfig
	if appConfig.MDM.EnabledAndConfigured &&
		appConfig.MDM.MacOSUpdates.EnabledForHost(host) {
		nudgeConfig, err = fleet.NewNudgeConfig(appConfig.MDM.MacOSUpdates)
		if err != nil {
			return fleet.OrbitConfig{}, err
		}
	}

	return fleet.OrbitConfig{
		Flags:         opts.CommandLineStartUpFlags,
		Extensions:    extensionsFiltered,
		Notifications: notifs,
		NudgeConfig:   nudgeConfig,
	}, nil
}

// filterExtensionsForHost filters a extensions configuration depending on the host platform and label membership.
//
// If all extensions are filtered, then it returns (nil, nil) (Orbit expects empty extensions if there
// are no extensions for the host.)
func (svc *Service) filterExtensionsForHost(ctx context.Context, extensions json.RawMessage, host *fleet.Host) (json.RawMessage, error) {
	if len(extensions) == 0 {
		return nil, nil
	}
	var extensionsInfo fleet.Extensions
	if err := json.Unmarshal(extensions, &extensionsInfo); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "unmarshal extensions config")
	}

	// Filter the extensions by platform.
	extensionsInfo.FilterByHostPlatform(host.Platform)

	// Filter the extensions by labels (premium only feature).
	if license, _ := license.FromContext(ctx); license != nil && license.IsPremium() {
		for extensionName, extensionInfo := range extensionsInfo {
			hostIsMemberOfAllLabels, err := svc.ds.HostMemberOfAllLabels(ctx, host.ID, extensionInfo.Labels)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "check host labels")
			}
			if hostIsMemberOfAllLabels {
				// Do not filter out, but there's no need to send the label names to the devices.
				extensionInfo.Labels = nil
				extensionsInfo[extensionName] = extensionInfo
			} else {
				delete(extensionsInfo, extensionName)
			}
		}
	}
	// Orbit expects empty message if no extensions apply.
	if len(extensionsInfo) == 0 {
		return nil, nil
	}
	extensionsFiltered, err := json.Marshal(extensionsInfo)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshal extensions config")
	}
	return extensionsFiltered, nil
}

/////////////////////////////////////////////////////////////////////////////////
// Ping orbit endpoint
/////////////////////////////////////////////////////////////////////////////////

type orbitPingRequest struct{}

type orbitPingResponse struct{}

func (r orbitPingResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	writeCapabilitiesHeader(w, fleet.GetServerOrbitCapabilities())
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

/////////////////////////////////////////////////////////////////////////////////
// Get Orbit pending script execution request
/////////////////////////////////////////////////////////////////////////////////

type orbitGetScriptRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
	ExecutionID  string `json:"execution_id"`
}

// interface implementation required by the OrbitClient
func (r *orbitGetScriptRequest) setOrbitNodeKey(nodeKey string) {
	r.OrbitNodeKey = nodeKey
}

// interface implementation required by orbit authentication
func (r *orbitGetScriptRequest) orbitHostNodeKey() string {
	return r.OrbitNodeKey
}

type orbitGetScriptResponse struct {
	Err error `json:"error,omitempty"`
	*fleet.HostScriptResult
}

func (r orbitGetScriptResponse) error() error { return r.Err }

func getOrbitScriptEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*orbitGetScriptRequest)
	script, err := svc.GetHostScript(ctx, req.ExecutionID)
	if err != nil {
		return orbitGetScriptResponse{Err: err}, nil
	}
	return orbitGetScriptResponse{HostScriptResult: script}, nil
}

func (svc *Service) GetHostScript(ctx context.Context, execID string) (*fleet.HostScriptResult, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

/////////////////////////////////////////////////////////////////////////////////
// Post Orbit script execution result
/////////////////////////////////////////////////////////////////////////////////

type orbitPostScriptResultRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
	*fleet.HostScriptResultPayload
}

// interface implementation required by the OrbitClient
func (r *orbitPostScriptResultRequest) setOrbitNodeKey(nodeKey string) {
	r.OrbitNodeKey = nodeKey
}

// interface implementation required by orbit authentication
func (r *orbitPostScriptResultRequest) orbitHostNodeKey() string {
	return r.OrbitNodeKey
}

type orbitPostScriptResultResponse struct {
	Err error `json:"error,omitempty"`
}

func (r orbitPostScriptResultResponse) error() error { return r.Err }

func postOrbitScriptResultEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*orbitPostScriptResultRequest)
	if err := svc.SaveHostScriptResult(ctx, req.HostScriptResultPayload); err != nil {
		return orbitPostScriptResultResponse{Err: err}, nil
	}
	return orbitPostScriptResultResponse{}, nil
}

func (svc *Service) SaveHostScriptResult(ctx context.Context, result *fleet.HostScriptResultPayload) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}
