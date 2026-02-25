package fleet

import (
	"net/http"
	"time"
)

type ListSoftwareTitlesRequest struct {
	SoftwareTitleListOptions
}

type ListSoftwareTitlesResponse struct {
	Meta            *PaginationMetadata       `json:"meta"`
	Count           int                       `json:"count"`
	CountsUpdatedAt *time.Time                `json:"counts_updated_at"`
	SoftwareTitles  []SoftwareTitleListResult `json:"software_titles"`
	Err             error                     `json:"error,omitempty"`
}

func (r ListSoftwareTitlesResponse) Error() error { return r.Err }

type GetSoftwareTitleRequest struct {
	ID     uint  `url:"id"`
	TeamID *uint `query:"team_id,optional" renameto:"fleet_id"`
}

type GetSoftwareTitleResponse struct {
	SoftwareTitle *SoftwareTitle `json:"software_title,omitempty"`
	Err           error          `json:"error,omitempty"`
}

func (r GetSoftwareTitleResponse) Error() error { return r.Err }

type UpdateSoftwareNameRequest struct {
	ID   uint   `url:"id"`
	Name string `json:"name"`
}

type UpdateSoftwareNameResponse struct {
	Err error `json:"error,omitempty"`
}

func (r UpdateSoftwareNameResponse) Error() error { return r.Err }

func (r UpdateSoftwareNameResponse) Status() int { return http.StatusResetContent }
