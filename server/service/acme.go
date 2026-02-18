package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"

	stepwh "github.com/smallstep/certificates/webhook"
)

type mdmACMEWebhookRequest struct {
	PermanentIdentifier string `json:"permanent_identifier"`
}

func (mdmACMEWebhookRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	// TODO: there's lots more that we should do here in terms of authenticating the incoming
	// request assuming that decide to stick with the standalone step-ca provisioner rather than
	// building an ACME provision into Fleet iteself
	// See https://github.com/smallstep/webhooks/blob/main/pkg/server/server.go#L34

	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, limit10KiB))
	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "unable to read request body",
			InternalErr: err,
		}
	}

	var decoded stepwh.RequestBody
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "unable to decode webhook request body",
			InternalErr: err,
		}
	}

	return &mdmACMEWebhookRequest{PermanentIdentifier: decoded.AttestationData.PermanentIdentifier}, nil
}

type mdmACMEWebhookResponse struct {
	Allow bool `json:"allow"`
	Data  any  `json:"data,omitempty"`

	Err error `json:"-"`
}

func (r mdmACMEWebhookResponse) Error() error { return r.Err }

func mdmACMEWebhookEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*mdmACMEWebhookRequest)
	allow, data, err := svc.MaybeAllowMDMACMEWebhook(ctx, req.PermanentIdentifier)
	if err != nil {
		return mdmACMEWebhookResponse{Err: err}, nil
	}

	return mdmACMEWebhookResponse{Allow: allow, Data: data}, nil
}

func (svc *Service) MaybeAllowMDMACMEWebhook(ctx context.Context, permanentIdentifier string) (bool, any, error) {
	// level.Info(svc.logger).Log("msg", "received MDM ACME webhook request", "permanent_identifier", permanentIdentifier)

	svc.authz.SkipAuthorization(ctx) // TODO: update this when we have a better idea of how we want to do auth for the ACME endpoints

	if permanentIdentifier == "" {
		return false, nil, &fleet.BadRequestError{
			Message:     "machine info is required",
			InternalErr: ctxerr.New(ctx, "missing permanent identifier in attestation data"),
		}
	}

	idents, err := svc.ds.GetHostMDMIdentifiers(ctx, permanentIdentifier, fleet.TeamFilter{User: &fleet.User{
		Name:       fleet.ActivityAutomationAuthor,
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}})
	switch {
	case err != nil:
		return false, nil, &fleet.BadRequestError{
			Message:     "machine info is required",
			InternalErr: ctxerr.Wrap(ctx, err, "getting host MDM identifiers from the database"),
		}
	case len(idents) != 1:
		return false, nil, &fleet.BadRequestError{
			Message:     "machine info is required",
			InternalErr: ctxerr.New(ctx, fmt.Sprintf("expected to find exactly 1 host with the given serial number, but found %d", len(idents))),
		}
	case idents[0] == nil:
		// this should never happen, but sanity check just in case
		return false, nil, &fleet.BadRequestError{
			Message:     "machine info is required",
			InternalErr: ctxerr.New(ctx, "host found with the given serial number has nil MDM identifiers"),
		}
	case idents[0].HardwareSerial != permanentIdentifier:
		// this should never happen since we're querying by the serial number, but we check just to
		// be safe and to avoid any potential shenanigans like matching on a different identifier
		// and then having a mismatch on the serial number
		return false, nil, &fleet.BadRequestError{
			Message:     "machine info is required",
			InternalErr: ctxerr.New(ctx, "host found with the given serial number has a different serial number than the one provided in the machine info"),
		}
	// case idents[0].UUID != "" && idents[0].UUID != machineInfo.UDID:
	// 	// Similar to the hardware serial check above, this is a sanity check to ensure that if we
	// 	// have stored a UDID for the host (i.e., not a pending DEP host), it matches the UDID provided in the machine info
	// 	return nil, &fleet.BadRequestError{
	// 		Message:     "machine info is required",
	// 		InternalErr: ctxerr.New(ctx, "host found with the given serial number has a different UDID than the one provided in the machine info"),
	// 	}
	case idents[0].Platform != "darwin":
		// TODO: expand this to work for iOS/iPadOS as well
		return false, nil, &fleet.BadRequestError{
			Message:     "machine info is required",
			InternalErr: ctxerr.New(ctx, "host found with the given serial number is not a darwin device"),
		}
	}

	// TODO: there's lots more we could do here to authorize the incoming request [1] beyond
	// just checking the serial number in the permanent identifier; additionally we might want to
	// enrich the response [2] with data that can be used for the certificate template by the ACME provisioner
	// [1] https://github.com/smallstep/webhooks/blob/main/pkg/server/server.go#L126
	// [2] https://smallstep.com/docs/step-ca/webhooks/#webhook-server-response

	return true, nil, nil
}
