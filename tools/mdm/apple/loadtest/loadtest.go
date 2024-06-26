// Package main implements the loadtest script defined in
// https://github.com/fleetdm/fleet/issues/11531.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
)

func printfAndPrompt(format string, a ...any) {
	printf(format+" (press enter to proceed)", a...)
	bufio.NewScanner(os.Stdin).Scan()
}

func printf(format string, a ...any) {
	fmt.Printf(time.Now().UTC().Format("2006-01-02T15:04:05Z")+": "+format, a...)
}

func main() {
	fleetURL := flag.String("fleet_url", "", "URL (with protocol and port of Fleet server)")
	apiToken := flag.String("api_token", "", "API authentication token to use on API calls")

	teamCount := flag.Int("team_count", 0, "Number of teams to create for the load test")
	teamExtraCount := flag.Int("team_extra_count", 0, "Number of extra teams to create for the load test")
	loopCount := flag.Int("loop_count", 1, "Number of times it will loop with the extra teams (default 1)")

	cleanupTeams := flag.Bool("cleanup_teams", false, "Cleans up (removes) teams from previous runs")

	flag.Parse()

	if *fleetURL == "" {
		log.Fatal("missing fleet_url argument")
	}
	if *apiToken == "" {
		log.Fatal("missing api_token argument")
	}
	apiClient, err := service.NewClient(*fleetURL, true, "", "")
	if err != nil {
		panic(err)
	}
	apiClient.SetToken(*apiToken)

	if *cleanupTeams {
		teams, err := apiClient.ListTeams("")
		if err != nil {
			log.Fatalf("list teams: %s", err)
		}
		printfAndPrompt("Cleaning up all (%d) teams...", len(teams))
		for _, team := range teams {
			if err := apiClient.DeleteTeam(team.ID); err != nil {
				log.Fatalf("delete team %s: %s", team.Name, err)
			}
		}
		return
	}

	if *teamCount == 0 {
		log.Fatal("missing team_count argument")
	}
	if *teamExtraCount == 0 {
		log.Fatal("missing team_extra_count argument")
	}

	const (
		hostCountPerTeam = 1
	)

	hosts, err := apiClient.GetHosts("")
	if err != nil {
		log.Fatalf("list hosts: %s", err)
	}

	if len(hosts) != *teamCount {
		log.Fatalf("host count (%d) must match expected team count (%d)", len(hosts), *teamCount)
	}

	printfAndPrompt("1. Creating %d teams...", *teamCount)
	start := time.Now()

	var teams []*fleet.Team
	for t := 0; t < *teamCount; t++ {
		team, err := apiClient.CreateTeam(fleet.TeamPayload{
			Name: ptr.String(fmt.Sprintf("Team %d", t)),
		})
		if err != nil {
			log.Fatalf("team create: %s", err)
		}
		teams = append(teams, team)
	}
	printf("1. Duration: %s\n", time.Since(start))

	printfAndPrompt("2. Transfering one host to each team...")
	start = time.Now()

	for i, host := range hosts {
		if err := apiClient.TransferHosts([]string{host.Hostname}, "", "", "", teams[i].Name); err != nil {
			log.Fatalf("transfer host %s to team %s: %s", host.Hostname, teams[i].Name, err)
		}
	}
	printf("2. Duration: %s\n", time.Since(start))

	printfAndPrompt("3a. Add %d profiles to all teams...", len(profiles))
	start = time.Now()

	for _, team := range teams {
		printf("Applying profiles to team %s...\n", team.Name)
		if err := apiClient.ApplyTeamProfiles(team.Name, profiles, fleet.ApplyTeamSpecOptions{}); err != nil {
			log.Fatalf("apply profiles to team %s: %s", team.Name, err)
		}
	}
	printf("3a. Duration: %s\n", time.Since(start))

	waitProfilesAppliedOnTeams := func(teamsToWait []*fleet.Team) {
		summaryDone := func(s fleet.MDMProfilesSummary) bool {
			return s.Verifying == uint(hostCountPerTeam)
		}
		teamSummaries := make(map[uint]fleet.MDMProfilesSummary)
		for {
			doneCount := 0
			for _, team := range teamsToWait {
				team := team
				if teamSummary, ok := teamSummaries[team.ID]; ok && summaryDone(teamSummary) {
					doneCount++
					continue
				}
				teamSummary, err := apiClient.GetConfigProfilesSummary(&team.ID)
				if err != nil {
					log.Fatalf("get config profile summary for team %s: %s", team.Name, err)
				}
				teamSummaries[team.ID] = *teamSummary
				if summaryDone(*teamSummary) {
					doneCount++
				}
			}
			if doneCount == len(teamsToWait) {
				break
			}

			printf("Waiting for all profiles to be applied on hosts..., summary: %+v\n", teamSummaries)
			time.Sleep(5 * time.Second)
		}
	}

	printf("3b. Waiting for all profiles to be applied on all teams...\n")
	start = time.Now()
	waitProfilesAppliedOnTeams(teams)
	printf("3b. Duration: %s\n", time.Since(start))

	printfAndPrompt("4a. Modify a profile on all teams...")
	start = time.Now()

	for _, team := range teams {
		teamProfiles, err := apiClient.ListProfiles(ptr.Uint(team.ID))
		if err != nil {
			log.Fatalf("load team %s profiles: ", team.Name)
		}
		if len(teamProfiles) != len(profiles) {
			log.Fatalf("invalid number of profiles in team %s: %d", team.Name, len(teamProfiles))
		}
		// Remove the last profile.
		lastProfile := teamProfiles[len(teamProfiles)-1]
		if err := apiClient.DeleteProfile(lastProfile.ProfileID); err != nil {
			log.Fatalf("delete profile %s for team %s", lastProfile.Identifier, team.Name)
		}
		// Add a new profile.
		if _, err := apiClient.AddProfile(team.ID, newProfile); err != nil {
			log.Fatalf("upload new profile for team %s", team.Name)
		}
	}
	printf("4a. Duration: %s\n", time.Since(start))

	printf("4b. Waiting for all profiles to be applied on all hosts of all teams...\n")
	start = time.Now()
	waitProfilesAppliedOnTeams(teams)
	printf("4b. Duration: %s\n", time.Since(start))

	for i := 0; i < *loopCount; i++ {
		printfAndPrompt("5. Creating extra %d teams...", *teamExtraCount)
		start = time.Now()

		var extraTeams []*fleet.Team
		for t := 0; t < *teamExtraCount; t++ {
			team, err := apiClient.CreateTeam(fleet.TeamPayload{
				Name: ptr.String(fmt.Sprintf("Team Extra %d", t)),
			})
			if err != nil {
				log.Fatalf("team create: %s", err)
			}
			extraTeams = append(extraTeams, team)
		}
		printf("5. Duration: %s\n", time.Since(start))

		printfAndPrompt("6a. Moving one host to each new extra %d teams...", *teamExtraCount)
		start = time.Now()

		for t := 0; t < *teamExtraCount; t++ {
			if err := apiClient.TransferHosts([]string{hosts[t].Hostname}, "", "", "", extraTeams[t].Name); err != nil {
				log.Fatalf("transfer host %s to team %s: %s", hosts[t].Hostname, extraTeams[t].Name, err)
			}
		}
		printf("6a. Duration: %s\n", time.Since(start))

		printf("6b. Waiting for all profiles to be applied on all hosts of the extra teams...\n")
		start = time.Now()
		waitProfilesAppliedOnTeams(extraTeams)
		printf("6b. Duration: %s\n", time.Since(start))

		printfAndPrompt("7a. Add %d profiles to all extra %d teams...", len(profiles), len(extraTeams))
		start = time.Now()

		for _, team := range extraTeams {
			if err := apiClient.ApplyTeamProfiles(team.Name, profiles, fleet.ApplyTeamSpecOptions{}); err != nil {
				log.Fatalf("apply profiles to extra team %s: %s", team.Name, err)
			}
		}
		printf("7a. Duration: %s\n", time.Since(start))

		printf("7b. Waiting for all profiles to be applied on all hosts of the extra teams...\n")
		start = time.Now()
		waitProfilesAppliedOnTeams(extraTeams)
		printf("7b. Duration: %s\n", time.Since(start))

		printfAndPrompt("8. Destroy %d extra teams...", len(extraTeams))
		start = time.Now()

		for _, extraTeam := range extraTeams {
			if err := apiClient.DeleteTeam(extraTeam.ID); err != nil {
				log.Fatalf("delete extra team %s: %s", extraTeam.Name, err)
			}
		}
		printf("8. Duration: %s\n", time.Since(start))
	}
}

