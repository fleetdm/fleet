package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/VividCortex/mysqlerr"
	"github.com/docker/go-units"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/groob/plist"
	"github.com/micromdm/micromdm/mdm/appmanifest"
	"github.com/micromdm/nanodep/godep"
	"github.com/micromdm/nanomdm/mdm"
	nanomdm_push "github.com/micromdm/nanomdm/push"
	nanomdm_storage "github.com/micromdm/nanomdm/storage"
)

type createMDMAppleEnrollmentProfileRequest struct {
	Type       fleet.MDMAppleEnrollmentType `json:"type"`
	DEPProfile *json.RawMessage             `json:"dep_profile"`
}

type createMDMAppleEnrollmentProfileResponse struct {
	EnrollmentProfile *fleet.MDMAppleEnrollmentProfile `json:"enrollment_profile"`
	Err               error                            `json:"error,omitempty"`
}

func (r createMDMAppleEnrollmentProfileResponse) error() error { return r.Err }

func createMDMAppleEnrollmentProfilesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*createMDMAppleEnrollmentProfileRequest)

	enrollmentProfile, err := svc.NewMDMAppleEnrollmentProfile(ctx, fleet.MDMAppleEnrollmentProfilePayload{
		Type:       req.Type,
		DEPProfile: req.DEPProfile,
	})
	if err != nil {
		return createMDMAppleEnrollmentProfileResponse{
			Err: err,
		}, nil
	}
	return createMDMAppleEnrollmentProfileResponse{
		EnrollmentProfile: enrollmentProfile,
	}, nil
}

func (svc *Service) NewMDMAppleEnrollmentProfile(ctx context.Context, enrollmentPayload fleet.MDMAppleEnrollmentProfilePayload) (*fleet.MDMAppleEnrollmentProfile, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleEnrollmentProfile{}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	// generate a token for the profile
	enrollmentPayload.Token = uuid.New().String()

	profile, err := svc.ds.NewMDMAppleEnrollmentProfile(ctx, enrollmentPayload)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	if profile.DEPProfile != nil {
		if err := svc.setDEPProfile(ctx, profile, appConfig); err != nil {
			return nil, ctxerr.Wrap(ctx, err)
		}
	}

	enrollmentURL, err := svc.mdmAppleEnrollURL(profile.Token, appConfig)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	profile.EnrollmentURL = enrollmentURL

	return profile, nil
}

func (svc *Service) mdmAppleEnrollURL(token string, appConfig *fleet.AppConfig) (string, error) {
	enrollURL, err := url.Parse(appConfig.ServerSettings.ServerURL)
	if err != nil {
		return "", err
	}
	enrollURL.Path = path.Join(enrollURL.Path, apple_mdm.EnrollPath)
	q := enrollURL.Query()
	q.Set("token", token)
	enrollURL.RawQuery = q.Encode()
	return enrollURL.String(), nil
}

// setDEPProfile define a "DEP profile" on https://mdmenrollment.apple.com and
// sets the returned Profile UUID as the current DEP profile to apply to newly sync DEP devices.
func (svc *Service) setDEPProfile(ctx context.Context, enrollmentProfile *fleet.MDMAppleEnrollmentProfile, appConfig *fleet.AppConfig) error {
	var depProfileRequest godep.Profile
	if err := json.Unmarshal(*enrollmentProfile.DEPProfile, &depProfileRequest); err != nil {
		return ctxerr.Wrap(ctx, err, "invalid DEP profile")
	}

	enrollURL, err := svc.mdmAppleEnrollURL(enrollmentProfile.Token, appConfig)
	if err != nil {
		return fmt.Errorf("generating enrollment URL: %w", err)
	}
	// Override url with Fleet's enroll path (publicly accessible address).
	depProfileRequest.URL = enrollURL

	// If the profile doesn't have a configuration_web_url, use Fleet's
	// enrollURL, otherwise the request for the enrollment profile is
	// submitted as a POST instead of GET.
	if depProfileRequest.ConfigurationWebURL == "" {
		depProfileRequest.ConfigurationWebURL = enrollURL
	}

	depClient := apple_mdm.NewDEPClient(svc.depStorage, svc.ds, svc.logger)
	res, err := depClient.DefineProfile(ctx, apple_mdm.DEPName, &depProfileRequest)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "apple POST /profile request failed")
	}

	if err := svc.depStorage.StoreAssignerProfile(ctx, apple_mdm.DEPName, res.ProfileUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "set profile UUID")
	}
	return nil
}

type listMDMAppleEnrollmentProfilesRequest struct{}

