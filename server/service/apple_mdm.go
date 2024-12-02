package service

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/go-units"
	eeservice "github.com/fleetdm/fleet/v4/ee/server/service"
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
	"github.com/fleetdm/fleet/v4/server/mdm/apple/gdmf"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/assets"
	mdmcrypto "github.com/fleetdm/fleet/v4/server/mdm/crypto"
	mdmlifecycle "github.com/fleetdm/fleet/v4/server/mdm/lifecycle"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	nano_service "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/sso"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/groob/plist"
	"go.mozilla.org/pkcs7"
)

const (
	// FleetVarNDESSCEPChallenge and other variables are used as $FLEET_VAR_<VARIABLE_NAME>.
	// For example: $FLEET_VAR_NDES_SCEP_CHALLENGE
	// Currently, we assume the variables are fully unique and not substrings of each other.
	FleetVarNDESSCEPChallenge   = "NDES_SCEP_CHALLENGE"
	FleetVarNDESSCEPProxyURL    = "NDES_SCEP_PROXY_URL"
	FleetVarHostEndUserEmailIDP = "HOST_END_USER_EMAIL_IDP"
)

var (
	profileVariableRegex            = regexp.MustCompile(`(\$FLEET_VAR_(?P<name1>\w+))|(\${FLEET_VAR_(?P<name2>\w+)})`)
	fleetVarNDESSCEPChallengeRegexp = regexp.MustCompile(fmt.Sprintf(`(\$FLEET_VAR_%s)|(\${FLEET_VAR_%s})`, FleetVarNDESSCEPChallenge,
		FleetVarNDESSCEPChallenge))
	fleetVarNDESSCEPProxyURLRegexp = regexp.MustCompile(fmt.Sprintf(`(\$FLEET_VAR_%s)|(\${FLEET_VAR_%s})`, FleetVarNDESSCEPProxyURL,
		FleetVarNDESSCEPProxyURL))
	fleetVarHostEndUserEmailIDPRegexp = regexp.MustCompile(fmt.Sprintf(`(\$FLEET_VAR_%s)|(\${FLEET_VAR_%s})`, FleetVarHostEndUserEmailIDP,
		FleetVarHostEndUserEmailIDP))
	fleetVarsSupportedInConfigProfiles = []string{FleetVarNDESSCEPChallenge, FleetVarNDESSCEPProxyURL, FleetVarHostEndUserEmailIDP}
)

type hostProfileUUID struct {
	HostUUID    string
	ProfileUUID string
}

// Functions that can be overwritten in tests
var getNDESSCEPChallenge = eeservice.GetNDESSCEPChallenge

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
		decoded.TeamID = uint(teamID) //nolint:gosec // dismiss G115
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
	cp, err := svc.NewMDMAppleConfigProfile(ctx, req.TeamID, ff, nil, fleet.LabelsIncludeAll)
	if err != nil {
		return &newMDMAppleConfigProfileResponse{Err: err}, nil
	}
	return &newMDMAppleConfigProfileResponse{
		ProfileID: cp.ProfileID,
	}, nil
}

func (svc *Service) NewMDMAppleConfigProfile(ctx context.Context, teamID uint, r io.Reader, labels []string, labelsMembershipMode fleet.MDMLabelsMode) (*fleet.MDMAppleConfigProfile, error) {
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
	switch labelsMembershipMode {
	case fleet.LabelsIncludeAll:
		cp.LabelsIncludeAll = labelMap
	case fleet.LabelsIncludeAny:
		cp.LabelsIncludeAny = labelMap
	case fleet.LabelsExcludeAny:
		cp.LabelsExcludeAny = labelMap
	default:
		// TODO what happens if mode is not set?s
	}
	err = validateConfigProfileFleetVariables(string(cp.Mobileconfig))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validating fleet variables")
	}

	newCP, err := svc.ds.NewMDMAppleConfigProfile(ctx, *cp)
	if err != nil {
		var existsErr existsErrorInterface
		if errors.As(err, &existsErr) {
			msg := "Couldn't upload. A configuration profile with this name already exists."
			if re, ok := existsErr.(interface{ Resource() string }); ok {
				if re.Resource() == "MDMAppleConfigProfile.PayloadIdentifier" {
					msg = "Couldn't upload. A configuration profile with this identifier (PayloadIdentifier) already exists."
				}
			}
			err = fleet.NewInvalidArgumentError("profile", msg).
				WithStatus(http.StatusConflict)
		}
		return nil, ctxerr.Wrap(ctx, err)
	}
	if _, err := svc.ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, []string{newCP.ProfileUUID}, nil); err != nil {
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
	if err := svc.NewActivity(
		ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeCreatedMacosProfile{
			TeamID:            actTeamID,
			TeamName:          actTeamName,
			ProfileName:       newCP.Name,
			ProfileIdentifier: newCP.Identifier,
		}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "logging activity for create mdm apple config profile")
	}

	return newCP, nil
}

func validateConfigProfileFleetVariables(contents string) error {
	fleetVars := findFleetVariables(contents)
	for k := range fleetVars {
		if !slices.Contains(fleetVarsSupportedInConfigProfiles, k) {
			return &fleet.BadRequestError{Message: fmt.Sprintf("Fleet variable $FLEET_VAR_%s is not supported in configuration profiles",
				k)}
		}
	}
	return nil
}

func (svc *Service) NewMDMAppleDeclaration(ctx context.Context, teamID uint, r io.Reader, labels []string, name string, labelsMembershipMode fleet.MDMLabelsMode) (*fleet.MDMAppleDeclaration, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: &teamID}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	// check that Apple MDM is enabled - the middleware of that endpoint checks
	// only that any MDM is enabled, maybe it's just Windows
	if err := svc.VerifyMDMAppleConfigured(ctx); err != nil {
		err := fleet.NewInvalidArgumentError("declaration", fleet.AppleMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest)
		return nil, ctxerr.Wrap(ctx, err, "check macOS MDM enabled")
	}

	fleetNames := mdm_types.FleetReservedProfileNames()
	if _, ok := fleetNames[name]; ok {
		err := fleet.NewInvalidArgumentError("declaration", fmt.Sprintf("Profile name %q is not allowed.", name)).WithStatus(http.StatusBadRequest)
		return nil, err
	}

	var teamName string
	if teamID >= 1 {
		tm, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, &teamID, nil)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err)
		}
		teamName = tm.Name
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var tmID *uint
	if teamID >= 1 {
		tmID = &teamID
	}

	validatedLabels, err := svc.validateDeclarationLabels(ctx, labels)
	if err != nil {
		return nil, err
	}

	if err := validateDeclarationFleetVariables(string(data)); err != nil {
		return nil, err
	}

	// TODO(roberto): Maybe GetRawDeclarationValues belongs inside NewMDMAppleDeclaration? We can refactor this in a follow up.
	rawDecl, err := fleet.GetRawDeclarationValues(data)
	if err != nil {
		return nil, err
	}

	if err := rawDecl.ValidateUserProvided(); err != nil {
		return nil, err
	}

	d := fleet.NewMDMAppleDeclaration(data, tmID, name, rawDecl.Type, rawDecl.Identifier)

	switch labelsMembershipMode {
	case fleet.LabelsIncludeAny:
		d.LabelsIncludeAny = validatedLabels
	case fleet.LabelsExcludeAny:
		d.LabelsExcludeAny = validatedLabels
	default:
		// default to include all
		d.LabelsIncludeAll = validatedLabels
	}

	decl, err := svc.ds.NewMDMAppleDeclaration(ctx, d)
	if err != nil {
		return nil, err
	}

	if _, err := svc.ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, []string{decl.DeclarationUUID}, nil); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "bulk set pending host declarations")
	}

	var (
		actTeamID   *uint
		actTeamName *string
	)
	if teamID > 0 {
		actTeamID = &teamID
		actTeamName = &teamName
	}
	if err := svc.NewActivity(
		ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeCreatedDeclarationProfile{
			TeamID:      actTeamID,
			TeamName:    actTeamName,
			ProfileName: decl.Name,
			Identifier:  decl.Identifier,
		}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "logging activity for create mdm apple declaration")
	}

	return decl, nil
}

func validateDeclarationFleetVariables(contents string) error {
	if len(findFleetVariables(contents)) > 0 {
		return &fleet.BadRequestError{Message: "Fleet variables ($FLEET_VAR_*) are not currently supported in DDM profiles"}
	}
	return nil
}

func (svc *Service) batchValidateDeclarationLabels(ctx context.Context, labelNames []string) (map[string]fleet.ConfigurationProfileLabel, error) {
	if len(labelNames) == 0 {
		return nil, nil
	}

	labels, err := svc.ds.LabelIDsByName(ctx, labelNames)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting label IDs by name")
	}

	uniqueNames := make(map[string]bool)
	for _, entry := range labelNames {
		if _, value := uniqueNames[entry]; !value {
			uniqueNames[entry] = true
		}
	}

	if len(labels) != len(uniqueNames) {
		return nil, &fleet.BadRequestError{
			Message:     "some or all the labels provided don't exist",
			InternalErr: fmt.Errorf("names provided: %v", labelNames),
		}
	}

	profLabels := make(map[string]fleet.ConfigurationProfileLabel)
	for labelName, labelID := range labels {
		profLabels[labelName] = fleet.ConfigurationProfileLabel{
			LabelName: labelName,
			LabelID:   labelID,
		}
	}
	return profLabels, nil
}

func (svc *Service) validateDeclarationLabels(ctx context.Context, labelNames []string) ([]fleet.ConfigurationProfileLabel, error) {
	labelMap, err := svc.batchValidateDeclarationLabels(ctx, labelNames)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validating declaration labels")
	}

	var declLabels []fleet.ConfigurationProfileLabel
	for _, label := range labelMap {
		declLabels = append(declLabels, label)
	}
	return declLabels, nil
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

func (svc *Service) GetMDMAppleDeclaration(ctx context.Context, profileUUID string) (*fleet.MDMAppleDeclaration, error) {
	// first we perform a perform basic authz check
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	cp, err := svc.ds.GetMDMAppleDeclaration(ctx, profileUUID)
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

	// This call will also delete host_mdm_apple_profiles references IFF the profile has not been sent to
	// the host yet.
	if err := svc.ds.DeleteMDMAppleConfigProfile(ctx, profileUUID); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	// cannot use the profile ID as it is now deleted
	if _, err := svc.ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, []string{profileUUID}, nil); err != nil {
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
	if err := svc.NewActivity(
		ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeDeletedMacosProfile{
			TeamID:            actTeamID,
			TeamName:          actTeamName,
			ProfileName:       cp.Name,
			ProfileIdentifier: cp.Identifier,
		}); err != nil {
		return ctxerr.Wrap(ctx, err, "logging activity for delete mdm apple config profile")
	}

	return nil
}

