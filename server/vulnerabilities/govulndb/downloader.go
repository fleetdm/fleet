package govulndb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/google/go-github/v37/github"
)

// The mirrored database is published as a release asset in this repository, the same one that
// carries the OSV artifacts.
const (
	githubOwner = "fleetdm"
	githubRepo  = "vulnerabilities"
)

// assetInfo is a release asset Fleet can download.
type assetInfo struct {
	Name   string
	ID     int64
	Digest string
}

type rawAsset struct {
	Name   string `json:"name"`
	ID     int64  `json:"id"`
	Digest string `json:"digest"`
}

type rawRelease struct {
	TagName string     `json:"tag_name"`
	Assets  []rawAsset `json:"assets"`
}

// latestArtifactAsset returns the newest Go vulnerability database asset in the latest release,
// or nil if the release carries none. Assets are named govulndb-YYYY-MM-DD.json.gz, so the
// newest is the last in lexical order.
func latestArtifactAsset(ctx context.Context) (*assetInfo, error) {
	httpClient := fleethttp.NewClient()

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating request")
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "fetching latest release")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ctxerr.Errorf(ctx, "github http status error: %d", resp.StatusCode)
	}

	var release rawRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decoding release")
	}
	if release.TagName == "" {
		return nil, ctxerr.New(ctx, "release tag name is empty")
	}

	var latest *assetInfo
	for _, asset := range release.Assets {
		if !isArtifactAsset(asset.Name) {
			continue
		}
		if latest == nil || asset.Name > latest.Name {
			latest = &assetInfo{Name: asset.Name, ID: asset.ID, Digest: asset.Digest}
		}
	}

	return latest, nil
}

// isArtifactAsset reports whether a release asset name is a Go vulnerability database artifact.
// A release carries assets for every data source Fleet mirrors, so anything not matching here
// is silently ignored.
func isArtifactAsset(name string) bool {
	return strings.HasPrefix(name, filePrefix) && strings.HasSuffix(name, fileExt)
}

// downloadAsset writes a release asset to dstPath.
func downloadAsset(ctx context.Context, assetID int64, dstPath string) error {
	ghClient := fleethttp.NewGithubClient()
	client := github.NewClient(ghClient)

	httpClient := fleethttp.NewClient()
	rc, redirectURL, err := client.Repositories.DownloadReleaseAsset(ctx, githubOwner, githubRepo, assetID, httpClient)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "downloading release asset")
	}

	if redirectURL != "" {
		if rc != nil {
			rc.Close()
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, redirectURL, nil)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "creating redirect request")
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "downloading from redirect URL")
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return ctxerr.Errorf(ctx, "download http status error: %d", resp.StatusCode)
		}
		rc = resp.Body
	}

	if rc == nil {
		return ctxerr.New(ctx, "release asset download returned no content")
	}
	defer rc.Close()

	// #nosec G304 -- dstPath is built from a Fleet-controlled vuln directory and an
	// asset name validated by isArtifactAsset
	out, err := os.Create(dstPath)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating destination file")
	}
	defer out.Close()

	if _, err := io.Copy(out, rc); err != nil {
		return ctxerr.Wrap(ctx, err, "writing to destination file")
	}

	return nil
}

// fileSHA256 returns the digest of a file in the format GitHub reports asset digests in.
func fileSHA256(path string) (string, error) {
	// #nosec G304 -- path is built from a Fleet-controlled vuln directory
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}
