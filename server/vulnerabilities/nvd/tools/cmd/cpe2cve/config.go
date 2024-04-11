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
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/BurntSushi/toml"
)

type config struct {
	// input fields
	CPEsAt int
	// output fields
	CVEsAt     int
	MatchesAt  int
	CWEsAt     int
	ProviderAt int
	// output score fields
	CVSS2At int
	CVSS3At int
	CVSSAt  int
	// output deleted fields
	EraseFields fieldsToSkip // []int

	// separators
	InFieldSeparator   string
	InRecordSeparator  string
	OutFieldSeparator  string
	OutRecordSeparator string

	// optimizations
	NumProcessors  int
	IndexDict      bool
	CacheSize      int64
	RequireVersion bool

	// profiling
	CPUProfile    string
	MemoryProfile string

	// feeds
	FeedOverrides multiString // []string
	Feeds         map[string][]string
}

func (cfg *config) addFlags() {
	// input
	flag.IntVar(&cfg.CPEsAt, "cpe", 0, "look for CPE names in input at this position (starts with 1)")

	// output
	flag.IntVar(&cfg.CVEsAt, "cve", 0, "output CVEs at this position (starts with 1)")
	flag.IntVar(&cfg.MatchesAt, "matches", 0, "output CPEs that matches CVE at this position; 0 disables the output")
	flag.IntVar(&cfg.CWEsAt, "cwe", 0, "output problem types (CWEs) at this position (starts with 1)")
	flag.IntVar(&cfg.ProviderAt, "provider_field", 0, "where should the provider be placed in the output (starts with 1).")
	flag.IntVar(&cfg.CVSS2At, "cvss2", 0, "output CVSS 2.0 base score at this position (starts with 1)")
	flag.IntVar(&cfg.CVSS3At, "cvss3", 0, "output CVSS 3.0 base score at this position (starts with 1)")
	flag.IntVar(&cfg.CVSSAt, "cvss", 0, "output CVSS base score (v3 if available, v2 otherwise) at this position (starts with 1)")
	flag.Var(&cfg.EraseFields, "e", "comma separated list of fields to erase from output; starts at 1, supports ranges (e.g. 1-3); processed before the vulnerablitie field added")

	// separators
	flag.StringVar(&cfg.InFieldSeparator, "d", "\t", "input columns delimiter")
	flag.StringVar(&cfg.InRecordSeparator, "d2", ",", "inner input columns delimiter: separates elements of list passed into a CSV columns")
	flag.StringVar(&cfg.OutFieldSeparator, "o", "\t", "output columns delimiter")
	flag.StringVar(&cfg.OutRecordSeparator, "o2", ",", "inner output columns delimiter: separates elements of lists in output CSV columns")

	// optimizations
	flag.IntVar(&cfg.NumProcessors, "nproc", 1, "number of concurrent goroutines that perform CVE lookup")
	flag.BoolVar(&cfg.IndexDict, "idxd", false, "build and use an index for CVE dictionary: increases the processing speed, but might miss some matches")
	flag.Int64Var(&cfg.CacheSize, "cache_size", 0, "limit the cache size to this amount in bytes; 0 removes the limit, -1 disables caching")
	flag.BoolVar(&cfg.RequireVersion, "require_version", false, "ignore matches of CPEs with version ANY")

	// profiling
	flag.StringVar(&cfg.CPUProfile, "cpuprofile", "", "file to store CPU profile data to; empty value disables CPU profiling")
	flag.StringVar(&cfg.MemoryProfile, "memprofile", "", "file to store memory profile data to; empty value disables memory profiling")

	// feeds
	flag.Var(&cfg.FeedOverrides, "r", "overRide: path to override feed, can be specified multiple times")
}

func (cfg *config) addFeedsFromArgs(provider string, feedFiles ...string) {
	if cfg.Feeds == nil {
		cfg.Feeds = make(map[string][]string)
	}
	cfg.Feeds[provider] = append(cfg.Feeds[provider], feedFiles...)
}

func (cfg *config) validate() error {
	if len(cfg.Feeds) == 0 {
		return fmt.Errorf("feed files weren't provided")
	}
	if cfg.ProviderAt != 0 {
		for provider, feed := range cfg.Feeds {
			if provider == "" {
				return fmt.Errorf("need to specify all providers when using provider in the output, but wasn't specified for feed %q", feed)
			}
		}
	}

	if cfg.CPEsAt <= 0 {
		return fmt.Errorf("-cpe flag wasn't provided")
	}
	if cfg.CVEsAt <= 0 {
		return fmt.Errorf("-cve flag wasn't provided")
	}
	if cfg.MatchesAt < 0 {
		return fmt.Errorf("-matches value is invalid %d", cfg.MatchesAt)
	}
	if cfg.CWEsAt < 0 {
		return fmt.Errorf("-cwe value is invalid %d", cfg.CWEsAt)
	}
	if cfg.CVSSAt < 0 {
		return fmt.Errorf("-cvss2 value is invalid %d", cfg.CVSS2At)
	}
	if cfg.CVSS3At < 0 {
		return fmt.Errorf("-cvss2 value is invalid %d", cfg.CVSS3At)
	}
	if cfg.CVSSAt < 0 {
		return fmt.Errorf("-cvss value is invalid %d", cfg.CVSSAt)
	}
	return nil
}

func readConfigFile(file string) (config, error) {
	f, err := os.Open(file)
	if err != nil {
		return config{}, err
	}
	defer f.Close()

	var cfg config

	switch ext := path.Ext(file); ext {
	case ".json":
		// Example:
		// {
		// 	"CVEsAt": 3,
		// 	"ProviderAt": 2,
		// 	"Feeds": {
		// 	  "foo": ["foo.json"],
		// 	  "bar": ["bar.json", "bar2.json.gz"]
		// 	 }
		// }
		err = json.NewDecoder(f).Decode(&cfg)
	case ".toml":
		// Example:
		// CVEsAt = 3
		// ProviderAt = 2
		// [Feeds]
		// foo = ["foo.json"]
		// bar = ["bar.json", "bar2.json.gz"]
		_, err = toml.NewDecoder(f).Decode(&cfg)
	default:
		return cfg, fmt.Errorf("unsupported file extension: %q", ext)
	}

	if err == nil {
		// set defaults
		if cfg.InFieldSeparator == "" {
			cfg.InFieldSeparator = "\t"
		}
		if cfg.InRecordSeparator == "" {
			cfg.InRecordSeparator = ","
		}
		if cfg.OutFieldSeparator == "" {
			cfg.OutFieldSeparator = "\t"
		}
		if cfg.OutRecordSeparator == "" {
			cfg.OutRecordSeparator = ","
		}
		if cfg.NumProcessors == 0 {
			cfg.NumProcessors = 1
		}
	}

	return cfg, err
}

func writeConfigFileDefinition(w io.Writer) {
	cfg := config{
		EraseFields:   fieldsToSkip{1: true},
		FeedOverrides: multiString{"override feed path"},
		Feeds:         map[string][]string{"provider": []string{"feed file 1", "feed file 2"}},
	}
	e := json.NewEncoder(w)
	e.SetIndent("\t", "\t")
	fmt.Fprint(w, "\nConfig file definition:\n\t")
	e.Encode(cfg)
	fmt.Fprint(w, "\n")
}
