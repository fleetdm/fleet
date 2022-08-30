package service

import (
	"context"
	"encoding/json"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log/level"
)

type orbitError struct {
	message     string
	nodeInvalid bool
}

func (e orbitError) Error() string {
	return e.message
}

func (e orbitError) NodeInvalid() bool {
	return e.nodeInvalid
}

//type enrollOrbitRequest struct {
//	EnrollSecret string `json:"enroll_secret"`
//	// HardwareUUID is the osquery system implemented one at from: select uuid from system_info
//	HardwareUUID string `json:"hardware_uuid"`
//}

//type enrollOrbitResponse struct {
//	OrbitNodeKey string `json:"orbit_node_key,omitempty"`
//	Err          error  `json:"error,omitempty"`
//}

func (r enrollOrbitResponse) error() error { return r.Err }

func enrollOrbitEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*enrollOrbitRequest)
	nodeKey, err := svc.EnrollOrbit(ctx, req.HardwareUUID, req.EnrollSecret)
	if err != nil {
		return enrollOrbitResponse{Err: err}, nil
	}
	return enrollOrbitResponse{OrbitNodeKey: nodeKey}, nil
}

// return a nodeKey on successful enroll
func (svc *Service) EnrollOrbit(ctx context.Context, hardwareUUID string, enrollSecret string) (string, error) {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)
	logging.WithExtras(ctx, "hardware_uuid", hardwareUUID)
	level.Debug(svc.logger).Log("background", "before verify secret")

	secret, err := svc.ds.VerifyEnrollSecret(ctx, enrollSecret)
	if err != nil {
		return "", orbitError{
			message:     "orbit enroll failed: " + err.Error(),
			nodeInvalid: true,
		}
	}
	_ = secret

	level.Debug(svc.logger).Log("background", "after verify secret")
	orbitNodeKey, err := server.GenerateRandomText(svc.config.Osquery.NodeKeySize)
	if err != nil {
		return "", orbitError{
			message:     "failed to generate orbit node key: " + err.Error(),
			nodeInvalid: true,
		}
	}

	host, err := svc.ds.EnrollOrbit(ctx, hardwareUUID, orbitNodeKey)
	if err != nil {
		return "", orbitError{
			message:     "failed to enroll " + err.Error(),
			nodeInvalid: true,
		}
	}
	_ = host

	level.Debug(svc.logger).Log("background", "after enroll")

	return orbitNodeKey, nil
}

func getOrbitFlagsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*orbitRequest)
	opts, err := svc.GetOrbitFlags(ctx, req.OrbitNodeKey)
	if err != nil {
		return enrollOrbitResponse{Err: err}, nil
	}
	return opts, nil
}

func (svc *Service) GetOrbitFlags(ctx context.Context, orbitNodeKey string) (json.RawMessage, error) {
	//svc.authz.IsAuthenticatedWith()
	svc.authz.SkipAuthorization(ctx)

	level.Debug(svc.logger).Log("Before Loading Host")

	host, err := svc.ds.LoadHostByOrbitNodeKey(ctx, orbitNodeKey)
	if err != nil {
		return nil, orbitError{
			message:     "failed to find host by orbit node key: " + orbitNodeKey,
			nodeInvalid: true,
		}
	}

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

	//j, err := json.Marshal(&config.AgentOptions)
	//if err != nil {
	//	return "", nil
	//}
	//return string(j), nil

	//opts, err := svc.ds.TeamAgentOptions(ctx, *host.TeamID)
	//if err != nil {
	//	return "", nil
	//}
	//j, err := json.Marshal(&opts)
	//s := string(j)
	//level.Debug(svc.logger).Log("Get FLAGS ====", s)
	//return string(j), nil
}
