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
	"encoding/json"
	"math/bits"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd"
	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
)

type bag map[string]interface{}

// ChunkKind is the type of chunks produced by a diff.
type ChunkKind string

const (
	// ChunkDescription indicates a difference in the description of a
	// vulnerability.
	ChunkDescription ChunkKind = "description"
	// ChunkScore indicates a difference in the score of a vulnerability.
	ChunkScore = "score"
)

type chunk uint32

const (
	chunkDescriptionShift = iota
	chunkScoreShift
	chunkMaxShift
)

const (
	chunkDescription chunk = 1 << iota
	chunkScore
)

var chunkKind = [chunkMaxShift]ChunkKind{
	ChunkDescription,
	ChunkScore,
}

func (kind ChunkKind) shift() int {
	for i, v := range chunkKind {
		if v == kind {
			return i
		}
	}
	return chunkMaxShift
}

type diffEntry struct {
	id   string
	bits chunk
}

type diffFeed struct {
	name string
	dict Dictionary
}

type diff struct {
	a, b diffFeed
}

func newDiff(a, b diffFeed) *diff {
	return &diff{
		a: a,
		b: b,
	}
}

// DiffStats is the result of a diff.
type DiffStats struct {
	diff *diff // back pointer to the diff these stats are for

	numVulnsA, numVulnsB int

	aNotB     []string // ids of vulns that are in a but not in b
	bNotA     []string // ids of vulns that are in b but not in a
	entries   []diffEntry
	bitCounts [chunkMaxShift]int
}

// NumVulnsA returns the vulnerability in A (the first input to Diff).
func (s *DiffStats) NumVulnsA() int {
	return s.numVulnsA
}

// NumVulnsB returns the vulnerability in A (the first input to Diff).
func (s *DiffStats) NumVulnsB() int {
	return s.numVulnsB
}

// VulnsANotB returns the vulnerabilities that are A (the first input to Diff) but
// are not in B (the second input to Diff).
func (s *DiffStats) VulnsANotB() []string {
	return s.aNotB
}

// NumVulnsANotB returns the numbers of vulnerabilities that are A (the first input
// to Diff) but are not in B (the second input to Diff).
func (s *DiffStats) NumVulnsANotB() int {
	return len(s.aNotB)
}

// VulnsBNotA returns the vulnerabilities that are A (the first input to Diff) but
// are not in B (the second input to Diff).
func (s *DiffStats) VulnsBNotA() []string {
	return s.bNotA
}

// NumVulnsBNotA returns the numbers of vulnerabilities that are B (the second input
// to Diff) but are not in A (the first input to Diff).
func (s *DiffStats) NumVulnsBNotA() int {
	return len(s.bNotA)
}

// NumDiffVulns returns the number of vulnerability that are in both A and B but
// are different (eg. different description, score, ...).
func (s *DiffStats) NumDiffVulns() int {
	return len(s.entries)
}

// NumChunk returns the number of different vulnerabilities that have a specific chunk.
func (s *DiffStats) NumChunk(chunk ChunkKind) int {
	return s.bitCounts[chunk.shift()]
}

// PercentChunk returns the percentage of different vulnerabilities that have a specific chunk.
func (s *DiffStats) PercentChunk(chunk ChunkKind) float64 {
	return float64(s.bitCounts[chunk.shift()]) / float64(len(s.entries)) * 100
}

func diffDetails(s *schema.NVDCVEFeedJSON10DefCVEItem, bit chunk) bag {
	var v interface{}

	switch bit {
	case chunkDescription:
		v = bag{
			"description": englishDescription(s),
		}
	case chunkScore:
		v = s.Impact
	}

	var data bag
	tmp, _ := json.Marshal(v)
	_ = json.Unmarshal(tmp, &data)

	return data
}

func genEntryDiffOutput(aFeed, bFeed *diffFeed, entry *diffEntry) []bag {
	a := aFeed.dict[entry.id].(*nvd.Vuln).Schema()
	b := bFeed.dict[entry.id].(*nvd.Vuln).Schema()
	outputs := make([]bag, bits.OnesCount32(uint32(entry.bits)))
	for i := 0; i < chunkMaxShift; i++ {
		if entry.bits&(1<<i) != 0 {
			outputs[i] = bag{
				"kind":     chunkKind[i],
				aFeed.name: diffDetails(a, 1<<i),
				bFeed.name: diffDetails(b, 1<<i),
			}
		}
	}
	return outputs
}

