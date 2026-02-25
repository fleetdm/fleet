package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

const (
	SecretVariablePrefix = "FLEET_SECRET_"
)

func createSecretVariablesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.CreateSecretVariablesRequest)
	err := svc.CreateSecretVariables(ctx, req.SecretVariables, req.DryRun)
	return fleet.CreateSecretVariablesResponse{Err: err}, nil
}

func (svc *Service) CreateSecretVariables(ctx context.Context, secretVariables []fleet.SecretVariable, dryRun bool) error {
	// Do authorization check first so that we don't have to worry about it later in the flow.
	if err := svc.authz.Authorize(ctx, &fleet.SecretVariable{}, fleet.ActionWrite); err != nil {
		return err
	}

	privateKey := svc.config.Server.PrivateKey
	if testSetEmptyPrivateKey {
		privateKey = ""
	}

	if len(privateKey) == 0 {
		return ctxerr.Wrap(ctx,
			&fleet.BadRequestError{Message: "Couldn't save secret variables. Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key"})
	}

	// Preprocess: strip FLEET_SECRET_ prefix from variable names
	for i, secretVariable := range secretVariables {
		secretVariables[i].Name = fleet.Preprocess(strings.TrimPrefix(secretVariable.Name, SecretVariablePrefix))
	}

	for _, secretVariable := range secretVariables {
		if err := fleet.ValidateSecretVariableName(secretVariable.Name); err != nil {
			return ctxerr.Wrap(ctx, err, "validate secret variable name")
		}
	}

	if dryRun {
		return nil
	}

	if err := svc.ds.UpsertSecretVariables(ctx, secretVariables); err != nil {
		return ctxerr.Wrap(ctx, err, "saving secret variables")
	}
	return nil
}

func createSecretVariableEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.CreateSecretVariableRequest)
	id, err := svc.CreateSecretVariable(ctx, req.Name, req.Value)
	if err != nil {
		return fleet.CreateSecretVariableResponse{Err: err}, nil
	}
	return fleet.CreateSecretVariableResponse{
		ID:   id,
		Name: req.Name,
	}, nil
}

func (svc *Service) CreateSecretVariable(ctx context.Context, name string, value string) (id uint, err error) {
	if err := svc.authz.Authorize(ctx, &fleet.SecretVariable{}, fleet.ActionWrite); err != nil {
		return 0, err
	}

	privateKey := svc.config.Server.PrivateKey
	if testSetEmptyPrivateKey {
		privateKey = ""
	}

	if len(privateKey) == 0 {
		return 0, ctxerr.Wrap(ctx,
			&fleet.BadRequestError{
				Message: "Couldn't save secret variable. Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key",
			})
	}

	if err := fleet.ValidateSecretVariableName(name); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "validate secret variable name")
	}
	if value == "" {
		return 0, fleet.NewInvalidArgumentError("name", "secret variable value cannot be empty")
	}

	id, err = svc.ds.CreateSecretVariable(ctx, name, value)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "saving secret variable")
	}

	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityCreatedCustomVariable{
			CustomVariableID:   id,
			CustomVariableName: name,
		},
	); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "create activity for secret variable creation")
	}

	return id, nil
}

func listSecretVariablesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ListSecretVariablesRequest)
	secretVariables, meta, count, err := svc.ListSecretVariables(ctx, req.ListOptions)
	return fleet.ListSecretVariablesResponse{
		CustomVariables: secretVariables,
		Meta:            meta,
		Count:           count,

		Err: err,
	}, nil
}

func (svc *Service) ListSecretVariables(
	ctx context.Context,
	opts fleet.ListOptions,
) (
	secretVariables []fleet.SecretVariableIdentifier,
	meta *fleet.PaginationMetadata,
	count int,
	err error,
) {
	if err := svc.authz.Authorize(ctx, &fleet.SecretVariable{}, fleet.ActionRead); err != nil {
		return nil, nil, 0, err
	}

	// MatchQuery/After currently not supported
	opts.MatchQuery = ""
	opts.After = ""
	// Always include pagination info.
	opts.IncludeMetadata = true
	// Default sort order is name ascending.
	opts.OrderKey = "name"
	opts.OrderDirection = fleet.OrderAscending

	secretVariables, meta, count, err = svc.ds.ListSecretVariables(ctx, opts)
	if err != nil {
		return nil, nil, 0, ctxerr.Wrap(ctx, err, "list secret variables")
	}

	return secretVariables, meta, count, nil
}

func deleteSecretVariableEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.DeleteSecretVariableRequest)
	err := svc.DeleteSecretVariable(ctx, req.ID)
	return fleet.DeleteSecretVariableResponse{
		Err: err,
	}, nil
}

func (svc *Service) DeleteSecretVariable(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.SecretVariable{}, fleet.ActionWrite); err != nil {
		return err
	}
	deletedSecretVariableName, err := svc.ds.DeleteSecretVariable(ctx, id)
	if err != nil {
		var secretUsedErr *fleet.SecretUsedError
		if errors.As(err, &secretUsedErr) {
			return ctxerr.Wrap(ctx, &fleet.ConflictError{
				Message: fmt.Sprintf("Couldn't delete. %s", secretUsedErr.Error()),
			}, "delete secret variable")
		}
		return ctxerr.Wrap(ctx, err, "delete secret variable")
	}
	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityDeletedCustomVariable{
			CustomVariableID:   id,
			CustomVariableName: deletedSecretVariableName,
		},
	); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for secret variable deletion")
	}
	return nil
}
