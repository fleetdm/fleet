package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type putSetupExperienceSoftwareRequest struct {
	TeamID      uint   `query:"team_id"`
	SoftwareIDs []uint `json:"software_ids"`
}

type putSetupExperienceSoftwareResponse struct {
	Err error `json:"error,omitempty"`
}

func (r putSetupExperienceSoftwareResponse) error() error { return r.Err }

func putSetupExperienceSoftware(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*putSetupExperienceSoftwareRequest)
	_ = req
	var err error
	err = nil
	if err != nil {
		return &putSetupExperienceSoftwareResponse{Err: err}, nil
	}

	return &putSetupExperienceSoftwareResponse{}, nil
}
