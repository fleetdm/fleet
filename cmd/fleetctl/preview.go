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
	"github.com/fleetdm/fleet/server/service"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

const (
	downloadUrl = "https://github.com/fleetdm/osquery-in-a-box/archive/master.zip"
)

func previewCommand() *cli.Command {
	return &cli.Command{
		Name:  "preview",
		Usage: "Set up a preview deployment of the Fleet server",
		UsageText: `Set up a preview deployment of the Fleet server using Docker and docker-compose. Docker tools must be available in the environment.

This command will create a directory fleet-preview in the current working directory. Configurations can be modified in that directory.`,
		Subcommands: []*cli.Command{},
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			if err := checkDocker(); err != nil {
				return err
			}

			// Download files if necessary
			previewDir := previewDirectory()
			if _, err := os.Stat(
				filepath.Join(previewDir, "docker-compose.yml"),
			); err != nil {
				fmt.Printf("Downloading dependencies into %s...\n", previewDir)
				if err := downloadFiles(); err != nil {
					return errors.Wrap(err, "Error downloading dependencies")
				}
			}

			if err := os.Chdir(previewDir); err != nil {
				return err
			}

			// Make sure the logs directory is writable, otherwise the Fleet
			// server errors on startup. This can be a problem when running on
			// Linux with a non-root user inside the container.
			if err := os.Chmod(filepath.Join(previewDir, "logs"), 0777); err != nil {
				return errors.Wrap(err, "make logs writable")
			}

			fmt.Println("Starting Docker containers...")
			out, err := exec.Command("docker-compose", "up", "-d", "--remove-orphans", "mysql01", "redis01", "fleet01").CombinedOutput()
			if err != nil {
				fmt.Println(string(out))
				return errors.Errorf("Failed to run docker-compose")
			}

			fmt.Println("Waiting for server to start up...")
			if err := waitStartup(); err != nil {
				return errors.Wrap(err, "wait for server startup")
			}

			fmt.Println("Initializing server...")
			const (
				address  = "https://localhost:8412"
				username = "admin"
				password = "admin123#"
			)

			fleet, err := service.NewClient(address, true, "", "")
			if err != nil {
				return errors.Wrap(err, "Error creating Fleet API client handler")
			}

			token, err := fleet.Setup(username, username, password, "Fleet Preview")
			if err != nil {
				if e, ok := err.(service.SetupAlreadyErr); !(ok && e.SetupAlready()) {
					return errors.Wrap(err, "Error setting up Fleet")
				}
			}

			configPath, context := c.String("config"), "default"

			contextConfig := Context{
				Address:       address,
				Email:         username,
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

			fmt.Println("Fleet is now available at https://localhost:8412.")
			fmt.Println("Username:", username)
			fmt.Println("Password:", password)
			fmt.Println("Note: You can safely ignore the browser warning \"Your connection is not private\". Click through this warning using the \"Advanced\" option. Chrome users may need to configure chrome://flags/#allow-insecure-localhost.")

			// Create client and get enroll secret
			client, err := unauthenticatedClientFromCLI(c)
			if err != nil {
				return errors.Wrap(err, "Error making fleetctl client")
			}

			token, err = client.Login(username, password)
			if err != nil {
				return errors.Wrap(err, "fleetctl login failed")
			}

			if err := setConfigValue(configPath, context, "token", token); err != nil {
				return errors.Wrap(err, "Error setting token for the current context")
			}
			client.SetToken(token)

			secrets, err := client.GetEnrollSecretSpec()
			if err != nil {
				return errors.Wrap(err, "Error retrieving enroll secret")
			}

			if len(secrets.Secrets) != 1 || !secrets.Secrets[0].Active {
				return errors.New("Expected 1 active enroll secret")
			}

			fmt.Println("Starting simulated hosts...")
			cmd := exec.Command("docker-compose", "up", "-d", "--remove-orphans")
			cmd.Dir = filepath.Join(previewDir, "osquery")
			cmd.Env = append(cmd.Env,
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
