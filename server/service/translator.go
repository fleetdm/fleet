package service

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type translatorRequest struct {
	List []fleet.TranslatePayload `json:"list"`
}

type translatorResponse struct {
	List []fleet.TranslatePayload `json:"list"`
	Err  error                    `json:"error,omitempty"`
}

func (r translatorResponse) error() error { return r.Err }

func translatorEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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
	labelIDs, err := ds.LabelIDsByName(ctx, []string{identifier})
	if err != nil {
		return 0, err
	}
	return labelIDs[identifier], nil
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
		case fleet.TranslatorTypeTeam:
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
			return nil, badRequestErr(
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
