package main

import (
	"archive/zip"
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/fleetdm/fleet/v4/orbit/pkg/packaging"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/mitchellh/go-ps"
	"github.com/urfave/cli/v2"
)

const (
	downloadUrl             = "https://github.com/fleetdm/osquery-in-a-box/archive/%s.zip"
	standardQueryLibraryUrl = "https://raw.githubusercontent.com/fleetdm/fleet/main/docs/01-Using-Fleet/standard-query-library/standard-query-library.yml"
	licenseKeyFlagName      = "license-key"
	tagFlagName             = "tag"
	previewConfigFlagName   = "preview-config"
)

func previewCommand() *cli.Command {
	return &cli.Command{
		Name:  "preview",
		Usage: "Start a preview deployment of the Fleet server",
		Description: `Start a preview deployment of the Fleet server using Docker and docker-compose. Docker tools must be available in the environment.

Use the stop and reset subcommands to manage the server and dependencies once started.`,
		Subcommands: []*cli.Command{
			previewStopCommand(),
			previewResetCommand(),
		},
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			debugFlag(),
			&cli.StringFlag{
				Name:  licenseKeyFlagName,
				Usage: "License key to enable Fleet Premium (optional)",
			},
			&cli.StringFlag{
				Name:  tagFlagName,
				Usage: "Run a specific version of Fleet",
				Value: "latest",
			},
			&cli.StringFlag{
				Name:  previewConfigFlagName,
				Usage: "Run a specific branch of the preview repository",
				Value: "production",
			},
		},
		Action: func(c *cli.Context) error {
			if err := checkDocker(); err != nil {
				return err
			}

			// Download files every time to ensure the user gets the most up to date versions
			previewDir := previewDirectory()
			osqueryBranch := c.String(previewConfigFlagName)
			fmt.Printf("Downloading dependencies from %s into %s...\n", osqueryBranch, previewDir)
			if err := downloadFiles(osqueryBranch); err != nil {
				return fmt.Errorf("Error downloading dependencies: %w", err)
			}

			if err := os.Chdir(previewDir); err != nil {
				return err
			}
			if _, err := os.Stat("docker-compose.yml"); err != nil {
				return fmt.Errorf("docker-compose file not found in preview directory: %w", err)
			}

			// Make sure the logs directory is writable, otherwise the Fleet
			// server errors on startup. This can be a problem when running on
			// Linux with a non-root user inside the container.
			if err := os.Chmod(filepath.Join(previewDir, "logs"), 0777); err != nil {
				return fmt.Errorf("make logs writable: %w", err)
			}
			if err := os.Chmod(filepath.Join(previewDir, "vulndb"), 0777); err != nil {
				return fmt.Errorf("make vulndb writable: %w", err)
			}

			if err := os.Setenv("FLEET_VERSION", c.String(tagFlagName)); err != nil {
				return fmt.Errorf("failed to set Fleet version: %w", err)
			}

			fmt.Println("Pulling Docker dependencies...")
			out, err := exec.Command("docker-compose", "pull").CombinedOutput()
			if err != nil {
				fmt.Println(string(out))
				return errors.New("Failed to run docker-compose")
			}

			fmt.Println("Starting Docker containers...")
			cmd := exec.Command("docker-compose", "up", "-d", "--remove-orphans", "mysql01", "redis01", "fleet01")
			cmd.Env = append(os.Environ(), "FLEET_LICENSE_KEY="+c.String(licenseKeyFlagName))
			out, err = cmd.CombinedOutput()
			if err != nil {
				fmt.Println(string(out))
				return errors.New("Failed to run docker-compose")
			}

			fmt.Println("Waiting for server to start up...")
			if err := waitStartup(); err != nil {
				return fmt.Errorf("wait for server startup: %w", err)
			}

			// Start fleet02 (UI server) after fleet01 (agent/fleetctl server)
			// has finished starting up so that there is no conflict with
			// running database migrations.
			cmd = exec.Command("docker-compose", "up", "-d", "--remove-orphans", "fleet02")
			cmd.Env = append(os.Environ(), "FLEET_LICENSE_KEY="+c.String(licenseKeyFlagName))
			out, err = cmd.CombinedOutput()
			if err != nil {
				fmt.Println(string(out))
				return errors.New("Failed to run docker-compose")
			}

			fmt.Println("Initializing server...")
			const (
				address  = "https://localhost:8412"
				email    = "admin@example.com"
				password = "admin123#"
			)

			fleetClient, err := service.NewClient(address, true, "", "")
			if err != nil {
				return fmt.Errorf("Error creating Fleet API client handler: %w", err)
			}

			token, err := fleetClient.Setup(email, "Admin", password, "Fleet for osquery")
			if err != nil {
				switch ctxerr.Cause(err).(type) {
				case service.SetupAlreadyErr:
					// Ignore this error
				default:
					return fmt.Errorf("Error setting up Fleet: %w", err)
				}
			}

			configPath, context := c.String("config"), "default"

			contextConfig := Context{
				Address:       address,
				Email:         email,
				Token:         token,
				TLSSkipVerify: true,
			}

			config, err := readConfig(configPath)
			if err != nil {
				// No existing config
				config.Contexts = map[string]Context{
					"default": contextConfig,
				}
			} else {
				fmt.Println("Configured fleetctl in the 'preview' context to avoid overwriting existing config.")
				context = "preview"
				config.Contexts["preview"] = contextConfig
			}
			c.Set("context", context)

			if err := writeConfig(configPath, config); err != nil {
				return fmt.Errorf("Error writing fleetctl configuration: %w", err)
			}

			// Create client and get enroll secret
			client, err := unauthenticatedClientFromCLI(c)
			if err != nil {
				return fmt.Errorf("Error making fleetctl client: %w", err)
			}

			token, err = client.Login(email, password)
			if err != nil {
				return fmt.Errorf("fleetctl login failed: %w", err)
			}

			if err := setConfigValue(configPath, context, "token", token); err != nil {
				return fmt.Errorf("Error setting token for the current context: %w", err)
			}
			client.SetToken(token)

			fmt.Println("Loading standard query library...")
			buf, err := downloadStandardQueryLibrary()
			if err != nil {
				return fmt.Errorf("failed to download standard query library: %w", err)
			}

			specGroup, err := specGroupFromBytes(buf)
			if err != nil {
				return fmt.Errorf("failed to parse standard query library: %w", err)
			}

			err = client.ApplyQueries(specGroup.Queries)
			if err != nil {
				return fmt.Errorf("failed to apply standard query library: %w", err)
			}

			// disable anonymous analytics collection and enable software inventory for preview
			if err := client.ApplyAppConfig(map[string]map[string]bool{
				"host_settings":   {"enable_software_inventory": true},
				"server_settings": {"enable_analytics": false},
			}); err != nil {
				return fmt.Errorf("failed to apply updated app config: %w", err)
			}

			fmt.Println("Applying Policies...")
			if err := loadPolicies(client); err != nil {
				fmt.Println("WARNING: Couldn't load policies:", err)
			}

			secrets, err := client.GetEnrollSecretSpec()
			if err != nil {
				return fmt.Errorf("Error retrieving enroll secret: %w", err)
			}

			if len(secrets.Secrets) != 1 {
				return errors.New("Expected 1 active enroll secret")
			}

			// disable anonymous analytics collection for preview
			if err := client.ApplyAppConfig(map[string]map[string]bool{
				"server_settings": {"enable_analytics": false},
			},
			); err != nil {
				return fmt.Errorf("Error disabling anonymous analytics collection in app config: %w", err)
			}

			fmt.Println("Fleet will now enroll your device and log you into the UI automatically.")
			fmt.Println("You can also open the UI at this URL: http://localhost:1337/previewlogin.")
			fmt.Println("Email:", email)
			fmt.Println("Password:", password)

			fmt.Println("Downloading Orbit and osqueryd...")

			if err := downloadOrbitAndStart(previewDir, secrets.Secrets[0].Secret, address); err != nil {
				return fmt.Errorf("downloading orbit and osqueryd: %w", err)
			}

			// Give it a bit of time so the current device is the one with id 1
			fmt.Println("Waiting for current host to enroll...")
			if err := waitFirstHost(client); err != nil {
				return fmt.Errorf("wait for current host: %w", err)
			}

			if err := openBrowser("http://localhost:1337/previewlogin"); err != nil {
				fmt.Println("Automatic browser open failed. Please navigate to http://localhost:1337/previewlogin.")
			}

			fmt.Println("Starting simulated Linux hosts...")
			cmd = exec.Command("docker-compose", "up", "-d", "--remove-orphans")
			cmd.Dir = filepath.Join(previewDir, "osquery")
			cmd.Env = append(os.Environ(),
				"ENROLL_SECRET="+secrets.Secrets[0].Secret,
				"FLEET_URL="+address,
			)
			out, err = cmd.CombinedOutput()
			if err != nil {
				fmt.Println(string(out))
				return errors.New("Failed to run docker-compose")
			}

			fmt.Println("Preview environment complete. Enjoy using Fleet!")

			return nil
		},
	}
}

