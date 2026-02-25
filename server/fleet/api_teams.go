package fleet

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

type ListTeamsRequest struct {
	ListOptions ListOptions `url:"list_options"`
}

type ListTeamsResponse struct {
	Teams []Team `json:"teams" renameto:"fleets"`
	Err   error  `json:"error,omitempty"`
}

func (r ListTeamsResponse) Error() error { return r.Err }

type GetTeamRequest struct {
	ID uint `url:"id"`
}

type GetTeamResponse struct {
	Team *Team `json:"team" renameto:"fleet"`
	Err  error `json:"error,omitempty"`
}

func (r GetTeamResponse) Error() error { return r.Err }

type DefaultTeamResponse struct {
	Team *DefaultTeam `json:"team" renameto:"fleet"`
	Err  error        `json:"error,omitempty"`
}

func (r DefaultTeamResponse) Error() error { return r.Err }

type CreateTeamRequest struct {
	TeamPayload
}

type TeamResponse struct {
	Team *Team `json:"team,omitempty" renameto:"fleet"`
	Err  error `json:"error,omitempty"`
}

func (r TeamResponse) Error() error { return r.Err }

type ModifyTeamRequest struct {
	ID uint `json:"-" url:"id"`
	TeamPayload
}

type DeleteTeamRequest struct {
	ID uint `url:"id"`
}

type DeleteTeamResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteTeamResponse) Error() error { return r.Err }

type ApplyTeamSpecsRequest struct {
	Force             bool                        `json:"-" query:"force,optional"`   // if true, bypass strict incoming json validation
	DryRun            bool                        `json:"-" query:"dry_run,optional"` // if true, apply validation but do not save changes
	DryRunAssumptions *TeamSpecsDryRunAssumptions `json:"dry_run_assumptions,omitempty"`
	Specs             []*TeamSpec                 `json:"specs"`
}

func (req *ApplyTeamSpecsRequest) DecodeBody(ctx context.Context, r io.Reader, u url.Values, c []*x509.Certificate) error {
	if err := JSONStrictDecode(r, req); err != nil {
		err = NewUserMessageError(err, http.StatusBadRequest)
		if !req.Force || !IsJSONUnknownFieldError(err) {
			// only unknown field errors can be forced at this point (other errors
			// can be forced later, after agent options' validations)
			return ctxerr.Wrap(ctx, err, "strict decode team specs")
		}
	}

	// the MacOSSettings field must be validated separately, since it
	// JSON-decodes into a free-form map.
	for _, spec := range req.Specs {
		if spec == nil || spec.MDM.MacOSSettings == nil {
			continue
		}

		var macOSSettings MacOSSettings
		validMap := macOSSettings.ToMap()

		// the keys provided must be valid
		for k := range spec.MDM.MacOSSettings {
			if _, ok := validMap[k]; !ok {
				return ctxerr.Wrap(ctx, NewUserMessageError(
					fmt.Errorf("json: unknown field %q", k),
					http.StatusBadRequest), "strict decode team specs")
			}
		}
	}
	return nil
}

type ApplyTeamSpecsResponse struct {
	Err           error           `json:"error,omitempty"`
	TeamIDsByName map[string]uint `json:"team_ids_by_name,omitempty" renameto:"fleet_ids_by_name"`
}

func (r ApplyTeamSpecsResponse) Error() error { return r.Err }

type ModifyTeamAgentOptionsRequest struct {
	ID     uint `json:"-" url:"id"`
	Force  bool `json:"-" query:"force,optional"`   // if true, bypass strict incoming json validation
	DryRun bool `json:"-" query:"dry_run,optional"` // if true, apply validation but do not save changes
	json.RawMessage
}

type ListTeamUsersRequest struct {
	TeamID      uint        `url:"id"`
	ListOptions ListOptions `url:"list_options"`
}

type ModifyTeamUsersRequest struct {
	TeamID uint `json:"-" url:"id"`
	// User ID and role must be specified for add users, user ID must be
	// specified for delete users.
	Users []TeamUser `json:"users"`
}

type TeamEnrollSecretsRequest struct {
	TeamID uint `url:"id"`
}

type TeamEnrollSecretsResponse struct {
	Secrets []*EnrollSecret `json:"secrets"`
	Err     error           `json:"error,omitempty"`
}

func (r TeamEnrollSecretsResponse) Error() error { return r.Err }

type ModifyTeamEnrollSecretsRequest struct {
	TeamID  uint           `url:"fleet_id"`
	Secrets []EnrollSecret `json:"secrets"`
}
