package packaging

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/Masterminds/semver"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/rs/zerolog/log"
)

// See helpful docs in http://bomutils.dyndns.org/tutorial.html

// BuildPkg builds a macOS .pkg.
//
// Building packages works out of the box in macOS, but it's also supported on
// Linux given that the necessary dependencies are installed and
// Options.NativeTooling is `true`
//
// Note: this function is not safe for concurrent use
func BuildPkg(opt Options) (string, error) {
	// Initialize directories
	tmpDir, err := initializeTempDir()
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	rootDir := filepath.Join(tmpDir, "root")
	if err := secure.MkdirAll(rootDir, constant.DefaultDirMode); err != nil {
		return "", fmt.Errorf("create root dir: %w", err)
	}
	orbitRoot := filepath.Join(rootDir, "opt", "orbit")
	if err := secure.MkdirAll(orbitRoot, constant.DefaultDirMode); err != nil {
		return "", fmt.Errorf("create orbit dir: %w", err)
	}

	// Initialize autoupdate metadata

	updateOpt := update.DefaultOptions

	updateOpt.RootDirectory = orbitRoot
	updateOpt.ServerURL = opt.UpdateURL
	updateOpt.Targets = update.DarwinTargets
	updateOpt.ServerCertificatePath = opt.UpdateTLSServerCertificate

	if opt.UpdateTLSClientCertificate != "" {
		updateClientCrt, err := tls.LoadX509KeyPair(opt.UpdateTLSClientCertificate, opt.UpdateTLSClientKey)
		if err != nil {
			return "", fmt.Errorf("error loading update client certificate and key: %w", err)
		}
		updateOpt.ClientCertificate = &updateClientCrt
	}

	if opt.Desktop {
		updateOpt.Targets[constant.DesktopTUFTargetName] = update.DesktopMacOSTarget
		// Override default channel with the provided value.
		updateOpt.Targets.SetTargetChannel(constant.DesktopTUFTargetName, opt.DesktopChannel)
	}

	// Override default channels with the provided values.
	updateOpt.Targets.SetTargetChannel(constant.OrbitTUFTargetName, opt.OrbitChannel)
	updateOpt.Targets.SetTargetChannel(constant.OsqueryTUFTargetName, opt.OsquerydChannel)

	if opt.UpdateRoots != "" {
		updateOpt.RootKeys = opt.UpdateRoots
	}

	updatesData, err := InitializeUpdates(updateOpt)
	if err != nil {
		return "", fmt.Errorf("initialize updates: %w", err)
	}
	log.Debug().Stringer("data", updatesData).Msg("updates initialized")
	if opt.Version == "" {
		// We set the package version to orbit's latest version.
		opt.Version = updatesData.OrbitVersion
	}

	if orbitSemVer, err := semver.NewVersion(updatesData.OrbitVersion); err == nil {
		if orbitSemVer.LessThan(semver.MustParse("0.0.11")) {
			opt.LegacyVarLibSymlink = true
		}
	}
	// If err != nil we assume non-legacy Orbit.

	// Write files

	if err := writePackageInfo(opt, tmpDir); err != nil {
		return "", fmt.Errorf("write PackageInfo: %w", err)
	}
	if err := writeDistribution(opt, tmpDir); err != nil {
		return "", fmt.Errorf("write Distribution: %w", err)
	}
	if err := writeScripts(opt, tmpDir); err != nil {
		return "", fmt.Errorf("write postinstall: %w", err)
	}
	if err := writeSecret(opt, orbitRoot); err != nil {
		return "", fmt.Errorf("write enroll secret: %w", err)
	}
	if err := writeOsqueryFlagfile(opt, orbitRoot); err != nil {
		return "", fmt.Errorf("write flagfile: %w", err)
	}
	if err := writeOsqueryCertPEM(opt, orbitRoot); err != nil {
		return "", fmt.Errorf("write certs.pem: %w", err)
	}

	if opt.StartService {
		if err := writeLaunchd(opt, rootDir); err != nil {
			return "", fmt.Errorf("write launchd: %w", err)
		}
	}

	if opt.FleetCertificate != "" {
		if err := writeFleetServerCertificate(opt, orbitRoot); err != nil {
			return "", fmt.Errorf("write fleet server certificate: %w", err)
		}
	}

	if opt.FleetTLSClientCertificate != "" {
		if err := writeFleetClientCertificate(opt, orbitRoot); err != nil {
			return "", fmt.Errorf("write fleet client certificate: %w", err)
		}
	}

	if opt.UpdateTLSServerCertificate != "" {
		if err := writeUpdateServerCertificate(opt, orbitRoot); err != nil {
			return "", fmt.Errorf("write update server certificate: %w", err)
		}
	}

	if opt.UpdateTLSClientCertificate != "" {
		if err := writeUpdateClientCertificate(opt, orbitRoot); err != nil {
			return "", fmt.Errorf("write update client certificate: %w", err)
		}
	}

	// Build package

	if err := xarBom(opt, tmpDir); err != nil {
		return "", fmt.Errorf("build pkg: %w", err)
	}

	generatedPath := filepath.Join(tmpDir, "orbit.pkg")
	isDarwin := runtime.GOOS == "darwin"
	isLinuxNative := runtime.GOOS == "linux" && opt.NativeTooling

	if len(opt.SignIdentity) != 0 {
		if len(opt.MacOSDevIDCertificateContent) != 0 {
			return "", errors.New("providing a sign identity and a Dev ID certificate is not supported")
		}

		log.Info().Str("identity", opt.SignIdentity).Msg("productsign package")
		if err := signPkg(generatedPath, opt.SignIdentity); err != nil {
			return "", fmt.Errorf("productsign: %w", err)
		}
	}

	if isLinuxNative && len(opt.MacOSDevIDCertificateContent) > 0 {
		if len(opt.SignIdentity) != 0 {
			return "", errors.New("providing a sign identity and a Dev ID certificate is not supported")
		}

		if err := rSign(generatedPath, opt.MacOSDevIDCertificateContent); err != nil {
			return "", fmt.Errorf("rcodesign: %w", err)
		}
	}

	if opt.Notarize {
		switch {
		case isDarwin:
			if err := NotarizeStaple(generatedPath, "com.fleetdm.orbit"); err != nil {
				return "", err
			}
		case isLinuxNative:
			if len(opt.AppStoreConnectAPIKeyID) == 0 || len(opt.AppStoreConnectAPIKeyIssuer) == 0 {
				return "", errors.New("both an App Store Connect API key and issuer must be set for native notarization")
			}

			if err := rNotarizeStaple(generatedPath, opt.AppStoreConnectAPIKeyID, opt.AppStoreConnectAPIKeyIssuer, opt.AppStoreConnectAPIKeyContent); err != nil {
				return "", err
			}
		default:
			return "", errors.New("notarization is not supported in this platform")
		}
	}

	filename := "fleet-osquery.pkg"
	if opt.CustomOutfile != "" {
		filename = opt.CustomOutfile
	}
	if opt.NativeTooling {
		filename = filepath.Join("build", filename)
	}
	fmt.Printf("WTF: %s, %s\n", generatedPath, filename)
	if err := file.Copy(generatedPath, filename, constant.DefaultFileMode); err != nil {
		return "", fmt.Errorf("rename pkg: %w", err)
	}
	log.Info().Str("path", filename).Msg("wrote pkg package")

	return filename, nil
}

