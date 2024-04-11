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

package rustsec

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"strings"
	"testing"
)

func TestConvertAdvisory(t *testing.T) {
	cve, err := ConvertAdvisory(bytes.NewBufferString(sampleAdvisory))
	if err != nil {
		t.Fatal(err)
	}

	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetIndent("", "\t")

	err = enc.Encode(cve)
	if err != nil {
		t.Fatal(err)
	}

	have := b.String()
	want := goldenCVE

	err = diff(have, want)
	if err != nil {
		t.Fatal(err)
	}
}

func diff(a, b string) error {
	f1, err := ioutil.TempFile("", "rustsec2nvd")
	if err != nil {
		return err
	}
	defer f1.Close()

	f2, err := ioutil.TempFile("", "rustsec2nvd")
	if err != nil {
		return err
	}
	defer f2.Close()

	if _, err = io.WriteString(f1, a); err != nil {
		return err
	}

	if _, err = io.WriteString(f2, b); err != nil {
		return err
	}

	var ob, eb bytes.Buffer
	cmd := exec.Command("diff", f1.Name(), f2.Name())
	cmd.Stdout = &ob
	cmd.Stderr = &eb

	err = cmd.Run()
	if err != nil {
		var sb strings.Builder
		fmt.Fprintf(&sb, "%v\n", err)
		fmt.Fprintf(&sb, "diff command: diff %s %s\n", f1.Name(), f2.Name())
		fmt.Fprintf(&sb, "diff stdout:\n%s\n--end--\n", ob.String())
		fmt.Fprintf(&sb, "diff stderr:\n%s\n--end--\n", eb.String())
		return errors.New(sb.String())
	}

	return nil
}

// copied from https://github.com/RustSec/advisory-db
// uncommented all settings, added patched ^1.2.1
var sampleAdvisory = "```toml\n" + `[advisory]
# Identifier for the advisory (mandatory). Will be assigned a "RUSTSEC-YYYY-NNNN"
# identifier e.g. RUSTSEC-2018-0001. Please use "RUSTSEC-0000-0000" in PRs.
id = "RUSTSEC-0000-0000"

# Name of the affected crate (mandatory)
package = "mycrate"

# Disclosure date of the advisory as an RFC 3339 date (mandatory)
date = "2017-02-25"

# Single-line description of a vulnerability (mandatory)
title = "Flaw in X allows Y"

# Enter a short-form description of the vulnerability here (mandatory)
description = """
Affected versions of this crate did not properly X.

This allows an attacker to Y.

The flaw was corrected by Z.
"""

# Versions which include fixes for this vulnerability (mandatory)
patched_versions = [">= 1.2.0", "^1.2.1"]

# Versions which were never vulnerable (optional)
unaffected_versions = ["< 1.1.0"]

# URL to a long-form description of this issue, e.g. a GitHub issue/PR,
# a change log entry, or a blogpost announcing the release (optional)
url = "https://github.com/mystuff/mycrate/issues/123"

# Keywords which describe this vulnerability, similar to Cargo (optional)
keywords = ["ssl", "mitm"]

# Vulnerability aliases, e.g. CVE IDs (optional but recommended)
# Request a CVE for your RustSec vulns: https://iwantacve.org/
aliases = ["CVE-2018-XXXX"]

# References to related vulnerabilities (optional)
# e.g. CVE for a C library wrapped by a -sys crate)
references = ["CVE-2018-YYYY", "CVE-2018-ZZZZ"]

# CPU architectures impacted by this vulnerability (optional)
# For a list of CPU architecture strings, see the "platforms" crate:
# <https://docs.rs/platforms/latest/platforms/target/enum.Arch.html>
affected_arch = ["x86", "x86_64"]

# Operating systems impacted by this vulnerability (optional)
# For a list of OS strings, see the "platforms" crate:
# <https://docs.rs/platforms/latest/platforms/target/enum.OS.html>
affected_os = ["windows"]

# List of canonical paths to vulnerable functions (optional)
# The path syntax is cratename::path::to::function, without any
# return type or parameters. More information:
# <https://github.com/RustSec/advisory-db/issues/68>
# For example, for RUSTSEC-2018-0003, this would look like:
affected_functions = ["smallvec::SmallVec::insert_many"]
` + "```" + `

# Flaw in X allows Y

Affected versions of this crate did not properly X.

This allows an attacker to Y.

The flaw was corrected by Z.
`

var goldenCVE = `{
	"cve": {
		"affects": null,
		"CVE_data_meta": {
			"ASSIGNER": "RustSec",
			"ID": "RUSTSEC-0000-0000"
		},
		"data_format": "MITRE",
		"data_type": "CVE",
		"data_version": "4.0",
		"description": {
			"description_data": [
				{
					"lang": "en",
					"value": "Affected versions of this crate did not properly X.\n\nThis allows an attacker to Y.\n\nThe flaw was corrected by Z.\n"
				}
			]
		},
		"problemtype": null,
		"references": {
			"reference_data": [
				{
					"name": "CVE-2018-XXXX",
					"url": ""
				},
				{
					"name": "CVE-2018-YYYY",
					"url": ""
				},
				{
					"name": "CVE-2018-ZZZZ",
					"url": ""
				},
				{
					"name": "Flaw in X allows Y",
					"url": "https://github.com/mystuff/mycrate/issues/123"
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
								"cpe_name": [
									{
										"cpe22Uri": "cpe:/a::mycrate",
										"cpe23Uri": "cpe:2.3:a:*:mycrate:*:*:*:*:*:*:*:*"
									}
								],
								"cpe23Uri": "cpe:2.3:a:*:mycrate:*:*:*:*:*:*:*:*",
								"versionStartIncluding": "0",
								"vulnerable": false
							}
						]
					},
					{
						"cpe_match": [
							{
								"cpe_name": [
									{
										"cpe22Uri": "cpe:/a::mycrate",
										"cpe23Uri": "cpe:2.3:a:*:mycrate:*:*:*:*:*:*:*:*"
									}
								],
								"cpe23Uri": "cpe:2.3:a:*:mycrate:*:*:*:*:*:*:*:*",
								"versionEndExcluding": "1.1.0",
								"vulnerable": false
							},
							{
								"cpe_name": [
									{
										"cpe22Uri": "cpe:/a::mycrate",
										"cpe23Uri": "cpe:2.3:a:*:mycrate:*:*:*:*:*:*:*:*"
									}
								],
								"cpe23Uri": "cpe:2.3:a:*:mycrate:*:*:*:*:*:*:*:*",
								"versionStartIncluding": "1.2.0",
								"vulnerable": false
							},
							{
								"cpe_name": [
									{
										"cpe22Uri": "cpe:/a::mycrate:1.2.1",
										"cpe23Uri": "cpe:2.3:a:*:mycrate:1.2.1:*:*:*:*:*:*:*"
									}
								],
								"cpe23Uri": "cpe:2.3:a:*:mycrate:1.2.1:*:*:*:*:*:*:*",
								"vulnerable": false
							}
						],
						"negate": true
					}
				],
				"operator": "AND"
			}
		]
	},
	"lastModifiedDate": "2017-02-25T00:00Z",
	"publishedDate": "2017-02-25T00:00Z"
}
`
