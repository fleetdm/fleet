package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
)

// ///////////////////////////////////////////////////////////////////////////
// List enforcement profiles
// ///////////////////////////////////////////////////////////////////////////

type listWindowsEnforcementProfilesRequest struct {
	TeamID      *uint             `query:"team_id,optional"`
	ListOptions fleet.ListOptions `url:"list_options"`
}

type listWindowsEnforcementProfilesResponse struct {
	Meta     *fleet.PaginationMetadata          `json:"meta"`
	Profiles []*fleet.WindowsEnforcementProfile `json:"profiles"`
	Err      error                              `json:"error,omitempty"`
}

func (r listWindowsEnforcementProfilesResponse) Error() error { return r.Err }

func listWindowsEnforcementProfilesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*listWindowsEnforcementProfilesRequest)
	profiles, err := svc.ListWindowsEnforcementProfiles(ctx, req.TeamID)
	if err != nil {
		return &listWindowsEnforcementProfilesResponse{Err: err}, nil
	}
	res := listWindowsEnforcementProfilesResponse{Profiles: profiles}
	if profiles == nil {
		res.Profiles = []*fleet.WindowsEnforcementProfile{}
	}
	return &res, nil
}

func (svc *Service) ListWindowsEnforcementProfiles(ctx context.Context, teamID *uint) ([]*fleet.WindowsEnforcementProfile, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	return svc.ds.ListWindowsEnforcementProfiles(ctx, teamID)
}

// ///////////////////////////////////////////////////////////////////////////
// Upload (create) enforcement profile
// ///////////////////////////////////////////////////////////////////////////

type uploadWindowsEnforcementProfileRequest struct {
	TeamID  uint
	Profile *multipart.FileHeader
}

type uploadWindowsEnforcementProfileResponse struct {
	ProfileUUID string `json:"profile_uuid"`
	Err         error  `json:"error,omitempty"`
}

func (r uploadWindowsEnforcementProfileResponse) Error() error { return r.Err }

func (uploadWindowsEnforcementProfileRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	decoded := uploadWindowsEnforcementProfileRequest{}
	err := parseMultipartForm(ctx, r, platform_http.MaxMultipartFormSize)
	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}
	val := r.MultipartForm.Value["team_id"]
	if len(val) > 0 {
		teamID, err := strconv.ParseUint(val[0], 10, 64)
		if err != nil {
			return nil, &fleet.BadRequestError{Message: "invalid team_id"}
		}
		decoded.TeamID = uint(teamID)
	}
	fhs, ok := r.MultipartForm.File["profile"]
	if !ok || len(fhs) < 1 {
		return nil, &fleet.BadRequestError{Message: "profile file is required"}
	}
	decoded.Profile = fhs[0]
	return &decoded, nil
}

func uploadWindowsEnforcementProfileEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*uploadWindowsEnforcementProfileRequest)

	ff, err := req.Profile.Open()
	if err != nil {
		return &uploadWindowsEnforcementProfileResponse{Err: err}, nil
	}
	defer ff.Close()

	data, err := io.ReadAll(ff)
	if err != nil {
		return &uploadWindowsEnforcementProfileResponse{Err: err}, nil
	}

	fileExt := filepath.Ext(req.Profile.Filename)
	profileName := strings.TrimSuffix(filepath.Base(req.Profile.Filename), fileExt)

	// Validate file extension
	ext := strings.ToLower(fileExt)
	if ext != ".yml" && ext != ".yaml" && ext != ".json" {
		return &uploadWindowsEnforcementProfileResponse{
			Err: &fleet.BadRequestError{Message: "Only .yml, .yaml, and .json files are supported"},
		}, nil
	}

	profile, err := svc.NewWindowsEnforcementProfile(ctx, req.TeamID, profileName, data)
	if err != nil {
		return &uploadWindowsEnforcementProfileResponse{Err: err}, nil
	}
	return &uploadWindowsEnforcementProfileResponse{ProfileUUID: profile.ProfileUUID}, nil
}

func (svc *Service) NewWindowsEnforcementProfile(ctx context.Context, teamID uint, name string, rawPolicy []byte) (*fleet.WindowsEnforcementProfile, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: &teamID}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	profiles := []*fleet.WindowsEnforcementProfile{
		{Name: name, RawPolicy: rawPolicy},
	}

	// Fetch existing profiles and merge the new one in.
	existing, err := svc.ds.ListWindowsEnforcementProfiles(ctx, &teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing existing enforcement profiles")
	}

	// Check for duplicate name and replace if found, otherwise append.
	found := false
	for i, ep := range existing {
		if ep.Name == name {
			existing[i].RawPolicy = rawPolicy
			found = true
			break
		}
	}
	if !found {
		existing = append(existing, profiles[0])
	}

	if err := svc.ds.BatchSetWindowsEnforcementProfiles(ctx, &teamID, existing); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "batch setting enforcement profiles")
	}

	// Get the saved profile to return its UUID.
	saved, err := svc.ds.ListWindowsEnforcementProfiles(ctx, &teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing saved enforcement profiles")
	}
	for _, p := range saved {
		if p.Name == name {
			return p, nil
		}
	}
	return nil, ctxerr.New(ctx, "enforcement profile not found after save")
}

