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
	"path/filepath"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

const (
	downloadUrl             = "https://github.com/fleetdm/osquery-in-a-box/archive/master.zip"
	standardQueryLibraryUrl = "https://raw.githubusercontent.com/fleetdm/fleet/main/docs/1-Using-Fleet/standard-query-library/standard-query-library.yml"
	licenseKeyFlagName      = "license-key"
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
				Usage: "License key to enable Fleet Basic (optional)",
			},
		},
		Action: func(c *cli.Context) error {
			if err := checkDocker(); err != nil {
				return err
			}

			// Download files every time to ensure the user gets the most up to date versions
			previewDir := previewDirectory()
			fmt.Printf("Downloading dependencies into %s...\n", previewDir)
			if err := downloadFiles(); err != nil {
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

			token, err := fleetClient.Setup(email, "Admin", password, "Fleet Preview")
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

			secrets, err := client.GetEnrollSecretSpec()
			if err != nil {
				return errors.Wrap(err, "Error retrieving enroll secret")
			}

			if len(secrets.Secrets) != 1 {
				return errors.New("Expected 1 active enroll secret")
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

func previewDirectory() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "~"
	}
	return filepath.Join(homeDir, ".fleet", "preview")
}

func downloadFiles() error {
	resp, err := http.Get(downloadUrl)
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

	if err := unzip(zipReader); err != nil {
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
func unzip(r *zip.Reader) error {
	previewDir := previewDirectory()

	// Closure to address file descriptors issue with all the deferred .Close()
	// methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		path := f.Name
		path = strings.Replace(path, "osquery-in-a-box-master", previewDir, 1)

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

			fmt.Println("Fleet preview server and dependencies reset. Start again with fleetctl preview.")

			return nil
		},
	}
}
