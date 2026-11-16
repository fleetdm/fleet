package update

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/micromdm/plist"
	"github.com/stretchr/testify/require"
)

// profilesShowOutput has the shape of `profiles show -output stdout-xml`: a certificate payload whose content is data, a
// Full Disk Access profile like Fleet's sample, and one from another vendor that disagrees on one path.
const profilesShowOutput = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>_computerlevel</key>
	<array>
		<dict>
			<key>ProfileIdentifier</key>
			<string>com.fleetdm.caroot</string>
			<key>ProfileItems</key>
			<array>
				<dict>
					<key>PayloadContent</key>
					<data>MIIB</data>
					<key>PayloadType</key>
					<string>com.apple.security.root</string>
				</dict>
			</array>
		</dict>
		<dict>
			<key>ProfileIdentifier</key>
			<string>com.fleetdm.fleetd-fda</string>
			<key>ProfileItems</key>
			<array>
				<dict>
					<key>PayloadContent</key>
					<dict>
						<key>Services</key>
						<dict>
							<key>SystemPolicyAllFiles</key>
							<array>
								<dict>
									<key>Allowed</key>
									<true/>
									<key>Identifier</key>
									<string>/opt/orbit/bin/orbit/macos/stable/orbit</string>
									<key>IdentifierType</key>
									<string>path</string>
								</dict>
								<dict>
									<key>Allowed</key>
									<integer>1</integer>
									<key>Identifier</key>
									<string>/opt/orbit/bin/orbit/macos/stable/Fleet</string>
									<key>IdentifierType</key>
									<string>path</string>
								</dict>
								<dict>
									<key>Allowed</key>
									<false/>
									<key>Identifier</key>
									<string>/opt/orbit/bin/orbit/macos/edge/orbit</string>
									<key>IdentifierType</key>
									<string>path</string>
								</dict>
								<dict>
									<key>Authorization</key>
									<string>Deny</string>
									<key>Identifier</key>
									<string>/opt/orbit/bin/orbit/macos/edge/Fleet</string>
									<key>IdentifierType</key>
									<string>path</string>
								</dict>
							</array>
						</dict>
					</dict>
					<key>PayloadType</key>
					<string>com.apple.TCC.configuration-profile-policy</string>
				</dict>
			</array>
		</dict>
		<dict>
			<key>ProfileIdentifier</key>
			<string>com.example.pppc</string>
			<key>ProfileItems</key>
			<array>
				<dict>
					<key>PayloadContent</key>
					<dict>
						<key>Services</key>
						<dict>
							<key>Camera</key>
							<array>
								<dict>
									<key>Allowed</key>
									<true/>
									<key>Identifier</key>
									<string>/opt/orbit/bin/orbit/macos/1.70.0/Fleet</string>
									<key>IdentifierType</key>
									<string>path</string>
								</dict>
							</array>
							<key>SystemPolicyAllFiles</key>
							<array>
								<dict>
									<key>Authorization</key>
									<string>Allow</string>
									<key>Identifier</key>
									<string>/opt/orbit/bin/orbit/macos/1.69.0/orbit</string>
									<key>IdentifierType</key>
									<string>path</string>
								</dict>
								<dict>
									<key>Allowed</key>
									<true/>
									<key>Identifier</key>
									<string>/opt/orbit/bin/orbit/macos/edge/Fleet</string>
									<key>IdentifierType</key>
									<string>path</string>
								</dict>
							</array>
						</dict>
					</dict>
					<key>PayloadType</key>
					<string>com.apple.TCC.configuration-profile-policy</string>
				</dict>
			</array>
		</dict>
	</array>
