package nvdsync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/go-kit/log/level"
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

func (s *CVE) getVulnCheckIndexCVEs(ctx context.Context, url, cursor *string, lastModStartDate time.Time) (VulnCheckResponse, error) {
	apiKey := os.Getenv("VULNCHECK_API_KEY")
	if apiKey == "" {
		return VulnCheckResponse{}, ctxerr.New(ctx, "VULNCHECK_API_KEY not set")
	}
	var vcr VulnCheckResponse

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, *url, nil)
	if err != nil {
		return vcr, fmt.Errorf("new request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+apiKey)

	q := req.URL.Query()
	if cursor != nil {
		q.Add("cursor", *cursor)
	}
	q.Add("limit", "50")
	q.Add("lastModStartDate", lastModStartDate.Format("2006-01-02"))
	req.URL.RawQuery = q.Encode()

	for attempt := 0; attempt < s.MaxTryAttempts; attempt++ {
		start := time.Now()
		resp, err := s.client.Do(req)
		if err != nil {
			if attempt < s.MaxTryAttempts-1 {
				level.Debug(s.logger).Log("msg", "Failed to do request", "attempt", attempt, "error", err)
				time.Sleep(s.WaitTimeForRetry)
				continue
			}
			return vcr, fmt.Errorf("do request: %w", err)
		}
		level.Debug(s.logger).Log("msg", "Vulncheck index request", "duration", time.Since(start))

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if attempt < s.MaxTryAttempts-1 {
				level.Debug(s.logger).Log("msg", "Non-OK HTTP status received, waiting %f s", waitTimeForRetry.Seconds(), "attempt", attempt, "status", resp.Status)
				time.Sleep(s.WaitTimeForRetry)
				continue
			}
			return vcr, fmt.Errorf("reached max retry attempts. response status: %s", resp.Status)
		}

		err = json.NewDecoder(resp.Body).Decode(&vcr)
		if err != nil {
			return vcr, fmt.Errorf("decode response: %w", err)
		}

		return vcr, nil
	}

	return vcr, ctxerr.New(ctx, "reach max retry attempts")
}
