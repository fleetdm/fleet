package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/gorilla/mux"
)

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	return endpoint_utils.EncodeCommonResponse(ctx, w, response, jsonMarshal)
}

func jsonMarshal(w http.ResponseWriter, response interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(response)
}

func uint32FromRequest(r *http.Request, name string) (uint32, error) {
	vars := mux.Vars(r)
	s, ok := vars[name]
	if !ok {
		return 0, endpoint_utils.ErrBadRoute
	}
	u, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, ctxerr.Wrap(r.Context(), err, "uint32FromRequest")
	}
	return uint32(u), nil
}

// default number of items to include per page
const defaultPerPage = 20

// listOptionsFromRequest parses the list options from the request parameters
func listOptionsFromRequest(r *http.Request) (fleet.ListOptions, error) {
	var err error

	pageString := r.URL.Query().Get("page")
	perPageString := r.URL.Query().Get("per_page")
	orderKey := r.URL.Query().Get("order_key")
	orderDirectionString := r.URL.Query().Get("order_direction")
	afterString := r.URL.Query().Get("after")

	var page int
	if pageString != "" {
		page, err = strconv.Atoi(pageString)
		if err != nil {
			return fleet.ListOptions{}, ctxerr.Wrap(r.Context(), badRequest("non-int page value"))
		}
		if page < 0 {
			return fleet.ListOptions{}, ctxerr.Wrap(r.Context(), badRequest("negative page value"))
		}
	}

	// We default to 0 for per_page so that not specifying any paging
	// information gets all results
	var perPage int
	if perPageString != "" {
		perPage, err = strconv.Atoi(perPageString)
		if err != nil {
			return fleet.ListOptions{}, ctxerr.Wrap(r.Context(), badRequest("non-int per_page value"))
		}
		if perPage <= 0 {
			return fleet.ListOptions{}, ctxerr.Wrap(r.Context(), badRequest("invalid per_page value"))
		}
	}

	if perPage == 0 && pageString != "" {
		// We explicitly set a non-zero default if a page is specified
		// (because the client probably intended for paging, and
		// leaving the 0 would turn that off)
		perPage = defaultPerPage
	}

	if orderKey == "" && orderDirectionString != "" {
		return fleet.ListOptions{}, ctxerr.Wrap(r.Context(), badRequest("order_key must be specified with order_direction"))
	}

	if orderKey == "" && afterString != "" {
		return fleet.ListOptions{}, ctxerr.Wrap(r.Context(), badRequest("order_key must be specified with after"))
	}

	var orderDirection fleet.OrderDirection
	switch orderDirectionString {
	case "desc":
		orderDirection = fleet.OrderDescending
	case "asc":
		orderDirection = fleet.OrderAscending
	case "":
		orderDirection = fleet.OrderAscending
	default:
		return fleet.ListOptions{},
			ctxerr.Wrap(r.Context(), badRequest("unknown order_direction: "+orderDirectionString))

	}

	query := r.URL.Query().Get("query")

	return fleet.ListOptions{
		Page:           uint(page),    //nolint:gosec // dismiss G115
		PerPage:        uint(perPage), //nolint:gosec // dismiss G115
		OrderKey:       orderKey,
		OrderDirection: orderDirection,
		MatchQuery:     strings.TrimSpace(query),
		After:          afterString,
	}, nil
}

