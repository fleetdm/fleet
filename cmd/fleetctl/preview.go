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

	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const (
	previewDirectory = "fleet-preview"
	downloadUrl      = "https://github.com/fleetdm/osquery-in-a-box/archive/master.zip"
)

func previewCommand() cli.Command {
	return cli.Command{
		Name:  "preview",
		Usage: "Set up a preview deployment of the Fleet server",
		UsageText: `Set up a preview deployment of the Fleet server using Docker and docker-compose. Docker tools must be available in the environment.

This command will create a directory fleet-preview in the current working directory. Configurations can be modified in that directory.`,
		Subcommands: []cli.Command{},
		Action: func(c *cli.Context) error {
			if _, err := exec.LookPath("docker-compose"); err != nil {
				return errors.New("Please install Docker (https://docs.docker.com/get-docker/).")
			}

			// Download files if necessary
			if _, err := os.Stat(
				filepath.Join(previewDirectory, "docker-compose.yml"),
			); err != nil {
				fmt.Println("Downloading dependencies into", previewDirectory)
				if err := downloadFiles(); err != nil {
					return errors.Wrap(err, "Error downloading dependencies")
				}
			}

			if err := os.Chdir(previewDirectory); err != nil {
				return err
			}

			fmt.Println("Starting Docker containers...")
			out, err := exec.Command("docker-compose", "up", "-d", "mysql01", "redis01", "fleet01").CombinedOutput()
			if err != nil {
				fmt.Println(string(out))
				return errors.Errorf("Failed to run docker-compose")
			}

			if err := waitStartup(); err != nil {
				return errors.Wrap(err, "wait for server startup")
			}

			fmt.Println("Fleet is now available at https://localhost:8412.")
			fmt.Println("Note: You can safely ignore the browser warning \"Your connection is not private\". Click through this warning using the \"Advanced\" option.")

			return nil
		},
	}
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
	// Closure to address file descriptors issue with all the deferred .Close()
	// methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		path := f.Name
		path = strings.Replace(path, "osquery-in-a-box-master", "fleet-preview", 1)

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
	fmt.Println("Waiting for server to start up...")
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
		backoff.NewExponentialBackOff(),
	); err != nil {
		return errors.Wrap(err, "checking server health")
	}

	return nil
}