func writePackageInfo(opt Options, rootPath string) error {
	// PackageInfo is metadata for the pkg
	path := filepath.Join(rootPath, "flat", "base.pkg", "PackageInfo")
	if err := secure.MkdirAll(filepath.Dir(path), constant.DefaultDirMode); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	var contents bytes.Buffer
	if err := macosPackageInfoTemplate.Execute(&contents, opt); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	if err := os.WriteFile(path, contents.Bytes(), constant.DefaultFileMode); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func writeScripts(opt Options, rootPath string) error {
	// Postinstall script
	path := filepath.Join(rootPath, "scripts", "postinstall")
	if err := secure.MkdirAll(filepath.Dir(path), constant.DefaultDirMode); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	var contents bytes.Buffer
	if err := macosPostinstallTemplate.Execute(&contents, opt); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	if err := os.WriteFile(path, contents.Bytes(), 0o744); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func writeLaunchd(opt Options, rootPath string) error {
	// launchd is the service mechanism on macOS
	path := filepath.Join(rootPath, "Library", "LaunchDaemons", "com.fleetdm.orbit.plist")
	if err := secure.MkdirAll(filepath.Dir(path), constant.DefaultDirMode); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	var contents bytes.Buffer
	if err := macosLaunchdTemplate.Execute(&contents, opt); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	if err := os.WriteFile(path, contents.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func writeDistribution(opt Options, rootPath string) error {
	// Distribution file is metadata for the pkg
	path := filepath.Join(rootPath, "flat", "Distribution")
	if err := secure.MkdirAll(filepath.Dir(path), constant.DefaultDirMode); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	var contents bytes.Buffer
	if err := macosDistributionTemplate.Execute(&contents, opt); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	if err := os.WriteFile(path, contents.Bytes(), constant.DefaultFileMode); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func writeFleetServerCertificate(opt Options, orbitRoot string) error {
	dstPath := filepath.Join(orbitRoot, "fleet.pem")

	if err := file.Copy(opt.FleetCertificate, dstPath, 0o644); err != nil {
		return fmt.Errorf("write fleet server certificate: %w", err)
	}

	return nil
}

func writeUpdateServerCertificate(opt Options, orbitRoot string) error {
	dstPath := filepath.Join(orbitRoot, "update.pem")

	if err := file.Copy(opt.UpdateTLSServerCertificate, dstPath, 0o644); err != nil {
		return fmt.Errorf("write update server certificate: %w", err)
	}

	return nil
}

func writeFleetClientCertificate(opt Options, orbitRoot string) error {
	dstPath := filepath.Join(orbitRoot, constant.FleetTLSClientCertificateFileName)
	if err := file.Copy(opt.FleetTLSClientCertificate, dstPath, constant.DefaultFileMode); err != nil {
		return fmt.Errorf("write fleet certificate file: %w", err)
	}
	dstPath = filepath.Join(orbitRoot, constant.FleetTLSClientKeyFileName)
	if err := file.Copy(opt.FleetTLSClientKey, dstPath, constant.DefaultFileMode); err != nil {
		return fmt.Errorf("write fleet key file: %w", err)
	}
	return nil
}

func writeUpdateClientCertificate(opt Options, orbitRoot string) error {
	dstPath := filepath.Join(orbitRoot, constant.UpdateTLSClientCertificateFileName)
	if err := file.Copy(opt.UpdateTLSClientCertificate, dstPath, constant.DefaultFileMode); err != nil {
		return fmt.Errorf("write update certificate file: %w", err)
	}
	dstPath = filepath.Join(orbitRoot, constant.UpdateTLSClientKeyFileName)
	if err := file.Copy(opt.UpdateTLSClientKey, dstPath, constant.DefaultFileMode); err != nil {
		return fmt.Errorf("write update key file: %w", err)
	}
	return nil
}

// xarBom creates the actual .pkg format. It's a xar archive with a BOM (Bill of
// materials). See http://bomutils.dyndns.org/tutorial.html.
//
// All operations (cpio, mkbom, xar) are implemented in pure Go — no external
// binaries or Docker images required.
func xarBom(_ Options, rootPath string) error {
	// Create compressed cpio archives for payload and scripts.
	if err := writeCPIOGzip(
		filepath.Join(rootPath, "root"),
		filepath.Join(rootPath, "flat", "base.pkg", "Payload"),
		0, 80,
	); err != nil {
		return fmt.Errorf("cpio Payload: %w", err)
	}
	if err := writeCPIOGzip(
		filepath.Join(rootPath, "scripts"),
		filepath.Join(rootPath, "flat", "base.pkg", "Scripts"),
		0, 80,
	); err != nil {
		return fmt.Errorf("cpio Scripts: %w", err)
	}

	// Create Bill of Materials (BOM) with uid=0 (root) and gid=80 (admin).
	if err := writeBOM(
		filepath.Join(rootPath, "root"),
		filepath.Join(rootPath, "flat", "base.pkg", "Bom"),
		0, 80,
	); err != nil {
		return fmt.Errorf("mkbom: %w", err)
	}

	// Create XAR archive (the .pkg file) with no compression.
	if err := writeXAR(
		filepath.Join(rootPath, "flat"),
		filepath.Join(rootPath, "orbit.pkg"),
	); err != nil {
		return fmt.Errorf("run xar: %w", err)
	}

	return nil
}

func signPkg(pkgPath, identity string) error {
	var outBuf bytes.Buffer
	cmdProductsign := exec.Command(
		"productsign",
		"--sign", identity,
		pkgPath,
		pkgPath+".signed",
	)
	cmdProductsign.Stdout = &outBuf
	cmdProductsign.Stderr = &outBuf
	if err := cmdProductsign.Run(); err != nil {
		fmt.Println(outBuf.String())
		return fmt.Errorf("productsign: %w", err)
	}

	if err := os.Rename(pkgPath+".signed", pkgPath); err != nil {
		return fmt.Errorf("rename signed: %w", err)
	}

	return nil
}
