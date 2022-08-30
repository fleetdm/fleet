package io

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/download"
	"github.com/google/go-github/v37/github"
)

type MSRCGithubAPI interface {
	Download(SecurityBulletinName, string) error
	Bulletins() (map[SecurityBulletinName]string, error)
}

type MSRCGithubClient struct {
	client *http.Client
	dstDir string
}

func NewMSRCGithubClient(client *http.Client, dir string) MSRCGithubClient {
	return MSRCGithubClient{client: client, dstDir: dir}
}

// Downloads the security bulletin to 'dir'.
func (gh MSRCGithubClient) Download(b SecurityBulletinName, urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return err
	}
	path := filepath.Join(gh.dstDir, string(b))
	return download.DownloadAndExtract(gh.client, u, path)
}

// Bulletins returns a map of 'name' => 'download URL' of the parsed security bulletins stored as assets on Github.
func (gh MSRCGithubClient) Bulletins() (map[SecurityBulletinName]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	releases, r, err := github.NewClient(gh.client).Repositories.ListReleases(
		ctx,
		"fleetdm",
		"nvd",
		&github.ListOptions{Page: 0, PerPage: 10},
	)
	if err != nil {
		return nil, err
	}

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github http status error: %d", r.StatusCode)
	}

	results := make(map[SecurityBulletinName]string)

	// TODO (juan): Since the nvd repo includes both NVD and MSRC assets, we will need to do some
	// filtering logic here. To be done in https://github.com/fleetdm/fleet/issues/7394.
	for _, e := range releases[0].Assets {
		name := e.GetName()
		if strings.HasPrefix(name, MSRCFilePrefix) {
			results[NewSecurityBulletinName(name)] = e.GetBrowserDownloadURL()
		}
	}
	return results, nil
}
