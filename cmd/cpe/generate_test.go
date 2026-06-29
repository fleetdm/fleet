package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cpedict"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

type mockClient struct {
	response *http.Response
}

func (m *mockClient) Do(*http.Request) (*http.Response, error) {
	return m.response, nil
}

// scriptedClient drives each request through a caller-supplied function, so tests can
// simulate resets, resumes, and status codes across successive calls.
type scriptedClient struct {
	do func(*http.Request) (*http.Response, error)
}

func (c *scriptedClient) Do(req *http.Request) (*http.Response, error) { return c.do(req) }

// partialBody yields its data and then returns err, simulating a body that is cut off
// mid-stream (err = io.ErrUnexpectedEOF) or completes cleanly (err = io.EOF).
type partialBody struct {
	data []byte
	err  error
}

func (b *partialBody) Read(p []byte) (int, error) {
	if len(b.data) == 0 {
		return 0, b.err
	}
	n := copy(p, b.data)
	b.data = b.data[n:]
	return n, nil
}

func (b *partialBody) Close() error { return nil }

// tarGzFeed builds a gzipped tar archive that mimics the NVD CPE Dictionary feed
// (nvdcpe-2.0.tar.gz), with one entry per provided file. The key/value of files is
// the entry name/contents.
func tarGzFeed(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}))
		_, err := tw.Write(content)
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

// feedClient returns a mock HTTP client whose response body is the given gzipped
// tar feed.
func feedClient(archive []byte) *mockClient {
	recorder := httptest.NewRecorder()
	recorder.Header().Add("Content-Type", "application/gzip")
	_, _ = recorder.Write(archive)
	return &mockClient{response: recorder.Result()}
}

// jsonClient returns a mock HTTP client whose body is the given raw JSON, mimicking a
// single NVD API CPE page.
func jsonClient(jsonData []byte) *mockClient {
	recorder := httptest.NewRecorder()
	recorder.Header().Add("Content-Type", "application/json")
	_, _ = recorder.Write(jsonData)
	return &mockClient{response: recorder.Result()}
}

// readCPERows returns every row of the cpe_2 table as string slices.
func readCPERows(t *testing.T, dbPath string) [][]string {
	t.Helper()
	db, err := sqlx.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()
	rows, err := db.Query("SELECT * FROM cpe_2")
	require.NoError(t, err)
	require.NoError(t, rows.Err())
	defer rows.Close()

	var result [][]string
	cols, _ := rows.Columns()
	for rows.Next() {
		pointers := make([]any, len(cols))
		container := make([]string, len(cols))
		for i := range pointers {
			pointers[i] = &container[i]
		}
		require.NoError(t, rows.Scan(pointers...))
		result = append(result, container)
	}
	return result
}

func TestCPEDB(t *testing.T) {
	// Each source delivers the same NVD 2.0 payload over a different transport (the feed
	// wraps it in a gzipped tar chunk; the API serves it as a raw JSON page). Both must
	// decode to identical CPE items, so the same golden file applies to both.
	sources := []struct {
		name  string
		fetch func(t *testing.T, jsonData []byte) ([]cpedict.CPEItem, error)
	}{
		{
			name: cpeSourceFeed,
			fetch: func(t *testing.T, jsonData []byte) ([]cpedict.CPEItem, error) {
				return fetchCPEFeed(feedClient(tarGzFeed(t, map[string][]byte{
					"nvdcpe-2.0-chunks/nvdcpe-2.0-chunk-00001.json": jsonData,
				})))
			},
		},
		{
			name: cpeSourceAPI,
			fetch: func(t *testing.T, jsonData []byte) ([]cpedict.CPEItem, error) {
				return fetchCPEsFromAPI(jsonClient(jsonData), "API_KEY")
			},
		},
	}

	// Find the paths of all input files in the testdata directory.
	paths, err := filepath.Glob(filepath.Join("testdata", "*.json"))
	require.NoError(t, err)

	for _, source := range sources {
		for _, p := range paths {
			source, path := source, p
			_, filename := filepath.Split(path)
			testName := filename[:len(filename)-len(filepath.Ext(path))]

			// e.g. "feed/test1", "api/test1" — both compared against test1.golden.
			t.Run(source.name+"/"+testName, func(t *testing.T) {
				t.Parallel()
				jsonData, err := os.ReadFile(path)
				require.NoError(t, err)

				cpes, err := source.fetch(t, jsonData)
				require.NoError(t, err)

				dbPath := filepath.Join(t.TempDir(), "cpe.sqlite")
				require.NoError(t, nvd.GenerateCPEDB(dbPath, cpes))

				result := readCPERows(t, dbPath)
				golden, err := os.ReadFile(filepath.Join("testdata", testName+".golden"))
				require.NoError(t, err)
				require.Equal(t, string(golden), fmt.Sprintf("%s", result))
			})
		}
	}
}

