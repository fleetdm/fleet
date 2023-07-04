package macoffice

import (
	"context"
	"fmt"
	"sort"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
	"github.com/google/go-github/v37/github"
)

// SyncFromGithub keeps the local mac Office release notes metadata in sync with the one published in Github.
func SyncFromGithub(ctx context.Context, dstDir string) error {
	client := fleethttp.NewGithubClient()
	rep := github.NewClient(client).Repositories

	gh := io.NewGitHubClient(client, rep, dstDir)
	fs := io.NewFSClient(dstDir)

	if err := sync(ctx, fs, gh); err != nil {
		return fmt.Errorf("macoffice release sync: %w", err)
	}

	return nil
}

func sync(
	ctx context.Context,
	fsClient io.FSAPI,
	ghClient io.GitHubAPI,
) error {
	remote, url, err := ghClient.MacOfficeReleaseNotes(ctx)
	if err != nil {
		return err
	}

	// Nothing published yet on remote repo, so we do nothing.
	if url == "" {
		return nil
	}

	local, err := fsClient.MacOfficeReleaseNotes()
	if err != nil {
		return err
	}

	if len(local) == 0 {
		if _, err := ghClient.Download(url); err != nil {
			return err
		}
		return nil
	}

	sort.Slice(local, func(i, j int) bool {
		return local[j].Before(local[i])
	})

	if local[0].Before(remote) {
		if _, err := ghClient.Download(url); err != nil {
			return err
		}
	}

	// Clean up out of date files
	for _, l := range local {
		if l.Before(remote) {
			if err := fsClient.Delete(l); err != nil {
				return err
			}
		}
	}
	return nil
}
