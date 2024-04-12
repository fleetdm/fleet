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
	"time"

	"github.com/facebookincubator/flog"
	nvd "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd/schema"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
)

const (
	timeLayout = "2006-01-02T15:04:05.000Z"
)

type configuration struct {
	Cpe23Uri       string // key
	Affected       []affected
	HasFixedBy     bool
	FixedByVersion string
}

type affected struct {
	Version string
	Prior   bool
}

func convertTime(idefenseTime string) (string, error) {
	t, err := time.Parse(timeLayout, idefenseTime)
	if err != nil { // should be parsable
		return "", err
	}
	return t.Format(nvd.TimeLayout), nil
}

func (item *Vulnerability) findConfigurations() []configuration {
	configMap := make(map[string]configuration)

	if item.Affects == nil {
		return confMap2Slice(configMap)
	}

	// add affected vuln tech
	for _, vulnTech := range item.Affects.VulnTechs {
		attrs, err := createAttributes(vulnTech.Part, vulnTech.Vendor, vulnTech.Product)
		if err != nil {
			flog.Errorln(err)
			continue
		}
		cpe23Uri := attrs.BindToFmtString()

		config, ok := configMap[cpe23Uri]
		if !ok {
			config = configuration{Cpe23Uri: cpe23Uri}
		}
		config.Affected = append(config.Affected, affected{
			Version: vulnTech.Version,
			Prior:   vulnTech.AndPriorVersions,
		})
		configMap[cpe23Uri] = config
	}

	// add affected packages
	for _, pkg := range item.Affects.Packages {
		attrs, err := createAttributes("a", "", pkg.PackageName)
		if err != nil {
			flog.Errorln(err)
			continue
		}
		cpe23Uri := attrs.BindToFmtString()

		config, ok := configMap[cpe23Uri]
		if !ok {
			config = configuration{Cpe23Uri: cpe23Uri}
		}
		config.Affected = append(config.Affected, affected{
			Version: pkg.PackageVersion,
			Prior:   pkg.AndPriorVersions,
		})
		configMap[cpe23Uri] = config
	}

	if item.FixedBy == nil {
		return confMap2Slice(configMap)
	}

	// add vuln tech fixes
	for _, vulnTech := range item.FixedBy.VulnTechs {
		attrs, err := createAttributes(vulnTech.Part, vulnTech.Vendor, vulnTech.Product)
		if err != nil {
			flog.Errorln(err)
			continue
		}
		cpe23Uri := attrs.BindToFmtString()

		if config, ok := configMap[cpe23Uri]; ok {
			config.HasFixedBy = true
			config.FixedByVersion = vulnTech.Version
			configMap[cpe23Uri] = config
		}
	}

	// add package fixes
	for _, pkg := range item.FixedBy.Packages {
		attrs, err := createAttributes("a", "", pkg.PackageName)
		if err != nil {
			flog.Errorln(err)
			continue
		}
		cpe23Uri := attrs.BindToFmtString()

		if config, ok := configMap[cpe23Uri]; ok {
			config.HasFixedBy = true
			config.FixedByVersion = pkg.PackageVersion
			configMap[cpe23Uri] = config
		}
	}

	return confMap2Slice(configMap)
}

func confMap2Slice(m map[string]configuration) []configuration {
	s := make([]configuration, len(m))
	for _, cfg := range m {
		s = append(s, cfg)
	}
	return s
}

func createAttributes(part, vendor, product string) (*wfn.Attributes, error) {
	var err error
	if part, err = wfn.WFNize(part); err != nil {
		return nil, err
	}
	if vendor, err = wfn.WFNize(vendor); err != nil {
		return nil, err
	}
	if product, err = wfn.WFNize(product); err != nil {
		return nil, err
	}

	v := wfn.Attributes{
		Part:    part,
		Vendor:  vendor,
		Product: product,
	}

	return &v, nil
}
