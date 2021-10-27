package main

import (
	"archive/zip"
	"bytes"
	"crypto/tls"
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
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/fleetdm/fleet/v4/orbit/pkg/packaging"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/pkg/errors"
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
				return errors.Wrap(err, "Error downloading dependencies")
			}

			if err := os.Chdir(previewDir); err != nil {
				return err
			}
			if _, err := os.Stat("docker-compose.yml"); err != nil {
				return errors.Wrap(err, "docker-compose file not found in preview directory")
			}

			// Make sure the logs directory is writable, otherwise the Fleet
			// server errors on startup. This can be a problem when running on
			// Linux with a non-root user inside the container.
			if err := os.Chmod(filepath.Join(previewDir, "logs"), 0777); err != nil {
				return errors.Wrap(err, "make logs writable")
			}
			if err := os.Chmod(filepath.Join(previewDir, "vulndb"), 0777); err != nil {
				return errors.Wrap(err, "make vulndb writable")
			}

			if err := os.Setenv("FLEET_VERSION", c.String(tagFlagName)); err != nil {
				return errors.Wrap(err, "failed to set Fleet version")
			}

			fmt.Println("Pulling Docker dependencies...")
			out, err := exec.Command("docker-compose", "pull").CombinedOutput()
			if err != nil {
				fmt.Println(string(out))
				return errors.Errorf("Failed to run docker-compose")
			}

			fmt.Println("Starting Docker containers...")
			cmd := exec.Command("docker-compose", "up", "-d", "--remove-orphans", "mysql01", "redis01", "fleet01")
			cmd.Env = append(os.Environ(), "FLEET_LICENSE_KEY="+c.String(licenseKeyFlagName))
			out, err = cmd.CombinedOutput()
			if err != nil {
				fmt.Println(string(out))
				return errors.Errorf("Failed to run docker-compose")
			}

			fmt.Println("Waiting for server to start up...")
			if err := waitStartup(); err != nil {
				return errors.Wrap(err, "wait for server startup")
			}

			// Start fleet02 (UI server) after fleet01 (agent/fleetctl server)
			// has finished starting up so that there is no conflict with
			// running database migrations.
			cmd = exec.Command("docker-compose", "up", "-d", "--remove-orphans", "fleet02")
			cmd.Env = append(os.Environ(), "FLEET_LICENSE_KEY="+c.String(licenseKeyFlagName))
			out, err = cmd.CombinedOutput()
			if err != nil {
				fmt.Println(string(out))
				return errors.Errorf("Failed to run docker-compose")
			}

			fmt.Println("Initializing server...")
			const (
				address  = "https://localhost:8412"
				email    = "admin@example.com"
				password = "admin123#"
			)

			fleetClient, err := service.NewClient(address, true, "", "")
			if err != nil {
				return errors.Wrap(err, "Error creating Fleet API client handler")
			}

			token, err := fleetClient.Setup(email, "Admin", password, "Fleet for osquery")
			if err != nil {
				switch errors.Cause(err).(type) {
				case service.SetupAlreadyErr:
					// Ignore this error
				default:
					return errors.Wrap(err, "Error setting up Fleet")
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
				return errors.Wrap(err, "Error writing fleetctl configuration")
			}

			fmt.Println("Fleet UI is now available at http://localhost:1337.")
			fmt.Println("Email:", email)
			fmt.Println("Password:", password)

			// Create client and get enroll secret
			client, err := unauthenticatedClientFromCLI(c)
			if err != nil {
				return errors.Wrap(err, "Error making fleetctl client")
			}

			token, err = client.Login(email, password)
			if err != nil {
				return errors.Wrap(err, "fleetctl login failed")
			}

			if err := setConfigValue(configPath, context, "token", token); err != nil {
				return errors.Wrap(err, "Error setting token for the current context")
			}
			client.SetToken(token)

			fmt.Println("Loading standard query library...")
			buf, err := downloadStandardQueryLibrary()
			if err != nil {
				return errors.Wrap(err, "failed to download standard query library")
			}

			specGroup, err := specGroupFromBytes(buf)
			if err != nil {
				return errors.Wrap(err, "failed to parse standard query library")
			}

			err = client.ApplyQueries(specGroup.Queries)
			if err != nil {
				return errors.Wrap(err, "failed to apply standard query library")
			}

			// disable anonymous analytics collection and enable software inventory for preview
			if err := client.ApplyAppConfig(map[string]map[string]bool{
				"host_settings":   {"enable_software_inventory": true},
				"server_settings": {"enable_analytics": false},
			}); err != nil {
				return errors.Wrap(err, "failed to apply updated app config")
			}

			fmt.Println("Applying Policies...")
			if err := loadPolicies(client); err != nil {
				fmt.Println("WARNING: Couldn't load policies:", err)
			}

			secrets, err := client.GetEnrollSecretSpec()
			if err != nil {
				return errors.Wrap(err, "Error retrieving enroll secret")
			}

			if len(secrets.Secrets) != 1 {
				return errors.New("Expected 1 active enroll secret")
			}

			// disable anonymous analytics collection for preview
			if err := client.ApplyAppConfig(map[string]map[string]bool{
				"server_settings": {"enable_analytics": false},
			},
			); err != nil {
				return errors.Wrap(err, "Error disabling anonymous analytics collection in app config")
			}

			fmt.Println("Downloading Orbit and osqueryd...")

			if err := downloadOrbitAndStart(previewDir, secrets.Secrets[0].Secret, address); err != nil {
				return errors.Wrap(err, "downloading orbit and osqueryd")
			}

			// Give it a bit of time so the current device is the one with id 1
			fmt.Println("Waiting for current host to enroll...")
			if err := waitFirstHost(client); err != nil {
				return errors.Wrap(err, "wait for current host")
			}

			fmt.Println("Starting simulated hosts...")
			cmd = exec.Command("docker-compose", "up", "-d", "--remove-orphans")
			cmd.Dir = filepath.Join(previewDir, "osquery")
			cmd.Env = append(os.Environ(),
				"ENROLL_SECRET="+secrets.Secrets[0].Secret,
				"FLEET_URL="+address,
			)
			out, err = cmd.CombinedOutput()
			if err != nil {
				fmt.Println(string(out))
				return errors.Errorf("Failed to run docker-compose")
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
		return errors.Errorf("download got status %d", resp.StatusCode)
	}

	zipContents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "read download contents")
	}

	zipReader, err := zip.NewReader(bytes.NewReader(zipContents), int64(len(zipContents)))
	if err != nil {
		return errors.Wrap(err, "open download contents for unzip")
	}
	// zip.NewReader does not need to be closed (and cannot be)

	if err := unzip(zipReader, branch); err != nil {
		return errors.Wrap(err, "unzip download contents")
	}

	return nil
}

