//go:build darwin
// +build darwin

// based on github.com/kolide/launcher/pkg/osquery/tables
package airport

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/airport/mocks"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/go-kit/log"
	"github.com/osquery/osquery-go/plugin/table"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_generateAirportData_HappyPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string

		// key = option to use in exec ex: "airport --option scan", value = path to data to return ex "testdata/scan.input.txt"
		optionsToReturnFilePath map[string]string

		// this is the dataflatten query that would be written as part of the sql cmd
		query string

		// path to to the file that is a json of the expected output
		expectedResultsFilePath string
	}{
		{
			name: "scan",
			optionsToReturnFilePath: map[string]string{
				"scan": "testdata/scan.input.txt",
			},
			expectedResultsFilePath: "testdata/scan.output.json",
		},
		{
			name: "scan_with_query",
			optionsToReturnFilePath: map[string]string{
				"scan": "testdata/scan.input.txt",
			},
			query:                   "/SSID",
			expectedResultsFilePath: "testdata/scan_with_query.output.json",
		},
		{
			name: "getinfo",
			optionsToReturnFilePath: map[string]string{
				"getinfo": "testdata/getinfo.input.txt",
			},
			expectedResultsFilePath: "testdata/getinfo.output.json",
		},
		{
			name: "getinfo_and_scan",
			optionsToReturnFilePath: map[string]string{
				"scan":    "testdata/scan.input.txt",
				"getinfo": "testdata/getinfo.input.txt",
			},
			expectedResultsFilePath: "testdata/getinfo_and_scan.output.json",
		},
		{
			name: "getinfo_and_scan_with_query",
			optionsToReturnFilePath: map[string]string{
				"scan":    "testdata/scan.input.txt",
				"getinfo": "testdata/getinfo.input.txt",
			},
			query:                   "/SSID",
			expectedResultsFilePath: "testdata/getinfo_and_scan_with_query.output.json",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			constraints := make(map[string][]string)

			executor := &mocks.Executor{}

			for option, filePath := range tt.optionsToReturnFilePath {
				// add option (key) to constraints
				constraints["option"] = append(constraints["option"], option)

				// get data from file
				inputBytes, err := os.ReadFile(filePath)
				require.NoError(t, err)
				executor.On("Exec", option).Return(inputBytes, nil).Once()
			}

			if tt.query != "" {
				constraints["query"] = []string{tt.query}
			}

			got, err := generateAirportData(tablehelpers.MockQueryContext(constraints), executor, log.NewNopLogger())
			require.NoError(t, err)

			executor.AssertExpectations(t)

			wantBytes, err := os.ReadFile(tt.expectedResultsFilePath)
			require.NoError(t, err)

			var want []map[string]string
			err = json.Unmarshal(wantBytes, &want)
			require.NoError(t, err)

			assert.ElementsMatch(t, want, got)
		})
	}
}

