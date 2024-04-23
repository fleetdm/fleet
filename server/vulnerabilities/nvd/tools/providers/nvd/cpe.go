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
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/facebookincubator/flog"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/providers/lib/client"
)

// CPE defines the CPE data feed for synchronization.
type CPE int

// Supported CPE feeds.
const (
	cpe23xmlGz  CPE = iota // CPE database in XML 2.3 format, gzip compressed.
	cpe23xmlZip            // CPE database in XML 2.3 format, zip compressed.
	cpe22xmlGz             // CPE database in XML 2.2 format, gzip compressed.
	cpe22xmlZip            // CPE database in XML 2.2 format, zip compressed.
)

// SupportedCPE contains all supported CPE data feeds indexed by name.
var SupportedCPE = map[string]CPE{
	"cpe-2.2.xml.gz":  cpe22xmlGz,
	"cpe-2.2.xml.zip": cpe22xmlZip,
	"cpe-2.3.xml.gz":  cpe23xmlGz,
	"cpe-2.3.xml.zip": cpe23xmlZip,
}

// Set implements the flag.Value interface.
func (c *CPE) Set(v string) error {
	feed, exists := SupportedCPE[v]
	if !exists {
		return fmt.Errorf("unsupported CPE feed: %q", v)
	}
	*c = feed
	return nil
}

// String implements the fmt.Stringer interface.
func (c CPE) String() string {
	return "cpe-" + c.version() + ".xml." + c.compression()
}

// Help returns the CPE flag help.
func (c CPE) Help() string {
	opts := make([]string, 0, len(SupportedCPE))
	for k := range SupportedCPE {
		opts = append(opts, k)
	}
	sort.Strings(opts)
	return fmt.Sprintf(
		"CPE feed to sync (default: %s)\navailable:\n%s",
		c, strings.Join(opts, "\n"),
	)
}

// compression returns the data feed compression: gz or zip.
func (c CPE) compression() string {
	switch c {
	case cpe22xmlGz, cpe23xmlGz:
		return "gz"
	case cpe22xmlZip, cpe23xmlZip:
		return "zip"
	default:
		panic("unsupported CPE compression")
	}
}

// version returns the data feed version.
func (c CPE) version() string {
	switch c {
	case cpe22xmlGz, cpe22xmlZip:
		return "2.2"
	case cpe23xmlGz, cpe23xmlZip:
		return "2.3"
	default:
		panic("unsupported CPE version")
	}
}

// Sync synchronizes the CPE feed to a local directory.
func (c CPE) Sync(ctx context.Context, src SourceConfig, localdir string) error {
	basename := "official-cpe-dictionary_v" + c.version()
	cf := cpeFile{
		CPE:      c,
		EtagFile: basename + ".etag",
		DataFile: basename + ".xml." + c.compression(),
	}
	return cf.Sync(ctx, src, localdir)
}

type cpeFile struct {
	CPE
	EtagFile string
	DataFile string
}

func (cf cpeFile) baseURL(src SourceConfig) string {
	u := url.URL{
		Scheme: src.Scheme,
		Host:   src.Host,
		Path:   src.CPEFeedPath,
	}
	baseURL := u.String()
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return baseURL
}

func (cf cpeFile) Sync(ctx context.Context, src SourceConfig, localdir string) error {
	baseURL := cf.baseURL(src)
	sourceURL := baseURL + cf.DataFile
	needsUpdate, err := cf.needsUpdate(ctx, sourceURL, localdir)
	if err != nil {
		return err
	}
	if !needsUpdate {
		return nil
	}
	etag, tempDataFilename, err := cf.download(ctx, sourceURL)
	if err != nil {
		return err
	}
	defer os.Remove(tempDataFilename)

	// write etag file
	etagFilename := filepath.Join(localdir, cf.EtagFile)
	err = ioutil.WriteFile(etagFilename, []byte(etag), 0o644)
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

func (cf cpeFile) needsUpdate(ctx context.Context, targetURL, localdir string) (bool, error) {
	flog.V(1).Infof("checking etag for %q", targetURL)
	req, err := httpNewRequestContext(ctx, "HEAD", targetURL)
	if err != nil {
		return false, err
	}
	resp, err := client.Default().Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if err = httpResponseNotOK(resp); err != nil {
		return false, err
	}
	remoteEtag := resp.Header.Get("Etag")
	if remoteEtag == "" {
		return false, fmt.Errorf("server not returning etag header for %q", targetURL)
	}
	etagBytes, err := ioutil.ReadFile(filepath.Join(localdir, cf.EtagFile))
	if err != nil {
		flog.V(1).Infof("etag file %q for not exist in %q, needs sync", cf.EtagFile, localdir)
		return true, nil
	}
	localEtag := string(etagBytes)
	if localEtag != remoteEtag {
		flog.V(1).Infof("data file %q needs update in %q: hash mismatch %q != %q", cf.DataFile, localdir, localEtag, remoteEtag)
		return true, nil
	}
	return false, nil
}

// download file from targetURL, returns etag and path to local file.
func (cf cpeFile) download(ctx context.Context, targetURL string) (string, string, error) {
	flog.V(1).Infof("downloading data file %q", targetURL)
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return "", "", err
	}
	resp, err := client.Default().Do(req.WithContext(ctx))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if err = httpResponseNotOK(resp); err != nil {
		return "", "", err
	}
	dataFile, err := ioutil.TempFile("", "nvdsync-data-")
	if err != nil {
		return "", "", err
	}
	_, err = io.Copy(dataFile, resp.Body)
	if err != nil {
		return "", "", err
	}
	return resp.Header.Get("Etag"), dataFile.Name(), nil
}
