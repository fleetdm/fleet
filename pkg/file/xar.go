package file

//		Copyright 2023 SAS Software
//
//	 Licensed under the Apache License, Version 2.0 (the "License");
//	 you may not use this file except in compliance with the License.
//	 You may obtain a copy of the License at
//
//	     http://www.apache.org/licenses/LICENSE-2.0
//
//	 Unless required by applicable law or agreed to in writing, software
//	 distributed under the License is distributed on an "AS IS" BASIS,
//	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	 See the License for the specific language governing permissions and
//	 limitations under the License.
//
// xar contains utilities to parse xar files, most of the logic here is a
// simplified version extracted from the logic to sign xar files in
// https://github.com/sassoftware/relic

import (
	"bytes"
	"compress/bzip2"
	"compress/zlib"
	"crypto"
	"crypto/sha256"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

const (
	// xarMagic is the [file signature][1] (or magic bytes) for xar
	//
	// [1]: https://en.wikipedia.org/wiki/List_of_file_signatures
	xarMagic = 0x78617221

	xarHeaderSize = 28
)

const (
	hashNone uint32 = iota
	hashSHA1
	hashMD5
	hashSHA256
	hashSHA512
)

var (
	// ErrInvalidType is used to signal that the provided package can't be
	// parsed because is an invalid file type.
	ErrInvalidType = errors.New("invalid file type")
	// ErrNotSigned is used to signal that the provided package doesn't
	// contain a signature.
	ErrNotSigned = errors.New("file is not signed")
)

type xarHeader struct {
	Magic            uint32
	HeaderSize       uint16
	Version          uint16
	CompressedSize   int64
	UncompressedSize int64
	HashType         uint32
}

type tocXar struct {
	TOC toc `xml:"toc"`
}

type toc struct {
	Signature  *any `xml:"signature"`
	XSignature *any `xml:"x-signature"`
}

type xmlXar struct {
	XMLName xml.Name `xml:"xar"`
	TOC     xmlTOC
}

type xmlTOC struct {
	XMLName xml.Name   `xml:"toc"`
	Files   []*xmlFile `xml:"file"`
}

type xmlFileData struct {
	XMLName  xml.Name `xml:"data"`
	Length   int64    `xml:"length"`
	Offset   int64    `xml:"offset"`
	Size     int64    `xml:"size"`
	Encoding struct {
		Style string `xml:"style,attr"`
	} `xml:"encoding"`
}

type xmlFile struct {
	XMLName xml.Name `xml:"file"`
	Name    string   `xml:"name"`
	Data    *xmlFileData
}

// distributionXML represents the structure of the distributionXML.xml
type distributionXML struct {
	Title   string               `xml:"title"`
	Product distributionProduct  `xml:"product"`
	PkgRefs []distributionPkgRef `xml:"pkg-ref"`
}

type packageInfoXML struct {
	Version         string               `xml:"version,attr"`
	InstallLocation string               `xml:"install-location,attr"`
	Identifier      string               `xml:"identifier,attr"`
	Bundles         []distributionBundle `xml:"bundle"`
}

// distributionProduct represents the product element
type distributionProduct struct {
	ID      string `xml:"id,attr"`
	Version string `xml:"version,attr"`
}

// distributionPkgRef represents the pkg-ref element
type distributionPkgRef struct {
	ID                string                      `xml:"id,attr"`
	Version           string                      `xml:"version,attr"`
	BundleVersions    []distributionBundleVersion `xml:"bundle-version"`
	MustClose         distributionMustClose       `xml:"must-close"`
	PackageIdentifier string                      `xml:"packageIdentifier,attr"`
	InstallKBytes     string                      `xml:"installKBytes,attr"`
}

// distributionBundleVersion represents the bundle-version element
type distributionBundleVersion struct {
	Bundles []distributionBundle `xml:"bundle"`
}

// distributionBundle represents the bundle element
type distributionBundle struct {
	Path                       string `xml:"path,attr"`
	ID                         string `xml:"id,attr"`
	CFBundleShortVersionString string `xml:"CFBundleShortVersionString,attr"`
}

// distributionMustClose represents the must-close element
type distributionMustClose struct {
	Apps []distributionApp `xml:"app"`
}

// distributionApp represents the app element
type distributionApp struct {
	ID string `xml:"id,attr"`
}

// ExtractXARMetadata extracts the name and version metadata from a .pkg file
// in the XAR format.
func ExtractXARMetadata(tfr *fleet.TempFileReader) (*InstallerMetadata, error) {
	var hdr xarHeader

	h := sha256.New()
	size, _ := io.Copy(h, tfr) // writes to a hash cannot fail

	if err := tfr.Rewind(); err != nil {
		return nil, fmt.Errorf("rewind reader: %w", err)
	}

	// read the file header
	if err := binary.Read(tfr, binary.BigEndian, &hdr); err != nil {
		return nil, fmt.Errorf("decode xar header: %w", err)
	}

	zr, err := zlib.NewReader(io.LimitReader(tfr, hdr.CompressedSize))
	if err != nil {
		return nil, fmt.Errorf("create zlib reader: %w", err)
	}
	defer zr.Close()

	// decode the TOC data (in XML inside the zlib-compressed data)
	var root xmlXar
	decoder := xml.NewDecoder(zr)
	decoder.Strict = false
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("decode xar xml: %w", err)
	}

	// look for the distribution file, with the metadata information
	heapOffset := xarHeaderSize + hdr.CompressedSize
	var packageInfoFile *xmlFile
	for _, f := range root.TOC.Files {
		switch f.Name {
		case "Distribution":
			contents, err := readCompressedFile(tfr, heapOffset, size, f)
			if err != nil {
				return nil, err
			}

			meta, err := parseDistributionFile(contents)
			if err != nil {
				return nil, fmt.Errorf("parsing Distribution file: %w", err)
			}
			meta.SHASum = h.Sum(nil)
			return meta, nil

		case "PackageInfo":
			// If Distribution archive was not found, we will use the top-level PackageInfo archive
			packageInfoFile = f
		}
	}

	if packageInfoFile != nil {
		contents, err := readCompressedFile(tfr, heapOffset, size, packageInfoFile)
		if err != nil {
			return nil, err
		}

		meta, err := parsePackageInfoFile(contents)
		if err != nil {
			return nil, fmt.Errorf("parsing PackageInfo file: %w", err)
		}
		meta.SHASum = h.Sum(nil)
		return meta, nil
	}

	return &InstallerMetadata{SHASum: h.Sum(nil)}, nil
}

