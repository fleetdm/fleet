package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/go-units"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	mdm_types "github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/appmanifest"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/sso"
	"github.com/fleetdm/fleet/v4/server/worker"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/groob/plist"
	"github.com/micromdm/nanodep/godep"
)

type getMDMAppleCommandResultsRequest struct {
	CommandUUID string `query:"command_uuid,optional"`
}

type getMDMAppleCommandResultsResponse struct {
	Results []*fleet.MDMCommandResult `json:"results,omitempty"`
	Err     error                     `json:"error,omitempty"`
}

func (r getMDMAppleCommandResultsResponse) error() error { return r.Err }

func getMDMAppleCommandResultsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getMDMAppleCommandResultsRequest)
	results, err := svc.GetMDMAppleCommandResults(ctx, req.CommandUUID)
	if err != nil {
		return getMDMAppleCommandResultsResponse{
			Err: err,
		}, nil
	}

	return getMDMAppleCommandResultsResponse{
		Results: results,
	}, nil
}

func (svc *Service) GetMDMAppleCommandResults(ctx context.Context, commandUUID string) ([]*fleet.MDMCommandResult, error) {
	// first, authorize that the user has the right to list hosts
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	// check that command exists first, to return 404 on invalid commands
	// (the command may exist but have no results yet).
	if _, err := svc.ds.GetMDMAppleCommandRequestType(ctx, commandUUID); err != nil {
		return nil, err
	}

	// next, we need to read the command results before we know what hosts (and
	// therefore what teams) we're dealing with.
	results, err := svc.ds.GetMDMAppleCommandResults(ctx, commandUUID)
	if err != nil {
		return nil, err
	}

	// now we can load the hosts (lite) corresponding to those command results,
	// and do the final authorization check with the proper team(s). Include observers,
	// as they are able to view command results for their teams' hosts.
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}
	hostUUIDs := make([]string, len(results))
	for i, res := range results {
		hostUUIDs[i] = res.HostUUID
	}
	hosts, err := svc.ds.ListHostsLiteByUUIDs(ctx, filter, hostUUIDs)
	if err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		// do not return 404 here, as it's possible for a command to not have
		// results yet
		return nil, nil
	}

	// collect the team IDs and verify that the user has access to view commands
	// on all affected teams. Index the hosts by uuid for easly lookup as
	// afterwards we'll want to store the hostname on the returned results.
	hostsByUUID := make(map[string]*fleet.Host, len(hosts))
	teamIDs := make(map[uint]bool)
	for _, h := range hosts {
		var id uint
		if h.TeamID != nil {
			id = *h.TeamID
		}
		teamIDs[id] = true
		hostsByUUID[h.UUID] = h
	}

	var commandAuthz fleet.MDMCommandAuthz
	for tmID := range teamIDs {
		commandAuthz.TeamID = &tmID
		if tmID == 0 {
			commandAuthz.TeamID = nil
		}

		if err := svc.authz.Authorize(ctx, commandAuthz, fleet.ActionRead); err != nil {
			return nil, ctxerr.Wrap(ctx, err)
		}
	}

	// add the hostnames to the results
	for _, res := range results {
		if h := hostsByUUID[res.HostUUID]; h != nil {
			res.Hostname = hostsByUUID[res.HostUUID].Hostname
		}
	}
	return results, nil
}

type listMDMAppleCommandsRequest struct {
	ListOptions fleet.ListOptions `url:"list_options"`
}

type listMDMAppleCommandsResponse struct {
	Results []*fleet.MDMAppleCommand `json:"results"`
	Err     error                    `json:"error,omitempty"`
}

func (r listMDMAppleCommandsResponse) error() error { return r.Err }

func listMDMAppleCommandsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*listMDMAppleCommandsRequest)
	results, err := svc.ListMDMAppleCommands(ctx, &fleet.MDMCommandListOptions{
		ListOptions: req.ListOptions,
	})
	if err != nil {
		return listMDMAppleCommandsResponse{
			Err: err,
		}, nil
	}

	return listMDMAppleCommandsResponse{
		Results: results,
	}, nil
}

func (svc *Service) ListMDMAppleCommands(ctx context.Context, opts *fleet.MDMCommandListOptions) ([]*fleet.MDMAppleCommand, error) {
	// first, authorize that the user has the right to list hosts
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	// get the list of commands so we know what hosts (and therefore what teams)
	// we're dealing with. Including the observers as they are allowed to view
	// MDM Apple commands.
	results, err := svc.ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{
		User:            vc.User,
		IncludeObserver: true,
	}, opts)
	if err != nil {
		return nil, err
	}

	// collect the different team IDs and verify that the user has access to view
	// commands on all affected teams, do not assume that ListMDMAppleCommands
	// only returned hosts that the user is authorized to view the command
	// results of (that is, always verify with our rego authz policy).
	teamIDs := make(map[uint]bool)
	for _, res := range results {
		var id uint
		if res.TeamID != nil {
			id = *res.TeamID
		}
		teamIDs[id] = true
	}

	// instead of returning an authz error if the user is not authorized for a
	// team, we remove those commands from the results (as we want to return
	// whatever the user is allowed to see). Since this can only be done after
	// retrieving the list of commands, this may result in returning less results
	// than requested, but it's ok - it's expected that the results retrieved
	// from the datastore will all be authorized for the user.
	var commandAuthz fleet.MDMCommandAuthz
	var authzErr error
	for tmID := range teamIDs {
		commandAuthz.TeamID = &tmID
		if tmID == 0 {
			commandAuthz.TeamID = nil
		}
		if err := svc.authz.Authorize(ctx, commandAuthz, fleet.ActionRead); err != nil {
			if authzErr == nil {
				authzErr = err
			}
			teamIDs[tmID] = false
		}
	}

	if authzErr != nil {
		level.Error(svc.logger).Log("err", "unauthorized to view some team commands", "details", authzErr)

		// filter-out the teams that the user is not allowed to view
		allowedResults := make([]*fleet.MDMAppleCommand, 0, len(results))
		for _, res := range results {
			var id uint
			if res.TeamID != nil {
				id = *res.TeamID
			}
			if teamIDs[id] {
				allowedResults = append(allowedResults, res)
			}
		}
		results = allowedResults
	}

	return results, nil
}

type newMDMAppleConfigProfileRequest struct {
	TeamID  uint
	Profile *multipart.FileHeader
}

type newMDMAppleConfigProfileResponse struct {
	ProfileID uint  `json:"profile_id"`
	Err       error `json:"error,omitempty"`
}

// TODO(lucas): We parse the whole body before running svc.authz.Authorize.
// An authenticated but unauthorized user could abuse this.
func (newMDMAppleConfigProfileRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	decoded := newMDMAppleConfigProfileRequest{}

	err := r.ParseMultipartForm(512 * units.MiB)
	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}

	val, ok := r.MultipartForm.Value["team_id"]
	if !ok || len(val) < 1 {
		// default is no team
		decoded.TeamID = 0
	} else {
		teamID, err := strconv.Atoi(val[0])
		if err != nil {
			return nil, &fleet.BadRequestError{Message: fmt.Sprintf("failed to decode team_id in multipart form: %s", err.Error())}
		}
		decoded.TeamID = uint(teamID)
	}

	fhs, ok := r.MultipartForm.File["profile"]
	if !ok || len(fhs) < 1 {
		return nil, &fleet.BadRequestError{Message: "no file headers for profile"}
	}
	decoded.Profile = fhs[0]

	return &decoded, nil
}

func (r newMDMAppleConfigProfileResponse) error() error { return r.Err }

func newMDMAppleConfigProfileEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*newMDMAppleConfigProfileRequest)

	ff, err := req.Profile.Open()
	if err != nil {
		return &newMDMAppleConfigProfileResponse{Err: err}, nil
	}
	defer ff.Close()
	// providing an empty set of labels since this endpoint is only maintained for backwards compat
	cp, err := svc.NewMDMAppleConfigProfile(ctx, req.TeamID, ff, nil)
	if err != nil {
		return &newMDMAppleConfigProfileResponse{Err: err}, nil
	}
	return &newMDMAppleConfigProfileResponse{
		ProfileID: cp.ProfileID,
	}, nil
}

func (svc *Service) NewMDMAppleConfigProfile(ctx context.Context, teamID uint, r io.Reader, labels []string) (*fleet.MDMAppleConfigProfile, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: &teamID}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	// check that Apple MDM is enabled - the middleware of that endpoint checks
	// only that any MDM is enabled, maybe it's just Windows
	if err := svc.VerifyMDMAppleConfigured(ctx); err != nil {
		err := fleet.NewInvalidArgumentError("profile", fleet.AppleMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest)
		return nil, ctxerr.Wrap(ctx, err, "check macOS MDM enabled")
	}

	var teamName string
	if teamID >= 1 {
		tm, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, &teamID, nil)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err)
		}
		teamName = tm.Name
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message:     "failed to read Apple config profile",
			InternalErr: err,
		})
	}

	cp, err := fleet.NewMDMAppleConfigProfile(b, &teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message: fmt.Sprintf("failed to parse config profile: %s", err.Error()),
		})
	}

	if err := cp.ValidateUserProvided(); err != nil {
		return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{Message: err.Error()})
	}

	labelMap, err := svc.validateProfileLabels(ctx, labels)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validating labels")
	}
	cp.Labels = labelMap

	newCP, err := svc.ds.NewMDMAppleConfigProfile(ctx, *cp)
	if err != nil {
		var existsErr existsErrorInterface
		if errors.As(err, &existsErr) {
			err = fleet.NewInvalidArgumentError("profile", "Couldn't upload. A configuration profile with this name already exists.").
				WithStatus(http.StatusConflict)
		}
		return nil, ctxerr.Wrap(ctx, err)
	}
	if err := svc.ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, []string{newCP.ProfileUUID}, nil); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "bulk set pending host profiles")
	}

	var (
		actTeamID   *uint
		actTeamName *string
	)
	if teamID > 0 {
		actTeamID = &teamID
		actTeamName = &teamName
	}
	if err := svc.ds.NewActivity(ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeCreatedMacosProfile{
		TeamID:            actTeamID,
		TeamName:          actTeamName,
		ProfileName:       newCP.Name,
		ProfileIdentifier: newCP.Identifier,
	}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "logging activity for create mdm apple config profile")
	}

	return newCP, nil
}

type listMDMAppleConfigProfilesRequest struct {
	TeamID uint `query:"team_id,optional"`
}

type listMDMAppleConfigProfilesResponse struct {
	ConfigProfiles []*fleet.MDMAppleConfigProfile `json:"profiles"`
	Err            error                          `json:"error,omitempty"`
}

func (r listMDMAppleConfigProfilesResponse) error() error { return r.Err }

func listMDMAppleConfigProfilesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*listMDMAppleConfigProfilesRequest)

	cps, err := svc.ListMDMAppleConfigProfiles(ctx, req.TeamID)
	if err != nil {
		return &listMDMAppleConfigProfilesResponse{Err: err}, nil
	}

	res := listMDMAppleConfigProfilesResponse{ConfigProfiles: cps}
	if cps == nil {
		res.ConfigProfiles = []*fleet.MDMAppleConfigProfile{} // return empty json array instead of json null
	}
	return &res, nil
}

func (svc *Service) ListMDMAppleConfigProfiles(ctx context.Context, teamID uint) ([]*fleet.MDMAppleConfigProfile, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: &teamID}, fleet.ActionRead); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	if teamID >= 1 {
		// confirm that team exists
		if _, err := svc.ds.Team(ctx, teamID); err != nil {
			return nil, ctxerr.Wrap(ctx, err)
		}
	}

	cps, err := svc.ds.ListMDMAppleConfigProfiles(ctx, &teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	return cps, nil
}

