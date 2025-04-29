package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/google/go-github/v37/github"
)

func main() {
	n := flag.Int("last-minor-releases", 0, "Output number of Fleet minor releases (with highest patch number)")
	flag.Parse()

	if *n <= 0 {
		log.Fatal("Set a valid --last-minor-releases value")
	}

	c := github.NewClient(fleethttp.NewGithubClient()).Repositories
	githubReleases, _, err := c.ListReleases(context.Background(), "fleetdm", "fleet", &github.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}

	var releaseVersions []string
	for _, gr := range githubReleases {
		releaseVersions = append(releaseVersions, strings.TrimPrefix(*gr.Name, "fleet-"))
	}

	sort.Slice(releaseVersions, func(i, j int) bool {
		return releaseVersions[i] > releaseVersions[j]
	})

	lastMinor := releaseVersions[0]
	outputReleases := []string{lastMinor}
	for _, version := range releaseVersions {
		if len(outputReleases) >= *n {
			break
		}
		lastMinorPart := strings.Split(lastMinor, ".")[1]
		minor := strings.Split(version, ".")[1]
		if minor < lastMinorPart {
			outputReleases = append(outputReleases, version)
			lastMinor = version
		}
	}

	for i, version := range outputReleases {
		if i != 0 {
			fmt.Printf(" ")
		}
		fmt.Printf("%s", version)
	}
}