var testOverridePreviewDirectory string

func previewDirectory() string {
	if testOverridePreviewDirectory != "" {
		return testOverridePreviewDirectory
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "~"
	}
	return filepath.Join(homeDir, ".fleet", "preview")
}

func downloadFiles(branch string) error {
	resp, err := http.Get(fmt.Sprintf(downloadUrl, branch))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download got status %d", resp.StatusCode)
	}

	zipContents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read download contents: %w", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(zipContents), int64(len(zipContents)))
	if err != nil {
		return fmt.Errorf("open download contents for unzip: %w", err)
	}
	// zip.NewReader does not need to be closed (and cannot be)

	if err := unzip(zipReader, branch); err != nil {
		return fmt.Errorf("unzip download contents: %w", err)
	}

	return nil
}

func downloadStandardQueryLibrary() ([]byte, error) {
	resp, err := http.Get(standardQueryLibraryUrl)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %d", resp.StatusCode)
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	return buf, nil
}

// Adapted from https://stackoverflow.com/a/24792688/491710
func unzip(r *zip.Reader, branch string) error {
	previewDir := previewDirectory()

	// Closure to address file descriptors issue with all the deferred .Close()
	// methods
	replacePath := fmt.Sprintf("osquery-in-a-box-%s", branch)
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		path := f.Name
		path = strings.Replace(path, replacePath, previewDir, 1)

		// We don't need to check for directory traversal as we are already
		// trusting the validity of this ZIP file.

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, f.Mode()); err != nil {
				return err
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(path), f.Mode()); err != nil {
				return err
			}
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

