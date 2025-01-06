package main

import (
	"crypto/tls"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	eefleetctl "github.com/fleetdm/fleet/v4/ee/fleetctl"
	"github.com/fleetdm/fleet/v4/orbit/pkg/packaging"
	"github.com/fleetdm/fleet/v4/pkg/filepath_windows"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"github.com/skratchdot/open-golang/open"
	"github.com/urfave/cli/v2"
)

var (
	opt               packaging.Options
	disableOpenFolder bool
)

func packageCommand() *cli.Command {
	return &cli.Command{
		Name:        "package",
		Aliases:     nil,
		Usage:       "Create a fleetd agent",
		Description: "An easy way to create fully boot-strapped installer packages for Windows, macOS, or Linux",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "type",
				Usage:    "Type of package to build",
				Required: true,
			},
			&cli.StringFlag{
				Name:        "arch",
				Usage:       "Target CPU Architecture for the installer package (Only supported with '--type' deb or rpm)",
				Destination: &opt.Architecture,
				Value:       "amd64",
			},
			&cli.StringFlag{
				Name:        "enroll-secret",
				Usage:       "Enroll secret for authenticating to Fleet server",
				Destination: &opt.EnrollSecret,
			},
			&cli.StringFlag{
				Name:        "fleet-url",
				Usage:       "URL (host:port) of Fleet server",
				Destination: &opt.FleetURL,
			},
			&cli.StringFlag{
				Name:        "fleet-certificate",
				Usage:       "Path to the Fleet server certificate chain",
				Destination: &opt.FleetCertificate,
			},
			&cli.StringFlag{
				Name:        "fleet-tls-client-certificate",
				Usage:       "Path to a TLS client certificate to use when connecting to the Fleet server. This functionality is licensed under the Fleet EE License. Usage requires a current Fleet EE subscription.",
				Destination: &opt.FleetTLSClientCertificate,
			},
			&cli.StringFlag{
				Name:        "fleet-tls-client-key",
				Usage:       "Path to a TLS client private key to use when connecting to the Fleet server. This functionality is licensed under the Fleet EE License. Usage requires a current Fleet EE subscription.",
				Destination: &opt.FleetTLSClientKey,
			},
			&cli.StringFlag{
				Name:        "fleet-desktop-alternative-browser-host",
				Usage:       "Alternative host:port to use for Fleet Desktop in the browser (this may be required when using TLS client authentication in the Fleet server)",
				Destination: &opt.FleetDesktopAlternativeBrowserHost,
			},
			&cli.StringFlag{
				Name:        "identifier",
				Usage:       "Identifier for package product",
				Value:       "com.fleetdm.orbit",
				Destination: &opt.Identifier,
			},
			&cli.BoolFlag{
				Name:        "insecure",
				Usage:       "Disable TLS certificate verification",
				Destination: &opt.Insecure,
			},
			&cli.BoolFlag{
				Name:        "service",
				Usage:       "Install orbit/osquery with a persistence service (launchd, systemd, etc.)",
				Value:       true,
				Destination: &opt.StartService,
			},
			&cli.StringFlag{
				Name:        "sign-identity",
				Usage:       "Identity to use for macOS codesigning",
				Destination: &opt.SignIdentity,
			},
			&cli.BoolFlag{
				Name:        "notarize",
				Usage:       "Whether to notarize macOS packages",
				Destination: &opt.Notarize,
			},
			&cli.StringFlag{
				Name:        "osqueryd-channel",
				Usage:       "Update channel of osqueryd to use",
				Value:       "stable",
				Destination: &opt.OsquerydChannel,
			},
			&cli.StringFlag{
				Name:        "desktop-channel",
				Usage:       "Update channel of desktop to use",
				Value:       "stable",
				Destination: &opt.DesktopChannel,
			},
			&cli.StringFlag{
				Name:        "orbit-channel",
				Usage:       "Update channel of Orbit to use",
				Value:       "stable",
				Destination: &opt.OrbitChannel,
			},
			&cli.BoolFlag{
				Name:        "disable-updates",
				Usage:       "Disable auto updates on the generated package",
				Destination: &opt.DisableUpdates,
			},
			&cli.StringFlag{
				Name:        "update-url",
				Usage:       "URL for update server",
				Value:       "https://tuf.fleetctl.com",
				Destination: &opt.UpdateURL,
			},
			&cli.StringFlag{
				Name:        "update-roots",
				Usage:       "Root key JSON metadata for update server (from fleetctl updates roots)",
				Destination: &opt.UpdateRoots,
			},
			&cli.StringFlag{
				Name:        "update-tls-certificate",
				Usage:       "Path to the update server TLS certificate chain",
				Destination: &opt.UpdateTLSServerCertificate,
			},
			&cli.StringFlag{
				Name:        "update-tls-client-certificate",
				Usage:       "Path to a TLS client certificate to use when connecting to the update server. This functionality is licensed under the Fleet EE License. Usage requires a current Fleet EE subscription.",
				Destination: &opt.UpdateTLSClientCertificate,
			},
			&cli.StringFlag{
				Name:        "update-tls-client-key",
				Usage:       "Path to a TLS client private key to use when connecting to the update server. This functionality is licensed under the Fleet EE License. Usage requires a current Fleet EE subscription.",
				Destination: &opt.UpdateTLSClientKey,
			},
			&cli.StringFlag{
				Name:        "osquery-flagfile",
				Usage:       "Flagfile to package and provide to osquery",
				Destination: &opt.OsqueryFlagfile,
			},
			&cli.BoolFlag{
				Name:        "debug",
				Usage:       "Enable debug logging in orbit",
				Destination: &opt.Debug,
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "Log detailed information when building the package",
			},
			&cli.BoolFlag{
				Name:        "fleet-desktop",
				Usage:       "Include the Fleet Desktop Application in the package",
				Destination: &opt.Desktop,
			},
			&cli.DurationFlag{
				Name:        "update-interval",
				Usage:       "Interval that Orbit will use to check for new updates (10s, 1h, etc.)",
				Value:       15 * time.Minute,
				Destination: &opt.OrbitUpdateInterval,
			},
			&cli.BoolFlag{
				Name:        "disable-open-folder",
				Usage:       "Disable opening the folder at the end",
				Destination: &disableOpenFolder,
			},
			&cli.BoolFlag{
				Name:        "native-tooling",
				Usage:       "Build the package using native tooling (only available in Linux)",
				EnvVars:     []string{"FLEETCTL_NATIVE_TOOLING"},
				Destination: &opt.NativeTooling,
			},
			eefleetctl.LocalWixDirFlag(&opt.LocalWixDir),
			&cli.StringFlag{
				Name:        "macos-devid-pem-content",
				Usage:       "Dev ID certificate keypair content in PEM format",
				EnvVars:     []string{"FLEETCTL_MACOS_DEVID_PEM_CONTENT"},
				Destination: &opt.MacOSDevIDCertificateContent,
			},
			&cli.StringFlag{
				Name:        "app-store-connect-api-key-id",
				Usage:       "App Store Connect API key used for notarization",
				EnvVars:     []string{"FLEETCTL_APP_STORE_CONNECT_API_KEY_ID"},
				Destination: &opt.AppStoreConnectAPIKeyID,
			},
			&cli.StringFlag{
				Name:        "app-store-connect-api-key-issuer",
				Usage:       "Issuer of the App Store Connect API key",
				EnvVars:     []string{"FLEETCTL_APP_STORE_CONNECT_API_KEY_ISSUER"},
				Destination: &opt.AppStoreConnectAPIKeyIssuer,
			},
			&cli.StringFlag{
				Name:        "app-store-connect-api-key-content",
				Usage:       "Contents of the .p8 App Store Connect API key",
				EnvVars:     []string{"FLEETCTL_APP_STORE_CONNECT_API_KEY_CONTENT"},
				Destination: &opt.AppStoreConnectAPIKeyContent,
			},
			&cli.BoolFlag{
				Name:        "use-system-configuration",
				Usage:       "Try to read --fleet-url and --enroll-secret using configuration in the host (currently only macOS profiles are supported)",
				EnvVars:     []string{"FLEETCTL_USE_SYSTEM_CONFIGURATION"},
				Destination: &opt.UseSystemConfiguration,
			},
			&cli.BoolFlag{
				Name:        "enable-scripts",
				Usage:       "Enable script execution",
				EnvVars:     []string{"FLEETCTL_ENABLE_SCRIPTS"},
				Destination: &opt.EnableScripts,
			},
			&cli.StringFlag{
				Name:        "host-identifier",
				Usage:       "Sets the host identifier that orbit and osquery will use when enrolling to Fleet. Options: 'uuid' and 'instance' (requires Fleet >= v4.42.0)",
				Value:       "uuid",
				EnvVars:     []string{"FLEETCTL_HOST_IDENTIFIER"},
				Destination: &opt.HostIdentifier,
			},
			&cli.StringFlag{
				Name:        "end-user-email",
				Usage:       "End user's email that populates human to host mapping in Fleet (only available on Windows and Linux)",
				EnvVars:     []string{"FLEETCTL_END_USER_EMAIL"},
				Destination: &opt.EndUserEmail,
			},
			&cli.BoolFlag{
				Name:        "disable-keystore",
				Usage:       "Disables the use of the keychain on macOS and Credentials Manager on Windows",
				EnvVars:     []string{"FLEETCTL_DISABLE_KEYSTORE"},
				Destination: &opt.DisableKeystore,
			},
			&cli.StringFlag{
				Name:        "osquery-db",
				Usage:       "Sets a custom osquery database directory, it must be an absolute path (requires orbit >= v1.22.0)",
				EnvVars:     []string{"FLEETCTL_OSQUERY_DB"},
				Destination: &opt.OsqueryDB,
			},
		},
		Action: func(c *cli.Context) error {
			if opt.FleetURL != "" || opt.EnrollSecret != "" {
				if opt.FleetURL == "" || opt.EnrollSecret == "" {
					return errors.New("--enroll-secret and --fleet-url must be provided together")
				}
			}

			if opt.Insecure && opt.FleetCertificate != "" {
				return errors.New("--insecure and --fleet-certificate may not be provided together")
			}

			if opt.Insecure && opt.UpdateTLSServerCertificate != "" {
				return errors.New("--insecure and --update-tls-certificate may not be provided together")
			}

			if opt.HostIdentifier != "uuid" && opt.HostIdentifier != "instance" {
				return fmt.Errorf("--host-identifier=%s is not supported, currently supported values are 'uuid' and 'instance'", opt.HostIdentifier)
			}

			// Perform checks on the provided fleet client certificate and key.
			if (opt.FleetTLSClientCertificate != "") != (opt.FleetTLSClientKey != "") {
				return errors.New("must specify both fleet-tls-client-certificate and fleet-tls-client-key")
			}
			if opt.FleetTLSClientKey != "" {
				if _, err := tls.LoadX509KeyPair(opt.FleetTLSClientCertificate, opt.FleetTLSClientKey); err != nil {
					return fmt.Errorf("error loading fleet client certificate and key: %w", err)
				}
			}

			// Perform checks on the provided update client certificate and key.
			if (opt.UpdateTLSClientCertificate != "") != (opt.UpdateTLSClientKey != "") {
				return errors.New("must specify both update-tls-client-certificate and update-tls-client-key")
			}
			if opt.UpdateTLSClientKey != "" {
				if _, err := tls.LoadX509KeyPair(opt.UpdateTLSClientCertificate, opt.UpdateTLSClientKey); err != nil {
					return fmt.Errorf("error loading update client certificate and key: %w", err)
				}
			}

			if opt.OsqueryDB != "" && !isAbsolutePath(opt.OsqueryDB, c.String("type")) {
				return fmt.Errorf("--osquery-db must be an absolute path: %q", opt.OsqueryDB)
			}

			if runtime.GOOS == "windows" && c.String("type") != "msi" {
				return errors.New("Windows can only build MSI packages.")
			}

			if opt.NativeTooling && runtime.GOOS != "linux" {
				return errors.New("native tooling is only available in Linux")
			}

			if opt.LocalWixDir != "" && runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
				return errors.New(
					`Could not use local WiX to generate an osquery installer. This option is only available on Windows and macOS.
				Visit https://wixtoolset.org/ for more information about how to use WiX.`)
			}

			if opt.EndUserEmail != "" {
				if !fleet.IsLooseEmail(opt.EndUserEmail) {
					return errors.New("Invalid email address specified for --end-user-email.")
				}

				switch c.String("type") {
				case "msi", "deb", "rpm":
					// ok
				default:
					return errors.New("Can only set --end-user-email when building an MSI, DEB, or RPM package.")
				}
			}

			if opt.FleetCertificate != "" {
				err := checkPEMCertificate(opt.FleetCertificate)
				if err != nil {
					return fmt.Errorf("failed to read fleet server certificate %q: %w", opt.FleetCertificate, err)
				}
			}

			if opt.UpdateTLSServerCertificate != "" {
				err := checkPEMCertificate(opt.UpdateTLSServerCertificate)
				if err != nil {
					return fmt.Errorf("failed to read update server certificate %q: %w", opt.UpdateTLSServerCertificate, err)
				}
			}

			if opt.UseSystemConfiguration && c.String("type") != "pkg" {
				return errors.New("--use-system-configuration is only available for pkg installers")
			}

			linuxPackage := false
			switch c.String("type") {
			case "deb", "rpm":
				linuxPackage = true
			}

			if opt.Architecture != packaging.ArchAmd64 && !linuxPackage {
				return fmt.Errorf("can't use '--arch' with '--type %s'", c.String("type"))
			}

			if opt.Architecture != packaging.ArchAmd64 && opt.Architecture != packaging.ArchArm64 {
				return errors.New("arch must be one of ('amd64', 'arm64')")
			}

			var buildFunc func(packaging.Options) (string, error)
			switch c.String("type") {
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
			if !c.Bool("verbose") {
				zlog.Logger = zerolog.Nop()
			}

			fmt.Println("Generating your fleetd agent...")
			path, err := buildFunc(opt)
			if err != nil {
				return err
			}

			path, _ = filepath.Abs(path)
			pathBase := filepath.Base(path)
			var installInstructions = "double-click the installer"
			var deviceType string
			switch c.String("type") {
			case "pkg":
				installInstructions += fmt.Sprintf(" or run the command `installer -pkg \"%s\" -target /`", pathBase)
				deviceType = "macOS"
			case "deb":
				installInstructions += fmt.Sprintf(" or run the command `sudo apt install \"%s\"`", pathBase)
				deviceType = "Debian-based Linux"
			case "rpm":
				installInstructions += fmt.Sprintf(" or run the command `sudo dnf install \"%s\"`", pathBase)
				deviceType = "RPM-based Linux"
			case "msi":
				installInstructions += fmt.Sprintf(" or run the command `msiexec /i \"%s\"` as administrator", pathBase)
				deviceType = "Windows"
			}

			fmt.Printf(`
Success! You generated fleetd at %s

To add a new %s device to Fleet, %s.

To add other devices to Fleet, distribute fleetd using Chef, Ansible, Jamf, or Puppet. Learn how: https://fleetdm.com/learn-more-about/enrolling-hosts
`, path, deviceType, installInstructions)
			if !disableOpenFolder {
				open.Start(filepath.Dir(path)) //nolint:errcheck
			}
			return nil
		},
	}
}

func checkPEMCertificate(path string) error {
	cert, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if p, _ := pem.Decode(cert); p == nil {
		return errors.New("invalid PEM file")
	}
	return nil
}

// isAbsolutePath returns whether a path is absolute.
// It does not make use of filepath.IsAbs to support
// checking Windows paths from Go code running in unix.
func isAbsolutePath(path, pkgType string) bool {
	if pkgType == "msi" {
		return filepath_windows.IsAbs(path)
	}
	return strings.HasPrefix(path, "/") // this is the unix implementation of filepath.IsAbs
}
