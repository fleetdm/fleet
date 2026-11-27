package fleet

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"
)

// HostResponse is the response struct that contains the full host information
// along with the host online status and the "display text" to be used when
// rendering in the UI.
type HostResponse struct {
	*Host
	Status                HostStatus   `json:"status" csv:"status"`
	DisplayText           string       `json:"display_text" csv:"display_text"`
	DisplayName           string       `json:"display_name" csv:"display_name"`
	Labels                []*Label     `json:"labels,omitempty" csv:"-"`
	Geolocation           *GeoLocation `json:"geolocation,omitempty" csv:"-"`
	CSVDeviceMapping      string       `json:"-" db:"-" csv:"device_mapping"`
	HardwareMarketingName string       `json:"hardware_marketing_name" csv:"hardware_marketing_name"`
}

// HostResponseForHost returns a HostResponse from Host with Geolocation.
func HostResponseForHost(ctx context.Context, svc Service, host *Host) *HostResponse {
	hr := HostResponseForHostCheap(host)
	hr.Geolocation = svc.LookupGeoIP(ctx, host.PublicIP)
	return hr
}

// HostResponseForHostCheap returns a new HostResponse from a Host without computing Geolocation.
func HostResponseForHostCheap(host *Host) *HostResponse {
	return &HostResponse{
		Host:                  host,
		Status:                host.Status(time.Now()),
		DisplayText:           host.Hostname,
		DisplayName:           host.DisplayName(),
		HardwareMarketingName: host.HardwareMarketingName(),
	}
}

// hostCSVComputedColumns are the hosts report (CSV) columns computed from other Host fields, by
// HostResponseForHostCheap or by the report itself (device_mapping), with the Host db fields each
// reads. TestHostCSVColumnFields fails if a column reads a field missing here.
var hostCSVComputedColumns = map[string][]string{
	"status":                  {"platform", "seen_time", "distributed_interval", "config_tls_refresh", "last_mdm_checked_in_at", "label_updated_at"},
	"display_text":            {"hostname"},
	"display_name":            {"computer_name", "hostname", "hardware_model", "hardware_serial"},
	"hardware_marketing_name": {"platform", "hardware_model"},
	"device_mapping":          {"device_mapping"},
}

// HostCSVColumnFields returns the Host db fields that the given hosts report (CSV) columns are
// rendered from, or nil (every field) when no columns are given or one is unknown.
func HostCSVColumnFields(columns []string) []string {
	if len(columns) == 0 {
		return nil
	}
	byColumn := hostCSVColumnDBFields()
	var fields []string
	for _, col := range columns {
		if computed, ok := hostCSVComputedColumns[col]; ok {
			fields = append(fields, computed...)
			continue
		}
		f, ok := byColumn[col]
		if !ok {
			return nil
		}
		fields = append(fields, f)
	}
	return fields
}

// hostCSVColumnDBFields maps each Host CSV column to the db field it's read from. A struct field
// with nested CSV columns (mdm.*) is loaded as a whole.
var hostCSVColumnDBFields = sync.OnceValue(func() map[string]string {
	m := make(map[string]string)
	walkHostDBFields(func(dbName string, f reflect.StructField) {
		ft := f.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct && ft != reflect.TypeFor[time.Time]() {
			for nested := range ft.Fields() {
				if col := csvColumnName(nested); col != "" {
					m[col] = dbName
				}
			}
			return
		}
		if col := csvColumnName(f); col != "" {
			m[col] = dbName
		}
	})
	return m
})

func walkHostDBFields(fn func(dbName string, f reflect.StructField)) {
	var walk func(t reflect.Type, index []int)
	walk = func(t reflect.Type, index []int) {
		for f := range t.Fields() {
			f.Index = append(slices.Clone(index), f.Index...)
			ft := f.Type
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if f.Anonymous && ft.Kind() == reflect.Struct {
				walk(ft, f.Index)
				continue
			}
			dbName, _, _ := strings.Cut(f.Tag.Get("db"), ",")
			if dbName == "" {
				dbName = strings.ToLower(f.Name)
			}
			fn(dbName, f)
		}
	}
	walk(reflect.TypeFor[Host](), nil)
}

func csvColumnName(f reflect.StructField) string {
	name, _, _ := strings.Cut(f.Tag.Get("csv"), ",")
	if name == "-" {
		return ""
	}
	return name
}

// HostResponsesForHostsCheap returns a HostResponses from Hosts without computing Geolocation.
func HostResponsesForHostsCheap(hosts []Host) []HostResponse {
	hrs := make([]HostResponse, len(hosts))
	for i, h := range hosts {
		h := h
		hrs[i] = *HostResponseForHostCheap(&h)
	}
	return hrs
}

// HostDetailResponse is the response struct that contains the full host information
// with the HostDetail details.
type HostDetailResponse struct {
	HostDetail
	Status                HostStatus   `json:"status"`
	DisplayText           string       `json:"display_text"`
	DisplayName           string       `json:"display_name"`
	Geolocation           *GeoLocation `json:"geolocation,omitempty"`
	HardwareMarketingName string       `json:"hardware_marketing_name"`
}
