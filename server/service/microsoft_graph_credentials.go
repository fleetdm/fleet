package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

/////////////////////////////////////////////////////////////////////////////////
// GET /microsoft_graph_credentials
/////////////////////////////////////////////////////////////////////////////////

type listMicrosoftGraphCredentialsResponse struct {
	MicrosoftGraphCredentials []*fleet.MicrosoftGraphCredentialMetadata `json:"microsoft_graph_credentials"`
	Err                       error                                     `json:"error,omitempty"`
}

func (r listMicrosoftGraphCredentialsResponse) Error() error { return r.Err }

func listMicrosoftGraphCredentialsEndpoint(ctx context.Context, _ any, svc fleet.Service) (fleet.Errorer, error) {
	creds, err := svc.ListMicrosoftGraphCredentials(ctx)
	if err != nil {
		return listMicrosoftGraphCredentialsResponse{Err: err}, nil
	}
	return listMicrosoftGraphCredentialsResponse{MicrosoftGraphCredentials: creds}, nil
}

func (svc *Service) ListMicrosoftGraphCredentials(ctx context.Context) ([]*fleet.MicrosoftGraphCredentialMetadata, error) {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return nil, fleet.ErrMissingLicense
}

/////////////////////////////////////////////////////////////////////////////////
// PUT /microsoft_graph_credentials
/////////////////////////////////////////////////////////////////////////////////

type applyMicrosoftGraphCredentialsRequest struct {
	MicrosoftGraphCredentials []fleet.MicrosoftGraphCredential `json:"microsoft_graph_credentials"`
	DryRun                    bool                             `json:"dry_run"`
}

type applyMicrosoftGraphCredentialsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyMicrosoftGraphCredentialsResponse) Error() error { return r.Err }

func applyMicrosoftGraphCredentialsEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*applyMicrosoftGraphCredentialsRequest)
	if err := svc.ApplyMicrosoftGraphCredentials(ctx, req.MicrosoftGraphCredentials, req.DryRun); err != nil {
		return applyMicrosoftGraphCredentialsResponse{Err: err}, nil
	}
	return applyMicrosoftGraphCredentialsResponse{}, nil
}

func (svc *Service) ApplyMicrosoftGraphCredentials(ctx context.Context, _ []fleet.MicrosoftGraphCredential, _ bool) error {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return fleet.ErrMissingLicense
}
