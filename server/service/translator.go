package service

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

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
		makeDecoderForType(translatorRequest{}),
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

func (svc Service) Translate(ctx context.Context, payloads []fleet.TranslatePayload) ([]fleet.TranslatePayload, error) {
	if err := svc.authz.Authorize(ctx, &fleet.User{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	var finalPayload []fleet.TranslatePayload

	for _, payload := range payloads {
		switch payload.Type {
		case fleet.TranslatorTypeUserEmail:
			emailToID := fleet.EmailToIdPayload{}
			err := json.Unmarshal(payload.Payload, &emailToID)
			if err != nil {
				return nil, err
			}
			user, err := svc.ds.UserByEmail(emailToID.Email)
			if err != nil {
				return nil, err
			}
			emailToID.ID = user.ID
			newPayload, err := repackageNewPayload(emailToID, payload)
			if err != nil {
				return nil, err
			}
			finalPayload = append(finalPayload, newPayload)
		default:
			return nil, fleet.NewErrorf(fleet.ErrNoUnknownTranslate, "Type %s is unknown.", payload.Type)
		}
	}

	return finalPayload, nil
}

func repackageNewPayload(translatedPayload interface{}, payload fleet.TranslatePayload) (fleet.TranslatePayload, error) {
	translatedPayloadBytes, err := json.Marshal(translatedPayload)
	if err != nil {
		return fleet.TranslatePayload{}, err
	}
	newPayload := fleet.TranslatePayload{
		Type:    payload.Type,
		Payload: translatedPayloadBytes,
	}
	return newPayload, nil
}

func (mw loggingMiddleware) Translate(ctx context.Context, payloads []fleet.TranslatePayload) ([]fleet.TranslatePayload, error) {
	var err error
	defer func(begin time.Time) {
		_ = mw.loggerDebug(err).Log("method", "ApplyUserRolesSpecs", "err", err, "took", time.Since(begin))
	}(time.Now())
	return mw.Service.Translate(ctx, payloads)
}