func waitStartup() error {
	retryStrategy := backoff.NewExponentialBackOff()
	retryStrategy.MaxInterval = 1 * time.Second

	client := fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{InsecureSkipVerify: true}))

	if err := backoff.Retry(
		func() error {
			resp, err := client.Get("https://localhost:8412/healthz")
			if err != nil {
				return err
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("got status code %d", resp.StatusCode)
			}
			return nil
		},
		retryStrategy,
	); err != nil {
		return fmt.Errorf("checking server health: %w", err)
	}

	return nil
}

func waitFirstHost(client *service.Client) error {
	retryStrategy := backoff.NewExponentialBackOff()
	retryStrategy.MaxInterval = 1 * time.Second

	if err := backoff.Retry(
		func() error {
			hosts, err := client.GetHosts("")
			if err != nil {
				return err
			}
			if len(hosts) == 0 {
				return errors.New("no hosts yet")
			}

			return nil
		},
		retryStrategy,
	); err != nil {
		return fmt.Errorf("checking host count: %w", err)
	}

	return nil
}

func checkDocker() error {
	// Check installed
	if _, err := exec.LookPath("docker"); err != nil {
		return errors.New("Docker is required for the fleetctl preview experience.\n\nPlease install Docker (https://docs.docker.com/get-docker/).")
	}
	if _, err := exec.LookPath("docker-compose"); err != nil {
		return errors.New("Docker Compose is required for the fleetctl preview experience.\n\nPlease install Docker Compose (https://docs.docker.com/compose/install/).")
	}

	// Check running
	if err := exec.Command("docker", "info").Run(); err != nil {
		return errors.New("Please start Docker daemon before running fleetctl preview.")
	}

	return nil
}

