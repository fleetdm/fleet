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

package runner

import (
	"flag"
	"fmt"
	"strconv"
	"time"

	nvd "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd/schema"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/providers/lib/client"
)

// Config is used to configure the execution of the converter
type Config struct {
	BaseURL       string
	ClientConfig  client.Config
	download      bool
	convert       bool
	downloadSince sinceTS
}

func (c *Config) addFlags() {
	flag.StringVar(&c.BaseURL, "base_url", c.BaseURL, "API base URL")
	c.ClientConfig.AddFlags()
	flag.BoolVar(&c.download, "download", false, "Should the data be downloaded or read from stdin/files")
	flag.BoolVar(&c.convert, "convert", false, "Should the feed be converted to NVD format or not")
	flag.Var(&c.downloadSince, "since", fmt.Sprintf("Since when to download. It can be a timestamp, golang duration or time in %q format. Default is timestamp=0", nvd.TimeLayout))
}

func (c *Config) validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("need to specify base url")
	}
	if err := c.ClientConfig.Validate(); err != nil {
		return err
	}
	if c.downloadSince < 0 {
		return fmt.Errorf("negative timestamp used %d", c.downloadSince)
	}

	return nil
}

// sinceTS is a timestamp since when we should download
type sinceTS int64

// String implements flag.Value interface
func (ts *sinceTS) String() string {
	if ts == nil {
		return ""
	}
	return fmt.Sprintf("%d", ts)
}

// Set implements flag.Value interface
func (ts *sinceTS) Set(val string) error {
	if ts == nil {
		*ts = 0
	}
	// try to parse it as a timestamp
	if timestamp, err := strconv.ParseInt(val, 10, 64); err == nil {
		*ts = sinceTS(timestamp)
		return nil
	}
	// try to parse it as a duration
	if dur, err := time.ParseDuration(val); err == nil {
		*ts = sinceTS(time.Now().Add(-dur).Unix())
		return nil
	}

	if t, err := time.Parse(nvd.TimeLayout, val); err == nil {
		*ts = sinceTS(t.Unix())
		return nil
	}

	return fmt.Errorf("can't parse %q as since value", val)
}
