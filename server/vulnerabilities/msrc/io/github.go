package io

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/download"
	"github.com/google/go-github/v37/github"
)

// ReleaseLister interface around github.NewClient(...).Repositories.
type ReleaseLister interface {
	ListReleases(
		context.Context,
		string,
		string,
		*github.ListOptions,
	) ([]*github.RepositoryRelease, *github.Response, error)
}

// GitHubAPI allows users to interact with the MSRC artifacts published on Github.
type GitHubAPI interface {
	Download(string) (string, error)
	Bulletins() (map[SecurityBulletinName]string, error)
}

type GitHubClient struct {
	httpClient *http.Client
	releases   ReleaseLister
	workDir    string
}

// NewGitHubClient returns a new GithubClient, 'workDir' will be used as the destination directory for
// downloading artifacts.
func NewGitHubClient(client *http.Client, releases ReleaseLister, workDir string) GitHubClient {
	return GitHubClient{
		httpClient: client,
		releases:   releases,
		workDir:    workDir,
	}
}

// Download downloads the security bulletin located at 'URL' in 'workDir', returns the path of
// the downloaded bulletin.
func (gh GitHubClient) Download(URL string) (string, error) {
	u, err := url.Parse(URL)
	if err != nil {
		return "", err
	}

	fPath := filepath.Join(gh.workDir, path.Base(u.Path))
	if err := download.Download(gh.httpClient, u, fPath); err != nil {
		return "", err
	}

	return fPath, nil
}

// Bulletins returns a map of 'bulletin name' => 'download URL' of the bulletins stored as assets on Github.
func (gh GitHubClient) Bulletins() (map[SecurityBulletinName]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

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

	for _, e := range releases[0].Assets {
		name := e.GetName()
		if strings.HasPrefix(name, mSRCFilePrefix) {
			results[NewSecurityBulletinName(name)] = e.GetBrowserDownloadURL()
		}
	}
	return results, nil
}