func hostListOptionsFromRequest(r *http.Request) (fleet.HostListOptions, error) {
	opt, err := listOptionsFromRequest(r)
	if err != nil {
		return fleet.HostListOptions{}, err
	}

	hopt := fleet.HostListOptions{ListOptions: opt}

	status := r.URL.Query().Get("status")
	switch fleet.HostStatus(status) {
	case fleet.StatusNew, fleet.StatusOnline, fleet.StatusOffline, fleet.StatusMIA, fleet.StatusMissing:
		hopt.StatusFilter = fleet.HostStatus(status)
	case "":
		// No error when unset
	default:
		return hopt, ctxerr.Wrap(r.Context(), badRequest(fmt.Sprintf("Invalid status: %s", status)))

	}

	additionalInfoFiltersString := r.URL.Query().Get("additional_info_filters")
	if additionalInfoFiltersString != "" {
		hopt.AdditionalFilters = strings.Split(additionalInfoFiltersString, ",")
	}

	teamID := r.URL.Query().Get("team_id")
	if teamID != "" {
		id, err := strconv.ParseUint(teamID, 10, 32)
		if err != nil {
			return hopt, ctxerr.Wrap(r.Context(), badRequest(fmt.Sprintf("Invalid team_id: %s", teamID)))
		}
		tid := uint(id)
		hopt.TeamFilter = &tid
	}

	policyID := r.URL.Query().Get("policy_id")
	if policyID != "" {
		id, err := strconv.ParseUint(policyID, 10, 32)
		if err != nil {
			return hopt, ctxerr.Wrap(r.Context(), badRequest(fmt.Sprintf("Invalid policy_id: %s", policyID)))
		}
		pid := uint(id)
		hopt.PolicyIDFilter = &pid
	}

	policyResponse := r.URL.Query().Get("policy_response")
	if policyResponse != "" {
		if hopt.PolicyIDFilter == nil {
			return hopt, ctxerr.Wrap(
				r.Context(), badRequest(
					"Missing policy_id (it must be present when policy_response is specified)",
				),
			)
		}
		var v *bool
		switch policyResponse {
		case "passing":
			v = ptr.Bool(true)
		case "failing":
			v = ptr.Bool(false)
		default:
			return hopt, ctxerr.Wrap(
				r.Context(),
				badRequest(
					fmt.Sprintf(
						"Invalid policy_response: %v (Valid options are 'passing' or 'failing')",
						policyResponse,
					),
				),
			)
		}
		hopt.PolicyResponseFilter = v
	}

	softwareID := r.URL.Query().Get("software_id")
	if softwareID != "" {
		id, err := strconv.ParseUint(softwareID, 10, 64)
		if err != nil {
			return hopt, ctxerr.Wrap(r.Context(), badRequest(fmt.Sprintf("Invalid software_id: %s", softwareID)))
		}
		sid := uint(id)
		hopt.SoftwareIDFilter = &sid
	}

	softwareVersionID := r.URL.Query().Get("software_version_id")
	if softwareVersionID != "" {
		id, err := strconv.ParseUint(softwareVersionID, 10, 64)
		if err != nil {
			return hopt, ctxerr.Wrap(r.Context(), badRequest(fmt.Sprintf("Invalid software_version_id: %s", softwareVersionID)))
		}
		sid := uint(id)
		hopt.SoftwareVersionIDFilter = &sid
	}

	softwareTitleID := r.URL.Query().Get("software_title_id")
	if softwareTitleID != "" {
		id, err := strconv.ParseUint(softwareTitleID, 10, 32)
		if err != nil {
			return hopt, ctxerr.Wrap(r.Context(), badRequest(fmt.Sprintf("Invalid software_title_id: %s", softwareTitleID)))
		}
		sid := uint(id)
		hopt.SoftwareTitleIDFilter = &sid
	}

	softwareStatus := fleet.SoftwareInstallerStatus(strings.ToLower(r.URL.Query().Get("software_status")))
	if softwareStatus != "" {
		if !softwareStatus.IsValid() {
			return hopt, ctxerr.Wrap(
				r.Context(), badRequest(fmt.Sprintf("Invalid software_status: %s", softwareStatus)),
			)
		}
		if hopt.SoftwareTitleIDFilter == nil {
			return hopt, ctxerr.Wrap(
				r.Context(), badRequest(
					"Missing software_title_id (it must be present when software_status is specified)",
				),
			)
		}
		if hopt.TeamFilter == nil {
			return hopt, ctxerr.Wrap(
				r.Context(), badRequest(
					"Missing team_id (it must be present when software_status is specified)",
				),
			)
		}
		hopt.SoftwareStatusFilter = &softwareStatus
	}

	osID := r.URL.Query().Get("os_id")
	if osID != "" {
		id, err := strconv.ParseUint(osID, 10, 32)
		if err != nil {
			return hopt, ctxerr.Wrap(r.Context(), badRequest(fmt.Sprintf("Invalid os_id: %s", osID)))
		}
		sid := uint(id)
		hopt.OSIDFilter = &sid
	}

	osVersionID := r.URL.Query().Get("os_version_id")
	if osVersionID != "" {
		id, err := strconv.ParseUint(osVersionID, 10, 32)
		if err != nil {
			return hopt, ctxerr.Wrap(r.Context(), badRequest(fmt.Sprintf("Invalid os_version_id: %s", osVersionID)))
		}
		sid := uint(id)
		hopt.OSVersionIDFilter = &sid
	}

	osName := r.URL.Query().Get("os_name")
	if osName != "" {
		hopt.OSNameFilter = &osName
	}

	osVersion := r.URL.Query().Get("os_version")
	if osVersion != "" {
		hopt.OSVersionFilter = &osVersion
	}

	cve := r.URL.Query().Get("vulnerability")
	if cve != "" {
		hopt.VulnerabilityFilter = &cve
	}

	if hopt.OSNameFilter != nil && hopt.OSVersionFilter == nil {
		return hopt, ctxerr.Wrap(
			r.Context(), badRequest(
				"Invalid os_version (os_version must be specified with os_name)",
			),
		)
	}
	if hopt.OSNameFilter == nil && hopt.OSVersionFilter != nil {
		return hopt, ctxerr.Wrap(
			r.Context(), badRequest(
				"Invalid os_name (os_name must be specified with os_version)",
			),
		)
	}

	// disable_failing_policies is a deprecated parameter and an alias for disable_issues
	// disable_issues is the new parameter name, which takes precedence over disable_failing_policies
	disableFailingPolicies := r.URL.Query().Get("disable_failing_policies")
	disableIssues := r.URL.Query().Get("disable_issues")
	if disableIssues != "" {
		boolVal, err := strconv.ParseBool(disableIssues)
		if err != nil {
			return hopt, ctxerr.Wrap(
				r.Context(), badRequest(
					fmt.Sprintf(
						"Invalid disable_issues: %s",
						disableIssues,
					),
				),
			)
		}
		hopt.DisableIssues = boolVal
	} else if disableFailingPolicies != "" {
		boolVal, err := strconv.ParseBool(disableFailingPolicies)
		if err != nil {
			return hopt, ctxerr.Wrap(
				r.Context(), badRequest(
					fmt.Sprintf(
						"Invalid disable_failing_policies: %s",
						disableFailingPolicies,
					),
				),
			)
		}
		hopt.DisableIssues = boolVal
	}
	if hopt.DisableIssues && r.URL.Query().Get("order_key") == "issues" {
		return hopt, ctxerr.Wrap(
			r.Context(), badRequest(
				"Invalid order_key (issues cannot be ordered when they are disabled)",
			),
		)
	}

	deviceMapping := r.URL.Query().Get("device_mapping")
	if deviceMapping != "" {
		boolVal, err := strconv.ParseBool(deviceMapping)
		if err != nil {
			return hopt, ctxerr.Wrap(r.Context(), badRequest(fmt.Sprintf("Invalid device_mapping: %s", deviceMapping)))
		}
		hopt.DeviceMapping = boolVal
	}

	mdmID := r.URL.Query().Get("mdm_id")
	if mdmID != "" {
		id, err := strconv.ParseUint(mdmID, 10, 32)
		if err != nil {
			return hopt, ctxerr.Wrap(r.Context(), badRequest(fmt.Sprintf("Invalid mdm_id: %s", mdmID)))
		}
		mid := uint(id)
		hopt.MDMIDFilter = &mid
	}

	if mdmName := r.URL.Query().Get("mdm_name"); mdmName != "" {
		hopt.MDMNameFilter = &mdmName
	}

	enrollmentStatus := r.URL.Query().Get("mdm_enrollment_status")
	switch fleet.MDMEnrollStatus(enrollmentStatus) {
	case fleet.MDMEnrollStatusManual, fleet.MDMEnrollStatusAutomatic, fleet.MDMEnrollStatusPersonal,
		fleet.MDMEnrollStatusPending, fleet.MDMEnrollStatusUnenrolled, fleet.MDMEnrollStatusEnrolled:
		hopt.MDMEnrollmentStatusFilter = fleet.MDMEnrollStatus(enrollmentStatus)
	case "":
		// No error when unset
	default:
		return hopt, ctxerr.Wrap(
			r.Context(), badRequest(fmt.Sprintf("Invalid mdm_enrollment_status: %s", enrollmentStatus)),
		)
	}

	connectedToFleet := r.URL.Query().Has("connected_to_fleet")
	if connectedToFleet {
		hopt.ConnectedToFleetFilter = ptr.Bool(true)
	}

	macOSSettingsStatus := r.URL.Query().Get("macos_settings")
	switch fleet.OSSettingsStatus(macOSSettingsStatus) {
	case fleet.OSSettingsFailed, fleet.OSSettingsPending, fleet.OSSettingsVerifying, fleet.OSSettingsVerified:
		hopt.MacOSSettingsFilter = fleet.OSSettingsStatus(macOSSettingsStatus)
	case "":
		// No error when unset
	default:
		return hopt, ctxerr.Wrap(
			r.Context(), badRequest(fmt.Sprintf("Invalid macos_settings: %s", macOSSettingsStatus)),
		)
	}

	macOSSettingsDiskEncryptionStatus := r.URL.Query().Get("macos_settings_disk_encryption")
	switch fleet.DiskEncryptionStatus(macOSSettingsDiskEncryptionStatus) {
	case
		fleet.DiskEncryptionVerifying,
		fleet.DiskEncryptionVerified,
		fleet.DiskEncryptionActionRequired,
		fleet.DiskEncryptionEnforcing,
		fleet.DiskEncryptionFailed,
		fleet.DiskEncryptionRemovingEnforcement:
		hopt.MacOSSettingsDiskEncryptionFilter = fleet.DiskEncryptionStatus(macOSSettingsDiskEncryptionStatus)
	case "":
		// No error when unset
	default:
		return hopt, ctxerr.Wrap(
			r.Context(),
			badRequest(fmt.Sprintf("Invalid macos_settings_disk_encryption: %s", macOSSettingsDiskEncryptionStatus)),
		)
	}

	osSettingsStatus := r.URL.Query().Get("os_settings")
	switch fleet.OSSettingsStatus(osSettingsStatus) {
	case fleet.OSSettingsFailed, fleet.OSSettingsPending, fleet.OSSettingsVerifying, fleet.OSSettingsVerified:
		hopt.OSSettingsFilter = fleet.OSSettingsStatus(osSettingsStatus)
	case "":
		// No error when unset
	default:
		return hopt, ctxerr.Wrap(
			r.Context(), badRequest(fmt.Sprintf("Invalid os_settings: %s", osSettingsStatus)),
		)
	}

	osSettingsDiskEncryptionStatus := r.URL.Query().Get("os_settings_disk_encryption")
	switch fleet.DiskEncryptionStatus(osSettingsDiskEncryptionStatus) {
	case
		fleet.DiskEncryptionVerifying,
		fleet.DiskEncryptionVerified,
		fleet.DiskEncryptionActionRequired,
		fleet.DiskEncryptionEnforcing,
		fleet.DiskEncryptionFailed,
		fleet.DiskEncryptionRemovingEnforcement:
		hopt.OSSettingsDiskEncryptionFilter = fleet.DiskEncryptionStatus(osSettingsDiskEncryptionStatus)
	case "":
		// No error when unset
	default:
		return hopt, ctxerr.Wrap(
			r.Context(),
			badRequest(fmt.Sprintf("Invalid os_settings_disk_encryption: %s", macOSSettingsDiskEncryptionStatus)),
		)
	}

	mdmBootstrapPackageStatus := r.URL.Query().Get("bootstrap_package")
	switch fleet.MDMBootstrapPackageStatus(mdmBootstrapPackageStatus) {
	case fleet.MDMBootstrapPackageFailed, fleet.MDMBootstrapPackagePending, fleet.MDMBootstrapPackageInstalled:
		bpf := fleet.MDMBootstrapPackageStatus(mdmBootstrapPackageStatus)
		hopt.MDMBootstrapPackageFilter = &bpf
	case "":
		// No error when unset
	default:
		return hopt, ctxerr.Wrap(
			r.Context(), badRequest(fmt.Sprintf("Invalid bootstrap_package: %s", mdmBootstrapPackageStatus)),
		)
	}

	profileUUID := r.URL.Query().Get("profile_uuid")
	profileStatus := r.URL.Query().Get("profile_status")
	switch {
	case profileUUID != "" && profileStatus != "":
		hopt.ProfileUUIDFilter = &profileUUID
		if fleet.OSSettingsStatus(profileStatus).IsValid() {
			psf := fleet.OSSettingsStatus(profileStatus)
			hopt.ProfileStatusFilter = &psf
		} else {
			return hopt, ctxerr.Wrap(r.Context(), badRequest(fmt.Sprintf("Invalid profile_status: %s", profileStatus)))
		}
	case profileUUID != "" && profileStatus == "":
		return hopt, ctxerr.Wrap(r.Context(), badRequest("Missing profile_status (it must be present when profile_uuid is specified)"))
	case profileUUID == "" && profileStatus != "":
		return hopt, ctxerr.Wrap(r.Context(), badRequest("Missing profile_uuid (it must be present when profile_status is specified)"))
	}

	munkiIssueID := r.URL.Query().Get("munki_issue_id")
	if munkiIssueID != "" {
		id, err := strconv.ParseUint(munkiIssueID, 10, 32)
		if err != nil {
			return hopt, ctxerr.Wrap(r.Context(), badRequest(fmt.Sprintf("Invalid munki_issue_id: %s", munkiIssueID)))
		}
		mid := uint(id)
		hopt.MunkiIssueIDFilter = &mid
	}

	lowDiskSpace := r.URL.Query().Get("low_disk_space")
	if lowDiskSpace != "" {
		v, err := strconv.Atoi(lowDiskSpace)
		if err != nil {
			return hopt, ctxerr.Wrap(r.Context(), badRequest(fmt.Sprintf("Invalid low_disk_space: %s", lowDiskSpace)))
		}
		if v < 1 || v > 100 {
			return hopt, ctxerr.Wrap(
				r.Context(), badRequest(
					fmt.Sprintf(
						"Invalid low_disk_space, must be between 1 and 100: %s", lowDiskSpace,
					),
				),
			)
		}
		hopt.LowDiskSpaceFilter = &v
	}

	batchScriptExecutionID := r.URL.Query().Get("script_batch_execution_id")
	if batchScriptExecutionID != "" {
		hopt.BatchScriptExecutionIDFilter = &batchScriptExecutionID
		batchScriptExecutionStatus := r.URL.Query().Get("script_batch_execution_status")
		if batchScriptExecutionStatus != "" {
			if fleet.BatchScriptExecutionStatus(batchScriptExecutionStatus).IsValid() {
				bsef := fleet.BatchScriptExecutionStatus(batchScriptExecutionStatus)
				hopt.BatchScriptExecutionStatusFilter = bsef
			} else {
				return hopt, ctxerr.Wrap(r.Context(), badRequest(fmt.Sprintf("Invalid script_batch_execution_status: %s", batchScriptExecutionStatus)))
			}
		}
	}

	populateSoftware := r.URL.Query().Get("populate_software")
	if populateSoftware == "without_vulnerability_details" {
		hopt.PopulateSoftware = true
		hopt.PopulateSoftwareVulnerabilityDetails = false
	} else if populateSoftware != "" {
		ps, err := strconv.ParseBool(populateSoftware)
		if err != nil {
			return hopt, ctxerr.Wrap(r.Context(), badRequest(`Invalid value for populate_software. Should be one of "true", "false", or "without_vulnerability_details".`))
		}
		hopt.PopulateSoftware = ps
		hopt.PopulateSoftwareVulnerabilityDetails = ps
	}

	populatePolicies := r.URL.Query().Get("populate_policies")
	if populatePolicies != "" {
		pp, err := strconv.ParseBool(populatePolicies)
		if err != nil {
			return hopt, ctxerr.Wrap(
				r.Context(), badRequest(fmt.Sprintf("Invalid boolean parameter populate_policies: %s", populateSoftware)),
			)
		}
		hopt.PopulatePolicies = pp
	}

	populateUsers := r.URL.Query().Get("populate_users")
	if populateUsers != "" {
		pu, err := strconv.ParseBool(populateUsers)
		if err != nil {
			return hopt, ctxerr.Wrap(
				r.Context(), badRequest(fmt.Sprintf("Invalid boolean parameter populate_users: %s", populateUsers)),
			)
		}
		hopt.PopulateUsers = pu
	}

	populateLabels := r.URL.Query().Get("populate_labels")
	if populateLabels != "" {
		pl, err := strconv.ParseBool(populateLabels)
		if err != nil {
			return hopt, ctxerr.Wrap(
				r.Context(), badRequest(fmt.Sprintf("Invalid boolean parameter populate_labels: %s", populateLabels)),
			)
		}
		hopt.PopulateLabels = pl
	}

	// cannot combine software_id, software_version_id, and software_title_id
	var softwareErrorLabel []string
	if hopt.SoftwareIDFilter != nil {
		softwareErrorLabel = append(softwareErrorLabel, "software_id")
	}
	if hopt.SoftwareVersionIDFilter != nil {
		softwareErrorLabel = append(softwareErrorLabel, "software_version_id")
	}
	if hopt.SoftwareTitleIDFilter != nil {
		softwareErrorLabel = append(softwareErrorLabel, "software_title_id")
	}
	if len(softwareErrorLabel) > 1 {
		return hopt, ctxerr.Wrap(r.Context(), badRequest(fmt.Sprintf("Invalid parameters. The combination of %s is not allowed.", strings.Join(softwareErrorLabel, " and "))))
	}

	return hopt, nil
}

