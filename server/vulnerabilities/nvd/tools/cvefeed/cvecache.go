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

package cvefeed

import (
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/facebookincubator/flog"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
)

const cacheEvictPercentage = 0.1 // every eviction cycle invalidates this part of cache size at once

// Index maps the CPEs to the entries in the NVD feed they mentioned in
type Index map[string][]Vuln

// NewIndex creates new Index from a slice of CVE entries
func NewIndex(d Dictionary) Index {
	idx := Index{}
	for _, entry := range d {
		set := map[string]bool{}
		for _, cpe := range entry.Config() {
			// Can happen, for instance, when the feed contains illegal binding of CPE name. Unfortunately, it happens to NVD,
			// e.g. embedded ? in cpe:2.3:a:disney:where\\'s_my_perry?_free:1.5.1:*:*:*:*:android:*:* of CVE-2014-5606
			if cpe == nil {
				continue
			}
			product := cpe.Product
			if wfn.HasWildcard(product) {
				product = wfn.Any
			}
			if !set[product] {
				set[product] = true
				idx[product] = append(idx[product], entry)
			}
		}
	}
	return idx
}

// MatchResult stores CVE and a slice of CPEs that matched it
type MatchResult struct {
	CVE  Vuln
	CPEs []*wfn.Attributes
}

// cachedCVEs stores cached CVEs, a channel to signal if the value is ready
type cachedCVEs struct {
	res           []MatchResult
	ready         chan struct{}
	size          int64
	evictionIndex int // position in eviction queue
}

// updateResSize calculates the size of cached MatchResult and assigns it to cves.size
func (cves *cachedCVEs) updateResSize(key string) {
	if cves == nil {
		return
	}
	cves.size = int64(int(unsafe.Sizeof(key)) + len(key))
	cves.size += int64(unsafe.Sizeof(cves.res))
	for i := range cves.res {
		cves.size += int64(unsafe.Sizeof(cves.res[i].CVE))
		for _, attr := range cves.res[i].CPEs {
			cves.size += int64(len(attr.Part)) + int64(unsafe.Sizeof(attr.Part))
			cves.size += int64(len(attr.Vendor)) + int64(unsafe.Sizeof(attr.Vendor))
			cves.size += int64(len(attr.Product)) + int64(unsafe.Sizeof(attr.Product))
			cves.size += int64(len(attr.Version)) + int64(unsafe.Sizeof(attr.Version))
			cves.size += int64(len(attr.Update)) + int64(unsafe.Sizeof(attr.Update))
			cves.size += int64(len(attr.Edition)) + int64(unsafe.Sizeof(attr.Edition))
			cves.size += int64(len(attr.SWEdition)) + int64(unsafe.Sizeof(attr.SWEdition))
			cves.size += int64(len(attr.TargetHW)) + int64(unsafe.Sizeof(attr.TargetHW))
			cves.size += int64(len(attr.Other)) + int64(unsafe.Sizeof(attr.Other))
			cves.size += int64(len(attr.Language)) + int64(unsafe.Sizeof(attr.Language))
		}
	}
}

// Cache caches CVEs for known CPEs
type Cache struct {
	// Used to compute the hit ratio
	numLookups int64
	numHits    int64

	// Actual cache data
	data           map[string]*cachedCVEs
	evictionQ      *evictionQueue
	mu             sync.Mutex
	Dict           Dictionary
	Idx            Index
	MaxSize        int64 // maximum size of the cache, 0 -- unlimited, -1 -- no caching
	size           int64 // current size of the cache
	RequireVersion bool  // ignore matching specifications that have Version == ANY
}

// NewCache creates new Cache instance with dictionary dict.
func NewCache(dict Dictionary) *Cache {
	return &Cache{Dict: dict, evictionQ: new(evictionQueue)}
}

// SetRequireVersion sets if the instance of cache fails matching the dictionary
// records without Version attribute of CPE name.
// Returns a pointer to the instance of Cache, for easy chaining.
func (c *Cache) SetRequireVersion(requireVersion bool) *Cache {
	c.RequireVersion = requireVersion
	return c
}

// SetMaxSize sets maximum size of the cache to some pre-defined value,
// size of 0 disables eviction (makes the cache grow indefinitely),
// negative size disables caching.
// Returns a pointer to the instance of Cache, for easy chaining.
func (c *Cache) SetMaxSize(size int64) *Cache {
	c.MaxSize = size
	return c
}