func (svc *Service) DeleteMDMAppleDeclaration(ctx context.Context, declUUID string) error {
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

	decl, err := svc.ds.GetMDMAppleDeclaration(ctx, declUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	if _, ok := mdm_types.FleetReservedProfileNames()[decl.Name]; ok {
		return &fleet.BadRequestError{
			Message:     "profiles managed by Fleet can't be deleted using this endpoint.",
			InternalErr: fmt.Errorf("deleting profile %s is not allowed because it's managed by Fleet", decl.Name),
		}
	}

	// TODO: refine our approach to deleting restricted/forbidden types of declarations so that we
	// can check that Fleet-managed aren't being deleted; this can be addressed once we add support
	// for more types of declarations
	var d fleet.MDMAppleRawDeclaration
	if err := json.Unmarshal(decl.RawJSON, &d); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshalling declaration")
	}
	if err := d.ValidateUserProvided(); err != nil {
		return ctxerr.Wrap(ctx, &fleet.BadRequestError{Message: err.Error()})
	}

	var teamName string
	teamID := *decl.TeamID
	if teamID >= 1 {
		tm, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, &teamID, nil)
		if err != nil {
			return ctxerr.Wrap(ctx, err)
		}
		teamName = tm.Name
	}

	// now we can do a specific authz check based on team id of profile before we delete the profile
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: decl.TeamID}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	if err := svc.ds.DeleteMDMAppleConfigProfile(ctx, declUUID); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	if _, err := svc.ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{teamID}, nil, nil); err != nil {
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
	if err := svc.NewActivity(
		ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeDeletedDeclarationProfile{
			TeamID:      actTeamID,
			TeamName:    actTeamName,
			ProfileName: decl.Name,
			Identifier:  decl.Identifier,
		}); err != nil {
		return ctxerr.Wrap(ctx, err, "logging activity for delete mdm apple declaration")
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

// Deprecated: Not in Use
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
	MachineInfo         *fleet.MDMAppleMachineInfo
}

func (mdmAppleEnrollRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	decoded := mdmAppleEnrollRequest{}

	tok := r.URL.Query().Get("token")
	if tok == "" {
		return nil, &fleet.BadRequestError{
			Message: "token is required",
		}
	}
	decoded.Token = tok

	er := r.URL.Query().Get("enrollment_reference")
	decoded.EnrollmentReference = er

	// Parse the machine info from the request body
	di := r.Header.Get("x-apple-aspen-deviceinfo")
	if di != "" {
		// extract x-apple-aspen-deviceinfo custom header from request
		parsed, err := apple_mdm.ParseDeviceinfo(di, false) // FIXME: use verify=true when we have better parsing for various Apple certs (https://github.com/fleetdm/fleet/issues/20879)
		if err != nil {
			return nil, &fleet.BadRequestError{
				Message:     "unable to parse deviceinfo header",
				InternalErr: err,
			}
		}
		decoded.MachineInfo = parsed
	}

	return &decoded, nil
}

func (r mdmAppleEnrollResponse) error() error { return r.Err }

type mdmAppleEnrollResponse struct {
	Err error `json:"error,omitempty"`

	// Profile field is used in hijackRender for the response.
	Profile []byte

	SoftwareUpdateRequired *fleet.MDMAppleSoftwareUpdateRequired
}

func (r mdmAppleEnrollResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	if r.SoftwareUpdateRequired != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		if err := json.NewEncoder(w).Encode(r.SoftwareUpdateRequired); err != nil {
			encodeError(ctx, ctxerr.New(ctx, "failed to encode software update required"), w)
		}
		return
	}

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

	sur, err := svc.CheckMDMAppleEnrollmentWithMinimumOSVersion(ctx, req.MachineInfo)
	if err != nil {
		return mdmAppleEnrollResponse{Err: err}, nil
	}
	if sur != nil {
		return mdmAppleEnrollResponse{
			SoftwareUpdateRequired: sur,
		}, nil
	}

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

	enrollURL, err := apple_mdm.AddEnrollmentRefToFleetURL(appConfig.MDMUrl(), ref)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "adding reference to fleet URL")
	}

	topic, err := svc.mdmPushCertTopic(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "extracting topic from APNs cert")
	}

	assets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetSCEPChallenge,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("loading SCEP challenge from the database: %w", err)
	}
	enrollmentProf, err := apple_mdm.GenerateEnrollmentProfileMobileconfig(
		appConfig.OrgInfo.OrgName,
		enrollURL,
		string(assets[fleet.MDMAssetSCEPChallenge].Value),
		topic,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generating enrollment profile")
	}

	signed, err := mdmcrypto.Sign(ctx, enrollmentProf, svc.ds)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "signing profile")
	}

	return signed, nil
}

func (svc *Service) CheckMDMAppleEnrollmentWithMinimumOSVersion(ctx context.Context, m *fleet.MDMAppleMachineInfo) (*fleet.MDMAppleSoftwareUpdateRequired, error) {
	// skipauth: The enroll profile endpoint is unauthenticated.
	svc.authz.SkipAuthorization(ctx)

	if m == nil {
		level.Debug(svc.logger).Log("msg", "no machine info, skipping os version check")
		return nil, nil
	}

	level.Debug(svc.logger).Log("msg", "checking os version", "serial", m.Serial, "current_version", m.OSVersion)

	if !m.MDMCanRequestSoftwareUpdate {
		level.Debug(svc.logger).Log("msg", "mdm cannot request software update, skipping os version check", "serial", m.Serial)
		return nil, nil
	}

	needsUpdate, err := svc.needsOSUpdateForDEPEnrollment(ctx, *m)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "checking os updates settings", "serial", m.Serial)
	}

	if !needsUpdate {
		level.Debug(svc.logger).Log("msg", "device is above minimum, skipping os version check", "serial", m.Serial)
		return nil, nil
	}

	sur, err := svc.getAppleSoftwareUpdateRequiredForDEPEnrollment(*m)
	if err != nil {
		// log for debugging but allow enrollment to proceed
		level.Info(svc.logger).Log("msg", "getting apple software update required", "serial", m.Serial, "err", err)
		return nil, nil
	}

	return sur, nil
}

func (svc *Service) needsOSUpdateForDEPEnrollment(ctx context.Context, m fleet.MDMAppleMachineInfo) (bool, error) {
	// NOTE: Under the hood, the datastore is joining host_dep_assignments to the hosts table to
	// look up DEP hosts by serial number. It grabs the team id and platform from the
	// hosts table. Then it uses the team id to get either the global config or team config.
	// Finally, it uses the platform to get os updates settings from the config for
	// one of ios, ipados, or darwin, as applicable. There's a lot of assumptions going on here, not
	// least of which is that the platform is correct in the hosts table. If the platform is wrong,
	// we'll end up with a meaningless comparison of unrelated versions. We could potentially add
	// some cross-check against the machine info to ensure that the platform of the host aligns with
	// what we expect from the machine info. But that would involve work to derive the platform from
	// the machine info (presumably from the product name, but that's not a 1:1 mapping).
	settings, err := svc.ds.GetMDMAppleOSUpdatesSettingsByHostSerial(ctx, m.Serial)
	if err != nil {
		if fleet.IsNotFound(err) {
			level.Info(svc.logger).Log("msg", "checking os updates settings, settings not found", "serial", m.Serial)
			return false, nil
		}
		return false, err
	}
	// TODO: confirm what this check should do
	if !settings.MinimumVersion.Set || !settings.MinimumVersion.Valid || settings.MinimumVersion.Value == "" {
		level.Info(svc.logger).Log("msg", "checking os updates settings, minimum version not set", "serial", m.Serial, "current_version", m.OSVersion, "minimum_version", settings.MinimumVersion.Value)
		return false, nil
	}

	needsUpdate, err := apple_mdm.IsLessThanVersion(m.OSVersion, settings.MinimumVersion.Value)
	if err != nil {
		level.Info(svc.logger).Log("msg", "checking os updates settings, cannot compare versions", "serial", m.Serial, "current_version", m.OSVersion, "minimum_version", settings.MinimumVersion.Value)
		return false, nil
	}

	return needsUpdate, nil
}

func (svc *Service) getAppleSoftwareUpdateRequiredForDEPEnrollment(m fleet.MDMAppleMachineInfo) (*fleet.MDMAppleSoftwareUpdateRequired, error) {
	latest, err := gdmf.GetLatestOSVersion(m)
	if err != nil {
		return nil, err
	}

	needsUpdate, err := apple_mdm.IsLessThanVersion(m.OSVersion, latest.ProductVersion)
	if err != nil {
		return nil, err
	} else if !needsUpdate {
		return nil, nil
	}

	return fleet.NewMDMAppleSoftwareUpdateRequired(fleet.MDMAppleSoftwareUpdateAsset{
		ProductVersion: latest.ProductVersion,
		Build:          latest.Build,
	}), nil
}

