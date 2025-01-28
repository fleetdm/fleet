// Copyright (c) Facebook, Inc. and its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nvd

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/facebookincubator/flog"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/providers/lib/client"
)

// CVE defines the CVE data feed for synchronization.
type CVE int

// Supported CVE feeds.
const (
	cve20xmlGz   CVE = iota // CVE database in XML 2.0 format, gzip compressed.
	cve20xmlZip             // CVE database in XML 2.0 format, zip compressed.
	cve12xmlGz              // CVE database in XML 1.2 format, gzip compressed.
	cve12xmlZip             // CVE database in XML 1.2 format, zip compressed.
	cve10jsonGz             // CVE database in JSON 1.0 format, gzip compressed.
	cve10jsonZip            // CVE database in JSON 1.0 format, zip compressed.
	cve11jsonGz             // CVE database in JSON 1.1 format, gzip compressed.
	cve11jsonZip            // CVE database in JSON 1.1 format, zip compressed.
)

// SupportedCVE contains all supported CVE feeds indexed by name.
var SupportedCVE = map[string]CVE{
	"cve-1.2.xml.gz":   cve12xmlGz,
	"cve-1.2.xml.zip":  cve12xmlZip,
	"cve-2.0.xml.gz":   cve20xmlGz,
	"cve-2.0.xml.zip":  cve20xmlZip,
	"cve-1.0.json.gz":  cve10jsonGz,
	"cve-1.0.json.zip": cve10jsonZip,
	"cve-1.1.json.gz":  cve11jsonGz,
	"cve-1.1.json.zip": cve11jsonZip,
}

// Set implements the flag.Value interface.
func (c *CVE) Set(v string) error {
	feed, exists := SupportedCVE[v]
	if !exists {
		return fmt.Errorf("unsupported CVE feed: %q", v)
	}
	*c = feed
	return nil
}

// String implements the fmt.Stringer interface.
func (c CVE) String() string {
	return "cve-" + c.version() + "." + c.encoding() + "." + c.compression()
}

// Help returns the CVE flag help.
func (c CVE) Help() string {
	opts := make([]string, 0, len(SupportedCVE))
	for k := range SupportedCVE {
		opts = append(opts, k)
	}
	sort.Strings(opts)
	return fmt.Sprintf(
		"CVE feed to sync (default: %s)\navailable:\n%s",
		c, strings.Join(opts, "\n"),
	)
}

// encoding returns the data feed encoding: xml or json.
func (c CVE) encoding() string {
	switch c {
	case cve12xmlGz, cve12xmlZip, cve20xmlGz, cve20xmlZip:
		return "xml"
	case cve10jsonGz, cve10jsonZip, cve11jsonGz, cve11jsonZip:
		return "json"
	default:
		panic("unsupported CVE encoding")
	}
}

// compression returns the data feed compression: gz or zip.
func (c CVE) compression() string {
	switch c {
	case cve10jsonGz, cve11jsonGz, cve12xmlGz, cve20xmlGz:
		return "gz"
	case cve10jsonZip, cve11jsonZip, cve12xmlZip, cve20xmlZip:
		return "zip"
	default:
		panic("unsupported CVE compression")
	}
}

// version returns the data feed version.
func (c CVE) version() string {
	switch c {
	case cve12xmlGz, cve12xmlZip:
		return "1.2"
	case cve20xmlGz, cve20xmlZip:
		return "2.0"
	case cve10jsonGz, cve10jsonZip:
		return "1.0"
	case cve11jsonGz, cve11jsonZip:
		return "1.1"
	default:
		panic("unsupported CVE version")
	}
}

// Sync synchronizes the CVE feed to a local directory.
func (c CVE) Sync(ctx context.Context, src SourceConfig, localdir string) error {
	var err error
	files := cveFileList(c)
	for _, f := range files {
		if err = f.Sync(ctx, src, localdir); err != nil {
			return err
		}
	}
	return nil
}

