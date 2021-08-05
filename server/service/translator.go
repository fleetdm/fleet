package service

import (
	"context"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	kithttp "github.com/go-kit/kit/transport/http"
)

type translatorRequest struct {
	List []fleet.TranslatePayload `json:"list"`
}

type translatorResponse struct {
	List []fleet.TranslatePayload `json:"list"`
	Err  error                    `json:"error,omitempty"`
}

func (r translatorResponse) error() error { return r.Err }

func makeTranslatorEndpoint(svc fleet.Service, opts []kithttp.ServerOption) http.Handler {
	return newServer(
		makeAuthenticatedServiceEndpoint(svc, translatorEndpoint),
		makeDecoder(translatorRequest{}),
		opts,
	)
}

func translatorEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*translatorRequest)
	resp, err := svc.Translate(ctx, req.List)
	if err != nil {
		return translatorResponse{Err: err}, nil
	}
	return translatorResponse{List: resp}, nil
}

type translateFunc func(ds fleet.Datastore, identifier string) (uint, error)

func translateEmailToUserID(ds fleet.Datastore, identifier string) (uint, error) {
	user, err := ds.UserByEmail(identifier)
	if err != nil {
		return 0, err
	}
	return user.ID, nil
}

func translateLabelToID(ds fleet.Datastore, identifier string) (uint, error) {
	labelIDs, err := ds.LabelIDsByName([]string{identifier})
	if err != nil {
		return 0, err
	}
	return labelIDs[0], nil
}

func translateTeamToID(ds fleet.Datastore, identifier string) (uint, error) {
	team, err := ds.TeamByName(identifier)
	if err != nil {
		return 0, err
	}
	return team.ID, nil
}

func translateHostToID(ds fleet.Datastore, identifier string) (uint, error) {
	host, err := ds.HostByIdentifier(identifier)
	if err != nil {
		return 0, err
	}
	return host.ID, nil
}

func (svc Service) Translate(ctx context.Context, payloads []fleet.TranslatePayload) ([]fleet.TranslatePayload, error) {
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
			return nil, fleet.NewErrorf(fleet.ErrNoUnknownTranslate, "Type %s is unknown.", payload.Type)
		}

		id, err := translateFunc(svc.ds, payload.Payload.Identifier)
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
