//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/orbit/pkg/packaging"
	"github.com/fleetdm/fleet/v4/pkg/buildpkg"
	zlog "github.com/rs/zerolog/log"
)

func main() {
	// Codesigning configuration
	codesignIdentity := os.Getenv("CODESIGN_IDENTITY")

	// Notarization configuration
	acUsername := os.Getenv("AC_USERNAME")
	acPassword := os.Getenv("AC_PASSWORD")
	acTeamID := os.Getenv("AC_TEAM_ID")

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
		binaryPath       = "orbit-darwin"
		bundleIdentifier = "com.fleetdm.orbit"
	)
	if err := buildOrbit(amdBinaryPath, "amd64"); err != nil {
		panic(err)
	}
	if err := buildOrbit(armBinaryPath, "arm64"); err != nil {
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
		if err := packaging.Notarize(binaryPath, bundleIdentifier); err != nil {
			panic(err)
		}
		if err := packaging.Staple(binaryPath); err != nil {
			panic(err)
		}
	}
}

func buildOrbit(binaryPath, arch string) error {
	/* #nosec G204 -- arguments are actually well defined */
	buildExec := exec.Command("go", "build",
		"-o", binaryPath,
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