type getMDMAppleConfigProfileRequest struct {
	ProfileID uint `url:"profile_id"`
}

type getMDMAppleConfigProfileResponse struct {
	Err error `json:"error,omitempty"`

	// file fields below are used in hijackRender for the response
	fileReader io.ReadCloser
	fileLength int64
	fileName   string
}

func (r getMDMAppleConfigProfileResponse) error() error { return r.Err }

func (r getMDMAppleConfigProfileResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.FormatInt(r.fileLength, 10))
	w.Header().Set("Content-Type", "application/x-apple-aspen-config")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s.mobileconfig"`, r.fileName))

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided
	wl, err := io.Copy(w, r.fileReader)
	if err != nil {
		logging.WithExtras(ctx, "mobileconfig_copy_error", err, "bytes_copied", wl)
	}
	r.fileReader.Close()
}

func getMDMAppleConfigProfileEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getMDMAppleConfigProfileRequest)

	cp, err := svc.GetMDMAppleConfigProfileByDeprecatedID(ctx, req.ProfileID)
	if err != nil {
		return getMDMAppleConfigProfileResponse{Err: err}, nil
	}
	reader := bytes.NewReader(cp.Mobileconfig)
	fileName := fmt.Sprintf("%s_%s", time.Now().Format("2006-01-02"), strings.ReplaceAll(cp.Name, " ", "_"))

	return getMDMAppleConfigProfileResponse{fileReader: io.NopCloser(reader), fileLength: reader.Size(), fileName: fileName}, nil
}

func (svc *Service) GetMDMAppleConfigProfileByDeprecatedID(ctx context.Context, profileID uint) (*fleet.MDMAppleConfigProfile, error) {
	// first we perform a perform basic authz check
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	cp, err := svc.ds.GetMDMAppleConfigProfileByDeprecatedID(ctx, profileID)
	if err != nil {
		if fleet.IsNotFound(err) {
			// call the standard service method with a profile UUID that will not be
			// found, just to ensure the same sequence of validations are applied.
			return svc.GetMDMAppleConfigProfile(ctx, "-")
		}
		return nil, ctxerr.Wrap(ctx, err)
	}
	return svc.GetMDMAppleConfigProfile(ctx, cp.ProfileUUID)
}

func (svc *Service) GetMDMAppleConfigProfile(ctx context.Context, profileUUID string) (*fleet.MDMAppleConfigProfile, error) {
	// first we perform a perform basic authz check
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	cp, err := svc.ds.GetMDMAppleConfigProfile(ctx, profileUUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	// now we can do a specific authz check based on team id of profile before we return the profile
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: cp.TeamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return cp, nil
}

type deleteMDMAppleConfigProfileRequest struct {
	ProfileID uint `url:"profile_id"`
}

type deleteMDMAppleConfigProfileResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteMDMAppleConfigProfileResponse) error() error { return r.Err }

func deleteMDMAppleConfigProfileEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteMDMAppleConfigProfileRequest)

	if err := svc.DeleteMDMAppleConfigProfileByDeprecatedID(ctx, req.ProfileID); err != nil {
		return &deleteMDMAppleConfigProfileResponse{Err: err}, nil
	}

	return &deleteMDMAppleConfigProfileResponse{}, nil
}

func (svc *Service) DeleteMDMAppleConfigProfileByDeprecatedID(ctx context.Context, profileID uint) error {
	// first we perform a perform basic authz check
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	// get the profile by ID and call the standard delete function
	cp, err := svc.ds.GetMDMAppleConfigProfileByDeprecatedID(ctx, profileID)
	if err != nil {
		if fleet.IsNotFound(err) {
			// call the standard service method with a profile UUID that will not be
			// found, just to ensure the same sequence of validations are applied.
			return svc.DeleteMDMAppleConfigProfile(ctx, "-")
		}
		return ctxerr.Wrap(ctx, err)
	}
	return svc.DeleteMDMAppleConfigProfile(ctx, cp.ProfileUUID)
}

func (svc *Service) DeleteMDMAppleConfigProfile(ctx context.Context, profileUUID string) error {
	// first we perform a perform basic authz check
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	// check that Apple MDM is enabled - the middleware of that endpoint checks
	// only that any MDM is enabled, maybe it's just Windows
	if err := svc.VerifyMDMAppleConfigured(ctx); err != nil {
		err := fleet.NewInvalidArgumentError("profile_uuid", fleet.AppleMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest)
		return ctxerr.Wrap(ctx, err, "check macOS MDM enabled")
	}

	cp, err := svc.ds.GetMDMAppleConfigProfile(ctx, profileUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	var teamName string
	teamID := *cp.TeamID
	if teamID >= 1 {
		tm, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, &teamID, nil)
		if err != nil {
			return ctxerr.Wrap(ctx, err)
		}
		teamName = tm.Name
	}

	// now we can do a specific authz check based on team id of profile before we delete the profile
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: cp.TeamID}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	// prevent deleting profiles that are managed by Fleet
	if _, ok := mobileconfig.FleetPayloadIdentifiers()[cp.Identifier]; ok {
		return &fleet.BadRequestError{
			Message:     "profiles managed by Fleet can't be deleted using this endpoint.",
			InternalErr: fmt.Errorf("deleting profile %s for team %s not allowed because it's managed by Fleet", cp.Identifier, teamName),
		}
	}

	if err := svc.ds.DeleteMDMAppleConfigProfile(ctx, profileUUID); err != nil {
		return ctxerr.Wrap(ctx, err)
	}
	// cannot use the profile ID as it is now deleted
	if err := svc.ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{teamID}, nil, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk set pending host profiles")
	}

	var (
		actTeamID   *uint
		actTeamName *string
	)
	if teamID > 0 {
		actTeamID = &teamID
		actTeamName = &teamName
	}
	if err := svc.ds.NewActivity(ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeDeletedMacosProfile{
		TeamID:            actTeamID,
		TeamName:          actTeamName,
		ProfileName:       cp.Name,
		ProfileIdentifier: cp.Identifier,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "logging activity for delete mdm apple config profile")
	}

	return nil
}

type getMDMAppleFileVaultSummaryRequest struct {
	TeamID *uint `query:"team_id,optional"`
}

type getMDMAppleFileVaultSummaryResponse struct {
	*fleet.MDMAppleFileVaultSummary
	Err error `json:"error,omitempty"`
}

func (r getMDMAppleFileVaultSummaryResponse) error() error { return r.Err }

func getMdmAppleFileVaultSummaryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getMDMAppleFileVaultSummaryRequest)

	fvs, err := svc.GetMDMAppleFileVaultSummary(ctx, req.TeamID)
	if err != nil {
		return &getMDMAppleFileVaultSummaryResponse{Err: err}, nil
	}

	return &getMDMAppleFileVaultSummaryResponse{
		MDMAppleFileVaultSummary: fvs,
	}, nil
}

func (svc *Service) GetMDMAppleFileVaultSummary(ctx context.Context, teamID *uint) (*fleet.MDMAppleFileVaultSummary, error) {
	if err := svc.authz.Authorize(ctx, fleet.MDMConfigProfileAuthz{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	fvs, err := svc.ds.GetMDMAppleFileVaultSummary(ctx, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	return fvs, nil
}

type getMDMAppleProfilesSummaryRequest struct {
	TeamID *uint `query:"team_id,optional"`
}

type getMDMAppleProfilesSummaryResponse struct {
	fleet.MDMProfilesSummary
	Err error `json:"error,omitempty"`
}

func (r getMDMAppleProfilesSummaryResponse) error() error { return r.Err }

func getMDMAppleProfilesSummaryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getMDMAppleProfilesSummaryRequest)
	res := getMDMAppleProfilesSummaryResponse{}

	ps, err := svc.GetMDMAppleProfilesSummary(ctx, req.TeamID)
	if err != nil {
		return &getMDMAppleProfilesSummaryResponse{Err: err}, nil
	}

	res.Verified = ps.Verified
	res.Verifying = ps.Verifying
	res.Failed = ps.Failed
	res.Pending = ps.Pending

	return &res, nil
}

func (svc *Service) GetMDMAppleProfilesSummary(ctx context.Context, teamID *uint) (*fleet.MDMProfilesSummary, error) {
	if err := svc.authz.Authorize(ctx, fleet.MDMConfigProfileAuthz{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	if err := svc.VerifyMDMAppleConfigured(ctx); err != nil {
		return &fleet.MDMProfilesSummary{}, nil
	}

	ps, err := svc.ds.GetMDMAppleProfilesSummary(ctx, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	return ps, nil
}

type uploadAppleInstallerRequest struct {
	Installer *multipart.FileHeader
}

type uploadAppleInstallerResponse struct {
	ID  uint  `json:"installer_id"`
	Err error `json:"error,omitempty"`
}

// TODO(lucas): We parse the whole body before running svc.authz.Authorize.
// An authenticated but unauthorized user could abuse this.
func (uploadAppleInstallerRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	err := r.ParseMultipartForm(512 * units.MiB)
	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}
	installer := r.MultipartForm.File["installer"][0]
	return &uploadAppleInstallerRequest{
		Installer: installer,
	}, nil
}

func (r uploadAppleInstallerResponse) error() error { return r.Err }

func uploadAppleInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*uploadAppleInstallerRequest)
	ff, err := req.Installer.Open()
	if err != nil {
		return uploadAppleInstallerResponse{Err: err}, nil
	}
	defer ff.Close()
	installer, err := svc.UploadMDMAppleInstaller(ctx, req.Installer.Filename, req.Installer.Size, ff)
	if err != nil {
		return uploadAppleInstallerResponse{Err: err}, nil
	}
	return &uploadAppleInstallerResponse{
		ID: installer.ID,
	}, nil
}

func (svc *Service) UploadMDMAppleInstaller(ctx context.Context, name string, size int64, installer io.Reader) (*fleet.MDMAppleInstaller, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleInstaller{}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	token := uuid.New().String()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	url := svc.installerURL(token, appConfig)

	var installerBuf bytes.Buffer
	manifest, err := createManifest(size, io.TeeReader(installer, &installerBuf), url)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	inst, err := svc.ds.NewMDMAppleInstaller(ctx, name, size, manifest, installerBuf.Bytes(), token)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	return inst, nil
}

func (svc *Service) installerURL(token string, appConfig *fleet.AppConfig) string {
	return fmt.Sprintf("%s%s?token=%s", appConfig.ServerSettings.ServerURL, apple_mdm.InstallerPath, token)
}

func createManifest(size int64, installer io.Reader, url string) (string, error) {
	manifest, err := appmanifest.New(&readerWithSize{
		Reader: installer,
		size:   size,
	}, url)
	if err != nil {
		return "", fmt.Errorf("create manifest file: %w", err)
	}
	var buf bytes.Buffer
	enc := plist.NewEncoder(&buf)
	enc.Indent("  ")
	if err := enc.Encode(manifest); err != nil {
		return "", fmt.Errorf("encode manifest: %w", err)
	}
	return buf.String(), nil
}

type readerWithSize struct {
	io.Reader
	size int64
}

func (r *readerWithSize) Size() int64 {
	return r.size
}

type getAppleInstallerDetailsRequest struct {
	ID uint `url:"installer_id"`
}

type getAppleInstallerDetailsResponse struct {
	Installer *fleet.MDMAppleInstaller
	Err       error `json:"error,omitempty"`
}

func (r getAppleInstallerDetailsResponse) error() error { return r.Err }

func getAppleInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getAppleInstallerDetailsRequest)
	installer, err := svc.GetMDMAppleInstallerByID(ctx, req.ID)
	if err != nil {
		return getAppleInstallerDetailsResponse{Err: err}, nil
	}
	return &getAppleInstallerDetailsResponse{
		Installer: installer,
	}, nil
}

