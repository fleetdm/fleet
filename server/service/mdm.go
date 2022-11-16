package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type getAppleMDMResponse struct {
	*fleet.AppleMDM
	Err error `json:"error,omitempty"`
}

func (r getAppleMDMResponse) error() error { return r.Err }

func getAppleMDMEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	appleMDM, err := svc.GetAppleMDM(ctx)
	if err != nil {
		return getAppleMDMResponse{Err: err}, nil
	}

	return getAppleMDMResponse{AppleMDM: appleMDM}, nil
}

func (svc *Service) GetAppleMDM(ctx context.Context) (*fleet.AppleMDM, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppleMDM{}, fleet.ActionRead); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	// if there is no apple mdm config, fail with a 404
	if svc.config.MDM.AppleAPNsCert == "" {
		return nil, notFoundError{}
	}

	apns, err := svc.config.MDM.AppleAPNs()
	if err != nil {
		return nil, err
	}

	appleMDM := &fleet.AppleMDM{
		CommonName: apns.Leaf.Subject.CommonName,
		Issuer:     apns.Leaf.Issuer.CommonName,
		RenewDate:  apns.Leaf.NotAfter,
	}
	if apns.Leaf.SerialNumber != nil {
		appleMDM.SerialNumber = apns.Leaf.SerialNumber.String()
	}

	return appleMDM, nil
}
