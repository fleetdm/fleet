package main

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