func (svc *Service) GetMDMAppleInstallerByID(ctx context.Context, id uint) (*fleet.MDMAppleInstaller, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleInstaller{}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	inst, err := svc.ds.MDMAppleInstallerDetailsByID(ctx, id)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	return inst, nil
}

type deleteAppleInstallerDetailsRequest struct {
	ID uint `url:"installer_id"`
}

type deleteAppleInstallerDetailsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteAppleInstallerDetailsResponse) error() error { return r.Err }

func deleteAppleInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteAppleInstallerDetailsRequest)
	if err := svc.DeleteMDMAppleInstaller(ctx, req.ID); err != nil {
		return deleteAppleInstallerDetailsResponse{Err: err}, nil
	}
	return &deleteAppleInstallerDetailsResponse{}, nil
}

func (svc *Service) DeleteMDMAppleInstaller(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleInstaller{}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	if err := svc.ds.DeleteMDMAppleInstaller(ctx, id); err != nil {
		return ctxerr.Wrap(ctx, err)
	}
	return nil
}

type listMDMAppleDevicesRequest struct{}

type listMDMAppleDevicesResponse struct {
	Devices []fleet.MDMAppleDevice `json:"devices"`
	Err     error                  `json:"error,omitempty"`
}

func (r listMDMAppleDevicesResponse) error() error { return r.Err }

func listMDMAppleDevicesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	devices, err := svc.ListMDMAppleDevices(ctx)
	if err != nil {
		return listMDMAppleDevicesResponse{Err: err}, nil
	}
	return &listMDMAppleDevicesResponse{
		Devices: devices,
	}, nil
}

func (svc *Service) ListMDMAppleDevices(ctx context.Context) ([]fleet.MDMAppleDevice, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleDevice{}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	return svc.ds.MDMAppleListDevices(ctx)
}

type listMDMAppleDEPDevicesRequest struct{}

type listMDMAppleDEPDevicesResponse struct {
	Devices []fleet.MDMAppleDEPDevice `json:"devices"`
	Err     error                     `json:"error,omitempty"`
}

func (r listMDMAppleDEPDevicesResponse) error() error { return r.Err }

func listMDMAppleDEPDevicesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	devices, err := svc.ListMDMAppleDEPDevices(ctx)
	if err != nil {
		return listMDMAppleDEPDevicesResponse{Err: err}, nil
	}
	return &listMDMAppleDEPDevicesResponse{
		Devices: devices,
	}, nil
}

func (svc *Service) ListMDMAppleDEPDevices(ctx context.Context) ([]fleet.MDMAppleDEPDevice, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleDEPDevice{}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	depClient := apple_mdm.NewDEPClient(svc.depStorage, svc.ds, svc.logger)

	// TODO(lucas): Use cursors and limit to fetch in multiple requests.
	// This single-request version supports up to 1000 devices (max to return in one call).
	fetchDevicesResponse, err := depClient.FetchDevices(ctx, apple_mdm.DEPName, godep.WithLimit(1000))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	devices := make([]fleet.MDMAppleDEPDevice, len(fetchDevicesResponse.Devices))
	for i := range fetchDevicesResponse.Devices {
		devices[i] = fleet.MDMAppleDEPDevice{Device: fetchDevicesResponse.Devices[i]}
	}
	return devices, nil
}

type newMDMAppleDEPKeyPairResponse struct {
	PublicKey  []byte `json:"public_key,omitempty"`
	PrivateKey []byte `json:"private_key,omitempty"`
	Err        error  `json:"error,omitempty"`
}

func (r newMDMAppleDEPKeyPairResponse) error() error { return r.Err }

func newMDMAppleDEPKeyPairEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	keyPair, err := svc.NewMDMAppleDEPKeyPair(ctx)
	if err != nil {
		return newMDMAppleDEPKeyPairResponse{
			Err: err,
		}, nil
	}

	return newMDMAppleDEPKeyPairResponse{
		PublicKey:  keyPair.PublicKey,
		PrivateKey: keyPair.PrivateKey,
	}, nil
}

func (svc *Service) NewMDMAppleDEPKeyPair(ctx context.Context) (*fleet.MDMAppleDEPKeyPair, error) {
	// skipauth: Generating a new key pair does not actually make any changes to fleet, or expose any
	// information. The user must configure fleet with the new key pair and restart the server.
	svc.authz.SkipAuthorization(ctx)

	publicKeyPEM, privateKeyPEM, err := apple_mdm.NewDEPKeyPairPEM()
	if err != nil {
		return nil, fmt.Errorf("generate key pair: %w", err)
	}

	return &fleet.MDMAppleDEPKeyPair{
		PublicKey:  publicKeyPEM,
		PrivateKey: privateKeyPEM,
	}, nil
}

type enqueueMDMAppleCommandRequest struct {
	Command   string   `json:"command"`
	DeviceIDs []string `json:"device_ids"`
}

type enqueueMDMAppleCommandResponse struct {
	*fleet.CommandEnqueueResult
	Err error `json:"error,omitempty"`
}

func (r enqueueMDMAppleCommandResponse) error() error { return r.Err }

// Deprecated: enqueueMDMAppleCommandEndpoint is now deprecated, replaced by
// the platform-agnostic runMDMCommandEndpoint. It is still supported
// indefinitely for backwards compatibility.
func enqueueMDMAppleCommandEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*enqueueMDMAppleCommandRequest)
	result, err := svc.EnqueueMDMAppleCommand(ctx, req.Command, req.DeviceIDs)
	if err != nil {
		return enqueueMDMAppleCommandResponse{Err: err}, nil
	}
	return enqueueMDMAppleCommandResponse{
		CommandEnqueueResult: result,
	}, nil
}

func (svc *Service) EnqueueMDMAppleCommand(
	ctx context.Context,
	rawBase64Cmd string,
	deviceIDs []string,
) (result *fleet.CommandEnqueueResult, err error) {
	hosts, err := svc.authorizeAllHostsTeams(ctx, deviceIDs, fleet.ActionWrite, &fleet.MDMCommandAuthz{})
	if err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		return nil, newNotFoundError()
	}

	// using a padding agnostic decoder because we released this using
	// base64.RawStdEncoding, but it was causing problems as many standard
	// libraries default to padded strings. We're now supporting both for
	// backwards compatibility.
	rawXMLCmd, err := server.Base64DecodePaddingAgnostic(rawBase64Cmd)
	if err != nil {
		err = fleet.NewInvalidArgumentError("command", "unable to decode base64 command").WithStatus(http.StatusBadRequest)

		return nil, ctxerr.Wrap(ctx, err, "decode base64 command")
	}

	return svc.enqueueAppleMDMCommand(ctx, rawXMLCmd, deviceIDs)
}

type mdmAppleEnrollRequest struct {
	Token               string `query:"token"`
	EnrollmentReference string `query:"enrollment_reference,optional"`
}

func (r mdmAppleEnrollResponse) error() error { return r.Err }

type mdmAppleEnrollResponse struct {
	Err error `json:"error,omitempty"`

	// Profile field is used in hijackRender for the response.
	Profile []byte
}

func (r mdmAppleEnrollResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(r.Profile)), 10))
	w.Header().Set("Content-Type", "application/x-apple-aspen-config")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", "attachment;fleet-enrollment-profile.mobileconfig")

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided.
	if n, err := w.Write(r.Profile); err != nil {
		logging.WithExtras(ctx, "err", err, "written", n)
	}
}

func mdmAppleEnrollEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*mdmAppleEnrollRequest)

	profile, err := svc.GetMDMAppleEnrollmentProfileByToken(ctx, req.Token, req.EnrollmentReference)
	if err != nil {
		return mdmAppleEnrollResponse{Err: err}, nil
	}
	return mdmAppleEnrollResponse{
		Profile: profile,
	}, nil
}

func (svc *Service) GetMDMAppleEnrollmentProfileByToken(ctx context.Context, token string, ref string) (profile []byte, err error) {
	// skipauth: The enroll profile endpoint is unauthenticated.
	svc.authz.SkipAuthorization(ctx)

	_, err = svc.ds.GetMDMAppleEnrollmentProfileByToken(ctx, token)
	if err != nil {
		if fleet.IsNotFound(err) {
			return nil, fleet.NewAuthFailedError("enrollment profile not found")
		}
		return nil, ctxerr.Wrap(ctx, err, "get enrollment profile")
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	enrollURL, err := apple_mdm.AddEnrollmentRefToFleetURL(appConfig.ServerSettings.ServerURL, ref)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "adding reference to fleet URL")
	}

	mobileconfig, err := apple_mdm.GenerateEnrollmentProfileMobileconfig(
		appConfig.OrgInfo.OrgName,
		enrollURL,
		svc.config.MDM.AppleSCEPChallenge,
		svc.mdmPushCertTopic,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	return mobileconfig, nil
}

type mdmAppleCommandRemoveEnrollmentProfileRequest struct {
	HostID uint `url:"id"`
}

type mdmAppleCommandRemoveEnrollmentProfileResponse struct {
	Err error `json:"error,omitempty"`
}

func (r mdmAppleCommandRemoveEnrollmentProfileResponse) error() error { return r.Err }

func mdmAppleCommandRemoveEnrollmentProfileEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*mdmAppleCommandRemoveEnrollmentProfileRequest)
	err := svc.EnqueueMDMAppleCommandRemoveEnrollmentProfile(ctx, req.HostID)
	if err != nil {
		return mdmAppleCommandRemoveEnrollmentProfileResponse{Err: err}, nil
	}
	return mdmAppleCommandRemoveEnrollmentProfileResponse{}, nil
}

func (svc *Service) EnqueueMDMAppleCommandRemoveEnrollmentProfile(ctx context.Context, hostID uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return err
	}

	h, err := svc.ds.HostLite(ctx, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting host info for mdm apple remove profile command")
	}

	info, err := svc.ds.GetHostMDMCheckinInfo(ctx, h.UUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting mdm checkin info for mdm apple remove profile command")
	}

	// Check authorization again based on host info for team-based permissions.
	if err := svc.authz.Authorize(ctx, fleet.MDMCommandAuthz{
		TeamID: h.TeamID,
	}, fleet.ActionWrite); err != nil {
		return err
	}

	nanoEnroll, err := svc.ds.GetNanoMDMEnrollment(ctx, h.UUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting mdm enrollment status for mdm apple remove profile command")
	}
	if nanoEnroll == nil || !nanoEnroll.Enabled {
		return fleet.NewUserMessageError(ctxerr.New(ctx, fmt.Sprintf("mdm is not enabled for host %d", hostID)), http.StatusConflict)
	}

	cmdUUID := uuid.New().String()
	err = svc.mdmAppleCommander.RemoveProfile(ctx, []string{h.UUID}, apple_mdm.FleetPayloadIdentifier, cmdUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "enqueuing mdm apple remove profile command")
	}

	if err := svc.ds.NewActivity(ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeMDMUnenrolled{
		HostSerial:       h.HardwareSerial,
		HostDisplayName:  h.DisplayName(),
		InstalledFromDEP: info.InstalledFromDEP,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "logging activity for mdm apple remove profile command")
	}

	return svc.pollResultMDMAppleCommandRemoveEnrollmentProfile(ctx, cmdUUID, h.UUID)
}

