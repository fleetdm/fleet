package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/io"
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

	now := time.Now()
	httpC := http.DefaultClient

	ghAPI := io.NewGithubClient(httpC, github.NewClient(httpC).Repositories, wd)
	msrcAPI := io.NewMSRCClient(httpC, wd, nil)

	fmt.Println("Downloading current feed...")
	f, err := msrcAPI.GetFeed(now.Month(), now.Year())
	panicif(err)

	fmt.Println("Parsing current feed...")
	nBulletins, err := msrc.ParseFeed(f)
	panicif(err)

	fmt.Println("Downloading existing bulletins...")
	eBulletins, err := ghAPI.Bulletins()
	panicif(err)

	fmt.Println("Mergin bulletins...")
	var bulletins []*parsed.SecurityBulletin
	for _, url := range eBulletins {
		fPath, err := ghAPI.Download(url)
		panicif(err)

		bulletin, err := parsed.UnmarshalBulletin(fPath)
		panicif(err)

		nB, ok := nBulletins[bulletin.ProductName]
		if ok {
			err = nB.Merge(bulletin)
			panicif(err)
		}

		bulletins = append(bulletins, bulletin)
	}

	fmt.Println("Saving bulletins...")
	for _, b := range bulletins {
		err := serialize(b, now, wd)
		panicif(err)
	}

	// TODO: Generate checksums

	fmt.Println("Parsed .")
	fmt.Println("Done.")
}

func serialize(b *parsed.SecurityBulletin, date time.Time, wd string) error {
	panic("not implemented")
}
