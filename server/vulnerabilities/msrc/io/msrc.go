package io

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/download"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

const (
	minFeedYear = 2020
	MSRCBaseURL = `https://api.msrc.microsoft.com`
)

// MSRCAPI allows users to interact with MSRC resources
type MSRCAPI interface {
	GetFeed(time.Month, uint) (string, error)
}

type MSRCClient struct {
	client  *http.Client
	workDir string
	baseURL *string
}

// NewMSRCClient returns a new MSRCClient that will store all downloaded files in 'workDir' and will
// use 'baseURL' for doing http requests. If no 'baseURL' is provided then 'MSRCBaseURL' will be used.
func NewMSRCClient(
	client *http.Client,
	workDir string,
	baseURL *string,
) MSRCClient {
	c := MSRCClient{client: client, workDir: workDir, baseURL: baseURL}
	if c.baseURL == nil {
		c.baseURL = ptr.String(MSRCBaseURL)
	}
	return c
}

func feedName(date time.Time) string {
	return date.Format("2006-Jan")
}

func (msrc MSRCClient) getURL(date time.Time) (*url.URL, error) {
	if msrc.baseURL == nil {
		return nil, errors.New("invalid base URL")
	}
	return url.Parse(*msrc.baseURL + "/cvrf/v2.0/document/" + feedName(date))
}

// GetFeed downloads the MSRC security feed for 'month' and 'year' into 'workDir', returning the
// path of the downloaded file.
func (msrc MSRCClient) GetFeed(month time.Month, year int) (string, error) {
	d := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	minD := time.Date(minFeedYear, time.January, 1, 0, 0, 0, 0, time.UTC)

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
		return "", err
	}

	return dst, nil
}