func Test_generateAirportData_EdgeCases(t *testing.T) {
	t.Parallel()

	type args struct {
		queryContext table.QueryContext
	}
	tests := []struct {
		name       string
		args       args
		execReturn func() ([]byte, error)
		want       []map[string]string
		assertion  assert.ErrorAssertionFunc
	}{
		{
			name: "exec_error",
			args: args{
				queryContext: table.QueryContext{
					Constraints: map[string]table.ConstraintList{
						"option": {Affinity: "TEXT", Constraints: []table.Constraint{{Operator: table.OperatorEquals, Expression: "getinfo"}}},
					},
				},
			},
			execReturn: func() ([]byte, error) {
				return nil, errors.New("exec error")
			},
			assertion: assert.NoError,
		},
		{
			name: "no_data",
			args: args{
				queryContext: table.QueryContext{
					Constraints: map[string]table.ConstraintList{
						"option": {Affinity: "TEXT", Constraints: []table.Constraint{{Operator: table.OperatorEquals, Expression: "getinfo"}}},
					},
				},
			},
			execReturn: func() ([]byte, error) {
				return nil, nil
			},
			assertion: assert.NoError,
		},
		{
			name: "blank_data",
			args: args{
				queryContext: table.QueryContext{
					Constraints: map[string]table.ConstraintList{
						"option": {Affinity: "TEXT", Constraints: []table.Constraint{{Operator: table.OperatorEquals, Expression: "getinfo"}}},
					},
				},
			},
			execReturn: func() ([]byte, error) {
				return []byte("   "), nil
			},
			assertion: assert.NoError,
		},
		{
			name: "partial_data",
			args: args{
				queryContext: table.QueryContext{
					Constraints: map[string]table.ConstraintList{
						"option": {Affinity: "TEXT", Constraints: []table.Constraint{{Operator: table.OperatorEquals, Expression: "getinfo"}}},
					},
				},
			},
			execReturn: func() ([]byte, error) {
				return []byte("some data:"), nil
			},
			assertion: assert.NoError,
		},
		{
			name: "unsupported_option",
			args: args{
				queryContext: table.QueryContext{
					Constraints: map[string]table.ConstraintList{
						"option": {Affinity: "TEXT", Constraints: []table.Constraint{{Operator: table.OperatorEquals, Expression: "unsupported"}}},
					},
				},
			},
			execReturn: func() ([]byte, error) {
				return nil, nil
			},
			assertion: assert.Error,
		},
		{
			name: "no_options",
			args: args{},
			execReturn: func() ([]byte, error) {
				return nil, nil
			},
			assertion: assert.Error,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			executor := &mocks.Executor{}

			executor.On("Exec", mock.Anything).Return(tt.execReturn()).Once()

			got, err := generateAirportData(tt.args.queryContext, executor, log.NewNopLogger())
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_unmarshallGetInfoOutput(t *testing.T) {
	t.Parallel()

	type args struct {
		reader io.Reader
	}
	tests := []struct {
		name string
		args args
		want map[string]interface{}
	}{
		{
			name: "happy_path",
			args: args{
				reader: strings.NewReader("\nagrCtlRSSI: -55\nagrExtRSSI: 0\n"),
			},
			want: map[string]interface{}{
				"agrCtlRSSI": "-55",
				"agrExtRSSI": "0",
			},
		},
		{
			name: "missing_value",
			args: args{
				reader: strings.NewReader("agrCtlRSSI: -55\nagrExtRSSI"),
			},
			want: map[string]interface{}{
				"agrCtlRSSI": "-55",
			},
		},
		{
			name: "no_data",
			args: args{
				reader: strings.NewReader(""),
			},
			want: map[string]interface{}{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, unmarshallGetInfoOutput(tt.args.reader))
		})
	}
}

func Test_unmarshallScanOuput(t *testing.T) {
	t.Parallel()

	type args struct {
		reader io.Reader
	}
	tests := []struct {
		name string
		args args
		want []map[string]interface{}
	}{
		{
			name: "happy_path",
			args: args{
				reader: strings.NewReader(`
                         SSID BSSID             RSSI CHANNEL HT CC SECURITY (auth/unicast/group)
                i got spaces! a0:a0:a0:a0:a0:a0 -92  108     Y  US WPA(PSK/AES,TKIP/TKIP) RSN(PSK/AES,TKIP/TKIP)
                    no-spaces b1:b1:b1:b1:b1:b1 -91  116     N  EU RSN(PSK/AES/AES)`),
			},
			want: []map[string]interface{}{
				{
					"SSID":                          "i got spaces!",
					"BSSID":                         "a0:a0:a0:a0:a0:a0",
					"RSSI":                          "-92",
					"CHANNEL":                       "108",
					"HT":                            "Y",
					"CC":                            "US",
					"SECURITY (auth/unicast/group)": "WPA(PSK/AES,TKIP/TKIP) RSN(PSK/AES,TKIP/TKIP)",
				},
				{
					"SSID":                          "no-spaces",
					"BSSID":                         "b1:b1:b1:b1:b1:b1",
					"RSSI":                          "-91",
					"CHANNEL":                       "116",
					"HT":                            "N",
					"CC":                            "EU",
					"SECURITY (auth/unicast/group)": "RSN(PSK/AES/AES)",
				},
			},
		},
		{
			name: "no_data",
			args: args{
				reader: strings.NewReader(""),
			},
			want: []map[string]interface{}(nil),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, unmarshallScanOuput(tt.args.reader))
		})
	}
}