// Get returns slice of CVEs for CPE names from cpes parameter;
// if CVEs aren't cached (and the feature is enabled) it finds them in cveDict and caches the results
func (c *Cache) Get(cpes []*wfn.Attributes) []MatchResult {
	atomic.AddInt64(&c.numLookups, 1)

	// negative max size of the cache disables caching
	if c.MaxSize < 0 {
		return c.match(cpes)
	}

	// otherwise, let's get to the business
	key := cacheKey(cpes)
	c.mu.Lock()
	if c.data == nil {
		c.data = make(map[string]*cachedCVEs)
	}
	cves := c.data[key]
	if cves != nil {
		atomic.AddInt64(&c.numHits, 1)

		// value is being computed, wait till ready
		c.mu.Unlock()
		<-cves.ready
		c.mu.Lock() // TODO: XXX: ugly, consider using atomic.Value instead
		cves.evictionIndex = c.evictionQ.touch(cves.evictionIndex)
		c.mu.Unlock()
		return cves.res
	}
	// first request; the goroutine that sent it computes the value
	cves = &cachedCVEs{ready: make(chan struct{})}
	c.data[key] = cves
	c.mu.Unlock()
	// now other requests for same key wait on the channel, and the requests for the different keys aren't blocked
	cves.res = c.match(cpes)
	cves.updateResSize(key)
	c.mu.Lock()
	if c.MaxSize != 0 && c.size+cves.size > c.MaxSize {
		c.evict(int64(cacheEvictPercentage*float64(c.MaxSize)) + cves.size)
	}
	c.size += cves.size
	cves.evictionIndex = c.evictionQ.push(key)
	c.mu.Unlock()
	close(cves.ready)
	return cves.res
}

// match will return all match results based on the given cpes
func (c *Cache) match(cpes []*wfn.Attributes) []MatchResult {
	d := c.Dict
	if c.Idx != nil {
		d = c.dictFromIndex(cpes)
	}
	return c.matchDict(cpes, d)
}

// dictFromIndex creates CVE dictionary from entries indexed by CPE names
func (c *Cache) dictFromIndex(cpes []*wfn.Attributes) Dictionary {
	d := Dictionary{}
	if c.Idx == nil {
		return d
	}

	knownEntries := map[Vuln]bool{}
	addVulns := func(product string) {
		for _, vuln := range c.Idx[product] {
			if !knownEntries[vuln] {
				knownEntries[vuln] = true
				d[vuln.ID()] = vuln
			}
		}
	}

	for _, cpe := range cpes {
		if cpe == nil { // should never happen
			flog.Warning("nil CPE in list")
			continue
		}
		// any of the CPEs having product=ANY would mean we need to match against the entire dictionary
		if cpe.Product == wfn.Any {
			return c.Dict
		}
		addVulns(cpe.Product)
	}
	addVulns(wfn.Any)

	return d
}

// match matches the CPE names against internal vulnerability dictionary and returns a slice of matching resutls
func (c *Cache) matchDict(cpes []*wfn.Attributes, dict Dictionary) (results []MatchResult) {
	for _, v := range dict {
		if matches := v.Match(cpes, c.RequireVersion); len(matches) > 0 {
			results = append(results, MatchResult{v, matches})
		}
	}
	return results
}

// evict the least recently used records untile nbytes of capacity is achieved or no more records left.
// It is not concurrency-safe, c.mu should be locked before calling it.
func (c *Cache) evict(nbytes int64) {
	for c.size > 0 && c.size+nbytes > c.MaxSize {
		key := c.evictionQ.pop()
		cd, ok := c.data[key]
		if !ok { // should not happen
			panic("attempted to evict non-existent record")
		}
		c.size -= cd.size
		delete(c.data, key)
	}
}

func cacheKey(cpes []*wfn.Attributes) string {
	parts := make([]string, 0, len(cpes))
	for _, cpe := range cpes {
		if cpe == nil {
			continue
		}
		var out strings.Builder
		out.WriteString(cpe.Part)
		out.WriteByte('^')
		out.WriteString(cpe.Vendor)
		out.WriteByte('^')
		out.WriteString(cpe.Product)
		out.WriteByte('^')
		out.WriteString(cpe.Version)
		out.WriteByte('^')
		out.WriteString(cpe.Update)
		out.WriteByte('^')
		out.WriteString(cpe.Edition)
		out.WriteByte('^')
		out.WriteString(cpe.SWEdition)
		out.WriteByte('^')
		out.WriteString(cpe.TargetSW)
		out.WriteByte('^')
		out.WriteString(cpe.TargetHW)
		out.WriteByte('^')
		out.WriteString(cpe.Other)
		out.WriteByte('^')
		out.WriteString(cpe.Language)
		parts = append(parts, out.String())
	}
	sort.Strings(parts)
	return strings.Join(parts, "#")
}

// HitRatio returns the cache hit ratio, the number of cache hits to the number
// of lookups, as a percentage.
func (c *Cache) HitRatio() float64 {
	if c.numLookups == 0 {
		return 0
	}
	return float64(c.numHits) / float64(c.numLookups) * 100
}