// ///////////////////////////////////////////////////////////////////////////
// Get enforcement profile
// ///////////////////////////////////////////////////////////////////////////

type getWindowsEnforcementProfileRequest struct {
	ProfileUUID string `url:"profile_uuid"`
	Alt         string `query:"alt,optional"`
}

type getWindowsEnforcementProfileResponse struct {
	*fleet.WindowsEnforcementProfile
	Err error `json:"error,omitempty"`
}

func (r getWindowsEnforcementProfileResponse) Error() error { return r.Err }

func getWindowsEnforcementProfileEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getWindowsEnforcementProfileRequest)
	profile, err := svc.GetWindowsEnforcementProfile(ctx, req.ProfileUUID)
	if err != nil {
		return &getWindowsEnforcementProfileResponse{Err: err}, nil
	}

	if req.Alt == "media" {
		return downloadFileResponse{
			content:     profile.RawPolicy,
			contentType: "application/octet-stream",
			filename:    fmt.Sprintf("%s_%s.yml", time.Now().Format("2006-01-02"), profile.Name),
		}, nil
	}

	return &getWindowsEnforcementProfileResponse{WindowsEnforcementProfile: profile}, nil
}

func (svc *Service) GetWindowsEnforcementProfile(ctx context.Context, profileUUID string) (*fleet.WindowsEnforcementProfile, error) {
	// Broad read check first.
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	profile, err := svc.ds.GetWindowsEnforcementProfile(ctx, profileUUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	// Team-specific check.
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: profile.TeamID}, fleet.ActionRead); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	return profile, nil
}

// ///////////////////////////////////////////////////////////////////////////
// Delete enforcement profile
// ///////////////////////////////////////////////////////////////////////////

type deleteWindowsEnforcementProfileRequest struct {
	ProfileUUID string `url:"profile_uuid"`
}

type deleteWindowsEnforcementProfileResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteWindowsEnforcementProfileResponse) Error() error { return r.Err }

func deleteWindowsEnforcementProfileEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*deleteWindowsEnforcementProfileRequest)
	err := svc.DeleteWindowsEnforcementProfile(ctx, req.ProfileUUID)
	if err != nil {
		return &deleteWindowsEnforcementProfileResponse{Err: err}, nil
	}
	return &deleteWindowsEnforcementProfileResponse{}, nil
}

func (svc *Service) DeleteWindowsEnforcementProfile(ctx context.Context, profileUUID string) error {
	// Broad read check first.
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return ctxerr.Wrap(ctx, err)
	}
	profile, err := svc.ds.GetWindowsEnforcementProfile(ctx, profileUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err)
	}
	// Team-specific write check.
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: profile.TeamID}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err)
	}
	if err := svc.ds.DeleteWindowsEnforcementProfile(ctx, profileUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting enforcement profile")
	}
	return nil
}

// ReconcileWindowsEnforcement is called by the windows_enforcement cron
// schedule. It diffs the desired enforcement state against the current host
// enforcement state and creates pending install/remove records.
func ReconcileWindowsEnforcement(ctx context.Context, ds fleet.Datastore, logger *slog.Logger) error {
	// Get profiles to install (desired minus current).
	toInstall, err := ds.ListWindowsEnforcementToInstall(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting enforcement profiles to install")
	}

	// Get profiles to remove (current minus desired).
	toRemove, err := ds.ListWindowsEnforcementToRemove(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting enforcement profiles to remove")
	}

	if len(toInstall) == 0 && len(toRemove) == 0 {
		return nil
	}

	// Build the bulk upsert payload.
	pending := fleet.MDMDeliveryPending
	payload := make([]*fleet.HostWindowsEnforcement, 0, len(toInstall)+len(toRemove))

	for _, p := range toInstall {
		payload = append(payload, &fleet.HostWindowsEnforcement{
			HostUUID:      p.HostUUID,
			ProfileUUID:   p.ProfileUUID,
			Name:          p.Name,
			Status:        &pending,
			OperationType: fleet.MDMOperationTypeInstall,
		})
	}

	for _, p := range toRemove {
		payload = append(payload, &fleet.HostWindowsEnforcement{
			HostUUID:      p.HostUUID,
			ProfileUUID:   p.ProfileUUID,
			Status:        &pending,
			OperationType: fleet.MDMOperationTypeRemove,
		})
	}

	if err := ds.BulkUpsertHostWindowsEnforcement(ctx, payload); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk upserting host enforcement status")
	}

	logger.Info(fmt.Sprintf("reconciled windows enforcement: %d to install, %d to remove",
		len(toInstall), len(toRemove)))

	return nil
}
