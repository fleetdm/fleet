package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) SetSetupExperienceSoftware(ctx context.Context, teamID uint, titleIDs []uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: &teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	if err := svc.ds.SetSetupExperienceSoftwareTitles(ctx, teamID, titleIDs); err != nil {
		return ctxerr.Wrap(ctx, err, "setting setup experience titles")
	}

	return nil
}

func (svc *Service) ListSetupExperienceSoftware(ctx context.Context, teamID uint, opts fleet.ListOptions) ([]fleet.SoftwareTitleListResult, int, *fleet.PaginationMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{
		TeamID: &teamID,
	}, fleet.ActionRead); err != nil {
		return nil, 0, nil, err
	}

	titles, count, meta, err := svc.ds.ListSetupExperienceSoftwareTitles(ctx, teamID, opts)
	if err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "retrieving list of software setup experience titles")
	}

	return titles, count, meta, nil
}

func (svc *Service) GetSetupExperienceScript(ctx context.Context, teamID *uint, withContent bool) (*fleet.Script, []byte, error) {
	// TODO: confirm auth entity
	if err := svc.authz.Authorize(ctx, &fleet.Script{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, nil, err
	}

	script, err := svc.ds.GetSetupExperienceScript(ctx, teamID)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "get setup experience script")
	}

	var content []byte
	if withContent {
		content, err = svc.ds.GetAnyScriptContents(ctx, script.ScriptContentID)
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "get setup experience script contents")
		}
	}

	return script, content, nil
}

func (svc *Service) SetSetupExperienceScript(ctx context.Context, teamID *uint, name string, r io.Reader) error {
	// TODO: confirm auth entity
	if err := svc.authz.Authorize(ctx, &fleet.Script{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "read setup experience script contents")
	}

	script := &fleet.Script{
		TeamID:         teamID,
		Name:           name,
		ScriptContents: string(b),
	}

	// setup experience is only supported for macOS currently so we need to override the file
	// extension check in the general script validation
	if filepath.Ext(script.Name) != ".sh" {
		return fleet.NewInvalidArgumentError("script", "File type not supported. Only .sh file type is allowed.")
	}
	// now we can do our normal script validation
	if err := script.ValidateNewScript(); err != nil {
		return fleet.NewInvalidArgumentError("script", err.Error())
	}

	if err := svc.ds.SetSetupExperienceScript(ctx, script); err != nil {
		var (
			existsErr fleet.AlreadyExistsError
			fkErr     fleet.ForeignKeyError
		)
		if errors.As(err, &existsErr) {
			err = fleet.NewInvalidArgumentError("script", err.Error()).WithStatus(http.StatusConflict) // TODO: confirm error message with product/frontend
		} else if errors.As(err, &fkErr) {
			err = fleet.NewInvalidArgumentError("team_id", "The team does not exist.").WithStatus(http.StatusNotFound)
		}
		return ctxerr.Wrap(ctx, err, "create setup experience script")
	}

	// NOTE: there is no activity specified for set setup experience script

	return nil
}

func (svc *Service) DeleteSetupExperienceScript(ctx context.Context, teamID *uint) error {
	// TODO: confirm auth entity
	if err := svc.authz.Authorize(ctx, &fleet.Script{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	if err := svc.ds.DeleteSetupExperienceScript(ctx, teamID); err != nil {
		return ctxerr.Wrap(ctx, err, "delete setup experience script")
	}

	// NOTE: there is no activity specified for delete setup experience script

	return nil
}
