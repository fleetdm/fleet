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

package cvefeed

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testFeed = `
{
  "CVE_Items": [
    {
      "cve": {
        "affects": null,
        "CVE_data_meta": {
          "ASSIGNER": "cve@mitre.org",
          "ID": "CVE-2020-1111"
        },
        "data_format": "MITRE",
        "data_type": "CVE",
        "data_version": "4.0",
        "description": {
          "description_data": [
            {
              "lang": "en",
              "value": ""
            }
          ]
        },
        "problemtype": {
          "problemtype_data": [
            {
              "description": [
                {
                  "lang": "en",
                  "value": "CWE-20"
                }
              ]
            }
          ]
        },
        "references": {
          "reference_data": [
            {
              "name": "",
              "refsource": "",
              "tags": [
                "Vendor Advisory"
              ],
              "url": ""
            },
            {
              "name": "test",
              "refsource": "MISC",
              "url": ""
            }
          ]
        }
      },
      "configurations": {
        "CVE_data_version": "4.0",
        "nodes": [
		  {
            "children": [
              {
                "cpe_match": [
                  {
                    "cpe23Uri": "cpe:2.3:a:test:test:-:*:*:*:*:*:*:*",
                    "vulnerable": true
                  }
                ],
                "operator": "OR"
              },
              {
                "cpe_match": [
                  {
                    "cpe23Uri": "cpe:2.3:h:test:test:-:*:*:*:*:*:*:*",
                    "vulnerable": false
                  }
                ],
                "operator": "OR"
              }
            ],
            "operator": "AND"
		  },
		  {
            "children": [
              {
                "cpe_match": [
                  {
                    "cpe23Uri": "cpe:2.3:a:test:test:-:*:*:*:*:*:*:*",
                    "vulnerable": true
                  }
                ],
                "operator": "OR"
              },
              {
                "cpe_match": [
                  {
                    "cpe23Uri": "cpe:2.3:h:test:test:-:*:*:*:*:*:*:*",
                    "vulnerable": false
				  },
				  {
                    "cpe23Uri": "cpe:2.3:h:test:test:-:*:*:*:*:*:*:*",
                    "vulnerable": false
                  }
                ],
                "operator": "OR"
              }
            ],
            "operator": "AND"
          },
          {
            "children": [
              {
                "cpe_match": [
                  {
                    "cpe23Uri": "cpe:2.3:o:test:test:-:*:*:*:*:*:*:*",
                    "vulnerable": true
                  }
                ],
                "operator": "OR"
              },
              {
                "cpe_match": [
                  {
                    "cpe23Uri": "cpe:2.3:a:test:test:-:*:*:*:*:*:*:*",
                    "vulnerable": false
                  },
                  {
                    "cpe23Uri": "cpe:2.3:h:test:test:-:*:*:*:*:*:*:*",
                    "vulnerable": false
                  }
                ],
                "operator": "OR"
              }
            ],
            "operator": "AND"
		  },
		  {
			"cpe_match": [
			  {
				"cpe23Uri": "cpe:2.3:a:test:test:-:*:*:*:*:*:*:*",
				"vulnerable": true
			  }
			],
			"operator": "OR"
		  }
        ]
      }
    }
  ]
}
`

func TestStats(t *testing.T) {
	file, err := ioutil.TempFile("/tmp", "test_nvd.json")
	assert.Nil(t, err, "Unexpected error occurred when creating a temp file")
	defer func() {
		assert.Nil(t, file.Close(), "Unexpected error occurred when closing a temp file")
		assert.Nil(t, os.Remove(file.Name()), "Unexpected error occurred when removing a temp file")
	}()
	_, err = file.WriteString(testFeed)
	assert.Nil(t, err, "Unexpected error occurred when writing to a temp file")
	feedDict, err := LoadJSONDictionary(file.Name())
	assert.Nil(t, err, "Unexpected error occurred when loading a test NVD JSON feed file")

	stats := NewStats()
	stats.Gather(feedDict)

	orgStdOut := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	stats.ReportOperatorAND()
	w.Close()
	output, _ := ioutil.ReadAll(r)
	os.Stdout = orgStdOut

	expectedOutput := `Total rules with AND operator: 75.00%
66.67%: (a AND h)
33.33%: (o AND (a OR h))
`
	assert.Equal(t, expectedOutput, string(output))
}
