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

package schema

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	nvd "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd/schema"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
)

// we try to use this to extract name and version from their product
var productRegex = *regexp.MustCompile(`^(.+)\s+([0-9.x]+)$`)

func findCPEs(product *Product) ([]string, error) {
	if product.HasCpe {
		var cpes []string
		for _, cpe := range product.Cpes {
			cpes = append(cpes, cpe.Name)
		}
		return cpes, nil
	}

	name, version, err := extractNameAndVersion(product.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to extract product and version from %q: %v", product.Name, err)
	}

	part := "a"
	if product.IsOS {
		part = "o"
	}

	attrs, err := createAttributes(part, name, version)
	if err != nil {
		return nil, fmt.Errorf("failed to create attributes from (%q, %q, %q): %v", part, name, version, err)
	}
	attrs.Version = strings.Replace(attrs.Version, "x", "*", 1) // 7\.x -> 7\.*

	return []string{attrs.BindToURI()}, nil
}

func convertTime(Time string) (string, error) {
	t, err := time.Parse("2006-01-02T15:04:05Z", Time)
	if err != nil { // should be parsable
		return "", err
	}
	return t.Format(nvd.TimeLayout), nil
}

func extractNameAndVersion(product string) (name, version string, err error) {
	if match := productRegex.FindStringSubmatch(product); match != nil {
		return match[1], match[2], nil
	}
	return "", "", fmt.Errorf("couldn't extract name and version using regex")
}

func createAttributes(part, product, version string) (*wfn.Attributes, error) {
	var err error
	if part, err = wfn.WFNize(part); err != nil {
		return nil, fmt.Errorf("failed to wfnize part %q: %v", part, err)
	}
	if product, err = wfn.WFNize(product); err != nil {
		return nil, fmt.Errorf("failed to wfnize product %q: %v", product, err)
	}
	if version, err = wfn.WFNize(version); err != nil {
		return nil, fmt.Errorf("failed to wfnize version %q: %v", version, err)
	}

	v := wfn.Attributes{
		Part:    part,
		Product: product,
		Version: version,
	}

	return &v, nil
}
