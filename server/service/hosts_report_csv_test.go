package service

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gocarina/gocsv"
	"github.com/stretchr/testify/require"
)

func TestHostsReportMatchesBufferedRendering(t *testing.T) {
	values := []string{"=1+1", "plain", "a,b", `say "hi"`, "two\r\nlines", "one\nline", " leading", "", "-1.5", "@sum", "ünïcode"}
	newHosts := func() []*fleet.HostResponse {
		var hosts []*fleet.HostResponse
		for i := range 2 * len(values) {
			h := &fleet.HostResponse{}
			fillCSVTestValues(reflect.ValueOf(h).Elem(), values, i)
			mapping := json.RawMessage(fmt.Sprintf(`[{"email":"a%d@example.com","source":"x"},{"email":"=b@example.com","source":"y"}]`, i))
			h.DeviceMapping = &mapping
			hosts = append(hosts, h)
		}
		return hosts
	}

	all := bufferedHostsReportCSV(t, newHosts(), nil)
	header, err := csv.NewReader(bytes.NewBufferString(all)).Read()
	require.NoError(t, err)
	cases := [][]string{nil, {"hostname", "status", "issues", "mdm.enrollment_status", "device_mapping", "fleet_name", "hardware_marketing_name"}}
	for _, col := range header {
		cases = append(cases, []string{col})
	}
	for _, cols := range cases {
		rr := httptest.NewRecorder()
		hostsReportResponse{Columns: cols, Hosts: newHosts()}.HijackRender(t.Context(), rr)
		require.Equal(t, http.StatusOK, rr.Code)
		require.Equal(t, bufferedHostsReportCSV(t, newHosts(), cols), rr.Body.String(), "columns %v", cols)
	}

	t.Run("no hosts", func(t *testing.T) {
		rr := httptest.NewRecorder()
		hostsReportResponse{Columns: []string{"hostname", "status"}}.HijackRender(t.Context(), rr)
		require.Equal(t, http.StatusOK, rr.Code)
		require.Equal(t, "hostname,status\n", rr.Body.String())
	})

	t.Run("invalid column", func(t *testing.T) {
		rr := httptest.NewRecorder()
		hostsReportResponse{Columns: []string{"hostname", "nosuchcolumn"}, Hosts: newHosts()}.HijackRender(t.Context(), rr)
		require.Equal(t, http.StatusBadRequest, rr.Code)
		require.Contains(t, rr.Body.String(), `invalid column name: \"nosuchcolumn\"`)
	})
}

// bufferedHostsReportCSV is how the report was rendered before it was streamed: gocsv on every
// column into a buffer, parsed back, then filtered.
func bufferedHostsReportCSV(t *testing.T, hosts []*fleet.HostResponse, columns []string) string {
	for _, h := range hosts {
		var dms []struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal(*h.DeviceMapping, &dms))
		h.CSVDeviceMapping = ""
		for i, dm := range dms {
			if i > 0 {
				h.CSVDeviceMapping += ","
			}
			h.CSVDeviceMapping += dm.Email
		}
	}

	var buf bytes.Buffer
	require.NoError(t, gocsv.Marshal(hosts, &buf))
	recs, err := csv.NewReader(&buf).ReadAll()
	require.NoError(t, err)

	var out [][]string
	if len(columns) == 0 {
		applyCSVColumnPlacements(recs)
		out = recs
	} else {
		hdrs := make(map[string]int, len(recs[0]))
		for i, hdr := range recs[0] {
			hdrs[hdr] = i
		}
		out = make([][]string, len(recs))
		for i, rec := range recs {
			for _, col := range columns {
				ix, ok := hdrs[col]
				require.True(t, ok, col)
				out[i] = append(out[i], rec[ix])
			}
		}
	}
	for _, row := range out {
		for i, cell := range row {
			row[i] = sanitizeCSVFormula(cell)
		}
	}

	var res bytes.Buffer
	require.NoError(t, csv.NewWriter(&res).WriteAll(out))
	return res.String()
}

// fillCSVTestValues sets every exported field to a value that depends on seed, leaving some
// pointers nil.
func fillCSVTestValues(v reflect.Value, values []string, seed int) {
	switch v.Kind() {
	case reflect.Pointer:
		if seed%3 == 2 && v.Type().Elem() != reflect.TypeFor[fleet.Host]() {
			return
		}
		v.Set(reflect.New(v.Type().Elem()))
		fillCSVTestValues(v.Elem(), values, seed)
	case reflect.Struct:
		if v.Type() == reflect.TypeFor[time.Time]() {
			v.Set(reflect.ValueOf(time.Date(2026, 1, 2, 3, 4, 5, seed*1000, time.UTC)))
			return
		}
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				fillCSVTestValues(v.Field(i), values, seed+i)
			}
		}
	case reflect.String:
		v.SetString(values[seed%len(values)])
	case reflect.Bool:
		v.SetBool(seed%2 == 0)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(int64(seed - 5))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(uint64(seed)) //nolint:gosec // test values
	case reflect.Float32, reflect.Float64:
		v.SetFloat(float64(seed) / 3)
	}
}