func (svc *Service) pollResultMDMAppleCommandRemoveEnrollmentProfile(ctx context.Context, cmdUUID string, deviceID string) error {
	ctx, cancelFn := context.WithDeadline(ctx, time.Now().Add(5*time.Second))
	ticker := time.NewTicker(300 * time.Millisecond)
	defer func() {
		ticker.Stop()
		cancelFn()
	}()

	for {
		select {
		case <-ctx.Done():
			// time out after 5 seconds
			return fleet.MDMAppleCommandTimeoutError{}
		case <-ticker.C:
			nanoEnroll, err := svc.ds.GetNanoMDMEnrollment(ctx, deviceID)
			if err != nil {
				level.Error(svc.logger).Log("err", "get nanomdm enrollment status", "details", err, "id", deviceID, "command_uuid", cmdUUID)
				return err
			}
			if nanoEnroll != nil && nanoEnroll.Enabled {
				// check again on the next tick
				continue
			}
			// success, mdm enrollment is no longer enabled for the device
			level.Info(svc.logger).Log("msg", "mdm disabled for device", "id", deviceID, "command_uuid", cmdUUID)
			return nil
		}
	}
}

type mdmAppleGetInstallerRequest struct {
	Token string `query:"token"`
}

func (r mdmAppleGetInstallerResponse) error() error { return r.Err }

type mdmAppleGetInstallerResponse struct {
	Err error `json:"error,omitempty"`

	// head is used by hijackRender for the response.
	head bool
	// Name field is used in hijackRender for the response.
	name string
	// Size field is used in hijackRender for the response.
	size int64
	// Installer field is used in hijackRender for the response.
	installer []byte
}

func (r mdmAppleGetInstallerResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.FormatInt(r.size, 10))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, r.name))

	if r.head {
		w.WriteHeader(http.StatusOK)
		return
	}

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided
	if n, err := w.Write(r.installer); err != nil {
		logging.WithExtras(ctx, "err", err, "bytes_copied", n)
	}
}

func mdmAppleGetInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*mdmAppleGetInstallerRequest)
	installer, err := svc.GetMDMAppleInstallerByToken(ctx, req.Token)
	if err != nil {
		return mdmAppleGetInstallerResponse{Err: err}, nil
	}
	return mdmAppleGetInstallerResponse{
		head:      false,
		name:      installer.Name,
		size:      installer.Size,
		installer: installer.Installer,
	}, nil
}

func (svc *Service) GetMDMAppleInstallerByToken(ctx context.Context, token string) (*fleet.MDMAppleInstaller, error) {
	// skipauth: The installer endpoint uses token authentication.
	svc.authz.SkipAuthorization(ctx)

	installer, err := svc.ds.MDMAppleInstaller(ctx, token)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	return installer, nil
}

type mdmAppleHeadInstallerRequest struct {
	Token string `query:"token"`
}

func mdmAppleHeadInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*mdmAppleHeadInstallerRequest)
	installer, err := svc.GetMDMAppleInstallerDetailsByToken(ctx, req.Token)
	if err != nil {
		return mdmAppleGetInstallerResponse{Err: err}, nil
	}
	return mdmAppleGetInstallerResponse{
		head: true,
		name: installer.Name,
		size: installer.Size,
	}, nil
}

func (svc *Service) GetMDMAppleInstallerDetailsByToken(ctx context.Context, token string) (*fleet.MDMAppleInstaller, error) {
	// skipauth: The installer endpoint uses token authentication.
	svc.authz.SkipAuthorization(ctx)

	installer, err := svc.ds.MDMAppleInstallerDetailsByToken(ctx, token)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	return installer, nil
}

type listMDMAppleInstallersRequest struct{}

type listMDMAppleInstallersResponse struct {
	Installers []fleet.MDMAppleInstaller `json:"installers"`
	Err        error                     `json:"error,omitempty"`
}

func (r listMDMAppleInstallersResponse) error() error { return r.Err }

func listMDMAppleInstallersEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	installers, err := svc.ListMDMAppleInstallers(ctx)
	if err != nil {
		return listMDMAppleInstallersResponse{
			Err: err,
		}, nil
	}
	return listMDMAppleInstallersResponse{
		Installers: installers,
	}, nil
}

func (svc *Service) ListMDMAppleInstallers(ctx context.Context) ([]fleet.MDMAppleInstaller, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleInstaller{}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	installers, err := svc.ds.ListMDMAppleInstallers(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	for i := range installers {
		installers[i].URL = svc.installerURL(installers[i].URLToken, appConfig)
	}
	return installers, nil
}

////////////////////////////////////////////////////////////////////////////////
// Lock a device
////////////////////////////////////////////////////////////////////////////////

type deviceLockRequest struct {
	HostID uint `url:"id"`
}

type deviceLockResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deviceLockResponse) error() error { return r.Err }

func (r deviceLockResponse) Status() int { return http.StatusNoContent }

func deviceLockEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deviceLockRequest)
	err := svc.MDMAppleDeviceLock(ctx, req.HostID)
	if err != nil {
		return deviceLockResponse{Err: err}, nil
	}
	return deviceLockResponse{}, nil
}

func (svc *Service) MDMAppleDeviceLock(ctx context.Context, hostID uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Wipe a device
////////////////////////////////////////////////////////////////////////////////

type deviceWipeRequest struct {
	HostID uint `url:"id"`
}

type deviceWipeResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deviceWipeResponse) error() error { return r.Err }

func (r deviceWipeResponse) Status() int { return http.StatusNoContent }

func deviceWipeEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deviceWipeRequest)
	err := svc.MDMAppleEraseDevice(ctx, req.HostID)
	if err != nil {
		return deviceWipeResponse{Err: err}, nil
	}
	return deviceWipeResponse{}, nil
}

func (svc *Service) MDMAppleEraseDevice(ctx context.Context, hostID uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Get profiles assigned to a host
////////////////////////////////////////////////////////////////////////////////

type getHostProfilesRequest struct {
	ID uint `url:"id"`
}

type getHostProfilesResponse struct {
	HostID   uint                           `json:"host_id"`
	Profiles []*fleet.MDMAppleConfigProfile `json:"profiles"`
	Err      error                          `json:"error,omitempty"`
}

func (r getHostProfilesResponse) error() error { return r.Err }

func getHostProfilesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getHostProfilesRequest)
	sums, err := svc.MDMListHostConfigurationProfiles(ctx, req.ID)
	if err != nil {
		return getHostProfilesResponse{Err: err}, nil
	}
	res := getHostProfilesResponse{Profiles: sums, HostID: req.ID}
	if res.Profiles == nil {
		res.Profiles = []*fleet.MDMAppleConfigProfile{} // return empty json array instead of json null
	}
	return res, nil
}

func (svc *Service) MDMListHostConfigurationProfiles(ctx context.Context, hostID uint) ([]*fleet.MDMAppleConfigProfile, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Batch Replace MDM Apple Profiles
////////////////////////////////////////////////////////////////////////////////

type batchSetMDMAppleProfilesRequest struct {
	TeamID   *uint    `json:"-" query:"team_id,optional"`
	TeamName *string  `json:"-" query:"team_name,optional"`
	DryRun   bool     `json:"-" query:"dry_run,optional"` // if true, apply validation but do not save changes
	Profiles [][]byte `json:"profiles"`
}

type batchSetMDMAppleProfilesResponse struct {
	Err error `json:"error,omitempty"`
}

func (r batchSetMDMAppleProfilesResponse) error() error { return r.Err }

func (r batchSetMDMAppleProfilesResponse) Status() int { return http.StatusNoContent }

func batchSetMDMAppleProfilesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*batchSetMDMAppleProfilesRequest)
	if err := svc.BatchSetMDMAppleProfiles(ctx, req.TeamID, req.TeamName, req.Profiles, req.DryRun, false); err != nil {
		return batchSetMDMAppleProfilesResponse{Err: err}, nil
	}
	return batchSetMDMAppleProfilesResponse{}, nil
}

func (svc *Service) BatchSetMDMAppleProfiles(ctx context.Context, tmID *uint, tmName *string, profiles [][]byte, dryRun, skipBulkPending bool) error {
	var err error
	tmID, tmName, err = svc.authorizeBatchProfiles(ctx, tmID, tmName)
	if err != nil {
		return err
	}

	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	if !appCfg.MDM.EnabledAndConfigured {
		// NOTE: in order to prevent an error when Fleet MDM is not enabled but no
		// profile is provided, which can happen if a user runs `fleetctl get
		// config` and tries to apply that YAML, as it will contain an empty/null
		// custom_settings key, we just return a success response in this
		// situation.
		if len(profiles) == 0 {
			return nil
		}

		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("mdm", "cannot set custom settings: Fleet MDM is not configured"))
	}

	// any duplicate identifier or name in the provided set results in an error
	profs := make([]*fleet.MDMAppleConfigProfile, 0, len(profiles))
	byName, byIdent := make(map[string]bool, len(profiles)), make(map[string]bool, len(profiles))
	for i, prof := range profiles {
		mdmProf, err := fleet.NewMDMAppleConfigProfile(prof, tmID)
		if err != nil {
			return ctxerr.Wrap(ctx,
				fleet.NewInvalidArgumentError(fmt.Sprintf("profiles[%d]", i), err.Error()),
				"invalid mobileconfig profile")
		}

		if err := mdmProf.ValidateUserProvided(); err != nil {
			return ctxerr.Wrap(ctx,
				fleet.NewInvalidArgumentError(fmt.Sprintf("profiles[%d]", i), err.Error()))
		}

		if byName[mdmProf.Name] {
			return ctxerr.Wrap(ctx,
				fleet.NewInvalidArgumentError(fmt.Sprintf("profiles[%d]", i), fmt.Sprintf("Couldn’t edit custom_settings. More than one configuration profile have the same name (PayloadDisplayName): %q", mdmProf.Name)),
				"duplicate mobileconfig profile by name")
		}
		byName[mdmProf.Name] = true

		if byIdent[mdmProf.Identifier] {
			return ctxerr.Wrap(ctx,
				fleet.NewInvalidArgumentError(fmt.Sprintf("profiles[%d]", i), fmt.Sprintf("Couldn’t edit custom_settings. More than one configuration profile have the same identifier (PayloadIdentifier): %q", mdmProf.Identifier)),
				"duplicate mobileconfig profile by identifier")
		}
		byIdent[mdmProf.Identifier] = true

		profs = append(profs, mdmProf)
	}

	if dryRun {
		return nil
	}
	if err := svc.ds.BatchSetMDMAppleProfiles(ctx, tmID, profs); err != nil {
		return err
	}
	var bulkTeamID uint
	if tmID != nil {
		bulkTeamID = *tmID
	}

	if !skipBulkPending {
		if err := svc.ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{bulkTeamID}, nil, nil); err != nil {
			return ctxerr.Wrap(ctx, err, "bulk set pending host profiles")
		}
	}

	if err := svc.ds.NewActivity(ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeEditedMacosProfile{
		TeamID:   tmID,
		TeamName: tmName,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "logging activity for edited macos profile")
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Preassign a profile to a host
////////////////////////////////////////////////////////////////////////////////

type preassignMDMAppleProfileRequest struct {
	fleet.MDMApplePreassignProfilePayload
}

type preassignMDMAppleProfileResponse struct {
	Err error `json:"error,omitempty"`
}

func (r preassignMDMAppleProfileResponse) error() error { return r.Err }

func (r preassignMDMAppleProfileResponse) Status() int { return http.StatusNoContent }

func preassignMDMAppleProfileEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*preassignMDMAppleProfileRequest)
	if err := svc.MDMApplePreassignProfile(ctx, req.MDMApplePreassignProfilePayload); err != nil {
		return preassignMDMAppleProfileResponse{Err: err}, nil
	}
	return preassignMDMAppleProfileResponse{}, nil
}

func (svc *Service) MDMApplePreassignProfile(ctx context.Context, payload fleet.MDMApplePreassignProfilePayload) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Match a set of pre-assigned profiles with a team
////////////////////////////////////////////////////////////////////////////////

type matchMDMApplePreassignmentRequest struct {
	ExternalHostIdentifier string `json:"external_host_identifier"`
}

type matchMDMApplePreassignmentResponse struct {
	Err error `json:"error,omitempty"`
}