func cveFileList(c CVE) []cveFile {
	filefmt := func(version, suffix, encoding, compression string) string {
		s := fmt.Sprintf("nvdcve-%s-%s.%s", version, suffix, encoding)
		if compression != "" {
			s += "." + compression
		}
		return s
	}

	// nvd data feeds start in 2002
	const startingYear = 2002
	currentYear := time.Now().Year()
	if currentYear < startingYear {
		panic("system date is in the past, cannot continue")
	}

	entries := (currentYear - startingYear) + 1
	f := make([]cveFile, entries+2) // +recent +modified

	version := c.version()
	encoding := c.encoding()
	compression := c.compression()

	for i := 0; i < entries; i++ {
		year := startingYear + i
		suffix := strconv.Itoa(year)
		f[i] = cveFile{
			CVE:      c,
			MetaFile: filefmt(version, suffix, "meta", ""),
			DataFile: filefmt(version, suffix, encoding, compression),
		}
	}

	// recent
	f[entries] = cveFile{
		CVE:      c,
		MetaFile: filefmt(version, "recent", "meta", ""),
		DataFile: filefmt(version, "recent", encoding, compression),
	}

	// modified
	f[entries+1] = cveFile{
		CVE:      c,
		MetaFile: filefmt(version, "modified", "meta", ""),
		DataFile: filefmt(version, "modified", encoding, compression),
	}

	return f
}

type cveFile struct {
	CVE
	MetaFile string
	DataFile string
}

func (cf cveFile) baseURL(src SourceConfig) (string, error) {
	tmpl, err := template.New("path").Parse(src.CVEFeedPath)
	if err != nil {
		return "", err
	}
	b := bytes.Buffer{}
	err = tmpl.Execute(&b, struct {
		Encoding string
		Version  string
	}{
		Encoding: cf.encoding(),
		Version:  cf.version(),
	})
	if err != nil {
		return "", err
	}
	u := url.URL{
		Scheme: src.Scheme,
		Host:   src.Host,
		Path:   b.String(),
	}
	baseURL := u.String()
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return baseURL, nil
}

func (cf cveFile) Sync(ctx context.Context, src SourceConfig, localdir string) error {
	baseURL, err := cf.baseURL(src)
	if err != nil {
		return err
	}
	remoteMetaURL := baseURL + cf.MetaFile
	flog.V(1).Infof("checking meta file %q for updates to %q", cf.MetaFile, cf.DataFile)
	remoteMeta, needsUpdate, err := cf.needsUpdate(ctx, remoteMetaURL, localdir)
	if err != nil {
		return err
	}
	if !needsUpdate {
		return nil
	}
	remoteFileURL := baseURL + cf.DataFile
	tempDataFilename, err := cf.downloadAndVerify(ctx, remoteMeta, remoteFileURL)
	if err != nil {
		return err
	}
	defer os.Remove(tempDataFilename)

	// write metadata file
	metaFilename := filepath.Join(localdir, cf.MetaFile)
	err = remoteMeta.WriteFile(metaFilename)
	if err != nil {
		return err
	}

	// write data file
	dataFilename := filepath.Join(localdir, cf.DataFile)
	bakDataFilename := dataFilename + ".bak"
	_ = xRename(dataFilename, bakDataFilename)
	if err = xRename(tempDataFilename, dataFilename); err != nil {
		_ = xRename(bakDataFilename, dataFilename)
		return err
	}
	os.Remove(bakDataFilename)
	return nil
}

func (cf cveFile) needsUpdate(ctx context.Context, remoteMetaURL, localdir string) (*metaFile, bool, error) {
	flog.V(1).Infof("downloading meta file %q", remoteMetaURL)
	remoteMeta, err := newMetaFromURL(ctx, remoteMetaURL)
	if err != nil {
		return nil, false, err
	}
	metaFilename := filepath.Join(localdir, cf.MetaFile)
	if _, err := os.Stat(metaFilename); os.IsNotExist(err) {
		flog.V(1).Infof("meta file %q does not exist in %q, needs sync", cf.MetaFile, localdir)
		return &remoteMeta, true, nil
	}
	localMeta, err := newMetaFromFile(metaFilename)
	if err != nil {
		return nil, false, err
	}
	if !localMeta.Equal(remoteMeta) {
		flog.V(1).Infof("data file %q needs update in %q: local%+v != remote%+v", cf.DataFile, localdir, localMeta, remoteMeta)
		return &remoteMeta, true, nil
	}
	dataFilename := filepath.Join(localdir, cf.DataFile)
	fi, err := os.Stat(dataFilename)
	if err != nil {
		if os.IsNotExist(err) {
			flog.V(1).Infof("data file %q does not exist in %q, needs sync", cf.DataFile, localdir)
			return &remoteMeta, true, nil
		}
		return nil, false, err
	}
	var sizeOK bool
	var hashFunc func(filename string) (string, error)
	switch cf.compression() {
	case "gz":
		sizeOK = fi.Size() == int64(localMeta.GzSize)
		hashFunc = gunzipFileAndComputeSHA256
	case "zip":
		sizeOK = fi.Size() == int64(localMeta.ZipSize)
		hashFunc = unzipFileAndComputeSHA256
	}
	if !sizeOK {
		flog.V(1).Infof("data file %q needs update in %q: size mismatch", cf.DataFile, localdir)
		return &remoteMeta, true, nil
	}
	hash, err := hashFunc(dataFilename)
	if err != nil {
		return nil, false, err
	}
	if hash != localMeta.SHA256 {
		flog.V(1).Infof("data file %q needs update in %q: hash mismatch %q != %q", cf.DataFile, localdir, hash, localMeta.SHA256)
		return &remoteMeta, true, nil
	}
	return &remoteMeta, false, nil
}

