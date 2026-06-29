package main

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type mockClient struct {
	response *http.Response
}

func (m *mockClient) Do(*http.Request) (*http.Response, error) {
	return m.response, nil
}

func TestCPEDB(t *testing.T) {

	// Find the paths of all input files in the testdata directory.
	paths, err := filepath.Glob(filepath.Join("testdata", "*.json"))
	if err != nil {
		t.Fatal(err)
	}

	for _, p := range paths {
		path := p
		_, filename := filepath.Split(path)
		testName := filename[:len(filename)-len(filepath.Ext(path))]

		// Each path turns into a test: the test name is the filename without the extension.
		t.Run(
			testName, func(t *testing.T) {
				t.Parallel()
				json, err := os.ReadFile(path)
				require.NoError(t, err)

				// Set up HTTP response
				recorder := httptest.NewRecorder()
				recorder.Header().Add("Content-Type", "application/json")
				_, _ = recorder.WriteString(string(json))
				expectedResponse := recorder.Result()

				// Create an HTTP client
				client := mockClient{response: expectedResponse}

				// Temporary directory, which will be automatically cleaned up
				dir := t.TempDir()

				// Call the function under test
				dbPath := getCPEs(&client, "API_KEY", dir)

				// Open up the created DB and get the rows
				db, err := sqlx.Open("sqlite3", dbPath)
				require.NoError(t, err)
				defer db.Close()
				rows, err := db.Query("SELECT * FROM cpe_2")
				require.NoError(t, err)
				require.NoError(t, rows.Err())
				defer rows.Close()

				// Convert rows to string for comparison
				var result [][]string
				cols, _ := rows.Columns()
				for rows.Next() {
					// Setting up for converting row to a string
					pointers := make([]interface{}, len(cols))
					container := make([]string, len(cols))
					for i := range pointers {
						pointers[i] = &container[i]
					}

					require.NoError(t, rows.Scan(pointers...))
					result = append(result, container)
				}

				// Compare result to the <testName>.golden file
				goldenFile := filepath.Join("testdata", testName+".golden")
				golden, err := os.ReadFile(goldenFile)
				require.NoError(t, err)
				require.Equal(t, string(golden), fmt.Sprintf("%s", result))
			},
		)
	}
}

// TestCheckResultCount covers the tolerance for NVD's overcount: a small shortfall
// below totalResults is accepted, but a grossly incomplete pull is rejected with a
// message that states the minimum accepted count and threshold.
func TestCheckResultCount(t *testing.T) {
	for _, tc := range []struct {
		name    string
		got     int
		total   int
		wantErr string // expected substring in the error; "" means no error
	}{
		{"exact match", 100, 100, ""},
		{"nvd overcount tolerated", 1760806, 1761245, ""}, // the real-world failing case
		{"at threshold", 95, 100, ""},
		{"below threshold", 94, 100, "need at least 95"},
		{"rounds up to next whole result", 3, 4, "need at least 4"}, // ceiling: 4*95% = 3.8 -> 4
		{"uninitialized total", 0, 1, "need at least"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkResultCount(tc.got, tc.total)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

// TestGetCPEsToleratesShortfall exercises the full path: NVD reports totalResults=20
// but returns only 19 products, exactly the 95% tolerance boundary, so generation
// should succeed and write all 19 rather than aborting.
func TestGetCPEsToleratesShortfall(t *testing.T) {
	const total, returned = 20, 19
	products := make([]string, returned)
	for i := range products {
		products[i] = fmt.Sprintf(
			`{"cpe":{"deprecated":false,"cpeName":"cpe:2.3:a:vendor:product_%d:1.0:*:*:*:*:*:*:*","cpeNameId":"id-%d"}}`, i, i)
	}
	body := fmt.Sprintf(
		`{"resultsPerPage":%d,"startIndex":0,"totalResults":%d,"format":"NVD_CPE","version":"2.0","timestamp":"2024-01-01T00:00:00.000","products":[%s]}`,
		total, total, strings.Join(products, ","))

	recorder := httptest.NewRecorder()
	recorder.Header().Add("Content-Type", "application/json")
	_, _ = recorder.WriteString(body)
	client := mockClient{response: recorder.Result()}

	dir := t.TempDir()
	dbPath := getCPEs(&client, "API_KEY", dir)

	db, err := sqlx.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()
	var count int
	require.NoError(t, db.Get(&count, "SELECT count(*) FROM cpe_2"))
	require.Equal(t, returned, count, "all returned products should be written despite totalResults being higher")
}