func carveListOptionsFromRequest(r *http.Request) (fleet.CarveListOptions, error) {
	opt, err := listOptionsFromRequest(r)
	if err != nil {
		return fleet.CarveListOptions{}, err
	}

	carveOpts := fleet.CarveListOptions{ListOptions: opt}

	expired := r.URL.Query().Get("expired")
	if expired == "" {
		carveOpts.Expired = false
	} else {
		boolVal, err := strconv.ParseBool(expired)
		if err != nil {
			return carveOpts, ctxerr.Wrap(
				r.Context(), badRequest(
					fmt.Sprintf("Invalid expired: %s", expired),
				),
			)
		}
		carveOpts.Expired = boolVal
	}
	return carveOpts, nil
}

func userListOptionsFromRequest(r *http.Request) (fleet.UserListOptions, error) {
	opt, err := listOptionsFromRequest(r)
	if err != nil {
		return fleet.UserListOptions{}, err
	}

	userOpts := fleet.UserListOptions{ListOptions: opt}

	if tid := r.URL.Query().Get("team_id"); tid != "" {
		teamID, err := strconv.ParseUint(tid, 10, 64)
		if err != nil {
			return userOpts, ctxerr.Wrap(r.Context(), badRequest(fmt.Sprintf("Invalid team_id: %s", tid)))
		}
		// GitHub CodeQL flags this as: Incorrect conversion between integer types. Previously waived: https://github.com/fleetdm/fleet/security/code-scanning/516
		userOpts.TeamID = uint(teamID)
	}

	return userOpts, nil
}

type getGenericSpecRequest struct {
	Name string `url:"name"`
}
