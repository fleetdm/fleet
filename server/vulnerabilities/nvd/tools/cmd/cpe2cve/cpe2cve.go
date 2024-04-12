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
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/facebookincubator/flog"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/stats"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
)

func processAll(in <-chan []string, out chan<- []string, caches map[string]*cvefeed.Cache, cfg config, nlines *uint64) {
	cpesAt := cfg.CPEsAt - 1
	for rec := range in {
		if cpesAt >= len(rec) {
			flog.Errorf("not enough fields in input (%d)", len(rec))
			continue
		}
		if stats.AreLogged() {
			stats.IncrementCounter("line.total")
		}
		cpeList := strings.Split(rec[cpesAt], cfg.InRecordSeparator)
		cpes := make([]*wfn.Attributes, 0, len(cpeList))
		for _, uri := range cpeList {
			if stats.AreLogged() {
				stats.IncrementCounter("cpe.total")
			}
			attr, err := wfn.Parse(uri)
			if err != nil {
				flog.Errorf("couldn't parse uri %q: %v", uri, err)
				continue
			}
			cpes = append(cpes, attr)
		}
		rec[cpesAt] = strings.Join(cpeList, cfg.OutRecordSeparator)

		// if performance seems to be the issue, we could try to make these cache.Get's concurrent:
		//
		// wg := sync.WaitGroup{}
		// for provider, cache := range caches {
		// 	provider, cache := provider, cache
		// 	wg.Add(1)
		// 	go func() {
		// 		defer wg.Done()
		// 		for _, matches := range cache.Get(cpes) {
		// ...
		for provider, cache := range caches {
			for _, matches := range cache.Get(cpes) {
				ml := len(matches.CPEs)
				if stats.AreLogged() {
					stats.IncrementCounterBy("cpe.match", int64(ml))
					if ml != 0 {
						stats.IncrementCounter("line.match")
					}
				}
				matchingCPEs := make([]string, ml)
				for i, attr := range matches.CPEs {
					if attr == nil {
						flog.Errorf("%s matches nil CPE", matches.CVE.ID())
						continue
					}
					matchingCPEs[i] = (*wfn.Attributes)(attr).BindToURI()
				}
				rec2 := make([]string, len(rec))
				copy(rec2, rec)
				cvss := matches.CVE.CVSSv3BaseScore()
				if cvss == 0 {
					cvss = matches.CVE.CVSSv2BaseScore()
				}
				rec2 = cfg.EraseFields.appendAt(
					rec2,
					cfg.CVEsAt-1, matches.CVE.ID(),
					cfg.MatchesAt-1, strings.Join(matchingCPEs, cfg.OutRecordSeparator),
					cfg.CWEsAt-1, strings.Join(matches.CVE.CWEs(), cfg.OutRecordSeparator),
					cfg.CVSS2At-1, fmt.Sprintf("%.1f", matches.CVE.CVSSv2BaseScore()),
					cfg.CVSS3At-1, fmt.Sprintf("%.1f", matches.CVE.CVSSv3BaseScore()),
					cfg.CVSSAt-1, fmt.Sprintf("%.1f", cvss),
					cfg.ProviderAt-1, provider,
				)
				out <- rec2
			}
		}

		n := atomic.AddUint64(nlines, 1)
		if n > 0 {
			if n%10000 == 0 {
				flog.V(1).Infoln(n, "lines processed")
			} else if n%1000 == 0 {
				flog.V(2).Infoln(n, "lines processed")
			} else if n%100 == 0 {
				flog.V(3).Infoln(n, "lines processed")
			}
		}
	}
}

func processInput(in io.Reader, out io.Writer, caches map[string]*cvefeed.Cache, cfg config) chan struct{} {
	done := make(chan struct{})
	procIn := make(chan []string)
	procOut := make(chan []string)

	r := csv.NewReader(in)
	r.Comma = rune(cfg.InFieldSeparator[0])

	w := csv.NewWriter(out)
	w.Comma = rune(cfg.OutFieldSeparator[0])

	// spawn processing goroutines
	var linesProcessed uint64
	var procWG sync.WaitGroup
	procWG.Add(cfg.NumProcessors)
	for i := 0; i < cfg.NumProcessors; i++ {
		go func() {
			processAll(procIn, procOut, caches, cfg, &linesProcessed)
			procWG.Done()
		}()
	}

	// write processed results in background
	go func() {
		for rec := range procOut {
			if err := w.Write(rec); err != nil {
				flog.Errorf("write error: %v", err)
			}
			w.Flush()
		}
		if err := w.Error(); err != nil {
			flog.Errorf("write error: %v", err)
		}
		close(done)
	}()

	start := time.Now()
	// main goroutine reads input and sends it to processors
	for line := 1; ; line++ {
		rec, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			flog.Errorf("read error at line %d: %v", line, err)
		}
		procIn <- rec
	}

	close(procIn)
	procWG.Wait()
	close(procOut)
	flog.V(1).Infof("processed %d lines in %v", linesProcessed, time.Since(start))
	return done
}

