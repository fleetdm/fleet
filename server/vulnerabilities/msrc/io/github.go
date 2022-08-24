package msrc_io

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v37/github"
)

// remoteBulletins returns a map of 'name' => 'download URL' of the parsed security bulletins stored as assets on Github.
func remoteBulletins(client *http.Client) (map[SecurityBulletinName]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	releases, r, err := github.NewClient(client).Repositories.ListReleases(
		ctx,
		"fleetdm",
		"nvd",
		&github.ListOptions{Page: 0, PerPage: 10},
	)
	if err != nil {
		return nil, err
	}

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github http status error: %d", r.StatusCode)
	}

	results := make(map[SecurityBulletinName]string)

	// TODO (juan): Since the nvd repo includes both NVD and MSRC assets, we will need to do some
	// filtering logic here.
	for _, e := range releases[0].Assets {
		name := e.GetName()
		if strings.HasPrefix(name, MSRCFilePrefix) {
			results[NewSecurityBulletinName(name)] = e.GetBrowserDownloadURL()
		}
	}
	return results, nil
}
