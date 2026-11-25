package mysql

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// hostFields is the set of Host db fields a host list query loads; nil means all of them.
type hostFields map[string]struct{}

func (f hostFields) want(field string) bool {
	if f == nil {
		return true
	}
	_, ok := f[field]
	return ok
}

// hostFieldExpr is a SELECT expression and the Host db field it loads.
type hostFieldExpr struct {
	field string
	expr  string
}

// selectHostFields returns the expressions that load fields, each preceded by a comma.
func selectHostFields(exprs []hostFieldExpr, fields hostFields) string {
	var sb strings.Builder
	for _, e := range exprs {
		if fields.want(e.field) {
			sb.WriteString(",\n    ")
			sb.WriteString(e.expr)
		}
	}
	return sb.String()
}

var hostListSelect = []hostFieldExpr{
	{"id", "h.id"},
	{"osquery_host_id", "h.osquery_host_id"},
	{"created_at", "h.created_at"},
	{"updated_at", "h.updated_at"},
	{"detail_updated_at", "h.detail_updated_at"},
	{"node_key", "h.node_key"},
	{"hostname", "h.hostname"},
	{"uuid", "h.uuid"},
	{"platform", "h.platform"},
	{"osquery_version", "h.osquery_version"},
	{"os_version", "h.os_version"},
	{"build", "h.build"},
	{"platform_like", "h.platform_like"},
	{"code_name", "h.code_name"},
	{"uptime", "h.uptime"},
	{"memory", "h.memory"},
	{"cpu_type", "h.cpu_type"},
	{"cpu_subtype", "h.cpu_subtype"},
	{"cpu_brand", "h.cpu_brand"},
	{"cpu_physical_cores", "h.cpu_physical_cores"},
	{"cpu_logical_cores", "h.cpu_logical_cores"},
	{"hardware_vendor", "h.hardware_vendor"},
	{"hardware_model", "h.hardware_model"},
	{"hardware_version", "h.hardware_version"},
	{"hardware_serial", "h.hardware_serial"},
	{"computer_name", "h.computer_name"},
	{"primary_ip_id", "h.primary_ip_id"},
	{"distributed_interval", "h.distributed_interval"},
	{"logger_tls_period", "h.logger_tls_period"},
	{"config_tls_refresh", "h.config_tls_refresh"},
	{"primary_ip", "h.primary_ip"},
	{"primary_mac", "h.primary_mac"},
	{"label_updated_at", "h.label_updated_at"},
	{"last_enrolled_at", "h.last_enrolled_at"},
	{"refetch_requested", "h.refetch_requested"},
	{"refetch_critical_queries_until", "h.refetch_critical_queries_until"},
	{"team_id", "h.team_id"},
	{"policy_updated_at", "h.policy_updated_at"},
	{"public_ip", "h.public_ip"},
	{"orbit_node_key", "h.orbit_node_key"},
	{"gigs_disk_space_available", "COALESCE(hd.gigs_disk_space_available, 0) as gigs_disk_space_available"},
	{"percent_disk_space_available", "COALESCE(hd.percent_disk_space_available, 0) as percent_disk_space_available"},
	{"gigs_total_disk_space", "COALESCE(hd.gigs_total_disk_space, 0) as gigs_total_disk_space"},
	{"gigs_all_disk_space", "hd.gigs_all_disk_space"},
	{"disk_encryption_enabled", "hd.encrypted as disk_encryption_enabled"},
	{"seen_time", "COALESCE(hst.seen_time, h.created_at) AS seen_time"},
	{"last_mdm_checked_in_at", "nstm.seen_time AS last_mdm_checked_in_at"},
	{"team_name", "t.name AS team_name"},
	{"software_updated_at", "COALESCE(hu.software_updated_at, h.created_at) AS software_updated_at"},
	{"last_restarted_at", "h.last_restarted_at"},
	{"timezone", "h.timezone"},
	{"orbit_version", "hoi.version AS orbit_version"},
	{"fleet_desktop_version", "hoi.desktop_version AS fleet_desktop_version"},
	{"group_tag", "had.group_tag AS group_tag"},
	{"os_name", hostOSNameSelect},
	{"os_version_only", hostOSVersionSelect},
	{"mdm_host_data", strings.TrimPrefix(hostMDMSelect, ",")},
}

var hostIssuesSelect = []hostFieldExpr{
	{"failing_policies_count", "COALESCE(host_issues.failing_policies_count, 0) AS failing_policies_count"},
	{"critical_vulnerabilities_count", "COALESCE(host_issues.critical_vulnerabilities_count, 0) AS critical_vulnerabilities_count"},
	{"total_issues_count", "COALESCE(host_issues.total_issues_count, 0) AS total_issues_count"},
}