type listMDMAppleEnrollmentProfilesResponse struct {
	EnrollmentProfiles []*fleet.MDMAppleEnrollmentProfile `json:"enrollment_profiles"`
	Err                error                              `json:"error,omitempty"`
}

func (r listMDMAppleEnrollmentProfilesResponse) error() error { return r.Err }

func listMDMAppleEnrollmentsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	enrollmentProfiles, err := svc.ListMDMAppleEnrollmentProfiles(ctx)
	if err != nil {
		return listMDMAppleEnrollmentProfilesResponse{
			Err: err,
		}, nil
	}
	return listMDMAppleEnrollmentProfilesResponse{
		EnrollmentProfiles: enrollmentProfiles,
	}, nil
}

func (svc *Service) ListMDMAppleEnrollmentProfiles(ctx context.Context) ([]*fleet.MDMAppleEnrollmentProfile, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleEnrollmentProfile{}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	enrollments, err := svc.ds.ListMDMAppleEnrollmentProfiles(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	for i := range enrollments {
		enrollURL, err := svc.mdmAppleEnrollURL(enrollments[i].Token, appConfig)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err)
		}
		enrollments[i].EnrollmentURL = enrollURL
	}
	return enrollments, nil
}

type getMDMAppleCommandResultsRequest struct {
	CommandUUID string `query:"command_uuid,optional"`
}