func (svc *Service) mdmPushCertTopic(ctx context.Context) (string, error) {
	assets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetAPNSCert,
	}, nil)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "loading SCEP keypair from the database")
	}

	block, _ := pem.Decode(assets[fleet.MDMAssetAPNSCert].Value)
	if block == nil || block.Type != "CERTIFICATE" {
		return "", ctxerr.Wrap(ctx, err, "decoding PEM data")
	}

	apnsCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "parsing APNs certificate")
	}

	mdmPushCertTopic, err := cryptoutil.TopicFromCert(apnsCert)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "extracting topic from APNs certificate")
	}

	return mdmPushCertTopic, nil
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

	switch h.Platform {
	case "ios":
		fallthrough
	case "ipados":
		return &fleet.BadRequestError{
			Message: fleet.CantTurnOffMDMForIOSOrIPadOSMessage,
		}
	case "windows":
		return &fleet.BadRequestError{
			Message: fleet.CantTurnOffMDMForWindowsHostsMessage,
		}
	default:
		// host is darwin, so continue
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

	if err := svc.NewActivity(
		ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeMDMUnenrolled{
			HostSerial:       h.HardwareSerial,
			HostDisplayName:  h.DisplayName(),
			InstalledFromDEP: info.InstalledFromDEP,
		}); err != nil {
		return ctxerr.Wrap(ctx, err, "logging activity for mdm apple remove profile command")
	}

	mdmLifecycle := mdmlifecycle.New(svc.ds, svc.logger)
	err = mdmLifecycle.Do(ctx, mdmlifecycle.HostOptions{
		Action:   mdmlifecycle.HostActionTurnOff,
		Platform: info.Platform,
		UUID:     h.UUID,
	})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "running turn off action in mdm lifecycle")
	}

	return nil
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
		if len(prof) > 1024*1024 {
			return ctxerr.Wrap(ctx,
				fleet.NewInvalidArgumentError(fmt.Sprintf("profiles[%d]", i), "maximum configuration profile file size is 1 MB"),
			)
		}
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

	if !skipBulkPending {
		// check for duplicates with existing profiles, skipBulkPending signals that the caller
		// is responsible for ensuring that the profiles names are unique (e.g., MDMAppleMatchPreassignment)
		allProfs, _, err := svc.ds.ListMDMConfigProfiles(ctx, tmID, fleet.ListOptions{PerPage: 0})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "list mdm config profiles")
		}
		for _, p := range allProfs {
			if byName[p.Name] {
				switch {
				case strings.HasPrefix(p.ProfileUUID, "a"):
					// do nothing, all existing mobileconfigs will be replaced and we've already checked
					// the new mobileconfigs for duplicates
					continue
				case strings.HasPrefix(p.ProfileUUID, "w"):
					err := fleet.NewInvalidArgumentError("PayloadDisplayName", fmt.Sprintf(
						"Couldn’t edit custom_settings. A Windows configuration profile shares the same name as a macOS configuration profile (PayloadDisplayName): %q", p.Name))
					return ctxerr.Wrap(ctx, err, "duplicate xml and mobileconfig by name")
				default:
					err := fleet.NewInvalidArgumentError("PayloadDisplayName", fmt.Sprintf(
						"Couldn’t edit custom_settings. More than one configuration profile have the same name (PayloadDisplayName): %q", p.Name))
					return ctxerr.Wrap(ctx, err, "duplicate json and mobileconfig by name")
				}
			}
			byName[p.Name] = true
		}
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
		if _, err := svc.ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{bulkTeamID}, nil, nil); err != nil {
			return ctxerr.Wrap(ctx, err, "bulk set pending host profiles")
		}
	}

	if err := svc.NewActivity(
		ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeEditedMacosProfile{
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
	if err := svc.UpdateMDMDiskEncryption(ctx, req.MDMAppleSettingsPayload.TeamID, req.MDMAppleSettingsPayload.EnableDiskEncryption); err != nil {
		return updateMDMAppleSettingsResponse{Err: err}, nil
	}
	return updateMDMAppleSettingsResponse{}, nil
}

func (svc *Service) updateAppConfigMDMDiskEncryption(ctx context.Context, enabled *bool) error {
	// appconfig is only used internally, it's fine to read it unobfuscated
	// (svc.AppConfigObfuscated must not be used because the write-only users
	// such as gitops will fail to access it).
	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return err
	}

	var didUpdate bool
	if enabled != nil {
		if ac.MDM.EnableDiskEncryption.Value != *enabled {
			if *enabled && svc.config.Server.PrivateKey == "" {
				return ctxerr.New(ctx, "Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key")
			}

			ac.MDM.EnableDiskEncryption = optjson.SetBool(*enabled)
			didUpdate = true
		}
	}

	if didUpdate {
		if err := svc.ds.SaveAppConfig(ctx, ac); err != nil {
			return err
		}
		if ac.MDM.EnabledAndConfigured { // if macOS MDM is configured, set up FileVault escrow
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
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), act); err != nil {
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
		decoded.TeamID = uint(teamID) //nolint:gosec // dismiss G115
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
	//
	// NOTE: this parameter is going to be removed in a future version.
	// Prefer other ways to allow gitops read access.
	// For context, see: https://github.com/fleetdm/fleet/issues/15337#issuecomment-1932878997
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
	ds           fleet.Datastore
	logger       kitlog.Logger
	commander    *apple_mdm.MDMAppleCommander
	mdmLifecycle *mdmlifecycle.HostLifecycle
}

func NewMDMAppleCheckinAndCommandService(ds fleet.Datastore, commander *apple_mdm.MDMAppleCommander, logger kitlog.Logger) *MDMAppleCheckinAndCommandService {
	mdmLifecycle := mdmlifecycle.New(ds, logger)
	return &MDMAppleCheckinAndCommandService{
		ds:           ds,
		commander:    commander,
		logger:       logger,
		mdmLifecycle: mdmLifecycle,
	}
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
	existingDeviceInfo, err := svc.ds.GetHostMDMCheckinInfo(r.Context, r.ID)
	if err != nil {
		var nfe fleet.NotFoundError
		if !errors.As(err, &nfe) {
			return ctxerr.Wrap(r.Context, err, "getting checkin info")
		}
	} else if existingDeviceInfo.SCEPRenewalInProgress {
		svc.logger.Log("info", "host lifecycle action received for a SCEP renewal in process, skipping host ingestion and cleanups", "host_uuid", r.ID)
		return nil
	}

	// iPhones and iPads send ProductName but not Model/ModelName,
	// thus we use this field as the device's Model (which is required on lifecycle stages).
	platform := "darwin"
	iPhone := strings.HasPrefix(m.ProductName, "iPhone")
	iPad := strings.HasPrefix(m.ProductName, "iPad")
	if iPhone || iPad {
		m.Model = m.ProductName
		if iPhone {
			platform = "ios"
		} else {
			platform = "ipados"
		}
	}

	err = svc.mdmLifecycle.Do(r.Context, mdmlifecycle.HostOptions{
		Action:         mdmlifecycle.HostActionReset,
		Platform:       platform,
		UUID:           m.UDID,
		HardwareSerial: m.SerialNumber,
		HardwareModel:  m.Model,
	})
	if err != nil {
		return err
	}

	// MDM state changes after is reset, fetch the checkin updatedInfo again
	updatedInfo, err := svc.ds.GetHostMDMCheckinInfo(r.Context, r.ID)
	if err != nil {
		return ctxerr.Wrap(r.Context, err, "getting checkin info in Authenticate message")
	}

	return newActivity(
		r.Context, nil, &fleet.ActivityTypeMDMEnrolled{
			HostSerial:       updatedInfo.HardwareSerial,
			HostDisplayName:  updatedInfo.DisplayName,
			InstalledFromDEP: updatedInfo.DEPAssignedToFleet,
			MDMPlatform:      fleet.MDMPlatformApple,
		}, svc.ds, svc.logger,
	)
}

// TokenUpdate handles MDM [TokenUpdate][1] requests.
//
// This method is executed after the request has been handled by nanomdm.
//
// [1]: https://developer.apple.com/documentation/devicemanagement/token_update
func (svc *MDMAppleCheckinAndCommandService) TokenUpdate(r *mdm.Request, m *mdm.TokenUpdate) error {
	info, err := svc.ds.GetHostMDMCheckinInfo(r.Context, r.ID)
	if err != nil {
		return ctxerr.Wrap(r.Context, err, "getting checkin info")
	}

	if info.SCEPRenewalInProgress {
		svc.logger.Log("info", "host lifecycle action received for a SCEP renewal in process", "host_uuid", r.ID)
		err := svc.ds.CleanSCEPRenewRefs(r.Context, r.ID)
		return ctxerr.Wrap(r.Context, err, "cleaning SCEP refs")
	}

	var hasSetupExpItems bool
	if m.AwaitingConfiguration {
		// Enqueue setup experience items and mark the host as being in setup experience
		hasSetupExpItems, err = svc.ds.EnqueueSetupExperienceItems(r.Context, r.ID, info.TeamID)
		if err != nil {
			return ctxerr.Wrap(r.Context, err, "queueing setup experience tasks")
		}
	}

	return svc.mdmLifecycle.Do(r.Context, mdmlifecycle.HostOptions{
		Action:                  mdmlifecycle.HostActionTurnOn,
		Platform:                info.Platform,
		UUID:                    r.ID,
		EnrollReference:         r.Params[mobileconfig.FleetEnrollReferenceKey],
		HasSetupExperienceItems: hasSetupExpItems,
	})
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

	err = svc.mdmLifecycle.Do(r.Context, mdmlifecycle.HostOptions{
		Action:   mdmlifecycle.HostActionTurnOff,
		Platform: info.Platform,
		UUID:     r.ID,
	})
	if err != nil {
		return err
	}

	return newActivity(
		r.Context, nil, &fleet.ActivityTypeMDMUnenrolled{
			HostSerial:       info.HardwareSerial,
			HostDisplayName:  info.DisplayName,
			InstalledFromDEP: info.InstalledFromDEP,
		}, svc.ds, svc.logger,
	)
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
func (svc *MDMAppleCheckinAndCommandService) DeclarativeManagement(r *mdm.Request, dm *mdm.DeclarativeManagement) ([]byte, error) {
	// DeclarativeManagement is handled by the MDMAppleDDMService.
	return nil, nil
}

// GetToken handles MDM [GetToken][1] requests.
//
// This method is executed after the request has been handled by nanomdm.
//
// [1]: https://developer.apple.com/documentation/devicemanagement/get_token
func (svc *MDMAppleCheckinAndCommandService) GetToken(_ *mdm.Request, _ *mdm.GetToken) (*mdm.GetTokenResponse, error) {
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

		return nil, nil
	}

	// Check if this is a result of a "refetch" command sent to iPhones/iPads
	// to fetch their device information periodically.
	if strings.HasPrefix(cmdResult.CommandUUID, fleet.RefetchBaseCommandUUIDPrefix) {
		return svc.handleRefetch(r, cmdResult)
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
	case "DeviceLock", "EraseDevice":
		// call into our datastore to update host_mdm_actions if the status is terminal
		if cmdResult.Status == fleet.MDMAppleStatusAcknowledged ||
			cmdResult.Status == fleet.MDMAppleStatusError ||
			cmdResult.Status == fleet.MDMAppleStatusCommandFormatError {
			return nil, svc.ds.UpdateHostLockWipeStatusFromAppleMDMResult(r.Context, cmdResult.UDID, cmdResult.CommandUUID, requestType,
				cmdResult.Status == fleet.MDMAppleStatusAcknowledged)
		}
	case "DeclarativeManagement":
		// set "pending-install" profiles to "verifying" or "failed"
		// depending on the status of the DeviceManagement command
		status := mdmAppleDeliveryStatusFromCommandStatus(cmdResult.Status)
		detail := fmt.Sprintf("%s. Make sure the host is on macOS 13+, iOS 17+, iPadOS 17+.", apple_mdm.FmtErrorChain(cmdResult.ErrorChain))
		err := svc.ds.MDMAppleSetPendingDeclarationsAs(r.Context, cmdResult.UDID, status, detail)
		return nil, ctxerr.Wrap(r.Context, err, "update declaration status on DeclarativeManagement ack")
	case "InstallApplication":
		// this might be a setup experience VPP install, so we'll try to update setup experience status
		// TODO: consider limiting this to only macOS hosts
		if updated, err := maybeUpdateSetupExperienceStatus(r.Context, svc.ds, fleet.SetupExperienceVPPInstallResult{
			HostUUID:      cmdResult.UDID,
			CommandUUID:   cmdResult.CommandUUID,
			CommandStatus: cmdResult.Status,
		}, true); err != nil {
			return nil, ctxerr.Wrap(r.Context, err, "updating setup experience status from VPP install result")
		} else if updated {
			// TODO: call next step of setup experience?
			level.Debug(svc.logger).Log("msg", "setup experience script result updated", "host_uuid", cmdResult.UDID, "execution_id", cmdResult.CommandUUID)
		}

		// create an activity for installing only if we're in a terminal state
		if cmdResult.Status == fleet.MDMAppleStatusAcknowledged ||
			cmdResult.Status == fleet.MDMAppleStatusError ||
			cmdResult.Status == fleet.MDMAppleStatusCommandFormatError {
			user, act, err := svc.ds.GetPastActivityDataForVPPAppInstall(r.Context, cmdResult)
			if err != nil {
				if fleet.IsNotFound(err) {
					// Then this isn't a VPP install, so no activity generated
					return nil, nil
				}

				return nil, ctxerr.Wrap(r.Context, err, "fetching data for installed app store app activity")
			}

			if err := newActivity(r.Context, user, act, svc.ds, svc.logger); err != nil {
				return nil, ctxerr.Wrap(r.Context, err, "creating activity for installed app store app")
			}
		}
	case "DeviceConfigured":
		if err := svc.ds.SetHostAwaitingConfiguration(r.Context, r.ID, false); err != nil {
			return nil, ctxerr.Wrap(r.Context, err, "failed to mark host as non longer awaiting configuration")
		}
	}

	return nil, nil
}

func (svc *MDMAppleCheckinAndCommandService) handleRefetch(r *mdm.Request, cmdResult *mdm.CommandResults) (*mdm.Command, error) {
	ctx := r.Context
	host, err := svc.ds.HostByIdentifier(ctx, cmdResult.UDID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to get host by identifier")
	}

	if strings.HasPrefix(cmdResult.CommandUUID, fleet.RefetchAppsCommandUUIDPrefix) {
		// We remove pending command first in case there is an error processing the results, so that we don't prevent another refetch.
		err = svc.ds.RemoveHostMDMCommand(ctx, fleet.HostMDMCommand{
			HostID:      host.ID,
			CommandType: fleet.RefetchAppsCommandUUIDPrefix,
		})
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "remove refetch apps command")
		}

		if host.Platform != "ios" && host.Platform != "ipados" {
			return nil, ctxerr.New(ctx, "refetch apps command sent to non-iOS/non-iPadOS host")
		}
		source := "ios_apps"
		if host.Platform == "ipados" {
			source = "ipados_apps"
		}

		response := cmdResult.Raw
		software, err := unmarshalAppList(ctx, response, source)
		if err != nil {
			return nil, err
		}
		_, err = svc.ds.UpdateHostSoftware(ctx, host.ID, software)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "update host software")
		}

		return nil, nil
	}

	// Otherwise, the command has prefix fleet.RefetchDeviceCommandUUIDPrefix, which is a refetch device command.
	// We remove pending command first in case there is an error processing the results, so that we don't prevent another refetch.
	err = svc.ds.RemoveHostMDMCommand(ctx, fleet.HostMDMCommand{
		HostID:      host.ID,
		CommandType: fleet.RefetchDeviceCommandUUIDPrefix,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "remove refetch device command")
	}

	var deviceInformationResponse struct {
		QueryResponses map[string]interface{} `plist:"QueryResponses"`
	}
	if err := plist.Unmarshal(cmdResult.Raw, &deviceInformationResponse); err != nil {
		return nil, ctxerr.Wrap(r.Context, err, "failed to unmarshal device information command result")
	}
	deviceName := deviceInformationResponse.QueryResponses["DeviceName"].(string)
	deviceCapacity := deviceInformationResponse.QueryResponses["DeviceCapacity"].(float64)
	availableDeviceCapacity := deviceInformationResponse.QueryResponses["AvailableDeviceCapacity"].(float64)
	osVersion := deviceInformationResponse.QueryResponses["OSVersion"].(string)
	wifiMac := deviceInformationResponse.QueryResponses["WiFiMAC"].(string)
	productName := deviceInformationResponse.QueryResponses["ProductName"].(string)
	host.ComputerName = deviceName
	host.Hostname = deviceName
	host.GigsDiskSpaceAvailable = availableDeviceCapacity
	host.GigsTotalDiskSpace = deviceCapacity
	var (
		osVersionPrefix string
		platform        string
	)
	if strings.HasPrefix(productName, "iPhone") {
		osVersionPrefix = "iOS"
		platform = "ios"
	} else { // iPad
		osVersionPrefix = "iPadOS"
		platform = "ipados"
	}
	host.OSVersion = osVersionPrefix + " " + osVersion
	host.PrimaryMac = wifiMac
	host.HardwareModel = productName
	host.DetailUpdatedAt = time.Now()
	host.RefetchRequested = false
	if err := svc.ds.UpdateHost(r.Context, host); err != nil {
		return nil, ctxerr.Wrap(r.Context, err, "failed to update host")
	}
	if err := svc.ds.SetOrUpdateHostDisksSpace(r.Context, host.ID, availableDeviceCapacity, 100*availableDeviceCapacity/deviceCapacity,
		deviceCapacity); err != nil {
		return nil, ctxerr.Wrap(r.Context, err, "failed to update host storage")
	}
	if err := svc.ds.UpdateHostOperatingSystem(r.Context, host.ID, fleet.OperatingSystem{
		Name:     osVersionPrefix,
		Version:  osVersion,
		Platform: platform,
	}); err != nil {
		return nil, ctxerr.Wrap(r.Context, err, "failed to update host operating system")
	}

	if host.MDM.EnrollmentStatus != nil && *host.MDM.EnrollmentStatus == "Pending" {
		// Since the device has been refetched, we can assume it's enrolled.
		err = svc.ds.UpdateMDMData(ctx, host.ID, true)
		if err != nil {
			return nil, ctxerr.Wrap(r.Context, err, "failed to update MDM data")
		}
	}
	return nil, nil
}

