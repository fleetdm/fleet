package fleet

import "time"

type ListSoftwareRequest struct {
	SoftwareListOptions
}

type ListSoftwareResponse struct {
	CountsUpdatedAt *time.Time `json:"counts_updated_at"`
	Software        []Software `json:"software,omitempty"`
	Err             error      `json:"error,omitempty"`
}

func (r ListSoftwareResponse) Error() error { return r.Err }

type ListSoftwareVersionsResponse struct {
	Count           int                 `json:"count"`
	CountsUpdatedAt *time.Time          `json:"counts_updated_at"`
	Software        []Software          `json:"software,omitempty"`
	Meta            *PaginationMetadata `json:"meta"`
	Err             error               `json:"error,omitempty"`
}

func (r ListSoftwareVersionsResponse) Error() error { return r.Err }

type GetSoftwareRequest struct {
	ID     uint  `url:"id"`
	TeamID *uint `query:"team_id,optional" renameto:"fleet_id"`
}

type GetSoftwareResponse struct {
	Software *Software `json:"software,omitempty"`
	Err      error     `json:"error,omitempty"`
}

func (r GetSoftwareResponse) Error() error { return r.Err }

type CountSoftwareRequest struct {
	SoftwareListOptions
}

type CountSoftwareResponse struct {
	Count int   `json:"count"`
	Err   error `json:"error,omitempty"`
}

func (r CountSoftwareResponse) Error() error { return r.Err }