</dict>
</plist>`

func TestParseFDAGrants(t *testing.T) {
	grants, err := parseFDAGrants([]byte(profilesShowOutput))
	require.NoError(t, err)
	require.True(t, grants.allows("/opt/orbit/bin/orbit/macos/stable/orbit"))
	require.True(t, grants.allows("/opt/orbit/bin/orbit/macos/stable/Fleet"))  // Allowed as an integer
	require.True(t, grants.allows("/opt/orbit/bin/orbit/macos/1.69.0/orbit"))  // Authorization, another profile
	require.False(t, grants.allows("/opt/orbit/bin/orbit/macos/edge/orbit"))   // not allowed
	require.False(t, grants.allows("/opt/orbit/bin/orbit/macos/edge/Fleet"))   // denied, though another profile allows it
	require.False(t, grants.allows("/opt/orbit/bin/orbit/macos/1.70.0/Fleet")) // another service

	// Only path entries in device profiles count.
	fdaProfile := func(identifierType, identifier string) string {
		return `<array><dict><key>ProfileItems</key><array><dict>
			<key>PayloadType</key><string>com.apple.TCC.configuration-profile-policy</string>
			<key>PayloadContent</key><dict><key>Services</key><dict><key>SystemPolicyAllFiles</key><array><dict>
				<key>Allowed</key><true/>
				<key>Identifier</key><string>` + identifier + `</string>
				<key>IdentifierType</key><string>` + identifierType + `</string>
			</dict></array></dict></dict>
		</dict></array></dict></array>`
	}
	grants, err = parseFDAGrants([]byte(`<plist version="1.0"><dict>
		<key>_computerlevel</key>` + fdaProfile("bundleID", "com.fleetdm.orbit") + `
		<key>someone</key>` + fdaProfile("path", "/opt/orbit/bin/orbit/macos/stable/Fleet") + `
	</dict></plist>`))
	require.NoError(t, err)
	require.False(t, grants.allows("/opt/orbit/bin/orbit/macos/stable/Fleet"))

	// No profiles.
	grants, err = parseFDAGrants([]byte(`<plist version="1.0"><dict/></plist>`))
	require.NoError(t, err)
	require.False(t, grants.allows("/opt/orbit/bin/orbit/macos/stable/orbit"))

	_, err = parseFDAGrants([]byte("not a plist"))
	require.Error(t, err)
}

func TestAgentPaths(t *testing.T) {
	require.Equal(t, filepath.Join("root", "bin", "orbit", "orbit"), OrbitLinkPath("root", "darwin"))
	require.Equal(t, filepath.Join("root", "bin", "orbit", "orbit"), OrbitLinkPath("root", "linux"))
	require.Equal(t, filepath.Join("root", "bin", "orbit", "orbit.exe"), OrbitLinkPath("root", "windows"))
	require.Equal(t, filepath.Join("root", "bin", "orbit", "Fleet"), FleetEntryPointPath("root"))
}

func TestChooseFleetPath(t *testing.T) {
	const fleetPath, orbitPath = "/opt/orbit/bin/orbit/macos/stable/Fleet", "/opt/orbit/bin/orbit/macos/stable/orbit"
	grants := func(paths ...string) fdaGrants {
		g := fdaGrants{allowed: map[string]bool{}, denied: map[string]bool{}}
		for _, p := range paths {
			g.allowed[p] = true
		}
		return g
	}
	testCases := []struct {
		name     string
		current  bool
		grants   fdaGrants
		hasFDA   bool
		expected bool
	}{
		{"profile allows both paths", false, grants(orbitPath, fleetPath), true, true},
		{"profile allows only the Fleet path", false, grants(fleetPath), false, true},
		{"profile allows only the orbit path: stay", false, grants(orbitPath), true, false},
		{"profile allows only the orbit path: move back", true, grants(orbitPath), false, false},
		{"no profile and no access to lose", false, grants(), false, true},
		{"no profile and no access, already moved", true, grants(), false, true},
		{"access granted by hand to the orbit path", false, grants(), true, false},
		{"access granted by hand to the Fleet path", true, grants(), true, true},
		{"profile for another channel only", false, grants("/opt/orbit/bin/orbit/macos/edge/orbit"), false, true},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, chooseFleetPath(tt.current, tt.grants, fleetPath, orbitPath, func() bool { return tt.hasFDA }))
		})
	}
}

// launchDaemon has the shape of the LaunchDaemon fleetd's packages install.
const launchDaemon = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>EnvironmentVariables</key>
	<dict>
		<key>ORBIT_FLEET_URL</key>
		<string>https://fleet.example.com</string>
		<key>ORBIT_FLEET_DESKTOP</key>
		<string>true</string>
	</dict>
	<key>KeepAlive</key>
	<true/>
	<key>Label</key>
	<string>com.fleetdm.orbit</string>
	<key>ProgramArguments</key>
	<array>
		<string>/opt/orbit/bin/orbit/orbit</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>StandardErrorPath</key>
	<string>/var/log/orbit/orbit.stderr.log</string>
	<key>ThrottleInterval</key>
	<integer>10</integer>
</dict>
</plist>`

