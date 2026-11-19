package externalrefs

import (
	"bytes"
	"cmp"
	"compress/zlib"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

// xarHeaderSize is the fixed size, in bytes, of a XAR archive's header:
// magic (4), header size (2), version (2), compressed TOC length (8),
// uncompressed TOC length (8), checksum algorithm (4).
const xarHeaderSize = 28

// maxTOCLen caps the credible size of a xar table of contents (a real
// munkitools pkg's runs tens of KB) so a garbled or malicious header can't
// turn into a request for gigabytes of range data.
const maxTOCLen = 10 << 20 // 10MiB

// maxDecompressedLen caps how much any single zlib stream in the pkg (the
// table of contents or the Distribution file) is allowed to inflate to. Both
// are small XML documents in every real munkitools release; this exists so a
// corrupt or adversarial stream can't be used as a decompression bomb.
const maxDecompressedLen = 10 << 20 // 10MiB

// munkiCorePackageID is the component rebuilt on every munkitools release, so
// its receipt version tells consecutive releases apart.
const munkiCorePackageID = "com.googlecode.munki.core"

// munkiAppBundleID is Managed Software Center.app, the bundle Fleet inventories.
// It isn't rebuilt on every release, so its version can repeat across releases.
const munkiAppBundleID = "com.googlecode.munki.ManagedSoftwareCenter"

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

// distributionPkgRefs is a minimal parse of a flat pkg's Distribution file:
// each top-level <pkg-ref id="..." version="...">, plus the app bundles listed
// in its <bundle-version>. The same id appears several times in a real
// Distribution, so callers select by id and a non-empty value rather than by
// position.
type distributionPkgRefs struct {
	PkgRefs []struct {
		ID      string `xml:"id,attr"`
		Version string `xml:"version,attr"`
		Bundles []struct {
			ID                 string `xml:"id,attr"`
			ShortVersionString string `xml:"CFBundleShortVersionString,attr"`
			BundleVersion      string `xml:"CFBundleVersion,attr"`
		} `xml:"bundle-version>bundle"`
	} `xml:"pkg-ref"`
}

// MunkiVersions are the versions a munkitools installer declares.
type MunkiVersions struct {
	// Core is the munkitools_core receipt version, which package_receipts reports.
	Core string
	// App is Managed Software Center.app's version as the apps table (and Fleet's
	// software inventory) reports it: CFBundleShortVersionString, else CFBundleVersion.
	App string
}

// MunkiPackageVersions reads the core receipt and app versions straight from the
// installer at installerURL, via small HTTP range requests against the pkg's xar
// structure (header, table of contents, and the Distribution file) rather than
// downloading the ~20MB installer.
func MunkiPackageVersions(installerURL string) (MunkiVersions, error) {
	return munkiPackageVersions(func(start, end int64) ([]byte, error) {
		return rangeGet(installerURL, start, end)
	})
}

func munkiPackageVersions(fetch func(start, end int64) ([]byte, error)) (MunkiVersions, error) {
	header, err := fetch(0, xarHeaderSize-1)
	if err != nil {
		return MunkiVersions{}, fmt.Errorf("fetching xar header: %w", err)
	}
	if len(header) != xarHeaderSize || string(header[0:4]) != "xar!" {
		return MunkiVersions{}, fmt.Errorf("unexpected xar header (got %d bytes)", len(header))
	}
	rawTOCLen := binary.BigEndian.Uint64(header[8:16])
	if rawTOCLen == 0 || rawTOCLen > maxTOCLen {
		return MunkiVersions{}, fmt.Errorf("implausible xar table of contents length: %d bytes", rawTOCLen)
	}
	tocLenCompressed := int64(rawTOCLen) //nolint:gosec // G115: bounds-checked above against maxTOCLen

	tocCompressed, err := fetch(xarHeaderSize, xarHeaderSize+tocLenCompressed-1)
	if err != nil {
		return MunkiVersions{}, fmt.Errorf("fetching xar table of contents: %w", err)
	}
	tocXML, err := zlibInflate(tocCompressed)
	if err != nil {
		return MunkiVersions{}, fmt.Errorf("inflating xar table of contents: %w", err)
	}

	var toc xarTOC
	if err := xml.Unmarshal(tocXML, &toc); err != nil {
		return MunkiVersions{}, fmt.Errorf("parsing xar table of contents: %w", err)
	}

	var dist *xarTOCFileData
	for _, f := range toc.Files {
		if f.Name == "Distribution" && f.Type == "file" && f.Data != nil {
			dist = f.Data
			break
		}
	}
	if dist == nil {
		return MunkiVersions{}, errors.New("no top-level Distribution file in xar table of contents")
	}

	heapStart := int64(xarHeaderSize) + tocLenCompressed
	distCompressed, err := fetch(heapStart+dist.Offset, heapStart+dist.Offset+dist.Length-1)
	if err != nil {
		return MunkiVersions{}, fmt.Errorf("fetching Distribution file: %w", err)
	}
	distXML, err := zlibInflate(distCompressed)
	if err != nil {
		return MunkiVersions{}, fmt.Errorf("inflating Distribution file: %w", err)
	}

	var dpr distributionPkgRefs
	if err := xml.Unmarshal(distXML, &dpr); err != nil {
		return MunkiVersions{}, fmt.Errorf("parsing Distribution file: %w", err)
	}

	var v MunkiVersions
	for _, ref := range dpr.PkgRefs {
		if v.Core == "" && ref.ID == munkiCorePackageID && ref.Version != "" {
			v.Core = ref.Version
		}
		for _, b := range ref.Bundles {
			if v.App == "" && b.ID == munkiAppBundleID {
				v.App = cmp.Or(b.ShortVersionString, b.BundleVersion)
			}
		}
	}
	if v.Core == "" {
		return MunkiVersions{}, fmt.Errorf("no versioned pkg-ref %q in Distribution file", munkiCorePackageID)
	}
	if v.App == "" {
		return MunkiVersions{}, fmt.Errorf("no versioned bundle %q in Distribution file", munkiAppBundleID)
	}

	return v, nil
}

// munkiRangeClient has a short timeout and a response cap: every request it
// makes asks for a specific, small byte range, so a response any larger than
// this means something (a redirect target that ignored Range, a bad proxy)
// went wrong rather than Munki's installer growing an enormous Distribution
// file.
var munkiRangeClient = fleethttp.NewClient(
	fleethttp.WithTimeout(30*time.Second),
	fleethttp.WithMaxResponseSize(10<<20),
)

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

	limited := io.LimitReader(r, maxDecompressedLen+1)
	out, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(out) > maxDecompressedLen {
		return nil, fmt.Errorf("decompressed data exceeds %d bytes", maxDecompressedLen)
	}

	return out, nil
}
