package packaging

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/fleetdm/orbit/pkg/constant"
	"github.com/fleetdm/orbit/pkg/update"
	"github.com/fleetdm/orbit/pkg/update/filestore"
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
	// TODO reenable
	//defer os.RemoveAll(tmpDir)
	log.Debug().Str("path", tmpDir).Msg("created temp dir")

	filesystemRoot := filepath.Join(tmpDir, "root")
	if err := os.MkdirAll(filesystemRoot, constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "create root dir")
	}
	// if err := os.MkdirAll(
	// 	filepath.Join(filesystemRoot, "Resources", "en.lproj"),
	// 	constant.DefaultDirMode,
	// ); err != nil {
	// 	return errors.Wrap(err, "create resources dir")
	// }
	orbitRoot := filepath.Join(filesystemRoot, "var", "lib", "fleet", "orbit")
	if err := os.MkdirAll(orbitRoot, constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "create orbit dir")
	}

	// Write files

	if err := writePackageInfo(opt, tmpDir); err != nil {
		return errors.Wrap(err, "write PackageInfo")
	}
	if err := writeDistribution(opt, tmpDir); err != nil {
		return errors.Wrap(err, "write Distribution")
	}

	// Initialize autoupdate metadata

	localStore, err := filestore.New(filepath.Join(orbitRoot, "tuf-metadata.json"))
	if err != nil {
		return errors.Wrap(err, "failed to create local metadata store")
	}
	updateOpt := update.DefaultOptions
	updateOpt.RootDirectory = orbitRoot
	updateOpt.ServerURL = "https://tuf.fleetctl.com"
	updateOpt.LocalStore = localStore
	updateOpt.Platform = "macos"

	updater, err := update.New(updateOpt)
	if err != nil {
		return errors.Wrap(err, "failed to init updater")
	}
	if err := updater.UpdateMetadata(); err != nil {
		return errors.Wrap(err, "failed to update metadata")
	}
	osquerydPath, err := updater.Get("osqueryd", "stable")
	if err != nil {
		return errors.Wrap(err, "failed to get osqueryd")
	}
	log.Debug().Str("path", osquerydPath).Msg("got osqueryd")

	// Build package
	if err := xarBom(opt, tmpDir); err != nil {
		return errors.Wrap(err, "build pkg")
	}

	filename := fmt.Sprintf("orbit-osquery_%s_amd64.pkg", opt.Version)
	if err := os.Rename(filepath.Join(tmpDir, "orbit.pkg"), filename); err != nil {
		return errors.Wrap(err, "rename pkg")
	}
	log.Info().Str("path", filename).Msg("wrote pkg package")

	return nil
}

func writePackageInfo(opt Options, rootPath string) error {
	path := filepath.Join(rootPath, "flat", "base.pkg", "PackageInfo")
	if err := os.MkdirAll(filepath.Dir(path), constant.DefaultDirMode); err != nil {
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

func writeDistribution(opt Options, rootPath string) error {
	path := filepath.Join(rootPath, "flat", "Distribution")
	if err := os.MkdirAll(filepath.Dir(path), constant.DefaultDirMode); err != nil {
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

// xarBom creates the actual .pkg format. It's a xar archive with a BOM (Bill of
// materials?). See http://bomutils.dyndns.org/tutorial.html.
func xarBom(opt Options, rootPath string) error {
	// Adapted from BSD licensed
	// https://github.com/go-flutter-desktop/hover/blob/v0.46.2/cmd/packaging/darwin-pkg.go

	// Copy payload
	payload, err := os.OpenFile(filepath.Join(rootPath, "flat", "base.pkg", "Payload"), os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return errors.Wrap(err, "open payload")
	}

	cmdFind := exec.Command("find", ".")
	cmdFind.Dir = filepath.Join(rootPath, "root")
	cmdCpio := exec.Command("cpio", "-o", "--format", "odc", "-R", "0:80")
	cmdCpio.Dir = filepath.Join(rootPath, "root")
	cmdGzip := exec.Command("gzip", "-c")

	// Pipes like this: find | cpio | gzip > Payload
	cmdCpio.Stdin, err = cmdFind.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "pipe cpio")
	}
	cmdGzip.Stdin, err = cmdCpio.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "pipe gzip")
	}
	cmdGzip.Stdout = payload

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
	err = payload.Close()
	if err != nil {
		return errors.Wrap(err, "close payload")
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
	if err := filepath.Walk(filepath.Join(rootPath, "flat"), func(path string, info os.FileInfo, err error) error {
		relativePath, err := filepath.Rel(filepath.Join(rootPath, "flat"), path)
		if err != nil {
			return err
		}
		files = append(files, relativePath)
		return nil
	}); err != nil {
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