func (r matchMDMApplePreassignmentResponse) error() error { return r.Err }

func (r matchMDMApplePreassignmentResponse) Status() int { return http.StatusNoContent }

func matchMDMApplePreassignmentEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*matchMDMApplePreassignmentRequest)
	if err := svc.MDMAppleMatchPreassignment(ctx, req.ExternalHostIdentifier); err != nil {
		return matchMDMApplePreassignmentResponse{Err: err}, nil
	}
	return matchMDMApplePreassignmentResponse{}, nil
}

func (svc *Service) MDMAppleMatchPreassignment(ctx context.Context, ref string) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Update MDM Apple Settings
////////////////////////////////////////////////////////////////////////////////

type updateMDMAppleSettingsRequest struct {
	fleet.MDMAppleSettingsPayload
}

type updateMDMAppleSettingsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r updateMDMAppleSettingsResponse) error() error { return r.Err }

func (r updateMDMAppleSettingsResponse) Status() int { return http.StatusNoContent }

// This endpoint is required because the UI must allow maintainers (in addition
// to admins) to update some MDM Apple settings, while the update config/update
// team endpoints only allow write access to admins.
func updateMDMAppleSettingsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*updateMDMAppleSettingsRequest)
	if err := svc.UpdateMDMAppleSettings(ctx, req.MDMAppleSettingsPayload); err != nil {
		return updateMDMAppleSettingsResponse{Err: err}, nil
	}
	return updateMDMAppleSettingsResponse{}, nil
}

func (svc *Service) UpdateMDMAppleSettings(ctx context.Context, payload fleet.MDMAppleSettingsPayload) error {
	// for now, assume all settings require premium (this is true for the first
	// supported setting, enable_disk_encryption. Adjust as needed in the future
	// if this is not always the case).
	lic, _ := license.FromContext(ctx)
	if lic == nil || !lic.IsPremium() {
		svc.authz.SkipAuthorization(ctx) // so that the error message is not replaced by "forbidden"
		return ErrMissingLicense
	}

	if err := svc.authz.Authorize(ctx, payload, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	if payload.TeamID != nil {
		tm, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, payload.TeamID, nil)
		if err != nil {
			return err
		}
		return svc.EnterpriseOverrides.UpdateTeamMDMAppleSettings(ctx, tm, payload)
	}
	return svc.updateAppConfigMDMAppleSettings(ctx, payload)
}

func (svc *Service) updateAppConfigMDMAppleSettings(ctx context.Context, payload fleet.MDMAppleSettingsPayload) error {
	// appconfig is only used internally, it's fine to read it unobfuscated
	// (svc.AppConfigObfuscated must not be used because the write-only users
	// such as gitops will fail to access it).
	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return err
	}

	var didUpdate, didUpdateMacOSDiskEncryption bool
	if payload.EnableDiskEncryption != nil {
		if ac.MDM.EnableDiskEncryption.Value != *payload.EnableDiskEncryption {
			ac.MDM.EnableDiskEncryption = optjson.SetBool(*payload.EnableDiskEncryption)
			didUpdate = true
			didUpdateMacOSDiskEncryption = true
		}
	}

	if didUpdate {
		if err := svc.ds.SaveAppConfig(ctx, ac); err != nil {
			return err
		}
		if didUpdateMacOSDiskEncryption {
			var act fleet.ActivityDetails
			if ac.MDM.EnableDiskEncryption.Value {
				act = fleet.ActivityTypeEnabledMacosDiskEncryption{}
				if err := svc.EnterpriseOverrides.MDMAppleEnableFileVaultAndEscrow(ctx, nil); err != nil {
					return ctxerr.Wrap(ctx, err, "enable no-team filevault and escrow")
				}
			} else {
				act = fleet.ActivityTypeDisabledMacosDiskEncryption{}
				if err := svc.EnterpriseOverrides.MDMAppleDisableFileVaultAndEscrow(ctx, nil); err != nil {
					return ctxerr.Wrap(ctx, err, "disable no-team filevault and escrow")
				}
			}
			if err := svc.ds.NewActivity(ctx, authz.UserFromContext(ctx), act); err != nil {
				return ctxerr.Wrap(ctx, err, "create activity for app config macos disk encryption")
			}
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Upload a bootstrap package
////////////////////////////////////////////////////////////////////////////////

type uploadBootstrapPackageRequest struct {
	Package *multipart.FileHeader
	TeamID  uint
}

type uploadBootstrapPackageResponse struct {
	Err error `json:"error,omitempty"`
}

// TODO: We parse the whole body before running svc.authz.Authorize.
// An authenticated but unauthorized user could abuse this.
func (uploadBootstrapPackageRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	decoded := uploadBootstrapPackageRequest{}
	err := r.ParseMultipartForm(512 * units.MiB)
	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}

	if r.MultipartForm.File["package"] == nil {
		return nil, &fleet.BadRequestError{
			Message:     "package multipart field is required",
			InternalErr: err,
		}
	}

	decoded.Package = r.MultipartForm.File["package"][0]
	if !file.IsValidMacOSName(decoded.Package.Filename) {
		return nil, &fleet.BadRequestError{
			Message:     "package name contains invalid characters",
			InternalErr: ctxerr.New(ctx, "package name contains invalid characters"),
		}
	}

	// default is no team
	decoded.TeamID = 0
	val, ok := r.MultipartForm.Value["team_id"]
	if ok && len(val) > 0 {
		teamID, err := strconv.Atoi(val[0])
		if err != nil {
			return nil, &fleet.BadRequestError{Message: fmt.Sprintf("failed to decode team_id in multipart form: %s", err.Error())}
		}
		decoded.TeamID = uint(teamID)
	}

	return &decoded, nil
}

func (r uploadBootstrapPackageResponse) error() error { return r.Err }

func uploadBootstrapPackageEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*uploadBootstrapPackageRequest)
	ff, err := req.Package.Open()
	if err != nil {
		return uploadBootstrapPackageResponse{Err: err}, nil
	}
	defer ff.Close()

	if err := svc.MDMAppleUploadBootstrapPackage(ctx, req.Package.Filename, ff, req.TeamID); err != nil {
		return uploadBootstrapPackageResponse{Err: err}, nil
	}
	return &uploadBootstrapPackageResponse{}, nil
}

func (svc *Service) MDMAppleUploadBootstrapPackage(ctx context.Context, name string, pkg io.Reader, teamID uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Download a bootstrap package
////////////////////////////////////////////////////////////////////////////////

type downloadBootstrapPackageRequest struct {
	Token string `query:"token"`
}

type downloadBootstrapPackageResponse struct {
	Err error `json:"error,omitempty"`

	// fields used by hijackRender for the response.
	pkg *fleet.MDMAppleBootstrapPackage
}

func (r downloadBootstrapPackageResponse) error() error { return r.Err }

func (r downloadBootstrapPackageResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.Itoa(len(r.pkg.Bytes)))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, r.pkg.Name))

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided
	if n, err := w.Write(r.pkg.Bytes); err != nil {
		logging.WithExtras(ctx, "err", err, "bytes_copied", n)
	}
}

func downloadBootstrapPackageEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*downloadBootstrapPackageRequest)
	pkg, err := svc.GetMDMAppleBootstrapPackageBytes(ctx, req.Token)
	if err != nil {
		return downloadBootstrapPackageResponse{Err: err}, nil
	}
	return downloadBootstrapPackageResponse{pkg: pkg}, nil
}

func (svc *Service) GetMDMAppleBootstrapPackageBytes(ctx context.Context, token string) (*fleet.MDMAppleBootstrapPackage, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Get metadata about a bootstrap package
////////////////////////////////////////////////////////////////////////////////

type bootstrapPackageMetadataRequest struct {
	TeamID uint `url:"team_id"`

	// ForUpdate is used to indicate that the authorization should be for a
	// "write" instead of a "read", this is needed specifically for the gitops
	// user which is a write-only user, but needs to call this endpoint to check
	// if it needs to upload the bootstrap package (if the hashes are different).
	ForUpdate bool `query:"for_update,optional"`
}

type bootstrapPackageMetadataResponse struct {
	Err                             error `json:"error,omitempty"`
	*fleet.MDMAppleBootstrapPackage `json:",omitempty"`
}

func (r bootstrapPackageMetadataResponse) error() error { return r.Err }

func bootstrapPackageMetadataEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*bootstrapPackageMetadataRequest)
	meta, err := svc.GetMDMAppleBootstrapPackageMetadata(ctx, req.TeamID, req.ForUpdate)
	if err != nil {
		return bootstrapPackageMetadataResponse{Err: err}, nil
	}
	return bootstrapPackageMetadataResponse{MDMAppleBootstrapPackage: meta}, nil
}

func (svc *Service) GetMDMAppleBootstrapPackageMetadata(ctx context.Context, teamID uint, forUpdate bool) (*fleet.MDMAppleBootstrapPackage, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Delete a bootstrap package
////////////////////////////////////////////////////////////////////////////////

type deleteBootstrapPackageRequest struct {
	TeamID uint `url:"team_id"`
}

type deleteBootstrapPackageResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteBootstrapPackageResponse) error() error { return r.Err }

func deleteBootstrapPackageEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteBootstrapPackageRequest)
	if err := svc.DeleteMDMAppleBootstrapPackage(ctx, &req.TeamID); err != nil {
		return deleteBootstrapPackageResponse{Err: err}, nil
	}
	return deleteBootstrapPackageResponse{}, nil
}

func (svc *Service) DeleteMDMAppleBootstrapPackage(ctx context.Context, teamID *uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Get aggregated summary about a team's bootstrap package
////////////////////////////////////////////////////////////////////////////////

type getMDMAppleBootstrapPackageSummaryRequest struct {
	TeamID *uint `query:"team_id,optional"`
}

type getMDMAppleBootstrapPackageSummaryResponse struct {
	fleet.MDMAppleBootstrapPackageSummary
	Err error `json:"error,omitempty"`
}

func (r getMDMAppleBootstrapPackageSummaryResponse) error() error { return r.Err }

func getMDMAppleBootstrapPackageSummaryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getMDMAppleBootstrapPackageSummaryRequest)
	summary, err := svc.GetMDMAppleBootstrapPackageSummary(ctx, req.TeamID)
	if err != nil {
		return getMDMAppleBootstrapPackageSummaryResponse{Err: err}, nil
	}
	return getMDMAppleBootstrapPackageSummaryResponse{MDMAppleBootstrapPackageSummary: *summary}, nil
}

func (svc *Service) GetMDMAppleBootstrapPackageSummary(ctx context.Context, teamID *uint) (*fleet.MDMAppleBootstrapPackageSummary, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return &fleet.MDMAppleBootstrapPackageSummary{}, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Create or update an MDM Apple Setup Assistant
////////////////////////////////////////////////////////////////////////////////

type createMDMAppleSetupAssistantRequest struct {
	TeamID            *uint           `json:"team_id"`
	Name              string          `json:"name"`
	EnrollmentProfile json.RawMessage `json:"enrollment_profile"`
}

type createMDMAppleSetupAssistantResponse struct {
	fleet.MDMAppleSetupAssistant
	Err error `json:"error,omitempty"`
}

func (r createMDMAppleSetupAssistantResponse) error() error { return r.Err }

func createMDMAppleSetupAssistantEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*createMDMAppleSetupAssistantRequest)
	asst, err := svc.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{
		TeamID:  req.TeamID,
		Name:    req.Name,
		Profile: req.EnrollmentProfile,
	})
	if err != nil {
		return createMDMAppleSetupAssistantResponse{Err: err}, nil
	}
	return createMDMAppleSetupAssistantResponse{MDMAppleSetupAssistant: *asst}, nil
}

