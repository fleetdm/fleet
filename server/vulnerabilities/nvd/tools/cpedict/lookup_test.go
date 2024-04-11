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

package cpedict

import (
	"fmt"
	"strings"
	"testing"

	"github.com/facebookincubator/nvdtools/wfn"
)

func TestSearch(t *testing.T) {
	xmlStr := `
<?xml version='1.0' encoding='UTF-8'?>
<cpe-list xmlns:scap-core="http://scap.nist.gov/schema/scap-core/0.3" xmlns="http://cpe.mitre.org/dictionary/2.0" xmlns:config="http://scap.nist.gov/schema/configuration/0.1" xmlns:ns6="http://scap.nist.gov/schema/scap-core/0.1" xmlns:cpe-23="http://scap.nist.gov/schema/cpe-extension/2.3" xmlns:meta="http://scap.nist.gov/schema/cpe-dictionary-metadata/0.2" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://scap.nist.gov/schema/cpe-extension/2.3 https://scap.nist.gov/schema/cpe/2.3/cpe-dictionary-extension_2.3.xsd http://cpe.mitre.org/dictionary/2.0 https://scap.nist.gov/schema/cpe/2.3/cpe-dictionary_2.3.xsd http://scap.nist.gov/schema/cpe-dictionary-metadata/0.2 https://scap.nist.gov/schema/cpe/2.1/cpe-dictionary-metadata_0.2.xsd http://scap.nist.gov/schema/scap-core/0.3 https://scap.nist.gov/schema/nvd/scap-core_0.3.xsd http://scap.nist.gov/schema/configuration/0.1 https://scap.nist.gov/schema/nvd/configuration_0.1.xsd http://scap.nist.gov/schema/scap-core/0.1 https://scap.nist.gov/schema/nvd/scap-core_0.1.xsd">
  <generator>
    <product_name>National Vulnerability Database (NVD)</product_name>
    <product_version>3.20</product_version>
    <schema_version>2.3</schema_version>
    <timestamp>2018-04-25T03:50:11.922Z</timestamp>
  </generator>
  <cpe-item name="cpe:/a:adobe:acrobat:_reader9.5.3">
    <title xml:lang="en-US">Adobe Acrobat Reader 9.5.3</title>
    <cpe-23:cpe23-item name="cpe:2.3:a:adobe:acrobat:_reader9.5.3:*:*:*:*:*:*:*"/>
  </cpe-item>
  <cpe-item name="cpe:/a:adobe:acrobat:6.0.1">
    <title xml:lang="en-US">Adobe Acrobat 6.0.1</title>
    <title xml:lang="ja-JP">アドビシステムズ アクロバット 6.0.1</title>
    <cpe-23:cpe23-item name="cpe:2.3:a:adobe:acrobat:6.0.1:*:*:*:*:*:*:*"/>
  </cpe-item>
  <cpe-item name="cpe:/a:adobe:acrobat:6.0.2">
    <title xml:lang="ja-JP">アドビシステムズ アクロバット 6.0.2</title>
    <title xml:lang="en-US">Adobe Acrobat 6.0.2</title>
    <cpe-23:cpe23-item name="cpe:2.3:a:adobe:acrobat:6.0.2:*:*:*:*:*:*:*"/>
  </cpe-item>
  <cpe-item name="cpe:/a:adobe:acrobat:6.0.3">
    <title xml:lang="en-US">Adobe Acrobat 6.0.3</title>
    <title xml:lang="ja-JP">アドビシステムズ アクロバット 6.0.3</title>
    <cpe-23:cpe23-item name="cpe:2.3:a:adobe:acrobat:6.0.3:*:*:*:*:*:*:*"/>
  </cpe-item>
  <cpe-item name="cpe:/a:adobe:acrobat:6.0.4">
    <title xml:lang="en-US">Adobe Acrobat 6.0.4</title>
    <title xml:lang="ja-JP">アドビシステムズ アクロバット 6.0.4</title>
    <cpe-23:cpe23-item name="cpe:2.3:a:adobe:acrobat:6.0.4:*:*:*:*:*:*:*"/>
  </cpe-item>
  <cpe-item name="cpe:/a:adobe:acrobat:6.0.5">
    <title xml:lang="ja-JP">アドビシステムズ アクロバット 6.0.5</title>
    <title xml:lang="en-US">Adobe Acrobat 6.0.5</title>
    <cpe-23:cpe23-item name="cpe:2.3:a:adobe:acrobat:6.0.5:*:*:*:*:*:*:*"/>
  </cpe-item>
  <cpe-item name="cpe:/a:adobe:acrobat:6.0.6">
    <title xml:lang="en-US">Adobe Acrobat 6.0.6</title>
    <cpe-23:cpe23-item name="cpe:2.3:a:adobe:acrobat:6.0.6:*:*:*:*:*:*:*"/>
  </cpe-item>
  <cpe-item name="cpe:/a:adobe:acrobat:8.0">
    <title xml:lang="en-US">Adobe Acrobat 8.0</title>
    <title xml:lang="ja-JP">アドビシステムズ アクロバット 8.0</title>
    <cpe-23:cpe23-item name="cpe:2.3:a:adobe:acrobat:8.0:*:*:*:*:*:*:*"/>
  </cpe-item>
  <cpe-item name="cpe:/a:adobe:acrobat:8.0.0" deprecated="true" deprecation_date="2011-03-24T16:01:18.500Z">
    <title xml:lang="ja-JP">アドビシステムズ アクロバット 8.0</title>
    <title xml:lang="en-US">Adobe Acrobat 8.0</title>
    <cpe-23:cpe23-item name="cpe:2.3:a:adobe:acrobat:8.0.0:*:*:*:*:*:*:*">
      <cpe-23:deprecation date="2011-03-24T12:01:18.500-04:00">
        <cpe-23:deprecated-by name="cpe:2.3:a:adobe:acrobat:8.0:*:*:*:*:*:*:*" type="NAME_CORRECTION"/>
      </cpe-23:deprecation>
    </cpe-23:cpe23-item>
  </cpe-item>
</cpe-list>
`
	dict, err := Decode(strings.NewReader(xmlStr))
	if err != nil {
		t.Fatalf("couldn't parse dictionary: %v", err)
	}

	cases := []struct {
		WfnURI       string
		Exact        bool
		MatchAs      MatchType
		NumMatches   int
		ExactMatches []string
	}{
		{"cpe:/a:adobe:acrobat:_reader9.6.0", false, None, 0, nil},
		{"cpe:/a:adobe:acrobat:_reader9.5.3", true, Exact, 1, nil},
		{"cpe:/a:adobe:acrobat:_reader9.5.3", false, Superset, 1, nil},
		{"cpe:/a:adobe:acrobat:6.0.*", false, Superset, 6, nil},
		{"cpe:/a:adobe:acrobat:6.0.6:-:pro", false, Subset, 1, nil},
		{"cpe:/a:adobe:acrobat:8.0.0", false, Superset, -1, []string{"cpe:/a:adobe:acrobat:8.0"}},
	}
	for n, c := range cases {
		t.Run(fmt.Sprintf("case_%d", n), func(t *testing.T) {
			attr, err := wfn.UnbindURI(c.WfnURI)
			if err != nil {
				t.Fatalf("failed to unbind CPE name %q: %v", c.WfnURI, err)
			}
			mm, as := dict.Search(NamePattern(*attr), c.Exact)
			if as != c.MatchAs {
				t.Fatalf("wrong match: expected %v, got %v", c.MatchAs, as)
			}
			if c.NumMatches == -1 {
				for _, em := range c.ExactMatches {
					attr, _ := wfn.UnbindURI(em)
					if !contains(mm, attr) {
						t.Fatalf("matches were expected to contain %v but don't\n%v", attr, mm)
					}
				}
			} else if len(mm) != c.NumMatches {
				t.Fatalf("wrong # of matches: expected %d, got %d", c.NumMatches, len(mm))
			}
		})
	}
}

func contains(items []CPEItem, name *wfn.Attributes) bool {
	for _, item := range items {
		cmp, _ := wfn.Compare(name, (*wfn.Attributes)(&item.Name))
		if cmp.IsEqual() {
			return true
		}
	}
	return false
}
