// Package wix runs the WiX packaging tools via Docker.
//
// WiX's documentation is available at https://wixtoolset.org/.
package wix

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const (
	directoryReference = "ORBITROOT"
	imageName          = "fleetdm/wix:latest"
	dockerPlatform     = "linux/amd64"
	Wine64Cmd          = "wine64"
	WineCmd            = "wine"
)

// Heat runs the WiX Heat command on the provided directory.
//
// The Heat command creates XML fragments allowing WiX to include the entire
// directory. See
// https://wixtoolset.org/documentation/manual/v3/overview/heat.html.
func Heat(path string, native bool, localWixDir string) error {
	var args []string

	if !native && localWixDir == "" {
		args = append(
			args,
			"docker", "run", "--rm", "--platform", dockerPlatform,
			"--volume", path+":/wix", // mount volume
			imageName, // image name
		)
	}

	heatPath := `heat`
	if localWixDir != "" {
		heatPath = filepath.Join(localWixDir, `heat.exe`)
		if runtime.GOOS == "darwin" {
			wineExec, err := darwinWineExecutable()
			if err != nil {
				return fmt.Errorf("determining wine executable: %w", err)
			}
			args = append(args, wineExec)
		}
	}

	args = append(args,
		heatPath, "dir", "root", // command
		"-out", "heat.wxs",
		"-gg", "-g1", // generate UUIDs (required by wix)
		"-cg", "OrbitFiles", // set ComponentGroup name
		"-scom", "-sfrag", "-srd", "-sreg", // suppress unneccesary generated items
		"-dr", directoryReference, // set reference name
		"-ke", // keep empty directories
	)

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr

	if args[0] == WineCmd || args[0] == Wine64Cmd {
		cmd.Env = append(os.Environ(), "WINEDEBUG=-all")
	}

	if native || localWixDir != "" {
		cmd.Dir = path
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("heat failed: %w", err)
	}

	return nil
}

func darwinWineExecutable() (string, error) {
	cmdOut, err := exec.Command("wine", "--version").Output()
	if err != nil {
		return "", fmt.Errorf("running wine to get version information: %w", err)
	}
	wineVerStr, found := strings.CutPrefix(string(cmdOut), "wine-")
	if !found {
		return "", fmt.Errorf("Unknown wine version: %q. Is Wine installed? Creating a fleetd agent for Windows (.msi) requires Wine. To install Wine see the script here: https://fleetdm.com/install-wine ", string(cmdOut))
	}
	wineVersion, err := strconv.ParseInt(strings.Split(wineVerStr, ".")[0], 10, 64)
	if err != nil {
		return "", fmt.Errorf("Unable to parse wine version: %q. Is Wine installed? Creating a fleetd agent for Windows (.msi) requires Wine. To install Wine see the script here: https://fleetdm.com/install-wine ", wineVerStr)
	}
	if wineVersion < 10 {
		return Wine64Cmd, nil
	}
	return WineCmd, nil
}

// Candle runs the WiX Candle command on the provided directory.
//
// See
// https://wixtoolset.org/documentation/manual/v3/overview/candle.html.
func Candle(path string, native bool, localWixDir string, arch string) error {
	var args []string

	if !native && localWixDir == "" {
		args = append(
			args,
			"docker", "run", "--rm", "--platform", dockerPlatform,
			"--volume", path+":/wix", // mount volume
			imageName, // image name
		)
	}

	candlePath := `candle`
	if localWixDir != "" {
		candlePath = filepath.Join(localWixDir, `candle.exe`)
		if runtime.GOOS == "darwin" {
			wineExec, err := darwinWineExecutable()
			if err != nil {
				return fmt.Errorf("determining wine executable: %w", err)
			}
			args = append(args, wineExec)
		}
	}

	wixArch := "x64"
	if arch == "arm64" {
		wixArch = "arm64"
	}

	args = append(args,
		candlePath, "heat.wxs", "main.wxs", // command
		"-ext", "WixUtilExtension",
		"-arch", wixArch,
	)

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr

	if args[0] == WineCmd || args[0] == Wine64Cmd {
		cmd.Env = append(os.Environ(), "WINEDEBUG=-all")
	}

	if native || localWixDir != "" {
		cmd.Dir = path
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("candle failed: %w", err)
	}

	return nil
}

// Light runs the WiX Light command on the provided directory.
//
// See
// https://wixtoolset.org/documentation/manual/v3/overview/light.html.
func Light(path string, native bool, localWixDir string) error {
	var args []string

	if !native && localWixDir == "" {
		args = append(
			args,
			"docker", "run", "--rm", "--platform", dockerPlatform,
			"--volume", path+":/wix", // mount volume
			imageName, // image name
		)
	}

	lightPath := `light`
	if localWixDir != "" {
		lightPath = filepath.Join(localWixDir, `light.exe`)
		if runtime.GOOS == "darwin" {
			wineExec, err := darwinWineExecutable()
			if err != nil {
				return fmt.Errorf("determining wine executable: %w", err)
			}
			args = append(args, wineExec)
		}
	}
	args = append(args,
		lightPath, "heat.wixobj", "main.wixobj", // command
		"-ext", "WixUtilExtension",
		"-b", "root", // Set directory for finding heat files
		"-out", "orbit.msi",
		"-sval", // skip validation (otherwise Wine crashes)
	)

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr

	if args[0] == WineCmd || args[0] == Wine64Cmd {
		cmd.Env = append(os.Environ(), "WINEDEBUG=-all")
	}

	if native || localWixDir != "" {
		cmd.Dir = path
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("light failed: %w", err)
	}

	return nil
}
