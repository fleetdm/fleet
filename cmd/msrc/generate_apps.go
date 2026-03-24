//go:build ignore

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc"
	msrcapps "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/apps"
	"github.com/google/go-github/v37/github"
)

func panicif(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	wd, err := os.Getwd()
	panicif(err)

	inPath := filepath.Join(wd, "msrc_app_in")
	err = os.MkdirAll(inPath, 0o755)
	panicif(err)

	outPath := filepath.Join(wd, "msrc_app_out")
	err = os.MkdirAll(outPath, 0o755)
	panicif(err)

	now := time.Now()

	ctx := context.Background()

	githubHttp := fleethttp.NewGithubClient()
	ghAPI := io.NewGitHubClient(githubHttp, github.NewClient(githubHttp).Repositories, wd)

	msrcHttp := fleethttp.NewClient()
	msrcAPI := msrc.NewMSRCClient(msrcHttp, inPath, msrc.MSRCBaseURL)

	fmt.Println("Downloading existing MSRC app bulletin...")
	existingMeta, existingURL, err := ghAPI.MSRCAppBulletin(ctx)
	panicif(err)

	var bulletins map[string]*msrcapps.AppBulletin
	if existingURL == "" {
		fmt.Println("None found, backfilling...")
		bulletins, err = backfillApps(now.Month(), now.Year(), msrcAPI)
		panicif(err)
	} else {
		fmt.Println("Updating existing bulletins")
		bulletins, err = updateApps(now.Month(), now.Year(), existingMeta, existingURL, msrcAPI, ghAPI)
		panicif(err)
	}

	fmt.Println("Saving app bulletins...")
	bulletinFile := msrcapps.FromMap(bulletins).WithMappings(msrcapps.DefaultMappings())
	err = bulletinFile.Serialize(now, outPath)
	panicif(err)

	fmt.Println("Done processing MSRC app feed.")
}

func updateApps(
	m time.Month,
	y int,
	existingMeta io.MetadataFileName,
	existingURL string,
	msrcClient msrc.MSRCAPI,
	ghClient io.GitHubAPI,
) (map[string]*msrcapps.AppBulletin, error) {
	fmt.Println("Downloading current feed...")
	currentFeed, err := msrcClient.GetFeed(m, y)
	if err != nil {
		if errors.Is(err, msrc.FeedNotFound) && windowsBulletinGracePeriod(m, y) {
			fmt.Printf("Current month feed %d-%d was not found, skipping...\n", y, m)
		} else {
			return nil, err
		}
	}

	var newBulletins map[string]*msrcapps.AppBulletin
	if currentFeed != "" {
		fmt.Println("Parsing current feed for apps...")
		newBulletins, err = msrc.ParseAppFeed(currentFeed)
		if err != nil {
			return nil, err
		}
	}

	// Download existing bulletin file and load it
	bulletins := make(map[string]*msrcapps.AppBulletin)

	if existingURL != "" {
		fPath, err := ghClient.Download(existingURL)
		if err != nil {
			return nil, err
		}

		existingFile, err := msrcapps.LoadAppBulletinFile(fPath)
		if err != nil {
			return nil, err
		}

		// Convert existing file to map for merging
		for _, p := range existingFile.Products {
			bulletins[p.Product] = &msrcapps.AppBulletin{
				ProductID:       p.ProductID,
				Product:         p.Product,
				SecurityUpdates: p.SecurityUpdates,
			}
		}
	}

	// Merge new updates into existing bulletins
	for name, newB := range newBulletins {
		if existing, ok := bulletins[name]; ok {
			msrcapps.MergeBulletins(existing, newB)
		} else {
			bulletins[name] = newB
		}
	}

	return bulletins, nil
}

func backfillApps(upToM time.Month, upToY int, client msrc.MSRCAPI) (map[string]*msrcapps.AppBulletin, error) {
	from := time.Date(msrc.MSRCMinYear, 1, 1, 0, 0, 0, 0, time.UTC)
	upTo := time.Date(upToY, upToM+1, 1, 0, 0, 0, 0, time.UTC)

	bulletins := make(map[string]*msrcapps.AppBulletin)
	for d := from; d.Before(upTo); d = d.AddDate(0, 1, 0) {
		fmt.Printf("Downloading feed for %d-%d...\n", d.Year(), d.Month())
		f, err := client.GetFeed(d.Month(), d.Year())
		if err != nil {
			if errors.Is(err, msrc.FeedNotFound) && windowsBulletinGracePeriod(d.Month(), d.Year()) {
				fmt.Printf("Current month feed %d-%d was not found, skipping...\n", d.Year(), d.Month())
				continue
			}
			return nil, err
		}

		fmt.Printf("Parsing feed for %d-%d for apps...\n", d.Year(), d.Month())
		r, err := msrc.ParseAppFeed(f)
		if err != nil {
			return nil, err
		}

		for name, newB := range r {
			if existing, ok := bulletins[name]; ok {
				msrcapps.MergeBulletins(existing, newB)
			} else {
				bulletins[name] = newB
			}
		}
	}

	return bulletins, nil
}

// windowsBulletinGracePeriod returns whether we are within the grace period for a MSRC monthly feed to exist.
func windowsBulletinGracePeriod(month time.Month, year int) bool {
	now := time.Now()
	return month == now.Month() && year == now.Year() && now.Day() <= 15
}
