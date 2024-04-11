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
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func TestCVEFile(t *testing.T) {
	f, err := createSampleCVE()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	feed, err := readNVDCVEJSON(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	if i := len(feed.CVEItems); i != 2 {
		t.Fatalf("unexpected number of items: want 2, have %d", i)
	}

	cve := cveItem{feed.CVEItems[0]}
	switch {
	case cve.ID() != "CVE-0000-0000":
		t.Fatal("unexpected value for CVE ID")
	case cve.Published().Format(TimeLayout) != "2018-09-06T17:29Z":
		t.Fatal("unexpected value for published time")
	case cve.Modified().Format(TimeLayout) != "2018-09-06T17:30Z":
		t.Fatal("unexpected value for modified time")
	case cve.Summary() != "hello world":
		t.Fatal("unexpected value for summary")
	case cve.BaseScore() != 0:
		t.Fatal("unexpected value for base score")
	case !bytes.Contains(cve.JSON(), []byte("CVE-0000-0000")):
		t.Fatal("missing CVE ID in JSON output")
	}

	j := &cveFile{}
	j.Add("CVE-0000-0000", cve.JSON())
	err = j.EncodeIndentedJSON(ioutil.Discard, "", "\t")
	if err != nil {
		t.Fatal(err)
	}
}

func createSampleCVE() (*os.File, error) {
	f, err := ioutil.TempFile("", "vulndb-cvefile")
	if err != nil {
		return nil, err
	}
	_, err = io.WriteString(f, sampleCVE)
	if err != nil {
		return nil, err
	}
	return f, nil
}

var sampleCVE = `
{"CVE_data_format":"","CVE_data_type":"","CVE_data_version":"","CVE_Items":[{"cve":{"affects":{"vendor":{"vendor_data":[]}},"CVE_data_meta":{"ASSIGNER":"test","ID":"CVE-0000-0000"},"data_format":"MITRE","data_type":"CVE","data_version":"4.0","description":{"description_data":[{"lang":"en","value":"hello world"}]},"problemtype":{"problemtype_data":[{"description":[]}]},"references":{"reference_data":[]}},"configurations":{"CVE_data_version":"4.0"},"impact":{},"lastModifiedDate":"2018-09-06T17:30Z","publishedDate":"2018-09-06T17:29Z"},{"cve":{"affects":{"vendor":{"vendor_data":[]}},"CVE_data_meta":{"ASSIGNER":"test","ID":"CVE-0000-0001"},"data_format":"MITRE","data_type":"CVE","data_version":"4.0","description":{"description_data":[{"lang":"en","value":"hello world"}]},"problemtype":{"problemtype_data":[{"description":[]}]},"references":{"reference_data":[]}},"configurations":{"CVE_data_version":"4.0"},"impact":{},"lastModifiedDate":"2018-09-06T17:30Z","publishedDate":"2018-09-06T17:29Z"}]}
`
