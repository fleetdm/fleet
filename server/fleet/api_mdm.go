package fleet

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/logging"
)

type GetAppleMDMResponse struct {
	*AppleMDM
	Err error `json:"error,omitempty"`
}

func (r GetAppleMDMResponse) Error() error { return r.Err }

type GetAppleBMResponse struct {
	*AppleBM
	Err error `json:"error,omitempty"`
}

func (r GetAppleBMResponse) Error() error { return r.Err }

type RequestMDMAppleCSRRequest struct {
	EmailAddress string `json:"email_address"`
	Organization string `json:"organization"`
}

type RequestMDMAppleCSRResponse struct {
	*AppleCSR
	Err error `json:"error,omitempty"`
}

func (r RequestMDMAppleCSRResponse) Error() error { return r.Err }

type CreateMDMEULARequest struct {
	EULA   *multipart.FileHeader
	DryRun bool `query:"dry_run,optional"` // if true, apply validation but do not save changes
}

type CreateMDMEULAResponse struct {
	Err error `json:"error,omitempty"`
}

func (r CreateMDMEULAResponse) Error() error { return r.Err }

type GetMDMEULARequest struct {
	Token string `url:"token"`
}

type GetMDMEULAResponse struct {
	Err error `json:"error,omitempty"`

	// EULA is used in HijackRender to build the response
	EULA *MDMEULA
}

func (r GetMDMEULAResponse) Error() error { return r.Err }

func (r GetMDMEULAResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.Itoa(len(r.EULA.Bytes)))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided
	if n, err := w.Write(r.EULA.Bytes); err != nil {
		logging.WithExtras(ctx, "err", err, "bytes_copied", n)
	}
}

type GetMDMEULAMetadataRequest struct{}

type GetMDMEULAMetadataResponse struct {
	*MDMEULA
	Err error `json:"error,omitempty"`
}

func (r GetMDMEULAMetadataResponse) Error() error { return r.Err }

type DeleteMDMEULARequest struct {
	Token  string `url:"token"`
	DryRun bool   `query:"dry_run,optional"` // if true, apply validation but do not delete
}

type DeleteMDMEULAResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteMDMEULAResponse) Error() error { return r.Err }

type RunMDMCommandRequest struct {
	Command   string   `json:"command"`
	HostUUIDs []string `json:"host_uuids"`
}

type RunMDMCommandResponse struct {
	*CommandEnqueueResult
	Err error `json:"error,omitempty"`
}

func (r RunMDMCommandResponse) Error() error { return r.Err }

type GetMDMCommandResultsRequest struct {
	CommandUUID    string `query:"command_uuid,optional"`
	HostIdentifier string `query:"host_identifier,optional"`
}

type GetMDMCommandResultsResponse struct {
	Results []*MDMCommandResult `json:"results,omitempty"`
	Err     error               `json:"error,omitempty"`
}

func (r GetMDMCommandResultsResponse) Error() error { return r.Err }

type ListMDMCommandsRequest struct {
	ListOptions    ListOptions `url:"list_options"`
	HostIdentifier string      `query:"host_identifier,optional"`
	RequestType    string      `query:"request_type,optional"`
	CommandStatus  string      `query:"command_status,optional"`
}

func (req ListMDMCommandsRequest) DecodeBody(ctx context.Context, r io.Reader, u url.Values, c []*x509.Certificate) error {
	if req.CommandStatus != "" && req.HostIdentifier == "" {
		return &BadRequestError{
			Message: `"host_identifier" must be specified when filtering by "command_status".`,
		}
	}

	if req.CommandStatus != "" {
		statuses := strings.Split(req.CommandStatus, ",")
		failed := false
		for _, status := range statuses {
			status = strings.TrimSpace(status)
			if !slices.Contains(AllMDMCommandStatusFilters, MDMCommandStatusFilter(status)) {
				failed = true
				break
			}
		}

		if failed {
			allowed := make([]string, len(AllMDMCommandStatusFilters))
			for i, v := range AllMDMCommandStatusFilters {
				allowed[i] = string(v)
			}
			return &BadRequestError{
				Message: fmt.Sprintf("command_status only accepts the following values: %s", strings.Join(allowed, ", ")),
			}
		}
	}

	return nil
}

