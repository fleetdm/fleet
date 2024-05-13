package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/kit/log/level"
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
	// OsqueryIdentifier holds the identifier used by osquery.
	// If not set, then the hardware UUID is used to match orbit and osquery.
	OsqueryIdentifier string `json:"osquery_identifier"`
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
		HardwareUUID:      req.HardwareUUID,
		HardwareSerial:    req.HardwareSerial,
		Hostname:          req.Hostname,
		Platform:          req.Platform,
		OsqueryIdentifier: req.OsqueryIdentifier,
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
			"osquery_identifier", hostInfo.OsqueryIdentifier,
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
	if !appConfig.MDM.WindowsEnabledAndConfigured {
		if host.IsEligibleForWindowsMDMUnenrollment() {
			notifs.NeedsProgrammaticWindowsMDMUnenrollment = true
		}
	}

	// load the pending script executions for that host
	if !appConfig.ServerSettings.ScriptsDisabled {
		pending, err := svc.ds.ListPendingHostScriptExecutions(ctx, host.ID)
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
	}

	pendingInstalls, err := svc.ds.ListPendingSoftwareInstalls(ctx, host.ID)
	if err != nil {
		return fleet.OrbitConfig{}, err
	}
	if len(pendingInstalls) > 0 {
		notifs.PendingSoftwareInstallerIDs = pendingInstalls
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
			hostOS, err := svc.ds.GetHostOperatingSystem(ctx, host.ID)
			if errors.Is(err, sql.ErrNoRows) {
				// host os has not been collected yet (no details query)
				hostOS = &fleet.OperatingSystem{}
			} else if err != nil {
				return fleet.OrbitConfig{}, err
			}
			requiresNudge, err := hostOS.RequiresNudge()
			if err != nil {
				return fleet.OrbitConfig{}, err
			}

			if requiresNudge {
				nudgeConfig, err = fleet.NewNudgeConfig(mdmConfig.MacOSUpdates)
				if err != nil {
					return fleet.OrbitConfig{}, err
				}
			}
		}

		if mdmConfig.EnableDiskEncryption &&
			host.IsEligibleForBitLockerEncryption() {
			notifs.EnforceBitLockerEncryption = true
		}

		var updateChannels *fleet.OrbitUpdateChannels
		if len(opts.UpdateChannels) > 0 {
			var uc fleet.OrbitUpdateChannels
			if err := json.Unmarshal(opts.UpdateChannels, &uc); err != nil {
				return fleet.OrbitConfig{}, err
			}
			updateChannels = &uc
		}

		return fleet.OrbitConfig{
			Flags:          opts.CommandLineStartUpFlags,
			Extensions:     extensionsFiltered,
			Notifications:  notifs,
			NudgeConfig:    nudgeConfig,
			UpdateChannels: updateChannels,
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
		hostOS, err := svc.ds.GetHostOperatingSystem(ctx, host.ID)
		if errors.Is(err, sql.ErrNoRows) {
			// host os has not been collected yet (no details query)
			hostOS = &fleet.OperatingSystem{}
		} else if err != nil {
			return fleet.OrbitConfig{}, err
		}
		requiresNudge, err := hostOS.RequiresNudge()
		if err != nil {
			return fleet.OrbitConfig{}, err
		}

		if requiresNudge {
			nudgeConfig, err = fleet.NewNudgeConfig(appConfig.MDM.MacOSUpdates)
			if err != nil {
				return fleet.OrbitConfig{}, err
			}
		}
	}

	if appConfig.MDM.WindowsEnabledAndConfigured &&
		appConfig.MDM.EnableDiskEncryption.Value &&
		host.IsEligibleForBitLockerEncryption() {
		notifs.EnforceBitLockerEncryption = true
	}

	var updateChannels *fleet.OrbitUpdateChannels
	if len(opts.UpdateChannels) > 0 {
		var uc fleet.OrbitUpdateChannels
		if err := json.Unmarshal(opts.UpdateChannels, &uc); err != nil {
			return fleet.OrbitConfig{}, err
		}
		updateChannels = &uc
	}

	return fleet.OrbitConfig{
		Flags:          opts.CommandLineStartUpFlags,
		Extensions:     extensionsFiltered,
		Notifications:  notifs,
		NudgeConfig:    nudgeConfig,
		UpdateChannels: updateChannels,
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

	if len(deviceAuthToken) == 0 {
		return badRequest("device auth token cannot be empty")
	}

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return newOsqueryError("internal error: missing host from request context")
	}

	if err := svc.ds.SetOrUpdateDeviceAuthToken(ctx, host.ID, deviceAuthToken); err != nil {
		if errors.As(err, &fleet.ConflictError{}) {
			return err
		}
		return newOsqueryError(fmt.Sprintf("internal error: failed to set or update device auth token: %s", err))
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
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, fleet.OrbitError{Message: "internal error: missing host from request context"}
	}

	// get the script's details
	script, err := svc.ds.GetHostScriptExecutionResult(ctx, execID)
	if err != nil {
		return nil, err
	}
	// ensure it cannot get access to a different host's script
	if script.HostID != host.ID {
		return nil, ctxerr.Wrap(ctx, newNotFoundError(), "no script found for this host")
	}
	return script, nil
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
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return fleet.OrbitError{Message: "internal error: missing host from request context"}
	}
	if result == nil {
		return ctxerr.Wrap(ctx, &fleet.BadRequestError{Message: "missing script result"}, "save host script result")
	}

	// always use the authenticated host's ID as host_id
	result.HostID = host.ID
	hsr, err := svc.ds.SetHostScriptExecutionResult(ctx, result)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "save host script result")
	}

	if hsr != nil {
		var user *fleet.User
		if hsr.UserID != nil {
			user, err = svc.ds.UserByID(ctx, *hsr.UserID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "get host script execution user")
			}
		}
		var scriptName string
		if hsr.ScriptID != nil {
			scr, err := svc.ds.Script(ctx, *hsr.ScriptID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "get saved script")
			}
			scriptName = scr.Name
		}

		// TODO(sarah): We may need to special case lock/unlock script results here?
		if err := svc.ds.NewActivity(
			ctx,
			user,
			fleet.ActivityTypeRanScript{
				HostID:            host.ID,
				HostDisplayName:   host.DisplayName(),
				ScriptExecutionID: hsr.ExecutionID,
				ScriptName:        scriptName,
				Async:             !hsr.SyncRequest,
			},
		); err != nil {
			return ctxerr.Wrap(ctx, err, "create activity for script execution request")
		}
	}
	return nil
}

