package fleetd_nix_packages

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
)

const hash = "0123456789abcdfghijklmnpqrsvwxyz"

func storePath(name string) string {
	return "/nix/store/" + hash + "-" + name
}

func TestParseStorePath(t *testing.T) {
	cases := []struct {
		name, drvName                 string
		wantName, wantVer, wantOutput string
		wantOK                        bool
	}{
		{name: "simple", drvName: "hello-2.12.1", wantName: "hello", wantVer: "2.12.1", wantOK: true},
		{name: "dashed name", drvName: "xorg-server-21.1.13", wantName: "xorg-server", wantVer: "21.1.13", wantOK: true},
		{name: "output", drvName: "openssl-3.0.14-bin", wantName: "openssl", wantVer: "3.0.14", wantOutput: "bin", wantOK: true},
		{name: "lib output", drvName: "gcc-13.2.0-lib", wantName: "gcc", wantVer: "13.2.0", wantOutput: "lib", wantOK: true},
		{name: "unwrapped", drvName: "firefox-unwrapped-130.0", wantName: "firefox-unwrapped", wantVer: "130.0", wantOK: true},
		{name: "python package", drvName: "python3.12-requests-2.31.0", wantName: "python3.12-requests", wantVer: "2.31.0", wantOK: true},
		{name: "digit name", drvName: "7zz-23.01", wantName: "7zz", wantVer: "23.01", wantOK: true},
		{name: "kernel", drvName: "linux-6.6.40", wantName: "linux", wantVer: "6.6.40", wantOK: true},
		{name: "prerelease suffix", drvName: "nix-2.24.0pre20240801_abcdef", wantName: "nix", wantVer: "2.24.0pre20240801_abcdef", wantOK: true},
		{name: "no version", drvName: "system-path", wantOK: false},
		{name: "unit file", drvName: "unit-sshd.service", wantOK: false},
		{name: "etc entry", drvName: "etc-os-release", wantOK: false},
		{name: "drv", drvName: "hello-2.12.1.drv", wantOK: false},
		{name: "tarball", drvName: "openssl-3.0.14.tar.gz", wantOK: false},
		{name: "patch", drvName: "fix-build-1.patch", wantOK: false},
		{name: "non-digit version", drvName: "foo-_bar", wantOK: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			name, ver, output, ok := parseStorePath(storePath(c.drvName))
			require.Equal(t, c.wantOK, ok)
			if !c.wantOK {
				return
			}
			require.Equal(t, c.wantName, name)
			require.Equal(t, c.wantVer, ver)
			require.Equal(t, c.wantOutput, output)
		})
	}

	_, _, _, ok := parseStorePath("/nix/store/tooshort-hello-1.0")
	require.False(t, ok)
}

func TestParsePasswd(t *testing.T) {
	accounts := `root:x:0:0:System administrator:/root:/run/current-system/sw/bin/bash
messagebus:x:4:4:D-Bus system message bus daemon user:/run/dbus:/run/current-system/sw/bin/nologin
nobody:x:65534:65534:Unprivileged account:/var/empty:/run/current-system/sw/bin/nologin
alice:x:1000:100::/home/alice:/run/current-system/sw/bin/zsh
malformed
bob:x:1001:100::/home/bob:/run/current-system/sw/bin/bash
`
	users, err := parsePasswd(strings.NewReader(accounts))
	require.NoError(t, err)
	require.Equal(t, []user{
		{name: "root", home: "/root"},
		{name: "alice", home: "/home/alice"},
		{name: "bob", home: "/home/bob"},
	}, users)
}

func TestDirectOnly(t *testing.T) {
	require.False(t, directOnly(table.QueryContext{}))
	require.True(t, directOnly(table.QueryContext{Constraints: map[string]table.ConstraintList{
		"direct": {Constraints: []table.Constraint{{Operator: table.OperatorEquals, Expression: "1"}}},
	}}))
	require.False(t, directOnly(table.QueryContext{Constraints: map[string]table.ConstraintList{
		"direct": {Constraints: []table.Constraint{{Operator: table.OperatorEquals, Expression: "0"}}},
	}}))
}

// fakeNix lays out a fake store and profile symlinks under a temp dir and answers
// nix-store queries from a fixed references/requisites graph.
type fakeNix struct {
	root       string
	references map[string][]string
	requisites map[string][]string
	queries    []string
}

func newFakeNix(t *testing.T) *fakeNix {
	// Resolved because profile links are resolved before being checked against the store dir.
	root, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "nix/store"), 0o755))
	return &fakeNix{root: root, references: map[string][]string{}, requisites: map[string][]string{}}
}

func (f *fakeNix) store(name string) string {
	p := filepath.Join(f.root, "nix/store", hash+"-"+name)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		if err := os.MkdirAll(p, 0o755); err != nil {
			panic(err)
		}
	}
	return p
}

func (f *fakeNix) link(t *testing.T, link, target string) {
	link = filepath.Join(f.root, link)
	require.NoError(t, os.MkdirAll(filepath.Dir(link), 0o755))
	require.NoError(t, os.Symlink(target, link))
}