// MarshalJSON implements a custom JSON marshaller.
func (s *DiffStats) MarshalJSON() ([]byte, error) {
	var differences []bag
	for _, entry := range s.entries {
		differences = append(differences, bag{
			"id":     entry.id,
			"chunks": genEntryDiffOutput(&s.diff.a, &s.diff.b, &entry),
		})
	}
	return json.Marshal(bag{
		"differences": differences,
	})
}

func englishDescription(s *schema.NVDCVEFeedJSON10DefCVEItem) string {
	for _, d := range s.CVE.Description.DescriptionData {
		if d.Lang == "en" {
			return d.Value
		}
	}
	return ""
}

func sameDescription(a, b *schema.NVDCVEFeedJSON10DefCVEItem) bool {
	return englishDescription(a) == englishDescription(b)
}

func sameScoreCVSSV2(a, b *schema.NVDCVEFeedJSON10DefCVEItem) bool {
	var aScore, bScore float64
	var aVector, bVector string

	if a.Impact.BaseMetricV2 != nil && a.Impact.BaseMetricV2.CVSSV2 != nil {
		aScore = a.Impact.BaseMetricV2.CVSSV2.BaseScore
		aVector = a.Impact.BaseMetricV2.CVSSV2.VectorString
	}
	if b.Impact.BaseMetricV2 != nil && b.Impact.BaseMetricV2.CVSSV2 != nil {
		bScore = b.Impact.BaseMetricV2.CVSSV2.BaseScore
		bVector = b.Impact.BaseMetricV2.CVSSV2.VectorString
	}
	return aScore == bScore && aVector == bVector
}

func sameScoreCVSSV3(a, b *schema.NVDCVEFeedJSON10DefCVEItem) bool {
	var aScore, bScore float64
	var aVector, bVector string

	if a.Impact.BaseMetricV3 != nil && a.Impact.BaseMetricV3.CVSSV3 != nil {
		aScore = a.Impact.BaseMetricV3.CVSSV3.BaseScore
		aVector = a.Impact.BaseMetricV3.CVSSV3.VectorString
	}
	if b.Impact.BaseMetricV3 != nil && b.Impact.BaseMetricV3.CVSSV3 != nil {
		bScore = b.Impact.BaseMetricV3.CVSSV3.BaseScore
		bVector = b.Impact.BaseMetricV3.CVSSV3.VectorString
	}
	return aScore == bScore && aVector == bVector
}

func sameScore(a, b *schema.NVDCVEFeedJSON10DefCVEItem) bool {
	return sameScoreCVSSV2(a, b) && sameScoreCVSSV3(a, b)
}

func (d *diff) stats() *DiffStats {
	stats := DiffStats{
		diff:      d,
		numVulnsA: len(d.a.dict),
		numVulnsB: len(d.b.dict),
	}

	// List of vulns that are in a but not in b.
	for key := range d.a.dict {
		if _, ok := d.b.dict[key]; !ok {
			stats.aNotB = append(stats.aNotB, key)
			continue
		}

		// key is in both a and b, let's compare further!
		a := d.a.dict[key].(*nvd.Vuln).Schema()
		b := d.b.dict[key].(*nvd.Vuln).Schema()

		var entry diffEntry

		if !sameDescription(a, b) {
			entry.bits |= chunkDescription
			stats.bitCounts[chunkDescriptionShift]++
		}
		if !sameScore(a, b) {
			entry.bits |= chunkScore
			stats.bitCounts[chunkScoreShift]++
		}

		if entry.bits != 0 {
			entry.id = key
			stats.entries = append(stats.entries, entry)
		}
	}

	// List of vulns that are in b but not in a.
	for key := range d.b.dict {
		if _, ok := d.a.dict[key]; !ok {
			stats.bNotA = append(stats.bNotA, key)
		}
	}

	return &stats
}

// Diff performs a diff between two Dictionaries.
func Diff(aName string, aDict Dictionary, bName string, bDict Dictionary) *DiffStats {
	diff := newDiff(diffFeed{aName, aDict}, diffFeed{bName, bDict})
	return diff.stats()
}
