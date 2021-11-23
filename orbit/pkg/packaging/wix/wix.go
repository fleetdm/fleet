// Package wix runs the WiX packaging tools via Docker.
//
// WiX's documentation is available at https://wixtoolset.org/.
package wix

import (
	"fmt"
	"os"
	"os/exec"
)

const (
	directoryReference = "ORBITROOT"
)

// Heat runs the WiX Heat command on the provided directory.
//
// The Heat command creates XML fragments allowing WiX to include the entire
// directory. See
// https://wixtoolset.org/documentation/manual/v3/overview/heat.html.
func Heat(path string) error {
	cmd := exec.Command(
		"docker", "run", "--rm", "--platform", "linux/amd64",
		"--volume", path+":/wix", // mount volume
		"fleetdm/wix:latest",  // image name
		"heat", "dir", "root", // command in image
		"-out", "heat.wxs",
		"-gg", "-g1", // generate UUIDs (required by wix)
		"-cg", "OrbitFiles", // set ComponentGroup name
		"-scom", "-sfrag", "-srd", "-sreg", // suppress unneccesary generated items
		"-dr", directoryReference, // set reference name
		"-ke", // keep empty directories
	)
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
func Candle(path string) error {
	cmd := exec.Command(
		"docker", "run", "--rm", "--platform", "linux/amd64",
		"--volume", path+":/wix", // mount volume
		"fleetdm/wix:latest",             // image name
		"candle", "heat.wxs", "main.wxs", // command in image
		"-ext", "WixUtilExtension",
		"-arch", "x64",
	)
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
func Light(path string) error {
	cmd := exec.Command(
		"docker", "run", "--rm", "--platform", "linux/amd64",
		"--volume", path+":/wix", // mount volume
		"fleetdm/wix:latest",                  // image name
		"light", "heat.wixobj", "main.wixobj", // command in image
		"-ext", "WixUtilExtension",
		"-b", "root", // Set directory for finding heat files
		"-out", "orbit.msi",
		"-sval", // skip validation (otherwise Wine crashes)
	)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("light failed: %w", err)
	}

	return nil
}
