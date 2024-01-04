package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/packaging"
	"github.com/fleetdm/fleet/v4/pkg/buildpkg"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/fleetdm/fleet/v4/server/version"
	"github.com/mitchellh/gon/package/zip"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

func main() {
	app := createApp(os.Stdin, os.Stdout, os.Stderr, exitErrHandler)
	app.Run(os.Args) //nolint:errcheck
}

// exitErrHandler implements cli.ExitErrHandlerFunc. If there is an error, prints it to stderr and exits with status 1.
func exitErrHandler(c *cli.Context, err error) {
	if err == nil {
		return
	}
	fmt.Fprintf(c.App.ErrWriter, "Error: %+v\n", err)
	cli.OsExiter(1)
}

func createApp(reader io.Reader, stdout io.Writer, stderr io.Writer, exitErrHandler cli.ExitErrHandlerFunc) *cli.App {
	app := cli.NewApp()
	app.Name = "desktop"
	app.Usage = "Tool to generate the Fleet Desktop application"
	app.ExitErrHandler = exitErrHandler
	cli.VersionPrinter = func(c *cli.Context) {
		version.PrintFull()
	}
	app.Reader = reader
	app.Writer = stdout
	app.ErrWriter = stderr
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "version",
			Usage:   "Version of the Fleet Desktop application",
			EnvVars: []string{"FLEET_DESKTOP_VERSION"},
		},
		&cli.BoolFlag{
			Name:    "verbose",
			Usage:   "Log detailed information when building the application",
			EnvVars: []string{"FLEET_DESKTOP_VERBOSE"},
		},
	}

	app.Commands = []*cli.Command{
		macos(),
	}
	return app
}

func macos() *cli.Command {
	return &cli.Command{
		Name:        "macos",
		Usage:       "Creates the Fleet Desktop Application for macOS",
		Description: "Builds and signs the Fleet Desktop .app bundle for macOS",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "authority",
				Usage:   "Authority to use on the codesign invocation (if not set, app is not signed)",
				EnvVars: []string{"FLEET_DESKTOP_APPLE_AUTHORITY"},
			},
			&cli.BoolFlag{
				Name:    "notarize",
				Usage:   "If true, the generated application will be notarized and stapled. Requires the `AC_USERNAME` and `AC_PASSWORD` to be set in the environment",
				EnvVars: []string{"FLEET_DESKTOP_NOTARIZE"},
			},
		},
		Action: func(c *cli.Context) error {
			if !c.Bool("verbose") {
				zlog.Logger = zerolog.Nop()
			}
			return createMacOSApp(c.String("version"), c.String("authority"), c.Bool("notarize"))
		},
	}
}

