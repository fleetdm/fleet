package msrcapps

import (
	"context"
	"fmt"
	"sort"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
	"github.com/google/go-github/v37/github"
)

// SyncFromGithub keeps local MSRC app bulletin in sync with the one published on GitHub.
func SyncFromGithub(ctx context.Context, dstDir string) error {
	client := fleethttp.NewGithubClient()
	rep := github.NewClient(client).Repositories

	gh := io.NewGitHubClient(client, rep, dstDir)
	fs := io.NewFSClient(dstDir)

	if err := sync(ctx, fs, gh); err != nil {
		return fmt.Errorf("msrc app sync: %w", err)
	}

	return nil
}

func sync(
	ctx context.Context,
	fsClient io.FSAPI,
	ghClient io.GitHubAPI,
) error {
	remote, url, err := ghClient.MSRCAppBulletin(ctx)
	if err != nil {
		return err
	}

	// Nothing published yet on remote repo
	if url == "" {
		return nil
	}

	local, err := fsClient.MSRCAppBulletin()
	if err != nil {
		return err
	}

	if len(local) == 0 {
		if _, err := ghClient.Download(url); err != nil {
			return err
		}
		return nil
	}

	// Sort to find the latest local file
	sort.Slice(local, func(i, j int) bool {
		return local[j].Before(local[i])
	})

	// If remote is newer, download it
	if local[0].Before(remote) {
		if _, err := ghClient.Download(url); err != nil {
			return err
		}
	}

	// Clean up old local files
	for _, l := range local {
		if l.Before(remote) {
			if err := fsClient.Delete(l); err != nil {
				return err
			}
		}
	}

	return nil
}