func (svc *Service) SetOrUpdateMDMAppleSetupAssistant(ctx context.Context, asst *fleet.MDMAppleSetupAssistant) (*fleet.MDMAppleSetupAssistant, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Get the MDM Apple Setup Assistant
////////////////////////////////////////////////////////////////////////////////

type getMDMAppleSetupAssistantRequest struct {
	TeamID *uint `query:"team_id,optional"`
}

type getMDMAppleSetupAssistantResponse struct {
	fleet.MDMAppleSetupAssistant
	Err error `json:"error,omitempty"`
}

func (r getMDMAppleSetupAssistantResponse) error() error { return r.Err }

func getMDMAppleSetupAssistantEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getMDMAppleSetupAssistantRequest)
	asst, err := svc.GetMDMAppleSetupAssistant(ctx, req.TeamID)
	if err != nil {
		return getMDMAppleSetupAssistantResponse{Err: err}, nil
	}
	return getMDMAppleSetupAssistantResponse{MDMAppleSetupAssistant: *asst}, nil
}

func (svc *Service) GetMDMAppleSetupAssistant(ctx context.Context, teamID *uint) (*fleet.MDMAppleSetupAssistant, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Delete an MDM Apple Setup Assistant
////////////////////////////////////////////////////////////////////////////////

type deleteMDMAppleSetupAssistantRequest struct {
	TeamID *uint `query:"team_id,optional"`
}

type deleteMDMAppleSetupAssistantResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteMDMAppleSetupAssistantResponse) error() error { return r.Err }
func (r deleteMDMAppleSetupAssistantResponse) Status() int  { return http.StatusNoContent }

func deleteMDMAppleSetupAssistantEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteMDMAppleSetupAssistantRequest)
	if err := svc.DeleteMDMAppleSetupAssistant(ctx, req.TeamID); err != nil {
		return deleteMDMAppleSetupAssistantResponse{Err: err}, nil
	}
	return deleteMDMAppleSetupAssistantResponse{}, nil
}

func (svc *Service) DeleteMDMAppleSetupAssistant(ctx context.Context, teamID *uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Update MDM Apple Setup
////////////////////////////////////////////////////////////////////////////////

type updateMDMAppleSetupRequest struct {
	fleet.MDMAppleSetupPayload
}

type updateMDMAppleSetupResponse struct {
	Err error `json:"error,omitempty"`
}

func (r updateMDMAppleSetupResponse) error() error { return r.Err }

func (r updateMDMAppleSetupResponse) Status() int { return http.StatusNoContent }

// This endpoint is required because the UI must allow maintainers (in addition
// to admins) to update some MDM Apple settings, while the update config/update
// team endpoints only allow write access to admins.
func updateMDMAppleSetupEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*updateMDMAppleSetupRequest)
	if err := svc.UpdateMDMAppleSetup(ctx, req.MDMAppleSetupPayload); err != nil {
		return updateMDMAppleSetupResponse{Err: err}, nil
	}
	return updateMDMAppleSetupResponse{}, nil
}

func (svc *Service) UpdateMDMAppleSetup(ctx context.Context, payload fleet.MDMAppleSetupPayload) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// POST /mdm/sso
////////////////////////////////////////////////////////////////////////////////

type initiateMDMAppleSSORequest struct{}

type initiateMDMAppleSSOResponse struct {
	URL string `json:"url,omitempty"`
	Err error  `json:"error,omitempty"`
}

func (r initiateMDMAppleSSOResponse) error() error { return r.Err }

func initiateMDMAppleSSOEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	idpProviderURL, err := svc.InitiateMDMAppleSSO(ctx)
	if err != nil {
		return initiateMDMAppleSSOResponse{Err: err}, nil
	}

	return initiateMDMAppleSSOResponse{URL: idpProviderURL}, nil
}

func (svc *Service) InitiateMDMAppleSSO(ctx context.Context) (string, error) {
	// skipauth: No authorization check needed due to implementation
	// returning only license error.
	svc.authz.SkipAuthorization(ctx)

	return "", fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// POST /mdm/sso/callback
////////////////////////////////////////////////////////////////////////////////

type callbackMDMAppleSSORequest struct{}

// TODO: these errors will result in JSON being returned, but we should
// redirect to the UI and let the UI display an error instead. The errors are
// rare enough (malformed data coming from the SSO provider) so they shouldn't
// affect many users.
func (callbackMDMAppleSSORequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	err := r.ParseForm()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message:     "failed to parse form",
			InternalErr: err,
		}, "decode sso callback")
	}
	authResponse, err := sso.DecodeAuthResponse(r.FormValue("SAMLResponse"))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message:     "failed to decode SAMLResponse",
			InternalErr: err,
		}, "decoding sso callback")
	}
	return authResponse, nil
}

type callbackMDMAppleSSOResponse struct {
	redirectURL string
}

func (r callbackMDMAppleSSOResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Location", r.redirectURL)
	w.WriteHeader(http.StatusSeeOther)
}

// Error will always be nil because errors are handled by sending a query
// parameter in the URL response, this way the UI is able to display an erorr
// message.
func (r callbackMDMAppleSSOResponse) error() error { return nil }

func callbackMDMAppleSSOEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	auth := request.(fleet.Auth)
	redirectURL := svc.InitiateMDMAppleSSOCallback(ctx, auth)
	return callbackMDMAppleSSOResponse{redirectURL: redirectURL}, nil
}

func (svc *Service) InitiateMDMAppleSSOCallback(ctx context.Context, auth fleet.Auth) string {
	// skipauth: No authorization check needed due to implementation
	// returning only license error.
	svc.authz.SkipAuthorization(ctx)

	return apple_mdm.FleetUISSOCallbackPath + "?error=true"
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/manual_enrollment_profile
////////////////////////////////////////////////////////////////////////////////

type getManualEnrollmentProfileRequest struct{}

func getManualEnrollmentProfileEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	profile, err := svc.GetMDMManualEnrollmentProfile(ctx)
	if err != nil {
		return getDeviceMDMManualEnrollProfileResponse{Err: err}, nil
	}

	// Using this type to keep code DRY as it already has all the functionality we need.
	return getDeviceMDMManualEnrollProfileResponse{Profile: profile}, nil
}