func previewStopCommand() *cli.Command {
	return &cli.Command{
		Name:  "stop",
		Usage: "Stop the Fleet preview server and dependencies",
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			if err := checkDocker(); err != nil {
				return err
			}

			previewDir := previewDirectory()
			if err := os.Chdir(previewDir); err != nil {
				return err
			}
			if _, err := os.Stat("docker-compose.yml"); err != nil {
				return fmt.Errorf("docker-compose file not found in preview directory: %w", err)
			}

			out, err := exec.Command("docker-compose", "stop").CombinedOutput()
			if err != nil {
				fmt.Println(string(out))
				return errors.New("Failed to run docker-compose stop for Fleet server and dependencies")
			}

			cmd := exec.Command("docker-compose", "stop")
			cmd.Dir = filepath.Join(previewDir, "osquery")
			cmd.Env = append(os.Environ(),
				// Note that these must be set even though they are unused while
				// stopping because docker-compose will error otherwise.
				"ENROLL_SECRET=empty",
				"FLEET_URL=empty",
			)
			out, err = cmd.CombinedOutput()
			if err != nil {
				fmt.Println(string(out))
				return errors.New("Failed to run docker-compose stop for simulated hosts")
			}

			if err := stopOrbit(previewDir); err != nil {
				return fmt.Errorf("Failed to stop orbit: %w", err)
			}

			fmt.Println("Fleet preview server and dependencies stopped. Start again with fleetctl preview.")

			return nil
		},
	}
}

func previewResetCommand() *cli.Command {
	return &cli.Command{
		Name:  "reset",
		Usage: "Reset the Fleet preview server and dependencies",
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			if err := checkDocker(); err != nil {
				return err
			}

			previewDir := previewDirectory()
			if err := os.Chdir(previewDir); err != nil {
				return err
			}
			if _, err := os.Stat("docker-compose.yml"); err != nil {
				return fmt.Errorf("docker-compose file not found in preview directory: %w", err)
			}

			out, err := exec.Command("docker-compose", "rm", "-sf").CombinedOutput()
			if err != nil {
				fmt.Println(string(out))
				return errors.New("Failed to run docker-compose rm -sf for Fleet server and dependencies.")
			}

			cmd := exec.Command("docker-compose", "rm", "-sf")
			cmd.Dir = filepath.Join(previewDir, "osquery")
			cmd.Env = append(os.Environ(),
				// Note that these must be set even though they are unused while
				// stopping because docker-compose will error otherwise.
				"ENROLL_SECRET=empty",
				"FLEET_URL=empty",
			)
			out, err = cmd.CombinedOutput()
			if err != nil {
				fmt.Println(string(out))
				return errors.New("Failed to run docker-compose rm -sf for simulated hosts.")
			}

			if err := stopOrbit(previewDir); err != nil {
				return fmt.Errorf("Failed to stop orbit: %w", err)
			}

			fmt.Println("Fleet preview server and dependencies reset. Start again with fleetctl preview.")

			return nil
		},
	}
}

func storePidFile(destDir string, pid int) error {
	pidFilePath := path.Join(destDir, "orbit.pid")
	err := os.WriteFile(pidFilePath, []byte(fmt.Sprint(pid)), os.FileMode(0644))
	if err != nil {
		return fmt.Errorf("error writing pidfile %s: %s", pidFilePath, err)
	}
	return nil
}

