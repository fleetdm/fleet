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
	"flag"
	"os"
	"reflect"
)

// SourceConfig is the configuration of the NVD data feed source.
type SourceConfig struct {
	Scheme      string `envconfig:"NVDSYNC_SCHEME" default:"https"`
	Host        string `envconfig:"NVDSYNC_HOST" default:"nvd.nist.gov"`
	CVEFeedPath string `envconfig:"NVDSYNC_CVE_FEED_PATH" default:"/feeds/{{.Encoding}}/cve/{{.Version}}/"`
	CPEFeedPath string `envconfig:"NVDSYNC_CPE_FEED_PATH" default:"/feeds/xml/cpe/dictionary/"`
}

// NewSourceConfig creates and initializes a new SourceConfig with values from envconfig.
func NewSourceConfig() *SourceConfig {
	sc := &SourceConfig{}

	valueFromStructTag := func(f reflect.StructField) string {
		k := f.Tag.Get("envconfig")
		if v := os.Getenv(k); v != "" {
			return v
		}
		return f.Tag.Get("default")
	}

	t := reflect.TypeOf(sc).Elem()
	p := reflect.ValueOf(sc).Elem()
	for i := 0; i < p.NumField(); i++ {
		field := t.Field(i)
		value := reflect.ValueOf(valueFromStructTag(field))
		p.Field(i).Set(value)
	}

	return sc
}

// AddFlags adds SourceConfig flags to the given FlagSet.
func (src *SourceConfig) AddFlags(_ *flag.FlagSet) {
	flag.StringVar(&src.Scheme, "src_scheme", src.Scheme, "source scheme\nenv: NVDSYNC_SCHEME")
	flag.StringVar(&src.Host, "src_host", src.Host, "source host\nenv: NVDSYNC_HOST")
	flag.StringVar(&src.CVEFeedPath, "src_cve_feed_path", src.CVEFeedPath, "source path for CVE feeds\nenv: NVDSYNC_CVE_FEED_PATH")
	flag.StringVar(&src.CPEFeedPath, "src_cpe_feed_path", src.CPEFeedPath, "source path for CPE feeds\nenv: NVDSYNC_CPE_FEED_PATH")
}
