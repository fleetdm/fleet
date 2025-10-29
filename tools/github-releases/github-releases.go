package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/go-github/v37/github"
	"golang.org/x/mod/semver"
)

func main() {
	lastMinorReleases := flag.Int("last-minor-releases", 0, "Output number of Fleet minor releases (with highest patch number)")
	separator := flag.String("separator", " ", "Separator string to use in the output")
	allCPEs := flag.Bool("all-cpes", false, "Output all Fleet version releases as CPEs")
	flag.Parse()

	if *lastMinorReleases <= 0 && !*allCPEs {
		log.Fatal("Set --last-minor-releases or --all-cpes value")
	}
	if *lastMinorReleases > 0 && *allCPEs {
		log.Fatal("Cannot set both --last-minor-releases or --all-cpes")
	}

	c := github.NewClient(fleethttp.NewGithubClient()).Repositories
	var (
		githubReleases []*github.RepositoryRelease
		err            error
	)
	var releaseVersions []string
	if *allCPEs {
		for page := 1; ; page++ {
			releases, _, err := c.ListReleases(context.Background(), "fleetdm", "fleet", &github.ListOptions{Page: page, PerPage: 100})
			if err != nil {
				log.Fatal(err)
			}
			if len(releases) == 0 {
				break
			}
			for _, release := range releases {
				if strings.HasPrefix(*release.Name, "orbit-") {
					continue
				}
				if strings.HasPrefix(*release.Name, "fleet-") {
					versionWithoutPrefix := strings.TrimPrefix(*release.Name, "fleet-")
					release.Name = ptr.String(versionWithoutPrefix)
				}
				if strings.HasPrefix(*release.Name, "Fleet ") {
					versionWithoutPrefix := strings.TrimPrefix(*release.Name, "Fleet ")
					release.Name = ptr.String(versionWithoutPrefix)
				}
				if (*release.Name)[0] != 'v' {
					versionWithPrefix := "v" + *release.Name
					release.Name = ptr.String(versionWithPrefix)
				}
				releaseVersions = append(releaseVersions, *release.Name)
			}
			time.Sleep(500 * time.Millisecond)
		}
	} else {
		githubReleases, _, err = c.ListReleases(context.Background(), "fleetdm", "fleet", &github.ListOptions{})
		if err != nil {
			log.Fatal(err)
		}
		for _, gr := range githubReleases {
			releaseVersions = append(releaseVersions, strings.TrimPrefix(*gr.Name, "fleet-"))
		}
	}

	semver.Sort(releaseVersions)
	slices.Reverse(releaseVersions)

	outputReleases := releaseVersions
	if *lastMinorReleases > 0 {
		outputReleases = runLastMinorReleases(releaseVersions, *lastMinorReleases)
	}

	var versions []string
	for _, version := range outputReleases {
		if *allCPEs {
			version = "cpe:2.3:a:fleetdm:fleet:" + version + ":*:*:*:*:*:*:*"
		}
		versions = append(versions, version)
	}
	fmt.Printf("%s", strings.Join(versions, *separator))
}

func runLastMinorReleases(releaseVersions []string, n int) []string {
	lastMinor := releaseVersions[0]
	outputReleases := []string{lastMinor}
	for _, version := range releaseVersions {
		if len(outputReleases) >= n {
			break
		}
		lastMinorPart := strings.Split(lastMinor, ".")[1]
		minor := strings.Split(version, ".")[1]
		if minor < lastMinorPart {
			outputReleases = append(outputReleases, version)
			lastMinor = version
		}
	}
	return outputReleases
}