func unmarshalAppList(ctx context.Context, response []byte, source string) ([]fleet.Software,
	error,
) {
	var appsResponse struct {
		InstalledApplicationList []map[string]interface{} `plist:"InstalledApplicationList"`
	}
	if err := plist.Unmarshal(response, &appsResponse); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to unmarshal installed application list command result")
	}

	truncateString := func(item interface{}, length int) string {
		str, ok := item.(string)
		if !ok {
			return ""
		}
		runes := []rune(str)
		if len(runes) > length {
			return string(runes[:length])
		}
		return str
	}

	var software []fleet.Software
	for _, app := range appsResponse.InstalledApplicationList {
		software = append(software, fleet.Software{
			Name:             truncateString(app["Name"], fleet.SoftwareNameMaxLength),
			Version:          truncateString(app["ShortVersion"], fleet.SoftwareVersionMaxLength),
			BundleIdentifier: truncateString(app["Identifier"], fleet.SoftwareBundleIdentifierMaxLength),
			Source:           source,
		})
	}

	return software, nil
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

// ensureFleetProfiles ensures there's a fleetd configuration profile in
// mdm_apple_configuration_profiles for each team and for "no team"
//
// We try our best to use each team's secret but we default to creating a
// profile with the global enroll secret if the team doesn't have any enroll
// secrets.
//
// This profile will be installed to all hosts in the team (or "no team",) but it
// will only be used by hosts that have a fleetd installation without an enroll
// secret and fleet URL (mainly DEP enrolled hosts).
func ensureFleetProfiles(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, signingCertDER []byte) error {
	appCfg, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching app config")
	}

	var rootCAProfContents bytes.Buffer
	params := mobileconfig.FleetCARootTemplateOptions{
		PayloadIdentifier: mobileconfig.FleetCARootConfigPayloadIdentifier,
		PayloadName:       mdm_types.FleetCAConfigProfileName,
		Certificate:       base64.StdEncoding.EncodeToString(signingCertDER),
	}

	if err := mobileconfig.FleetCARootTemplate.Execute(&rootCAProfContents, params); err != nil {
		return ctxerr.Wrap(ctx, err, "executing fleet root CA config template")
	}

	b := rootCAProfContents.Bytes()

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
			ServerURL:    appCfg.ServerSettings.ServerURL, // ServerURL must be set to the Fleet URL.  Do not use appCfg.MDMUrl() here.
			PayloadType:  mobileconfig.FleetdConfigPayloadIdentifier,
			PayloadName:  mdm_types.FleetdConfigProfileName,
		}

		if err := mobileconfig.FleetdProfileTemplate.Execute(&contents, params); err != nil {
			return ctxerr.Wrap(ctx, err, "executing fleetd config template")
		}

		cp, err := fleet.NewMDMAppleConfigProfile(contents.Bytes(), es.TeamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building fleetd configuration profile")
		}
		profiles = append(profiles, cp)

		rootCAProf, err := fleet.NewMDMAppleConfigProfile(b, es.TeamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building root CA configuration profile")
		}
		profiles = append(profiles, rootCAProf)
	}

	if err := ds.BulkUpsertMDMAppleConfigProfiles(ctx, profiles); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk-upserting configuration profiles")
	}

	return nil
}

func SendPushesToPendingDevices(
	ctx context.Context,
	ds fleet.Datastore,
	commander *apple_mdm.MDMAppleCommander,
	logger kitlog.Logger,
) error {
	uuids, err := ds.GetHostUUIDsWithPendingMDMAppleCommands(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting host uuids with pending commands")
	}

	if len(uuids) == 0 {
		return nil
	}

	if err := commander.SendNotifications(ctx, uuids); err != nil {
		var apnsErr *apple_mdm.APNSDeliveryError
		if errors.As(err, &apnsErr) {
			level.Info(logger).Log("msg", "failed to send APNs notification to some hosts", "error", apnsErr.Error())
			return nil
		}

		return ctxerr.Wrap(ctx, err, "sending push notifications")

	}

	return nil
}

func ReconcileAppleDeclarations(
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

	// batch set declarations as pending
	changedHosts, err := ds.MDMAppleBatchSetHostDeclarationState(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "updating host declaration state")
	}

	if len(changedHosts) == 0 {
		level.Info(logger).Log("msg", "no hosts with changed declarations")
		return nil
	}

	// send a DeclarativeManagement command to start a sync
	if err := commander.DeclarativeManagement(ctx, changedHosts, uuid.NewString()); err != nil {
		return ctxerr.Wrap(ctx, err, "issuing DeclarativeManagement command")
	}

	level.Info(logger).Log("msg", "sent DeclarativeManagement command", "host_number", len(changedHosts))

	return nil
}

