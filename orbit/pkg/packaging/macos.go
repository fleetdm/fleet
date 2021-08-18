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
	"github.com/fleetdm/fleet/v4/secure"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// See helful docs in http://bomutils.dyndns.org/tutorial.html

// BuildPkg builds a macOS .pkg. So far this is tested only on macOS but in theory it works with bomutils on
// Linux.
func BuildPkg(opt Options) error {
	// Initialize directories

	tmpDir, err := ioutil.TempDir("", "orbit-package")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(tmpDir)
	log.Debug().Str("path", tmpDir).Msg("created temp dir")

	filesystemRoot := filepath.Join(tmpDir, "root")
	if err := secure.MkdirAll(filesystemRoot, constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "create root dir")
	}
	orbitRoot := filepath.Join(filesystemRoot, "var", "lib", "orbit")
	if err := secure.MkdirAll(orbitRoot, constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "create orbit dir")
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

	if err := initializeUpdates(updateOpt); err != nil {
		return errors.Wrap(err, "initialize updates")
	}

	// Write files

	if err := writePackageInfo(opt, tmpDir); err != nil {
		return errors.Wrap(err, "write PackageInfo")
	}
	if err := writeDistribution(opt, tmpDir); err != nil {
		return errors.Wrap(err, "write Distribution")
	}
	if err := writeScripts(opt, tmpDir); err != nil {
		return errors.Wrap(err, "write postinstall")
	}
	if err := writeSecret(opt, orbitRoot); err != nil {
		return errors.Wrap(err, "write enroll secret")
	}
	if opt.StartService {
		if err := writeLaunchd(opt, filesystemRoot); err != nil {
			return errors.Wrap(err, "write launchd")
		}
	}
	if opt.FleetCertificate != "" {
		if err := writeCertificate(opt, orbitRoot); err != nil {
			return errors.Wrap(err, "write fleet certificate")
		}
	}

	// TODO gate behind a flag and allow copying a local orbit
	// if err := copyFile(
	// 	"./orbit",
	// 	filepath.Join(orbitRoot, "bin", "orbit", "macos", "current", "orbit"),
	// 	0755,
	// ); err != nil {
	// 	return errors.Wrap(err, "write orbit")
	// }

	// Build package

	if err := xarBom(opt, tmpDir); err != nil {
		return errors.Wrap(err, "build pkg")
	}

	generatedPath := filepath.Join(tmpDir, "orbit.pkg")

	if len(opt.SignIdentity) != 0 {
		log.Info().Str("identity", opt.SignIdentity).Msg("productsign package")
		if err := signPkg(generatedPath, opt.SignIdentity); err != nil {
			return errors.Wrap(err, "productsign")
		}
	}

	if opt.Notarize {
		if err := notarizePkg(generatedPath); err != nil {
			return err
		}
	}

	filename := fmt.Sprintf("orbit-osquery_%s_amd64.pkg", opt.Version)
	if err := os.Rename(generatedPath, filename); err != nil {
		return errors.Wrap(err, "rename pkg")
	}
	log.Info().Str("path", filename).Msg("wrote pkg package")

	return nil
}

func writePackageInfo(opt Options, rootPath string) error {
	// PackageInfo is metadata for the pkg
	path := filepath.Join(rootPath, "flat", "base.pkg", "PackageInfo")
	if err := secure.MkdirAll(filepath.Dir(path), constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "mkdir")
	}

	var contents bytes.Buffer
	if err := macosPackageInfoTemplate.Execute(&contents, opt); err != nil {
		return errors.Wrap(err, "execute template")
	}

	if err := ioutil.WriteFile(path, contents.Bytes(), constant.DefaultFileMode); err != nil {
		return errors.Wrap(err, "write file")
	}

	return nil
}

func writeScripts(opt Options, rootPath string) error {
	// Postinstall script
	path := filepath.Join(rootPath, "scripts", "postinstall")
	if err := secure.MkdirAll(filepath.Dir(path), constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "mkdir")
	}

	var contents bytes.Buffer
	if err := macosPostinstallTemplate.Execute(&contents, opt); err != nil {
		return errors.Wrap(err, "execute template")
	}

	if err := ioutil.WriteFile(path, contents.Bytes(), 0744); err != nil {
		return errors.Wrap(err, "write file")
	}

	return nil
}