type ListMDMCommandsResponse struct {
	Meta    *PaginationMetadata `json:"meta"`
	Count   *int64              `json:"count"`
	Results []*MDMCommand       `json:"results"`
	Err     error               `json:"error,omitempty"`
}

func (r ListMDMCommandsResponse) Error() error { return r.Err }

type GetMDMDiskEncryptionSummaryRequest struct {
	TeamID *uint `query:"team_id,optional" renameto:"fleet_id"`
}

type GetMDMDiskEncryptionSummaryResponse struct {
	*MDMDiskEncryptionSummary
	Err error `json:"error,omitempty"`
}

func (r GetMDMDiskEncryptionSummaryResponse) Error() error { return r.Err }

type GetMDMProfilesSummaryRequest struct {
	TeamID *uint `query:"team_id,optional" renameto:"fleet_id"`
}

type GetMDMProfilesSummaryResponse struct {
	MDMProfilesSummary
	Err error `json:"error,omitempty"`
}

func (r GetMDMProfilesSummaryResponse) Error() error { return r.Err }

type GetMDMConfigProfileRequest struct {
	ProfileUUID string `url:"profile_uuid"`
	Alt         string `query:"alt,optional"`
}

type GetMDMConfigProfileResponse struct {
	*MDMConfigProfilePayload
	Err error `json:"error,omitempty"`
}

func (r GetMDMConfigProfileResponse) Error() error { return r.Err }

type DeleteMDMConfigProfileRequest struct {
	ProfileUUID string `url:"profile_uuid"`
}

type DeleteMDMConfigProfileResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteMDMConfigProfileResponse) Error() error { return r.Err }

type NewMDMConfigProfileRequest struct {
	TeamID           uint
	Profile          *multipart.FileHeader
	LabelsIncludeAll []string
	LabelsIncludeAny []string
	LabelsExcludeAny []string
}

type NewMDMConfigProfileResponse struct {
	ProfileUUID string `json:"profile_uuid"`
	Err         error  `json:"error,omitempty"`
}

func (r NewMDMConfigProfileResponse) Error() error { return r.Err }

type BatchModifyMDMConfigProfilesRequest struct {
	TeamID                *uint                                `json:"-" query:"team_id,optional" renameto:"fleet_id"`
	TeamName              *string                              `json:"-" query:"team_name,optional" renameto:"fleet_name"`
	DryRun                bool                                 `json:"-" query:"dry_run,optional"` // if true, apply validation but do not save changes
	ConfigurationProfiles []BatchModifyMDMConfigProfilePayload `json:"configuration_profiles"`
}

type BatchModifyMDMConfigProfilesResponse struct {
	Err error `json:"error,omitempty"`
}

func (r BatchModifyMDMConfigProfilesResponse) Error() error { return r.Err }

func (r BatchModifyMDMConfigProfilesResponse) Status() int { return http.StatusNoContent }

// BackwardsCompatProfilesParam supports both the old map format and the new
// array format for MDM profile batch payloads.
type BackwardsCompatProfilesParam []MDMProfileBatchPayload

func (bcp *BackwardsCompatProfilesParam) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	if lookAhead := bytes.TrimSpace(data); len(lookAhead) > 0 && lookAhead[0] == '[' {
		// use []MDMProfileBatchPayload to prevent infinite recursion if we
		// use `BackwardsCompatProfilesParam`
		var profs []MDMProfileBatchPayload
		if err := json.Unmarshal(data, &profs); err != nil {
			return fmt.Errorf("unmarshal profile spec. Error using new format: %w", err)
		}
		*bcp = profs
		return nil
	}

	var backwardsCompat map[string][]byte
	if err := json.Unmarshal(data, &backwardsCompat); err != nil {
		return fmt.Errorf("unmarshal profile spec. Error using old format: %w", err)
	}

	*bcp = make(BackwardsCompatProfilesParam, 0, len(backwardsCompat))
	for name, contents := range backwardsCompat {
		*bcp = append(*bcp, MDMProfileBatchPayload{Name: name, Contents: contents})
	}
	return nil
}

