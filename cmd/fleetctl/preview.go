package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/packaging"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/open"
	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/google/go-github/v37/github"
	"github.com/mitchellh/go-ps"
	"github.com/urfave/cli/v2"
)

type dockerComposeVersion int

const (
	standardQueryLibraryUrl   = "https://raw.githubusercontent.com/fleetdm/fleet/main/docs/01-Using-Fleet/standard-query-library/standard-query-library.yml"
	licenseKeyFlagName        = "license-key"
	tagFlagName               = "tag"
	previewConfigFlagName     = "preview-config"
	noHostsFlagName           = "no-hosts"
	orbitChannel              = "orbit-channel"
	osquerydChannel           = "osqueryd-channel"
	updateURL                 = "update-url"
	updateRootKeys            = "update-roots"
	stdQueryLibFilePath       = "std-query-lib-file-path"
	previewConfigPathFlagName = "preview-config-path"
	disableOpenBrowser        = "disable-open-browser"

	dockerComposeV1 dockerComposeVersion = 1
	dockerComposeV2 dockerComposeVersion = 2
)

type dockerCompose struct {
	version dockerComposeVersion
}

func (d dockerCompose) String() string {
	if d.version == dockerComposeV1 {
		return "`docker-compose`"
	}

	return "`docker compose`"
}

func (d dockerCompose) Command(arg ...string) *exec.Cmd {
	if d.version == dockerComposeV1 {
		return exec.Command("docker-compose", arg...)
	}

	return exec.Command("docker", append([]string{"compose"}, arg...)...)
}

func newDockerCompose() (dockerCompose, error) {
	// first, check if `docker compose` is available
	if err := exec.Command("docker", "compose").Run(); err == nil {
		return dockerCompose{dockerComposeV2}, nil
	}

	// if not, try to use `docker-compose`
	if _, err := exec.LookPath("docker-compose"); err == nil {
		return dockerCompose{dockerComposeV1}, nil
	}

	return dockerCompose{}, errors.New("`docker compose` is required for the fleetctl preview experience.\n\nPlease install `docker compose` (https://docs.docker.com/compose/install/).")
}

