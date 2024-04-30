package nvdsync

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pandatix/nvdapi/v2"
)

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

type VulnCheckResponse struct {
	Meta VulnCheckResponseMeta `json:"_meta"`
	Data []VulnCheckCVE        `json:"data"`
}

type VulnCheckResponseMeta struct {
	NextCursor string `json:"next_cursor"`
}

func getVulnCheckIndexCVEs(c *http.Client, url, cursor *string, lastModStartDate time.Time) (VulnCheckResponse, error) {
	var vcr VulnCheckResponse

	req, err := http.NewRequest(http.MethodGet, *url, nil)
	if err != nil {
		return vcr, err
	}

	q := req.URL.Query()
	if cursor != nil {
		q.Add("cursor", *cursor)
	}
	q.Add("lastModStartDate", lastModStartDate.Format("2006-01-02"))

	resp, err := c.Do(req)
	if err != nil {
		return vcr, fmt.Errorf("do request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return vcr, err
	}

	err = json.NewDecoder(resp.Body).Decode(&vcr)
	if err != nil {
		return vcr, fmt.Errorf("decode response: %w", err)
	}

	return vcr, nil
}