// hostReportComputedFields are the report columns the service computes from other Host fields.
var hostReportComputedFields = map[string][]string{
	"status":                  {"platform", "seen_time", "distributed_interval", "config_tls_refresh", "last_mdm_checked_in_at", "label_updated_at"},
	"display_text":            {"hostname"},
	"display_name":            {"computer_name", "hostname", "hardware_model", "hardware_serial"},
	"hardware_marketing_name": {"platform", "hardware_model"},
	"device_mapping":          {"device_mapping"},
	// setHostAPIOS derives both from the OS inventory row and the os_version display string.
	"os_name":    {"os_name", "os_version_only", "os_version"},
	"os_version": {"os_name", "os_version_only", "os_version"},
}

// hostReportColumnFields maps each Host CSV column to the db field that loads it. A struct field
// with nested CSV columns (mdm.*) is loaded as a whole.
var hostReportColumnFields = sync.OnceValue(func() map[string]string {
	m := make(map[string]string)
	var walk func(t reflect.Type)
	walk = func(t reflect.Type) {
		for f := range t.Fields() {
			ft := f.Type
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if f.Anonymous && ft.Kind() == reflect.Struct {
				walk(ft)
				continue
			}
			dbName, _, _ := strings.Cut(f.Tag.Get("db"), ",")
			if dbName == "" {
				dbName = strings.ToLower(f.Name)
			}
			if ft.Kind() == reflect.Struct && ft != reflect.TypeFor[time.Time]() {
				for nested := range ft.Fields() {
					if col := csvColumnName(nested); col != "" {
						m[col] = dbName
					}
				}
				continue
			}
			if col := csvColumnName(f); col != "" {
				m[col] = dbName
			}
		}
	}
	walk(reflect.TypeFor[fleet.Host]())
	return m
})

func csvColumnName(f reflect.StructField) string {
	name, _, _ := strings.Cut(f.Tag.Get("csv"), ",")
	if name == "-" {
		return ""
	}
	return name
}

// hostReportFields returns the Host db fields the report columns need, or nil (every field) when no
// columns are given or one isn't known here.
func hostReportFields(columns []string) hostFields {
	if len(columns) == 0 {
		return nil
	}
	byColumn := hostReportColumnFields()
	fields := hostFields{"id": {}, "platform": {}}
	for _, col := range columns {
		if computed, ok := hostReportComputedFields[col]; ok {
			for _, f := range computed {
				fields[f] = struct{}{}
			}
			continue
		}
		f, ok := byColumn[col]
		if !ok {
			return nil
		}
		fields[f] = struct{}{}
	}
	return fields
}

// optionalHostJoin is a LEFT JOIN that matches at most one row per host, so it can be dropped
// from a query that doesn't reference its alias without changing which rows come back.
type optionalHostJoin struct {
	sql  string
	used *regexp.Regexp
}

func newOptionalHostJoin(sql, alias string) optionalHostJoin {
	return optionalHostJoin{sql: sql, used: regexp.MustCompile(`(^|[^\w.])` + regexp.QuoteMeta(alias) + `\.`)}
}

// A join that another join's ON clause references must come before it.
var hostListOptionalJoins = []optionalHostJoin{
	newOptionalHostJoin(hostSeenTimesJoin, "hst"),
	newOptionalHostJoin(hostUpdatesJoin, "hu"),
	newOptionalHostJoin(hostTeamsJoin, "t"),
	newOptionalHostJoin(hostDisksJoin, "hd"),
	newOptionalHostJoin(hostOrbitInfoJoin, "hoi"),
	newOptionalHostJoin(hostAutopilotJoin, "had"),
	newOptionalHostJoin(hostMDMDataJoin, "hmdm"),
	newOptionalHostJoin(hostDiskEncryptionJoin, "hdek"),
	newOptionalHostJoin(hostDEPAssignmentsJoin, "hdep"),
	newOptionalHostJoin(hostIssuesJoin, "host_issues"),
	newOptionalHostJoin(hostOSInventoryJoin, "hos"),
	newOptionalHostJoin(hostOSRowJoin, "os"),
	newOptionalHostJoin(hostMobileMDMEnrollmentJoin, "nesm"),
	newOptionalHostJoin(hostMobileMDMSeenJoin, "nstm"),
}

// MySQL keeps every LEFT JOIN of a query even when nothing reads it, and each one costs a lookup
// per host, so a report that loads a few columns drops the joins its SELECT, filters and ORDER BY
// don't use.
func dropUnusedHostJoins(stmt string, joins []optionalHostJoin) string {
	for _, j := range slices.Backward(joins) {
		at := strings.Index(stmt, j.sql)
		if at < 0 {
			continue
		}
		without := stmt[:at] + stmt[at+len(j.sql):]
		if !j.used.MatchString(without) {
			stmt = without
		}
	}
	return stmt
}

