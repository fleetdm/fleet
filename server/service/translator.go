package service

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
)

type translatorRequest struct {
	List []fleet.TranslatePayload `json:"list"`
}

type translatorResponse struct {
	List []fleet.TranslatePayload `json:"list"`
	Err  error                    `json:"error,omitempty"`
}

func (r translatorResponse) Error() error { return r.Err }

func translatorEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*translatorRequest)
	resp, err := svc.Translate(ctx, req.List)
	if err != nil {
		return translatorResponse{Err: err}, nil
	}
	return translatorResponse{List: resp}, nil
}

type translateFunc func(ctx context.Context, ds fleet.Datastore, identifier string) (uint, error)

func translateEmailToUserID(ctx context.Context, ds fleet.Datastore, identifier string) (uint, error) {
	user, err := ds.UserByEmail(ctx, identifier)
	if err != nil {
		return 0, err
	}
	return user.ID, nil
}

func translateLabelToID(ctx context.Context, ds fleet.Datastore, identifier string) (uint, error) {
	labelIDsByName, err := ds.AllLabelIDsByName(ctx, []string{identifier}, fleet.TeamFilter{User: authz.UserFromContext(ctx)})
	if err != nil {
		return 0, err
	}
	// Every row matched the one requested name, possibly as case variants stored on different fleets.
	var matches []uint
	for _, ids := range labelIDsByName {
		matches = append(matches, ids...)
	}
	if len(matches) > 1 {
		return 0, ctxerr.Wrap(ctx, fleet.NewAmbiguousLabelNameError(identifier))
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	return 0, nil
}

func translateTeamToID(ctx context.Context, ds fleet.Datastore, identifier string) (uint, error) {
	team, err := ds.TeamByName(ctx, identifier)
	if err != nil {
		return 0, err
	}
	return team.ID, nil
}

func translateHostToID(ctx context.Context, ds fleet.Datastore, identifier string) (uint, error) {
	host, err := ds.HostByIdentifier(ctx, identifier)
	if err != nil {
		return 0, err
	}
	return host.ID, nil
}

func (svc *Service) Translate(ctx context.Context, payloads []fleet.TranslatePayload) ([]fleet.TranslatePayload, error) {
	if len(payloads) == 0 {
		// skip auth since there is no case in which this request will make sense with no payloads
		svc.authz.SkipAuthorization(ctx)
		return nil, badRequest("payloads must not be empty")
	}

	var finalPayload []fleet.TranslatePayload

	for _, payload := range payloads {
		var translateFunc translateFunc

		switch payload.Type {
		case fleet.TranslatorTypeUserEmail:
			if err := svc.authz.Authorize(ctx, &fleet.User{}, fleet.ActionRead); err != nil {
				return nil, err
			}
			translateFunc = translateEmailToUserID
		case fleet.TranslatorTypeLabel:
			if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionRead); err != nil {
				return nil, err
			}
			translateFunc = translateLabelToID
		case fleet.TranslatorTypeFleet:
			if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
				return nil, err
			}
			translateFunc = translateTeamToID
		case fleet.TranslatorTypeHost:
			if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionRead); err != nil {
				return nil, err
			}
			translateFunc = translateHostToID
		default:
			// if no supported payload type, this is bad regardless of authorization
			svc.authz.SkipAuthorization(ctx)
			if entry, ok := removednames.Find(removednames.KindTargetType, payload.Type, ""); ok {
				return nil, fleet.NewInvalidArgumentError("type", entry.Message())
			}
			return nil, endpointer.BadRequestErr(
				fmt.Sprintf("Type %s is unknown. ", payload.Type),
				fleet.NewErrorf(
					fleet.ErrNoUnknownTranslate,
					"Type %s is unknown.",
					payload.Type),
			)
		}

		id, err := translateFunc(ctx, svc.ds, payload.Payload.Identifier)
		if err != nil {
			return nil, err
		}
		payload.Payload.ID = id
		finalPayload = append(finalPayload, fleet.TranslatePayload{
			Type:    payload.Type,
			Payload: payload.Payload,
		})
	}

	return finalPayload, nil
}