func writeLaunchd(opt Options, rootPath string) error {
	// launchd is the service mechanism on macOS
	path := filepath.Join(rootPath, "Library", "LaunchDaemons", "com.fleetdm.orbit.plist")
	if err := secure.MkdirAll(filepath.Dir(path), constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "mkdir")
	}

	var contents bytes.Buffer
	if err := macosLaunchdTemplate.Execute(&contents, opt); err != nil {
		return errors.Wrap(err, "execute template")
	}

	if err := ioutil.WriteFile(path, contents.Bytes(), 0644); err != nil {
		return errors.Wrap(err, "write file")
	}

	return nil
}

func writeDistribution(opt Options, rootPath string) error {
	// Distribution file is metadata for the pkg
	path := filepath.Join(rootPath, "flat", "Distribution")
	if err := secure.MkdirAll(filepath.Dir(path), constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "mkdir")
	}

	var contents bytes.Buffer
	if err := macosDistributionTemplate.Execute(&contents, opt); err != nil {
		return errors.Wrap(err, "execute template")
	}

	if err := ioutil.WriteFile(path, contents.Bytes(), constant.DefaultFileMode); err != nil {
		return errors.Wrap(err, "write file")
	}

	return nil
}

func writeCertificate(opt Options, orbitRoot string) error {
	// Fleet TLS certificate
	dstPath := filepath.Join(orbitRoot, "fleet.pem")

	if err := copyFile(opt.FleetCertificate, dstPath, 0644); err != nil {
		return errors.Wrap(err, "write orbit")
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
		return errors.Wrap(err, "cpio Payload")
	}
	if err := cpio(
		filepath.Join(rootPath, "scripts"),
		filepath.Join(rootPath, "flat", "base.pkg", "Scripts"),
	); err != nil {
		return errors.Wrap(err, "cpio Scripts")
	}

	// Make bom
	var cmdMkbom *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmdMkbom = exec.Command("mkbom", filepath.Join(rootPath, "root"), filepath.Join("flat", "base.pkg", "Bom"))
	case "linux":
		cmdMkbom = exec.Command("mkbom", "-u", "0", "-g", "80", filepath.Join(rootPath, "flat", "root"), filepath.Join("flat", "base.pkg", "Bom"))
	}
	cmdMkbom.Dir = rootPath
	cmdMkbom.Stdout = os.Stdout
	cmdMkbom.Stderr = os.Stderr
	if err := cmdMkbom.Run(); err != nil {
		return errors.Wrap(err, "mkbom")
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
		return errors.Wrap(err, "iterate files")
	}

	// Make xar
	cmdXar := exec.Command("xar", append([]string{"--compression", "none", "-cf", filepath.Join("..", "orbit.pkg")}, files...)...)
	cmdXar.Dir = filepath.Join(rootPath, "flat")
	cmdXar.Stdout = os.Stdout
	cmdXar.Stderr = os.Stderr

	if err := cmdXar.Run(); err != nil {
		return errors.Wrap(err, "run xar")
	}

	return nil
}

func cpio(srcPath, dstPath string) error {
	// This is the compression routine that is expected for pkg files.
	dst, err := secure.OpenFile(dstPath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return errors.Wrap(err, "open dst")
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
		return errors.Wrap(err, "pipe cpio")
	}
	cmdGzip.Stdin, err = cmdCpio.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "pipe gzip")
	}
	cmdGzip.Stdout = dst

	err = cmdGzip.Start()
	if err != nil {
		return errors.Wrap(err, "start gzip")
	}
	err = cmdCpio.Start()
	if err != nil {
		return errors.Wrap(err, "start cpio")
	}
	err = cmdFind.Run()
	if err != nil {
		return errors.Wrap(err, "run find")
	}
	err = cmdCpio.Wait()
	if err != nil {
		return errors.Wrap(err, "wait cpio")
	}
	err = cmdGzip.Wait()
	if err != nil {
		return errors.Wrap(err, "wait gzip")
	}
	err = dst.Sync()
	if err != nil {
		return errors.Wrap(err, "sync dst")
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
		return errors.Wrap(err, "productsign")
	}

	if err := os.Rename(pkgPath+".signed", pkgPath); err != nil {
		return errors.Wrap(err, "rename signed")
	}

	return nil
}
