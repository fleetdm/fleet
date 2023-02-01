package docker

import (
	"context"
)

type releasesGetter interface {
	GetReleases(context.Context) ([]DesktopRelease, error)
}

func SyncDockerDesktopReleases(
	ctx context.Context,
	dstDir string,
	getter releasesGetter,
) error {
	return nil
}
