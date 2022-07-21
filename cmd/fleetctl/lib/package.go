package lib

import (
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"

	"github.com/fleetdm/fleet/v4/orbit/pkg/packaging"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"github.com/skratchdot/open-golang/open"
)

type PackageBaseConfig packaging.Options

type PackageConfig struct {
	PackageBaseConfig
	Type              string
	Verbose           bool
	DisableOpenFolder bool
}

//
func PackageAction(config PackageConfig) error {

	if config.FleetURL != "" || config.EnrollSecret != "" {
		if config.FleetURL == "" || config.EnrollSecret == "" {
			return errors.New("--enroll-secret and --fleet-url must be provided together")
		}
	}

	if config.Insecure && config.FleetCertificate != "" {
		return errors.New("--insecure and --fleet-certificate may not be provided together")
	}

	if runtime.GOOS == "windows" && config.Type != "msi" {
		return errors.New("Windows can only build MSI packages.")
	}

	if config.NativeTooling && runtime.GOOS != "linux" {
		return errors.New("native tooling is only available in Linux")
	}

	if config.FleetCertificate != "" {
		err := checkPEMCertificate(config.FleetCertificate)
		if err != nil {
			return fmt.Errorf("failed to read certificate %q: %w", config.FleetCertificate, err)
		}
	}

	var buildFunc func(packaging.Options) (string, error)
	switch config.Type {
	case "pkg":
		buildFunc = packaging.BuildPkg
	case "deb":
		buildFunc = packaging.BuildDeb
	case "rpm":
		buildFunc = packaging.BuildRPM
	case "msi":
		buildFunc = packaging.BuildMSI
	default:
		return errors.New("type must be one of ('pkg', 'deb', 'rpm', 'msi')")
	}

	// disable detailed logging unless verbose is set
	if !config.Verbose {
		zlog.Logger = zerolog.Nop()
	}

	fmt.Println("Generating your osquery installer...")
	path, err := buildFunc(packaging.Options{})
	if err != nil {
		return err
	}
	path, _ = filepath.Abs(path)
	fmt.Printf(`
Success! You generated an osquery installer at %s

To add this device to Fleet, double-click to open your installer.

To add other devices to Fleet, distribute this installer using Chef, Ansible, Jamf, or Puppet. Learn how: https://fleetdm.com/docs/using-fleet/adding-hosts
`, path)
	if !config.DisableOpenFolder {
		open.Start(filepath.Dir(path))
	}
	return nil
}

func checkPEMCertificate(path string) error {
	cert, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	if p, _ := pem.Decode(cert); p == nil {
		return errors.New("invalid PEM file")
	}
	return nil
}
