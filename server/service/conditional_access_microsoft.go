package service

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type conditionalAccessMicrosoftCreateRequest struct {
	MicrosoftTenantID string `json:"microsoft_tenant_id"`
}

type conditionalAccessMicrosoftCreateResponse struct {
	MicrosoftAuthenticationURL string `json:"microsoft_authentication_url"`
	Err                        error  `json:"error,omitempty"`
}

func (r conditionalAccessMicrosoftCreateResponse) Error() error { return r.Err }

func conditionalAccessMicrosoftCreateEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*conditionalAccessMicrosoftCreateRequest)
	adminConsentURL, err := svc.ConditionalAccessMicrosoftCreateIntegration(ctx, req.MicrosoftTenantID)
	if err != nil {
		return conditionalAccessMicrosoftCreateResponse{Err: err}, nil
	}
	return conditionalAccessMicrosoftCreateResponse{
		MicrosoftAuthenticationURL: adminConsentURL,
	}, nil
}

func (svc *Service) ConditionalAccessMicrosoftCreateIntegration(ctx context.Context, tenantID string) (adminConsentURL string, err error) {
	// 0. Check user is authorized to create an integration.
	if err := svc.authz.Authorize(ctx, &fleet.ConditionalAccessMicrosoftIntegration{}, fleet.ActionWrite); err != nil {
		return "", ctxerr.Wrap(ctx, err, "failed to authorize")
	}

	// Load current integration, if any.
	existingIntegration, err := svc.ConditionalAccessMicrosoftGet(ctx)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "failed to load the integration")
	}
	switch {
	case existingIntegration != nil && existingIntegration.TenantID == tenantID:
		// Nothing to do, integration with same tenant ID has already been created.
		// Retrieve settings of the integration to get the admin consent URL.
		getResponse, err := svc.conditionalAccessMicrosoftProxy.Get(ctx, existingIntegration.TenantID, existingIntegration.ProxyServerSecret)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "failed to get the integration settings")
		}
		return getResponse.AdminConsentURL, nil
	case existingIntegration != nil && existingIntegration.SetupDone:
		return "", ctxerr.New(ctx, "integration already setup")
	}

	//
	// At this point we have two scenarios:
	//	- There's no integration yet, so we need to create a new one.
	//	- There's an integration already with a different TenantID and has not been setup.
	//

	// Create integration on the proxy.
	proxyCreateResponse, err := svc.conditionalAccessMicrosoftProxy.Create(ctx, tenantID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "failed to create integration in proxy")
	}

	// Create integration in datastore.
	if err := svc.ds.ConditionalAccessMicrosoftCreateIntegration(ctx, proxyCreateResponse.TenantID, proxyCreateResponse.Secret); err != nil {
		return "", ctxerr.Wrap(ctx, err, "failed to create integration in datastore")
	}

	// Retrieve settings of the integration to get the admin consent URL.
	getResponse, err := svc.conditionalAccessMicrosoftProxy.Get(ctx, proxyCreateResponse.TenantID, proxyCreateResponse.Secret)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "failed to get the integration settings")
	}
	return getResponse.AdminConsentURL, nil
}

type conditionalAccessMicrosoftConfirmRequest struct{}

type conditionalAccessMicrosoftConfirmResponse struct {
	ConfigurationCompleted bool  `json:"configuration_completed"`
	Err                    error `json:"error,omitempty"`
}

func (r conditionalAccessMicrosoftConfirmResponse) Error() error { return r.Err }

func conditionalAccessMicrosoftConfirmEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	_ = request.(*conditionalAccessMicrosoftConfirmRequest)
	configurationCompleted, err := svc.ConditionalAccessMicrosoftConfirm(ctx)
	if err != nil {
		return conditionalAccessMicrosoftConfirmResponse{Err: err}, nil
	}
	return conditionalAccessMicrosoftConfirmResponse{
		ConfigurationCompleted: configurationCompleted,
	}, nil
}

func (svc *Service) ConditionalAccessMicrosoftConfirm(ctx context.Context) (configurationCompleted bool, err error) {
	// Check user is authorized to write integrations.
	if err := svc.authz.Authorize(ctx, &fleet.ConditionalAccessMicrosoftIntegration{}, fleet.ActionWrite); err != nil {
		return false, ctxerr.Wrap(ctx, err, "failed to authorize")
	}

	// Load current integration.
	integration, err := svc.ds.ConditionalAccessMicrosoftGet(ctx)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "failed to load the integration")
	}

	if integration.SetupDone {
		return true, nil
	}

	getResponse, err := svc.conditionalAccessMicrosoftProxy.Get(ctx, integration.TenantID, integration.ProxyServerSecret)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "failed to get integration")
	}

	if !getResponse.SetupDone {
		return false, nil
	}

	if err := svc.ds.ConditionalAccessMicrosoftMarkSetupDone(ctx); err != nil {
		return false, ctxerr.Wrap(ctx, err, "failed to mark setup_done=true")
	}
	return true, nil
}

type conditionalAccessMicrosoftDeleteRequest struct{}

type conditionalAccessMicrosoftDeleteResponse struct {
	Err error `json:"error,omitempty"`
}

func (r conditionalAccessMicrosoftDeleteResponse) Error() error { return r.Err }

func conditionalAccessMicrosoftDeleteEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	_ = request.(*conditionalAccessMicrosoftDeleteRequest)
	if err := svc.ConditionalAccessMicrosoftDelete(ctx); err != nil {
		return conditionalAccessMicrosoftDeleteResponse{Err: err}, nil
	}
	return conditionalAccessMicrosoftDeleteResponse{}, nil
}

func (svc *Service) ConditionalAccessMicrosoftDelete(ctx context.Context) error {
	// Check user is authorized to delete an integration.
	if err := svc.authz.Authorize(ctx, &fleet.ConditionalAccessMicrosoftIntegration{}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err, "failed to authorize")
	}

	// Load current integration.
	integration, err := svc.ds.ConditionalAccessMicrosoftGet(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "failed to load the integration")
	}

	// Delete integration on the proxy.
	deleteResponse, err := svc.conditionalAccessMicrosoftProxy.Delete(ctx, integration.TenantID, integration.ProxyServerSecret)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "failed to delete the integration on the proxy")
	}
	if deleteResponse.Error != "" {
		return ctxerr.Wrap(ctx, errors.New(deleteResponse.Error), "delete on the proxy failed")
	}

	// Delete integration in datastore.
	if err := svc.ds.ConditionalAccessMicrosoftDelete(ctx); err != nil {
		return ctxerr.Wrap(ctx, err, "failed to delete integration in datastore")
	}

	return nil
}

func (svc *Service) ConditionalAccessMicrosoftGet(ctx context.Context) (*fleet.ConditionalAccessMicrosoftIntegration, error) {
	// Check user is authorized to read app config (which is where expose integration information)
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionRead); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to authorize")
	}

	// Load current integration.
	integration, err := svc.ds.ConditionalAccessMicrosoftGet(ctx)
	if err != nil {
		if fleet.IsNotFound(err) {
			return nil, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "failed to load the integration")
	}

	return integration, nil
}
