package packaging

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/rs/zerolog/log"
)

// See helful docs in http://bomutils.dyndns.org/tutorial.html

// BuildPkg builds a macOS .pkg. So far this is tested only on macOS but in theory it works with bomutils on
// Linux.
func BuildPkg(opt Options) (string, error) {
	// Initialize directories
	tmpDir, err := initializeTempDir()
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	filesystemRoot := filepath.Join(tmpDir, "root")
	if err := secure.MkdirAll(filesystemRoot, constant.DefaultDirMode); err != nil {
		return "", fmt.Errorf("create root dir: %w", err)
	}
	orbitRoot := filepath.Join(filesystemRoot, "var", "lib", "orbit")
	if err := secure.MkdirAll(orbitRoot, constant.DefaultDirMode); err != nil {
		return "", fmt.Errorf("create orbit dir: %w", err)
	}

	// Initialize autoupdate metadata

	updateOpt := update.DefaultOptions
	updateOpt.Platform = "macos"
	updateOpt.RootDirectory = orbitRoot
	updateOpt.OrbitChannel = opt.OrbitChannel
	updateOpt.OsquerydChannel = opt.OsquerydChannel
	updateOpt.ServerURL = opt.UpdateURL
	if opt.UpdateRoots != "" {
		updateOpt.RootKeys = opt.UpdateRoots
	}

	if err := InitializeUpdates(updateOpt); err != nil {
		return "", fmt.Errorf("initialize updates: %w", err)
	}

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
		if err := writeLaunchd(opt, filesystemRoot); err != nil {
			return "", fmt.Errorf("write launchd: %w", err)
		}
	}
	if opt.FleetCertificate != "" {
		if err := writeCertificate(opt, orbitRoot); err != nil {
			return "", fmt.Errorf("write fleet certificate: %w", err)
		}
	}

	// TODO gate behind a flag and allow copying a local orbit
	// if err := file.Copy(
	// 	"./orbit",
	// 	filepath.Join(orbitRoot, "bin", "orbit", "macos", "current", "orbit"),
	// 	0755,
	// ); err != nil {
	// 	return errors.Wrap(err, "write orbit")
	// }

	// Build package

	if err := xarBom(opt, tmpDir); err != nil {
		return "", fmt.Errorf("build pkg: %w", err)
	}

	generatedPath := filepath.Join(tmpDir, "orbit.pkg")

	if len(opt.SignIdentity) != 0 {
		log.Info().Str("identity", opt.SignIdentity).Msg("productsign package")
		if err := signPkg(generatedPath, opt.SignIdentity); err != nil {
			return "", fmt.Errorf("productsign: %w", err)
		}
	}

	if opt.Notarize {
		if err := notarizePkg(generatedPath); err != nil {
			return "", err
		}
	}

	filename := "fleet-osquery.pkg"
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

	if err := ioutil.WriteFile(path, contents.Bytes(), constant.DefaultFileMode); err != nil {
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

	if err := ioutil.WriteFile(path, contents.Bytes(), 0744); err != nil {
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

	if err := ioutil.WriteFile(path, contents.Bytes(), 0644); err != nil {
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

	if err := ioutil.WriteFile(path, contents.Bytes(), constant.DefaultFileMode); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func writeCertificate(opt Options, orbitRoot string) error {
	// Fleet TLS certificate
	dstPath := filepath.Join(orbitRoot, "fleet.pem")

	if err := file.Copy(opt.FleetCertificate, dstPath, 0644); err != nil {
		return fmt.Errorf("write orbit: %w", err)
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

	// Make bom
	var cmdMkbom *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmdMkbom = exec.Command("mkbom", filepath.Join(rootPath, "root"), filepath.Join("flat", "base.pkg", "Bom"))
		cmdMkbom.Dir = rootPath
	default:
		cmdMkbom = exec.Command(
			"docker", "run", "--rm", "-v", rootPath+":/root", "fleetdm/bomutils",
			"mkbom", "-u", "0", "-g", "80",
			// Use / instead of filepath.Join because these will always be paths within the Docker
			// container (so Linux file paths)
			"/root/root", "/root/flat/base.pkg/Bom",
		)
	}
	cmdMkbom.Stdout = os.Stdout
	cmdMkbom.Stderr = os.Stderr
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
	switch runtime.GOOS {
	case "darwin":
		cmdXar = exec.Command("xar", append([]string{"--compression", "none", "-cf", filepath.Join("..", "orbit.pkg")}, files...)...)
		cmdXar.Dir = filepath.Join(rootPath, "flat")
	default:
		cmdXar = exec.Command(
			"docker", "run", "--rm", "-v", rootPath+":/root", "-w", "/root/flat", "fleetdm/bomutils",
			"xar",
		)
		cmdXar.Args = append(cmdXar.Args, append([]string{"--compression", "none", "-cf", "/root/orbit.pkg"}, files...)...)
	}
	cmdXar.Stdout = os.Stdout
	cmdXar.Stderr = os.Stderr

	if err := cmdXar.Run(); err != nil {
		return fmt.Errorf("run xar: %w", err)
	}

	return nil
}

func cpio(srcPath, dstPath string) error {
	// This is the compression routine that is expected for pkg files.
	dst, err := secure.OpenFile(dstPath, os.O_RDWR|os.O_CREATE, 0755)
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