func readCompressedFile(rat io.ReaderAt, heapOffset int64, sectionLength int64, f *xmlFile) ([]byte, error) {
	var fileReader io.Reader
	heapReader := io.NewSectionReader(rat, heapOffset, sectionLength-heapOffset)
	fileReader = io.NewSectionReader(heapReader, f.Data.Offset, f.Data.Length)

	// the distribution file can be compressed differently than the TOC, the
	// actual compression is specified in the Encoding.Style field.
	if strings.Contains(f.Data.Encoding.Style, "x-gzip") {
		// despite the name, x-gzip fails to decode with the gzip package
		// (invalid header), but it works with zlib.
		zr, err := zlib.NewReader(fileReader)
		if err != nil {
			return nil, fmt.Errorf("create zlib reader: %w", err)
		}
		defer zr.Close()
		fileReader = zr
	} else if strings.Contains(f.Data.Encoding.Style, "x-bzip2") {
		fileReader = bzip2.NewReader(fileReader)
	}
	// TODO: what other compression methods are supported?

	contents, err := io.ReadAll(fileReader)
	if err != nil {
		return nil, fmt.Errorf("reading %s file: %w", f.Name, err)
	}
	return contents, nil
}

func parseDistributionFile(rawXML []byte) (*InstallerMetadata, error) {
	var distXML distributionXML
	if err := xml.Unmarshal(rawXML, &distXML); err != nil {
		return nil, fmt.Errorf("unmarshal Distribution XML: %w", err)
	}

	name, identifier, version, packageIDs := getDistributionInfo(&distXML)
	return &InstallerMetadata{
		Name:             name,
		Version:          version,
		BundleIdentifier: identifier,
		PackageIDs:       packageIDs,
	}, nil
}