func downloadStandardQueryLibrary() ([]byte, error) {
	resp, err := http.Get(standardQueryLibraryUrl)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("status: %d", resp.StatusCode)
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read response body")
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

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	if err := backoff.Retry(
		func() error {
			resp, err := client.Get("https://localhost:8412/healthz")
			if err != nil {
				return err
			}
			if resp.StatusCode != http.StatusOK {
				return errors.Errorf("got status code %d", resp.StatusCode)
			}
			return nil
		},
		retryStrategy,
	); err != nil {
		return errors.Wrap(err, "checking server health")
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
		return errors.Wrap(err, "checking host count")
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
				return errors.Wrap(err, "docker-compose file not found in preview directory")
			}

			out, err := exec.Command("docker-compose", "stop").CombinedOutput()
			if err != nil {
				fmt.Println(string(out))
				return errors.Errorf("Failed to run docker-compose stop for Fleet server and dependencies")
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
				return errors.Errorf("Failed to run docker-compose stop for simulated hosts")
			}

			if err := stopOrbit(previewDir); err != nil {
				return errors.Wrap(err, "Failed to stop orbit")
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
				return errors.Wrap(err, "docker-compose file not found in preview directory")
			}

			out, err := exec.Command("docker-compose", "rm", "-sf").CombinedOutput()
			if err != nil {
				fmt.Println(string(out))
				return errors.Errorf("Failed to run docker-compose rm -sf for Fleet server and dependencies.")
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
				return errors.Errorf("Failed to run docker-compose rm -sf for simulated hosts.")
			}

			if err := stopOrbit(previewDir); err != nil {
				return errors.Wrap(err, "Failed to stop orbit")
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

func readPidFromFile(destDir string) (int, error) {
	pidFilePath := path.Join(destDir, "orbit.pid")
	data, err := os.ReadFile(pidFilePath)
	if err != nil {
		return -1, fmt.Errorf("error reading pidfile %s: %s", pidFilePath, err)
	}
	return strconv.Atoi(string(data))
}

func isOrbitAlreadyRunning(destDir string) bool {
	pid, err := readPidFromFile(destDir)
	if err != nil {
		// if any error occurs reading the pid file, we assume orbit is not running
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		// if there are any errors looking for process, we assume orbit is not running
		return false
	}
	// otherwise, we found the process, so it's running
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		// Unix will always return a process for the pid, so we try sending a signal to see if it's running
		return false
	}
	return true
}

func downloadOrbitAndStart(destDir string, enrollSecret string, address string) error {
	if isOrbitAlreadyRunning(destDir) {
		fmt.Println("Orbit is already running.")
		return nil
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
		return errors.Errorf("unsupported arch: %s", runtime.GOOS)
	}
	updateOpt.ServerURL = "https://tuf.fleetctl.com"
	updateOpt.RootDirectory = destDir

	if err := packaging.InitializeUpdates(updateOpt); err != nil {
		return errors.Wrap(err, "initialize updates")
	}

	cmd := exec.Command(
		path.Join(destDir, "bin", "orbit", updateOpt.Platform, updateOpt.OrbitChannel, "orbit"),
		"--root-dir", destDir,
		"--fleet-url", address,
		"--insecure",
		"--enroll-secret", enrollSecret,
		"--log-file", path.Join(destDir, "orbit.log"),
	)
	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "starting orbit")
	}
	if err := storePidFile(destDir, cmd.Process.Pid); err != nil {
		return errors.Wrap(err, "saving pid file")
	}

	return nil
}

