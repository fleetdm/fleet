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
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/facebookincubator/flog"
	nvd "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd/schema"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/providers/lib/client"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/stats"
)

// Convertible is any struct which knows how to convert itself to NVD CVE Item
type Convertible interface {
	// ID should return vulnerabilities ID
	ID() string
	// Convert should return a new CVE Item, or an error if it's not possible
	Convert() (*nvd.NVDCVEFeedJSON10DefCVEItem, error)
}

// Read should read the vulnerabilities from the given reader and push them into the channel
// The contents of the reader should be a slice of structs which are convertibles
// channel will be created and mustn't be closed
type Read func(io.Reader, chan Convertible) error

// FetchSince knows how to fetch vulnerabilities from an API
// it should create a new channel, fetch everything concurrently and close the channel
type FetchSince func(ctx context.Context, c client.Client, baseURL string, since int64) (<-chan Convertible, error)

// Runner knows how to run everything together, based on the config values
// if config.Download is set, it will use the fetcher, otherwise it will use Reader to read stdin or files
type Runner struct {
	Config
	FetchSince
	Read
}

// Run should be called in main function of the converter
// It will run the fetchers/runners (and convert vulnerabilities)
// Finally, it will output it as json to stdout
func (r *Runner) Run() error {
	r.Config.addFlags()
	stats.AddFlags()
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}
	flag.Parse()

	defer func(startTime time.Time) {
		stats.TrackTime("run.time", startTime, time.Second)
		stats.WriteAndLogError()
	}(time.Now())

	if err := r.Config.validate(); err != nil {
		return fmt.Errorf("config is invalid: %v", err)
	}

	var vulns <-chan Convertible
	var err error
	if r.Config.download {
		vulns, err = r.downloadVulnerabilities(context.Background())
	} else {
		vulns, err = r.readVulnerabilities()
	}
	if err != nil {
		return fmt.Errorf("couldn't get vulnerabilities: %v", err)
	}

	if r.Config.convert {
		if err := convert(vulns); err != nil {
			return fmt.Errorf("failed to convert vulns: %v", err)
		}
		return nil
	}

	m := make(map[string]Convertible)
	for v := range vulns {
		m[v.ID()] = v
	}
	if err := json.NewEncoder(os.Stdout).Encode(m); err != nil {
		return fmt.Errorf("couldn't write vulnerabilities: %v", err)
	}

	return nil
}

func (r *Runner) downloadVulnerabilities(ctx context.Context) (<-chan Convertible, error) {
	c := client.Default()
	c = r.Config.ClientConfig.Configure(c)
	return r.FetchSince(ctx, c, r.Config.BaseURL, int64(r.Config.downloadSince))
}

func (r *Runner) readVulnerabilities() (<-chan Convertible, error) {
	vulns := make(chan Convertible)

	if flag.NArg() == 0 {
		// read from stdin
		go func() {
			defer close(vulns)
			if err := r.Read(os.Stdin, vulns); err != nil {
				flog.Errorf("error while reading from stdin: %v", err)
			}
		}()
		return vulns, nil
	}

	// read from files in args
	wg := sync.WaitGroup{}
	for _, filename := range flag.Args() {
		wg.Add(1)
		go func(filename string) {
			defer wg.Done()
			file, err := os.Open(filename)
			if err != nil {
				flog.Errorf("couldn't open file %q: %v", filename, err)
				return
			}
			defer file.Close()
			if err := r.Read(file, vulns); err != nil {
				flog.Errorf("error while reading from file %q: %v", filename, err)
			}
		}(filename)
	}
	go func() {
		defer close(vulns)
		wg.Wait()
	}()

	return vulns, nil
}

// getNVDFeed will convert the vulns in channel to NVD Feed
func convert(vulns <-chan Convertible) error {
	defer stats.TrackTime("convert.time", time.Now(), time.Second)
	var feed nvd.NVDCVEFeedJSON10
	for vuln := range vulns {
		converted, err := vuln.Convert()
		if err != nil {
			flog.Errorf("error while converting vuln: %v", err)
			continue
		}
		feed.CVEItems = append(feed.CVEItems, converted)
	}

	if err := json.NewEncoder(os.Stdout).Encode(feed); err != nil {
		return fmt.Errorf("couldn't write NVD feed: %v", err)
	}
	return nil
}
