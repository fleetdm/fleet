package fleet

import (
	"reflect"
	"time"
)

// AndroidAgentVitals are the vitals the Fleet Android agent collects on the
// device itself, for what AMAPI doesn't report. The agent sends them to
// PUT /api/fleetd/vitals, they're stored in host_android_agent_vitals, and
// the host API returns them flattened into the host.
//
// To add a vital, add a nullable column to host_android_agent_vitals and a
// field here: a *bool, *int64 or *string whose json and db tags are the
// column name, with csv:"-" (see HostMDMAndroidDeviceVitals for why). Tag it
// personal:"hide" when it must not be shown for personally owned hosts.
// Strings are truncated to MDMAndroidDeviceVitalMaxLength, so the column is a
// varchar(255).
//
// Every field is optional. A report writes only the fields it carries and
// leaves the others as they were, so an agent that can't read a vital (or
// predates it) doesn't clear it.
type AndroidAgentVitals struct {
	// Roaming is whether the device's cellular radio is registered on a
	// roaming network. It reveals the owner's travel.
	Roaming *bool `json:"roaming,omitempty" db:"roaming" csv:"-" personal:"hide"`
}

// HostAndroidAgentVitals is what the host API returns for an Android host's
// agent-reported vitals. ReportedAt is when the agent last reported any of
// them.
type HostAndroidAgentVitals struct {
	AndroidAgentVitals
	ReportedAt *time.Time `json:"agent_vitals_reported_at,omitempty" csv:"-"`
}

// IsEmpty reports whether v carries no vitals.
func (v AndroidAgentVitals) IsEmpty() bool {
	columns, _ := vitalColumns(v)
	return len(columns) == 0
}

// ReportedColumns returns the column and value of each vital v carries, in
// field order.
func (v AndroidAgentVitals) ReportedColumns() (columns []string, values []any) {
	return vitalColumns(v)
}

// AndroidAgentVitalColumns returns the column of every Android agent vital,
// in field order.
func AndroidAgentVitalColumns() []string {
	t := reflect.TypeFor[AndroidAgentVitals]()
	columns := make([]string, 0, t.NumField())
	for field := range t.Fields() {
		columns = append(columns, field.Tag.Get("db"))
	}
	return columns
}

// ClearPersonal clears the vitals that must not be shown for a personally
// owned host.
func (v *AndroidAgentVitals) ClearPersonal() {
	clearPersonalVitals(v)
}

func vitalColumns(vitals any) (columns []string, values []any) {
	rv := reflect.ValueOf(vitals)
	for i := range rv.NumField() {
		field := rv.Field(i)
		if field.IsNil() {
			continue
		}
		value := field.Elem().Interface()
		if s, ok := value.(string); ok {
			value = truncateVital(s)
		}
		columns = append(columns, rv.Type().Field(i).Tag.Get("db"))
		values = append(values, value)
	}
	return columns, values
}

func clearPersonalVitals(vitals any) {
	rv := reflect.ValueOf(vitals).Elem()
	for i := range rv.NumField() {
		if rv.Type().Field(i).Tag.Get("personal") == "hide" {
			rv.Field(i).SetZero()
		}
	}
}

// truncateVital counts characters, not bytes, as MySQL does for varchar.
func truncateVital(s string) string {
	if runes := []rune(s); len(runes) > MDMAndroidDeviceVitalMaxLength {
		return string(runes[:MDMAndroidDeviceVitalMaxLength])
	}
	return s
}