func (svc *Service) GetMDMManualEnrollmentProfile(ctx context.Context) ([]byte, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// FileVault-related free version implementation
////////////////////////////////////////////////////////////////////////////////

func (svc *Service) MDMAppleEnableFileVaultAndEscrow(ctx context.Context, teamID *uint) error {
	return fleet.ErrMissingLicense
}

func (svc *Service) MDMAppleDisableFileVaultAndEscrow(ctx context.Context, teamID *uint) error {
	return fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Implementation of nanomdm's CheckinAndCommandService interface
////////////////////////////////////////////////////////////////////////////////

type MDMAppleCheckinAndCommandService struct {
	ds        fleet.Datastore
	logger    kitlog.Logger
	commander *apple_mdm.MDMAppleCommander
}

func NewMDMAppleCheckinAndCommandService(ds fleet.Datastore, commander *apple_mdm.MDMAppleCommander, logger kitlog.Logger) *MDMAppleCheckinAndCommandService {
	return &MDMAppleCheckinAndCommandService{ds: ds, commander: commander, logger: logger}
}

// Authenticate handles MDM [Authenticate][1] requests.
//
// This method is executed after the request has been handled by nanomdm, note
// that at this point you can't send any commands to the device yet because we
// haven't received a token, nor a PushMagic.
//
// We use it to perform post-enrollment tasks such as creating a host record,
// adding activities to the log, etc.
//
// [1]: https://developer.apple.com/documentation/devicemanagement/authenticate
func (svc *MDMAppleCheckinAndCommandService) Authenticate(r *mdm.Request, m *mdm.Authenticate) error {
	host := fleet.MDMAppleHostDetails{}
	host.SerialNumber = m.SerialNumber
	host.UDID = m.UDID
	host.Model = m.Model
	if err := svc.ds.IngestMDMAppleDeviceFromCheckin(r.Context, host); err != nil {
		return ctxerr.Wrap(r.Context, err, "ingesting device in Authenticate message")
	}
	if err := svc.ds.ResetMDMAppleEnrollment(r.Context, host.UDID); err != nil {
		return ctxerr.Wrap(r.Context, err, "resetting nano enrollment info in Authenticate message")
	}
	info, err := svc.ds.GetHostMDMCheckinInfo(r.Context, m.Enrollment.UDID)
	if err != nil {
		return ctxerr.Wrap(r.Context, err, "getting checkin info in Authenticate message")
	}
	return svc.ds.NewActivity(r.Context, nil, &fleet.ActivityTypeMDMEnrolled{
		HostSerial:       info.HardwareSerial,
		HostDisplayName:  info.DisplayName,
		InstalledFromDEP: info.DEPAssignedToFleet,
		MDMPlatform:      fleet.MDMPlatformApple,
	})
}

// TokenUpdate handles MDM [TokenUpdate][1] requests.
//
// This method is executed after the request has been handled by nanomdm.
//
// [1]: https://developer.apple.com/documentation/devicemanagement/token_update
func (svc *MDMAppleCheckinAndCommandService) TokenUpdate(r *mdm.Request, m *mdm.TokenUpdate) error {
	nanoEnroll, err := svc.ds.GetNanoMDMEnrollment(r.Context, r.ID)
	if err != nil {
		return err
	}
	if nanoEnroll != nil && nanoEnroll.Enabled &&
		nanoEnroll.Type == "Device" && nanoEnroll.TokenUpdateTally == 1 {
		// device is enrolled for the first time, not a token update
		if err := svc.ds.BulkSetPendingMDMHostProfiles(r.Context, nil, nil, nil, []string{r.ID}); err != nil {
			return err
		}

		info, err := svc.ds.GetHostMDMCheckinInfo(r.Context, m.Enrollment.UDID)
		if err != nil {
			return err
		}

		var tmID *uint
		if info.TeamID != 0 {
			tmID = &info.TeamID
		}

		// TODO: improve this to not enqueue the job if a host that is
		// assigned in ABM is manually enrolling for some reason.
		if info.DEPAssignedToFleet || info.InstalledFromDEP {
			svc.logger.Log("info", "queueing post-enroll task for newly enrolled DEP device", "host_uuid", r.ID)
			if err := worker.QueueAppleMDMJob(
				r.Context,
				svc.ds,
				svc.logger,
				worker.AppleMDMPostDEPEnrollmentTask,
				r.ID,
				tmID,
				r.Params[mobileconfig.FleetEnrollReferenceKey],
			); err != nil {
				return ctxerr.Wrap(r.Context, err, "queue DEP post-enroll task")
			}
		}

		// manual MDM enrollments that are not fleet-enrolled yet
		if !info.InstalledFromDEP && !info.OsqueryEnrolled {
			if err := worker.QueueAppleMDMJob(
				r.Context,
				svc.ds,
				svc.logger,
				worker.AppleMDMPostManualEnrollmentTask,
				r.ID,
				tmID,
				r.Params[mobileconfig.FleetEnrollReferenceKey],
			); err != nil {
				return ctxerr.Wrap(r.Context, err, "queue manual post-enroll task")
			}
		}
	}
	return nil
}

// CheckOut handles MDM [CheckOut][1] requests.
//
// This method is executed after the request has been handled by nanomdm, note
// that this message is sent on a best-effort basis, don't rely exclusively on
// it.
//
// [1]: https://developer.apple.com/documentation/devicemanagement/check_out
func (svc *MDMAppleCheckinAndCommandService) CheckOut(r *mdm.Request, m *mdm.CheckOut) error {
	info, err := svc.ds.GetHostMDMCheckinInfo(r.Context, m.Enrollment.UDID)
	if err != nil {
		return err
	}

	if err := svc.ds.UpdateHostTablesOnMDMUnenroll(r.Context, m.UDID); err != nil {
		return err
	}
	return svc.ds.NewActivity(r.Context, nil, &fleet.ActivityTypeMDMUnenrolled{
		HostSerial:       info.HardwareSerial,
		HostDisplayName:  info.DisplayName,
		InstalledFromDEP: info.InstalledFromDEP,
	})
}

// SetBootstrapToken handles MDM [SetBootstrapToken][1] requests.
//
// This method is executed after the request has been handled by nanomdm.
//
// [1]: https://developer.apple.com/documentation/devicemanagement/set_bootstrap_token
func (svc *MDMAppleCheckinAndCommandService) SetBootstrapToken(*mdm.Request, *mdm.SetBootstrapToken) error {
	return nil
}

// GetBootstrapToken handles MDM [GetBootstrapToken][1] requests.
//
// This method is executed after the request has been handled by nanomdm.
//
// [1]: https://developer.apple.com/documentation/devicemanagement/get_bootstrap_token
func (svc *MDMAppleCheckinAndCommandService) GetBootstrapToken(*mdm.Request, *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	return nil, nil
}

// UserAuthenticate handles MDM [UserAuthenticate][1] requests.
//
// This method is executed after the request has been handled by nanomdm.
//
// [1]: https://developer.apple.com/documentation/devicemanagement/userauthenticate
func (svc *MDMAppleCheckinAndCommandService) UserAuthenticate(*mdm.Request, *mdm.UserAuthenticate) ([]byte, error) {
	return nil, nil
}

// DeclarativeManagement handles MDM [DeclarativeManagement][1] requests.
//
// This method is executed after the request has been handled by nanomdm.
//
// [1]: https://developer.apple.com/documentation/devicemanagement/declarative_management_checkin
func (svc *MDMAppleCheckinAndCommandService) DeclarativeManagement(*mdm.Request, *mdm.DeclarativeManagement) ([]byte, error) {
	return nil, nil
}

// CommandAndReportResults handles MDM [Commands and Queries][1].
//
// This method is executed after the request has been handled by nanomdm.
//
// [1]: https://developer.apple.com/documentation/devicemanagement/commands_and_queries
func (svc *MDMAppleCheckinAndCommandService) CommandAndReportResults(r *mdm.Request, cmdResult *mdm.CommandResults) (*mdm.Command, error) {
	if cmdResult.Status == "Idle" {
		// macOS hosts are considered unlocked if they are online any time
		// after they have been unlocked. If the host has been seen after a
		// successful unlock, take the opportunity and update the value in the
		// db as well.
		//
		// TODO: sanity check if this approach is still valid after we implement wipe
		if err := svc.ds.CleanMacOSMDMLock(r.Context, cmdResult.UDID); err != nil {
			return nil, ctxerr.Wrap(r.Context, err, "cleaning macOS host lock/wipe status")
		}
	}

	// We explicitly get the request type because it comes empty. There's a
	// RequestType field in the struct, but it's used when a mdm.Command is
	// issued.
	requestType, err := svc.ds.GetMDMAppleCommandRequestType(r.Context, cmdResult.CommandUUID)
	if err != nil {
		return nil, ctxerr.Wrap(r.Context, err, "command service")
	}

	switch requestType {
	case "InstallProfile":
		return nil, apple_mdm.HandleHostMDMProfileInstallResult(
			r.Context,
			svc.ds,
			cmdResult.UDID,
			cmdResult.CommandUUID,
			mdmAppleDeliveryStatusFromCommandStatus(cmdResult.Status),
			apple_mdm.FmtErrorChain(cmdResult.ErrorChain),
		)
	case "RemoveProfile":
		return nil, svc.ds.UpdateOrDeleteHostMDMAppleProfile(r.Context, &fleet.HostMDMAppleProfile{
			CommandUUID:   cmdResult.CommandUUID,
			HostUUID:      cmdResult.UDID,
			Status:        mdmAppleDeliveryStatusFromCommandStatus(cmdResult.Status),
			Detail:        apple_mdm.FmtErrorChain(cmdResult.ErrorChain),
			OperationType: fleet.MDMOperationTypeRemove,
		})
	}
	return nil, nil
}

// mdmAppleDeliveryStatusFromCommandStatus converts a MDM command status to a
// fleet.MDMAppleDeliveryStatus.
//
// NOTE: this mapping does not include all
// possible delivery statuses (e.g., verified status is not included) is intended to
// only be used in the context of CommandAndReportResults in the MDMAppleCheckinAndCommandService.
// Extra care should be taken before using this function in other contexts.
func mdmAppleDeliveryStatusFromCommandStatus(cmdStatus string) *fleet.MDMDeliveryStatus {
	switch cmdStatus {
	case fleet.MDMAppleStatusAcknowledged:
		return &fleet.MDMDeliveryVerifying
	case fleet.MDMAppleStatusError, fleet.MDMAppleStatusCommandFormatError:
		return &fleet.MDMDeliveryFailed
	case fleet.MDMAppleStatusIdle, fleet.MDMAppleStatusNotNow:
		return &fleet.MDMDeliveryPending
	default:
		return nil
	}
}

// ensureFleetdConfig ensures there's a fleetd configuration profile in
// mdm_apple_configuration_profiles for each team and for "no team"
//
// We try our best to use each team's secret but we default to creating a
// profile with the global enroll secret if the team doesn't have any enroll
// secrets.
//
// This profile will be installed to all hosts in the team (or "no team",) but it
// will only be used by hosts that have a fleetd installation without an enroll
// secret and fleet URL (mainly DEP enrolled hosts).
func ensureFleetdConfig(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger) error {
	appCfg, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching app config")
	}

	enrollSecrets, err := ds.AggregateEnrollSecretPerTeam(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting enroll secrets aggregates")
	}

	globalSecret := ""
	for _, es := range enrollSecrets {
		if es.TeamID == nil {
			globalSecret = es.Secret
		}
	}

	var profiles []*fleet.MDMAppleConfigProfile
	for _, es := range enrollSecrets {
		if es.Secret == "" {
			var msg string
			if es.TeamID != nil {
				msg += fmt.Sprintf("team_id %d doesn't have an enroll secret, ", *es.TeamID)
			}
			if globalSecret == "" {
				logger.Log("err", msg+"no global enroll secret found, skipping the creation of a com.fleetdm.fleetd.config profile")
				continue
			}
			logger.Log("err", msg+"using a global enroll secret for com.fleetdm.fleetd.config profile")
			es.Secret = globalSecret
		}

		var contents bytes.Buffer
		params := mobileconfig.FleetdProfileOptions{
			EnrollSecret: es.Secret,
			ServerURL:    appCfg.ServerSettings.ServerURL,
			PayloadType:  mobileconfig.FleetdConfigPayloadIdentifier,
			PayloadName:  mdm_types.FleetdConfigProfileName,
		}

		if err := mobileconfig.FleetdProfileTemplate.Execute(&contents, params); err != nil {
			return ctxerr.Wrap(ctx, err, "executing fleetd config template")
		}

		cp, err := fleet.NewMDMAppleConfigProfile(contents.Bytes(), es.TeamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building configuration profile")
		}

		profiles = append(profiles, cp)

	}

	if err := ds.BulkUpsertMDMAppleConfigProfiles(ctx, profiles); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk-upserting configuration profiles")
	}

	return nil
}

