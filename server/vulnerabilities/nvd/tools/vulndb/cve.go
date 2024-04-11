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

package vulndb

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/pkg/errors"

	nvd "github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
)

type cveFile struct {
	items []*nvd.NVDCVEFeedJSON10DefCVEItem
}

func (c *cveFile) Add(cve string, nvdjson []byte) error {
	var item nvd.NVDCVEFeedJSON10DefCVEItem
	err := json.Unmarshal(nvdjson, &item)
	if err != nil {
		return errors.Wrapf(err, "%s json payload is corrupted: %v", cve, err)
	}
	c.items = append(c.items, &item)
	return nil
}

func (c *cveFile) EncodeJSON(w io.Writer) error {
	err := json.NewEncoder(w).Encode(&nvd.NVDCVEFeedJSON10{
		CVEItems: c.items,
	})
	if err != nil {
		return errors.Wrap(err, "cannot encode NVD CVE JSON file")
	}
	return nil
}

func (c *cveFile) EncodeIndentedJSON(w io.Writer, prefix, indent string) error {
	var b, o bytes.Buffer
	err := c.EncodeJSON(&b)
	if err != nil {
		return err
	}
	err = json.Indent(&o, b.Bytes(), prefix, indent)
	if err != nil {
		return errors.Wrap(err, "cannot indent NVD CVE JSON file")
	}
	_, err = io.Copy(w, &o)
	if err != nil {
		return errors.Wrap(err, "cannot copy indented NVD CVE JSON file")
	}
	return nil
}

// cveItem is a helper for extracting information from CVE items.
type cveItem struct {
	item *nvd.NVDCVEFeedJSON10DefCVEItem
}

func (c cveItem) ID() string {
	cve := c.item.CVE
	if cve != nil && cve.CVEDataMeta != nil {
		return cve.CVEDataMeta.ID
	}
	return ""
}

func (c cveItem) Published() time.Time {
	t, err := ParseTime(c.item.PublishedDate)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (c cveItem) Modified() time.Time {
	t, err := ParseTime(c.item.LastModifiedDate)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (c cveItem) Summary() string {
	cve := c.item.CVE
	if cve != nil && cve.Description != nil {
		// TODO: handle multi-language descriptions.
		if len(cve.Description.DescriptionData) > 0 {
			return cve.Description.DescriptionData[0].Value
		}
	}
	return ""
}

func (c cveItem) BaseScore() float64 {
	impact := c.item.Impact
	if impact != nil {
		v3 := impact.BaseMetricV3
		if v3 != nil && v3.CVSSV3 != nil && v3.CVSSV3.BaseScore > 0 {
			return v3.CVSSV3.BaseScore
		}
		v2 := impact.BaseMetricV2
		if v2 != nil && v2.CVSSV2 != nil {
			return v2.CVSSV2.BaseScore
		}
	}
	return 0
}

func (c cveItem) JSON() []byte {
	b, err := json.Marshal(c.item)
	if err != nil {
		panic(err)
	}
	return b
}

func readNVDCVEJSON(filename string) (*nvd.NVDCVEFeedJSON10, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return parseNVDCVEJSON(f)
}

// parseNVDCVEJSON parses NVD CVE JSON data from r, decompressing as needed.
func parseNVDCVEJSON(r io.Reader) (*nvd.NVDCVEFeedJSON10, error) {
	br := bufio.NewReader(r)
	b, err := br.Peek(2)
	if err != nil {
		return nil, errors.Wrap(err, "cannot peek into NVD CVE JSON feed")
	}

	r = br

	if b[0] == 31 && b[1] == 139 {
		gr, err := gzip.NewReader(r)
		if err != nil {
			return nil, errors.Wrap(err, "cannot gunzip NVD CVE JSON feed")
		}
		defer gr.Close()
		r = gr
	}

	b, err = ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read NVD CVE JSON feed")
	}

	var f nvd.NVDCVEFeedJSON10
	err = json.NewDecoder(bytes.NewReader(b)).Decode(&f)
	if err != nil {
		if jsonErr, ok := err.(*json.SyntaxError); ok {
			err = jsonSyntaxError(string(b), jsonErr)
		}

		return nil, errors.Wrap(err, "cannot decode NVD CVE JSON feed")
	}

	return &f, nil
}

func jsonSyntaxError(input string, jsonErr *json.SyntaxError) error {
	offset := int(jsonErr.Offset)

	if offset > len(input) || offset < 0 {
		return jsonErr
	}

	lf := rune(0x0A)
	line, col := 1, 1

	for i, c := range input {
		if c == lf {
			line++
			col = 1
		}

		col++
		if i == offset {
			return errors.Wrapf(jsonErr,
				"syntax error on line %d and column %d", line, col)
		}
	}

	return jsonErr
}