func init() {
	flog.AddFlags(flag.CommandLine, nil)
	stats.AddFlags()
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [flags] nvd_feed.xml.gz...\n", path.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "flags:\n")
		flag.PrintDefaults()
		if flog.V(1) {
			writeConfigFileDefinition(os.Stderr)
		}
		os.Exit(1)
	}
	flag.Set("logtostderr", "true")
}

func main() {
	// we do it like this because if we exit in Main, deferred functions don't get called
	os.Exit(Main())
}

func Main() int {
	var cfg config
	cfg.addFlags()
	provider := flag.String("provider", "", "feed provider. used as a provider name for the feeds passed in through the command line")
	cfgFile := flag.String("config", "", "path to a config file (JSON or TOML); see usage to see how it's configured (pass -v=1 flag for verbose help). Mutually exclusive with command line flags => when used, other flags are ignored")
	flag.Parse()

	var err error
	if *cfgFile != "" {
		// override config from config file
		cfg, err = readConfigFile(*cfgFile)
	}
	if err == nil {
		// add all feeds from cmdline
		cfg.addFeedsFromArgs(*provider, flag.Args()...)
		err = cfg.validate()
	}
	if err != nil {
		flog.Error(err)
		flag.Usage()
	}

	start := time.Now()

	if stats.AreLogged() {
		defer func(start time.Time) {
			stats.TrackTime("run.time", start, time.Second)
			stats.WriteAndLogError()
		}(start)
	}

	flog.V(1).Info("loading NVD feeds...")

	var overrides cvefeed.Dictionary
	dicts := map[string]cvefeed.Dictionary{} // provider -> dictionary
	for provider, files := range cfg.Feeds {
		dict, err := cvefeed.LoadJSONDictionary(files...)
		if err != nil {
			flog.Errorf("failed to load dictionary for provider %s: %v", provider, err)
		}
		dicts[provider] = dict
	}

	allEmpty := true
	for _, dict := range dicts {
		if len(dict) != 0 {
			allEmpty = false
			break
		}
	}
	if allEmpty {
		flog.Error(fmt.Errorf("all dictionaries are empty"))
		return -1
	}

	overrides, err = cvefeed.LoadJSONDictionary(cfg.FeedOverrides...)
	if err != nil {
		flog.Error(err)
		return -1
	}

	flog.V(1).Infof("...done in %v", time.Since(start))

	if len(overrides) != 0 {
		start = time.Now()
		flog.V(1).Info("applying overrides...")
		for _, dict := range dicts {
			dict.Override(overrides)
		}
		flog.V(1).Infof("...done in %v", time.Since(start))
	}

	caches := map[string]*cvefeed.Cache{}
	for provider, dict := range dicts {
		caches[provider] = cvefeed.NewCache(dict).SetRequireVersion(cfg.RequireVersion).SetMaxSize(cfg.CacheSize)
	}

	if cfg.IndexDict {
		start = time.Now()
		flog.V(1).Info("indexing dictionaries...")
		for provider, cache := range caches {
			cache.Idx = cvefeed.NewIndex(dicts[provider])
			if flog.V(2) {
				var named, total int
				for k, v := range cache.Idx {
					if k != wfn.Any {
						named += len(v)
					}
					total += len(v)
				}
				flog.Infof("%d out of %d records are named", named, total)
			}
		}
		flog.V(1).Infof("...done in %v", time.Since(start))
	}

	if cfg.CPUProfile != "" {
		f, err := os.Create(cfg.CPUProfile)
		if err != nil {
			flog.Error(err)
			return 1
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	done := processInput(os.Stdin, os.Stdout, caches, cfg)

	if cfg.MemoryProfile != "" {
		f, err := os.Create(cfg.MemoryProfile)
		if err != nil {
			flog.Error(err)
			return 1
		}
		runtime.GC()
		if err = pprof.WriteHeapProfile(f); err != nil {
			flog.Errorf("couldn't write heap profile: %v", err)
		}
		f.Close()
	}

	<-done
	return 0
}