type getMDMAppleCommandResultsResponse struct {
	Results map[string]*fleet.MDMAppleCommandResult `json:"results,omitempty"`
	Err     error                                   `json:"error,omitempty"`
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

func (svc *Service) GetMDMAppleCommandResults(ctx context.Context, commandUUID string) (map[string]*fleet.MDMAppleCommandResult, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleCommandResult{}, fleet.ActionRead); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	results, err := svc.ds.GetMDMAppleCommandResults(ctx, commandUUID)
	if err != nil {
		return nil, err
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
	cp, err := svc.NewMDMAppleConfigProfile(ctx, req.TeamID, ff, req.Profile.Size)
	if err != nil {
		return &newMDMAppleConfigProfileResponse{Err: err}, nil
	}
	return &newMDMAppleConfigProfileResponse{
		ProfileID: cp.ProfileID,
	}, nil
}

func (svc *Service) NewMDMAppleConfigProfile(ctx context.Context, teamID uint, r io.Reader, size int64) (*fleet.MDMAppleConfigProfile, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleConfigProfile{TeamID: &teamID}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	var teamName string
	if teamID >= 1 {
		tm, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, &teamID, nil)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err)
		}
		teamName = tm.Name
	}

	b := make([]byte, size)
	_, err := r.Read(b)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message:     "failed to read config profile",
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

	newCP, err := svc.ds.NewMDMAppleConfigProfile(ctx, *cp)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	if err := svc.ds.BulkSetPendingMDMAppleHostProfiles(ctx, nil, nil, []uint{newCP.ProfileID}, nil); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "bulk set pending host profiles")
	}

	if err := svc.ds.NewActivity(ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeCreatedMacosProfile{
		TeamID:            &teamID,
		TeamName:          &teamName,
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
	res := listMDMAppleConfigProfilesResponse{}

	cps, err := svc.ListMDMAppleConfigProfiles(ctx, req.TeamID)
	if err != nil {
		res.Err = err
		return &res, err
	}
	res.ConfigProfiles = cps

	return &res, nil
}

func (svc *Service) ListMDMAppleConfigProfiles(ctx context.Context, teamID uint) ([]*fleet.MDMAppleConfigProfile, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleConfigProfile{TeamID: &teamID}, fleet.ActionRead); err != nil {
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

	cp, err := svc.GetMDMAppleConfigProfile(ctx, req.ProfileID)
	if err != nil {
		return getMDMAppleConfigProfileResponse{Err: err}, nil
	}
	reader := bytes.NewReader(cp.Mobileconfig)
	fileName := fmt.Sprintf("%s_%s", time.Now().Format("2006-01-02"), strings.ReplaceAll(cp.Name, " ", "_"))

	return getMDMAppleConfigProfileResponse{fileReader: io.NopCloser(reader), fileLength: reader.Size(), fileName: fileName}, nil
}

func (svc *Service) GetMDMAppleConfigProfile(ctx context.Context, profileID uint) (*fleet.MDMAppleConfigProfile, error) {
	// first we perform a perform basic authz check
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	cp, err := svc.ds.GetMDMAppleConfigProfile(ctx, profileID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	// now we can do a specific authz check based on team id of profile before we return the profile
	if err := svc.authz.Authorize(ctx, cp, fleet.ActionRead); err != nil {
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

	if err := svc.DeleteMDMAppleConfigProfile(ctx, req.ProfileID); err != nil {
		return &deleteMDMAppleConfigProfileResponse{Err: err}, nil
	}

	return &deleteMDMAppleConfigProfileResponse{}, nil
}

func (svc *Service) DeleteMDMAppleConfigProfile(ctx context.Context, profileID uint) error {
	// first we perform a perform basic authz check
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	cp, err := svc.ds.GetMDMAppleConfigProfile(ctx, profileID)
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
	if err := svc.authz.Authorize(ctx, cp, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	// prevent deleting profiles that are managed by Fleet
	if _, ok := mobileconfig.FleetPayloadIdentifiers()[cp.Identifier]; ok {
		return &fleet.BadRequestError{
			Message:     "profiles managed by Fleet can't be deleted using this endpoint.",
			InternalErr: fmt.Errorf("deleting profile %s for team %s not allowed because it's managed by Fleet", cp.Identifier, teamName),
		}
	}

	if err := svc.ds.DeleteMDMAppleConfigProfile(ctx, profileID); err != nil {
		return ctxerr.Wrap(ctx, err)
	}
	// cannot use the profile ID as it is now deleted
	if err := svc.ds.BulkSetPendingMDMAppleHostProfiles(ctx, nil, []uint{teamID}, nil, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk set pending host profiles")
	}

	if err := svc.ds.NewActivity(ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeDeletedMacosProfile{
		TeamID:            &teamID,
		TeamName:          &teamName,
		ProfileName:       cp.Name,
		ProfileIdentifier: cp.Identifier,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "logging activity for delete mdm apple config profile")
	}

	return nil
}

type getMDMAppleProfilesSummaryRequest struct {
	TeamID *uint `query:"team_id,optional"`
}

type getMDMAppleProfilesSummaryResponse struct {
	fleet.MDMAppleHostsProfilesSummary
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

	res.Latest = ps.Latest
	res.Failed = ps.Failed
	res.Pending = ps.Pending

	return &res, nil
}

func (svc *Service) GetMDMAppleProfilesSummary(ctx context.Context, teamID *uint) (*fleet.MDMAppleHostsProfilesSummary, error) {
	if err := svc.authz.Authorize(ctx, fleet.MDMAppleConfigProfile{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	ps, err := svc.ds.GetMDMAppleHostsProfilesSummary(ctx, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	return ps, nil
}

type getMDMAppleFileVaultSummaryRequest struct {
	TeamID *uint `query:"team_id,optional"`
}

type getMDMAppleFileVauleSummaryResponse struct {
	*fleet.MDMAppleFileVaultSummary
	Err error `json:"error,omitempty"`
}

func (r getMDMAppleFileVauleSummaryResponse) error() error { return r.Err }

func getMdmAppleFileVaultSummaryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getMDMAppleFileVaultSummaryRequest)

	fvs, err := svc.GetMDMAppleFileVaultSummary(ctx, req.TeamID)
	if err != nil {
		return &getMDMAppleFileVauleSummaryResponse{Err: err}, nil
	}

	return &getMDMAppleFileVauleSummaryResponse{
		MDMAppleFileVaultSummary: fvs,
	}, nil
}

// QUESTION: workflow for developing new APIs? whats your setup quickly test code working?
func (svc *Service) GetMDMAppleFileVaultSummary(ctx context.Context, teamID *uint) (*fleet.MDMAppleFileVaultSummary, error) {
	if err := svc.authz.Authorize(ctx, fleet.MDMAppleConfigProfile{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	fvs, err := svc.ds.GetMDMAppleFileVaultSummary(ctx, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	return fvs, nil
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
	manifest, err := appmanifest.Create(&readerWithSize{
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
	status int   `json:"-"`
	Err    error `json:"error,omitempty"`
}

func (r enqueueMDMAppleCommandResponse) error() error { return r.Err }
func (r enqueueMDMAppleCommandResponse) Status() int  { return r.status }

func enqueueMDMAppleCommandEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*enqueueMDMAppleCommandRequest)
	status, result, err := svc.EnqueueMDMAppleCommand(ctx, req.Command, req.DeviceIDs, false)
	if err != nil {
		return enqueueMDMAppleCommandResponse{Err: err}, nil
	}
	return enqueueMDMAppleCommandResponse{
		status:               status,
		CommandEnqueueResult: result,
	}, nil
}

func (svc *Service) EnqueueMDMAppleCommand(
	ctx context.Context,
	rawBase64Cmd string,
	deviceIDs []string,
	noPush bool,
) (status int, result *fleet.CommandEnqueueResult, err error) {
	premiumCommands := map[string]bool{
		"EraseDevice": true,
		"DeviceLock":  true,
	}

	// load hosts (lite) by uuids, check that the user has the rigts to run
	// commands for every affected team.
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return 0, nil, ctxerr.Wrap(ctx, err)
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return 0, nil, fleet.ErrNoContext
	}
	// for the team filter, we don't include observers as we require maintainer
	// and up to run commands.
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: false}
	hosts, err := svc.ds.ListHostsLiteByUUIDs(ctx, filter, deviceIDs)
	if err != nil {
		return 0, nil, err
	}
	if len(hosts) == 0 {
		return 0, nil, newNotFoundError()
	}

	// collect the team IDs and verify that the user has access to run commands
	// on all affected teams.
	teamIDs := make(map[uint]bool)
	for _, h := range hosts {
		var id uint
		if h.TeamID != nil {
			id = *h.TeamID
		}
		teamIDs[id] = true
	}

	var command fleet.MDMAppleCommand
	for tmID := range teamIDs {
		command.TeamID = &tmID
		if tmID == 0 {
			command.TeamID = nil
		}

		if err := svc.authz.Authorize(ctx, command, fleet.ActionWrite); err != nil {
			return 0, nil, ctxerr.Wrap(ctx, err)
		}
	}

	rawXMLCmd, err := base64.RawStdEncoding.DecodeString(rawBase64Cmd)
	if err != nil {
		return 0, nil, ctxerr.Wrap(ctx, err, "decode base64 command")
	}
	cmd, err := mdm.DecodeCommand(rawXMLCmd)
	if err != nil {
		return 0, nil, ctxerr.Wrap(ctx, err, "decode plist command")
	}

	if premiumCommands[strings.TrimSpace(cmd.Command.RequestType)] {
		lic, err := svc.License(ctx)
		if err != nil {
			return 0, nil, ctxerr.Wrap(ctx, err, "get license")
		}
		if !lic.IsPremium() {
			return 0, nil, fleet.ErrMissingLicense
		}
	}

	if err := svc.mdmAppleCommander.enqueue(ctx, deviceIDs, string(rawXMLCmd)); err != nil {
		// if at least one UUID enqueued properly, return success, otherwise return 500
		var apnsErr *APNSDeliveryError
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &apnsErr) {
			if len(apnsErr.FailedUUIDs) < len(deviceIDs) {
				// some hosts properly received the command, so return success, with the list
				// of failed uuids.
				return http.StatusOK, &fleet.CommandEnqueueResult{
					CommandUUID: cmd.CommandUUID,
					RequestType: cmd.Command.RequestType,
					FailedUUIDs: apnsErr.FailedUUIDs,
				}, nil
			}
		} else if errors.As(err, &mysqlErr) {
			// enqueue may fail with a foreign key constraint error 1452 when one of
			// the hosts provided is not enrolled in nano_enrollments. Detect when
			// that's the case and add information to the error.
			if mysqlErr.Number == mysqlerr.ER_NO_REFERENCED_ROW_2 {
				err := fleet.NewInvalidArgumentError("device_ids", fmt.Sprintf("at least one of the hosts is not enrolled in MDM: %v", err))
				return http.StatusInternalServerError, nil, ctxerr.Wrap(ctx, err, "enqueue command")
			}
		}
		return http.StatusInternalServerError, nil, ctxerr.Wrap(ctx, err, "enqueue command")
	}
	return http.StatusOK, &fleet.CommandEnqueueResult{
		CommandUUID: cmd.CommandUUID,
		RequestType: cmd.Command.RequestType,
	}, nil
}

type mdmAppleEnrollRequest struct {
	Token string `query:"token"`
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

	profile, err := svc.GetMDMAppleEnrollmentProfileByToken(ctx, req.Token)
	if err != nil {
		return mdmAppleEnrollResponse{Err: err}, nil
	}
	return mdmAppleEnrollResponse{
		Profile: profile,
	}, nil
}

func (svc *Service) GetMDMAppleEnrollmentProfileByToken(ctx context.Context, token string) (profile []byte, err error) {
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

	// TODO(lucas): Actually use enrollment (when we define which configuration we want to define
	// on enrollments).
	mobileconfig, err := apple_mdm.GenerateEnrollmentProfileMobileconfig(
		appConfig.OrgInfo.OrgName,
		appConfig.ServerSettings.ServerURL,
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

	// check authorization again based on host info for team-based permissions
	if err := svc.authz.Authorize(ctx, h, fleet.ActionMDMCommand); err != nil {
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

	// Since the host is unenrolled, delete all profiles assigned to the
	// host manually, the device won't Acknowledge any more requests (eg:
	// to delete profiles) and profiles are automatically removed on
	// unenrollment.
	if err := svc.ds.DeleteMDMAppleProfilesForHost(ctx, h.UUID); err != nil {
		return ctxerr.Wrap(ctx, err, "removing all profiles from host")
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
	if err := svc.BatchSetMDMAppleProfiles(ctx, req.TeamID, req.TeamName, req.Profiles, req.DryRun); err != nil {
		return batchSetMDMAppleProfilesResponse{Err: err}, nil
	}
	return batchSetMDMAppleProfilesResponse{}, nil
}

func (svc *Service) BatchSetMDMAppleProfiles(ctx context.Context, tmID *uint, tmName *string, profiles [][]byte, dryRun bool) error {
	if tmID != nil && tmName != nil {
		svc.authz.SkipAuthorization(ctx) // so that the error message is not replaced by "forbidden"
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("team_name", "cannot specify both team_id and team_name"))
	}
	if tmID != nil || tmName != nil {
		license, err := svc.License(ctx)
		if err != nil {
			svc.authz.SkipAuthorization(ctx) // so that the error message is not replaced by "forbidden"
			return err
		}
		if !license.IsPremium() {
			field := "team_id"
			if tmName != nil {
				field = "team_name"
			}
			svc.authz.SkipAuthorization(ctx) // so that the error message is not replaced by "forbidden"
			return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError(field, ErrMissingLicense.Error()))
		}
	}

	// if the team name is provided, load the corresponding team to get its id.
	// vice-versa, if the id is provided, load it to get the name (required for
	// the activity).
	if tmName != nil || tmID != nil {
		tm, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, tmID, tmName)
		if err != nil {
			return err
		}
		if tmID == nil {
			tmID = &tm.ID
		} else {
			tmName = &tm.Name
		}
	}

	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleConfigProfile{TeamID: tmID}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	appCfg, err := svc.AppConfigObfuscated(ctx)
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
	if err := svc.ds.BulkSetPendingMDMAppleHostProfiles(ctx, nil, []uint{bulkTeamID}, nil, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk set pending host profiles")
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
	license, err := svc.License(ctx)
	if err != nil {
		svc.authz.SkipAuthorization(ctx) // so that the error message is not replaced by "forbidden"
		return err
	}
	if !license.IsPremium() {
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
	ac, err := svc.AppConfigObfuscated(ctx)
	if err != nil {
		return err
	}

	var didUpdate, didUpdateMacOSDiskEncryption bool
	if payload.EnableDiskEncryption != nil {
		if ac.MDM.MacOSSettings.EnableDiskEncryption != *payload.EnableDiskEncryption {
			ac.MDM.MacOSSettings.EnableDiskEncryption = *payload.EnableDiskEncryption
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
			if ac.MDM.MacOSSettings.EnableDiskEncryption {
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
	ds fleet.Datastore
}

func NewMDMAppleCheckinAndCommandService(ds fleet.Datastore) *MDMAppleCheckinAndCommandService {
	return &MDMAppleCheckinAndCommandService{ds: ds}
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
		return err
	}
	info, err := svc.ds.GetHostMDMCheckinInfo(r.Context, m.Enrollment.UDID)
	if err != nil {
		return err
	}
	return svc.ds.NewActivity(r.Context, nil, &fleet.ActivityTypeMDMEnrolled{
		HostSerial:       info.HardwareSerial,
		HostDisplayName:  info.DisplayName,
		InstalledFromDEP: info.InstalledFromDEP,
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
		if err := svc.ds.BulkSetPendingMDMAppleHostProfiles(r.Context, nil, nil, nil, []string{r.ID}); err != nil {
			return err
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
func (svc *MDMAppleCheckinAndCommandService) CommandAndReportResults(r *mdm.Request, res *mdm.CommandResults) (*mdm.Command, error) {
	// Sometimes we get results with Status == "Idle" which don't contain a command
	// UUID and are not actionable anyways.
	if res.CommandUUID == "" {
		return nil, nil
	}

	// We explicitly get the request type because it comes empty. There's a
	// RequestType field in the struct, but it's used when a mdm.Command is
	// issued.
	requestType, err := svc.ds.GetMDMAppleCommandRequestType(r.Context, res.CommandUUID)
	if err != nil {
		return nil, ctxerr.Wrap(r.Context, err, "command service")
	}

	switch requestType {
	case "InstallProfile":
		return nil, svc.ds.UpdateOrDeleteHostMDMAppleProfile(r.Context, &fleet.HostMDMAppleProfile{
			CommandUUID:   res.CommandUUID,
			HostUUID:      res.UDID,
			Status:        fleet.MDMAppleDeliveryStatusFromCommandStatus(res.Status),
			Detail:        svc.fmtErrorChain(res.ErrorChain),
			OperationType: fleet.MDMAppleOperationTypeInstall,
		})
	case "RemoveProfile":
		return nil, svc.ds.UpdateOrDeleteHostMDMAppleProfile(r.Context, &fleet.HostMDMAppleProfile{
			CommandUUID:   res.CommandUUID,
			HostUUID:      res.UDID,
			Status:        fleet.MDMAppleDeliveryStatusFromCommandStatus(res.Status),
			Detail:        svc.fmtErrorChain(res.ErrorChain),
			OperationType: fleet.MDMAppleOperationTypeRemove,
		})
	}
	return nil, nil
}

func (svc *MDMAppleCheckinAndCommandService) fmtErrorChain(chain []mdm.ErrorChain) string {
	var sb strings.Builder
	for _, mdmErr := range chain {
		desc := mdmErr.USEnglishDescription
		if desc == "" {
			desc = mdmErr.LocalizedDescription
		}
		sb.WriteString(fmt.Sprintf("%s (%d): %s\n", mdmErr.ErrorDomain, mdmErr.ErrorCode, desc))
	}
	return sb.String()
}

// MDMAppleCommander contains methods to enqueue commands managed by Fleet and
// send push notifications to hosts.
//
// It's intentionally decoupled from fleet.Service so it can be used internally
// in crons and other services, leaving authentication/permission handling to
// the caller.
type MDMAppleCommander struct {
	storage nanomdm_storage.AllStorage
	pusher  nanomdm_push.Pusher
}

// NewMDMAppleCommander creates a new commander instance.
func NewMDMAppleCommander(mdmStorage nanomdm_storage.AllStorage, mdmPushService nanomdm_push.Pusher) *MDMAppleCommander {
	return &MDMAppleCommander{
		storage: mdmStorage,
		pusher:  mdmPushService,
	}
}

// InstallProfile sends the homonymous MDM command to the given hosts, it also
// takes care of the base64 encoding of the provided profile bytes.
func (svc *MDMAppleCommander) InstallProfile(ctx context.Context, hostUUIDs []string, profile mobileconfig.Mobileconfig, uuid string) error {
	base64Profile := base64.StdEncoding.EncodeToString(profile)
	raw := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CommandUUID</key>
	<string>%s</string>
	<key>Command</key>
	<dict>
		<key>RequestType</key>
		<string>InstallProfile</string>
		<key>Payload</key>
		<data>%s</data>
	</dict>
</dict>
</plist>`, uuid, base64Profile)
	err := svc.enqueue(ctx, hostUUIDs, raw)
	return ctxerr.Wrap(ctx, err, "commander install profile")
}

// InstallProfile sends the homonymous MDM command to the given hosts.
func (svc *MDMAppleCommander) RemoveProfile(ctx context.Context, hostUUIDs []string, profileIdentifier string, uuid string) error {
	raw := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CommandUUID</key>
	<string>%s</string>
	<key>Command</key>
	<dict>
		<key>RequestType</key>
		<string>RemoveProfile</string>
		<key>Identifier</key>
		<string>%s</string>
	</dict>
</dict>
</plist>`, uuid, profileIdentifier)
	err := svc.enqueue(ctx, hostUUIDs, raw)
	return ctxerr.Wrap(ctx, err, "commander remove profile")
}

func (svc *MDMAppleCommander) DeviceLock(ctx context.Context, hostUUIDs []string, uuid string) error {
	pin := apple_mdm.GenerateRandomPin(6)
	raw := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>CommandUUID</key>
    <string>%s</string>
    <key>Command</key>
    <dict>
      <key>RequestType</key>
      <string>DeviceLock</string>
      <key>PIN</key>
      <string>%s</string>
    </dict>
  </dict>
</plist>`, uuid, pin)
	return svc.enqueue(ctx, hostUUIDs, raw)
}

func (svc *MDMAppleCommander) EraseDevice(ctx context.Context, hostUUIDs []string, uuid string) error {
	pin := apple_mdm.GenerateRandomPin(6)
	raw := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>CommandUUID</key>
    <string>%s</string>
    <key>Command</key>
    <dict>
      <key>RequestType</key>
      <string>EraseDevice</string>
      <key>PIN</key>
      <string>%s</string>
    </dict>
  </dict>
</plist>`, uuid, pin)
	return svc.enqueue(ctx, hostUUIDs, raw)
}

// enqueue takes care of enqueuing the commands and sending push notifications
// to the devices.
//
// Always sending the push notification when a command is enqueued was decided
// internally, leaving making pushes optional as an optimization to be tackled
// later.
func (svc *MDMAppleCommander) enqueue(ctx context.Context, hostUUIDs []string, rawCommand string) error {
	cmd, err := mdm.DecodeCommand([]byte(rawCommand))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "commander enqueue")
	}

	// MySQL implementation always returns nil for the first parameter
	_, err = svc.storage.EnqueueCommand(ctx, hostUUIDs, cmd)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "commander enqueue")
	}

	apnsResponses, err := svc.pusher.Push(ctx, hostUUIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "commander push")
	}

	// Even if we didn't get an error, some of the APNs
	// responses might have failed, signal that to the caller.
	var failed []string
	for uuid, response := range apnsResponses {
		if response.Err != nil {
			failed = append(failed, uuid)
		}
	}
	if len(failed) > 0 {
		return &APNSDeliveryError{FailedUUIDs: failed, Err: err}
	}

	return nil
}

// APNSDeliveryError records an error and the associated host UUIDs in which it
// occurred.
type APNSDeliveryError struct {
	FailedUUIDs []string
	Err         error
}

func (e *APNSDeliveryError) Error() string {
	return fmt.Sprintf("APNS delivery failed with: %e, for UUIDs: %v", e.Err, e.FailedUUIDs)
}

func (e *APNSDeliveryError) Unwrap() error { return e.Err }

func (e *APNSDeliveryError) StatusCode() int { return http.StatusBadGateway }

// ensureFleetdConfig ensures there's a fleetd configuration profile in
// mdm_apple_configuration_profiles for each team and for "no team"
//
// We try our best to use each team's secret but we default to creating a
// profile with the global enroll secret if the team doesn't have any enroll
// secrets.
//
// This profile will be applied to all hosts in the team (or "no team",) but it
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
			if globalSecret == "" {
				logger.Log("err", "team_id %d doesn't have an enroll secret, and couldn't find a global enroll secret, skipping the creation of a com.fleetdm.fleetd.config profile")
				continue
			}

			logger.Log("err", "team_id %d doesn't have an enroll secret, using a global enroll secret")
			es.Secret = globalSecret
		}

		var contents bytes.Buffer
		params := mobileconfig.FleetdProfileOptions{
			EnrollSecret: es.Secret,
			ServerURL:    appCfg.ServerSettings.ServerURL,
			PayloadType:  mobileconfig.FleetdConfigPayloadIdentifier,
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

func ReconcileProfiles(
	ctx context.Context,
	ds fleet.Datastore,
	commander *MDMAppleCommander,
	logger kitlog.Logger,
) error {
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

	// toGetContents contains the IDs of all the profiles from which we
	// need to retrieve contents. Since the previous query returns one row
	// per host, it would be too expensive to retrieve the profile contents
	// there, so we make another request. Using a map to deduplicate.
	toGetContents := make(map[uint]bool)

	// toInstallByTuple is a map keyed by a string in the form of "HostUUID,ProfileIdentifier".
	// It is used to check when a profile removal can be skipped (because installing profile
	// overwrites an existing profile with the same identifier there is no need for a separate
	// remove profile command).
	toInstallByTuple := make(map[string]bool)

	// hostProfiles tracks each host_mdm_apple_profile we need to upsert
	// with the new status, operation_type, etc.
	hostProfiles := make([]*fleet.MDMAppleBulkUpsertHostProfilePayload, 0, len(toInstall)+len(toRemove))

	// replacedHostProfiles tracks each host_mdm_apple_profile we need to delete
	// because another profile with the same identifier is being installed.
	replacedHostProfiles := make([]fleet.MDMAppleBulkDeleteHostProfilePayload, 0, len(toRemove))

	// install/removeTargets are maps from profileID -> command uuid and host
	// UUIDs as the underlying MDM services are optimized to send one command to
	// multiple hosts at the same time. Note that the same command uuid is used
	// for all hosts in a given install/remove target operation.
	type cmdTarget struct {
		cmdUUID   string
		profIdent string
		hostUUIDs []string
	}
	installTargets, removeTargets := make(map[uint]*cmdTarget), make(map[uint]*cmdTarget)
	for _, p := range toInstall {
		toGetContents[p.ProfileID] = true
		toInstallByTuple[fmt.Sprintf("%s,%s", p.HostUUID, p.ProfileIdentifier)] = true

		target := installTargets[p.ProfileID]
		if target == nil {
			target = &cmdTarget{
				cmdUUID:   uuid.New().String(),
				profIdent: p.ProfileIdentifier,
			}
			installTargets[p.ProfileID] = target
		}
		target.hostUUIDs = append(target.hostUUIDs, p.HostUUID)

		hostProfiles = append(hostProfiles, &fleet.MDMAppleBulkUpsertHostProfilePayload{
			ProfileID:         p.ProfileID,
			HostUUID:          p.HostUUID,
			OperationType:     fleet.MDMAppleOperationTypeInstall,
			Status:            &fleet.MDMAppleDeliveryPending,
			CommandUUID:       target.cmdUUID,
			ProfileIdentifier: p.ProfileIdentifier,
			ProfileName:       p.ProfileName,
		})
	}

	for _, p := range toRemove {
		if isReplaced := toInstallByTuple[fmt.Sprintf("%s,%s", p.HostUUID, p.ProfileIdentifier)]; isReplaced {
			replacedHostProfiles = append(replacedHostProfiles, fleet.MDMAppleBulkDeleteHostProfilePayload{
				HostUUID:          p.HostUUID,
				ProfileID:         p.ProfileID,
				ProfileIdentifier: p.ProfileIdentifier,
			})
			continue
		}

		target := removeTargets[p.ProfileID]
		if target == nil {
			target = &cmdTarget{
				cmdUUID:   uuid.New().String(),
				profIdent: p.ProfileIdentifier,
			}
			removeTargets[p.ProfileID] = target
		}
		target.hostUUIDs = append(target.hostUUIDs, p.HostUUID)

		hostProfiles = append(hostProfiles, &fleet.MDMAppleBulkUpsertHostProfilePayload{
			ProfileID:         p.ProfileID,
			HostUUID:          p.HostUUID,
			OperationType:     fleet.MDMAppleOperationTypeRemove,
			Status:            &fleet.MDMAppleDeliveryPending,
			CommandUUID:       target.cmdUUID,
			ProfileIdentifier: p.ProfileIdentifier,
			ProfileName:       p.ProfileName,
		})
	}

	// First update all the profiles in the database before sending the
	// commands, this prevents race conditions where we could get a
	// response from the device before we set its status as 'pending'
	//
	// We'll do another pass at the end to revert any changes for failed
	// delivieries.
	if err := ds.BulkDeleteMDMAppleHostProfiles(ctx, replacedHostProfiles); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting replaced host profiles")
	}
	if err := ds.BulkUpsertMDMAppleHostProfiles(ctx, hostProfiles); err != nil {
		return ctxerr.Wrap(ctx, err, "updating host profiles")
	}

	// Grab the contents of all the profiles we need to install
	profileIDs := make([]uint, 0, len(toGetContents))
	for pid := range toGetContents {
		profileIDs = append(profileIDs, pid)
	}
	profileContents, err := ds.GetMDMAppleProfilesContents(ctx, profileIDs)
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

	execCmd := func(profID uint, target *cmdTarget, op fleet.MDMAppleOperationType) {
		defer wgProd.Done()

		var err error
		switch op {
		case fleet.MDMAppleOperationTypeInstall:
			err = commander.InstallProfile(ctx, target.hostUUIDs, profileContents[profID], target.cmdUUID)
		case fleet.MDMAppleOperationTypeRemove:
			err = commander.RemoveProfile(ctx, target.hostUUIDs, target.profIdent, target.cmdUUID)
		}

		var e *APNSDeliveryError
		switch {
		case errors.As(err, &e):
			level.Debug(logger).Log("err", "sending push notifications, profiles still enqueued", "details", err)
		case err != nil:
			level.Error(logger).Log("err", fmt.Sprintf("enqueue command to %s profiles", op), "details", err)
			ch <- remoteResult{err, target.cmdUUID}
		}
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

	for profID, target := range removeTargets {
		wgProd.Add(1)
		go execCmd(profID, target, fleet.MDMAppleOperationTypeRemove)
	}
	wgProd.Wait() // wait for all removals to be enqueued before proceeding to installs
	for profID, target := range installTargets {
		wgProd.Add(1)
		go execCmd(profID, target, fleet.MDMAppleOperationTypeInstall)
	}
	wgProd.Wait()
	close(ch) // done sending at this point, this triggers end of for loop in consumer
	wgCons.Wait()

	if err := ds.BulkUpsertMDMAppleHostProfiles(ctx, failed); err != nil {
		return ctxerr.Wrap(ctx, err, "reverting status of failed profiles")
	}
	return nil
}