// getDistributionInfo gets the name, bundle identifier and version of a PKG distribution file
func getDistributionInfo(d *distributionXML) (name string, identifier string, version string, packageIDs []string) {
	var appVersion string

	// find the package ids that have an installation size
	packageIDSet := make(map[string]struct{}, 1)
	for _, pkg := range d.PkgRefs {
		if pkg.InstallKBytes != "" && pkg.InstallKBytes != "0" {
			var id string
			if pkg.PackageIdentifier != "" {
				id = pkg.PackageIdentifier
			} else if pkg.ID != "" {
				id = pkg.ID
			}
			if id != "" {
				packageIDSet[id] = struct{}{}
			}
		}
	}
	if len(packageIDSet) == 0 {
		// if we didn't find any package IDs with installation size, then grab all of them
		for _, pkg := range d.PkgRefs {
			var id string
			if pkg.PackageIdentifier != "" {
				id = pkg.PackageIdentifier
			} else if pkg.ID != "" {
				id = pkg.ID
			}
			if id != "" {
				packageIDSet[id] = struct{}{}
			}
		}
	}
	for id := range packageIDSet {
		packageIDs = append(packageIDs, id)
	}

out:
	// look in all the bundle versions for one that has a `path` attribute
	// that is not nested, this is generally the case for packages that distribute
	// `.app` files, which are ultimately picked up as an installed app by osquery
	for _, pkg := range d.PkgRefs {
		for _, versions := range pkg.BundleVersions {
			for _, bundle := range versions.Bundles {
				if base, isValid := isValidAppFilePath(bundle.Path); isValid {
					identifier = bundle.ID
					name = base
					appVersion = bundle.CFBundleShortVersionString
					break out
				}
			}
		}
	}

	// if we didn't find anything, look for any <pkg-ref> elements and grab
	// the first `<must-close>`, `packageIdentifier` or `id` attribute we
	// find as the bundle identifier, in that order
	if identifier == "" {
		for _, pkg := range d.PkgRefs {
			if len(pkg.MustClose.Apps) > 0 {
				identifier = pkg.MustClose.Apps[0].ID
				break
			}
		}
	}

	if identifier == "" {
		for _, pkg := range d.PkgRefs {
			if pkg.PackageIdentifier != "" {
				identifier = pkg.PackageIdentifier
				break
			}

			if pkg.ID != "" {
				identifier = pkg.ID
				break
			}
		}
	}

	// if the identifier is still empty, try to use the product id
	if identifier == "" && d.Product.ID != "" {
		identifier = d.Product.ID
	}

	// if package IDs are still empty, use the identifier as the package ID
	if len(packageIDs) == 0 && identifier != "" {
		packageIDs = append(packageIDs, identifier)
	}

	// for the name, try to use the title and fallback to the bundle
	// identifier
	if name == "" && d.Title != "" {
		name = d.Title
	}
	// "DISTRIBUTION_TITLE" is a default title that is sometimes left in the Distribution file. Fall
	// back to the bundle ID if this is the case.
	if name == "" || name == "DISTRIBUTION_TITLE" {
		name = identifier
	}

	// for the version, try to use the top-level product version, if not,
	// fallback to any version definition alongside the name or the first
	// version in a pkg-ref we find.
	if d.Product.Version != "" {
		version = d.Product.Version
	}
	if version == "" && appVersion != "" {
		version = appVersion
	}
	if version == "" {
		for _, pkgRef := range d.PkgRefs {
			if pkgRef.Version != "" {
				version = pkgRef.Version
			}
		}
	}

	return name, identifier, version, packageIDs
}

func parsePackageInfoFile(rawXML []byte) (*InstallerMetadata, error) {
	var packageInfo packageInfoXML
	if err := xml.Unmarshal(rawXML, &packageInfo); err != nil {
		return nil, fmt.Errorf("unmarshal PackageInfo XML: %w", err)
	}

	name, identifier, version, packageIDs := getPackageInfo(&packageInfo)
	return &InstallerMetadata{
		Name:             name,
		Version:          version,
		BundleIdentifier: identifier,
		PackageIDs:       packageIDs,
	}, nil
}