var hostsInLabelSelect = []hostFieldExpr{
	{"id", "h.id"},
	{"osquery_host_id", "h.osquery_host_id"},
	{"created_at", "h.created_at"},
	{"updated_at", "h.updated_at"},
	{"detail_updated_at", "h.detail_updated_at"},
	{"node_key", "h.node_key"},
	{"hostname", "h.hostname"},
	{"uuid", "h.uuid"},
	{"platform", "h.platform"},
	{"osquery_version", "h.osquery_version"},
	{"os_version", "h.os_version"},
	{"build", "h.build"},
	{"platform_like", "h.platform_like"},
	{"code_name", "h.code_name"},
	{"uptime", "h.uptime"},
	{"memory", "h.memory"},
	{"cpu_type", "h.cpu_type"},
	{"cpu_subtype", "h.cpu_subtype"},
	{"cpu_brand", "h.cpu_brand"},
	{"cpu_physical_cores", "h.cpu_physical_cores"},
	{"cpu_logical_cores", "h.cpu_logical_cores"},
	{"hardware_vendor", "h.hardware_vendor"},
	{"hardware_model", "h.hardware_model"},
	{"hardware_version", "h.hardware_version"},
	{"hardware_serial", "h.hardware_serial"},
	{"computer_name", "h.computer_name"},
	{"primary_ip_id", "h.primary_ip_id"},
	{"distributed_interval", "h.distributed_interval"},
	{"logger_tls_period", "h.logger_tls_period"},
	{"config_tls_refresh", "h.config_tls_refresh"},
	{"primary_ip", "h.primary_ip"},
	{"primary_mac", "h.primary_mac"},
	{"label_updated_at", "h.label_updated_at"},
	{"last_enrolled_at", "h.last_enrolled_at"},
	{"refetch_requested", "h.refetch_requested"},
	{"refetch_critical_queries_until", "h.refetch_critical_queries_until"},
	{"team_id", "h.team_id"},
	{"policy_updated_at", "h.policy_updated_at"},
	{"public_ip", "h.public_ip"},
	{"gigs_disk_space_available", "COALESCE(hd.gigs_disk_space_available, 0) as gigs_disk_space_available"},
	{"percent_disk_space_available", "COALESCE(hd.percent_disk_space_available, 0) as percent_disk_space_available"},
	{"gigs_total_disk_space", "COALESCE(hd.gigs_total_disk_space, 0) as gigs_total_disk_space"},
	{"seen_time", "COALESCE(hst.seen_time, h.created_at) as seen_time"},
	{"last_mdm_checked_in_at", "nstm.seen_time AS last_mdm_checked_in_at"},
	{"software_updated_at", "COALESCE(hu.software_updated_at, h.created_at) AS software_updated_at"},
	{"last_restarted_at", "h.last_restarted_at"},
	{"timezone", "h.timezone"},
	{"orbit_version", "hoi.version AS orbit_version"},
	{"fleet_desktop_version", "hoi.desktop_version AS fleet_desktop_version"},
	{"team_name", "(SELECT name FROM teams t WHERE t.id = h.team_id) AS team_name"},
	{"os_name", hostOSNameSelect},
	{"os_version_only", hostOSVersionSelect},
	{"mdm_host_data", strings.TrimPrefix(hostMDMSelect, ",")},
}

const (
	hostInLabelSeenTimesJoin = "\n    LEFT JOIN host_seen_times hst ON (h.id=hst.host_id)"
	hostInLabelUpdatesJoin   = "\n    LEFT JOIN host_updates hu ON (h.id = hu.host_id)"
	hostInLabelDisksJoin     = "\n    LEFT JOIN host_disks hd ON (h.id=hd.host_id)"
	hostInLabelOrbitInfoJoin = "\n    LEFT JOIN host_orbit_info hoi ON (h.id = hoi.host_id)"
)

func hostInLabelDeviceMappingJoin() string {
	return fmt.Sprintf(`
	LEFT JOIN (
	SELECT
		host_id,
		CONCAT('[', GROUP_CONCAT(JSON_OBJECT('email', email, 'source', %s)), ']') AS device_mapping
	FROM
		host_emails
	GROUP BY
		host_id) dm ON dm.host_id = h.id`, deviceMappingTranslateSourceColumn(""))
}

// A join that another join's ON clause references must come before it.
var hostsInLabelOptionalJoins = []optionalHostJoin{
	newOptionalHostJoin(hostInLabelSeenTimesJoin, "hst"),
	newOptionalHostJoin(hostMobileMDMEnrollmentJoin, "nesm"),
	newOptionalHostJoin(hostMobileMDMSeenJoin, "nstm"),
	newOptionalHostJoin(hostInLabelUpdatesJoin, "hu"),
	newOptionalHostJoin(hostInLabelDisksJoin, "hd"),
	newOptionalHostJoin(hostInLabelOrbitInfoJoin, "hoi"),
	newOptionalHostJoin(hostOSInventoryJoin, "hos"),
	newOptionalHostJoin(hostOSRowJoin, "os"),
	newOptionalHostJoin(hostMDMDataJoin, "hmdm"),
	newOptionalHostJoin(hostDiskEncryptionJoin, "hdek"),
	newOptionalHostJoin(hostDEPAssignmentsJoin, "hdep"),
	newOptionalHostJoin(hostIssuesJoin, "host_issues"),
	newOptionalHostJoin(hostInLabelDeviceMappingJoin(), "dm"),
}
