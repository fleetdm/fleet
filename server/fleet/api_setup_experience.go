package fleet

import (
	"fmt"
	"mime/multipart"
	"slices"
	"strings"
)

func validateSetupExperiencePlatform(platforms string) error {
	for platform := range strings.SplitSeq(platforms, ",") {
		if platform != "" && !slices.Contains(SetupExperienceSupportedPlatforms, platform) {
			quotedPlatforms := strings.Join(SetupExperienceSupportedPlatforms, "\", \"")
			quotedPlatforms = fmt.Sprintf("\"%s\"", quotedPlatforms)
			return &BadRequestError{Message: fmt.Sprintf("platform %q unsupported, platform must be one of %s", platform, quotedPlatforms)}
		}
	}
	return nil
}

type PutSetupExperienceSoftwareRequest struct {
	Platform string `json:"platform"`
	TeamID   uint   `json:"team_id" renameto:"fleet_id"`
	TitleIDs []uint `json:"software_title_ids"`
}

func (r *PutSetupExperienceSoftwareRequest) ValidateRequest() error {
	return validateSetupExperiencePlatform(r.Platform)
}

type PutSetupExperienceSoftwareResponse struct {
	Err error `json:"error,omitempty"`
}

func (r PutSetupExperienceSoftwareResponse) Error() error { return r.Err }

type GetSetupExperienceSoftwareRequest struct {
	// Platforms can be a comma separated list
	Platforms string `query:"platform,optional"`
	ListOptions
	TeamID uint `query:"team_id" renameto:"fleet_id"`
}

func (r *GetSetupExperienceSoftwareRequest) ValidateRequest() error {
	return validateSetupExperiencePlatform(r.Platforms)
}

type GetSetupExperienceSoftwareResponse struct {
	SoftwareTitles []SoftwareTitleListResult `json:"software_titles"`
	Count          int                       `json:"count"`
	Meta           *PaginationMetadata       `json:"meta"`
	Err            error                     `json:"error,omitempty"`
}

func (r GetSetupExperienceSoftwareResponse) Error() error { return r.Err }

type GetSetupExperienceScriptRequest struct {
	TeamID *uint  `query:"team_id,optional" renameto:"fleet_id"`
	Alt    string `query:"alt,optional"`
}

type GetSetupExperienceScriptResponse struct {
	*Script
	Err error `json:"error,omitempty"`
}

func (r GetSetupExperienceScriptResponse) Error() error { return r.Err }

type SetSetupExperienceScriptRequest struct {
	TeamID *uint
	Script *multipart.FileHeader
}

type SetSetupExperienceScriptResponse struct {
	Err error `json:"error,omitempty"`
}

func (r SetSetupExperienceScriptResponse) Error() error { return r.Err }

type DeleteSetupExperienceScriptRequest struct {
	TeamID *uint `query:"team_id,optional" renameto:"fleet_id"`
}

type DeleteSetupExperienceScriptResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteSetupExperienceScriptResponse) Error() error { return r.Err }
