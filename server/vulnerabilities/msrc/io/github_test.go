package io

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/go-github/v37/github"
	"github.com/stretchr/testify/require"
)

type mockGHReleaseLister struct{}

func (m mockGHReleaseLister) ListReleases(
	ctx context.Context,
	owner string,
	repo string,
	opts *github.ListOptions,
) ([]*github.RepositoryRelease, *github.Response, error) {
	var releases []*github.RepositoryRelease
	releases = append(releases, &github.RepositoryRelease{
		Assets: []*github.ReleaseAsset{
			{
				ID:                 ptr.Int64(76142088),
				URL:                ptr.String("https://api.github.com/repos/fleetdm/nvd/releases/assets/76142088"),
				Name:               ptr.String("cpe-80f8ec9cfb9d810.sqlite.gz"),
				Label:              ptr.String(""),
				State:              ptr.String("uploaded"),
				ContentType:        ptr.String("application/gzip"),
				Size:               ptr.Int(52107588),
				DownloadCount:      ptr.Int(683),
				BrowserDownloadURL: ptr.String("https://github.com/fleetdm/nvd/releases/download/202208290017/cpe-80f8ec9cfb9d810.sqlite.gz"),
				NodeID:             ptr.String("RA_kwDOF19pRs4EidYI"),
			},
			{
				ID:                 ptr.Int64(76142089),
				URL:                ptr.String("https://api.github.com/repos/fleetdm/nvd/releases/assets/76142089"),
				Name:               ptr.String(fmt.Sprintf("%sWindows_10-2022_09_10.json.gz", MSRCFilePrefix)),
				Label:              ptr.String(""),
				State:              ptr.String("uploaded"),
				ContentType:        ptr.String("application/gzip"),
				Size:               ptr.Int(52107588),
				DownloadCount:      ptr.Int(683),
				BrowserDownloadURL: ptr.String(fmt.Sprintf("https://github.com/fleetdm/nvd/releases/download/202208290017/%sWindows_10-2022_09_10.json.gz", MSRCFilePrefix)),
				NodeID:             ptr.String("RA_kwDOF19pRs4EidYA"),
			},
			{
				ID:                 ptr.Int64(76142090),
				URL:                ptr.String("https://api.github.com/repos/fleetdm/nvd/releases/assets/76142089"),
				Name:               ptr.String(fmt.Sprintf("%sWindows_11-2022_09_10.json.gz", MSRCFilePrefix)),
				Label:              ptr.String(""),
				State:              ptr.String("uploaded"),
				ContentType:        ptr.String("application/gzip"),
				Size:               ptr.Int(52107588),
				DownloadCount:      ptr.Int(683),
				BrowserDownloadURL: ptr.String(fmt.Sprintf("https://github.com/fleetdm/nvd/releases/download/202208290017/%sWindows_11-2022_09_10.json.gz", MSRCFilePrefix)),
				NodeID:             ptr.String("RA_kwDOF19pRs4EidYA"),
			},
		},
	})

	res := &github.Response{
		Response: &http.Response{
			StatusCode: http.StatusOK,
		},
	}
	return releases, res, nil
}

func TestGithubClient(t *testing.T) {
	t.Run("#Bulletins", func(t *testing.T) {
		sut := NewGithubClient(nil, mockGHReleaseLister{}, t.TempDir())

		bulletins, err := sut.Bulletins()
		require.NoError(t, err)
		require.Len(t, bulletins, 2)

		expectedBulletins := []SecurityBulletinName{
			NewSecurityBulletinName(fmt.Sprintf("%sWindows_10-2022_09_10.json.gz", MSRCFilePrefix)),
			NewSecurityBulletinName(fmt.Sprintf("%sWindows_11-2022_09_10.json.gz", MSRCFilePrefix)),
		}

		for _, e := range expectedBulletins {
			require.NotEmpty(t, bulletins[e])
		}
	})
}
