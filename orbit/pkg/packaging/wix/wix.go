// Package wix runs the WiX packaging tools via Docker.
//
// WiX's documentation is available at https://wixtoolset.org/.
package wix

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	directoryReference   = "ORBITROOT"
	linuxPathSeparator   = "/"
	windowsPathSeparator = "\\"
)

// windowsJoin returns the result of replacing each slash ('/') character in
// each path with a Windows separator character ('\') and joining them using
// the Windows separator character.
//
// We can't use filepath.FromSlash because this func is run in a *nix
// machine.
func windowsJoin(paths ...string) string {
	s := strings.Join(paths, windowsPathSeparator)
	return strings.ReplaceAll(s, linuxPathSeparator, windowsPathSeparator)
}

// Heat runs the WiX Heat command on the provided directory.
//
// The Heat command creates XML fragments allowing WiX to include the entire
// directory. See
// https://wixtoolset.org/documentation/manual/v3/overview/heat.html.
func Heat(path string, native bool) error {
	var args []string

	if !native {
		args = append(
			args,
			"docker", "run", "--rm", "--platform", "linux/amd64",
			"--volume", path+":"+path, // mount volume
			"fleetdm/wix:latest", // image name
		)
	}

	args = append(args,
		"heat", "dir", windowsJoin(path, "root"), // command
		"-out", windowsJoin(path, "heat.wxs"),
		"-gg", "-g1", // generate UUIDs (required by wix)
		"-cg", "OrbitFiles", // set ComponentGroup name
		"-scom", "-sfrag", "-srd", "-sreg", // suppress unneccesary generated items
		"-dr", directoryReference, // set reference name
		"-ke", // keep empty directories
	)

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("heat failed: %w", err)
	}

	return nil
}

// Candle runs the WiX Candle command on the provided directory.
//
// See
// https://wixtoolset.org/documentation/manual/v3/overview/candle.html.
func Candle(path string, native bool) error {
	var args []string

	if !native {
		args = append(
			args,
			"docker", "run", "--rm", "--platform", "linux/amd64",
			"--volume", path+":"+path, // mount volume
			"fleetdm/wix:latest", // image name
		)
	}

	args = append(args,
		"candle", windowsJoin(path, "heat.wxs"), windowsJoin(path, "main.wxs"), // command
		"-out", windowsJoin(path, ""),
		"-ext", "WixUtilExtension",
		"-arch", "x64",
	)

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("candle failed: %w", err)
	}

	return nil
}

// Light runs the WiX Light command on the provided directory.
//
// See
// https://wixtoolset.org/documentation/manual/v3/overview/light.html.
func Light(path string, native bool) error {
	var args []string

	if !native {
		args = append(
			args,
			"docker", "run", "--rm", "--platform", "linux/amd64",
			"--volume", path+":"+path, // mount volume
			"fleetdm/wix:latest", // image name
		)
	}

	args = append(args,
		"light", windowsJoin(path, "heat.wixobj"), windowsJoin(path, "main.wixobj"), // command
		"-ext", "WixUtilExtension",
		"-b", windowsJoin(path, "root"), // Set directory for finding heat files
		"-out", windowsJoin(path, "orbit.msi"),
		"-sval", // skip validation (otherwise Wine crashes)
	)

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("light failed: %w", err)
	}

	return nil
}