func previewCommand() *cli.Command {
	return &cli.Command{
		Name:    "preview",
		Aliases: []string{"sandbox"},
		Usage:   "Start a sandbox deployment of the Fleet server",
		Description: `Start a sandbox deployment of the Fleet server using Docker and docker compose. Docker tools must be available in the environment.

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
				Value: "main",
			},
			&cli.BoolFlag{
				Name:  noHostsFlagName,
				Usage: "Start the server without adding any hosts",
				Value: false,
			},
			&cli.StringFlag{
				Name:  orbitChannel,
				Usage: "Use a custom orbit channel",
				Value: "stable",
			},
			&cli.StringFlag{
				Name:  osquerydChannel,
				Usage: "Use a custom osqueryd channel",
				Value: "stable",
			},
			&cli.StringFlag{
				Name:  updateURL,
				Usage: "Use a custom update TUF URL",
				Value: "",
			},
			&cli.StringFlag{
				Name:  updateRootKeys,
				Usage: "Use custom update TUF root keys",
				Value: "",
			},
			&cli.StringFlag{
				Name:  stdQueryLibFilePath,
				Usage: "Use custom standard query library yml file (used for development/testing)",
				Value: "",
			},
			&cli.StringFlag{
				Name:   previewConfigPathFlagName,
				Usage:  "Use custom local directory for the pulling configuration from fleetdm/fleet (used for development/testing)",
				Value:  "",
				Hidden: true,
			},
			&cli.BoolFlag{
				Name:  disableOpenBrowser,
				Usage: "Disable opening the browser",
			},
		},
		Action: func(c *cli.Context) error {
			if err := checkDocker(); err != nil {
				return err
			}

			compose, err := newDockerCompose()
			if err != nil {
				return err
			}

			previewDir := previewDirectory()

			if previewConfigDir := c.String(previewConfigPathFlagName); previewConfigDir != "" {
				fmt.Printf("Copying dependencies from %s...\n", previewConfigDir)
				if err := os.MkdirAll(previewDir, 0o777); err != nil {
					return fmt.Errorf("create directory %q: %w", previewDir, err)
				}
				if err := copyDirectory(previewDir, previewConfigDir); err != nil {
					return fmt.Errorf("copying directories: %w", err)
				}
			} else {
				// Download files every time to ensure the user gets the most up to date versions
				fleetBranch := c.String(previewConfigFlagName)
				fmt.Printf("Downloading dependencies from fleetdm/fleet:%s into %s...\n", fleetBranch, previewDir)
				githubHTTPClient := fleethttp.NewGithubClient()
				githubClient := github.NewClient(githubHTTPClient)
				if err := downloadFromFleetRepo(
					context.Background(),
					githubHTTPClient,
					githubClient,
					"tools/osquery/in-a-box",
					fleetBranch,
					previewDirectory(),
				); err != nil {
					return fmt.Errorf("downloading dependencies: %w", err)
				}
			}

			if err := os.Chdir(previewDir); err != nil {
				return err
			}
			if _, err := os.Stat("docker-compose.yml"); err != nil {
				return fmt.Errorf("docker-compose file not found in preview directory: %w", err)
			}

			// Make sure directories are writable, otherwise the Fleet
			// server errors on startup. This can be a problem when running on
			// Linux with a non-root user inside the container.
			for _, item := range []string{
				filepath.Join(previewDir, "logs"),
				filepath.Join(previewDir, "vulndb"),
				filepath.Join(previewDir, "config"),
			} {
				if err := os.MkdirAll(item, 0o777); err != nil {
					return fmt.Errorf("create directory %q: %w", item, err)
				}
				if err := os.Chmod(item, 0o777); err != nil {
					return fmt.Errorf("make %s writable: %w", item, err)
				}
			}

			generatePrivateKey := func(n int) (string, error) {
				bytes := make([]byte, n/2)
				if _, err := rand.Read(bytes); err != nil {
					return "", err
				}
				return hex.EncodeToString(bytes)[:n], nil
			}

			// Create a random private key for MDM asset encryption and save it to the filesystem
			// for use in subsequent runs. If one already exists, use that one.
			getPrivateKey := func() (string, error) {
				pkFilename := filepath.Join(previewDir, "config", ".private_key")
				filePK, err := os.ReadFile(pkFilename)
				if err != nil {
					if errors.Is(err, os.ErrNotExist) {
						genPK, err := generatePrivateKey(32) // use AES-256
						if err != nil {
							return "", fmt.Errorf("generating private key: %w", err)
						}

						if err := os.WriteFile(pkFilename, []byte(genPK), 0o777); err != nil {
							return "", fmt.Errorf("writing private key file: %w", err)
						}

						return genPK, nil
					}

					return "", fmt.Errorf("reading private key file: %w", err)
				}

				return string(filePK), nil
			}

			pk, err := getPrivateKey()
			if err != nil {
				return fmt.Errorf("getting private key: %w", err)
			}

			if err := os.Setenv("FLEET_SERVER_PRIVATE_KEY", pk); err != nil {
				return fmt.Errorf("failed to set private key: %w", err)
			}

			if err := os.Setenv("FLEET_VERSION", c.String(tagFlagName)); err != nil {
				return fmt.Errorf("failed to set Fleet version: %w", err)
			}

			fmt.Println("Pulling Docker dependencies...")
			out, err := compose.Command("pull").CombinedOutput()
			if err != nil {
				fmt.Println(string(out))
				return fmt.Errorf("Failed to run %s", compose)
			}

			fmt.Println("Starting Docker containers...")
			cmd := compose.Command("up", "-d", "--remove-orphans", "mysql01", "redis01", "fleet01")
			cmd.Env = append(os.Environ(), "FLEET_LICENSE_KEY="+c.String(licenseKeyFlagName))
			out, err = cmd.CombinedOutput()
			if err != nil {
				fmt.Println(string(out))
				return fmt.Errorf("Failed to run %s", compose)
			}

			fmt.Println("Waiting for server to start up...")
			if err := waitStartup(); err != nil {
				return fmt.Errorf("wait for server startup: %w", err)
			}

			// Start fleet02 (UI server) after fleet01 (agent/fleetctl server)
			// has finished starting up so that there is no conflict with
			// running database migrations.
			cmd = compose.Command("up", "-d", "--remove-orphans", "fleet02")
			cmd.Env = append(os.Environ(), "FLEET_LICENSE_KEY="+c.String(licenseKeyFlagName))
			out, err = cmd.CombinedOutput()
			if err != nil {
				fmt.Println(string(out))
				return fmt.Errorf("Failed to run %s", compose)
			}

			fmt.Println("Initializing server...")
			const (
				address  = "https://localhost:8412"
				email    = "admin@example.com"
				password = "preview1337#"
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
			if err := c.Set("context", context); err != nil {
				return fmt.Errorf("Error setting context: %w", err)
			}

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
			var buf []byte
			if fp := c.String(stdQueryLibFilePath); fp != "" {
				var err error
				buf, err = os.ReadFile(fp)
				if err != nil {
					return fmt.Errorf("failed to read standard query library file %q: %w", fp, err)
				}
			} else {
				var err error
				buf, err = downloadStandardQueryLibrary()
				if err != nil {
					return fmt.Errorf("failed to download standard query library: %w", err)
				}
			}

			specs, err := spec.GroupFromBytes(buf)
			if err != nil {
				return err
			}
			logf := func(format string, a ...interface{}) {
				fmt.Fprintf(c.App.Writer, format, a...)
			}
			// this only applies standard queries, the base directory is not used,
			// so pass in the current working directory.
			teamsSoftwareInstallers := make(map[string][]fleet.SoftwarePackageResponse)
			teamsScripts := make(map[string][]fleet.ScriptResponse)
			_, _, _, err = client.ApplyGroup(c.Context, false, specs, ".", logf, nil, fleet.ApplyClientSpecOptions{}, teamsSoftwareInstallers, teamsScripts)
			if err != nil {
				return err
			}

			// disable analytics collection and enable software inventory for preview
			// TODO(roperzh): replace `host_settings` with `features` once the
			// Docker image used for preview (fleetdm/fleetctl:latest) is released
			if err := client.ApplyAppConfig(map[string]map[string]bool{
				"host_settings":   {"enable_software_inventory": true},
				"server_settings": {"enable_analytics": false},
			}, fleet.ApplySpecOptions{}); err != nil {
				return fmt.Errorf("failed to apply updated app config: %w", err)
			}

			secrets, err := client.GetEnrollSecretSpec()
			if err != nil {
				return fmt.Errorf("Error retrieving enroll secret: %w", err)
			}

			if len(secrets.Secrets) == 0 {
				return errors.New("Expected at least one enroll secret")
			}

			// disable analytics collection for preview
			if err := client.ApplyAppConfig(map[string]map[string]bool{
				"server_settings": {"enable_analytics": false},
			}, fleet.ApplySpecOptions{}); err != nil {
				return fmt.Errorf("Error disabling analytics collection in app config: %w", err)
			}

			fmt.Println("Fleet will now log you into the UI automatically.")
			fmt.Println("You can also open the UI at this URL: http://localhost:1337/previewlogin.")
			fmt.Println("Email:", email)
			fmt.Println("Password:", password)

			if !c.Bool(noHostsFlagName) {
				fmt.Println("Enrolling local host...")

				orbitDir := filepath.Join(previewDir, "orbit")
				if err := downloadOrbitAndStart(orbitDir, secrets.Secrets[0].Secret, address, c.String(orbitChannel), c.String(osquerydChannel), c.String(updateURL), c.String(updateRootKeys)); err != nil {
					return fmt.Errorf("downloading orbit and osqueryd: %w", err)
				}

				// Give it a bit of time so the current device is the one with id 1
				fmt.Println("Waiting for host to enroll...")
				if err := waitFirstHost(client); err != nil {
					return fmt.Errorf("wait for current host: %w", err)
				}

				if !c.Bool(disableOpenBrowser) {
					if err := open.Browser("http://localhost:1337/previewlogin"); err != nil {
						fmt.Println("Automatic browser open failed. Please navigate to http://localhost:1337/previewlogin.")
					}
				}

				fmt.Println("Starting simulated Linux hosts...")
				cmd = compose.Command("up", "-d", "--remove-orphans")
				cmd.Dir = filepath.Join(previewDir, "osquery")
				cmd.Env = append(os.Environ(),
					"ENROLL_SECRET="+secrets.Secrets[0].Secret,
					"FLEET_URL="+address,
				)
				out, err = cmd.CombinedOutput()
				if err != nil {
					fmt.Println(string(out))
					return fmt.Errorf("Failed to run %s", compose)
				}
			} else if !c.Bool(disableOpenBrowser) {
				if err := open.Browser("http://localhost:1337/previewlogin"); err != nil {
					fmt.Println("Automatic browser open failed. Please navigate to http://localhost:1337/previewlogin.")
				}
			}

			fmt.Println("Preview environment complete. Enjoy using Fleet!")

			return nil
		},
	}
}

func copyDirectory(destDir, sourceDir string) error {
	sourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		return fmt.Errorf("absolute path could not be determined for %q: %w", sourceDir, err)
	}
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		relPath := strings.Replace(path, sourceDir, "", 1)
		if relPath == "" {
			return nil
		}
		if info.IsDir() {
			return os.MkdirAll(filepath.Join(destDir, relPath), 0o777)
		}
		data, err1 := os.ReadFile(filepath.Join(sourceDir, relPath))
		if err1 != nil {
			return err1
		}
		return os.WriteFile(filepath.Join(destDir, relPath), data, 0o777)
	})
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

// downloadFromFleetRepo downloads a specific directory from the fleetdm/fleet repository.
// It creates files and folders as world writable as this is mainly used for testing.
func downloadFromFleetRepo(
	ctx context.Context,
	client *http.Client,
	githubClient *github.Client,
	repoDirectory string,
	branch string,
	outputDirectory string,
) error {
	if err := os.MkdirAll(outputDirectory, 0o777); err != nil {
		return fmt.Errorf("create directory %q: %w", outputDirectory, err)
	}

	_, contents, response, err := githubClient.Repositories.GetContents(ctx,
		"fleetdm",
		"fleet",
		repoDirectory,
		&github.RepositoryContentGetOptions{
			Ref: branch,
		},
	)
	if err != nil {
		return fmt.Errorf("get contents request: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed: %s", response.Status)
	}
	for _, content := range contents {
		if content.GetType() == "file" {
			if downloadURL := content.GetDownloadURL(); downloadURL != "" {
				response, err := client.Get(downloadURL)
				if err != nil {
					return fmt.Errorf("download file: %q: %w", downloadURL, err)
				}
				defer response.Body.Close()
				if response.StatusCode != http.StatusOK {
					return fmt.Errorf("download file response: %q: %s", downloadURL, response.Status)
				}
				b, err := io.ReadAll(response.Body)
				if err != nil {
					return fmt.Errorf("read file body: %w", err)
				}
				if err := os.WriteFile(filepath.Join(outputDirectory, content.GetName()), b, 0o777); err != nil {
					return fmt.Errorf("write file: %w", err)
				}
			}
		} else if content.GetType() == "dir" {
			return downloadFromFleetRepo(ctx,
				client,
				githubClient,
				path.Join(repoDirectory, content.GetName()),
				branch,
				filepath.Join(outputDirectory, content.GetName()),
			)
		}
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
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	return buf, nil
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

			compose, err := newDockerCompose()
			if err != nil {
				return err
			}

			previewDir := previewDirectory()
			if err := os.Chdir(previewDir); err != nil {
				return err
			}
			if _, err := os.Stat("docker-compose.yml"); err != nil {
				return fmt.Errorf("docker-compose file not found in preview directory: %w", err)
			}

			out, err := compose.Command("stop").CombinedOutput()
			if err != nil {
				fmt.Println(string(out))
				return fmt.Errorf("Failed to run %s stop for Fleet server and dependencies", compose)
			}

			cmd := compose.Command("stop")
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
				return fmt.Errorf("Failed to run %d stop for simulated hosts", compose)
			}

			orbitDir := filepath.Join(previewDir, "orbit")
			if err := stopOrbit(orbitDir); err != nil {
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

			compose, err := newDockerCompose()
			if err != nil {
				return err
			}

			previewDir := previewDirectory()
			if err := os.Chdir(previewDir); err != nil {
				return err
			}
			if _, err := os.Stat("docker-compose.yml"); err != nil {
				return fmt.Errorf("docker-compose file not found in preview directory: %w", err)
			}

			out, err := compose.Command("rm", "-sf").CombinedOutput()
			if err != nil {
				fmt.Println(string(out))
				return fmt.Errorf("Failed to run %s rm -sf for Fleet server and dependencies.", compose)
			}

			cmd := compose.Command("rm", "-sf")
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
				return fmt.Errorf("Failed to run %s rm -sf for simulated hosts.", compose)
			}

			orbitDir := filepath.Join(previewDir, "orbit")
			if err := stopOrbit(orbitDir); err != nil {
				return fmt.Errorf("Failed to stop orbit: %w", err)
			}

			if err := os.RemoveAll(filepath.Join(orbitDir, update.MetadataFileName)); err != nil {
				return fmt.Errorf("failed to remove preview update metadata file: %w", err)
			}
			if err := os.RemoveAll(filepath.Join(orbitDir, "bin")); err != nil {
				return fmt.Errorf("failed to remove preview bin directory: %w", err)
			}

			fmt.Println("Fleet preview server and dependencies reset. Start again with fleetctl preview.")

			return nil
		},
	}
}

func storePidFile(destDir string, pid int) error {
	pidFilePath := filepath.Join(destDir, "orbit.pid")
	err := os.WriteFile(pidFilePath, []byte(fmt.Sprint(pid)), os.FileMode(0o644))
	if err != nil {
		return fmt.Errorf("error writing pidfile %s: %s", pidFilePath, err)
	}
	return nil
}

func readPidFromFile(destDir string, what string) (int, error) {
	pidFilePath := filepath.Join(destDir, what)
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

func downloadOrbitAndStart(destDir, enrollSecret, address, orbitChannel, osquerydChannel, updateURL, updateRoots string) error {
	if err := os.MkdirAll(destDir, constant.DefaultDirMode); err != nil {
		return fmt.Errorf("create orbit directory %q: %w", destDir, err)
	}

	// Stop any current intance of orbit running, otherwise the configured enroll secret
	// won't match the generated in the preview run.
	if err := stopOrbit(destDir); err != nil {
		fmt.Println("Failed to stop an existing instance of orbit running: ", err)
		return err
	}

	fmt.Println("Trying to clear orbit and osquery directories...")
	if err := os.RemoveAll(filepath.Join(destDir, "osquery.db")); err != nil {
		fmt.Println("Warning: clearing osquery db dir:", err)
	}
	if err := os.RemoveAll(filepath.Join(destDir, "secret-orbit-node-key.txt")); err != nil {
		fmt.Println("Warning: clearing orbit db dir:", err)
	}
	if err := cleanUpSocketFiles(destDir); err != nil {
		fmt.Println("Warning: cleaning up socket files:", err)
	}

	updateOpt := update.DefaultOptions

	// Override default channels with the provided values.
	updateOpt.Targets.SetTargetChannel(constant.OrbitTUFTargetName, orbitChannel)
	updateOpt.Targets.SetTargetChannel(constant.OsqueryTUFTargetName, osquerydChannel)

	updateOpt.RootDirectory = destDir

	if updateURL != "" {
		updateOpt.ServerURL = updateURL
	}
	if updateRoots != "" {
		updateOpt.RootKeys = updateRoots
	}

	if _, err := packaging.InitializeUpdates(updateOpt); err != nil {
		return fmt.Errorf("initialize updates: %w", err)
	}

	orbitPath, err := update.NewDisabled(updateOpt).ExecutableLocalPath(constant.OrbitTUFTargetName)
	if err != nil {
		return fmt.Errorf("failed to locate executable for %s: %w", constant.OrbitTUFTargetName, err)
	}

	cmd := exec.Command(orbitPath,
		"--root-dir", destDir,
		"--fleet-url", address,
		"--insecure",
		"--debug",
		"--enroll-secret", enrollSecret,
		"--orbit-channel", orbitChannel,
		"--osqueryd-channel", osquerydChannel,
		"--log-file", filepath.Join(destDir, "orbit.log"),
	)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting orbit: %w", err)
	}
	if err := storePidFile(destDir, cmd.Process.Pid); err != nil {
		return fmt.Errorf("saving pid file: %w", err)
	}

	return nil
}

// cleanUpSocketFiles cleans up fleet-osqueryd's socket file
// ("orbit-osquery.em") and osquery extension socket files
// ("orbit-osquery.em.*").
func cleanUpSocketFiles(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("read dir: %w", err)
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "orbit-osquery.em") {
			continue
		}
		entryPath := filepath.Join(path, entry.Name())
		if err := os.Remove(entryPath); err != nil {
			return fmt.Errorf("remove %q: %w", entryPath, err)
		}
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
