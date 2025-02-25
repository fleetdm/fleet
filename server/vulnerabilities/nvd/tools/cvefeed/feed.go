// Package cvefeed defines types and methods necessary to parse NVD vulnerability
// feed and match an inventory of CPE names against it.
//
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

// Package cvefeed provides an API to NVD CVE feeds parsing and matching.
package cvefeed

import (
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd/schema"
)

// ParseJSON parses JSON dictionary from NVD vulnerability feed
func ParseJSON(in io.Reader) ([]Vuln, error) {
	feed, err := getFeed(in)
	if err != nil {
		return nil, fmt.Errorf("cvefeed.ParseJSON: %v", err)
	}

	vulns := make([]Vuln, 0, len(feed.CVEItems))
	for _, cve := range feed.CVEItems {
		if cve != nil && cve.Configurations != nil {
			vulns = append(vulns, nvd.ToVuln(cve))
		}
	}
	return vulns, nil
}

func getFeed(in io.Reader) (*schema.NVDCVEFeedJSON10, error) {
	reader, err := setupReader(in)
	if err != nil {
		return nil, fmt.Errorf("can't setup reader: %v", err)
	}
	defer reader.Close()

	var feed schema.NVDCVEFeedJSON10
	if err := json.NewDecoder(reader).Decode(&feed); err != nil {
		return nil, err
	}
	return &feed, nil
}

func setupReader(in io.Reader) (src io.ReadCloser, err error) {
	r := bufio.NewReader(in)
	header, err := r.Peek(2)
	if err != nil {
		return nil, err
	}
	// assume plain text first
	src = ioutil.NopCloser(r)
	// replace with gzip.Reader if gzip'ed
	if header[0] == 0x1f && header[1] == 0x8b { // file is gzip'ed
		zr, err := gzip.NewReader(r)
		if err != nil {
			return nil, err
		}
		src = zr
	} else if header[0] == 'B' && header[1] == 'Z' {
		// or with bzip2.Reader if bzip2'ed
		src = ioutil.NopCloser(bzip2.NewReader(r))
	}
	// TODO: maybe support .zip
	return src, nil
}
