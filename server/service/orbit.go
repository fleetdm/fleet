package service

import (
	"context"
	"encoding/json"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type orbitError struct {
	message string
}

type enrollOrbitRequest struct {
	EnrollSecret string `json:"enroll_secret"`
	HardwareUUID string `json:"hardware_uuid"`
}

type enrollOrbitResponse struct {
	OrbitNodeKey string `json:"orbit_node_key,omitempty"`
	Err          error  `json:"error,omitempty"`
}

type orbitGetConfigRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
}

func (r *orbitGetConfigRequest) orbitHostNodeKey() string {
	return r.OrbitNodeKey
}

type orbitGetConfigResponse struct {
	Flags json.RawMessage `json:"command_line_startup_flags,omitempty"`
	Err   error           `json:"error,omitempty"`
}

func (e orbitError) Error() string {
	return e.message
}

func (r enrollOrbitResponse) error() error { return r.Err }

func enrollOrbitEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*enrollOrbitRequest)
	nodeKey, err := svc.EnrollOrbit(ctx, req.HardwareUUID, req.EnrollSecret)
	if err != nil {
		return enrollOrbitResponse{Err: err}, nil
	}
	return enrollOrbitResponse{OrbitNodeKey: nodeKey}, nil
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

// EnrollOrbit returns an orbit nodeKey on successful enroll
func (svc *Service) EnrollOrbit(ctx context.Context, hardwareUUID string, enrollSecret string) (string, error) {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)
	logging.WithExtras(ctx, "hardware_uuid", hardwareUUID)

	secret, err := svc.ds.VerifyEnrollSecret(ctx, enrollSecret)
	if err != nil {
		return "", orbitError{message: "orbit enroll failed: " + err.Error()}
	}

	orbitNodeKey, err := server.GenerateRandomText(svc.config.Osquery.NodeKeySize)
	if err != nil {
		return "", orbitError{message: "failed to generate orbit node key: " + err.Error()}
	}

	_, err = svc.ds.EnrollOrbit(ctx, hardwareUUID, orbitNodeKey, secret.TeamID)
	if err != nil {
		return "", orbitError{message: "failed to enroll " + err.Error()}
	}

	return orbitNodeKey, nil
}

func getOrbitConfigEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	opts, err := svc.GetOrbitFlags(ctx)
	if err != nil {
		return orbitGetConfigResponse{Err: err}, nil
	}
	return orbitGetConfigResponse{Flags: opts}, nil
}

func (svc *Service) GetOrbitFlags(ctx context.Context) (json.RawMessage, error) {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, orbitError{message: "internal error: missing host from request context"}
	}

	// team ID is not nil, get team specific flags and options
	if host.TeamID != nil {
		teamAgentOptions, err := svc.ds.TeamAgentOptions(ctx, *host.TeamID)
		if err != nil {
			return nil, err
		}

		if teamAgentOptions != nil && len(*teamAgentOptions) > 0 {
			var opts fleet.AgentOptions
			if err := json.Unmarshal(*teamAgentOptions, &opts); err != nil {
				return nil, err
			}
			return opts.CommandLineStartUpFlags, nil
		}
	}

	// team ID is nil, get global flags and options
	config, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, err
	}
	var opts fleet.AgentOptions
	if config.AgentOptions != nil {
		if err := json.Unmarshal(*config.AgentOptions, &opts); err != nil {
			return nil, err
		}
	}
	return opts.CommandLineStartUpFlags, nil
}
