package io

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/go-github/v37/github"
	"github.com/stretchr/testify/require"
)

type mockGHReleaseListerWithInvalidNames struct{ N int }

func (m mockGHReleaseListerWithInvalidNames) ListReleases(
	ctx context.Context,
	owner string,
	repo string,
	opts *github.ListOptions,
) ([]*github.RepositoryRelease, *github.Response, error) {
	var releases []*github.RepositoryRelease
	var assets []*github.ReleaseAsset

	for i := 0; i < m.N; i++ {
		asset := github.ReleaseAsset{
			ID:                 ptr.Int64(76142088),
			URL:                ptr.String("https://api.github.com/repos/fleetdm/nvd/releases/assets/76142088"),
			Name:               ptr.String(fmt.Sprintf("%smacoffice-200002%d_09_01.json", macOfficeReleaseNotesPrefix, i)),
			Label:              ptr.String(""),
			State:              ptr.String("uploaded"),
			ContentType:        ptr.String("application/gzip"),
			Size:               ptr.Int(52107588),
			DownloadCount:      ptr.Int(683),
			BrowserDownloadURL: ptr.String(fmt.Sprintf("https://github.com/fleetdm/nvd/releases/download/202208290017/%d.json", i)),
			NodeID:             ptr.String("RA_kwDOF19pRs4EidYI"),
		}
		assets = append(assets, &asset)
	}
	release := github.RepositoryRelease{Assets: assets}
	releases = append(releases, &release)

	res := &github.Response{
		Response: &http.Response{
			StatusCode: http.StatusOK,
		},
	}
	return releases, res, nil
}

type mockGHReleaseListerWithError struct {
	N          int
	StatusCode int
}

func (m mockGHReleaseListerWithError) ListReleases(
	ctx context.Context,
	owner string,
	repo string,
	opts *github.ListOptions,
) ([]*github.RepositoryRelease, *github.Response, error) {
	return nil, nil, errors.New("some error")
}

type mockGHReleaseListerForMacOfficeReleaseNotes struct {
	N          int
	StatusCode int
}

