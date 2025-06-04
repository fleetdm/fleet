package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func main() {
	url := flag.String("url", "", "URL of the custom package")
	path := flag.String("path", "", "File path of the custom package (or a directory of packages)")
	flag.Parse()

	switch {
	case *url != "" && *path != "":
		log.Fatalf("-url and -path are mutually exclusive")
	case *url != "":
		metadata, err := processPackageFromUrl(*url)
		if err != nil {
			log.Fatal(err)
		}
		output(metadata)
	case *path != "":
		pathInfo, err := os.Stat(*path)
		if err != nil {
			log.Fatal(err)
		}
		if pathInfo.IsDir() {
			files, err := os.ReadDir(*path)
			if err != nil {
				log.Fatal(err)
			}

			for _, selectedFile := range files {
				if selectedFile.IsDir() || strings.HasPrefix(selectedFile.Name(), ".") {
					continue
				}

				fmt.Printf("File: %s\n", selectedFile.Name())
				metadata, err := processPackageFromLocal(filepath.Join(*path, selectedFile.Name()))
				if err != nil {
					log.Fatal(err)
				}
				output(metadata)
			}
		} else {
			metadata, err := processPackageFromLocal(*path)
			if err != nil {
				log.Fatal(err)
			}
			output(metadata)
		}
	default:
		flag.Usage()
	}
}

func output(metadata *file.InstallerMetadata) {
	slices.Sort(metadata.PackageIDs)
	fmt.Printf(
		"- Name: '%s'\n- Bundle Identifier: '%s'\n- Package IDs: '%s'\n- Version: %s\n\n",
		metadata.Name, metadata.BundleIdentifier, strings.Join(metadata.PackageIDs, ","), metadata.Version,
	)
}

func processPackageFromUrl(url string) (*file.InstallerMetadata, error) {
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

	tfr, err := fleet.NewTempFileReader(resp.Body, nil)
	if err != nil {
		return nil, fmt.Errorf("reading custom package: %d", resp.StatusCode)
	}
	defer tfr.Close()

	metadata, err := file.ExtractInstallerMetadata(tfr)
	if err != nil {
		return nil, fmt.Errorf("extract installer metadata: %s", err)
	}
	return metadata, nil
}

func processPackageFromLocal(path string) (*file.InstallerMetadata, error) {
	fp, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %s", err)
	}
	tfr := &fleet.TempFileReader{
		File: fp,
	}

	metadata, err := file.ExtractInstallerMetadata(tfr)
	if err != nil {
		return nil, fmt.Errorf("extract installer metadata: %s", err)
	}
	return metadata, nil
}
