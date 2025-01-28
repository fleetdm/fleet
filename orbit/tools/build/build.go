//go:build ignore
// +build ignore

package main

// This tool builds Orbit as macOS Universal Binary, codesigns it and notarizes it.
// It currently doesn't support stapling of the binary.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/packaging"
	"github.com/fleetdm/fleet/v4/pkg/buildpkg"
	"github.com/mitchellh/gon/package/zip"
	zlog "github.com/rs/zerolog/log"
)

func main() {
	// Codesigning configuration
	codesignIdentity := os.Getenv("CODESIGN_IDENTITY")

	// Notarization configuration
	acUsername := os.Getenv("AC_USERNAME")
	acPassword := os.Getenv("AC_PASSWORD")
	acTeamID := os.Getenv("AC_TEAM_ID")

	version := os.Getenv("ORBIT_VERSION")
	commit := os.Getenv("ORBIT_COMMIT")
	date := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	binaryPath := os.Getenv("ORBIT_BINARY_PATH")
	if binaryPath == "" {
		binaryPath = "orbit-darwin"
	}

	codesign := false
	if codesignIdentity != "" {
		codesign = true
	} else {
		zlog.Info().Msg("skipping running codesign: CODESIGN_IDENTITY not set")
	}

	notarize := false
	if acUsername != "" && acPassword != "" && acTeamID != "" {
		notarize = true
	} else {
		zlog.Info().Msg("skipping running notarization: AC_USERNAME, AC_PASSWORD, AC_TEAM_ID not all set")
	}

	const (
		amdBinaryPath    = "orbit-darwin-amd64"
		armBinaryPath    = "orbit-darwin-arm64"
		bundleIdentifier = "com.fleetdm.orbit"
	)
	if err := buildOrbit(amdBinaryPath, "amd64", version, commit, date); err != nil {
		panic(err)
	}
	if err := buildOrbit(armBinaryPath, "arm64", version, commit, date); err != nil {
		panic(err)
	}

	if err := buildpkg.MakeMacOSFatExecutable(binaryPath, amdBinaryPath, armBinaryPath); err != nil {
		panic(err)
	}
	if err := os.Remove(amdBinaryPath); err != nil {
		panic(err)
	}
	if err := os.Remove(armBinaryPath); err != nil {
		panic(err)
	}

	if codesign {
		codeSign := exec.Command("codesign", "-s", codesignIdentity, "-i", bundleIdentifier,
			"-f", "-v", "--timestamp", "--options", "runtime", binaryPath,
		)
		zlog.Info().Str("command", codeSign.String()).Msgf("signing %s", binaryPath)

		codeSign.Stderr = os.Stderr
		codeSign.Stdout = os.Stdout
		if err := codeSign.Run(); err != nil {
			panic(err)
		}
	}

	if notarize {
		const notarizationZip = "orbit.zip"
		// NOTE(lucas): The binary needs to be zipped in order to upload to Apple for Notarization.
		if err := zip.Zip(context.Background(), &zip.Options{Files: []string{binaryPath}, OutputPath: notarizationZip}); err != nil {
			panic(err)
		}
		defer os.Remove(notarizationZip)

		if err := packaging.Notarize(notarizationZip, bundleIdentifier); err != nil {
			panic(err)
		}
		// TODO(lucas): packaging.Staple doesn't work on plain binaries.
	}
}

func buildOrbit(binaryPath, arch, version, commit, date string) error {
	/* #nosec G204 -- arguments are actually well defined */
	buildExec := exec.Command("go", "build",
		"-o", binaryPath,
		"-ldflags", fmt.Sprintf("-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=%s -X github.com/fleetdm/fleet/v4/orbit/pkg/build.Commit=%s -X github.com/fleetdm/fleet/v4/orbit/pkg/build.Date=%s", version, commit, date),
		"./"+filepath.Join("orbit", "cmd", "orbit"),
	)
	buildExec.Env = append(os.Environ(), "GOOS=darwin", "GOARCH="+arch)
	buildExec.Stderr = os.Stderr
	buildExec.Stdout = os.Stdout

	zlog.Info().Str("command", buildExec.String()).Str("arch", arch).Msg("build orbit executable")
	if err := buildExec.Run(); err != nil {
		return fmt.Errorf("compile for %s: %w", arch, err)
	}
	return nil
}
