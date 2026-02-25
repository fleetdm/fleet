package fleet

import (
	"net/http"
	"time"
)

type ListVulnerabilitiesRequest struct {
	VulnListOptions
}

type ListVulnerabilitiesResponse struct {
	Vulnerabilities []VulnerabilityWithMetadata `json:"vulnerabilities"`
	Count           uint                        `json:"count"`
	CountsUpdatedAt time.Time                   `json:"counts_updated_at"`
	Meta            *PaginationMetadata         `json:"meta,omitempty"`
	Err             error                       `json:"error,omitempty"`
}

func (r ListVulnerabilitiesResponse) Error() error { return r.Err }

type GetVulnerabilityRequest struct {
	CVE    string `url:"cve"`
	TeamID *uint  `query:"team_id,optional" renameto:"fleet_id"`
}

type GetVulnerabilityResponse struct {
	Vulnerability *VulnerabilityWithMetadata `json:"vulnerability"`
	OSVersions    []*VulnerableOS            `json:"os_versions"`
	Software      []*VulnerableSoftware      `json:"software"`
	Err           error                      `json:"error,omitempty"`
	StatusCode    int
}

func (r GetVulnerabilityResponse) Error() error { return r.Err }

func (r GetVulnerabilityResponse) Status() int {
	if r.StatusCode == 0 {
		return http.StatusOK
	}
	return r.StatusCode
}
