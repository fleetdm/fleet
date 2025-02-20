package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func main() {
	url := flag.String("url", "", "URL of the custom package")
	path := flag.String("path", "", "File path of the custom package")
	flag.Parse()

	if *url == "" && *path == "" {
		log.Fatal("missing -url or -path argument")
	}
	if *url != "" && *path != "" {
		log.Fatal("cannot set both -url and -path")
	}

	metadata, err := processPackage(*url, *path)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf(
		"- Name: '%s'\n- Bundle Identifier: '%s'\n- Package IDs: '%s'\n- Version: %s\n\n",
		metadata.Name, metadata.BundleIdentifier, strings.Join(metadata.PackageIDs, ","), metadata.Version,
	)
}

func processPackage(url, path string) (*file.InstallerMetadata, error) {
	var tfr *fleet.TempFileReader
	if url != "" {
		client := fleethttp.NewClient()
		client.Transport = fleethttp.NewSizeLimitTransport(fleet.MaxSoftwareInstallerSize)

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("create http request: %s", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("get request: %s", err)
		}
		defer resp.Body.Close()

		// Allow all 2xx and 3xx status codes in this pass.
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("get request failed with status: %d", resp.StatusCode)
		}

		tfr, err = fleet.NewTempFileReader(resp.Body, nil)
		if err != nil {
			return nil, fmt.Errorf("reading custom package: %d", resp.StatusCode)
		}
		defer tfr.Close()
	} else { // -path
		fp, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open file: %s", err)
		}
		tfr = &fleet.TempFileReader{
			File: fp,
		}
	}

	metadata, err := file.ExtractInstallerMetadata(tfr)
	if err != nil {
		return nil, fmt.Errorf("extract installer metadata: %s", err)
	}
	return metadata, nil
}
