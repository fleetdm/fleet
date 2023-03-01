package msrc

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/parsed"
	"github.com/google/go-github/v37/github"
)

// bulletinsDelta returns what bulletins should be download from GH and what bulletins should be removed
// from the local file system based what OS are installed, what local bulletins we have and what
// remote bulletins exist.
func bulletinsDelta(
	os []fleet.OperatingSystem,
	local []io.MetadataFileName,
	remote []io.MetadataFileName,
) (
	[]io.MetadataFileName,
	[]io.MetadataFileName,
) {
	if len(os) == 0 {
		return remote, nil
	}

	var matching []io.MetadataFileName
	for _, r := range remote {
		for _, o := range os {
			product := parsed.NewProductFromOS(o)
			if r.ProductName() == product.Name() {
				matching = append(matching, r)
			}
		}
	}

	var toDownload []io.MetadataFileName
	var toDelete []io.MetadataFileName
	for _, m := range matching {
		var found bool
		for _, l := range local {
			if m.ProductName() == l.ProductName() {
				found = true
				// out of date
				if l.Before(m) {
					toDownload = append(toDownload, m)
					toDelete = append(toDelete, l)
				}
				break
			}
		}
		if !found {
			toDownload = append(toDownload, m)
		}
	}
	return toDownload, toDelete
}

// SyncFromGithub syncs the local msrc security bulletins (contained in dstDir) for one or more operating
// systems with the security bulletin published in Github.
//
// If 'os' is nil, then all security bulletins will be synched.
func SyncFromGithub(ctx context.Context, dstDir string, os []fleet.OperatingSystem) error {
	client := fleethttp.NewGithubClient()
	rep := github.NewClient(client).Repositories
	gh := io.NewGitHubClient(client, rep, dstDir)
	fs := io.NewFSClient(dstDir)

	if err := sync(ctx, os, fs, gh); err != nil {
		return fmt.Errorf("msrc sync: %w", err)
	}

	return nil
}

func sync(
	ctx context.Context,
	os []fleet.OperatingSystem,
	fsClient io.FSAPI,
	ghClient io.GitHubAPI,
) error {
	remoteURLs, err := ghClient.MSRCBulletins(ctx)
	if err != nil {
		return err
	}

	var remote []io.MetadataFileName
	for r := range remoteURLs {
		remote = append(remote, r)
	}

	local, err := fsClient.MSRCBulletins()
	if err != nil {
		return err
	}

	toDownload, toDelete := bulletinsDelta(os, local, remote)
	for _, b := range toDownload {
		if _, err := ghClient.Download(remoteURLs[b]); err != nil {
			return err
		}
	}
	for _, d := range toDelete {
		if err := fsClient.Delete(d); err != nil {
			return err
		}
	}

	return nil
}
