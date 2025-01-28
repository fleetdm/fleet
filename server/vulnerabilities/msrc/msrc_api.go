package msrc

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/download"
)

const (
	// Pre 2020 there are some weirdness around the way the 'Supersedes' field is defined for Vulnerabilities, sometimes
	// it does not reference a KBID.
	MSRCMinYear = 2020
	MSRCBaseURL = `https://api.msrc.microsoft.com`
)

// MSRCAPI allows users to interact with MSRC resources
type MSRCAPI interface {
	GetFeed(time.Month, int) (string, error)
}

// FeedNotFound is returned when a MSRC feed was not found.
//
// E.g. September 2024 bulleting was released on the 2nd.
var FeedNotFound = errors.New("feed not found")

type MSRCClient struct {
	client  *http.Client
	workDir string
	baseURL string
}

// NewMSRCClient returns a new MSRCClient that will store all downloaded files in 'workDir' and will
// use 'baseURL' for doing http requests.
func NewMSRCClient(
	client *http.Client,
	workDir string,
	baseURL string,
) MSRCClient {
	return MSRCClient{client: client, workDir: workDir, baseURL: baseURL}
}

func feedName(date time.Time) string {
	return date.Format("2006-Jan")
}

func (msrc MSRCClient) getURL(date time.Time) (*url.URL, error) {
	return url.Parse(msrc.baseURL + "/cvrf/v3.0/document/" + feedName(date))
}

// GetFeed downloads the MSRC security feed for 'month' and 'year' into 'workDir', returning the
// path of the downloaded file.
func (msrc MSRCClient) GetFeed(month time.Month, year int) (string, error) {
	d := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	minD := time.Date(MSRCMinYear, time.January, 1, 0, 0, 0, 0, time.UTC)

	if d.Before(minD) {
		return "", fmt.Errorf("min allowed date is %s", minD)
	}

	if d.After(time.Now().UTC()) {
		return "", errors.New("date can't be in the future")
	}

	dst := filepath.Join(msrc.workDir, fmt.Sprintf("%s.xml", feedName(d)))
	u, err := msrc.getURL(d)
	if err != nil {
		return "", err
	}

	if err := download.Download(msrc.client, u, dst); err != nil {
		if errors.Is(err, download.NotFound) {
			return "", FeedNotFound
		}
		return "", err
	}

	return dst, nil
}