// downloadAndVerify downloads a remote file into a temporary local file, and performs checksum using size and hash from m.
// Returns the path to the local file.
func (cf cveFile) downloadAndVerify(ctx context.Context, m *metaFile, remoteFileURL string) (string, error) {
	req, err := httpNewRequestContext(ctx, "GET", remoteFileURL)
	if err != nil {
		return "", err
	}
	flog.V(1).Infof("downloading data file %q", remoteFileURL)
	resp, err := client.Default().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if err = httpResponseNotOK(resp); err != nil {
		return "", err
	}
	var wantSize int64
	var hashFunc func(filename string) (string, error)
	switch cf.compression() {
	case "gz":
		wantSize = int64(m.GzSize)
		hashFunc = gunzipFileAndComputeSHA256
	case "zip":
		wantSize = int64(m.ZipSize)
		hashFunc = unzipFileAndComputeSHA256
	}
	if resp.ContentLength != wantSize {
		return "", fmt.Errorf(
			"unexpected size for %q (%s): want %d, have %d",
			remoteFileURL, resp.Status, wantSize, resp.ContentLength,
		)
	}
	dataFile, err := ioutil.TempFile("", "nvdsync-data-")
	if err != nil {
		return "", err
	}
	_, err = io.Copy(dataFile, resp.Body)
	dataFile.Close()
	if err != nil {
		return "", err
	}
	hash, err := hashFunc(dataFile.Name())
	if err != nil {
		defer os.Remove(dataFile.Name()) // TODO: delet?
		return "", err
	}
	if hash != m.SHA256 {
		defer os.Remove(dataFile.Name()) // TODO: delet?
		return "", fmt.Errorf(
			"unexpected hash for %q (%s): want %q, have %q",
			remoteFileURL, resp.Status, m.SHA256, hash,
		)
	}
	return dataFile.Name(), nil
}

// metaFile represents a .meta file from CVE data feeds.
type metaFile struct {
	LastModifiedDate time.Time
	Size             int
	ZipSize          int
	GzSize           int
	SHA256           string
}

// Equal compares two meta files.
func (m metaFile) Equal(other metaFile) bool {
	switch {
	case
		!m.LastModifiedDate.Equal(other.LastModifiedDate),
		m.Size != other.Size,
		m.ZipSize != other.ZipSize,
		m.GzSize != other.GzSize,
		m.SHA256 != other.SHA256:
		return false
	}
	return true
}

// WriteTo writes the contents of m to w.
func (m metaFile) WriteTo(w io.Writer) (int64, error) {
	lines := []string{
		"lastModifiedDate:%s\r\n",
		"size:%d\r\n",
		"zipSize:%d\r\n",
		"gzSize:%d\r\n",
		"sha256:%s\r\n",
	}
	params := []interface{}{
		m.LastModifiedDate.Format(time.RFC3339),
		m.Size,
		m.ZipSize,
		m.GzSize,
		strings.ToUpper(m.SHA256),
	}
	var total int64
	for i := 0; i < len(lines); i++ {
		n, err := fmt.Fprintf(w, lines[i], params[i])
		if err != nil {
			return 0, err
		}
		total += int64(n)
	}
	return total, nil
}

