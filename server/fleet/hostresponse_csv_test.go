package fleet

import (
	"encoding/csv"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/stretchr/testify/require"
)

// The hosts report loads only the fields HostCSVColumnFields lists, so a column must render the
// same with every other field left empty.
func TestHostCSVColumnFields(t *testing.T) {
	header := hostCSVRow(t, nil)
	now := time.Now()
	appleModel := slices.Sorted(maps.Keys(AppleHardwareModelsToMarketingNames))[0]
	// Each host reaches a different branch of Status and DisplayName, so dropping any one of their
	// inputs changes a column for at least one of them.
	hosts := map[string]func(h *Host){
		"macOS": func(h *Host) {
			h.Platform, h.HardwareModel = "darwin", appleModel
			h.SeenTime = now.Add(-30 * time.Minute)
		},
		"iOS checked in": func(h *Host) {
			h.Platform = "ios"
			h.SeenTime, h.LabelUpdatedAt = now.Add(-48*time.Hour), now.Add(-48*time.Hour)
			h.LastMDMCheckedInAt = new(now.Add(-time.Minute))
		},
		"no computer name": func(h *Host) {
			h.ComputerName = ""
		},
		"model and serial only": func(h *Host) {
			h.ComputerName, h.Hostname = "", ""
		},
		"iOS labels updated": func(h *Host) {
			h.Platform = "ios"
			h.SeenTime, h.LabelUpdatedAt = now.Add(-48*time.Hour), now.Add(-time.Minute)
			h.LastMDMCheckedInAt = nil
		},
	}
	for name, setup := range hosts {
		t.Run(name, func(t *testing.T) {
			host := &Host{}
			var seq int
			fillHostTestValues(reflect.ValueOf(host).Elem(), now, &seq)
			setup(host)
			want := hostCSVRow(t, host)

			for i, col := range header {
				fields := HostCSVColumnFields([]string{col})
				require.NotNil(t, fields, "report column %q isn't mapped to the fields it's rendered from", col)
				walkHostDBFields(func(dbName string, f reflect.StructField) {
					if slices.Contains(fields, dbName) || f.Tag.Get("db") == "-" {
						return
					}
					without := *host
					v := reflect.ValueOf(&without).Elem().FieldByIndex(f.Index)
					v.Set(reflect.Zero(v.Type()))
					require.Equal(t, want[i], hostCSVRow(t, &without)[i],
						"column %q changes without field %q; add it to hostCSVComputedColumns", col, dbName)
				})
			}
		})
	}
}

// hostCSVRow renders host as the report does, or returns the header when host is nil.
func hostCSVRow(t *testing.T, host *Host) []string {
	resps := []*HostResponse{}
	if host != nil {
		resps = append(resps, HostResponseForHostCheap(host))
	}
	out, err := gocsv.MarshalString(resps)
	require.NoError(t, err)
	rows, err := csv.NewReader(strings.NewReader(out)).ReadAll()
	require.NoError(t, err)
	return rows[len(rows)-1]
}

// fillHostTestValues gives every field a distinct value, so falling back from one field to
// another (as display_name does) shows up as a change.
func fillHostTestValues(v reflect.Value, now time.Time, seq *int) {
	*seq++
	switch v.Kind() {
	case reflect.Pointer:
		v.Set(reflect.New(v.Type().Elem()))
		fillHostTestValues(v.Elem(), now, seq)
	case reflect.Struct:
		if v.Type() == reflect.TypeFor[time.Time]() {
			v.Set(reflect.ValueOf(now.Add(-time.Hour)))
			return
		}
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				fillHostTestValues(v.Field(i), now, seq)
			}
		}
	case reflect.String:
		v.SetString(fmt.Sprintf("value-%d", *seq))
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(3600)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(3600)
	case reflect.Float32, reflect.Float64:
		v.SetFloat(3600)
	}
}
