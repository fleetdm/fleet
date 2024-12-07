package installer_cache

import (
	"log"
	"os"
	"sync"

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
}

func (c *Metadata) Get(key uint, orbitClient *service.OrbitClient) (meta *file.InstallerMetadata,
	cacheMiss bool, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cache == nil {
		c.cache = make(map[uint]*file.InstallerMetadata, 1)
	}

	meta, ok := c.cache[key]
	if !ok {
		var err error
		meta, err = populateMetadata(orbitClient, key)
		if err != nil {
			return nil, false, err
		}
		c.cache[key] = meta
		cacheMiss = true
	}
	return meta, cacheMiss, nil
}

func populateMetadata(orbitClient *service.OrbitClient, installerID uint) (*file.InstallerMetadata, error) {
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		log.Println("create temp dir:", err)
		return nil, err
	}
	defer os.RemoveAll(tmpDir)
	path, err := orbitClient.DownloadSoftwareInstaller(installerID, tmpDir)
	if err != nil {
		log.Println("download software installer:", err)
		return nil, err
	}
	// Figure out what we're actually installing here and add it to software inventory
	tfr, err := fleet.NewKeepFileReader(path)
	if err != nil {
		log.Println("open installer:", err)
		return nil, err
	}
	defer tfr.Close()
	item, err := file.ExtractInstallerMetadata(tfr)
	if err != nil {
		log.Println("extract installer metadata:", err)
		return nil, err
	}
	return item, nil
}
