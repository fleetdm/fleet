package oval

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	ovalSourcesFileName = "oval_sources.json"
)

// OvalSources represents a platform => web url dictionary
type OvalSources map[Platform]string

// getOvalSources gets the 'oval sources' file.
// The 'oval sources' is a metadata file hosted in the NVD repo, it contains
// where to find the OVAL definitions for a given platform.
func getOvalSources(getter func(string) (io.ReadCloser, error)) (OvalSources, error) {
	src, err := getter(ovalSourcesFileName)
	if err != nil {
		return nil, err
	}
	defer src.Close()

	contents, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}

	sources := make(OvalSources)
	err = json.Unmarshal(contents, &sources)
	if err != nil {
		return nil, err
	}

	return sources, nil
}

// downloadDefinitions downloads the OVAL definitions for a given 'platform-major os version'.
// Returns the filepath to the downloaded oval definitions.
func downloadDefinitions(
	sources OvalSources,
	platform Platform,
	downloader func(string, string) error,
) (string, error) {
	url, ok := sources[platform]
	if !ok {
		return "", fmt.Errorf("could not find platform %s on oval sources", platform)
	}

	dstPath := filepath.Join(os.TempDir(), platform.ToFilename(time.Now(), "xml"))
	err := downloader(url, dstPath)
	if err != nil {
		return "", fmt.Errorf("download definitions: %w", err)
	}

	return dstPath, nil
}
