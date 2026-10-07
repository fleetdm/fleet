// Package fleetd_nix_packages implements a table that inventories the Nix store
// paths reachable from the active NixOS system profile and the users' Nix and
// Home Manager profiles.
//
// NixOS has no package database: every package is a /nix/store/<hash>-<name>-<version>
// path, and the store also holds build-time dependencies, old generations and
// generated config. Only paths reachable from an active profile are "installed".
package fleetd_nix_packages

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

const TableName = "fleetd_nix_packages"

// IsNixOS reports whether the host runs NixOS, using the marker file nixos-rebuild
// itself checks.
func IsNixOS() bool {
	_, err := os.Stat("/etc/NIXOS")
	return err == nil
}

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("name"),
		table.TextColumn("version"),
		// output is the derivation output (e.g. "bin", "dev", "man"), empty for the default one.
		table.TextColumn("output"),
		table.TextColumn("store_path"),
		// direct is 1 for packages listed in a profile (e.g. environment.systemPackages)
		// and 0 for their runtime dependencies.
		table.IntegerColumn("direct"),
		// profiles lists the profiles that directly include the package, e.g. "system,user:alice".
		table.TextColumn("profiles"),
	}
}

// Generate is called to return the results for the table at query time.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	return defaultEnv().generate(ctx, directOnly(queryContext))
}

// directOnly reports whether the query only asks for direct packages, which skips
// computing the (much larger) runtime closure.
func directOnly(queryContext table.QueryContext) bool {
	c, ok := queryContext.Constraints["direct"]
	if !ok {
		return false
	}
	for _, cons := range c.Constraints {
		if cons.Operator == table.OperatorEquals && cons.Expression == "1" {
			return true
		}
	}
	return false
}

type env struct {
	storeDir           string
	systemProfile      string
	etcProfilesDir     string
	perUserProfilesDir string
	passwdFile         string
	nixStoreBin        string
	// query runs nix-store --query with the given flag over the given store paths.
	query func(ctx context.Context, bin, flag string, paths ...string) ([]string, error)
}

func defaultEnv() env {
	return env{
		storeDir:           "/nix/store",
		systemProfile:      "/run/current-system",
		etcProfilesDir:     "/etc/profiles/per-user",
		perUserProfilesDir: "/nix/var/nix/profiles/per-user",
		passwdFile:         "/etc/passwd",
		// Only the root-owned NixOS system profile: fleetd runs nix-store as root, and
		// single-user Nix installs on other distributions make /nix user-writable.
		nixStoreBin: "/run/current-system/sw/bin/nix-store",
		query:       nixStoreQuery,
	}
}

func nixStoreQuery(ctx context.Context, bin, flag string, paths ...string) ([]string, error) {
	args := append([]string{"--query", flag}, paths...)
	out, err := exec.CommandContext(ctx, bin, args...).Output()
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", bin, strings.Join(args, " "), err)
	}
	return strings.Fields(string(out)), nil
}

// profile is an active profile whose packages are inventoried.
type profile struct {
	label string
	// closureRoot is the store path whose runtime closure is inventoried.
	closureRoot string
	// packagesEnv is the buildEnv store path whose references are the packages
	// directly installed in the profile.
	packagesEnv string
}

func (e env) generate(ctx context.Context, directOnly bool) ([]map[string]string, error) {
	bin := e.nixStoreBin
	if _, err := os.Stat(bin); err != nil {
		return nil, nil
	}

	profiles := e.findProfiles()
	if len(profiles) == 0 {
		return nil, nil
	}

	directLabels := make(map[string][]string)
	var closureRoots []string
	for _, p := range profiles {
		refs, err := e.query(ctx, bin, "--references", p.packagesEnv)
		if err != nil {
			log.Debug().Err(err).Str("profile", p.label).Msg("fleetd_nix_packages: failed to query profile packages")
			continue
		}
		for _, ref := range refs {
			if !slices.Contains(directLabels[ref], p.label) {
				directLabels[ref] = append(directLabels[ref], p.label)
			}
		}
		if !slices.Contains(closureRoots, p.closureRoot) {
			closureRoots = append(closureRoots, p.closureRoot)
		}
	}

	var closure []string
	if !directOnly && len(closureRoots) > 0 {
		var err error
		closure, err = e.query(ctx, bin, "--requisites", closureRoots...)
		if err != nil {
			log.Debug().Err(err).Msg("fleetd_nix_packages: failed to query profile closures")
		}
	}

	skip := make(map[string]bool)
	for _, p := range profiles {
		skip[p.closureRoot] = true
		skip[p.packagesEnv] = true
	}
	return buildRows(directLabels, closure, skip), nil
}

