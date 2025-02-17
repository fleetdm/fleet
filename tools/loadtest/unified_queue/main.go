package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
)

func printf(format string, a ...any) {
	fmt.Printf(time.Now().UTC().Format("2006-01-02T15:04:05Z")+": "+format, a...)
}

func main() {
	fleetURL := flag.String("fleet_url", "", "URL (with protocol and port of Fleet server)")
	apiToken := flag.String("api_token", "", "API authentication token to use on API calls")
	debug := flag.Bool("debug", false, "Debug mode")

	flag.Parse()

	if *fleetURL == "" {
		log.Fatal("missing fleet_url argument")
	}
	if *apiToken == "" {
		log.Fatal("missing api_token argument")
	}
	var clientOpts []service.ClientOption
	if *debug {
		clientOpts = append(clientOpts, service.EnableClientDebug())
	}
	apiClient, err := service.NewClient(*fleetURL, true, "", "", clientOpts...)
	if err != nil {
		log.Fatal(err)
	}
	apiClient.SetToken(*apiToken)

	printf("Fetching hosts...\n")
	records, err := apiClient.GetHostsReport("id", "hostname", "platform")
	if err != nil {
		log.Fatal(err)
	}
	type smallHost struct {
		ID       uint
		Hostname string
		Platform string
	}
	var (
		macOSHosts   []smallHost
		windowsHosts []smallHost
		linuxHosts   []smallHost
	)
	for i, record := range records {
		if i == 0 {
			continue
		}
		hostID, _ := strconv.Atoi(record[0])
		hostname := record[1]
		platform := fleet.PlatformFromHost(record[2])
		switch platform {
		case "linux":
			linuxHosts = append(linuxHosts, smallHost{ID: uint(hostID), Hostname: hostname, Platform: platform}) // nolint:gosec
		case "darwin":
			macOSHosts = append(macOSHosts, smallHost{ID: uint(hostID), Hostname: hostname, Platform: platform}) // nolint:gosec
		case "windows":
			windowsHosts = append(windowsHosts, smallHost{ID: uint(hostID), Hostname: hostname, Platform: platform}) // nolint:gosec
		}
	}
	printf("Got linux=%d, windows=%d, macOS=%d\n", len(linuxHosts), len(windowsHosts), len(macOSHosts))

	titles, err := apiClient.ListSoftwareTitles("per_page=1000&team_id=0&available_for_install=1")
	if err != nil {
		log.Fatal(err)
	}

	var (
		macOSSoftware   []fleet.SoftwareTitleListResult
		windowsSoftware []fleet.SoftwareTitleListResult
	)
	for _, title := range titles {
		if title.AppStoreApp != nil {
			macOSSoftware = append(macOSSoftware, title)
		} else if title.SoftwarePackage != nil {
			if ext := filepath.Ext(title.SoftwarePackage.Name); ext == ".exe" || ext == ".msi" {
				windowsSoftware = append(windowsSoftware, title)
			} else {
				macOSSoftware = append(macOSSoftware, title)
			}
		}
	}
	printf("Got software titles windows=%d, macOS=%d\n", len(windowsSoftware), len(macOSSoftware))

	scripts, err := apiClient.ListScripts("per_page=1000&team_id=0")
	if err != nil {
		log.Fatal(err)
	}

	var (
		macOSScripts   []string
		windowsScripts []string
	)
	for _, script := range scripts {
		if strings.HasSuffix(script.Name, ".sh") {
			macOSScripts = append(macOSScripts, script.Name)
		} else if strings.HasSuffix(script.Name, ".ps1") {
			windowsScripts = append(windowsScripts, script.Name)
		}
	}
	printf("Got scripts windows=%d, macOS=%d\n", len(windowsScripts), len(macOSScripts))

	var queuedScripts, queuedInstalls, hostsTargeted, errors int
	targetedHosts := append(macOSHosts, windowsHosts...) // nolint:gocritic
	rand.Shuffle(len(targetedHosts), func(i, j int) {
		targetedHosts[i], targetedHosts[j] = targetedHosts[j], targetedHosts[i]
	})

	tick := time.Tick(300 * time.Millisecond)
	for i, host := range targetedHosts {
		<-tick

		if hostsTargeted > 0 && hostsTargeted%500 == 0 {
			printf("In progress: queued scripts=%d, queued installs=%d, hosts targeted=%d, errors=%d\n", queuedScripts, queuedInstalls, hostsTargeted, errors)
		}

		switch host.Platform {
		case "darwin":
			hostsTargeted++

			// enqueue a software install and a couple scripts
			_, err := apiClient.RunHostScriptAsync(host.ID, nil, macOSScripts[i%len(macOSScripts)], 0)
			if err != nil {
				printf("Failed to run script on host %v (%v): %v\n", host.Hostname, host.Platform, err)
				errors++
				continue
			}
			queuedScripts++
			_, err = apiClient.RunHostScriptAsync(host.ID, nil, macOSScripts[(i+1)%len(macOSScripts)], 0)
			if err != nil {
				printf("Failed to run script on host %v (%v): %v\n", host.Hostname, host.Platform, err)
				errors++
				continue
			}
			queuedScripts++

			err = apiClient.InstallSoftware(host.ID, macOSSoftware[i%len(macOSSoftware)].ID)
			if err != nil {
				printf("Failed to install software on host %v (%v): %v\n", host.Hostname, host.Platform, err)
				errors++
				continue
			}
			queuedInstalls++

		case "windows":
			hostsTargeted++

			// enqueue a couple software installs and a script
			err = apiClient.InstallSoftware(host.ID, windowsSoftware[i%len(windowsSoftware)].ID)
			if err != nil {
				printf("Failed to install software on host %v (%v): %v\n", host.Hostname, host.Platform, err)
				errors++
				continue
			}
			queuedInstalls++

			err = apiClient.InstallSoftware(host.ID, windowsSoftware[(i+1)%len(windowsSoftware)].ID)
			if err != nil {
				printf("Failed to install software on host %v (%v): %v\n", host.Hostname, host.Platform, err)
				errors++
				continue
			}
			queuedInstalls++

			_, err = apiClient.RunHostScriptAsync(host.ID, nil, windowsScripts[i%len(windowsScripts)], 0)
			if err != nil {
				printf("Failed to run script on host %v (%v): %v\n", host.Hostname, host.Platform, err)
				errors++
				continue
			}
			queuedScripts++
		}
	}

	printf("Done: queued scripts=%d, queued installs=%d, hosts targeted=%d, errors=%d\n", queuedScripts, queuedInstalls, hostsTargeted, errors)
}
