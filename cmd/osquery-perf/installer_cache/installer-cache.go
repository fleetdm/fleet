package installer_cache

import (
	"log"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/fleetdm/fleet/v4/cmd/osquery-perf/osquery_perf"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
)

// Metadata holds the metadata for software installers.
// To extract the metadata, we must download the file. Once the file has been downloaded once and analyzed,
// the other agents can use the cache to get the appropriate metadata.
type Metadata struct {
	mu    sync.Mutex
	cache map[uint]*file.InstallerMetadata
	Stats *osquery_perf.Stats
}

func (c *Metadata) Get(installer *fleet.SoftwareInstallDetails, orbitClient *service.OrbitClient) (meta *file.InstallerMetadata,
	cacheMiss bool, err error,
) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cache == nil {
		c.cache = make(map[uint]*file.InstallerMetadata, 1)
	}

	meta, ok := c.cache[installer.InstallerID]
	if !ok {
		var err error
		meta, err = c.populateMetadata(installer, orbitClient)
		if err != nil {
			return nil, false, err
		}
		c.cache[installer.InstallerID] = meta
		cacheMiss = true
	}
	return meta, cacheMiss, nil
}

func (c *Metadata) populateMetadata(installer *fleet.SoftwareInstallDetails, orbitClient *service.OrbitClient) (*file.InstallerMetadata,
	error,
) {
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		c.Stats.IncrementOrbitErrors()
		log.Println("level=error, create temp dir:", err)
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	var installerPath string
	if installer.SoftwareInstallerURL != nil {
		installerPath, err = orbitClient.DownloadSoftwareInstallerFromURL(installer.SoftwareInstallerURL.URL,
			installer.SoftwareInstallerURL.Filename, tmpDir, func(n int) {
			})
		if err != nil {
			log.Printf("level=error, msg=download software installer from URL; is CloudFront CDN configured correctly?, err=%s", err)
			c.Stats.IncrementOrbitErrors()
			return nil, err
		}
	}

	if installerPath == "" {
		installerPath, err = orbitClient.DownloadSoftwareInstaller(installer.InstallerID, tmpDir, func(n int) {
		})
		if err != nil {
			log.Printf("level=error, msg=download software installer, err=%s", err)
			c.Stats.IncrementOrbitErrors()
			return nil, err
		}
	}
	
	// Installer might from a Fleet-maintained app. If so, it might be a .dmg or .zip.
	// file.ExtractInstallerMetadata doesn't support .dmg or .zip files, so we have to create
	// an InstallerMetadata manually.
	var extension string
	switch {
	case strings.HasSuffix(installerPath, ".dmg"):
		extension = ".dmg"
	case strings.HasSuffix(installerPath, ".zip"):
		extension = ".zip"
	}

	if extension != "" {
		return &file.InstallerMetadata{
			Name:      path.Base(installerPath),
			Extension: extension,
		}, nil
	}

	// Figure out what we're actually installing here and add it to software inventory
	tfr, err := fleet.NewKeepFileReader(installerPath)
	if err != nil {
		c.Stats.IncrementOrbitErrors()
		log.Println("level=error, open installer:", err)
		return nil, err
	}
	defer tfr.Close()
	item, err := file.ExtractInstallerMetadata(tfr)
	if err != nil {
		c.Stats.IncrementOrbitErrors()
		log.Println("level=error, extract installer metadata:", err)
		return nil, err
	}
	return item, nil
}