func stopOrbit(destDir string) error {
	pid, err := readPidFromFile(destDir)
	if err != nil {
		return errors.Wrap(err, "reading pid")
	}
	err = killPID(pid)
	if err != nil {
		return errors.Wrapf(err, "killing orbit %d", pid)
	}
	return nil
}

func loadPolicies(client *service.Client) error {
	policies := []struct {
		name, query, description, resolution string
	}{
		{
			"Is Gatekeeper enabled on macOS devices?",
			"SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
			"Checks to make sure that the Gatekeeper feature is enabled on macOS devices. Gatekeeper tries to ensure only trusted software is run on a mac machine.",
			"Run the following command in the Terminal app: /usr/sbin/spctl --master-enable",
		},
		{
			"Is disk encryption enabled on Windows devices?",
			"SELECT 1 FROM bitlocker_info where protection_status = 1;",
			"Checks to make sure that device encryption is enabled on Windows devices.",
			"Option 1: Select the Start button. Select Settings > Update & Security > Device encryption. If Device encryption doesn't appear, skip to Option 2. If device encryption is turned off, select Turn on. Option 2: Select the Start button. Under Windows System, select Control Panel. Select System and Security. Under BitLocker Drive Encryption, select Manage BitLocker. Select Turn on BitLocker and then follow the instructions.",
		},
		{
			"Is Filevault enabled on macOS devices?",
			`SELECT 1 FROM disk_encryption WHERE user_uuid IS NOT "" AND filevault_status = 'on' LIMIT 1;`,
			"Checks to make sure that the Filevault feature is enabled on macOS devices.",
			"Choose Apple menu > System Preferences, then click Security & Privacy. Click the FileVault tab. Click the Lock icon, then enter an administrator name and password. Click Turn On FileVault.",
		},
	}

	for _, policy := range policies {
		q, err := client.CreateQuery(policy.name, policy.query, policy.description)
		if err != nil {
			return errors.Wrap(err, "creating query")
		}
		err = client.CreatePolicy(q.ID, policy.resolution)
		if err != nil {
			return errors.Wrap(err, "creating policy")
		}
	}

	return nil
}
