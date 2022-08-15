package service

import (
	"context"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type orbitError struct {
	message     string
	nodeInvalid bool
}

func (e orbitError) Error() string {
	return e.message
}

type enrollOrbitRequest struct {
	EnrollSecret string `json:"enroll_secret"`
	// HardwareUUID is the osquery system implemented one at from: select uuid from system_info
	HardwareUUID string `json:"hardware_uuid"`
}

type enrollOrbitResponse struct {
	OrbitNodeKey string `json:"orbit_node_key,omitempty"`
	Err          error  `json:"error,omitempty"`
}

func enrollOrbitEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*enrollOrbitRequest)
	nodeKey, err := svc.EnrollOrbit(ctx, req.HardwareUUID, req.EnrollSecret)
	if err != nil {
		return enrollOrbitResponse{Err: err}, nil
	}
	return enrollOrbitResponse{OrbitNodeKey: nodeKey}, nil
}

// return a nodeKey, error on successful enroll
func (svc *Service) EnrollOrbit(ctx context.Context, hardwareUUID string, enrollSecret string) (string, error) {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	secret, err := svc.ds.VerifyEnrollSecret(ctx, enrollSecret)
	_ = secret
	if err != nil {
		return "", orbitError{
			message:     "orbit enroll failed: " + err.Error(),
			nodeInvalid: true,
		}
	}

	orbitNodeKey, err := server.GenerateRandomText(svc.config.Osquery.NodeKeySize)
	if err != nil {
		return "", orbitError{
			message:     "failed to generate orbit node key: " + err.Error(),
			nodeInvalid: true,
		}
	}

	// now we have the orbitNodeKey
	// check if hardwareUUID exists in `hosts` table
	// select uuid from hosts
	// if uuid == hardwareUUID

	host, err := svc.ds.EnrollOrbit(ctx, hardwareUUID, orbitNodeKey)
	if err != nil {
		return "", orbitError{
			message:     "failed to enroll " + err.Error(),
			nodeInvalid: true,
		}
	}
	_ = host

	//host, err := svc.ds.HostByIdentifier(ctx, hardwareUUID)
	//if err != nil {
	//	return "", orbitError{
	//		message:     "failed to generate orbit node key: " + err.Error(),
	//		nodeInvalid: true,
	//	}
	//}
	//
	//// extra sure that it's the same one
	//if host.UUID != hardwareUUID {
	//	return "", nil
	//}

	return orbitNodeKey, nil
}
