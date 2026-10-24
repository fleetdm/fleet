package nvdsync

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd/schema"
	"github.com/google/go-cmp/cmp"
	"github.com/pandatix/nvdapi/v2"
	"github.com/stretchr/testify/require"
)

var (
	legacyCVEFeedsDir = filepath.Join("testdata", "cve", "legacy_feeds")
	api20CVEDir       = filepath.Join("testdata", "cve", "api_2.0")
)

func TestStoreCVEsLegacyFormat(t *testing.T) {
	t.Parallel()
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
			convertedLegacyVuln := convertAPI20CVEToLegacy(t.Context(), api20Vuln.CVE, slog.New(slog.DiscardHandler))
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

	// We now fall back to secondary CVSS score sources when primary isn't available, which legacy didn't do
	if v1.Impact.BaseMetricV2 == nil {
		v2.Impact.BaseMetricV2 = nil
	}
	if v1.Impact.BaseMetricV3 == nil {
		v2.Impact.BaseMetricV3 = nil
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

func TestEnhanceNVDwithVulncheck(t *testing.T) {
	// gzip the vulncheck data
	testDataPath := filepath.Join("testdata", "cve", "vulncheck_test_data")
	nvdFile := filepath.Join(testDataPath, "nvdcve-1.1-2024.json")
	vulncheckFile1 := filepath.Join(testDataPath, "nvdcve-2.0-122.json")
	vulncheckFile2 := filepath.Join(testDataPath, "nvdcve-2.0-121.json")
	gzipFile1 := filepath.Join(testDataPath, "nvdcve-2.0-122.json.gz")
	gzipFile2 := filepath.Join(testDataPath, "nvdcve-2.0-121.json.gz")
	zFile := filepath.Join(testDataPath, "vulncheck.zip")

	// backup the original data to new directory
	backupPath := filepath.Join(testDataPath, "backup")
	err := os.MkdirAll(backupPath, os.ModePerm)
	require.NoError(t, err)

	err = copyFile(nvdFile, filepath.Join(backupPath, "nvdcve-1.1-2024.json"))
	require.NoError(t, err)

	err = copyFile(vulncheckFile1, filepath.Join(backupPath, "nvdcve-2.0-122.json"))
	require.NoError(t, err)

	err = copyFile(vulncheckFile2, filepath.Join(backupPath, "nvdcve-2.0-121.json"))
	require.NoError(t, err)

	// compress the vulncheck file to mimic the real data
	err = CompressFile(vulncheckFile1, gzipFile1)
	require.NoError(t, err)

	err = CompressFile(vulncheckFile2, gzipFile2)
	require.NoError(t, err)

	err = zipFiles([]string{gzipFile1, gzipFile2}, zFile)
	require.NoError(t, err)

	defer func() {
		// restore the original data
		err := copyFile(filepath.Join(backupPath, "nvdcve-1.1-2024.json"), filepath.Join(testDataPath, "nvdcve-1.1-2024.json"))
		require.NoError(t, err)

		err = copyFile(filepath.Join(backupPath, "nvdcve-2.0-122.json"), filepath.Join(testDataPath, "nvdcve-2.0-122.json"))
		require.NoError(t, err)

		err = copyFile(filepath.Join(backupPath, "nvdcve-2.0-121.json"), filepath.Join(testDataPath, "nvdcve-2.0-121.json"))
		require.NoError(t, err)

		err = os.RemoveAll(backupPath)
		require.NoError(t, err)

		err = os.Remove(gzipFile1)
		require.NoError(t, err)

		err = os.Remove(gzipFile2)
		require.NoError(t, err)

		err = os.Remove(zFile)
		require.NoError(t, err)
	}()

	syncer, err := NewCVE(testDataPath)
	require.NoError(t, err)

	err = syncer.processVulnCheckFile(t.Context(), "vulncheck.zip")
	require.NoError(t, err)

	// compare the enhanced data with the expected data
	enhancedDataPath := filepath.Join(testDataPath, "nvdcve-1.1-2024.json")
	expectedDataPath := filepath.Join(testDataPath, "nvdcve-1.1-2024-expected.json")
	enhancedData, err := os.ReadFile(enhancedDataPath)
	require.NoError(t, err)
	expectedData, err := os.ReadFile(expectedDataPath)
	require.NoError(t, err)

	require.Equal(t, string(expectedData), string(enhancedData))
}

func TestFetchVulnCheckDownloadURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"data": [{"url": "http://example.com/vulncheck.zip"}]}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	syncer, err := NewCVE("foo")
	require.NoError(t, err)

	if _, ok := os.LookupEnv("VULNCHECK_API_KEY"); !ok {
		os.Setenv("VULNCHECK_API_KEY", "foo")
	}

	url, err := syncer.fetchVulnCheckDownloadURL(context.Background(), server.URL)
	require.NoError(t, err)

	require.Equal(t, "http://example.com/vulncheck.zip", url)
}

func TestFetchVulnCheckDownloadURLWithRetries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	syncer, err := NewCVE("foo")
	require.NoError(t, err)

	syncer.MaxTryAttempts = 3
	syncer.WaitTimeForRetry = 5 * time.Millisecond

	if _, ok := os.LookupEnv("VULNCHECK_API_KEY"); !ok {
		os.Setenv("VULNCHECK_API_KEY", "foo")
	}

	_, err = syncer.fetchVulnCheckDownloadURL(context.Background(), server.URL)
	require.Error(t, err)
}

func copyFile(src, dst string) error {
	// Open the source file for reading
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create the destination file with write and read permissions
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy the contents of the source file to the destination file
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Ensure that the copied contents are flushed to stable storage
	return destFile.Sync()
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// stalledBody returns prefix, then blocks until ctx is done, like a response
// whose body stops arriving mid-stream.
type stalledBody struct {
	ctx    context.Context
	prefix []byte
}

func (b *stalledBody) Read(p []byte) (int, error) {
	if len(b.prefix) > 0 {
		n := copy(p, b.prefix)
		b.prefix = b.prefix[n:]
		return n, nil
	}
	<-b.ctx.Done()
	return 0, b.ctx.Err()
}

func (b *stalledBody) Close() error { return nil }

func TestSyncTimesOutStalledNVDRequest(t *testing.T) {
	// Runs on synctest's fake clock, so the real timeouts and retry waits elapse
	// instantly. Without a per-request deadline, the stalled body never returns
	// and synctest fails the test as a deadlock.
	synctest.Test(t, func(t *testing.T) {
		var requests int
		syncer, err := NewCVE(t.TempDir())
		require.NoError(t, err)
		syncer.client = fleethttp.NewClient(fleethttp.WithNoTimeout())
		syncer.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			requests++
			// Headers and part of the body arrive, then the body stalls: the
			// response-header timeout doesn't apply, so only a per-request deadline
			// can end the request.
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": {"application/json"}},
				Body:       &stalledBody{ctx: r.Context(), prefix: []byte(`{"resultsPerPage": 2000, "vulnerabilities": [`)},
				Request:    r,
			}, nil
		})

		start := time.Now()
		_, err = syncer.sync(t.Context(), nil)
		require.ErrorIs(t, err, context.DeadlineExceeded)

		// The first attempt plus maxRetryAttempts+1 retries, each ended by the
		// per-request deadline and followed by a retry wait.
		attempts := maxRetryAttempts + 2
		require.Equal(t, attempts, requests)
		require.Equal(t, time.Duration(attempts)*nvdRequestTimeout+time.Duration(attempts-1)*waitTimeForRetry, time.Since(start))
	})
}
