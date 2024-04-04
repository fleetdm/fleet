package nvdsync

import "github.com/pandatix/nvdapi/v2"

type VulnCheckCVE struct {
	nvdapi.CVE
	VcConfigurations []nvdapi.Config `json:"vcConfigurations,omitempty"`
}

type VulnCheckResponse struct {
	Meta            VulnCheckMeta  `json:"_meta"`
	Vulnerabilities []VulnCheckCVE `json:"data"`
}

type VulnCheckMeta struct {
	NextCursor *string `json:"next_cursor"`
}