/////////////////////////////////////////////////////////////////////////////////
// Post Orbit device mapping (custom email)
/////////////////////////////////////////////////////////////////////////////////

type orbitPutDeviceMappingRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
	Email        string `json:"email"`
}

// interface implementation required by the OrbitClient
func (r *orbitPutDeviceMappingRequest) setOrbitNodeKey(nodeKey string) {
	r.OrbitNodeKey = nodeKey
}

// interface implementation required by orbit authentication
func (r *orbitPutDeviceMappingRequest) orbitHostNodeKey() string {
	return r.OrbitNodeKey
}

type orbitPutDeviceMappingResponse struct {
	Err error `json:"error,omitempty"`
}

func (r orbitPutDeviceMappingResponse) error() error { return r.Err }

func putOrbitDeviceMappingEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*orbitPutDeviceMappingRequest)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := newOsqueryError("internal error: missing host from request context")
		return orbitPutDeviceMappingResponse{Err: err}, nil
	}

	_, err := svc.SetCustomHostDeviceMapping(ctx, host.ID, req.Email)
	return orbitPutDeviceMappingResponse{Err: err}, nil
}

/////////////////////////////////////////////////////////////////////////////////
// Post Orbit disk encryption key
/////////////////////////////////////////////////////////////////////////////////

type orbitPostDiskEncryptionKeyRequest struct {
	OrbitNodeKey  string `json:"orbit_node_key"`
	EncryptionKey []byte `json:"encryption_key"`
	ClientError   string `json:"client_error"`
}

// interface implementation required by the OrbitClient
func (r *orbitPostDiskEncryptionKeyRequest) setOrbitNodeKey(nodeKey string) {
	r.OrbitNodeKey = nodeKey
}

