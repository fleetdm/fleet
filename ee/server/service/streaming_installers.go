package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// streamingInstallerStore returns the store to stream installer downloads
// to, or nil if streaming is disabled (cloud_storage.stream_installer_downloads)
// or the installer store doesn't support it.
func (svc *Service) streamingInstallerStore() fleet.StreamingInstallerStore {
	if !svc.config.CloudStorage.StreamInstallerDownloads {
		return nil
	}
	store, _ := svc.softwareInstallStore.(fleet.StreamingInstallerStore)
	return store
}

// putInstaller stores the installer content under storageID. Content already
// streamed to a staging object is promoted with a server-side copy instead of
// being uploaded again.
func (svc *Service) putInstaller(ctx context.Context, storageID string, content fleet.InstallerReader) error {
	return putInstallerToStore(ctx, svc.softwareInstallStore, storageID, content)
}

func putInstallerToStore(ctx context.Context, store fleet.SoftwareInstallerStore, storageID string, content fleet.InstallerReader) error {
	if staged, ok := content.(*fleet.RemoteInstallerReader); ok {
		if _, ok := store.(fleet.StreamingInstallerStore); !ok {
			return ctxerr.New(ctx, "installer store doesn't support staged installers")
		}
		return staged.Promote(ctx, storageID)
	}
	return store.Put(ctx, storageID, content)
}