func (m mockGHReleaseListerForMacOfficeReleaseNotes) ListReleases(
	ctx context.Context,
	owner string,
	repo string,
	opts *github.ListOptions,
) ([]*github.RepositoryRelease, *github.Response, error) {
	var releases []*github.RepositoryRelease
	var assets []*github.ReleaseAsset

	for i := 0; i < m.N; i++ {
		asset := github.ReleaseAsset{
			ID:                 ptr.Int64(76142088),
			URL:                ptr.String("https://api.github.com/repos/fleetdm/nvd/releases/assets/76142088"),
			Name:               ptr.String(fmt.Sprintf("%smacoffice-202%d_09_01.json", macOfficeReleaseNotesPrefix, i)),
			Label:              ptr.String(""),
			State:              ptr.String("uploaded"),
			ContentType:        ptr.String("application/gzip"),
			Size:               ptr.Int(52107588),
			DownloadCount:      ptr.Int(683),
			BrowserDownloadURL: ptr.String(fmt.Sprintf("https://github.com/fleetdm/nvd/releases/download/202208290017/%d.json", i)),
			NodeID:             ptr.String("RA_kwDOF19pRs4EidYI"),
		}
		assets = append(assets, &asset)
	}
	release := github.RepositoryRelease{Assets: assets}
	releases = append(releases, &release)

	statusCode := http.StatusOK
	if m.StatusCode != 0 {
		statusCode = m.StatusCode
	}

	res := &github.Response{
		Response: &http.Response{
			StatusCode: statusCode,
		},
	}
	return releases, res, nil
}

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
				BrowserDownloadURL: ptr.String("https://github.com/fleetdm/nvd/releases/download/202208290017/cpe-80f8ec9cfb9d810.sqlite"),
				NodeID:             ptr.String("RA_kwDOF19pRs4EidYI"),
			},
			{
				ID:                 ptr.Int64(76142089),
				URL:                ptr.String("https://api.github.com/repos/fleetdm/nvd/releases/assets/76142089"),
				Name:               ptr.String(fmt.Sprintf("%sWindows_10-2022_09_10.json", mSRCFilePrefix)),
				Label:              ptr.String(""),
				State:              ptr.String("uploaded"),
				ContentType:        ptr.String("application/json"),
				Size:               ptr.Int(52107588),
				DownloadCount:      ptr.Int(683),
				BrowserDownloadURL: ptr.String(fmt.Sprintf("https://github.com/fleetdm/nvd/releases/download/202208290017/%sWindows_10-2022_09_10.json", mSRCFilePrefix)),
				NodeID:             ptr.String("RA_kwDOF19pRs4EidYA"),
			},
			{
				ID:                 ptr.Int64(76142090),
				URL:                ptr.String("https://api.github.com/repos/fleetdm/nvd/releases/assets/76142089"),
				Name:               ptr.String(fmt.Sprintf("%sWindows_11-2022_09_10.json", mSRCFilePrefix)),
				Label:              ptr.String(""),
				State:              ptr.String("uploaded"),
				ContentType:        ptr.String("application/json"),
				Size:               ptr.Int(52107588),
				DownloadCount:      ptr.Int(683),
				BrowserDownloadURL: ptr.String(fmt.Sprintf("https://github.com/fleetdm/nvd/releases/download/202208290017/%sWindows_11-2022_09_10.json", mSRCFilePrefix)),
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

func TestIntegrationsGithubClient(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("MacOfficeReleaseNotes", func(t *testing.T) {
		t.Run("with invalid remote file names", func(t *testing.T) {
			sut := NewGitHubClient(nil, mockGHReleaseListerWithInvalidNames{N: 1}, t.TempDir())

			_, url, err := sut.MacOfficeReleaseNotes(ctx)
			require.Error(t, err)
			require.Empty(t, url)
		})
		t.Run("with HTTP error code", func(t *testing.T) {
			sut := NewGitHubClient(nil, mockGHReleaseListerForMacOfficeReleaseNotes{N: 1, StatusCode: http.StatusInternalServerError}, t.TempDir())

			_, url, err := sut.MacOfficeReleaseNotes(ctx)
			require.Error(t, err)
			require.Empty(t, url)
		})

		t.Run("with no GH assets", func(t *testing.T) {
			sut := NewGitHubClient(nil, mockGHReleaseListerForMacOfficeReleaseNotes{N: 0}, t.TempDir())

			name, url, err := sut.MacOfficeReleaseNotes(ctx)
			require.NoError(t, err)
			require.Empty(t, url)
			require.Empty(t, name.String())
		})
		t.Run("with a single release note asset", func(t *testing.T) {
			sut := NewGitHubClient(nil, mockGHReleaseListerForMacOfficeReleaseNotes{N: 1}, t.TempDir())

			name, url, err := sut.MacOfficeReleaseNotes(ctx)
			require.NoError(t, err)
			require.Equal(t, "https://github.com/fleetdm/nvd/releases/download/202208290017/0.json", url)
			require.Equal(t, fmt.Sprintf("%smacoffice-2020_09_01.json", macOfficeReleaseNotesPrefix), name.String())
		})
		t.Run("with more than one release note asset", func(t *testing.T) {
			sut := NewGitHubClient(nil, mockGHReleaseListerForMacOfficeReleaseNotes{N: 2}, t.TempDir())

			relNotes, url, err := sut.MacOfficeReleaseNotes(ctx)
			require.Error(t, err, "found more than one MacOffice release notes")
			require.Empty(t, url)
			require.Empty(t, relNotes)
		})

		t.Run("on error", func(t *testing.T) {
			sut := NewGitHubClient(nil, mockGHReleaseListerWithError{}, t.TempDir())

			relNotes, url, err := sut.MacOfficeReleaseNotes(ctx)
			require.Error(t, err, "some error")
			require.Empty(t, url)
			require.Empty(t, relNotes)
		})
	})

	t.Run("#Download", func(t *testing.T) {
		fileName := fmt.Sprintf("%sWindows_11-2022_09_10.json", mSRCFilePrefix)
		urlPath := fmt.Sprintf("/fleetdm/nvd/releases/download/202208290017/%s", fileName)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == urlPath {
				w.Header().Add("content-type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("some payload"))
				require.NoError(t, err)
			}
		}))
		t.Cleanup(server.Close)

		dstDir := t.TempDir()
		expectedPath := filepath.Join(dstDir, fileName)
		url := server.URL + urlPath

		sut := NewGitHubClient(server.Client(), mockGHReleaseLister{}, dstDir)
		actualPath, err := sut.Download(url)
		require.NoError(t, err)
		require.Equal(t, expectedPath, actualPath)
		require.FileExists(t, expectedPath)

		t.Run("with invalid URL", func(t *testing.T) {
			badURL := "some bad url"
			actualPath, err := sut.Download(badURL)
			require.Error(t, err)
			require.Empty(t, actualPath)
		})
	})

	t.Run("#MSRCBulletins", func(t *testing.T) {
		sut := NewGitHubClient(nil, mockGHReleaseLister{}, t.TempDir())

		bulletins, err := sut.MSRCBulletins(ctx)
		require.NoError(t, err)
		require.Len(t, bulletins, 2)

		a, err := NewMSRCMetadata(fmt.Sprintf("%sWindows_10-2022_09_10.json", mSRCFilePrefix))
		require.NoError(t, err)
		b, err := NewMSRCMetadata(fmt.Sprintf("%sWindows_11-2022_09_10.json", mSRCFilePrefix))
		require.NoError(t, err)

		expectedBulletins := []MetadataFileName{a, b}

		for _, e := range expectedBulletins {
			require.NotEmpty(t, bulletins[e])
		}
	})
}
