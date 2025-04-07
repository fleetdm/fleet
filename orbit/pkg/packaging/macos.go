package packaging

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/Masterminds/semver"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/rs/zerolog/log"
)

var bomRegexp = regexp.MustCompile(`(.+)\t([0-9]+/[0-9]+)`)

// See helful docs in http://bomutils.dyndns.org/tutorial.html

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
	if opt.NativeTooling {
		filename = filepath.Join("build", filename)
	}
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
// materials?). See http://bomutils.dyndns.org/tutorial.html.
func xarBom(opt Options, rootPath string) error {
	// Adapted from BSD licensed
	// https://github.com/go-flutter-desktop/hover/blob/v0.46.2/cmd/packaging/darwin-pkg.go

	// Copy payload/scripts
	if err := cpio(
		filepath.Join(rootPath, "root"),
		filepath.Join(rootPath, "flat", "base.pkg", "Payload"),
	); err != nil {
		return fmt.Errorf("cpio Payload: %w", err)
	}
	if err := cpio(
		filepath.Join(rootPath, "scripts"),
		filepath.Join(rootPath, "flat", "base.pkg", "Scripts"),
	); err != nil {
		return fmt.Errorf("cpio Scripts: %w", err)
	}

	// Make Bill of materials (bom)
	var cmdMkbom *exec.Cmd
	isDarwin := runtime.GOOS == "darwin"
	isLinuxNative := runtime.GOOS == "linux" && opt.NativeTooling

	switch {
	case isDarwin:
		// Using mkbom directly results in permissions listed for the current user and group. We
		// transform the output in order to explicitly set root (0) and admin (80).
		inBomPath := filepath.Join(rootPath, "inBom")
		cmd := exec.Command("mkbom", filepath.Join(rootPath, "root"), inBomPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("initial mkbom: %w", err)
		}
		bomContents, err := exec.Command("lsbom", inBomPath).Output()
		if err != nil {
			return fmt.Errorf("lsbom inBom: %w", err)
		}
		bomContents = bomReplace(bomContents)
		if err := os.WriteFile(inBomPath, bomContents, 0); err != nil {
			return fmt.Errorf("write inBom: %w", err)
		}

		// Use the file list (with transformed permissions) via -i flag
		cmdMkbom = exec.Command("mkbom", "-i", "inBom", filepath.Join("flat", "base.pkg", "Bom"))
		cmdMkbom.Dir = rootPath

	// No need for transformation when using the Linux mkbom because of the -u and -g flags
	// available in that command.
	case isLinuxNative:
		cmdMkbom = exec.Command(
			"mkbom", "-u", "0", "-g", "80",
			filepath.Join(rootPath, "root"), filepath.Join("flat", "base.pkg", "Bom"),
		)
		cmdMkbom.Dir = rootPath
	default:
		// Same as linux native, but modified for running in Docker. This should
		// be either Windows, or Linux without the --native-tooling flag.
		cmdMkbom = exec.Command(
			"docker", "run", "--rm", "-v", rootPath+":/root", "fleetdm/bomutils",
			"mkbom", "-u", "0", "-g", "80",
			// Use / instead of filepath.Join because these will always be paths within the Docker
			// container (so Linux file paths) -- if we use filepath.Join we'll get invalid paths on
			// Windows due to use of backslashes.
			"/root/root", "/root/flat/base.pkg/Bom",
		)
	}

	cmdMkbom.Stdout, cmdMkbom.Stderr = os.Stdout, os.Stderr
	if err := cmdMkbom.Run(); err != nil {
		return fmt.Errorf("mkbom: %w", err)
	}

	// List files for xar
	var files []string
	err := filepath.Walk(
		filepath.Join(rootPath, "flat"),
		func(path string, info os.FileInfo, _ error) error {
			relativePath, err := filepath.Rel(filepath.Join(rootPath, "flat"), path)
			if err != nil {
				return err
			}
			files = append(files, relativePath)
			return nil
		},
	)
	if err != nil {
		return fmt.Errorf("iterate files: %w", err)
	}

	// Make xar
	var cmdXar *exec.Cmd
	switch {
	case isDarwin, isLinuxNative:
		cmdXar = exec.Command("xar", append([]string{"--compression", "none", "-cf", filepath.Join("..", "orbit.pkg")}, files...)...)
		cmdXar.Dir = filepath.Join(rootPath, "flat")
	default:
		cmdXar = exec.Command(
			"docker", "run", "--rm", "-v", rootPath+":/root", "-w", "/root/flat", "fleetdm/bomutils",
			"xar",
		)
		cmdXar.Args = append(cmdXar.Args, append([]string{"--compression", "none", "-cf", "/root/orbit.pkg"}, files...)...)
	}

	cmdXar.Stdout, cmdXar.Stderr = os.Stdout, os.Stderr
	if err := cmdXar.Run(); err != nil {
		return fmt.Errorf("run xar: %w", err)
	}

	return nil
}

// bomReplace replaces the permission strings (typically "501/20") with the appropriate string ("0/80")
func bomReplace(inBom []byte) []byte {
	return bomRegexp.ReplaceAll(inBom, []byte("$1\t0/80"))
}

func cpio(srcPath, dstPath string) error {
	// This is the compression routine that is expected for pkg files.
	dst, err := secure.OpenFile(dstPath, os.O_RDWR|os.O_CREATE, 0o755)
	if err != nil {
		return fmt.Errorf("open dst: %w", err)
	}
	defer dst.Close()

	cmdFind := exec.Command("find", ".")
	cmdFind.Dir = srcPath
	cmdCpio := exec.Command("cpio", "-o", "--format", "odc", "-R", "0:80")
	cmdCpio.Dir = srcPath
	cmdGzip := exec.Command("gzip", "-c")

	// Pipes like this: find | cpio | gzip > dstPath
	cmdCpio.Stdin, err = cmdFind.StdoutPipe()
	if err != nil {
		return fmt.Errorf("pipe cpio: %w", err)
	}
	cmdGzip.Stdin, err = cmdCpio.StdoutPipe()
	if err != nil {
		return fmt.Errorf("pipe gzip: %w", err)
	}
	cmdGzip.Stdout = dst

	err = cmdGzip.Start()
	if err != nil {
		return fmt.Errorf("start gzip: %w", err)
	}
	err = cmdCpio.Start()
	if err != nil {
		return fmt.Errorf("start cpio: %w", err)
	}
	err = cmdFind.Run()
	if err != nil {
		return fmt.Errorf("run find: %w", err)
	}
	err = cmdCpio.Wait()
	if err != nil {
		return fmt.Errorf("wait cpio: %w", err)
	}
	err = cmdGzip.Wait()
	if err != nil {
		return fmt.Errorf("wait gzip: %w", err)
	}
	err = dst.Sync()
	if err != nil {
		return fmt.Errorf("sync dst: %w", err)
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