// getPackageInfo gets the name, bundle identifier and version of a PKG top level PackageInfo file
func getPackageInfo(p *packageInfoXML) (name string, identifier string, version string, packageIDs []string) {
	packageIDSet := make(map[string]struct{}, 1)
	for _, bundle := range p.Bundles {
		installPath := bundle.Path
		if p.InstallLocation != "" {
			installPath = filepath.Join(p.InstallLocation, installPath)
		}
		installPath = strings.TrimPrefix(installPath, "/")
		installPath = strings.TrimPrefix(installPath, "./")
		if base, isValid := isValidAppFilePath(installPath); isValid {
			identifier = fleet.Preprocess(bundle.ID)
			name = base
			version = fleet.Preprocess(bundle.CFBundleShortVersionString)
		}
		bundleID := fleet.Preprocess(bundle.ID)
		if bundleID != "" {
			packageIDSet[bundleID] = struct{}{}
		}
	}

	for id := range packageIDSet {
		packageIDs = append(packageIDs, id)
	}

	// if we didn't find a version, grab the version from pkg-info element
	// Note: this version may be wrong since it is the version of the package and not the app
	if version == "" {
		version = fleet.Preprocess(p.Version)
	}

	// if we didn't find a bundle identifier, grab the identifier from pkg-info element
	if identifier == "" {
		identifier = fleet.Preprocess(p.Identifier)
	}

	// if we didn't find a name, grab the name from the identifier
	if name == "" {
		idParts := strings.Split(identifier, ".")
		if len(idParts) > 0 {
			name = idParts[len(idParts)-1]
		}
	}

	// if we didn't find package IDs, use the identifier as the package ID
	if len(packageIDs) == 0 && identifier != "" {
		packageIDs = append(packageIDs, identifier)
	}

	return name, identifier, version, packageIDs
}

// isValidAppFilePath checks if the given input is a file name ending with .app
// or if it's in the "Applications" directory with a .app extension.
func isValidAppFilePath(input string) (string, bool) {
	dir, file := filepath.Split(input)

	if dir == "" && file == input {
		return file, true
	}

	if strings.HasSuffix(file, ".app") {
		if dir == "Applications/" {
			return file, true
		}
	}

	return "", false
}

// CheckPKGSignature checks if the provided bytes correspond to a signed pkg
// (xar) file.
//
// - If the file is not xar, it returns a ErrInvalidType error
// - If the file is not signed, it returns a ErrNotSigned error
func CheckPKGSignature(pkg io.Reader) error {
	buff := bytes.NewBuffer(nil)
	if _, err := io.Copy(buff, pkg); err != nil {
		return err
	}
	r := bytes.NewReader(buff.Bytes())

	hdr, hashType, err := parseHeader(io.NewSectionReader(r, 0, 28))
	if err != nil {
		return err
	}

	base := int64(hdr.HeaderSize)
	toc, err := parseTOC(io.NewSectionReader(r, base, hdr.CompressedSize), hashType)
	if err != nil {
		return err
	}

	if toc.Signature == nil && toc.XSignature == nil {
		return ErrNotSigned
	}

	return nil
}

func decompress(r io.Reader) ([]byte, error) {
	zr, err := zlib.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}

func parseTOC(r io.Reader, hashType crypto.Hash) (*toc, error) {
	tocHash := hashType.New()
	r = io.TeeReader(r, tocHash)
	decomp, err := decompress(r)
	if err != nil {
		return nil, fmt.Errorf("decompressing TOC: %w", err)
	}
	var toc tocXar
	if err := xml.Unmarshal(decomp, &toc); err != nil {
		return nil, fmt.Errorf("decoding TOC: %w", err)
	}
	return &toc.TOC, nil
}

func parseHeader(r io.Reader) (xarHeader, crypto.Hash, error) {
	var hdr xarHeader
	if err := binary.Read(r, binary.BigEndian, &hdr); err != nil {
		return xarHeader{}, 0, err
	}

	if hdr.Magic != xarMagic {
		return hdr, 0, ErrInvalidType
	}

	var hashType crypto.Hash
	switch hdr.HashType {
	case hashSHA1:
		hashType = crypto.SHA1
	case hashSHA256:
		hashType = crypto.SHA256
	case hashSHA512:
		hashType = crypto.SHA512
	default:
		return xarHeader{}, 0, fmt.Errorf("unknown hash algorithm %d", hdr.HashType)
	}

	return hdr, hashType, nil
}