func readPidFromFile(destDir string, what string) (int, error) {
	pidFilePath := path.Join(destDir, what)
	data, err := os.ReadFile(pidFilePath)
	if err != nil {
		return 0, fmt.Errorf("error reading pidfile %s: %w", pidFilePath, err)
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

// processNameMatches returns whether the process running with the given pid matches
// the executable name (case insensitive).
//
// If there's no process running with the given pid then (false, nil) is returned.
func processNameMatches(pid int, expectedPrefix string) (bool, error) {
	process, err := ps.FindProcess(pid)
	if err != nil {
		return false, fmt.Errorf("find process: %d: %w", pid, err)
	}
	if process == nil {
		return false, nil
	}
	return strings.HasPrefix(strings.ToLower(process.Executable()), strings.ToLower(expectedPrefix)), nil
}

func downloadOrbitAndStart(destDir string, enrollSecret string, address string) error {
	// Stop any current intance of orbit running, otherwise the configured enroll secret
	// won't match the generated in the preview run.
	if err := stopOrbit(destDir); err != nil {
		fmt.Println("Failed to stop an existing instance of orbit running: ", err)
		return err
	}

	fmt.Println("Trying to clear orbit and osquery directories...")
	if err := os.RemoveAll(path.Join(destDir, "osquery.db")); err != nil {
		fmt.Println("Warning: clearing osquery db dir:", err)
	}
	if err := os.RemoveAll(path.Join(destDir, "orbit.db")); err != nil {
		fmt.Println("Warning: clearing orbit db dir:", err)
	}

	updateOpt := update.DefaultOptions
	switch runtime.GOOS {
	case "linux":
		updateOpt.Platform = "linux"
	case "darwin":
		updateOpt.Platform = "macos"
	case "windows":
		updateOpt.Platform = "windows"
	default:
		return fmt.Errorf("unsupported arch: %s", runtime.GOOS)
	}
	updateOpt.ServerURL = "https://tuf.fleetctl.com"
	updateOpt.RootDirectory = destDir

	if err := packaging.InitializeUpdates(updateOpt); err != nil {
		return fmt.Errorf("initialize updates: %w", err)
	}

	cmd := exec.Command(
		path.Join(destDir, "bin", "orbit", updateOpt.Platform, updateOpt.OrbitChannel, "orbit"),
		"--root-dir", destDir,
		"--fleet-url", address,
		"--insecure",
		"--debug",
		"--enroll-secret", enrollSecret,
		"--log-file", path.Join(destDir, "orbit.log"),
	)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting orbit: %w", err)
	}
	if err := storePidFile(destDir, cmd.Process.Pid); err != nil {
		return fmt.Errorf("saving pid file: %w", err)
	}

	return nil
}

func stopOrbit(destDir string) error {
	err := killFromPIDFile(destDir, "osquery.pid", "osqueryd")
	if err != nil {
		return err
	}
	err = killFromPIDFile(destDir, "orbit.pid", "orbit")
	if err != nil {
		return err
	}
	return nil
}

func killFromPIDFile(destDir string, pidFileName string, expectedExecName string) error {
	pid, err := readPidFromFile(destDir, pidFileName)
	switch {
	case err == nil:
		// OK
	case errors.Is(err, os.ErrNotExist):
		return nil // we assume it's not running
	default:
		return fmt.Errorf("reading pid from: %s: %w", destDir, err)
	}
	matches, err := processNameMatches(pid, expectedExecName)
	if err != nil {
		return fmt.Errorf("inspecting process %d: %w", pid, err)
	}
	if !matches {
		// Nothing to do, another process may be running with this pid
		// (e.g. could happen after a restart).
		return nil
	}
	if err := killPID(pid); err != nil {
		return fmt.Errorf("killing %d: %w", pid, err)
	}
	return nil
}

func loadPolicies(client *service.Client) error {
	policies := []struct {
		name, query, description, resolution, platform string
	}{
		{
			"Is Gatekeeper enabled on macOS devices?",
			"SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
			"Checks to make sure that the Gatekeeper feature is enabled on macOS devices. Gatekeeper tries to ensure only trusted software is run on a mac machine.",
			"Run the following command in the Terminal app: /usr/sbin/spctl --master-enable",
			"darwin",
		},
		{
			"Is disk encryption enabled on Windows devices?",
			"SELECT 1 FROM bitlocker_info where protection_status = 1;",
			"Checks to make sure that device encryption is enabled on Windows devices.",
			"Option 1: Select the Start button. Select Settings > Update & Security > Device encryption. If Device encryption doesn't appear, skip to Option 2. If device encryption is turned off, select Turn on. Option 2: Select the Start button. Under Windows System, select Control Panel. Select System and Security. Under BitLocker Drive Encryption, select Manage BitLocker. Select Turn on BitLocker and then follow the instructions.",
			"windows",
		},
		{
			"Is Filevault enabled on macOS devices?",
			`SELECT 1 FROM disk_encryption WHERE user_uuid IS NOT "" AND filevault_status = 'on' LIMIT 1;`,
			"Checks to make sure that the Filevault feature is enabled on macOS devices.",
			"Choose Apple menu > System Preferences, then click Security & Privacy. Click the FileVault tab. Click the Lock icon, then enter an administrator name and password. Click Turn On FileVault.",
			"darwin",
		},
	}

	for _, policy := range policies {
		err := client.CreateGlobalPolicy(policy.name, policy.query, policy.description, policy.resolution, policy.platform)
		if err != nil {
			return fmt.Errorf("creating policy: %w", err)
		}
	}

	return nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // xdg-open is available on most Linux-y systems
		cmd = exec.Command("xdg-open", url)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open in browser: %w", err)
	}
	return nil
}