// install/removeTargets are maps from profileUUID -> command uuid and host
// UUIDs as the underlying MDM services are optimized to send one command to
// multiple hosts at the same time. Note that the same command uuid is used
// for all hosts in a given install/remove target operation.
type cmdTarget struct {
	cmdUUID   string
	profIdent string
	hostUUIDs []string
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

	assets, err := ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetCACert,
	}, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting Apple SCEP")
	}

	block, _ := pem.Decode(assets[fleet.MDMAssetCACert].Value)
	if block == nil || block.Type != "CERTIFICATE" {
		return ctxerr.Wrap(ctx, err, "failed to decode PEM block from SCEP certificate")
	}

	if err := ensureFleetProfiles(ctx, ds, logger, block.Bytes); err != nil {
		logger.Log("err", "unable to ensure a fleetd configuration profiles are in place", "details", err)
	}

	// retrieve the profiles to install/remove.
	toInstall, err := ds.ListMDMAppleProfilesToInstall(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting profiles to install")
	}

	// Exclude macOS only profiles from iPhones/iPads.
	toInstall = fleet.FilterMacOSOnlyProfilesFromIOSIPadOS(toInstall)

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

	// Index host profiles to install by host and profile UUID, for easier bulk error processing
	hostProfilesToInstallMap := make(map[hostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload, len(toInstall))

	installTargets, removeTargets := make(map[string]*cmdTarget), make(map[string]*cmdTarget)
	for _, p := range toInstall {
		if pp, ok := profileIntersection.GetMatchingProfileInCurrentState(p); ok {
			// if the profile was in any other status than `failed`
			// and the checksums match (the profiles are exactly
			// the same) we don't send another InstallProfile
			// command.
			if pp.Status != &fleet.MDMDeliveryFailed && bytes.Equal(pp.Checksum, p.Checksum) {
				hostProfile := &fleet.MDMAppleBulkUpsertHostProfilePayload{
					ProfileUUID:       p.ProfileUUID,
					HostUUID:          p.HostUUID,
					ProfileIdentifier: p.ProfileIdentifier,
					ProfileName:       p.ProfileName,
					Checksum:          p.Checksum,
					OperationType:     pp.OperationType,
					Status:            pp.Status,
					CommandUUID:       pp.CommandUUID,
					Detail:            pp.Detail,
				}
				hostProfiles = append(hostProfiles, hostProfile)
				hostProfilesToInstallMap[hostProfileUUID{HostUUID: p.HostUUID, ProfileUUID: p.ProfileUUID}] = hostProfile
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

		hostProfile := &fleet.MDMAppleBulkUpsertHostProfilePayload{
			ProfileUUID:       p.ProfileUUID,
			HostUUID:          p.HostUUID,
			OperationType:     fleet.MDMOperationTypeInstall,
			Status:            &fleet.MDMDeliveryPending,
			CommandUUID:       target.cmdUUID,
			ProfileIdentifier: p.ProfileIdentifier,
			ProfileName:       p.ProfileName,
			Checksum:          p.Checksum,
		}
		hostProfiles = append(hostProfiles, hostProfile)
		hostProfilesToInstallMap[hostProfileUUID{HostUUID: p.HostUUID, ProfileUUID: p.ProfileUUID}] = hostProfile
	}

	for _, p := range toRemove {
		if _, ok := profileIntersection.GetMatchingProfileInDesiredState(p); ok {
			hostProfilesToCleanup = append(hostProfilesToCleanup, p)
			continue
		}

		if p.DidNotInstallOnHost() {
			// then we shouldn't send an additional remove command since it wasn't installed on the
			// host.
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
	// deliveries.
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

	// Insert variables into profile contents
	err = preprocessProfileContents(ctx, appConfig, ds, installTargets, profileContents, hostProfilesToInstallMap)
	if err != nil {
		return err
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

func preprocessProfileContents(
	ctx context.Context,
	appConfig *fleet.AppConfig,
	ds fleet.Datastore,
	targets map[string]*cmdTarget,
	profileContents map[string]mobileconfig.Mobileconfig,
	hostProfilesToInstallMap map[hostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload,
) error {
	// This method replaces Fleet variables ($FLEET_VAR_<NAME>) in the profile contents, generating a unique profile for each host.
	// For a 2KB profile and 30K hosts, this method may generate ~60MB of profile data in memory.

	isNDESSCEPConfigured := func(profUUID string, target *cmdTarget) (bool, error) {
		if !license.IsPremium(ctx) {
			profilesToUpdate := make([]*fleet.MDMAppleBulkUpsertHostProfilePayload, 0, len(target.hostUUIDs))
			for _, hostUUID := range target.hostUUIDs {
				profile, ok := hostProfilesToInstallMap[hostProfileUUID{HostUUID: hostUUID, ProfileUUID: profUUID}]
				if !ok { // Should never happen
					continue
				}
				profile.Status = &fleet.MDMDeliveryFailed
				profile.Detail = "NDES SCEP Proxy requires a Fleet Premium license."
				profilesToUpdate = append(profilesToUpdate, profile)
			}
			if err := ds.BulkUpsertMDMAppleHostProfiles(ctx, profilesToUpdate); err != nil {
				return false, err
			}
			return false, nil
		}
		if !appConfig.Integrations.NDESSCEPProxy.Valid {
			profilesToUpdate := make([]*fleet.MDMAppleBulkUpsertHostProfilePayload, 0, len(target.hostUUIDs))
			for _, hostUUID := range target.hostUUIDs {
				profile, ok := hostProfilesToInstallMap[hostProfileUUID{HostUUID: hostUUID, ProfileUUID: profUUID}]
				if !ok { // Should never happen
					continue
				}
				profile.Status = &fleet.MDMDeliveryFailed
				profile.Detail = "NDES SCEP Proxy is not configured. " +
					"Please configure in Settings > Integrations > Mobile Device Management > Simple Certificate Enrollment Protocol."
				profilesToUpdate = append(profilesToUpdate, profile)
			}
			if err := ds.BulkUpsertMDMAppleHostProfiles(ctx, profilesToUpdate); err != nil {
				return false, err
			}
			return false, nil
		}
		return appConfig.Integrations.NDESSCEPProxy.Valid, nil
	}

	// Copy of NDES SCEP config which will contain unencrypted password, if needed
	var ndesConfig *fleet.NDESSCEPProxyIntegration

	var addedTargets map[string]*cmdTarget
	for profUUID, target := range targets {
		contents, ok := profileContents[profUUID]
		if !ok {
			// This should never happen
			continue
		}

		// Check if Fleet variables are present.
		contentsStr := string(contents)
		fleetVars := findFleetVariables(contentsStr)
		if len(fleetVars) == 0 {
			continue
		}

		// Do common validation that applies to all hosts in the target
		valid := true
		for fleetVar := range fleetVars {
			switch fleetVar {
			case FleetVarNDESSCEPChallenge, FleetVarNDESSCEPProxyURL:
				configured, err := isNDESSCEPConfigured(profUUID, target)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "checking NDES SCEP configuration")
				}
				if !configured {
					valid = false
					break
				}
			case FleetVarHostEndUserEmailIDP:
				// No extra validation needed for this variable
			default:
				// Error out if we find an unknown variable
				profilesToUpdate := make([]*fleet.MDMAppleBulkUpsertHostProfilePayload, 0, len(target.hostUUIDs))
				for _, hostUUID := range target.hostUUIDs {
					profile, ok := hostProfilesToInstallMap[hostProfileUUID{HostUUID: hostUUID, ProfileUUID: profUUID}]
					if !ok { // Should never happen
						continue
					}
					profile.Status = &fleet.MDMDeliveryFailed
					profile.Detail = fmt.Sprintf("Unknown Fleet variable $FLEET_VAR_%s found in profile. Please update or remove.",
						fleetVar)
					profilesToUpdate = append(profilesToUpdate, profile)
				}
				if err := ds.BulkUpsertMDMAppleHostProfiles(ctx, profilesToUpdate); err != nil {
					return ctxerr.Wrap(ctx, err, "updating host MDM Apple profiles for unknown variable")
				}
				valid = false
			}
		}
		if !valid {
			// We marked the profile as failed, so we will not do any additional processing on it
			delete(targets, profUUID)
			continue
		}

		// Currently, all supported Fleet variables are unique per host, so we split the profile into multiple profiles.
		// We generate a new temporary profileUUID which is currently only used to install the profile.
		// The profileUUID in host_mdm_apple_profiles is still the original profileUUID.
		// We also generate a new commandUUID which is used to install the profile via nano_commands table.
		if addedTargets == nil {
			addedTargets = make(map[string]*cmdTarget, 1)
		}
		// We store the timestamp when the challenge was retrieved to know if it has expired.
		var managedCertificatePayloads []*fleet.MDMBulkUpsertManagedCertificatePayload
		// We need to update the profiles of each host with the new command UUID
		profilesToUpdate := make([]*fleet.MDMAppleBulkUpsertHostProfilePayload, 0, len(target.hostUUIDs))
		for _, hostUUID := range target.hostUUIDs {
			tempProfUUID := uuid.NewString()
			// Use the same UUID for command UUID, which will be the primary key for nano_commands
			tempCmdUUID := tempProfUUID
			profile, ok := hostProfilesToInstallMap[hostProfileUUID{HostUUID: hostUUID, ProfileUUID: profUUID}]
			if !ok { // Should never happen
				continue
			}
			profile.CommandUUID = tempCmdUUID

			hostContents := contentsStr

			failed := false
			for fleetVar := range fleetVars {
				switch fleetVar {
				case FleetVarNDESSCEPChallenge:
					if ndesConfig == nil {
						// Retrieve the NDES admin password. This is done once per run.
						configAssets, err := ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetNDESPassword}, nil)
						if err != nil {
							return ctxerr.Wrap(ctx, err, "getting NDES password")
						}
						// Copy config struct by value
						configWithPassword := appConfig.Integrations.NDESSCEPProxy.Value
						configWithPassword.Password = string(configAssets[fleet.MDMAssetNDESPassword].Value)
						// Store the config with the password for later use
						ndesConfig = &configWithPassword
					}
					// Insert the SCEP challenge into the profile contents
					challenge, err := getNDESSCEPChallenge(ctx, *ndesConfig)
					if err != nil {
						detail := ""
						switch {
						case errors.As(err, &eeservice.NDESInvalidError{}):
							detail = fmt.Sprintf("Invalid NDES admin credentials. "+
								"Fleet couldn't populate $FLEET_VAR_%s. "+
								"Please update credentials in Settings > Integrations > Mobile Device Management > Simple Certificate Enrollment Protocol.",
								FleetVarNDESSCEPChallenge)
						case errors.As(err, &eeservice.NDESPasswordCacheFullError{}):
							detail = fmt.Sprintf("The NDES password cache is full. "+
								"Fleet couldn't populate $FLEET_VAR_%s. "+
								"Please increase the number of cached passwords in NDES and try again.",
								FleetVarNDESSCEPChallenge)
						case errors.As(err, &eeservice.NDESInsufficientPermissionsError{}):
							detail = fmt.Sprintf("This account does not have sufficient permissions to enroll with SCEP. "+
								"Fleet couldn't populate $FLEET_VAR_%s. "+
								"Please update the account with NDES SCEP enroll permissions and try again.",
								FleetVarNDESSCEPChallenge)
						default:
							detail = fmt.Sprintf("Fleet couldn't populate $FLEET_VAR_%s. %s", FleetVarNDESSCEPChallenge, err.Error())
						}
						err := ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
							CommandUUID:   target.cmdUUID,
							HostUUID:      hostUUID,
							Status:        &fleet.MDMDeliveryFailed,
							Detail:        detail,
							OperationType: fleet.MDMOperationTypeInstall,
						})
						if err != nil {
							return ctxerr.Wrap(ctx, err, "updating host MDM Apple profile for NDES SCEP challenge")
						}
						failed = true
						break
					}
					payload := &fleet.MDMBulkUpsertManagedCertificatePayload{
						HostUUID:             hostUUID,
						ProfileUUID:          profUUID,
						ChallengeRetrievedAt: ptr.Time(time.Now()),
					}
					managedCertificatePayloads = append(managedCertificatePayloads, payload)

					hostContents = replaceFleetVariable(fleetVarNDESSCEPChallengeRegexp, hostContents, challenge)
				case FleetVarNDESSCEPProxyURL:
					// Insert the SCEP URL into the profile contents
					proxyURL := fmt.Sprintf("%s%s%s", appConfig.MDMUrl(), apple_mdm.SCEPProxyPath,
						url.PathEscape(fmt.Sprintf("%s,%s", hostUUID, profUUID)))
					hostContents = replaceFleetVariable(fleetVarNDESSCEPProxyURLRegexp, hostContents, proxyURL)
				case FleetVarHostEndUserEmailIDP:
					// Insert the end user email IDP into the profile contents
					emails, err := ds.GetHostEmails(ctx, hostUUID, fleet.DeviceMappingMDMIdpAccounts)
					if err != nil {
						// This is a server error, so we exit.
						return ctxerr.Wrap(ctx, err, "getting host emails")
					}
					if len(emails) == 0 {
						// Error if we can't retrieve the end user email IDP
						err := ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
							CommandUUID: target.cmdUUID,
							HostUUID:    hostUUID,
							Status:      &fleet.MDMDeliveryFailed,
							Detail: fmt.Sprintf("There is no IdP email for this host. "+
								"Fleet couldn't populate $FLEET_VAR_%s. "+
								"[Learn more](https://fleetdm.com/learn-more-about/idp-email)",
								FleetVarHostEndUserEmailIDP),
							OperationType: fleet.MDMOperationTypeInstall,
						})
						if err != nil {
							return ctxerr.Wrap(ctx, err, "updating host MDM Apple profile for end user email IdP")
						}
						failed = true
						break
					}
					hostContents = replaceFleetVariable(fleetVarHostEndUserEmailIDPRegexp, hostContents, emails[0])
				default:
					// This was handled in the above switch statement, so we should never reach this case
				}
			}
			if !failed {
				addedTargets[tempProfUUID] = &cmdTarget{
					cmdUUID:   tempCmdUUID,
					profIdent: target.profIdent,
					hostUUIDs: []string{hostUUID},
				}
				profileContents[tempProfUUID] = mobileconfig.Mobileconfig(hostContents)
				profilesToUpdate = append(profilesToUpdate, profile)
			}
		}
		// Update profiles with the new command UUID
		if err := ds.BulkUpsertMDMAppleHostProfiles(ctx, profilesToUpdate); err != nil {
			return ctxerr.Wrap(ctx, err, "updating host profiles")
		}
		err := ds.BulkUpsertMDMManagedCertificates(ctx, managedCertificatePayloads)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "updating managed certificates")
		}
		// Remove the parent target, since we will use host-specific targets
		delete(targets, profUUID)
	}
	if len(addedTargets) > 0 {
		// Add the new host-specific targets to the original targets map
		for profUUID, target := range addedTargets {
			targets[profUUID] = target
		}
	}
	return nil
}

