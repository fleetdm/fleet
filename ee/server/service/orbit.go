package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) GetOrbitSetupExperienceStatus(ctx context.Context, orbitNodeKey string) (*fleet.SetupExperienceStatusPayload, error) {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)
	host, err := svc.ds.LoadHostByOrbitNodeKey(ctx, orbitNodeKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "loading host by orbit node key")
	}

	res, err := svc.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing setup experience results")
	}

	payload := &fleet.SetupExperienceStatusPayload{Software: make([]*fleet.SetupExperienceStatusResult, 0)}
	for _, r := range res {
		if r.IsForScript() {
			payload.Script = r
		}

		if r.IsForSoftware() {
			payload.Software = append(payload.Software, r)
		}
	}

	return payload, nil
}
