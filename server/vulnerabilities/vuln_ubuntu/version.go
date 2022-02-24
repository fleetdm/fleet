package vuln_ubuntu

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Version represents a package version (http://man.he.net/man5/deb-version).
type Version struct {
	Epoch           int
	UpstreamVersion string
	DebianRevision  string
}

// NewVersion returns a parsed version
func NewVersion(ver string) (Version, error) {
	ver = strings.TrimSpace(ver)

	var version Version

	// Parse epoch
	splitted := strings.SplitN(ver, ":", 2)
	if len(splitted) == 1 {
		version.Epoch = 0
		ver = splitted[0]
	} else {
		var err error
		version.Epoch, err = strconv.Atoi(splitted[0])
		if err != nil {
			return Version{}, fmt.Errorf("epoch parse error: %v", err)
		}

		if version.Epoch < 0 {
			return Version{}, errors.New("epoch is negative")
		}
		ver = splitted[1]
	}

	// Parse upstream_version and debian_revision
	index := strings.LastIndex(ver, "-")
	if index >= 0 {
		version.UpstreamVersion = ver[:index]
		version.DebianRevision = ver[index+1:]
	} else {
		version.UpstreamVersion = ver
	}

	return version, nil
}