func replaceFleetVariable(regExp *regexp.Regexp, contents string, replacement string) string {
	// Escape XML characters
	b := make([]byte, 0, len(replacement))
	buf := bytes.NewBuffer(b)
	// error is always nil for Buffer.Write method, so we ignore it
	_ = xml.EscapeText(buf, []byte(replacement))
	return regExp.ReplaceAllString(contents, buf.String())
}

func findFleetVariables(contents string) map[string]interface{} {
	var result map[string]interface{}
	matches := profileVariableRegex.FindAllStringSubmatch(contents, -1)
	if len(matches) == 0 {
		return nil
	}
	nameToIndex := make(map[string]int, 2)
	for i, name := range profileVariableRegex.SubexpNames() {
		if name == "" {
			continue
		}
		nameToIndex[name] = i
	}
	for _, match := range matches {
		for _, i := range nameToIndex {
			if match[i] != "" {
				if result == nil {
					result = make(map[string]interface{})
				}
				result[match[i]] = struct{}{}
			}
		}
	}
	return result
}

// scepCertRenewalThresholdDays defines the number of days before a SCEP
// certificate must be renewed.
const scepCertRenewalThresholdDays = 180

// maxCertsRenewalPerRun specifies the maximum number of certificates to renew
// in a single cron run.
//
// Assuming that the cron runs every hour, we'll enqueue 24,000 renewals per
// day, and we have room for 24,000 * scepCertRenewalThresholdDays total
// renewals.
//
// For a default of 180 days as a threshold this gives us room for a fleet of
// ~4 million devices expiring at the same time.
const maxCertsRenewalPerRun = 100

func RenewSCEPCertificates(
	ctx context.Context,
	logger kitlog.Logger,
	ds fleet.Datastore,
	config *config.FleetConfig,
	commander *apple_mdm.MDMAppleCommander,
) error {
	renewalDisable, exists := os.LookupEnv("FLEET_MDM_APPLE_SCEP_RENEWAL_DISABLE")
	if exists && (strings.EqualFold(renewalDisable, "true") || renewalDisable == "1") {
		level.Info(logger).Log("msg", "skipping renewal of macOS SCEP certificates as FLEET_MDM_APPLE_SCEP_RENEWAL_DISABLE is set to true")
		return nil
	}

	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return fmt.Errorf("reading app config: %w", err)
	}
	if !appConfig.MDM.EnabledAndConfigured {
		level.Debug(logger).Log("msg", "skipping renewal of macOS SCEP certificates as MDM is not fully configured")
		return nil
	}

	if commander == nil {
		level.Debug(logger).Log("msg", "skipping renewal of macOS SCEP certificates as apple_mdm.MDMAppleCommander was not provided")
		return nil
	}

	// for each hash, grab the host that uses it as its identity certificate
	certAssociations, err := ds.GetHostCertAssociationsToExpire(ctx, scepCertRenewalThresholdDays, maxCertsRenewalPerRun)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting host cert associations")
	}

	if len(certAssociations) == 0 {
		level.Debug(logger).Log("msg", "no certs to renew")
		return nil
	}

	// assocsWithRefs stores hosts that have enrollment references on their
	// enrollment profiles. This is the case for ADE-enrolled hosts using
	// SSO to authenticate.
	assocsWithRefs := []fleet.SCEPIdentityAssociation{}
	// assocsWithoutRefs stores hosts that don't have an enrollment
	// reference in their enrollment profile.
	assocsWithoutRefs := []fleet.SCEPIdentityAssociation{}
	// assocsFromMigration stores hosts that were migrated from another MDM
	// using the process described in
	// https://github.com/fleetdm/fleet/issues/19387
	assocsFromMigration := []fleet.SCEPIdentityAssociation{}
	for _, assoc := range certAssociations {
		if assoc.EnrolledFromMigration {
			assocsFromMigration = append(assocsFromMigration, assoc)
			continue
		}

		if assoc.EnrollReference != "" {
			assocsWithRefs = append(assocsWithRefs, assoc)
			continue
		}
		assocsWithoutRefs = append(assocsWithoutRefs, assoc)
	}

	mdmPushCertTopic, err := assets.APNSTopic(ctx, ds)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "extracting topic from APNs certificate")
	}

	assets, err := ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetSCEPChallenge,
	}, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "loading SCEP challenge from the database")
	}
	scepChallenge := string(assets[fleet.MDMAssetSCEPChallenge].Value)

	// send a single command for all the hosts without references.
	if len(assocsWithoutRefs) > 0 {
		profile, err := apple_mdm.GenerateEnrollmentProfileMobileconfig(
			appConfig.OrgInfo.OrgName,
			appConfig.MDMUrl(),
			scepChallenge,
			mdmPushCertTopic,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "generating enrollment profile for hosts without enroll reference")
		}

		if err := renewSCEPWithProfile(ctx, ds, commander, logger, assocsWithoutRefs, profile); err != nil {
			return ctxerr.Wrap(ctx, err, "sending profile to hosts without associations")
		}
	}

	// send individual commands for each host with a reference
	for _, assoc := range assocsWithRefs {
		enrollURL, err := apple_mdm.AddEnrollmentRefToFleetURL(appConfig.MDMUrl(), assoc.EnrollReference)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "adding reference to fleet URL")
		}

		profile, err := apple_mdm.GenerateEnrollmentProfileMobileconfig(
			appConfig.OrgInfo.OrgName,
			enrollURL,
			scepChallenge,
			mdmPushCertTopic,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "generating enrollment profile for hosts with enroll reference")
		}

		// each host with association needs a different enrollment profile, and thus a different command.
		if err := renewSCEPWithProfile(ctx, ds, commander, logger, []fleet.SCEPIdentityAssociation{assoc}, profile); err != nil {
			return ctxerr.Wrap(ctx, err, "sending profile to hosts without associations")
		}
	}

	decodedMigrationEnrollmentProfile, err := base64.StdEncoding.DecodeString(os.Getenv("FLEET_SILENT_MIGRATION_ENROLLMENT_PROFILE"))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "failed to decode silent migration enrollment profile")
	}
	hasAssocsFromMigration := len(assocsFromMigration) > 0

	migrationEnrollmentProfile := string(decodedMigrationEnrollmentProfile)
	if migrationEnrollmentProfile == "" && hasAssocsFromMigration {
		level.Debug(logger).Log("msg", "found devices from migration that need SCEP renewals but FLEET_SILENT_MIGRATION_ENROLLMENT_PROFILE is empty")
	}
	if migrationEnrollmentProfile != "" && hasAssocsFromMigration {
		profileBytes := []byte(migrationEnrollmentProfile)
		if err := renewSCEPWithProfile(ctx, ds, commander, logger, assocsFromMigration, profileBytes); err != nil {
			return ctxerr.Wrap(ctx, err, "sending profile to hosts from migration")
		}
	}

	return nil
}

func renewSCEPWithProfile(
	ctx context.Context,
	ds fleet.Datastore,
	commander *apple_mdm.MDMAppleCommander,
	logger kitlog.Logger,
	assocs []fleet.SCEPIdentityAssociation,
	profile []byte,
) error {
	cmdUUID := uuid.NewString()
	var uuids []string
	duplicateUUIDCheck := map[string]struct{}{}
	for _, assoc := range assocs {
		// this should never happen if our DB logic is on point.
		// This sanity check is in place to prevent issues like
		// https://github.com/fleetdm/fleet/issues/19311 where a
		// single duplicated UUID prevents _all_ the commands from
		// being enqueued.
		if _, ok := duplicateUUIDCheck[assoc.HostUUID]; ok {
			logger.Log("inf", "duplicated host UUID while renewing associations", "host_uuid", assoc.HostUUID)
			continue
		}

		duplicateUUIDCheck[assoc.HostUUID] = struct{}{}
		uuids = append(uuids, assoc.HostUUID)
	}

	if err := commander.InstallProfile(ctx, uuids, profile, cmdUUID); err != nil {
		return ctxerr.Wrapf(ctx, err, "sending InstallProfile command for hosts %s", uuids)
	}

	if err := ds.SetCommandForPendingSCEPRenewal(ctx, assocs, cmdUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "setting pending command associations")
	}

	return nil
}

// MDMAppleDDMService is the service that handles MDM [DeclarativeManagement][1] requests.
//
// [1]: https://developer.apple.com/documentation/devicemanagement/declarative_management_checkin
type MDMAppleDDMService struct {
	ds     fleet.Datastore
	logger kitlog.Logger
}

func NewMDMAppleDDMService(ds fleet.Datastore, logger kitlog.Logger) *MDMAppleDDMService {
	return &MDMAppleDDMService{
		ds:     ds,
		logger: logger,
	}
}