func ReconcileAppleProfiles(
	ctx context.Context,
	ds fleet.Datastore,
	commander *apple_mdm.MDMAppleCommander,
	logger kitlog.Logger,
) error {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return fmt.Errorf("reading app config: %w", err)
	}
	if !appConfig.MDM.EnabledAndConfigured {
		return nil
	}
	if err := ensureFleetdConfig(ctx, ds, logger); err != nil {
		logger.Log("err", "unable to ensure a fleetd configuration profiles are in place", "details", err)
	}

	// retrieve the profiles to install/remove.
	toInstall, err := ds.ListMDMAppleProfilesToInstall(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting profiles to install")
	}
	toRemove, err := ds.ListMDMAppleProfilesToRemove(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting profiles to remove")
	}

	// Perform aggregations to support all the operations we need to do

	// toGetContents contains the UUIDs of all the profiles from which we
	// need to retrieve contents. Since the previous query returns one row
	// per host, it would be too expensive to retrieve the profile contents
	// there, so we make another request. Using a map to deduplicate.
	toGetContents := make(map[string]bool)

	// hostProfiles tracks each host_mdm_apple_profile we need to upsert
	// with the new status, operation_type, etc.
	hostProfiles := make([]*fleet.MDMAppleBulkUpsertHostProfilePayload, 0, len(toInstall)+len(toRemove))

	// profileIntersection tracks profilesToAdd ∩ profilesToRemove, this is used to avoid:
	//
	// - Sending a RemoveProfile followed by an InstallProfile for a
	// profile with an identifier that's already installed, which can cause
	// racy behaviors.
	// - Sending a InstallProfile command for a profile that's exactly the
	// same as the one installed. Customers have reported that sending the
	// command causes unwanted behavior.
	profileIntersection := apple_mdm.NewProfileBimap()
	profileIntersection.IntersectByIdentifierAndHostUUID(toInstall, toRemove)

	// hostProfilesToCleanup is used to track profiles that should be removed
	// from the database directly without having to issue a RemoveProfile
	// command.
	hostProfilesToCleanup := []*fleet.MDMAppleProfilePayload{}

	// install/removeTargets are maps from profileUUID -> command uuid and host
	// UUIDs as the underlying MDM services are optimized to send one command to
	// multiple hosts at the same time. Note that the same command uuid is used
	// for all hosts in a given install/remove target operation.
	type cmdTarget struct {
		cmdUUID   string
		profIdent string
		hostUUIDs []string
	}
	installTargets, removeTargets := make(map[string]*cmdTarget), make(map[string]*cmdTarget)
	for _, p := range toInstall {
		if pp, ok := profileIntersection.GetMatchingProfileInCurrentState(p); ok {
			// if the profile was in any other status than `failed`
			// and the checksums match (the profiles are exactly
			// the same) we don't send another InstallProfile
			// command.
			if pp.Status != &fleet.MDMDeliveryFailed && bytes.Equal(pp.Checksum, p.Checksum) {
				hostProfiles = append(hostProfiles, &fleet.MDMAppleBulkUpsertHostProfilePayload{
					ProfileUUID:       p.ProfileUUID,
					HostUUID:          p.HostUUID,
					ProfileIdentifier: p.ProfileIdentifier,
					ProfileName:       p.ProfileName,
					Checksum:          p.Checksum,
					OperationType:     pp.OperationType,
					Status:            pp.Status,
					CommandUUID:       pp.CommandUUID,
					Detail:            pp.Detail,
				})
				continue
			}
		}
		toGetContents[p.ProfileUUID] = true

		target := installTargets[p.ProfileUUID]
		if target == nil {
			target = &cmdTarget{
				cmdUUID:   uuid.New().String(),
				profIdent: p.ProfileIdentifier,
			}
			installTargets[p.ProfileUUID] = target
		}
		target.hostUUIDs = append(target.hostUUIDs, p.HostUUID)

		hostProfiles = append(hostProfiles, &fleet.MDMAppleBulkUpsertHostProfilePayload{
			ProfileUUID:       p.ProfileUUID,
			HostUUID:          p.HostUUID,
			OperationType:     fleet.MDMOperationTypeInstall,
			Status:            &fleet.MDMDeliveryPending,
			CommandUUID:       target.cmdUUID,
			ProfileIdentifier: p.ProfileIdentifier,
			ProfileName:       p.ProfileName,
			Checksum:          p.Checksum,
		})
	}

	for _, p := range toRemove {
		if _, ok := profileIntersection.GetMatchingProfileInDesiredState(p); ok {
			hostProfilesToCleanup = append(hostProfilesToCleanup, p)
			continue
		}

		target := removeTargets[p.ProfileUUID]
		if target == nil {
			target = &cmdTarget{
				cmdUUID:   uuid.New().String(),
				profIdent: p.ProfileIdentifier,
			}
			removeTargets[p.ProfileUUID] = target
		}
		target.hostUUIDs = append(target.hostUUIDs, p.HostUUID)

		hostProfiles = append(hostProfiles, &fleet.MDMAppleBulkUpsertHostProfilePayload{
			ProfileUUID:       p.ProfileUUID,
			HostUUID:          p.HostUUID,
			OperationType:     fleet.MDMOperationTypeRemove,
			Status:            &fleet.MDMDeliveryPending,
			CommandUUID:       target.cmdUUID,
			ProfileIdentifier: p.ProfileIdentifier,
			ProfileName:       p.ProfileName,
			Checksum:          p.Checksum,
		})
	}

	// delete all profiles that have a matching identifier to be installed.
	// This is to prevent sending both a `RemoveProfile` and an
	// `InstallProfile` for the same identifier, which can cause race
	// conditions. It's better to "update" the profile by sending a single
	// `InstallProfile` command.
	if err := ds.BulkDeleteMDMAppleHostsConfigProfiles(ctx, hostProfilesToCleanup); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting profiles that didn't change")
	}

	// First update all the profiles in the database before sending the
	// commands, this prevents race conditions where we could get a
	// response from the device before we set its status as 'pending'
	//
	// We'll do another pass at the end to revert any changes for failed
	// delivieries.
	if err := ds.BulkUpsertMDMAppleHostProfiles(ctx, hostProfiles); err != nil {
		return ctxerr.Wrap(ctx, err, "updating host profiles")
	}

	// Grab the contents of all the profiles we need to install
	profileUUIDs := make([]string, 0, len(toGetContents))
	for pUUID := range toGetContents {
		profileUUIDs = append(profileUUIDs, pUUID)
	}
	profileContents, err := ds.GetMDMAppleProfilesContents(ctx, profileUUIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get profile contents")
	}

	type remoteResult struct {
		Err     error
		CmdUUID string
	}

	// Send the install/remove commands for each profile.
	var wgProd, wgCons sync.WaitGroup
	ch := make(chan remoteResult)

	execCmd := func(profUUID string, target *cmdTarget, op fleet.MDMOperationType) {
		defer wgProd.Done()

		var err error
		switch op {
		case fleet.MDMOperationTypeInstall:
			err = commander.InstallProfile(ctx, target.hostUUIDs, profileContents[profUUID], target.cmdUUID)
		case fleet.MDMOperationTypeRemove:
			err = commander.RemoveProfile(ctx, target.hostUUIDs, target.profIdent, target.cmdUUID)
		}

		var e *apple_mdm.APNSDeliveryError
		switch {
		case errors.As(err, &e):
			level.Debug(logger).Log("err", "sending push notifications, profiles still enqueued", "details", err)
		case err != nil:
			level.Error(logger).Log("err", fmt.Sprintf("enqueue command to %s profiles", op), "details", err)
			ch <- remoteResult{err, target.cmdUUID}
		}
	}
	for profUUID, target := range installTargets {
		wgProd.Add(1)
		go execCmd(profUUID, target, fleet.MDMOperationTypeInstall)
	}
	for profUUID, target := range removeTargets {
		wgProd.Add(1)
		go execCmd(profUUID, target, fleet.MDMOperationTypeRemove)
	}

	// index the host profiles by cmdUUID, for ease of error processing in the
	// consumer goroutine below.
	hostProfsByCmdUUID := make(map[string][]*fleet.MDMAppleBulkUpsertHostProfilePayload, len(installTargets)+len(removeTargets))
	for _, hp := range hostProfiles {
		hostProfsByCmdUUID[hp.CommandUUID] = append(hostProfsByCmdUUID[hp.CommandUUID], hp)
	}

	// Grab all the failed deliveries and update the status so they're picked up
	// again in the next run.
	//
	// Note that if the APNs push failed we won't try again, as the command was
	// successfully enqueued, this is only to account for internal errors like DB
	// failures.
	failed := []*fleet.MDMAppleBulkUpsertHostProfilePayload{}
	wgCons.Add(1)
	go func() {
		defer wgCons.Done()

		for resp := range ch {
			hostProfs := hostProfsByCmdUUID[resp.CmdUUID]
			for _, hp := range hostProfs {
				// clear the command as it failed to enqueue, will need to emit a new command
				hp.CommandUUID = ""
				// set status to nil so it is retried on the next cron run
				hp.Status = nil
				failed = append(failed, hp)
			}
		}
	}()

	wgProd.Wait()
	close(ch) // done sending at this point, this triggers end of for loop in consumer
	wgCons.Wait()

	if err := ds.BulkUpsertMDMAppleHostProfiles(ctx, failed); err != nil {
		return ctxerr.Wrap(ctx, err, "reverting status of failed profiles")
	}

	return nil
}

func (svc *Service) maybeRestorePendingDEPHost(ctx context.Context, host *fleet.Host) error {
	if host.Platform != "darwin" {
		return nil
	}

	license, ok := license.FromContext(ctx)
	if !ok {
		return ctxerr.New(ctx, "maybe restore pending DEP host: missing license")
	} else if license.Tier != fleet.TierPremium {
		// only premium tier supports DEP so nothing more to do
		return nil
	}

	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "maybe restore pending DEP host: get app config")
	} else if !ac.MDM.AppleBMEnabledAndConfigured {
		// if ABM is not enabled and configured, nothing more to do
		return nil
	}

	dep, err := svc.ds.GetHostDEPAssignment(ctx, host.ID)
	switch {
	case err != nil && !fleet.IsNotFound(err):
		return ctxerr.Wrap(ctx, err, "maybe restore pending DEP host: get host dep assignment")
	case dep != nil && dep.DeletedAt == nil:
		return svc.restorePendingDEPHost(ctx, host, ac)
	default:
		// no DEP assignment was found or the DEP assignment was deleted in ABM
		// so nothing more to do
	}

	return nil
}

func (svc *Service) restorePendingDEPHost(ctx context.Context, host *fleet.Host, appCfg *fleet.AppConfig) error {
	tmID, err := svc.getConfigAppleBMDefaultTeamID(ctx, appCfg)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "restore pending dep host")
	}
	host.TeamID = tmID

	if err := svc.ds.RestoreMDMApplePendingDEPHost(ctx, host); err != nil {
		return ctxerr.Wrap(ctx, err, "restore pending dep host")
	}

	if err := worker.QueueMacosSetupAssistantJob(ctx, svc.ds, svc.logger,
		worker.MacosSetupAssistantHostsTransferred, tmID, host.HardwareSerial); err != nil {
		return ctxerr.Wrap(ctx, err, "restore pending dep host")
	}

	return nil
}

func (svc *Service) getConfigAppleBMDefaultTeamID(ctx context.Context, appCfg *fleet.AppConfig) (*uint, error) {
	var tmID *uint
	if name := appCfg.MDM.AppleBMDefaultTeam; name != "" {
		team, err := svc.ds.TeamByName(ctx, name)
		switch {
		case fleet.IsNotFound(err):
			level.Debug(svc.logger).Log(
				"msg",
				"unable to find default team assigned in config, mdm devices won't be assigned to a team",
				"team_name",
				name,
			)
			return nil, nil
		case err != nil:
			return nil, ctxerr.Wrap(ctx, err, "get default team for mdm devices")
		case team != nil:
			tmID = &team.ID
		}
	}

	return tmID, nil
}

// scepCertRenewalThresholdDays defines the number of days before a SCEP
// certificate must be renewed.
const scepCertRenewalThresholdDays = 30

// maxCertsRenewalPerRun specifies the maximum number of certificates to renew
// in a single cron run.
//
// Assuming that the cron runs every hour, we'll enqueue 24,000 renewals per
// day, and we have room for 24,000 * scepCertRenewalThresholdDays total
// renewals.
//
// For a default of 30 days as a threshold this gives us room for a fleet of
// 720,000 devices expiring at the same time.
const maxCertsRenewalPerRun = 100

func RenewSCEPCertificates(
	ctx context.Context,
	logger kitlog.Logger,
	ds fleet.Datastore,
	config *config.FleetConfig,
	commander *apple_mdm.MDMAppleCommander,
) error {
	if !config.MDM.IsAppleSCEPSet() {
		logger.Log("inf", "skipping renewal of macOS SCEP certificates as MDM is not fully configured")
		return nil
	}

	if commander == nil {
		logger.Log("inf", "skipping renewal of macOS SCEP certificates as apple_mdm.MDMAppleCommander was not provided")
		return nil
	}

	// for each hash, grab the host that uses it as its identity certificate
	certAssociations, err := ds.GetHostCertAssociationsToExpire(ctx, scepCertRenewalThresholdDays, maxCertsRenewalPerRun)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting host cert associations")
	}

	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting AppConfig")
	}

	mdmPushCertTopic, err := config.MDM.AppleAPNsTopic()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting certificate topic")
	}

	// assocsWithRefs stores hosts that have enrollment references on their
	// enrollment profiles. This is the case for ADE-enrolled hosts using
	// SSO to authenticate.
	assocsWithRefs := []fleet.SCEPIdentityAssociation{}
	// assocsWithoutRefs stores hosts that don't have an enrollment
	// reference in their enrollment profile.
	assocsWithoutRefs := []fleet.SCEPIdentityAssociation{}
	for _, assoc := range certAssociations {
		if assoc.EnrollReference != "" {
			assocsWithRefs = append(assocsWithRefs, assoc)
			continue
		}
		assocsWithoutRefs = append(assocsWithoutRefs, assoc)
	}

	// send a single command for all the hosts without references.
	if len(assocsWithoutRefs) > 0 {
		profile, err := apple_mdm.GenerateEnrollmentProfileMobileconfig(
			appConfig.OrgInfo.OrgName,
			appConfig.ServerSettings.ServerURL,
			config.MDM.AppleSCEPChallenge,
			mdmPushCertTopic,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "generating enrollment profile for hosts without enroll reference")
		}

		cmdUUID := uuid.NewString()
		var uuids []string
		for _, assoc := range assocsWithoutRefs {
			uuids = append(uuids, assoc.HostUUID)
			assoc.RenewCommandUUID = cmdUUID
		}

		if err := commander.InstallProfile(ctx, uuids, profile, cmdUUID); err != nil {
			return ctxerr.Wrapf(ctx, err, "sending InstallProfile command for hosts %s", assocsWithoutRefs)
		}

		if err := ds.SetCommandForPendingSCEPRenewal(ctx, assocsWithoutRefs, cmdUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "setting pending command associations")
		}
	}

	// send individual commands for each host with a reference
	for _, assoc := range assocsWithRefs {
		enrollURL, err := apple_mdm.AddEnrollmentRefToFleetURL(appConfig.ServerSettings.ServerURL, assoc.EnrollReference)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "adding reference to fleet URL")
		}

		profile, err := apple_mdm.GenerateEnrollmentProfileMobileconfig(
			appConfig.OrgInfo.OrgName,
			enrollURL,
			config.MDM.AppleSCEPChallenge,
			mdmPushCertTopic,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "generating enrollment profile for hosts with enroll reference")
		}
		cmdUUID := uuid.NewString()
		if err := commander.InstallProfile(ctx, []string{assoc.HostUUID}, profile, cmdUUID); err != nil {
			return ctxerr.Wrapf(ctx, err, "sending InstallProfile command for hosts %s", assocsWithRefs)
		}

		if err := ds.SetCommandForPendingSCEPRenewal(ctx, []fleet.SCEPIdentityAssociation{assoc}, cmdUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "setting pending command associations")
		}
	}

	return nil
}