var profiles = []fleet.MDMProfileBatchPayload{
	{
		Name: "Ensure Install Security Responses and System Files Is Enabled",
		Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>PayloadContent</key>
        <array>
                <dict>
                        <key>PayloadDisplayName</key>
                        <string>test</string>
                        <key>PayloadType</key>
                        <string>com.apple.SoftwareUpdate</string>
                        <key>PayloadIdentifier</key>
                        <string>com.fleetdm.cis-1.6.check</string>
                        <key>PayloadUUID</key>
                        <string>0D8F676A-A705-4F57-8FF8-3118360EFDEB</string>
                        <key>ConfigDataInstall</key>
                        <true/>
                        <key>CriticalUpdateInstall</key>
                        <true/>
                </dict>
        </array>
        <key>PayloadDescription</key>
        <string>test</string>
        <key>PayloadDisplayName</key>
        <string>Ensure Install Security Responses and System Files Is Enabled</string>
        <key>PayloadIdentifier</key>
        <string>com.fleetdm.cis-1.6</string>
        <key>PayloadRemovalDisallowed</key>
        <false/>
        <key>PayloadScope</key>
        <string>System</string>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadUUID</key>
        <string>EBEE9B81-9D33-477F-AFBE-9691360B7A74</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
</dict>
</plist>`),
	},
	{
		Name: "Ensure Software Update Deferment Is Less Than or Equal to 30 Days",
		Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>PayloadContent</key>
        <array>
                <dict>
                        <key>PayloadDisplayName</key>
                        <string>test</string>
                        <key>PayloadType</key>
                        <string>com.apple.applicationaccess</string>
                        <key>PayloadIdentifier</key>
                        <string>com.fleetdm.cis-1.7.check</string>
                        <key>PayloadUUID</key>
                        <string>123FD592-D1C3-41FD-BC41-F91F3E1E2CF4</string>
                        <key>enforcedSoftwareUpdateDelay</key>
                        <integer>29</integer>
                </dict>
        </array>
        <key>PayloadDescription</key>
        <string>test</string>
        <key>PayloadDisplayName</key>
        <string>Ensure Software Update Deferment Is Less Than or Equal to 30 Days</string>
        <key>PayloadIdentifier</key>
        <string>com.zwass.cis-1.7</string>
        <key>PayloadRemovalDisallowed</key>
        <false/>
        <key>PayloadScope</key>
        <string>System</string>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadUUID</key>
        <string>385A0C13-2472-41B3-851C-1311FA12EB49</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
</dict>
</plist>`),
	},
	{
		Name: "Ensure Auto Update Is Enabled",
		Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>PayloadContent</key>
        <array>
                <dict>
                        <key>PayloadDisplayName</key>
                        <string>test</string>
                        <key>PayloadType</key>
                        <string>com.apple.SoftwareUpdate</string>
                        <key>PayloadIdentifier</key>
                        <string>com.fleetdm.cis-1.2.check</string>
                        <key>PayloadUUID</key>
                        <string>4DC539B5-837E-4DC3-B60B-43A8C556A8F0</string>
                        <key>AutomaticCheckEnabled</key>
                        <true/>
                </dict>
        </array>
        <key>PayloadDescription</key>
        <string>test</string>
        <key>PayloadDisplayName</key>
        <string>Ensure Auto Update Is Enabled</string>
        <key>PayloadIdentifier</key>
        <string>com.fleetdm.cis-1.2</string>
        <key>PayloadRemovalDisallowed</key>
        <false/>
        <key>PayloadScope</key>
        <string>System</string>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadUUID</key>
        <string>03E69A02-02CE-4CA0-8F17-3BAAD5D3852F</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
</dict>
</plist>`),
	},
	{
		Name: "Ensure Download New Updates When Available Is Enabled",
		Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>PayloadContent</key>
        <array>
                <dict>
                        <key>PayloadDisplayName</key>
                        <string>test</string>
                        <key>PayloadType</key>
                        <string>com.apple.SoftwareUpdate</string>
                        <key>PayloadIdentifier</key>
                        <string>com.fleetdm.cis-1.3.check</string>
                        <key>PayloadUUID</key>
                        <string>5FDE6D58-79CD-447A-AFB0-BA32D889C396</string>
                        <key>AutomaticDownload</key>
                        <true/>
                </dict>
        </array>
        <key>PayloadDescription</key>
        <string>test</string>
        <key>PayloadDisplayName</key>
        <string>Ensure Download New Updates When Available Is Enabled</string>
        <key>PayloadIdentifier</key>
        <string>com.fleetdm.cis-1.3</string>
        <key>PayloadRemovalDisallowed</key>
        <false/>
        <key>PayloadScope</key>
        <string>System</string>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadUUID</key>
        <string>0A1C2F97-D6FA-4CDB-ABB6-47DF2B151F4F</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
</dict>
</plist>`),
	},
	{
		Name: "Ensure Install of macOS Updates Is Enabled",
		Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>PayloadContent</key>
        <array>
                <dict>
                        <key>PayloadDisplayName</key>
                        <string>test</string>
                        <key>PayloadType</key>
                        <string>com.apple.SoftwareUpdate</string>
                        <key>PayloadIdentifier</key>
                        <string>com.fleetdm.cis-1.4.check</string>
                        <key>PayloadUUID</key>
                        <string>15BF7634-276A-411B-8C4E-52D89B4ED82C</string>
                        <key>AutomaticallyInstallMacOSUpdates</key>
                        <true/>
                </dict>
        </array>
        <key>PayloadDescription</key>
        <string>test</string>
        <key>PayloadDisplayName</key>
        <string>Ensure Install of macOS Updates Is Enabled</string>
        <key>PayloadIdentifier</key>
        <string>com.fleetdm.cis-1.4</string>
        <key>PayloadRemovalDisallowed</key>
        <false/>
        <key>PayloadScope</key>
        <string>System</string>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadUUID</key>
        <string>7DB8733E-BD11-4E88-9AE0-273EF2D0974B</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
</dict>
</plist>`),
	},
	{
		Name: "Ensure Firewall Logging Is Enabled and Configured",
		Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>PayloadContent</key>
        <array>
                <dict>
                        <key>PayloadDisplayName</key>
                        <string>test</string>
                        <key>PayloadType</key>
                        <string>com.apple.security.firewall</string>
                        <key>PayloadIdentifier</key>
                        <string>com.fleetdm.cis-3.6.check</string>
                        <key>PayloadUUID</key>
                        <string>604D8218-D7B6-43B1-95E6-DFCA4C25D73D</string>
                        <key>EnableFirewall</key>
                        <true/>
                        <key>EnableLogging</key>
                        <true/>
                        <key>LoggingOption</key>
                        <string>detail</string>
                </dict>
        </array>
        <key>PayloadDescription</key>
        <string>test</string>
        <key>PayloadDisplayName</key>
        <string>Ensure Firewall Logging Is Enabled and Configured</string>
        <key>PayloadIdentifier</key>
        <string>com.fleetdm.cis-3.6</string>
        <key>PayloadRemovalDisallowed</key>
        <false/>
        <key>PayloadScope</key>
        <string>System</string>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadUUID</key>
        <string>5E27501E-50DF-4804-9DEC-0E63C34E8831</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
</dict>
</plist>`),
	},
	{
		Name: "Ensure Bonjour Advertising Services Is Disabled",
		Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>PayloadContent</key>
        <array>
                <dict>
                        <key>PayloadDisplayName</key>
                        <string>test</string>
                        <key>PayloadType</key>
                        <string>com.apple.mDNSResponder</string>
                        <key>PayloadIdentifier</key>
                        <string>com.fleetdm.cis-4.1.check</string>
                        <key>PayloadUUID</key>
                        <string>08FEA43B-CE9B-4098-804C-11459D109992</string>
                        <key>NoMulticastAdvertisements</key>
                        <true/>
                </dict>
        </array>
        <key>PayloadDescription</key>
        <string>test</string>
        <key>PayloadDisplayName</key>
        <string>Ensure Bonjour Advertising Services Is Disabled</string>
        <key>PayloadIdentifier</key>
        <string>com.fleetdm.cis-4.1</string>
        <key>PayloadRemovalDisallowed</key>
        <false/>
        <key>PayloadScope</key>
        <string>System</string>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadUUID</key>
        <string>25BD1312-2B79-40C7-99FA-E60B49A1883E</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
</dict>
</plist>`),
	},
	{
		Name: "Disable Bluetooth sharing",
		Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>

        <key>PayloadDescription</key>
        <string>This profile configuration is designed to apply the CIS Benchmark for macOS 10.14 (v2.0.0), 10.15 (v2.0.0), 11.0 (v2.0.0), and 12.0 (v1.0.0)</string>
        <key>PayloadDisplayName</key>
        <string>Disable Bluetooth sharing</string>
        <key>PayloadEnabled</key>
        <true/>
        <key>PayloadIdentifier</key>
        <string>cis.macOSBenchmark.section2.BluetoothSharing</string>
        <key>PayloadScope</key>
        <string>System</string>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadUUID</key>
        <string>5CEBD712-28EB-432B-84C7-AA28A5A383D8</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
    <key>PayloadRemovalDisallowed</key>
    <true/>
        <key>PayloadContent</key>
        <array>
                <dict>
                        <key>PayloadContent</key>
                        <dict>
                                <key>com.apple.Bluetooth</key>
                                <dict>
                                        <key>Forced</key>
                                        <array>
                                                <dict>
                                                        <key>mcx_preference_settings</key>
                                                        <dict>
                                                                <key>PrefKeyServicesEnabled</key>
                                                                <false/>
                                                        </dict>
                                                </dict>
                                        </array>
                                </dict>
                        </dict>
                        <key>PayloadDescription</key>
                        <string>Disables Bluetooth Sharing</string>
                        <key>PayloadDisplayName</key>
                        <string>Custom</string>
                        <key>PayloadEnabled</key>
                        <true/>
                        <key>PayloadIdentifier</key>
                        <string>0240DD1C-70DC-4766-9018-04322BFEEAD1</string>
                        <key>PayloadType</key>
                        <string>com.apple.ManagedClient.preferences</string>
                        <key>PayloadUUID</key>
                        <string>0240DD1C-70DC-4766-9018-04322BFEEAD1</string>
                        <key>PayloadVersion</key>
                        <integer>1</integer>
                </dict>
        </array>
</dict>
</plist>`),
	},
	{
		Name: "Ensure Install Application Updates from the App Store Is Enabled",
		Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>PayloadContent</key>
        <array>
                <dict>
                        <key>PayloadDisplayName</key>
                        <string>test</string>
                        <key>PayloadType</key>
                        <string>com.apple.SoftwareUpdate</string>
                        <key>PayloadIdentifier</key>
                        <string>com.fleetdm.cis-1.5.check</string>
                        <key>PayloadUUID</key>
                        <string>6B0285F8-5DB8-4F68-AA6E-2333CCD6CE04</string>
                        <key>AutomaticallyInstallAppUpdates</key>
                        <true/>
                </dict>
        </array>
        <key>PayloadDescription</key>
        <string>test</string>
        <key>PayloadDisplayName</key>
        <string>Ensure Install Application Updates from the App Store Is Enabled</string>
        <key>PayloadIdentifier</key>
        <string>com.fleetdm.cis-1.5</string>
        <key>PayloadRemovalDisallowed</key>
        <false/>
        <key>PayloadScope</key>
        <string>System</string>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadUUID</key>
        <string>1C4C0EC4-64A7-4AF0-8807-A3DD44A6DC76</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
</dict>
</plist>`),
	},
	{
		Name: "Disable iCloud Drive storage solution usage",
		Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>PayloadContent</key>
        <array>
                <dict>
                        <key>PayloadDisplayName</key>
                        <string>test</string>
                        <key>PayloadType</key>
                        <string>com.apple.applicationaccess</string>
                        <key>PayloadIdentifier</key>
                        <string>com.fleetdm.cis-2.1.1.2.check-disable</string>
                        <key>PayloadUUID</key>
                        <string>1028E002-9AFE-446A-84E0-27DA5DA39B4A</string>
                        <key>allowCloudDocumentSync</key>
                        <false/>
                </dict>
        </array>
        <key>PayloadDescription</key>
        <string>test</string>
        <key>PayloadDisplayName</key>
        <string>Disable iCloud Drive storage solution usage</string>
        <key>PayloadIdentifier</key>
        <string>com.fleetdm.cis-2.1.1.2-disable</string>
        <key>PayloadRemovalDisallowed</key>
        <false/>
        <key>PayloadScope</key>
        <string>System</string>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadUUID</key>
        <string>7B3DE4EA-0AFA-44F5-9716-37526EE441EA</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
</dict>
</plist>`),
	},
}