// DeclarativeManagement handles MDM [DeclarativeManagement][1] requests.
//
// This method is when the request has been handled by nanomdm.
//
// [1]: https://developer.apple.com/documentation/devicemanagement/declarative_management_checkin
func (svc *MDMAppleDDMService) DeclarativeManagement(r *mdm.Request, dm *mdm.DeclarativeManagement) ([]byte, error) {
	if dm == nil {
		level.Debug(svc.logger).Log("msg", "ddm request received with nil payload")
		return nil, nil
	}
	level.Debug(svc.logger).Log("msg", "ddm request received", "endpoint", dm.Endpoint)

	if err := svc.ds.InsertMDMAppleDDMRequest(r.Context, dm.UDID, dm.Endpoint, dm.Data); err != nil {
		return nil, ctxerr.Wrap(r.Context, err, "insert ddm request history")
	}

	if dm.UDID == "" {
		return nil, nano_service.NewHTTPStatusError(http.StatusBadRequest, ctxerr.New(r.Context, "missing UDID in request"))
	}

	switch {
	case dm.Endpoint == "tokens":
		level.Debug(svc.logger).Log("msg", "received tokens request")
		return svc.handleTokens(r.Context, dm.UDID)

	case dm.Endpoint == "declaration-items":
		level.Debug(svc.logger).Log("msg", "received declaration-items request")
		return svc.handleDeclarationItems(r.Context, dm.UDID)

	case dm.Endpoint == "status":
		level.Debug(svc.logger).Log("msg", "received status request")
		return nil, svc.handleDeclarationStatus(r.Context, dm)

	case strings.HasPrefix(dm.Endpoint, "declaration/"):
		level.Debug(svc.logger).Log("msg", "received declarations request")
		return svc.handleDeclarationsResponse(r.Context, dm.Endpoint, dm.UDID)

	default:
		return nil, nano_service.NewHTTPStatusError(http.StatusBadRequest, ctxerr.New(r.Context, fmt.Sprintf("unrecognized declarations endpoint: %s", dm.Endpoint)))
	}
}

func (svc *MDMAppleDDMService) handleTokens(ctx context.Context, hostUUID string) ([]byte, error) {
	tok, err := svc.ds.MDMAppleDDMDeclarationsToken(ctx, hostUUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting synchronization tokens")
	}

	b, err := json.Marshal(fleet.MDMAppleDDMTokensResponse{
		SyncTokens: *tok,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshaling synchronization tokens")
	}

	return b, nil
}

func (svc *MDMAppleDDMService) handleDeclarationItems(ctx context.Context, hostUUID string) ([]byte, error) {
	di, err := svc.ds.MDMAppleDDMDeclarationItems(ctx, hostUUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting synchronization tokens")
	}

	activations := []fleet.MDMAppleDDMManifest{}
	configurations := []fleet.MDMAppleDDMManifest{}
	for _, d := range di {
		configurations = append(configurations, fleet.MDMAppleDDMManifest(d))
		activations = append(activations, fleet.MDMAppleDDMManifest{
			Identifier:  fmt.Sprintf("%s.activation", d.Identifier),
			ServerToken: d.ServerToken,
		})
	}

	// TODO: Look for ways to optimize the declaration item query so that we don't have to get the declarations token separately.
	dTok, err := svc.ds.MDMAppleDDMDeclarationsToken(ctx, hostUUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting declarations token")
	}

	b, err := json.Marshal(fleet.MDMAppleDDMDeclarationItemsResponse{
		Declarations: fleet.MDMAppleDDMManifestItems{
			Activations:    activations,
			Configurations: configurations,
			Assets:         []fleet.MDMAppleDDMManifest{},
			Management:     []fleet.MDMAppleDDMManifest{},
		},
		DeclarationsToken: dTok.DeclarationsToken,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshaling synchronization tokens")
	}

	return b, nil
}

func (svc *MDMAppleDDMService) handleDeclarationsResponse(ctx context.Context, endpoint string, hostUUID string) ([]byte, error) {
	parts := strings.Split(endpoint, "/")
	if len(parts) != 3 {
		return nil, nano_service.NewHTTPStatusError(http.StatusBadRequest, ctxerr.Errorf(ctx, "unrecognized declarations endpoint: %s", endpoint))
	}
	level.Debug(svc.logger).Log("msg", "parsed declarations request", "type", parts[1], "identifier", parts[2])

	switch parts[1] {
	case "activation":
		return svc.handleActivationDeclaration(ctx, parts, hostUUID)
	case "configuration":
		return svc.handleConfigurationDeclaration(ctx, parts, hostUUID)
	default:
		return nil, nano_service.NewHTTPStatusError(http.StatusNotFound, ctxerr.Errorf(ctx, "declaration type not supported: %s", parts[1]))
	}
}

func (svc *MDMAppleDDMService) handleActivationDeclaration(ctx context.Context, parts []string, hostUUID string) ([]byte, error) {
	references := strings.TrimSuffix(parts[2], ".activation")

	// ensure the declaration for the requested activation still exists
	d, err := svc.ds.MDMAppleDDMDeclarationsResponse(ctx, references, hostUUID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return nil, nano_service.NewHTTPStatusError(http.StatusNotFound, err)
		}
		return nil, ctxerr.Wrap(ctx, err, "getting linked configuration for activation declaration")
	}

	response := fmt.Sprintf(`
{
  "Identifier": "%s",
  "Payload": {
    "StandardConfigurations": ["%s"]
  },
  "ServerToken": "%s",
  "Type": "com.apple.activation.simple"
}`, parts[2], references, d.Checksum)

	return []byte(response), nil
}

func (svc *MDMAppleDDMService) handleConfigurationDeclaration(ctx context.Context, parts []string, hostUUID string) ([]byte, error) {
	d, err := svc.ds.MDMAppleDDMDeclarationsResponse(ctx, parts[2], hostUUID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return nil, nano_service.NewHTTPStatusError(http.StatusNotFound, err)
		}
		return nil, ctxerr.Wrap(ctx, err, "getting declaration response")
	}

	var tempd map[string]any
	if err := json.Unmarshal(d.RawJSON, &tempd); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "unmarshaling stored declaration")
	}
	tempd["ServerToken"] = d.Checksum

	b, err := json.Marshal(tempd)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshaling declaration")
	}
	return b, nil
}

func (svc *MDMAppleDDMService) handleDeclarationStatus(ctx context.Context, dm *mdm.DeclarativeManagement) error {
	var status fleet.MDMAppleDDMStatusReport
	if err := json.Unmarshal(dm.Data, &status); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshalling response")
	}

	configurationReports := status.StatusItems.Management.Declarations.Configurations
	updates := make([]*fleet.MDMAppleHostDeclaration, len(configurationReports))
	for i, r := range configurationReports {
		var status fleet.MDMDeliveryStatus
		var detail string
		switch {
		case r.Active && r.Valid == fleet.MDMAppleDeclarationValid:
			status = fleet.MDMDeliveryVerified
		case r.Valid == fleet.MDMAppleDeclarationInvalid:
			status = fleet.MDMDeliveryFailed
			detail = apple_mdm.FmtDDMError(r.Reasons)
		default:
			status = fleet.MDMDeliveryVerifying
		}

		updates[i] = &fleet.MDMAppleHostDeclaration{
			Status:        &status,
			OperationType: fleet.MDMOperationTypeInstall,
			Detail:        detail,
			Checksum:      r.ServerToken,
		}
	}

	// MDMAppleStoreDDMStatusReport takes care of cleaning ("pending", "remove")
	// pairs for the host.
	//
	// TODO(roberto): in the DDM documentation, it's mentioned that status
	// report will give you a "remove" status so the server can track
	// removals. In my testing, I never saw this (after spending
	// considerable time trying to make it work.)
	//
	// My current guess is that the documentation is implicitly referring
	// to asset declarations (which deliver tangible "assets" to the host)
	//
	// The best indication I found so far, is that if the declaration is
	// not in the report, then it's implicitly removed.
	if err := svc.ds.MDMAppleStoreDDMStatusReport(ctx, dm.UDID, updates); err != nil {
		return ctxerr.Wrap(ctx, err, "updating host declaration status with reports")
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Generate ABM keypair endpoint
////////////////////////////////////////////////////////////////////////////////

type generateABMKeyPairResponse struct {
	PublicKey []byte `json:"public_key,omitempty"`
	Err       error  `json:"error,omitempty"`
}

func (r generateABMKeyPairResponse) error() error { return r.Err }

func generateABMKeyPairEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	keyPair, err := svc.GenerateABMKeyPair(ctx)
	if err != nil {
		return generateABMKeyPairResponse{
			Err: err,
		}, nil
	}

	return generateABMKeyPairResponse{
		PublicKey: keyPair.PublicKey,
	}, nil
}

func (svc *Service) GenerateABMKeyPair(ctx context.Context) (*fleet.MDMAppleDEPKeyPair, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppleBM{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	privateKey := svc.config.Server.PrivateKey
	if testSetEmptyPrivateKey {
		privateKey = ""
	}

	if len(privateKey) == 0 {
		return nil, ctxerr.New(ctx, "Couldn't download public key. Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key")
	}

	var publicKeyPEM, privateKeyPEM []byte
	assets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetABMCert,
		fleet.MDMAssetABMKey,
	}, nil)
	if err != nil {
		// allow not found errors as it means that we're generating the
		// keypair for the first time
		if !fleet.IsNotFound(err) {
			return nil, ctxerr.Wrap(ctx, err, "loading ABM keys from the database")
		}
	}

	// if we don't have any certificates, create a new keypair, otherwise
	// return the already stored values to allow for the renewal flow.
	if len(assets) == 0 {
		publicKeyPEM, privateKeyPEM, err = apple_mdm.NewDEPKeyPairPEM()
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "generate key pair")
		}

		err = svc.ds.InsertMDMConfigAssets(ctx, []fleet.MDMConfigAsset{
			{Name: fleet.MDMAssetABMCert, Value: publicKeyPEM},
			{Name: fleet.MDMAssetABMKey, Value: privateKeyPEM},
		}, nil)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "saving ABM keypair in database")
		}
	} else {
		// we can trust that the keys exist due to the contract specified by
		// the datastore method
		publicKeyPEM = assets[fleet.MDMAssetABMCert].Value
		privateKeyPEM = assets[fleet.MDMAssetABMKey].Value
	}

	return &fleet.MDMAppleDEPKeyPair{
		PublicKey:  publicKeyPEM,
		PrivateKey: privateKeyPEM,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Upload ABM token endpoint
////////////////////////////////////////////////////////////////////////////////

type uploadABMTokenRequest struct {
	Token *multipart.FileHeader
}

func (uploadABMTokenRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	err := r.ParseMultipartForm(512 * units.MiB)
	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}

	token, ok := r.MultipartForm.File["token"]
	if !ok || len(token) < 1 {
		return nil, &fleet.BadRequestError{Message: "no file headers for token"}
	}

	return &uploadABMTokenRequest{
		Token: token[0],
	}, nil
}

type uploadABMTokenResponse struct {
	Token *fleet.ABMToken `json:"abm_token,omitempty"`
	Err   error           `json:"error,omitempty"`
}

func (r uploadABMTokenResponse) error() error { return r.Err }

func uploadABMTokenEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*uploadABMTokenRequest)
	ff, err := req.Token.Open()
	if err != nil {
		return uploadABMTokenResponse{Err: err}, nil
	}
	defer ff.Close()

	token, err := svc.UploadABMToken(ctx, ff)
	if err != nil {
		return uploadABMTokenResponse{
			Err: err,
		}, nil
	}

	return uploadABMTokenResponse{Token: token}, nil
}

func (svc *Service) UploadABMToken(ctx context.Context, token io.Reader) (*fleet.ABMToken, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Disable ABM endpoint
////////////////////////////////////////////////////////////////////////////////

type deleteABMTokenRequest struct {
	TokenID uint `url:"id"`
}

type deleteABMTokenResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteABMTokenResponse) error() error { return r.Err }
func (r deleteABMTokenResponse) Status() int  { return http.StatusNoContent }

func deleteABMTokenEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteABMTokenRequest)
	if err := svc.DeleteABMToken(ctx, req.TokenID); err != nil {
		return deleteABMTokenResponse{Err: err}, nil
	}

	return deleteABMTokenResponse{}, nil
}

func (svc *Service) DeleteABMToken(ctx context.Context, tokenID uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// List ABM tokens endpoint
////////////////////////////////////////////////////////////////////////////////

type listABMTokensResponse struct {
	Err    error             `json:"error,omitempty"`
	Tokens []*fleet.ABMToken `json:"abm_tokens"`
}

func (r listABMTokensResponse) error() error { return r.Err }

func listABMTokensEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	tokens, err := svc.ListABMTokens(ctx)
	if err != nil {
		return &listABMTokensResponse{Err: err}, nil
	}

	if tokens == nil {
		tokens = []*fleet.ABMToken{}
	}

	return &listABMTokensResponse{Tokens: tokens}, nil
}

func (svc *Service) ListABMTokens(ctx context.Context) ([]*fleet.ABMToken, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

// //////////////////////////////////////////////////////////////////////////////
// Count ABM tokens endpoint
// //////////////////////////////////////////////////////////////////////////////

type countABMTokensResponse struct {
	Err   error  `json:"error,omitempty"`
	Count uint32 `json:"count"`
}

func (r countABMTokensResponse) error() error { return r.Err }

func countABMTokensEndpoint(ctx context.Context, _ interface{}, svc fleet.Service) (errorer, error) {
	tokenCount, err := svc.CountABMTokens(ctx)
	if err != nil {
		return &countABMTokensResponse{Err: err}, nil
	}

	return &countABMTokensResponse{Count: tokenCount}, nil
}

func (svc *Service) CountABMTokens(ctx context.Context) (uint32, error) {
	// Automatic enrollment (ABM/ADE/DEP) is a feature that requires a license.
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return 0, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Update ABM token teams endpoint
////////////////////////////////////////////////////////////////////////////////

type updateABMTokenTeamsRequest struct {
	TokenID      uint  `url:"id"`
	MacOSTeamID  *uint `json:"macos_team_id"`
	IOSTeamID    *uint `json:"ios_team_id"`
	IPadOSTeamID *uint `json:"ipados_team_id"`
}

type updateABMTokenTeamsResponse struct {
	ABMToken *fleet.ABMToken `json:"abm_token,omitempty"`
	Err      error           `json:"error,omitempty"`
}

func (r updateABMTokenTeamsResponse) error() error { return r.Err }

func updateABMTokenTeamsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*updateABMTokenTeamsRequest)

	tok, err := svc.UpdateABMTokenTeams(ctx, req.TokenID, req.MacOSTeamID, req.IOSTeamID, req.IPadOSTeamID)
	if err != nil {
		return &updateABMTokenTeamsResponse{Err: err}, nil
	}

	return &updateABMTokenTeamsResponse{ABMToken: tok}, nil
}

func (svc *Service) UpdateABMTokenTeams(ctx context.Context, tokenID uint, macOSTeamID, iOSTeamID, iPadOSTeamID *uint) (*fleet.ABMToken, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Renew ABM token endpoint
////////////////////////////////////////////////////////////////////////////////

type renewABMTokenRequest struct {
	TokenID uint `url:"id"`
	Token   *multipart.FileHeader
}

func (renewABMTokenRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	err := r.ParseMultipartForm(512 * units.MiB)
	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}

	token, ok := r.MultipartForm.File["token"]
	if !ok || len(token) < 1 {
		return nil, &fleet.BadRequestError{Message: "no file headers for token"}
	}

	// because we are in this method, we know that the path has 7 parts, e.g:
	// /api/latest/fleet/abm_tokens/19/renew

	id, err := intFromRequest(r, "id")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to parse abm token id")
	}

	return &renewABMTokenRequest{
		Token:   token[0],
		TokenID: uint(id), //nolint:gosec // dismiss G115
	}, nil
}

type renewABMTokenResponse struct {
	ABMToken *fleet.ABMToken `json:"abm_token,omitempty"`
	Err      error           `json:"error,omitempty"`
}

func (r renewABMTokenResponse) error() error { return r.Err }

func renewABMTokenEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*renewABMTokenRequest)
	ff, err := req.Token.Open()
	if err != nil {
		return &renewABMTokenResponse{Err: err}, nil
	}
	defer ff.Close()

	tok, err := svc.RenewABMToken(ctx, ff, req.TokenID)
	if err != nil {
		return &renewABMTokenResponse{Err: err}, nil
	}

	return &renewABMTokenResponse{ABMToken: tok}, nil
}

func (svc *Service) RenewABMToken(ctx context.Context, token io.Reader, tokenID uint) (*fleet.ABMToken, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// GET /enrollment_profiles/ota
////////////////////////////////////////////////////////////////////////////////

type getOTAProfileRequest struct {
	EnrollSecret string `query:"enroll_secret"`
}

func getOTAProfileEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getOTAProfileRequest)
	profile, err := svc.GetOTAProfile(ctx, req.EnrollSecret)
	if err != nil {
		return &getMDMAppleConfigProfileResponse{Err: err}, err
	}

	reader := bytes.NewReader(profile)
	return &getMDMAppleConfigProfileResponse{fileReader: io.NopCloser(reader), fileLength: reader.Size(), fileName: "fleet-mdm-enrollment-profile"}, nil
}

func (svc *Service) GetOTAProfile(ctx context.Context, enrollSecret string) ([]byte, error) {
	// Skip authz as this endpoint is used by end users from their iPhones or iPads; authz is done
	// by the enroll secret verification below
	svc.authz.SkipAuthorization(ctx)

	cfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting app config to get org name")
	}

	profBytes, err := apple_mdm.GenerateOTAEnrollmentProfileMobileconfig(cfg.OrgInfo.OrgName, cfg.MDMUrl(), enrollSecret)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generating ota mobileconfig file")
	}

	signed, err := mdmcrypto.Sign(ctx, profBytes, svc.ds)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "signing profile")
	}

	return signed, nil
}

////////////////////////////////////////////////////////////////////////////////
// POST /ota_enrollment?enroll_secret=xyz
////////////////////////////////////////////////////////////////////////////////

type mdmAppleOTARequest struct {
	EnrollSecret string `query:"enroll_secret"`
	Certificates []*x509.Certificate
	RootSigner   *x509.Certificate
	DeviceInfo   fleet.MDMAppleMachineInfo
}

func (mdmAppleOTARequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	enrollSecret := r.URL.Query().Get("enroll_secret")
	if enrollSecret == "" {
		return nil, &fleet.OTAForbiddenError{
			InternalErr: errors.New("enroll_secret query parameter was empty"),
		}
	}

	rawData, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "reading body from request")
	}

	p7, err := pkcs7.Parse(rawData)
	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "invalid request body",
			InternalErr: err,
		}
	}

	var request mdmAppleOTARequest
	err = plist.Unmarshal(p7.Content, &request.DeviceInfo)
	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "invalid request body",
			InternalErr: err,
		}
	}

	if request.DeviceInfo.Serial == "" {
		return nil, &fleet.BadRequestError{
			Message: "SERIAL is required",
		}
	}

	request.EnrollSecret = enrollSecret
	request.Certificates = p7.Certificates
	request.RootSigner = p7.GetOnlySigner()
	return &request, nil
}

type mdmAppleOTAResponse struct {
	Err error `json:"error,omitempty"`
	xml []byte
}

func (r mdmAppleOTAResponse) error() error { return r.Err }

func (r mdmAppleOTAResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(r.xml)))
	w.Header().Set("Content-Type", "application/x-apple-aspen-config")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if _, err := w.Write(r.xml); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func mdmAppleOTAEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*mdmAppleOTARequest)
	xml, err := svc.MDMAppleProcessOTAEnrollment(ctx, req.Certificates, req.RootSigner, req.EnrollSecret, req.DeviceInfo)
	if err != nil {
		return mdmAppleGetInstallerResponse{Err: err}, nil
	}
	return mdmAppleOTAResponse{xml: xml}, nil
}

// NOTE: this method and how OTA works is documented in full in the interface definition.
func (svc *Service) MDMAppleProcessOTAEnrollment(
	ctx context.Context,
	certificates []*x509.Certificate,
	rootSigner *x509.Certificate,
	enrollSecret string,
	deviceInfo fleet.MDMAppleMachineInfo,
) ([]byte, error) {
	// authorization is performed via the enroll secret and the provided certificates
	svc.authz.SkipAuthorization(ctx)

	if len(certificates) == 0 {
		return nil, authz.ForbiddenWithInternal("no certificates provided", nil, nil, nil)
	}

	// first check is for the enroll secret, we'll only let the host
	// through if it has a valid secret.
	enrollSecretInfo, err := svc.ds.VerifyEnrollSecret(ctx, enrollSecret)
	if err != nil {
		if fleet.IsNotFound(err) {
			return nil, &fleet.OTAForbiddenError{
				InternalErr: err,
			}
		}

		return nil, ctxerr.Wrap(ctx, err, "validating enroll secret")
	}

	assets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetSCEPChallenge,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("loading SCEP challenge from the database: %w", err)
	}
	scepChallenge := string(assets[fleet.MDMAssetSCEPChallenge].Value)

	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "reading app config")
	}

	mdmURL := appCfg.MDMUrl()

	// if the root signer was issued by Apple's CA, it means we're in the
	// first phase and we should return a SCEP payload.
	if err := apple_mdm.VerifyFromAppleIphoneDeviceCA(rootSigner); err == nil {
		scepURL, err := apple_mdm.ResolveAppleSCEPURL(mdmURL)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "resolve Apple SCEP url")
		}

		var buf bytes.Buffer
		if err := apple_mdm.OTASCEPTemplate.Execute(&buf, struct {
			SCEPURL       string
			SCEPChallenge string
		}{
			SCEPURL:       scepURL,
			SCEPChallenge: scepChallenge,
		}); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "execute template")
		}
		return buf.Bytes(), nil
	}

	// otherwise we might be in the second phase, check if the signing cert
	// was issued by Fleet, only let the enrollment through if so.
	certVerifier := mdmcrypto.NewSCEPVerifier(svc.ds)
	if err := certVerifier.Verify(ctx, rootSigner); err != nil {
		return nil, authz.ForbiddenWithInternal(fmt.Sprintf("payload signed with invalid certificate: %s", err), nil, nil, nil)
	}

	topic, err := svc.mdmPushCertTopic(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "extracting topic from APNs cert")
	}

	enrollmentProf, err := apple_mdm.GenerateEnrollmentProfileMobileconfig(
		appCfg.OrgInfo.OrgName,
		mdmURL,
		string(assets[fleet.MDMAssetSCEPChallenge].Value),
		topic,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generating manual enrollment profile")
	}

	// before responding, create a host record, and assign the host to the
	// team that matches the enroll secret provided.
	err = svc.ds.IngestMDMAppleDeviceFromOTAEnrollment(ctx, enrollSecretInfo.TeamID, deviceInfo)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating new host record")
	}

	// at this point we know the device can be enrolled, so we respond with
	// a signed enrollment profile
	signed, err := mdmcrypto.Sign(ctx, enrollmentProf, svc.ds)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "signing profile")
	}

	return signed, nil
}
