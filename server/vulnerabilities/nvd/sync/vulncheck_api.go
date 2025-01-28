package nvdsync

import "github.com/pandatix/nvdapi/v2"

type VulnCheckCVEItem struct {
	Item VulnCheckCVE `json:"cve"`
}

type VulnCheckCVE struct {
	nvdapi.CVE
	VcConfigurations []nvdapi.Config `json:"vcConfigurations,omitempty"`
}

type VulnCheckBackupResponse struct {
	Data []VulnCheckBackupData `json:"data"`
}

type VulnCheckBackupData struct {
	URL string `json:"url"`
}

type VulnCheckBackupDataFile struct {
	Vulnerabilities []VulnCheckCVEItem `json:"vulnerabilities"`
}