var newProfile = []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
        <dict>
                <key>PayloadContent</key>
                <array>
                        <dict>
                                <key>PayloadDisplayName</key>
                                <string>test</string>
                                <key>PayloadType</key>
                                <string>com.apple.Safari</string>
                                <key>PayloadIdentifier</key>
                                <string>com.fleetdm.cis-6.3.1.check</string>
                                <key>PayloadUUID</key>
                                <string>3CAAC721-D492-45AC-95E4-8ECBF81EA21E</string>
                                <key>AutoOpenSafeDownloads</key>
                                <false/>
                        </dict>
                </array>
                <key>PayloadDescription</key>
                <string>test</string>
                <key>PayloadDisplayName</key>
                <string>Ensure Automatic Opening of Safe Files in Safari Is Disabled</string>
                <key>PayloadIdentifier</key>
                <string>com.fleetdm.cis-6.3.1</string>
                <key>PayloadRemovalDisallowed</key>
                <false/>
                <key>PayloadScope</key>
                <string>System</string>
                <key>PayloadType</key>
                <string>Configuration</string>
                <key>PayloadUUID</key>
                <string>2556F162-9AE5-4163-92C1-F89A2847C80E</string>
                <key>PayloadVersion</key>
                <integer>1</integer>
        </dict>
</plist>`)
