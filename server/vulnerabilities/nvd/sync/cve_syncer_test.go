package nvdsync

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/go-kit/log"
	"github.com/google/go-cmp/cmp"
	"github.com/pandatix/nvdapi/v2"
	"github.com/stretchr/testify/require"
)

var (
	legacyCVEFeedsDir = filepath.Join("testdata", "cve", "legacy_feeds")
	api20CVEDir       = filepath.Join("testdata", "cve", "api_2.0")
)

func TestStoreCVEsLegacyFormat(t *testing.T) {
	year := 2023
	t.Run(fmt.Sprintf("%d", year), func(t *testing.T) {
		// Load CVEs from legacy feed.
		legacyCVEFilePath := filepath.Join(legacyCVEFeedsDir, fmt.Sprintf("%d.json.gz", year))
		var legacyCVEs schema.NVDCVEFeedJSON10
		loadJSONGz(t, legacyCVEFilePath, &legacyCVEs)

		// Load CVEs from new API 2.0 feed.
		api20CVEFilePath := filepath.Join(api20CVEDir, fmt.Sprintf("%d.json.gz", year))
		var api20CVEs []nvdapi.CVEItem
		loadJSONGz(t, api20CVEFilePath, &api20CVEs)

		// Setup map of legacy CVEs.
		legacyVulns := make(map[string]schema.NVDCVEFeedJSON10DefCVEItem) // key is the CVE ID.
		for _, legacyVuln := range legacyCVEs.CVEItems {
			legacyVulns[legacyVuln.CVE.CVEDataMeta.ID] = *legacyVuln
		}

		// Convert CVEs fetched using the new API 2.0 to the old legacy feeds format
		// and compare them with the corresponding fetched legacy CVE.
		var (
			vulnsNotFoundInLegacy []string
			mismatched            []string
			matched               = 0
		)
		for _, api20Vuln := range api20CVEs {
			convertedLegacyVuln := convertAPI20CVEToLegacy(api20Vuln, log.NewNopLogger())
			legacyVuln, ok := legacyVulns[*api20Vuln.CVE.ID]
			if !ok {
				vulnsNotFoundInLegacy = append(vulnsNotFoundInLegacy, *api20Vuln.CVE.ID)
				continue
			}

			if compareVulnerabilities(legacyVuln, *convertedLegacyVuln) {
				matched++
			} else {
				mismatched = append(mismatched, *api20Vuln.CVE.ID)
			}
		}
		matchRate := float64(matched) / float64(len(api20CVEs))
		require.Greater(t, matchRate, .99)
		t.Logf("%d: CVEs count: %d, match count: %d, match rate: %f", year, len(api20CVEs), matched, matchRate)
		// TODO(lucas): Review these CVEs to check they are a-ok to be skipped.
		t.Logf("%d: Vulnerabilities not found in legacy store: %s", year, strings.Join(vulnsNotFoundInLegacy, ", "))
		t.Logf("%d: Vulnerabilities that mismatch from legacy store: %s", year, strings.Join(mismatched, ", "))
	})
}

func compareVulnerabilities(v1 schema.NVDCVEFeedJSON10DefCVEItem, v2 schema.NVDCVEFeedJSON10DefCVEItem) bool {
	clearDifferingFields := func(v *schema.NVDCVEFeedJSON10DefCVEItem) {
		sort.Slice(v.CVE.References.ReferenceData, func(i, j int) bool {
			return v.CVE.References.ReferenceData[i].URL < v.CVE.References.ReferenceData[j].URL
		})
		sortChildren(v.Configurations.Nodes)
		for _, referenceData := range v.CVE.References.ReferenceData {
			referenceData.Refsource = ""
			referenceData.Name = referenceData.URL
		}
		// These fields mostly match, but sometimes differ.
		v.CVE.CVEDataMeta.ASSIGNER = ""
		v.CVE.Problemtype = nil
	}

	clearDifferingFields(&v1)
	clearDifferingFields(&v2)
	return cmp.Equal(v1, v2)
}

func loadJSONGz(t *testing.T, path string, v any) {
	legacyCVEJSONGz, err := os.ReadFile(path)
	require.NoError(t, err)
	legacyCVEGzipReader, err := gzip.NewReader(bytes.NewReader(legacyCVEJSONGz))
	require.NoError(t, err)
	err = json.NewDecoder(legacyCVEGzipReader).Decode(v)
	require.NoError(t, err)
	require.NoError(t, legacyCVEGzipReader.Close())
}

func cpeMatchHash(v schema.NVDCVEFeedJSON10DefCPEMatch) string {
	s := v.Cpe23Uri +
		v.VersionEndExcluding +
		v.VersionEndIncluding +
		v.VersionStartExcluding +
		v.VersionStartIncluding +
		strconv.FormatBool(v.Vulnerable) +
		v.Cpe22Uri
	h := sha256.Sum256([]byte(s))
	return string(h[:])
}

func childrenHash(v schema.NVDCVEFeedJSON10DefNode) string {
	var s string
	for _, cpeMatch := range v.CPEMatch {
		s += cpeMatchHash(*cpeMatch)
	}
	for _, child := range v.Children {
		s += childrenHash(*child)
	}
	s += v.Operator + strconv.FormatBool(v.Negate)
	h := sha256.Sum256([]byte(s))
	return string(h[:])
}

func sortChildren(children []*schema.NVDCVEFeedJSON10DefNode) {
	for _, child := range children {
		sort.Slice(child.CPEMatch, func(i, j int) bool {
			return cpeMatchHash(*child.CPEMatch[i]) < cpeMatchHash(*child.CPEMatch[j])
		})
		sortChildren(child.Children)
	}
	sort.Slice(children, func(i, j int) bool {
		return childrenHash(*children[i]) < childrenHash(*children[j])
	})
}
