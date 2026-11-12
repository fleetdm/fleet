//go:build darwin

package update

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnsureSymlink(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "Fleet")

	// A fresh install has no entry point; a regular file there is replaced too.
	for _, setup := range []func(){
		func() {},
		func() { require.NoError(t, os.WriteFile(link, []byte("x"), 0o600)) },
		func() { require.NoError(t, os.Symlink(filepath.Join(dir, "elsewhere"), link)) },
	} {
		_ = os.Remove(link)
		setup()
		require.NoError(t, ensureSymlink(link, filepath.Join(dir, "orbit")))
		target, err := os.Readlink(link)
		require.NoError(t, err)
		require.Equal(t, filepath.Join(dir, "orbit"), target)
	}
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1) // no temporary file left
}

func TestWriteFileAtomically(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "com.fleetdm.orbit.plist")
	require.NoError(t, os.WriteFile(path, []byte("old"), 0o600))

	require.NoError(t, writeFileAtomically(path, []byte("new"), 0o644))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "new", string(data))
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), info.Mode().Perm())
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
}

func TestMoveAgentPath(t *testing.T) {
	fdaProfiles := func(paths ...string) []byte {
		entries := ""
		for _, p := range paths {
			entries += `<dict><key>Allowed</key><true/><key>Identifier</key><string>` + p +
				`</string><key>IdentifierType</key><string>path</string></dict>`
		}
		return []byte(`<plist version="1.0"><dict><key>_computerlevel</key><array><dict><key>ProfileItems</key><array><dict>
			<key>PayloadType</key><string>com.apple.TCC.configuration-profile-policy</string>
			<key>PayloadContent</key><dict><key>Services</key><dict><key>SystemPolicyAllFiles</key><array>` + entries +
			`</array></dict></dict></dict></array></dict></array></dict></plist>`)
	}
	const orbitFile, fleetFile, otherProgram = "orbit", "Fleet", "/usr/local/bin/orbit"

	testCases := []struct {
		name         string
		running      string // file name in the channel directory, or otherProgram for an executable elsewhere
		program      string // what the LaunchDaemon starts: "link", "entry" or otherProgram
		allowed      []string
		profilesErr  error
		checkErr     error
		wantMoved    bool
		wantErr      bool
		wantProgram  string
		wantCopiedTo string // file name the running executable was copied to
	}{
		{name: "moves when the profile allows the Fleet path", running: orbitFile, program: "link",
			allowed: []string{orbitFile, fleetFile}, wantMoved: true, wantProgram: "entry", wantCopiedTo: fleetFile},
		{name: "moves when already running the Fleet file", running: fleetFile, program: "link",
			allowed: []string{fleetFile}, wantMoved: true, wantProgram: "entry", wantCopiedTo: fleetFile},
		{name: "moves back when the profile only allows the orbit path", running: fleetFile, program: "entry",
			allowed: []string{orbitFile}, wantMoved: true, wantProgram: "link", wantCopiedTo: orbitFile},
		{name: "stays when the profile only allows the orbit path", running: orbitFile, program: "link",
			allowed: []string{orbitFile}, wantProgram: "link"},
		{name: "a failed check leaves the LaunchDaemon alone", running: orbitFile, program: "link",
			allowed: []string{orbitFile, fleetFile}, checkErr: errors.New("killed"), wantErr: true, wantProgram: "link"},
		{name: "a profiles error changes nothing", running: orbitFile, program: "link",
			profilesErr: errors.New("timeout"), wantErr: true, wantProgram: "link"},
		{name: "a customized LaunchDaemon is left alone", running: orbitFile, program: otherProgram,
			allowed: []string{fleetFile}, wantErr: true, wantProgram: otherProgram},
		{name: "an executable outside the updater's directory is left alone", running: otherProgram, program: "link",
			allowed: []string{fleetFile}, wantProgram: "link"},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			channelDir := filepath.Join(root, "bin", "orbit", "macos", "stable")
			require.NoError(t, os.MkdirAll(channelDir, 0o755))
			for _, name := range []string{orbitFile, fleetFile} {
				// Distinct contents show which file was copied where.
				require.NoError(t, os.WriteFile(filepath.Join(channelDir, name), []byte("old "+name), 0o755))
			}
			exe := filepath.Join(channelDir, tt.running)
			if tt.running == otherProgram {
				exe = filepath.Join(root, "dev", "orbit")
			}
			running, _ := os.ReadFile(exe) // nil for an executable elsewhere
			linkPath, entryPoint := OrbitLinkPath(root, "darwin"), FleetEntryPointPath(root)
			require.NoError(t, os.Symlink(exe, linkPath))
			programs := map[string]string{"link": linkPath, "entry": entryPoint, otherProgram: otherProgram}
			daemon, err := setLaunchDaemonProgram([]byte(launchDaemon), programs[tt.program])
			require.NoError(t, err)
			daemonPath := filepath.Join(root, "com.fleetdm.orbit.plist")
			require.NoError(t, os.WriteFile(daemonPath, daemon, 0o644))

			var allowed []string
			for _, name := range tt.allowed {
				allowed = append(allowed, filepath.Join(channelDir, name))
			}
			var checked []string
			moved, err := moveAgentPath(agentPathEnv{
				rootDir:          root,
				exe:              exe,
				launchDaemonPath: daemonPath,
				profiles: func() ([]byte, error) {
					return fdaProfiles(allowed...), tt.profilesErr
				},
				hasFullDiskAccess: func() bool { return true },
				checkExecutable: func(path string) error {
					checked = append(checked, path)
					return tt.checkErr
				},
			})
			require.Equal(t, tt.wantErr, err != nil, err)
			require.Equal(t, tt.wantMoved, moved)

			data, err := os.ReadFile(daemonPath)
			require.NoError(t, err)
			program, err := launchDaemonProgram(data)
			require.NoError(t, err)
			require.Equal(t, programs[tt.wantProgram], program)

			// bin/orbit/orbit is the update runner's to change.
			target, err := os.Readlink(linkPath)
			require.NoError(t, err)
			require.Equal(t, exe, target)

			if tt.wantCopiedTo != "" {
				copied, err := os.ReadFile(filepath.Join(channelDir, tt.wantCopiedTo))
				require.NoError(t, err)
				require.Equal(t, running, copied)
				require.Equal(t, []string{filepath.Join(channelDir, tt.wantCopiedTo)}, checked)
				entryTarget, err := os.Readlink(entryPoint)
				require.NoError(t, err)
				require.Equal(t, linkPath, entryTarget)
			}
			if !tt.wantMoved && tt.checkErr == nil {
				require.Empty(t, checked)
			}
		})
	}
}