// interface implementation required by orbit authentication
func (r *orbitPostDiskEncryptionKeyRequest) orbitHostNodeKey() string {
	return r.OrbitNodeKey
}

type orbitPostDiskEncryptionKeyResponse struct {
	Err error `json:"error,omitempty"`
}

func (r orbitPostDiskEncryptionKeyResponse) error() error { return r.Err }
func (r orbitPostDiskEncryptionKeyResponse) Status() int  { return http.StatusNoContent }

func postOrbitDiskEncryptionKeyEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*orbitPostDiskEncryptionKeyRequest)
	if err := svc.SetOrUpdateDiskEncryptionKey(ctx, string(req.EncryptionKey), req.ClientError); err != nil {
		return orbitPostDiskEncryptionKeyResponse{Err: err}, nil
	}
	return orbitPostDiskEncryptionKeyResponse{}, nil
}

func (svc *Service) SetOrUpdateDiskEncryptionKey(ctx context.Context, encryptionKey, clientError string) error {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return newOsqueryError("internal error: missing host from request context")
	}
	if !host.MDMInfo.IsFleetEnrolled() {
		return badRequest("host is not enrolled with fleet")
	}

	var (
		encryptedEncryptionKey string
		decryptable            *bool
	)

	// only set the encryption key if there was no client error
	if clientError == "" && encryptionKey != "" {
		wstepCert, _, _, err := svc.config.MDM.MicrosoftWSTEP()
		if err != nil {
			// should never return an error because the WSTEP is first parsed and
			// cached at the start of the fleet serve process.
			return ctxerr.Wrap(ctx, err, "get WSTEP certificate")
		}
		enc, err := microsoft_mdm.Encrypt(encryptionKey, wstepCert.Leaf)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "encrypt the key with WSTEP certificate")
		}
		encryptedEncryptionKey = enc
		decryptable = ptr.Bool(true)
	}

	if err := svc.ds.SetOrUpdateHostDiskEncryptionKey(ctx, host.ID, encryptedEncryptionKey, clientError, decryptable); err != nil {
		return ctxerr.Wrap(ctx, err, "set or update disk encryption key")
	}

	return nil
}

/////////////////////////////////////////////////////////////////////////////////
// Get Orbit pending software installations
/////////////////////////////////////////////////////////////////////////////////

type orbitGetSoftwareInstallRequest struct {
	OrbitNodeKey string `json:"orbot_node_key"`
	InstallUUID  string `json:"install_uuid"`
}

// interface implementation required by the OrbitClient
func (r *orbitGetSoftwareInstallRequest) setOrbitNodeKey(nodeKey string) {
	r.OrbitNodeKey = nodeKey
}

// interface implementation required by the OrbitClient
func (r *orbitGetSoftwareInstallRequest) orbitHostNodeKey() string {
	return r.OrbitNodeKey
}

type orbitGetSoftwareInstallResponse struct {
	Err error `json:"error,omitempty"`
	*fleet.SoftwareInstallDetails
}

func (r orbitGetSoftwareInstallResponse) error() error { return r.Err }

func getOrbitSoftwareInstallDetails(ctx context.Context, request any, svc fleet.Service) (errorer, error) {
	req := request.(*orbitGetSoftwareInstallRequest)
	details, err := svc.GetSoftwareInstallDetails(ctx, req.InstallUUID)
	if err != nil {
		return orbitGetSoftwareInstallResponse{Err: err}, nil
	}

	return orbitGetSoftwareInstallResponse{SoftwareInstallDetails: details}, nil
}

func (svc *Service) GetSoftwareInstallDetails(ctx context.Context, installUUID string) (*fleet.SoftwareInstallDetails, error) {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, fleet.OrbitError{Message: "internal error: missing host from request context"}
	}

	details, err := svc.ds.GetSoftwareInstallDetails(ctx, installUUID)
	if err != nil {
		return nil, err
	}

	// ensure it cannot get access to a different host's installers
	if details.HostID != host.ID {
		return nil, ctxerr.Wrap(ctx, newNotFoundError(), "no installer found for this host")
	}
	return details, nil
}