// WriteFile writes m to a file.
func (m metaFile) WriteFile(name string) error {
	f, err := ioutil.TempFile("", "nvdsync-meta-")
	if err != nil {
		return err
	}
	_, err = m.WriteTo(f)
	f.Close()
	if err != nil {
		return err
	}
	bak := name + ".bak"
	_ = xRename(name, bak)
	if err = xRename(f.Name(), name); err != nil {
		_ = xRename(bak, name)
		return err
	}
	os.Remove(bak)
	return err
}

// newMetaFile loads metadata from r.
func newMetaFile(r io.Reader) (metaFile, error) {
	m := metaFile{}
	r = io.LimitReader(r, 16*1024)
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return m, err
	}
	lines := bytes.Split(b, []byte("\r\n"))
	for i, line := range lines {
		if len(line) == 0 {
			break
		}
		lineno := i + 1
		parts := bytes.SplitN(line, []byte(":"), 2)
		if len(parts) != 2 {
			return m, fmt.Errorf("line %d: expecting key:value not %q", lineno, string(line))
		}
		key := string(parts[0])
		val := string(parts[1])
		switch key {
		case "lastModifiedDate":
			t, err := time.Parse(time.RFC3339, val)
			if err != nil {
				return m, fmt.Errorf("line %d: expecting lastModifiedDate={RFC3339} not %q", lineno, string(line))
			}
			m.LastModifiedDate = t
		case "size":
			v, err := strconv.Atoi(val)
			if err != nil {
				return m, fmt.Errorf("line %d: expecting size={int} not %q", lineno, string(line))
			}
			m.Size = v
		case "zipSize":
			v, err := strconv.Atoi(val)
			if err != nil {
				return m, fmt.Errorf("line %d: expecting zipSize={int} not %q", lineno, string(line))
			}
			m.ZipSize = v
		case "gzSize":
			v, err := strconv.Atoi(val)
			if err != nil {
				return m, fmt.Errorf("line %d: expecting gzSize={int} not %q", lineno, string(line))
			}
			m.GzSize = v
		case "sha256":
			m.SHA256 = strings.ToUpper(val)
		}
	}
	return m, nil
}

// newMetaFromURL loads metadata from a URL pointing to a .meta file.
func newMetaFromURL(ctx context.Context, url string) (metaFile, error) {
	m := metaFile{}
	req, err := httpNewRequestContext(ctx, "GET", url)
	if err != nil {
		return m, err
	}
	resp, err := client.Default().Do(req)
	if err != nil {
		return m, err
	}
	defer resp.Body.Close()
	if err = httpResponseNotOK(resp); err != nil {
		return m, err
	}
	m, err = newMetaFile(resp.Body)
	if err != nil {
		return m, fmt.Errorf("malformed data in remote metadata %q: %v", url, err)
	}
	return m, nil
}

// newMetaFromFile loads metadata from a local .meta file.
func newMetaFromFile(filename string) (metaFile, error) {
	m := metaFile{}
	f, err := os.Open(filename)
	if err != nil {
		return m, err
	}
	defer f.Close()
	m, err = newMetaFile(f)
	if err != nil {
		return m, fmt.Errorf("malformed data in local metadata %q: %v", filename, err)
	}
	return m, nil
}

func computeSHA256(r io.Reader) (string, error) {
	hasher := sha256.New()
	_, err := io.Copy(hasher, r)
	if err != nil {
		return "", err
	}
	hash := hasher.Sum(nil)
	return strings.ToUpper(hex.EncodeToString(hash)), nil
}

func gunzipAndComputeSHA256(r io.Reader) (string, error) {
	f, err := gzip.NewReader(r)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return computeSHA256(f)
}

func gunzipFileAndComputeSHA256(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return gunzipAndComputeSHA256(f)
}

func unzipFileAndComputeSHA256(filename string) (string, error) {
	f, err := zip.OpenReader(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if len(f.File) != 1 {
		return "", fmt.Errorf(
			"unexpected number of files in zip %q: want 1, have %d",
			filename, len(f.File),
		)
	}
	ff, err := f.File[0].Open()
	if err != nil {
		return "", err
	}
	defer ff.Close()
	return computeSHA256(ff)
}