func (f *fakeNix) env(t *testing.T, accounts string) env {
	passwdFile := filepath.Join(f.root, "etc/passwd")
	require.NoError(t, os.MkdirAll(filepath.Dir(passwdFile), 0o755))
	require.NoError(t, os.WriteFile(passwdFile, []byte(accounts), 0o644))
	bin := filepath.Join(f.root, "run/current-system/sw/bin/nix-store")
	return env{
		storeDir:           filepath.Join(f.root, "nix/store"),
		systemProfile:      filepath.Join(f.root, "run/current-system"),
		etcProfilesDir:     filepath.Join(f.root, "etc/profiles/per-user"),
		perUserProfilesDir: filepath.Join(f.root, "nix/var/nix/profiles/per-user"),
		passwdFile:         passwdFile,
		nixStoreBin:        bin,
		query: func(_ context.Context, _ string, flag string, paths ...string) ([]string, error) {
			f.queries = append(f.queries, flag)
			var out []string
			for _, p := range paths {
				switch flag {
				case "--references":
					out = append(out, f.references[p]...)
				case "--requisites":
					out = append(out, f.requisites[p]...)
				}
			}
			return out, nil
		},
	}
}

func TestGenerate(t *testing.T) {
	f := newFakeNix(t)

	toplevel := f.store("nixos-system-host-24.11.20241001.abcdef")
	systemPath := f.store("system-path")
	curl := f.store("curl-8.9.1-bin")
	curlLib := f.store("curl-8.9.1")
	openssl := f.store("openssl-3.0.14")
	sshdUnit := f.store("unit-sshd.service")
	f.link(t, "run/current-system", toplevel)
	require.NoError(t, os.Symlink(systemPath, filepath.Join(toplevel, "sw")))
	// Pretend nix-store is installed in the system profile.
	require.NoError(t, os.MkdirAll(filepath.Join(systemPath, "bin"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(systemPath, "bin/nix-store"), nil, 0o600))

	// alice installs ripgrep with nix profile, reachable through two links.
	aliceProfile := f.store("profile")
	ripgrep := f.store("ripgrep-14.1.0")
	f.link(t, "home/alice/.nix-profile", aliceProfile)
	f.link(t, "nix/var/nix/profiles/per-user/alice/profile", aliceProfile)

	// alice also uses standalone Home Manager, which installs curl too.
	hmGen := f.store("home-manager-generation")
	hmPath := f.store("home-manager-path")
	f.link(t, "home/alice/.local/state/home-manager/gcroots/current-home", hmGen)
	require.NoError(t, os.Symlink(hmPath, filepath.Join(hmGen, "home-path")))

	// bob's profile link points outside the store and must be ignored.
	outside := filepath.Join(f.root, "tmp/evil")
	require.NoError(t, os.MkdirAll(outside, 0o755))
	f.link(t, "home/bob/.nix-profile", outside)

	f.references[systemPath] = []string{curl, openssl}
	f.references[aliceProfile] = []string{ripgrep}
	f.references[hmPath] = []string{curl}
	f.requisites[toplevel] = []string{toplevel, systemPath, curl, curlLib, openssl, sshdUnit}
	f.requisites[aliceProfile] = []string{aliceProfile, ripgrep}
	f.requisites[hmGen] = []string{hmGen, hmPath, curl, curlLib}

	accounts := strings.Join([]string{
		"alice:x:1000:100::" + filepath.Join(f.root, "home/alice") + ":/bin/sh",
		"bob:x:1001:100::" + filepath.Join(f.root, "home/bob") + ":/bin/sh",
	}, "\n")
	e := f.env(t, accounts)

	rows, err := e.generate(t.Context(), false)
	require.NoError(t, err)
	require.Equal(t, []map[string]string{
		{"name": "curl", "version": "8.9.1", "output": "", "store_path": curlLib, "direct": "0", "profiles": ""},
		{"name": "curl", "version": "8.9.1", "output": "bin", "store_path": curl, "direct": "1", "profiles": "home-manager:alice,system"},
		{"name": "openssl", "version": "3.0.14", "output": "", "store_path": openssl, "direct": "1", "profiles": "system"},
		{"name": "ripgrep", "version": "14.1.0", "output": "", "store_path": ripgrep, "direct": "1", "profiles": "user:alice"},
	}, rows)
	require.Equal(t, []string{"--references", "--references", "--references", "--requisites"}, f.queries)

	f.queries = nil
	rows, err = e.generate(t.Context(), true)
	require.NoError(t, err)
	require.Len(t, rows, 3)
	for _, row := range rows {
		require.Equal(t, "1", row["direct"])
	}
	require.NotContains(t, f.queries, "--requisites")
}

func TestGenerateNoNix(t *testing.T) {
	f := newFakeNix(t)
	e := f.env(t, "")
	rows, err := e.generate(t.Context(), false)
	require.NoError(t, err)
	require.Empty(t, rows)
	require.Empty(t, f.queries)
}