// Download Orbit software installer request
/////////////////////////////////////////////////////////////////////////////////

type orbitDownloadSoftwareInstallerRequest struct {
	Alt          string `query:"alt"`
	OrbitNodeKey string `json:"orbit_node_key"`
	InstallerID  uint   `json:"installer_id"`
}

// interface implementation required by the OrbitClient
func (r *orbitDownloadSoftwareInstallerRequest) setOrbitNodeKey(nodeKey string) {
	r.OrbitNodeKey = nodeKey
}

// interface implementation required by orbit authentication
func (r *orbitDownloadSoftwareInstallerRequest) orbitHostNodeKey() string {
	return r.OrbitNodeKey
}

func orbitDownloadSoftwareInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*orbitDownloadSoftwareInstallerRequest)

	downloadRequested := req.Alt == "media"
	if !downloadRequested {
		// TODO: confirm error handling
		return orbitDownloadSoftwareInstallerResponse{Err: &fleet.BadRequestError{Message: "only alt=media is supported"}}, nil
	}

	p, err := svc.OrbitDownloadSoftwareInstaller(ctx, req.InstallerID)
	if err != nil {
		return orbitDownloadSoftwareInstallerResponse{Err: err}, nil
	}
	return orbitDownloadSoftwareInstallerResponse{payload: p}, nil
}

func (svc *Service) OrbitDownloadSoftwareInstaller(ctx context.Context, installerID uint) (*fleet.DownloadSoftwareInstallerPayload, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

/////////////////////////////////////////////////////////////////////////////////
// Post Orbit software install result
/////////////////////////////////////////////////////////////////////////////////

type orbitPostSoftwareInstallResultRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
	*fleet.HostSoftwareInstallResultPayload
}

// interface implementation required by the OrbitClient
func (r *orbitPostSoftwareInstallResultRequest) setOrbitNodeKey(nodeKey string) {
	r.OrbitNodeKey = nodeKey
}

func (r *orbitPostSoftwareInstallResultRequest) orbitHostNodeKey() string {
	return r.OrbitNodeKey
}

type orbitPostSoftwareInstallResultResponse struct {
	Err error `json:"error,omitempty"`
}

func (r orbitPostSoftwareInstallResultResponse) error() error { return r.Err }
func (r orbitPostSoftwareInstallResultResponse) Status() int  { return http.StatusNoContent }

func postOrbitSoftwareInstallResultEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*orbitPostSoftwareInstallResultRequest)
	if err := svc.SaveHostSoftwareInstallResult(ctx, req.HostSoftwareInstallResultPayload); err != nil {
		return orbitPostSoftwareInstallResultResponse{Err: err}, nil
	}
	return orbitPostSoftwareInstallResultResponse{}, nil
}

func (svc *Service) SaveHostSoftwareInstallResult(ctx context.Context, result *fleet.HostSoftwareInstallResultPayload) error {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return newOsqueryError("internal error: missing host from request context")
	}

	// always use the authenticated host's ID as host_id
	result.HostID = host.ID
	if err := svc.ds.SetHostSoftwareInstallResult(ctx, result); err != nil {
		return ctxerr.Wrap(ctx, err, "save host software installation result")
	}

	if status := result.Status(); status != fleet.SoftwareInstallerPending {
		hsi, err := svc.ds.GetSoftwareInstallResults(ctx, result.InstallUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get host software installation result information")
		}

		var user *fleet.User
		if hsi.UserID != nil {
			user, err = svc.ds.UserByID(ctx, *hsi.UserID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "get host software installation user")
			}
		}

		if err := svc.ds.NewActivity(
			ctx,
			user,
			fleet.ActivityTypeInstalledSoftware{
				HostID:          host.ID,
				HostDisplayName: host.DisplayName(),
				SoftwareTitle:   hsi.SoftwareTitle,
				InstallUUID:     result.InstallUUID,
				Status:          string(status),
			},
		); err != nil {
			return ctxerr.Wrap(ctx, err, "create activity for software installation")
		}
	}
	return nil
}
