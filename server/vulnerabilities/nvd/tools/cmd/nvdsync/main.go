// nvdsync is a command line tool to synchronize NVD data feeds to local directory.
//
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
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/facebookincubator/flog"
	"github.com/facebookincubator/nvdtools/providers/nvd"
)

func init() {
	flog.AddFlags(flag.CommandLine, nil)
	flag.Set("logtostderr", "true")
}

func main() {
	var (
		cvefeed   nvd.CVE
		cpefeed   nvd.CPE
		timeout   time.Duration
		userAgent string
		source    = nvd.NewSourceConfig()
	)

	flag.Var(&cvefeed, "cve_feed", cvefeed.Help())
	flag.Var(&cpefeed, "cpe_feed", cpefeed.Help())
	flag.DurationVar(&timeout, "timeout", 5*time.Minute, "sync timeout")
	flag.StringVar(&userAgent, "user_agent", nvd.UserAgent(), "HTTP request User-Agent header")
	source.AddFlags(flag.CommandLine)

	flag.Usage = func() {
		fmt.Printf("nvdsync %s\n\n", nvd.Version)
		fmt.Printf("use: %s [flags] dir\n", os.Args[0])
		fmt.Printf("Synchronizes NVD data feeds to local directory.\n\n")
		fmt.Printf("Flags:\n\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	flag.Parse()

	localdir := flag.Arg(0)
	if localdir == "" {
		flag.Usage()
	}

	// determine User-Agent header
	// check if it's only ascii characters
	if err := nvd.SetUserAgent(userAgent); err != nil {
		flog.Warningf("could not set User-Agent HTTP header, using default: %v", err)
	}
	flog.Infof("Using http User-Agent: %s", nvd.UserAgent())

	dfs := nvd.Sync{
		Feeds:    []nvd.Syncer{cvefeed, cpefeed},
		Source:   source,
		LocalDir: localdir,
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := dfs.Do(ctx); err != nil {
		flog.Fatal(err)
	}
}