func createMacOSApp(version, authority string, notarize bool) error {
	const (
		appDir           = "Fleet Desktop.app"
		bundleIdentifier = "com.fleetdm.desktop"
		// infoPList is the Info.plist file to use for the macOS .app bundle.
		//
		// 	- NSHighResolutionCapable=true: avoid having a blurry icon and text.
		//	- LSUIElement=1: avoid showing the app on the Dock.
		infoPList = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>fleet-desktop</string>
	<key>CFBundleIdentifier</key>
	<string>%s</string>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
	<key>CFBundleName</key>
	<string>fleet-desktop</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>%s</string>
	<key>CFBundleVersion</key>
	<string>%s</string>
	<key>NSHighResolutionCapable</key>
	<string>True</string>
	<key>LSUIElement</key>
	<string>1</string>
</dict>
</plist>
`
	)

	if runtime.GOOS != "darwin" {
		return errors.New(`the "Fleet Desktop" macOS app can only be created from macOS`)
	}

	defer os.RemoveAll(appDir)

	contentsDir := filepath.Join(appDir, "Contents")
	macOSDir := filepath.Join(contentsDir, "MacOS")
	if err := secure.MkdirAll(macOSDir, constant.DefaultDirMode); err != nil {
		return fmt.Errorf("create directory %q: %w", macOSDir, err)
	}

	infoFile := filepath.Join(contentsDir, "Info.plist")
	infoPListContents := fmt.Sprintf(infoPList, bundleIdentifier, version, version)
	if err := os.WriteFile(infoFile, []byte(infoPListContents), 0o644); err != nil {
		return fmt.Errorf("create Info.plist file %q: %w", infoFile, err)
	}

	binaryPath := filepath.Join(macOSDir, constant.DesktopAppExecName)

	amdBinaryPath := binaryPath + "_amd64"
	/* #nosec G204 -- arguments are actually well defined */
	buildExec := exec.Command("go", "build",
		"-o", amdBinaryPath,
		"-ldflags", os.ExpandEnv("-X=main.version=$FLEET_DESKTOP_VERSION"),
		"./"+filepath.Join("orbit", "cmd", "desktop"),
	)
	buildExec.Env = append(os.Environ(), "CGO_ENABLED=1", "GOOS=darwin", "GOARCH=amd64")
	buildExec.Stderr = os.Stderr
	buildExec.Stdout = os.Stdout

	zlog.Info().Str("command", buildExec.String()).Msg("Build fleet-desktop executable amd64")
	if err := buildExec.Run(); err != nil {
		return fmt.Errorf("compile for amd64: %w", err)
	}

	armBinaryPath := binaryPath + "_arm64"
	/* #nosec G204 -- arguments are actually well defined */
	buildExec = exec.Command("go", "build",
		"-o", armBinaryPath,
		"-ldflags", os.ExpandEnv("-X=main.version=$FLEET_DESKTOP_VERSION"),
		"./"+filepath.Join("orbit", "cmd", "desktop"),
	)
	buildExec.Env = append(os.Environ(), "CGO_ENABLED=1", "GOOS=darwin", "GOARCH=arm64")
	buildExec.Stderr = os.Stderr
	buildExec.Stdout = os.Stdout

	zlog.Info().Str("command", buildExec.String()).Msg("Build fleet-desktop executable arm64")
	if err := buildExec.Run(); err != nil {
		return fmt.Errorf("compile for arm64: %w", err)
	}

	// Make the fat exe and remove the separate binaries
	if err := buildpkg.MakeMacOSFatExecutable(binaryPath, amdBinaryPath, armBinaryPath); err != nil {
		return fmt.Errorf("make fat executable: %w", err)
	}
	if err := os.Remove(amdBinaryPath); err != nil {
		return fmt.Errorf("remove amd64 binary: %w", err)
	}
	if err := os.Remove(armBinaryPath); err != nil {
		return fmt.Errorf("remove arm64 binary: %w", err)
	}

	if authority != "" {
		codeSign := exec.Command("codesign", "-s", authority, "-i", bundleIdentifier, "-f", "-v", "--timestamp", "--options", "runtime", appDir)

		zlog.Info().Str("command", codeSign.String()).Msg("Sign Fleet Desktop.app")

		codeSign.Stderr = os.Stderr
		codeSign.Stdout = os.Stdout
		if err := codeSign.Run(); err != nil {
			return fmt.Errorf("sign application: %w", err)
		}
	}

	if notarize {
		const notarizationZip = "desktop.zip"
		// Note that the app needs to be zipped in order to upload to Apple for Notarization, but
		// the Stapling has to happen on just the app (not zipped). Apple is a bit inconsistent here.
		if err := zip.Zip(context.Background(), &zip.Options{Files: []string{appDir}, OutputPath: notarizationZip}); err != nil {
			return fmt.Errorf("zip app for notarization: %w", err)
		}
		defer os.Remove(notarizationZip)

		if err := packaging.Notarize(notarizationZip, "com.fleetdm.desktop"); err != nil {
			return err
		}

		if err := packaging.Staple(appDir); err != nil {
			return err
		}

	}

	const tarGzName = "desktop.app.tar.gz"
	if err := compressDir(tarGzName, appDir); err != nil {
		return fmt.Errorf("compress app: %w", err)
	}
	fmt.Printf("Generated %s successfully.\n", tarGzName)

	return nil
}

func compressDir(outPath, dirPath string) error {
	out, err := secure.OpenFile(outPath, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer out.Close()

	gw := gzip.NewWriter(out)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	if err := filepath.Walk(dirPath, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}

		// From https://golang.org/src/archive/tar/common.go?#L626
		//
		//	"Since fs.FileInfo's Name method only returns the base name of
		// 	the file it describes, it may be necessary to modify Header.Name
		// 	to provide the full path name of the file."
		header.Name = filepath.ToSlash(file)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if !fi.IsDir() {
			f, err := os.Open(file)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("walk directory: %w", err)
	}

	if err := tw.Close(); err != nil {
		return fmt.Errorf("close tar: %w", err)
	}
	if err := gw.Close(); err != nil {
		return fmt.Errorf("close gzip: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}

	return nil
}
