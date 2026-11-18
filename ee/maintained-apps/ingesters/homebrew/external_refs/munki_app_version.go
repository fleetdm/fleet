package externalrefs

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

// munkiRangeClient has a short timeout and a response cap: every request it
// makes asks for a specific, small byte range, so a response any larger than
// this means something (a redirect target that ignored Range, a bad proxy)
// went wrong rather than Munki's installer growing an enormous Distribution
// file.
var munkiRangeClient = fleethttp.NewClient(
	fleethttp.WithTimeout(30*time.Second),
	fleethttp.WithMaxResponseSize(10<<20),
)

// xarHeaderSize is the fixed size, in bytes, of a XAR archive's header:
// magic (4), header size (2), version (2), compressed TOC length (8),
// uncompressed TOC length (8), checksum algorithm (4).
const xarHeaderSize = 28

// maxTOCLen caps the credible size of a xar table of contents (a real
// munkitools pkg's runs tens of KB) so a garbled or malicious header can't
// turn into a request for gigabytes of range data.
const maxTOCLen = 10 << 20 // 10MiB

var munkiManagedSoftwareCenterVersionPattern = regexp.MustCompile(
	`CFBundleShortVersionString="([^"]+)"[^>]*\bid="com\.googlecode\.munki\.ManagedSoftwareCenter"`,
)

type xarTOC struct {
	Files []xarTOCFile `xml:"toc>file"`
}

type xarTOCFile struct {
	Name string          `xml:"name"`
	Type string          `xml:"type"`
	Data *xarTOCFileData `xml:"data"`
}

type xarTOCFileData struct {
	Offset int64 `xml:"offset"`
	Length int64 `xml:"length"`
}

// MunkiAppVersionFromPkg overrides the manifest version with the version
// Managed Software Center.app actually ships, read from the real installer.
// Munki's release process doesn't always rebuild the GUI app when the
// overall munkitools version bumps (see fleetdm/fleet#59183 for version
// drift measured across several releases), so the cask-reported version
// regularly doesn't match what osquery sees post-install, which makes the
// generated patched query fail forever. The installer is a flat (xar)
// package; this reads only the header, table of contents, and Distribution
// file via HTTP range requests rather than downloading the ~20MB installer.
func MunkiAppVersionFromPkg(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	version, err := munkiManagedSoftwareCenterVersion(func(start, end int64) ([]byte, error) {
		return rangeGet(app.InstallerURL, start, end)
	})
	if err != nil {
		return app, fmt.Errorf("reading Managed Software Center.app version from %s: %w", app.InstallerURL, err)
	}

	app.Version = version
	return app, nil
}

// munkiManagedSoftwareCenterVersion walks a flat pkg's xar structure via fetch
// (a byte-range reader over the installer) to find the CFBundleShortVersionString
// Apple's installer tooling recorded for Managed Software Center.app.
func munkiManagedSoftwareCenterVersion(fetch func(start, end int64) ([]byte, error)) (string, error) {
	header, err := fetch(0, xarHeaderSize-1)
	if err != nil {
		return "", fmt.Errorf("fetching xar header: %w", err)
	}
	if len(header) != xarHeaderSize || string(header[0:4]) != "xar!" {
		return "", fmt.Errorf("unexpected xar header (got %d bytes)", len(header))
	}
	rawTOCLen := binary.BigEndian.Uint64(header[8:16])
	if rawTOCLen == 0 || rawTOCLen > maxTOCLen {
		return "", fmt.Errorf("implausible xar table of contents length: %d bytes", rawTOCLen)
	}
	tocLenCompressed := int64(rawTOCLen) //nolint:gosec // G115: bounds-checked above against maxTOCLen

	tocCompressed, err := fetch(xarHeaderSize, xarHeaderSize+tocLenCompressed-1)
	if err != nil {
		return "", fmt.Errorf("fetching xar table of contents: %w", err)
	}
	tocXML, err := zlibInflate(tocCompressed)
	if err != nil {
		return "", fmt.Errorf("inflating xar table of contents: %w", err)
	}

	var toc xarTOC
	if err := xml.Unmarshal(tocXML, &toc); err != nil {
		return "", fmt.Errorf("parsing xar table of contents: %w", err)
	}

	var dist *xarTOCFileData
	for _, f := range toc.Files {
		if f.Name == "Distribution" && f.Type == "file" && f.Data != nil {
			dist = f.Data
			break
		}
	}
	if dist == nil {
		return "", fmt.Errorf("no top-level Distribution file in xar table of contents")
	}

	heapStart := int64(xarHeaderSize) + tocLenCompressed
	distCompressed, err := fetch(heapStart+dist.Offset, heapStart+dist.Offset+dist.Length-1)
	if err != nil {
		return "", fmt.Errorf("fetching Distribution file: %w", err)
	}
	distXML, err := zlibInflate(distCompressed)
	if err != nil {
		return "", fmt.Errorf("inflating Distribution file: %w", err)
	}

	m := munkiManagedSoftwareCenterVersionPattern.FindSubmatch(distXML)
	if m == nil {
		return "", fmt.Errorf("Managed Software Center.app version not found in Distribution file")
	}

	return string(m[1]), nil
}

func rangeGet(url string, start, end int64) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	resp, err := munkiRangeClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf("server did not honor range request (status %d)", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func zlibInflate(b []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return io.ReadAll(r)
}
