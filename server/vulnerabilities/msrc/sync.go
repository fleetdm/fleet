package msrc

import (
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	msrc_io "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/io"
	msrc_parsed "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/parsed"
)

// bulletinsDelta returns what bulletins should be download and what bulletins should be removed
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
	// remoteBulletins, err := downloadBulletinList()
	// if err != nil {
	// 	return err
	// }

	// localBulletins, err := getLocalBulletinList(dstDir)
	// if err != nil {
	// 	return err
	// }

	// Compare remoteBulletins and localBulletins
	// and figure out what to download and what to remove.
	panic("not implemented")
}

func sync(
	remoteBulletinListGetter func() ([]string, error),
	localBulletinListGetter func() ([]string, error),
	remoteBulletinGetter func(file string) (*msrc_parsed.SecurityBulletin, error),
) error {
	panic("not implemented")
}
