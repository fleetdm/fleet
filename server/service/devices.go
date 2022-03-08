package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

/////////////////////////////////////////////////////////////////////////////////
// Get Current Device's Host
/////////////////////////////////////////////////////////////////////////////////

type getDeviceHostRequest struct {
	Token string `url:"token"`
}

func (r *getDeviceHostRequest) deviceAuthToken() string {
	return r.Token
}

func getDeviceHostEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getDeviceHostRequest)
	/*
		host, err := svc.GetHost(ctx, req.ID)
		if err != nil {
			return getHostResponse{Err: err}, nil
		}

		resp, err := hostDetailResponseForHost(ctx, svc, host)
		if err != nil {
			return getHostResponse{Err: err}, nil
		}
	*/

	_ = req
	return getHostResponse{Host: nil}, nil
}
