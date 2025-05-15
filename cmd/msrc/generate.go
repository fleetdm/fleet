package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/parsed"
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

	inPath := filepath.Join(wd, "msrc_in")
	err = os.MkdirAll(inPath, 0o755)
	panicif(err)

	outPath := filepath.Join(wd, "msrc_out")
	err = os.MkdirAll(outPath, 0o755)
	panicif(err)

	now := time.Now()

	ctx := context.Background()

	githubHttp := fleethttp.NewGithubClient()
	ghAPI := io.NewGitHubClient(githubHttp, github.NewClient(githubHttp).Repositories, wd)

	msrcHttp := fleethttp.NewClient() // don't reuse the GitHub client as it has an OAuth token baked in
	msrcAPI := msrc.NewMSRCClient(msrcHttp, inPath, msrc.MSRCBaseURL)

	fmt.Println("Downloading existing MSRC bulletins...")
	eBulletins, err := ghAPI.MSRCBulletins(ctx)
	panicif(err)

	var bulletins []*parsed.SecurityBulletin
	if len(eBulletins) == 0 {
		fmt.Println("None found, backfilling...")
		bulletins, err = backfill(now.Month(), now.Year(), msrcAPI)
		panicif(err)
	} else {
		fmt.Println("Updating existing bulletins")
		bulletins, err = update(now.Month(), now.Year(), eBulletins, msrcAPI, ghAPI)
		panicif(err)
	}

	fmt.Println("Saving bulletins...")
	for _, b := range bulletins {
		err := serialize(b, now, outPath)
		panicif(err)
	}

	fmt.Println("Done processing MSRC feed.")
}

// windowsBulletinGracePeriod returns whether we are within the grace period for a MSRC monthly feed to exist.
//
// E.g. September 2024 bulletin was released on the 2nd, thus we add some grace period (5 days)
// for Microsoft to publish the current month bulletin.
func windowsBulletinGracePeriod(month time.Month, year int) bool {
	now := time.Now()
	return month == now.Month() && year == now.Year() && now.Day() <= 5
}

func update(
	m time.Month,
	y int,
	eBulletins map[io.MetadataFileName]string,
	msrcClient msrc.MSRCAPI,
	ghClient io.GitHubAPI,
) ([]*parsed.SecurityBulletin, error) {
	fmt.Println("Downloading current feed...")
	currentFeed, err := msrcClient.GetFeed(m, y)
	if err != nil {
		if errors.Is(err, msrc.FeedNotFound) && windowsBulletinGracePeriod(m, y) {
			fmt.Printf("Current month feed %d-%d was not found, skipping...\n", y, m)
		} else {
			return nil, err
		}
	}

	var nBulletins map[string]*parsed.SecurityBulletin
	if currentFeed != "" {
		fmt.Println("Parsing current feed...")
		nBulletins, err = msrc.ParseFeed(currentFeed)
		if err != nil {
			return nil, err
		}
	}

	var bulletins []*parsed.SecurityBulletin
	for _, url := range eBulletins {
		fPath, err := ghClient.Download(url)
		if err != nil {
			return nil, err
		}

		eB, err := parsed.UnmarshalBulletin(fPath)
		if err != nil {
			return nil, err
		}

		nB, ok := nBulletins[eB.ProductName]
		if ok {
			if err = eB.Merge(nB); err != nil {
				return nil, err
			}
		}

		bulletins = append(bulletins, eB)
	}

	return bulletins, nil
}

func backfill(upToM time.Month, upToY int, client msrc.MSRCAPI) ([]*parsed.SecurityBulletin, error) {
	from := time.Date(msrc.MSRCMinYear, 1, 1, 0, 0, 0, 0, time.UTC)
	upTo := time.Date(upToY, upToM+1, 1, 0, 0, 0, 0, time.UTC)

	bulletins := make(map[string]*parsed.SecurityBulletin)
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

		fmt.Printf("Parsing feed for %d-%d...\n", d.Year(), d.Month())
		r, err := msrc.ParseFeed(f)
		if err != nil {
			return nil, err
		}

		for name, nB := range r {
			eB, ok := bulletins[name]
			if !ok {
				bulletins[name] = nB
				continue
			}

			if err = eB.Merge(nB); err != nil {
				return nil, err
			}
		}
	}

	var r []*parsed.SecurityBulletin
	for _, b := range bulletins {
		r = append(r, b)
	}

	return r, nil
}

func serialize(b *parsed.SecurityBulletin, d time.Time, dir string) error {
	payload, err := json.Marshal(b)
	if err != nil {
		return err
	}
	fileName := io.MSRCFileName(b.ProductName, d)
	filePath := filepath.Join(dir, fileName)

	return os.WriteFile(filePath, payload, 0o644)
}