type BatchSetMDMProfilesRequest struct {
	TeamID        *uint                        `json:"-" query:"team_id,optional" renameto:"fleet_id"`
	TeamName      *string                      `json:"-" query:"team_name,optional" renameto:"fleet_name"`
	DryRun        bool                         `json:"-" query:"dry_run,optional"`        // if true, apply validation but do not save changes
	AssumeEnabled *bool                        `json:"-" query:"assume_enabled,optional"` // if true, assume MDM is enabled
	Profiles      BackwardsCompatProfilesParam `json:"profiles"`
	NoCache       bool                         `json:"-" query:"no_cache,optional"`
}

type BatchSetMDMProfilesResponse struct {
	Err error `json:"error,omitempty"`
}

func (r BatchSetMDMProfilesResponse) Error() error { return r.Err }

func (r BatchSetMDMProfilesResponse) Status() int { return http.StatusNoContent }

type ListMDMConfigProfilesRequest struct {
	TeamID      *uint       `query:"team_id,optional" renameto:"fleet_id"`
	ListOptions ListOptions `url:"list_options"`
}

type ListMDMConfigProfilesResponse struct {
	Meta     *PaginationMetadata        `json:"meta"`
	Profiles []*MDMConfigProfilePayload `json:"profiles"`
	Err      error                      `json:"error,omitempty"`
}

func (r ListMDMConfigProfilesResponse) Error() error { return r.Err }

type UpdateDiskEncryptionRequest struct {
	TeamID               *uint `json:"team_id" renameto:"fleet_id"`
	EnableDiskEncryption bool  `json:"enable_disk_encryption"`
	RequireBitLockerPIN  bool  `json:"windows_require_bitlocker_pin"`
}

type UpdateMDMDiskEncryptionResponse struct {
	Err error `json:"error,omitempty"`
}

func (r UpdateMDMDiskEncryptionResponse) Error() error { return r.Err }

func (r UpdateMDMDiskEncryptionResponse) Status() int { return http.StatusNoContent }

type ResendHostMDMProfileRequest struct {
	HostID      uint   `url:"host_id"`
	ProfileUUID string `url:"profile_uuid"`
}

type ResendHostMDMProfileResponse struct {
	Err error `json:"error,omitempty"`
}

func (r ResendHostMDMProfileResponse) Error() error { return r.Err }

func (r ResendHostMDMProfileResponse) Status() int { return http.StatusAccepted }

type GetMDMAppleCSRRequest struct{}

type GetMDMAppleCSRResponse struct {
	CSR []byte `json:"csr"` // base64 encoded
	Err error  `json:"error,omitempty"`
}

func (r GetMDMAppleCSRResponse) Error() error { return r.Err }

type UploadMDMAppleAPNSCertRequest struct {
	File *multipart.FileHeader
}

type UploadMDMAppleAPNSCertResponse struct {
	Err error `json:"error,omitempty"`
}

func (r UploadMDMAppleAPNSCertResponse) Error() error {
	return r.Err
}

func (r UploadMDMAppleAPNSCertResponse) Status() int { return http.StatusAccepted }

type DeleteMDMAppleAPNSCertRequest struct{}

type DeleteMDMAppleAPNSCertResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteMDMAppleAPNSCertResponse) Error() error {
	return r.Err
}

type BatchResendMDMProfileToHostsRequest struct {
	ProfileUUID string `json:"profile_uuid"`
	Filters     struct {
		ProfileStatus string `json:"profile_status"`
	} `json:"filters"`
}

type BatchResendMDMProfileToHostsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r BatchResendMDMProfileToHostsResponse) Error() error { return r.Err }

func (r BatchResendMDMProfileToHostsResponse) Status() int { return http.StatusAccepted }

type GetMDMConfigProfileStatusRequest struct {
	ProfileUUID string `url:"profile_uuid"`
}

type GetMDMConfigProfileStatusResponse struct {
	MDMConfigProfileStatus
	Err error `json:"error,omitempty"`
}

func (r GetMDMConfigProfileStatusResponse) Error() error { return r.Err }

type MdmUnenrollRequest struct {
	HostID uint `url:"id"`
}

type MdmUnenrollResponse struct {
	Err error `json:"error,omitempty"`
}

func (r MdmUnenrollResponse) Error() error { return r.Err }

func (r MdmUnenrollResponse) Status() int { return http.StatusNoContent }
