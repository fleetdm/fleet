package io

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/download"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/parsed"
	"github.com/google/go-github/v37/github"
)

type ReleaseLister interface {
	ListReleases(
		context.Context,
		string,
		string,
		*github.ListOptions,
	) ([]*github.RepositoryRelease, *github.Response, error)
}

type GithubAPI interface {
	Get(SecurityBulletinName, string) error
	Bulletins() (map[SecurityBulletinName]string, error)
}

type GithubClient struct {
	httpClient *http.Client
	releases   ReleaseLister
	workDir    string
}

func NewGithubClient(client *http.Client, releases ReleaseLister, dir string) GithubClient {
	return GithubClient{
		httpClient: client,
		releases:   releases,
		workDir:    dir,
	}
}

// Get and returns the security bulletin referenced by 'b'
func (gh GithubClient) Get(b SecurityBulletinName, URL string) (*parsed.SecurityBulletin, error) {
	u, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}

	path := filepath.Join(gh.workDir, string(b))
	if err := download.DownloadAndExtract(gh.httpClient, u, path); err != nil {
		return nil, err
	}

	payload, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	bulletin := parsed.SecurityBulletin{}
	err = json.Unmarshal(payload, &bulletin)
	if err != nil {
		return nil, err
	}

	return &bulletin, nil
}

// Bulletins returns a map of 'name' => 'download URL' of the parsed security bulletins stored as assets on Github.
func (gh GithubClient) Bulletins() (map[SecurityBulletinName]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	fmt.Println(gh.releases)

	releases, r, err := gh.releases.ListReleases(
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
