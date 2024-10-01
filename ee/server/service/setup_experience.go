package service

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) GetSetupExperienceScript(ctx context.Context, teamID uint, withContent bool) (*fleet.Script, []byte, error) {
	// TODO: confirm auth entity
	if err := svc.authz.Authorize(ctx, &fleet.Script{TeamID: &teamID}, fleet.ActionRead); err != nil {
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

func (svc Service) SetSetupExperienceScript(ctx context.Context, teamID uint, name string, r io.Reader) error {
	// TODO: confirm auth entity
	if err := svc.authz.Authorize(ctx, &fleet.Script{TeamID: &teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "read setup experience script contents")
	}

	script := &fleet.Script{
		TeamID:         &teamID,
		Name:           name,
		ScriptContents: string(b),
	}
	if err := script.ValidateNewScript(); err != nil {
		return fleet.NewInvalidArgumentError("script", err.Error())
	}

	if err := svc.ds.SetSetupExperienceScript(ctx, script); err != nil {
		// TODO: Add unique constraint on global_or_team_id to enforce only one SE script per team?
		// If so, how to detect/handle that error? (existsErr below is for name uniqueness)
		var (
			existsErr fleet.AlreadyExistsError
			fkErr     fleet.ForeignKeyError
		)
		if errors.As(err, &existsErr) {
			err = fleet.NewInvalidArgumentError("script", "A script with this name already exists.").WithStatus(http.StatusConflict)
		} else if errors.As(err, &fkErr) {
			err = fleet.NewInvalidArgumentError("team_id", "The team does not exist.").WithStatus(http.StatusNotFound)
		}
		return ctxerr.Wrap(ctx, err, "create setup experience script")
	}

	// NOTE: there is no activity specified for set setup experience script

	return nil
}

func (svc Service) DeleteSetupExperienceScript(ctx context.Context, teamID uint) error {
	// TODO: confirm auth entity
	if err := svc.authz.Authorize(ctx, &fleet.Script{TeamID: &teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	if err := svc.ds.DeleteSetupExperienceScript(ctx, teamID); err != nil {
		return ctxerr.Wrap(ctx, err, "delete setup experience script")
	}

	// NOTE: there is no activity specified for delete setup experience script

	return nil
}
