package msrc

import (
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	msrc_io "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/io"
	msrc_parsed "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/parsed"
)

// bulletinsDelta returns what bulletins should be download from GH and what bulletins should be removed
// from the local file system based what OS are installed, what local bulletins we have and what
// remote bulletins exist.
func bulletinsDelta(
	os []fleet.OperatingSystem,
	local []msrc_io.SecurityBulletinName,
	remote []msrc_io.SecurityBulletinName,
) (
	[]msrc_io.SecurityBulletinName,
	[]msrc_io.SecurityBulletinName,
) {
	if len(os) == 0 {
		return remote, nil
	}

	var matching []msrc_io.SecurityBulletinName
	for _, r := range remote {
		for _, o := range os {
			product := msrc_parsed.NewFullProductName(o.Name)
			if r.ProductName() == product.Name() {
				matching = append(matching, r)
			}
		}
	}

	var toDownload []msrc_io.SecurityBulletinName
	var toDelete []msrc_io.SecurityBulletinName
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

// Sync syncs the local msrc security bulletins (contained in dstDir) for one or more operating systems with the security
// bulletin published in Github.
// If 'os' is nil, then all security bulletins will be synched.
func Sync(client *http.Client, dstDir string, os []fleet.OperatingSystem) error {
	gh := msrc_io.NewMSRCGithubClient(client, dstDir)
	fs := msrc_io.NewMSRCFSClient(dstDir)

	if err := sync(os, fs, gh); err != nil {
		return fmt.Errorf("msrc sync: %w", err)
	}

	return nil
}

func sync(
	os []fleet.OperatingSystem,
	fsClient msrc_io.MSRCFSAPI,
	ghClient msrc_io.MSRCGithubAPI,
) error {
	remoteURLs, err := ghClient.Bulletins()
	if err != nil {
		return err
	}

	var remote []msrc_io.SecurityBulletinName
	for r := range remoteURLs {
		remote = append(remote, r)
	}

	local, err := fsClient.Bulletins()
	if err != nil {
		return err
	}

	toDownload, toDelete := bulletinsDelta(os, local, remote)
	for _, b := range toDownload {
		if err := ghClient.Download(b, remoteURLs[b]); err != nil {
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