func (e env) findProfiles() []profile {
	var profiles []profile
	add := func(label, closureLink, packagesLink string) {
		closureRoot, ok := e.resolveStorePath(closureLink)
		if !ok {
			return
		}
		packagesEnv, ok := e.resolveStorePath(packagesLink)
		if !ok {
			return
		}
		for _, p := range profiles {
			if p.label == label && p.packagesEnv == packagesEnv {
				return
			}
		}
		profiles = append(profiles, profile{label: label, closureRoot: closureRoot, packagesEnv: packagesEnv})
	}

	add("system", e.systemProfile, filepath.Join(e.systemProfile, "sw"))

	users, err := e.users()
	if err != nil {
		log.Debug().Err(err).Msg("fleetd_nix_packages: failed to list users")
	}
	for _, u := range users {
		userLabel := "user:" + u.name
		for _, link := range []string{
			filepath.Join(e.etcProfilesDir, u.name),
			filepath.Join(u.home, ".nix-profile"),
			filepath.Join(u.home, ".local/state/nix/profiles/profile"),
			filepath.Join(e.perUserProfilesDir, u.name, "profile"),
		} {
			add(userLabel, link, link)
		}

		hmLabel := "home-manager:" + u.name
		for _, gen := range []string{
			filepath.Join(u.home, ".local/state/home-manager/gcroots/current-home"),
			filepath.Join(e.perUserProfilesDir, u.name, "home-manager"),
		} {
			add(hmLabel, gen, filepath.Join(gen, "home-path"))
		}
	}
	return profiles
}

// resolveStorePath resolves a profile symlink to its top-level store path. Links
// live in user-writable home directories, so anything outside the store is ignored.
func (e env) resolveStorePath(link string) (string, bool) {
	resolved, err := filepath.EvalSymlinks(link)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(e.storeDir, resolved)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") || strings.Contains(rel, string(filepath.Separator)) {
		return "", false
	}
	return resolved, true
}

type user struct {
	name string
	home string
}

func (e env) users() ([]user, error) {
	f, err := os.Open(e.passwdFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parsePasswd(f)
}

// parsePasswd returns root and the non-system accounts from /etc/passwd content.
func parsePasswd(r io.Reader) ([]user, error) {
	var users []user
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		// username:password:uid:gid:gecos:home:shell
		fields := strings.SplitN(scanner.Text(), ":", 7)
		if len(fields) < 7 {
			continue
		}
		uid, err := strconv.Atoi(fields[2])
		if err != nil || (uid != 0 && uid < 1000) {
			continue
		}
		home := fields[5]
		if home == "" || home == "/" || home == "/var/empty" || home == "/dev/null" {
			continue
		}
		users = append(users, user{name: fields[0], home: home})
	}
	return users, scanner.Err()
}

func buildRows(directLabels map[string][]string, closure []string, skip map[string]bool) []map[string]string {
	paths := make([]string, 0, len(directLabels)+len(closure))
	for p := range directLabels {
		paths = append(paths, p)
	}
	paths = append(paths, closure...)
	slices.Sort(paths)
	paths = slices.Compact(paths)

	rows := make([]map[string]string, 0, len(paths))
	for _, p := range paths {
		if skip[p] {
			continue
		}
		name, version, output, ok := parseStorePath(p)
		if !ok {
			continue
		}
		labels := directLabels[p]
		slices.Sort(labels)
		direct := "0"
		if len(labels) > 0 {
			direct = "1"
		}
		rows = append(rows, map[string]string{
			"name":       name,
			"version":    version,
			"output":     output,
			"store_path": p,
			"direct":     direct,
			"profiles":   strings.Join(labels, ","),
		})
	}
	return rows
}

// storeHashLen is the length of the base32 hash prefix of a store path's basename.
const storeHashLen = 32

// knownOutputs are the derivation output names that Nix appends to a multi-output
// package's store path (e.g. openssl-3.0.14-bin).
var knownOutputs = []string{"bin", "dev", "lib", "out", "man", "doc", "devdoc", "info", "debug", "static", "dist", "modules", "python"}

// nonPackageSuffixes are store paths for files rather than packages, which would
// otherwise parse with the file extension as part of the version.
var nonPackageSuffixes = []string{".drv", ".patch", ".diff", ".tar.gz", ".tar.xz", ".tar.bz2", ".tar.zst", ".tgz", ".zip"}

// parseStorePath splits a store path into package name, version and output. It
// returns false for paths without a version, which in a NixOS closure are mostly
// generated config (unit files, /etc entries, wrappers) rather than software.
func parseStorePath(storePath string) (name, version, output string, ok bool) {
	base := filepath.Base(storePath)
	if len(base) <= storeHashLen+1 || base[storeHashLen] != '-' {
		return "", "", "", false
	}
	drvName := base[storeHashLen+1:]
	for _, suffix := range nonPackageSuffixes {
		if strings.HasSuffix(drvName, suffix) {
			return "", "", "", false
		}
	}

	name, version = parseDrvName(drvName)
	if version == "" || version[0] < '0' || version[0] > '9' {
		return "", "", "", false
	}
	for _, o := range knownOutputs {
		if v, found := strings.CutSuffix(version, "-"+o); found {
			version, output = v, o
			break
		}
	}
	return name, version, output, true
}

// parseDrvName follows Nix's builtins.parseDrvName: the version starts after the
// first dash that is not followed by a letter.
func parseDrvName(s string) (name, version string) {
	for i := 0; i+1 < len(s); i++ {
		if s[i] == '-' && !isLetter(s[i+1]) {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
