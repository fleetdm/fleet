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
	"bytes"
	"sync"
	"testing"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
)

func TestCacheEviction(t *testing.T) {
	items, err := LoadFeed(func(_ string) ([]Vuln, error) {
		return ParseJSON(bytes.NewBufferString(testJSONdict))
	}, "")
	if err != nil {
		t.Fatalf("failed to parse the dictionary: %v", err)
	}
	cache := NewCache(items).SetMaxSize(2 * 1024)
	matchingItem := &wfn.Attributes{Part: "a", Vendor: "microsoft", Product: "ie", Version: "5\\.4"}

	// first, run concurrently and enjoy different sizes of cache logged on each run
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(variant int) {
			defer wg.Done()
			inventory := []*wfn.Attributes{
				matchingItem,
			}
			for i := 0; i < variant; i++ {
				inventory = append(inventory, &wfn.Attributes{Vendor: "huh", Product: "brah"})
			}
			matches := cache.Get(inventory)
			if len(matches) != 1 {
				t.Errorf("variant %d: cache.Get() returned wrong amount of matches (%d, 1 was expected)", variant, len(matches))
				return
			}
			if len(matches[0].CPEs) != 1 {
				t.Errorf("variant %d: cache.Get() returned wrong a match with wrong number of CPEs (%d, 1 was expected)", variant, len(matches[0].CPEs))
			}
			if *matches[0].CPEs[0] != *matchingItem {
				t.Errorf("variant %d: cache.Get() returned wrong match:\n%+v\n%+v was expected", variant, *matches[0].CPEs[0], *matchingItem)
			}
		}(i)
	}
	wg.Wait()
	if cache.size > cache.MaxSize {
		t.Errorf("concurrent run: cache size exceeds maximum: %d bytes out of %d bytes", cache.size, cache.MaxSize)
	}
	t.Logf("concurrent run: cache size %d/%d; %d records cached", cache.size, cache.MaxSize, len(cache.data))

	// now let's get serious and get some deterministic resutls
	for i := 0; i < 50; i++ {
		variant := i
		inventory := []*wfn.Attributes{
			matchingItem,
		}
		for i := 0; i < variant; i++ {
			inventory = append(inventory, &wfn.Attributes{Vendor: "huh", Product: "brah"})
		}
		matches := cache.Get(inventory)
		if len(matches) != 1 {
			t.Fatalf("variant %d: cache.Get() returned wrong amount of matches (%d, 1 was expected)", variant, len(matches))
		}
		if len(matches[0].CPEs) != 1 {
			t.Errorf("variant %d: cache.Get() returned wrong a match with wrong number of CPEs (%d, 1 was expected)", variant, len(matches[0].CPEs))
		}
		if *matches[0].CPEs[0] != *matchingItem {
			t.Errorf("variant %d: cache.Get() returned wrong match:\n%+v\n%+v was expected", variant, *matches[0].CPEs[0], *matchingItem)
		}
	}
	if cache.size > cache.MaxSize {
		t.Errorf("sequential run #1: cache size exceeds maximum: %d bytes out of %d bytes", cache.size, cache.MaxSize)
	}
	// the latest cached items are almost 1K long, so there should be only 1 left in the cache
	if len(cache.data) > 1 {
		t.Errorf("sequential run #1: more than 1 record cached (%d)", len(cache.data))
	}
	t.Logf("sequential run #1: cache size %d/%d; %d records cached", cache.size, cache.MaxSize, len(cache.data))

	// and now let's go the other way around and make cache evict the bigger records first
	for i := 39; i >= 0; i-- {
		variant := i
		inventory := []*wfn.Attributes{
			matchingItem,
		}
		for i := 0; i < variant; i++ {
			inventory = append(inventory, &wfn.Attributes{Vendor: "huh", Product: "brah"})
		}
		matches := cache.Get(inventory)
		if len(matches) != 1 {
			t.Errorf("variant %d: cache.Get() returned wrong amount of matches (%d, 1 was expected)", variant, len(matches))
		}
		if len(matches[0].CPEs) != 1 {
			t.Errorf("variant %d: cache.Get() returned wrong a match with wrong number of CPEs (%d, 1 was expected)", variant, len(matches[0].CPEs))
		}
		if *matches[0].CPEs[0] != *matchingItem {
			t.Errorf("variant %d: cache.Get() returned wrong match:\n%+v\n%+v was expected", variant, *matches[0].CPEs[0], *matchingItem)
		}
	}
	if cache.size > cache.MaxSize {
		t.Errorf("sequential run #2: cache size exceeds maximum: %d bytes out of %d bytes", cache.size, cache.MaxSize)
	}
	// Since we touch the smaller records first, we should have more of these cached
	if len(cache.data) < 5 {
		t.Errorf("sequential run #2: more than 1 record cached (%d)", len(cache.data))
	}
	t.Logf("sequential run #2: cache size %d/%d; %d records cached", cache.size, cache.MaxSize, len(cache.data))
}
