package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyOne(t *testing.T) {
	cases := []struct {
		name     string
		expected string
		actual   string
		want     resultKind
	}{
		{"exact match", "cpe:2.3:a:vendor:product:1.0:*:*:*:*:*:*:*", "cpe:2.3:a:vendor:product:1.0:*:*:*:*:*:*:*", resultPass},
		{"both empty is true negative", "", "", resultPassTrueNegative},
		{"expected empty but got value is false positive", "", "cpe:2.3:a:v:p:1:*:*:*:*:*:*:*", resultFailFalsePositive},
		{"expected value but got empty is missing", "cpe:2.3:a:v:p:1:*:*:*:*:*:*:*", "", resultFailMissing},
		{"expected and actual differ is mismatch", "cpe:2.3:a:v1:p:1:*:*:*:*:*:*:*", "cpe:2.3:a:v2:p:1:*:*:*:*:*:*:*", resultFailMismatch},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, classifyOne(tc.expected, tc.actual))
		})
	}
}

func TestResultKindIsPass(t *testing.T) {
	assert.True(t, resultPass.IsPass())
	assert.True(t, resultPassTrueNegative.IsPass())
	assert.False(t, resultFailMismatch.IsPass())
	assert.False(t, resultFailMissing.IsPass())
	assert.False(t, resultFailFalsePositive.IsPass())
}

func TestSummarize(t *testing.T) {
	results := []caseResult{
		{Case: accuracyCase{Software: softwareInput{Source: "apps"}}, Kind: resultPass},
		{Case: accuracyCase{Software: softwareInput{Source: "apps"}}, Kind: resultPass},
		{Case: accuracyCase{Software: softwareInput{Source: "apps"}}, Kind: resultFailMismatch},
		{Case: accuracyCase{Software: softwareInput{Source: "homebrew_packages"}}, Kind: resultPassTrueNegative},
		{Case: accuracyCase{Software: softwareInput{Source: "homebrew_packages"}}, Kind: resultFailMissing},
		{Case: accuracyCase{Software: softwareInput{Source: "chrome_extensions"}}, Kind: resultFailFalsePositive},
	}

	s := summarize(results)

	assert.Equal(t, 2, s.Pass)
	assert.Equal(t, 1, s.PassTrueNegative)
	assert.Equal(t, 1, s.FailMismatch)
	assert.Equal(t, 1, s.FailMissing)
	assert.Equal(t, 1, s.FailFalsePositive)

	// Per-source breakdown (non-nil require before field deref satisfies nilaway).
	apps := s.BySource["apps"]
	require.NotNil(t, apps)
	assert.Equal(t, 3, apps.Total)
	assert.Equal(t, 2, apps.Passed)

	brew := s.BySource["homebrew_packages"]
	require.NotNil(t, brew)
	assert.Equal(t, 2, brew.Total)
	assert.Equal(t, 1, brew.Passed)

	ext := s.BySource["chrome_extensions"]
	require.NotNil(t, ext)
	assert.Equal(t, 1, ext.Total)
	assert.Equal(t, 0, ext.Passed)
}

func TestBuildSoftwareAssignsUniqueIDsAndSentinel(t *testing.T) {
	suites := map[string]*accuracySuite{
		"cves_one.json": {
			TestCases: []accuracyCase{
				{ID: "a", Software: softwareInput{Name: "A.app", Source: "apps"}},
				{ID: "b", Software: softwareInput{Name: "B.app", Source: "apps"}},
			},
		},
		"cves_two.json": {
			TestCases: []accuracyCase{
				{ID: "c", Software: softwareInput{Name: "C", Source: "homebrew_packages"}},
			},
		},
	}

	software, index := buildSoftware(suites)

	assert.Len(t, software, 3)
	assert.Len(t, index, 3)

	// All software should carry the sentinel so empty generated CPEs still produce a
	// delete event (rather than being skipped by TranslateSoftwareToCPE).
	for _, sw := range software {
		assert.Equal(t, sentinelCPE, sw.GenerateCPE)
	}

	// IDs must be unique and non-zero.
	ids := make(map[uint]struct{})
	for _, sw := range software {
		assert.NotZero(t, sw.ID)
		_, dup := ids[sw.ID]
		assert.False(t, dup, "duplicate ID %d", sw.ID)
		ids[sw.ID] = struct{}{}
	}
}

func TestBuildSoftwareIsDeterministic(t *testing.T) {
	suites := map[string]*accuracySuite{
		"cves_b.json": {TestCases: []accuracyCase{{ID: "b1", Software: softwareInput{Name: "B1"}}}},
		"cves_a.json": {TestCases: []accuracyCase{{ID: "a1", Software: softwareInput{Name: "A1"}}}},
	}

	sw1, _ := buildSoftware(suites)
	sw2, _ := buildSoftware(suites)
	sw3, _ := buildSoftware(suites)

	// Stable ordering across invocations: suites are sorted by filename before iteration.
	require.Len(t, sw1, 2)
	assert.Equal(t, sw1, sw2)
	assert.Equal(t, sw1, sw3)
	assert.Equal(t, "A1", sw1[0].Name, "files should be sorted alphabetically before being emitted")
	assert.Equal(t, "B1", sw1[1].Name)
}