func TestSetLaunchDaemonProgram(t *testing.T) {
	program, err := launchDaemonProgram([]byte(launchDaemon))
	require.NoError(t, err)
	require.Equal(t, "/opt/orbit/bin/orbit/orbit", program)

	out, err := setLaunchDaemonProgram([]byte(launchDaemon), "/opt/orbit/bin/orbit/Fleet")
	require.NoError(t, err)
	program, err = launchDaemonProgram(out)
	require.NoError(t, err)
	require.Equal(t, "/opt/orbit/bin/orbit/Fleet", program)

	// Everything else is unchanged.
	before, after := map[string]any{}, map[string]any{}
	require.NoError(t, plist.Unmarshal([]byte(launchDaemon), &before))
	require.NoError(t, plist.Unmarshal(out, &after))
	before["ProgramArguments"] = []any{"/opt/orbit/bin/orbit/Fleet"}
	require.Equal(t, before, after)

	// A customized LaunchDaemon is left alone.
	for _, daemon := range []string{
		`<plist version="1.0"><dict><key>ProgramArguments</key><array><string>/opt/orbit/bin/orbit/orbit</string><string>--debug</string></array></dict></plist>`,
		`<plist version="1.0"><dict><key>Program</key><string>/opt/orbit/bin/orbit/orbit</string><key>ProgramArguments</key><array><string>orbit</string></array></dict></plist>`,
		`<plist version="1.0"><dict><key>Label</key><string>com.fleetdm.orbit</string></dict></plist>`,
	} {
		_, err := setLaunchDaemonProgram([]byte(daemon), "/opt/orbit/bin/orbit/Fleet")
		require.Error(t, err, daemon)
	}
}

func TestAgentPathReceiver(t *testing.T) {
	capabilities := fleet.CapabilityMap{}
	var checks int
	var moved bool
	var checkErr error
	var restarts []string
	r := &AgentPathReceiver{
		serverCapabilities: func() fleet.CapabilityMap { return capabilities },
		triggerRestart:     func(reason string) { restarts = append(restarts, reason) },
		ensure: func() (bool, error) {
			checks++
			return moved, checkErr
		},
	}

	// A server older than Fleet 5 doesn't advertise the capability: nothing changes.
	require.NoError(t, r.Run(&fleet.OrbitConfig{}))
	require.Zero(t, checks)

	// A failure doesn't fail the config fetch, and the wait before the next check doubles up to a limit.
	capabilities[fleet.CapabilityFleetAgentNames] = struct{}{}
	checkErr = errors.New("profiles failed")
	require.NoError(t, r.Run(&fleet.OrbitConfig{}))
	require.NoError(t, r.Run(&fleet.OrbitConfig{}))
	require.Equal(t, 1, checks)
	require.Empty(t, restarts)
	require.WithinDuration(t, time.Now().Add(2*agentPathCheckInterval), r.nextCheck, time.Minute)
	for range 10 {
		r.nextCheck = time.Time{}
		require.NoError(t, r.Run(&fleet.OrbitConfig{}))
	}
	require.Equal(t, 11, checks)
	require.WithinDuration(t, time.Now().Add(agentPathMaxRetryWait), r.nextCheck, time.Minute)

	// A move restarts orbit, and checks return to the normal interval.
	checkErr, moved = nil, true
	r.nextCheck = time.Time{}
	require.NoError(t, r.Run(&fleet.OrbitConfig{}))
	require.Equal(t, 12, checks)
	require.Len(t, restarts, 1)
	require.Zero(t, r.retryWait)
	require.WithinDuration(t, time.Now().Add(agentPathCheckInterval), r.nextCheck, time.Minute)

	// Nothing to move: no restart.
	moved = false
	r.nextCheck = time.Time{}
	require.NoError(t, r.Run(&fleet.OrbitConfig{}))
	require.Equal(t, 13, checks)
	require.Len(t, restarts, 1)

	// A later failure starts the backoff over.
	checkErr = errors.New("profiles failed")
	r.nextCheck = time.Time{}
	require.NoError(t, r.Run(&fleet.OrbitConfig{}))
	require.Equal(t, 2*agentPathCheckInterval, r.retryWait)
}