// TestCPEDBMultiChunk verifies that products split across multiple chunk files are
// all accumulated. Each chunk reports the full totalResults but only carries a slice
// of the products, exactly like the real nvdcpe-2.0.tar.gz feed.
func TestCPEDBMultiChunk(t *testing.T) {
	const totalResults = 3
	chunk := func(total int, cpeNames ...string) []byte {
		products := make([]string, 0, len(cpeNames))
		for _, name := range cpeNames {
			products = append(products, fmt.Sprintf(
				`{"cpe":{"deprecated":false,"cpeName":%q,"cpeNameId":%q,"lastModified":"2024-01-01T00:00:00.000","created":"2024-01-01T00:00:00.000","titles":[{"title":"title for %s","lang":"en"}]}}`,
				name, name, name,
			))
		}
		return fmt.Appendf(nil,
			`{"resultsPerPage":%d,"startIndex":0,"totalResults":%d,"format":"NVD_CPE","version":"2.0","timestamp":"2024-01-01T00:00:00.000","products":[%s]}`,
			len(cpeNames), total, strings.Join(products, ","),
		)
	}

	client := feedClient(tarGzFeed(t, map[string][]byte{
		"nvdcpe-2.0-chunks/nvdcpe-2.0-chunk-00001.json": chunk(totalResults,
			"cpe:2.3:a:vendor:product_a:1.0:*:*:*:*:*:*:*",
			"cpe:2.3:a:vendor:product_b:2.0:*:*:*:*:*:*:*",
		),
		"nvdcpe-2.0-chunks/nvdcpe-2.0-chunk-00002.json": chunk(totalResults,
			"cpe:2.3:a:vendor:product_c:3.0:*:*:*:*:*:*:*",
		),
	}))

	dir := t.TempDir()
	dbPath := getCPEs(client, dir)

	result := readCPERows(t, dbPath)
	require.Len(t, result, totalResults, "all products across both chunks should be present")
}

// TestFetchCPEFeedIncomplete verifies that a feed whose decoded product count does
// not match the reported totalResults is rejected (so the caller retries), which is
// how a truncated/dropped chunk download surfaces.
func TestFetchCPEFeedIncomplete(t *testing.T) {
	// totalResults says 5 but only one product is present.
	body := []byte(`{"resultsPerPage":1,"startIndex":0,"totalResults":5,"format":"NVD_CPE","version":"2.0","timestamp":"2024-01-01T00:00:00.000","products":[{"cpe":{"deprecated":false,"cpeName":"cpe:2.3:a:vendor:product_a:1.0:*:*:*:*:*:*:*","cpeNameId":"id","lastModified":"2024-01-01T00:00:00.000","created":"2024-01-01T00:00:00.000"}}]}`)
	client := feedClient(tarGzFeed(t, map[string][]byte{
		"nvdcpe-2.0-chunks/nvdcpe-2.0-chunk-00001.json": body,
	}))

	_, err := fetchCPEFeed(client)
	require.Error(t, err)
	require.Contains(t, err.Error(), "incomplete result")
}

// TestFetchCPEsFromAPIRequiresKey verifies the API source fails fast without a key.
func TestFetchCPEsFromAPIRequiresKey(t *testing.T) {
	_, err := fetchCPEsFromAPI(&mockClient{}, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), apiKeyEnvVar)
}

// TestDownloadFeedResume verifies that a download cut off mid-stream resumes from the
// current byte offset via a Range request instead of starting over.
func TestDownloadFeedResume(t *testing.T) {
	full := []byte("hello world, this is the CPE feed archive payload")
	const split = 12

	orig := waitTimeForRetry
	waitTimeForRetry = 0
	t.Cleanup(func() { waitTimeForRetry = orig })

	calls := 0
	client := &scriptedClient{do: func(req *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1:
			require.Empty(t, req.Header.Get("Range"))
			// Deliver the first part, then simulate a connection reset.
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       &partialBody{data: append([]byte(nil), full[:split]...), err: io.ErrUnexpectedEOF},
			}, nil
		case 2:
			require.Equal(t, fmt.Sprintf("bytes=%d-", split), req.Header.Get("Range"))
			return &http.Response{
				StatusCode: http.StatusPartialContent,
				Body:       &partialBody{data: append([]byte(nil), full[split:]...), err: io.EOF},
			}, nil
		default:
			t.Fatalf("unexpected request #%d", calls)
			return nil, nil
		}
	}}

	tmp, err := os.CreateTemp(t.TempDir(), "feed-*")
	require.NoError(t, err)
	t.Cleanup(func() { tmp.Close() })

	require.NoError(t, downloadFeed(client, tmp))
	require.Equal(t, 2, calls, "download should have resumed once")

	got, err := os.ReadFile(tmp.Name())
	require.NoError(t, err)
	require.Equal(t, full, got, "resumed download should reassemble the full payload")
}

// TestDownloadFeedRangeIgnored verifies that if the server ignores the Range header on
// resume (returns 200 instead of 206), the file is rewritten from scratch.
func TestDownloadFeedRangeIgnored(t *testing.T) {
	full := []byte("complete archive bytes from the very beginning")
	const split = 8

	orig := waitTimeForRetry
	waitTimeForRetry = 0
	t.Cleanup(func() { waitTimeForRetry = orig })

	calls := 0
	client := &scriptedClient{do: func(req *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       &partialBody{data: append([]byte(nil), full[:split]...), err: io.ErrUnexpectedEOF},
			}, nil
		case 2:
			// Server ignores Range and serves the whole file again with 200.
			require.Equal(t, fmt.Sprintf("bytes=%d-", split), req.Header.Get("Range"))
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       &partialBody{data: append([]byte(nil), full...), err: io.EOF},
			}, nil
		default:
			t.Fatalf("unexpected request #%d", calls)
			return nil, nil
		}
	}}

	tmp, err := os.CreateTemp(t.TempDir(), "feed-*")
	require.NoError(t, err)
	t.Cleanup(func() { tmp.Close() })

	require.NoError(t, downloadFeed(client, tmp))

	got, err := os.ReadFile(tmp.Name())
	require.NoError(t, err)
	require.Equal(t, full, got, "file should be rewritten from scratch, not duplicated")
}
