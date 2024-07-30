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
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
)

func TestDecode(t *testing.T) {
	xmlStr := `
<?xml version='1.0' encoding='UTF-8'?>
<cpe-list xmlns:scap-core="http://scap.nist.gov/schema/scap-core/0.3" xmlns="http://cpe.mitre.org/dictionary/2.0" xmlns:config="http://scap.nist.gov/schema/configuration/0.1" xmlns:ns6="http://scap.nist.gov/schema/scap-core/0.1" xmlns:cpe-23="http://scap.nist.gov/schema/cpe-extension/2.3" xmlns:meta="http://scap.nist.gov/schema/cpe-dictionary-metadata/0.2" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://scap.nist.gov/schema/cpe-extension/2.3 https://scap.nist.gov/schema/cpe/2.3/cpe-dictionary-extension_2.3.xsd http://cpe.mitre.org/dictionary/2.0 https://scap.nist.gov/schema/cpe/2.3/cpe-dictionary_2.3.xsd http://scap.nist.gov/schema/cpe-dictionary-metadata/0.2 https://scap.nist.gov/schema/cpe/2.1/cpe-dictionary-metadata_0.2.xsd http://scap.nist.gov/schema/scap-core/0.3 https://scap.nist.gov/schema/nvd/scap-core_0.3.xsd http://scap.nist.gov/schema/configuration/0.1 https://scap.nist.gov/schema/nvd/configuration_0.1.xsd http://scap.nist.gov/schema/scap-core/0.1 https://scap.nist.gov/schema/nvd/scap-core_0.1.xsd">
  <generator>
    <product_name>National Vulnerability Database (NVD)</product_name>
    <product_version>3.20</product_version>
    <schema_version>2.3</schema_version>
    <timestamp>2018-04-25T03:50:11.922Z</timestamp>
  </generator>
  <cpe-item name="cpe:/a:%240.99_kindle_books_project:%240.99_kindle_books:6::~~~android~~">
    <title xml:lang="en-US">$0.99 Kindle Books project $0.99 Kindle Books (aka com.kindle.books.for99) for android 6.0</title>
    <references>
      <reference href="https://play.google.com/store/apps/details?id=com.kindle.books.for99">Product information</reference>
      <reference href="https://docs.google.com/spreadsheets/d/1t5GXwjw82SyunALVJb2w0zi3FoLRIkfGPc7AMjRF0r4/edit?pli=1#gid=1053404143">Government Advisory</reference>
    </references>
    <cpe-23:cpe23-item name="cpe:2.3:a:\$0.99_kindle_books_project:\$0.99_kindle_books:6:*:*:*:*:android:*:*"/>
  </cpe-item>
  <cpe-item name="cpe:/a:adobe:flex_sdk:-">
    <title xml:lang="ja-JP">アドビシステムズ Flex</title>
    <title xml:lang="en-US">Adobe Flex</title>
    <cpe-23:cpe23-item name="cpe:2.3:a:adobe:flex_sdk:-:*:*:*:*:*:*:*"/>
  </cpe-item>
  <cpe-item name="cpe:/a:3com:tippingpoint_ips_tos:2.1.3.6323" deprecated="true" deprecation_date="2010-12-28T17:35:59.740Z">
    <title xml:lang="en-US">3Com TippingPoint IPS TOS 2.1.3.6323</title>
    <cpe-23:cpe23-item name="cpe:2.3:a:3com:tippingpoint_ips_tos:2.1.3.6323:*:*:*:*:*:*:*">
      <cpe-23:deprecation date="2010-12-28T12:35:59.740-05:00">
        <cpe-23:deprecated-by name="cpe:2.3:o:3com:tippingpoint_ips_tos:2.1.3.6323:*:*:*:*:*:*:*" type="NAME_CORRECTION"/>
      </cpe-23:deprecation>
    </cpe-23:cpe23-item>
  </cpe-item>
</cpe-list>
`
	data, err := Decode(strings.NewReader(xmlStr))
	if err != nil {
		t.Fatalf("failed to decode xml: %v", err)
	}

	gentm, _ := time.Parse(time.RFC3339, "2018-04-25T03:50:11.922Z")
	generator := Generator{"National Vulnerability Database (NVD)", "3.20", "2.3", gentm}
	if data.Generator != generator {
		t.Errorf("bad generator:\n\texpected %+v\n\tgot %+v", data.Generator, generator)
	}

	wfname, _ := wfn.Parse("cpe:/a:%240.99_kindle_books_project:%240.99_kindle_books:6::~~~android~~")
	item := data.Items[0]
	if item.Name != item.CPE23.Name || item.Name != NamePattern(*wfname) {
		t.Errorf("bad CPE name:\n\t2.2 is %+v\n\t2.3 is %+v", item.Name, item.CPE23.Name)
	}
	if len(item.References) != 2 {
		t.Errorf("item was expected to have 2 references, %d found\n\t%v", len(item.References), item)
	}

	wfname, _ = wfn.Parse("cpe:2.3:o:3com:tippingpoint_ips_tos:2.1.3.6323:*:*:*:*:*:*:*")
	name := NamePattern(*wfname)
	deptm, _ := time.Parse(time.RFC3339, "2010-12-28T17:35:59.740Z")
	item = data.Items[len(data.Items)-1]
	if !item.Deprecated {
		t.Errorf("item was expected to be deprecated, but isn't:\n\t%+v", item)
	}
	if !item.DeprecationDate.Equal(deptm) {
		t.Errorf("item's deprecation time was expected to be\n\t%v\ngot\n\t%v", deptm, item.DeprecationDate)
	}
	if item.CPE23.Deprecation == nil {
		t.Fatal("item was expected to have Deprecation info, but it doesn't")
	}
	dep := item.CPE23.Deprecation
	if !dep.Date.Equal(item.DeprecationDate) {
		t.Errorf("cpe23 deprecation date doesn't match the cpe22 one:\n\t%v\n\t%v", dep.Date, item.DeprecationDate)
	}
	if dep.DeprecatedBy[0].Name != name {
		t.Errorf("item was expected to be deprecated by\n\t%v\n\tgot %v", dep.DeprecatedBy[0].Name, name)
	}
	if dep.DeprecatedBy[0].Type != "NAME_CORRECTION" {
		t.Errorf("item was expected to be deprecated because of NAME_CORRECTION, got %v", dep.DeprecatedBy[0].Type)
	}
}
