package osquery_utils

import (
	"context"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/publicip"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/async"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/spf13/cast"
)

type DetailQuery struct {
	// Query is the SQL query string.
	Query string
	// Discovery is the SQL query that defines whether the query will run on the host or not.
	// If not set, Fleet makes sure the query will always run.
	Discovery string
	// Platforms is a list of platforms to run the query on. If this value is
	// empty, run on all platforms.
	Platforms []string
	// IngestFunc translates a query result into an update to the host struct,
	// around data that lives on the hosts table.
	IngestFunc func(ctx context.Context, logger log.Logger, host *fleet.Host, rows []map[string]string) error
	// DirectIngestFunc gathers results from a query and directly works with the datastore to
	// persist them. This is usually used for host data that is stored in a separate table.
	// DirectTaskIngestFunc must not be set if this is set.
	DirectIngestFunc func(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string, failed bool) error
	// DirectTaskIngestFunc is similar to DirectIngestFunc except that it uses a task to
	// ingest the results. This is for ingestion that can be either sync or async.
	// DirectIngestFunc must not be set if this is set.
	DirectTaskIngestFunc func(ctx context.Context, logger log.Logger, host *fleet.Host, task *async.Task, rows []map[string]string, failed bool) error
}

// RunsForPlatform determines whether this detail query should run on the given platform
func (q *DetailQuery) RunsForPlatform(platform string) bool {
	if len(q.Platforms) == 0 {
		return true
	}
	for _, p := range q.Platforms {
		if p == platform {
			return true
		}
	}
	return false
}

// hostDetailQueries defines the detail queries that should be run on the host, as
// well as how the results of those queries should be ingested into the
// fleet.Host data model (via IngestFunc).
//
// This map should not be modified at runtime.
var hostDetailQueries = map[string]DetailQuery{
	"network_interface": {
		Query: `select ia.address, id.mac, id.interface
                        from interface_details id join interface_addresses ia
                               on ia.interface = id.interface where length(mac) > 0
                               order by (ibytes + obytes) desc`,
		IngestFunc: func(ctx context.Context, logger log.Logger, host *fleet.Host, rows []map[string]string) (err error) {
			if len(rows) == 0 {
				logger.Log("component", "service", "method", "IngestFunc", "err",
					"detail_query_network_interface expected 1 or more results")
				return nil
			}

			// Rows are ordered by traffic, so we will get the most active
			// interface by iterating in order
			var firstIPv4, firstIPv6 map[string]string
			for _, row := range rows {
				ip := net.ParseIP(row["address"])
				if ip == nil {
					continue
				}

				// Skip link-local and loopback interfaces
				if ip.IsLinkLocalUnicast() || ip.IsLoopback() {
					continue
				}

				// Skip docker interfaces as these are sometimes heavily
				// trafficked, but rarely the interface that Fleet users want to
				// see. https://github.com/fleetdm/fleet/issues/4754.
				if strings.Contains(row["interface"], "docker") {
					continue
				}

				if strings.Contains(row["address"], ":") {
					// IPv6
					if firstIPv6 == nil {
						firstIPv6 = row
					}
				} else {
					// IPv4
					if firstIPv4 == nil {
						firstIPv4 = row
					}
				}
			}

			var selected map[string]string
			switch {
			// Prefer IPv4
			case firstIPv4 != nil:
				selected = firstIPv4
			// Otherwise IPv6
			case firstIPv6 != nil:
				selected = firstIPv6
			// If only link-local and loopback found, still use the first
			// interface so that we don't get an empty value.
			default:
				selected = rows[0]
			}

			host.PrimaryIP = selected["address"]
			host.PrimaryMac = selected["mac"]
			host.PublicIP = publicip.FromContext(ctx)
			return nil
		},
	},
	"os_version": {
		// Collect operating system information for the `hosts` table.
		// Note that data for `operating_system` and `host_operating_system` tables are ingested via
		// the `os_unix_like` extra detail query below.
		Query: "SELECT * FROM os_version LIMIT 1",
		IngestFunc: func(ctx context.Context, logger log.Logger, host *fleet.Host, rows []map[string]string) error {
			if len(rows) != 1 {
				logger.Log("component", "service", "method", "IngestFunc", "err",
					fmt.Sprintf("detail_query_os_version expected single result got %d", len(rows)))
				return nil
			}

			if build, ok := rows[0]["build"]; ok {
				host.Build = build
			}

			host.Platform = rows[0]["platform"]
			host.PlatformLike = rows[0]["platform_like"]
			host.CodeName = rows[0]["codename"]

			// On centos6 there is an osquery bug that leaves
			// platform empty. Here we workaround.
			if host.Platform == "" &&
				strings.Contains(strings.ToLower(rows[0]["name"]), "centos") {
				host.Platform = "centos"
			}

			if host.Platform != "windows" {
				// Populate `host.OSVersion` for non-Windows hosts.
				// Note Windows-specific registry query is required to populate `host.OSVersion` for
				// Windows that is handled in `os_version_windows` detail query below.
				host.OSVersion = fmt.Sprintf("%v %v", rows[0]["name"], parseOSVersion(
					rows[0]["name"],
					rows[0]["version"],
					rows[0]["major"],
					rows[0]["minor"],
					rows[0]["patch"],
					rows[0]["build"],
				))
			}

			return nil
		},
	},
	"os_version_windows": {
		// Windows-specific registry query is required to populate `host.OSVersion` for Windows.
		Query: `
WITH dv AS (
	SELECT
		data
	FROM
		registry
	WHERE
		path = 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\DisplayVersion'
),
rid AS (
	SELECT
		data
	FROM
		registry
	WHERE
		path = 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\ReleaseId'
)
SELECT
	(
		SELECT
			name
		FROM
			os_version) AS name,
	CASE WHEN EXISTS (
		SELECT
			1
		FROM
			dv) THEN
		(
			SELECT
				data
			FROM
				dv)
		ELSE
			""
	END AS display_version,
	CASE WHEN EXISTS (
		SELECT
			1
		FROM
			rid) THEN
		(
			SELECT
				data
			FROM
				rid)
		ELSE
			""
	END AS release_id`,
		Platforms: []string{"windows"},
		IngestFunc: func(ctx context.Context, logger log.Logger, host *fleet.Host, rows []map[string]string) error {
			if len(rows) != 1 {
				logger.Log("component", "service", "method", "IngestFunc", "err",
					fmt.Sprintf("detail_query_os_version_windows expected single result got %d", len(rows)))
				return nil
			}

			var version string
			switch {
			case rows[0]["display_version"] != "":
				// prefer display version if available
				version = rows[0]["display_version"]
			case rows[0]["release_id"] != "":
				// otherwise release_id if available
				version = rows[0]["release_id"]
			default:
				// empty value
			}
			if version == "" {
				level.Debug(logger).Log(
					"msg", "unable to identify windows version",
					"host", host.Hostname,
				)
			}

			s := fmt.Sprintf("%v %v", rows[0]["name"], version)
			// Shorten "Microsoft Windows" to "Windows" to facilitate display and sorting in UI
			s = strings.Replace(s, "Microsoft Windows", "Windows", 1)
			host.OSVersion = s

			return nil
		},
	},
	"osquery_flags": {
		// Collect the interval info (used for online status
		// calculation) from the osquery flags. We typically control
		// distributed_interval (but it's not required), and typically
		// do not control config_tls_refresh.
		Query: `select name, value from osquery_flags where name in ("distributed_interval", "config_tls_refresh", "config_refresh", "logger_tls_period")`,
		IngestFunc: func(ctx context.Context, logger log.Logger, host *fleet.Host, rows []map[string]string) error {
			var configTLSRefresh, configRefresh uint
			var configRefreshSeen, configTLSRefreshSeen bool
			for _, row := range rows {
				switch row["name"] {

				case "distributed_interval":
					interval, err := strconv.Atoi(EmptyToZero(row["value"]))
					if err != nil {
						return fmt.Errorf("parsing distributed_interval: %w", err)
					}
					host.DistributedInterval = uint(interval)

				case "config_tls_refresh":
					// Prior to osquery 2.4.6, the flag was
					// called `config_tls_refresh`.
					interval, err := strconv.Atoi(EmptyToZero(row["value"]))
					if err != nil {
						return fmt.Errorf("parsing config_tls_refresh: %w", err)
					}
					configTLSRefresh = uint(interval)
					configTLSRefreshSeen = true

				case "config_refresh":
					// After 2.4.6 `config_tls_refresh` was
					// aliased to `config_refresh`.
					interval, err := strconv.Atoi(EmptyToZero(row["value"]))
					if err != nil {
						return fmt.Errorf("parsing config_refresh: %w", err)
					}
					configRefresh = uint(interval)
					configRefreshSeen = true

				case "logger_tls_period":
					interval, err := strconv.Atoi(EmptyToZero(row["value"]))
					if err != nil {
						return fmt.Errorf("parsing logger_tls_period: %w", err)
					}
					host.LoggerTLSPeriod = uint(interval)
				}
			}

			// Since the `config_refresh` flag existed prior to
			// 2.4.6 and had a different meaning, we prefer
			// `config_tls_refresh` if it was set, and use
			// `config_refresh` as a fallback.
			if configTLSRefreshSeen {
				host.ConfigTLSRefresh = configTLSRefresh
			} else if configRefreshSeen {
				host.ConfigTLSRefresh = configRefresh
			}

			return nil
		},
	},
	"osquery_info": {
		Query: "select * from osquery_info limit 1",
		IngestFunc: func(ctx context.Context, logger log.Logger, host *fleet.Host, rows []map[string]string) error {
			if len(rows) != 1 {
				logger.Log("component", "service", "method", "IngestFunc", "err",
					fmt.Sprintf("detail_query_osquery_info expected single result got %d", len(rows)))
				return nil
			}

			host.OsqueryVersion = rows[0]["version"]

			return nil
		},
	},
	"system_info": {
		Query: "select * from system_info limit 1",
		IngestFunc: func(ctx context.Context, logger log.Logger, host *fleet.Host, rows []map[string]string) error {
			if len(rows) != 1 {
				logger.Log("component", "service", "method", "IngestFunc", "err",
					fmt.Sprintf("detail_query_system_info expected single result got %d", len(rows)))
				return nil
			}

			var err error
			host.Memory, err = strconv.ParseInt(EmptyToZero(rows[0]["physical_memory"]), 10, 64)
			if err != nil {
				return err
			}
			host.Hostname = rows[0]["hostname"]
			host.UUID = rows[0]["uuid"]
			host.CPUType = rows[0]["cpu_type"]
			host.CPUSubtype = rows[0]["cpu_subtype"]
			host.CPUBrand = rows[0]["cpu_brand"]
			host.CPUPhysicalCores, err = strconv.Atoi(EmptyToZero(rows[0]["cpu_physical_cores"]))
			if err != nil {
				return err
			}
			host.CPULogicalCores, err = strconv.Atoi(EmptyToZero(rows[0]["cpu_logical_cores"]))
			if err != nil {
				return err
			}
			host.HardwareVendor = rows[0]["hardware_vendor"]
			host.HardwareModel = rows[0]["hardware_model"]
			host.HardwareVersion = rows[0]["hardware_version"]
			host.HardwareSerial = rows[0]["hardware_serial"]
			host.ComputerName = rows[0]["computer_name"]
			return nil
		},
	},
	"uptime": {
		Query: "select * from uptime limit 1",
		IngestFunc: func(ctx context.Context, logger log.Logger, host *fleet.Host, rows []map[string]string) error {
			if len(rows) != 1 {
				logger.Log("component", "service", "method", "IngestFunc", "err",
					fmt.Sprintf("detail_query_uptime expected single result got %d", len(rows)))
				return nil
			}

			uptimeSeconds, err := strconv.Atoi(EmptyToZero(rows[0]["total_seconds"]))
			if err != nil {
				return err
			}
			host.Uptime = time.Duration(uptimeSeconds) * time.Second

			return nil
		},
	},
	"disk_space_unix": {
		Query: `
SELECT (blocks_available * 100 / blocks) AS percent_disk_space_available,
       round((blocks_available * blocks_size *10e-10),2) AS gigs_disk_space_available
FROM mounts WHERE path = '/' LIMIT 1;`,
		Platforms:        append(fleet.HostLinuxOSs, "darwin"),
		DirectIngestFunc: directIngestDiskSpace,
	},
	"disk_space_windows": {
		Query: `
SELECT ROUND((sum(free_space) * 100 * 10e-10) / (sum(size) * 10e-10)) AS percent_disk_space_available,
       ROUND(sum(free_space) * 10e-10) AS gigs_disk_space_available
FROM logical_drives WHERE file_system = 'NTFS' LIMIT 1;`,
		Platforms:        []string{"windows"},
		DirectIngestFunc: directIngestDiskSpace,
	},
	"kubequery_info": {
		Query:      `SELECT * from kubernetes_info`,
		IngestFunc: ingestKubequeryInfo,
		Discovery:  discoveryTable("kubernetes_info"),
	},
}

// extraDetailQueries defines extra detail queries that should be run on the host, as
// well as how the results of those queries should be ingested into the hosts related tables
// (via DirectIngestFunc).
//
// This map should not be modified at runtime.
var extraDetailQueries = map[string]DetailQuery{
	"mdm": {
		Query:            `select enrolled, server_url, installed_from_dep from mdm;`,
		DirectIngestFunc: directIngestMDM,
		Platforms:        []string{"darwin"},
		Discovery:        discoveryTable("mdm"),
	},
	"munki_info": {
		Query:            `select version, errors, warnings from munki_info;`,
		DirectIngestFunc: directIngestMunkiInfo,
		Platforms:        []string{"darwin"},
		Discovery:        discoveryTable("munki_info"),
	},
	"google_chrome_profiles": {
		Query:            `SELECT email FROM google_chrome_profiles WHERE NOT ephemeral AND email <> ''`,
		DirectIngestFunc: directIngestChromeProfiles,
		Discovery:        discoveryTable("google_chrome_profiles"),
	},
	"battery": {
		Query:            `SELECT serial_number, cycle_count, health FROM battery;`,
		Platforms:        []string{"darwin"},
		DirectIngestFunc: directIngestBattery,
		// the "battery" table doesn't need a Discovery query as it is an official
		// osquery table on darwin (https://osquery.io/schema/5.3.0#battery), it is
		// always present.
	},
	"os_windows": {
		// This query is used to populate the `operating_systems` and `host_operating_system`
		// tables. Separately, the `hosts` table is populated via the `os_version` and
		// `os_version_windows` detail queries above.
		Query: `
WITH dv AS (
	SELECT
		data
	FROM
		registry
	WHERE
		path = 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\DisplayVersion'
),
rid AS (
	SELECT
		data
	FROM
		registry
	WHERE
		path = 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\ReleaseId'
)
SELECT
	os.name,
	os.platform,
	os.arch,
	k.version as kernel_version,
	CASE WHEN EXISTS (
		SELECT
			1
		FROM
			dv) THEN
		(
			SELECT
				data
			FROM
				dv)
		ELSE
			""
	END AS display_version,
	CASE WHEN EXISTS (
		SELECT
			1
		FROM
			rid) THEN
		(
			SELECT
				data
			FROM
				rid)
		ELSE
			""
	END AS release_id
FROM 
    os_version os,
    kernel_info k`,
		Platforms:        []string{"windows"},
		DirectIngestFunc: directIngestOSWindows,
	},
	"os_unix_like": {
		// This query is used to populate the `operating_systems` and `host_operating_system`
		// tables. Separately, the `hosts` table is populated via the `os_version` detail
		// query above.
		Query: `
	SELECT
		os.name,
		os.major,
		os.minor,
		os.patch,
		os.build,
		os.arch,
		os.platform,
		os.version AS version,
		k.version AS kernel_version
	FROM
		os_version os,
		kernel_info k`,
		Platforms:        append(fleet.HostLinuxOSs, "darwin"),
		DirectIngestFunc: directIngestOSUnixLike,
	},
}

// discoveryTable returns a query to determine whether a table exists or not.
func discoveryTable(tableName string) string {
	return fmt.Sprintf("SELECT 1 FROM osquery_registry WHERE active = true AND registry = 'table' AND name = '%s';", tableName)
}

const usersQueryStr = `WITH cached_groups AS (select * from groups)
 SELECT uid, username, type, groupname, shell
 FROM users LEFT JOIN cached_groups USING (gid)
 WHERE type <> 'special' AND shell NOT LIKE '%/false' AND shell NOT LIKE '%/nologin' AND shell NOT LIKE '%/shutdown' AND shell NOT LIKE '%/halt' AND username NOT LIKE '%$' AND username NOT LIKE '\_%' ESCAPE '\' AND NOT (username = 'sync' AND shell ='/bin/sync' AND directory <> '')`

func withCachedUsers(query string) string {
	return fmt.Sprintf(query, usersQueryStr)
}

var windowsUpdateHistory = DetailQuery{
	Query:            `SELECT date, title FROM windows_update_history WHERE result_code = 'Succeeded'`,
	Platforms:        []string{"windows"},
	Discovery:        discoveryTable("windows_update_history"),
	DirectIngestFunc: directIngestWindowsUpdateHistory,
}

var softwareMacOS = DetailQuery{
	// Note that we create the cached_users CTE (the WITH clause) in order to suggest to SQLite
	// that it generates the users once instead of once for each UNIONed query. We use CROSS JOIN to
	// ensure that the nested loops in the query generation are ordered correctly for the _extensions
	// tables that need a uid parameter. CROSS JOIN ensures that SQLite does not reorder the loop
	// nesting, which is important as described in https://youtu.be/hcn3HIcHAAo?t=77.
	Query: withCachedUsers(`WITH cached_users AS (%s)
SELECT
  name AS name,
  bundle_short_version AS version,
  'Application (macOS)' AS type,
  bundle_identifier AS bundle_identifier,
  'apps' AS source,
  last_opened_time AS last_opened_at
FROM apps
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Python)' AS type,
  '' AS bundle_identifier,
  'python_packages' AS source,
  0 AS last_opened_at
FROM python_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Chrome)' AS type,
  '' AS bundle_identifier,
  'chrome_extensions' AS source,
  0 AS last_opened_at
FROM cached_users CROSS JOIN chrome_extensions USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Firefox)' AS type,
  '' AS bundle_identifier,
  'firefox_addons' AS source,
  0 AS last_opened_at
FROM cached_users CROSS JOIN firefox_addons USING (uid)
UNION
SELECT
  name As name,
  version AS version,
  'Browser plugin (Safari)' AS type,
  '' AS bundle_identifier,
  'safari_extensions' AS source,
  0 AS last_opened_at
FROM cached_users CROSS JOIN safari_extensions USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Atom)' AS type,
  '' AS bundle_identifier,
  'atom_packages' AS source,
  0 AS last_opened_at
FROM cached_users CROSS JOIN atom_packages USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Homebrew)' AS type,
  '' AS bundle_identifier,
  'homebrew_packages' AS source,
  0 AS last_opened_at
FROM homebrew_packages;
`),
	Platforms:        []string{"darwin"},
	DirectIngestFunc: directIngestSoftware,
}

var scheduledQueryStats = DetailQuery{
	Query: `
			SELECT *,
				(SELECT value from osquery_flags where name = 'pack_delimiter') AS delimiter
			FROM osquery_schedule`,
	DirectTaskIngestFunc: directIngestScheduledQueryStats,
}

var softwareLinux = DetailQuery{
	Query: withCachedUsers(`WITH cached_users AS (%s)
SELECT
  name AS name,
  version AS version,
  'Package (deb)' AS type,
  'deb_packages' AS source,
  '' AS release,
  '' AS vendor,
  '' AS arch
FROM deb_packages
WHERE status = 'install ok installed'
UNION
SELECT
  package AS name,
  version AS version,
  'Package (Portage)' AS type,
  'portage_packages' AS source,
  '' AS release,
  '' AS vendor,
  '' AS arch
FROM portage_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Package (RPM)' AS type,
  'rpm_packages' AS source,
  release AS release,
  vendor AS vendor,
  arch AS arch
FROM rpm_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Package (NPM)' AS type,
  'npm_packages' AS source,
  '' AS release,
  '' AS vendor,
  '' AS arch
FROM npm_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Chrome)' AS type,
  'chrome_extensions' AS source,
  '' AS release,
  '' AS vendor,
  '' AS arch
FROM cached_users CROSS JOIN chrome_extensions USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Firefox)' AS type,
  'firefox_addons' AS source,
  '' AS release,
  '' AS vendor,
  '' AS arch
FROM cached_users CROSS JOIN firefox_addons USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Atom)' AS type,
  'atom_packages' AS source,
  '' AS release,
  '' AS vendor,
  '' AS arch
FROM cached_users CROSS JOIN atom_packages USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Python)' AS type,
  'python_packages' AS source,
  '' AS release,
  '' AS vendor,
  '' AS arch
FROM python_packages;
`),
	Platforms:        fleet.HostLinuxOSs,
	DirectIngestFunc: directIngestSoftware,
}

var softwareWindows = DetailQuery{
	Query: withCachedUsers(`WITH cached_users AS (%s)
SELECT
  name AS name,
  version AS version,
  'Program (Windows)' AS type,
  'programs' AS source,
  publisher AS vendor
FROM programs
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Python)' AS type,
  'python_packages' AS source,
  '' AS vendor
FROM python_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (IE)' AS type,
  'ie_extensions' AS source,
  '' AS vendor
FROM ie_extensions
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Chrome)' AS type,
  'chrome_extensions' AS source,
  '' AS vendor
FROM cached_users CROSS JOIN chrome_extensions USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Firefox)' AS type,
  'firefox_addons' AS source,
  '' AS vendor
FROM cached_users CROSS JOIN firefox_addons USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Chocolatey)' AS type,
  'chocolatey_packages' AS source,
  '' AS vendor
FROM chocolatey_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Atom)' AS type,
  'atom_packages' AS source,
  '' AS vendor
FROM cached_users CROSS JOIN atom_packages USING (uid);
`),
	Platforms:        []string{"windows"},
	DirectIngestFunc: directIngestSoftware,
}

var usersQuery = DetailQuery{
	// Note we use the cached_groups CTE (`WITH` clause) here to suggest to SQLite that it generate
	// the `groups` table only once. Without doing this, on some Windows systems (Domain Controllers)
	// with many user accounts and groups, this query could be very expensive as the `groups` table
	// was generated once for each user.
	Query:            usersQueryStr,
	DirectIngestFunc: directIngestUsers,
}

// directIngestOSWindows ingests selected operating system data from a host on a Windows platform
func directIngestOSWindows(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string, failed bool) error {
	if failed {
		level.Error(logger).Log("op", "directIngestOSWindows", "err", "failed")
		return nil
	}
	if len(rows) != 1 {
		return ctxerr.Errorf(ctx, "directIngestOSWindows invalid number of rows: %d", len(rows))
	}

	hostOS := fleet.OperatingSystem{
		Name:          rows[0]["name"],
		Arch:          rows[0]["arch"],
		KernelVersion: rows[0]["kernel_version"],
		Platform:      rows[0]["platform"],
	}

	var version string
	switch {
	case rows[0]["display_version"] != "":
		// prefer display version if available
		version = rows[0]["display_version"]
	case rows[0]["release_id"] != "":
		// otherwise release_id if available
		version = rows[0]["release_id"]
	default:
		// empty value
	}
	if version == "" {
		level.Debug(logger).Log(
			"msg", "unable to identify windows version",
			"host", host.Hostname,
		)
	}
	hostOS.Version = version

	if err := ds.UpdateHostOperatingSystem(ctx, host.ID, hostOS); err != nil {
		return ctxerr.Wrap(ctx, err, "directIngestOSWindows update host operating system")
	}
	return nil
}

// directIngestOSUnixLike ingests selected operating system data from a host on a Unix-like platform
// (e.g., darwin or linux operating systems)
func directIngestOSUnixLike(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string, failed bool) error {
	if failed {
		level.Error(logger).Log("op", "directIngestOSUnixLike", "err", "failed")
		return nil
	}
	if len(rows) != 1 {
		return ctxerr.Errorf(ctx, "directIngestOSUnixLike invalid number of rows: %d", len(rows))
	}
	name := rows[0]["name"]
	version := rows[0]["version"]
	major := rows[0]["major"]
	minor := rows[0]["minor"]
	patch := rows[0]["patch"]
	build := rows[0]["build"]
	arch := rows[0]["arch"]
	kernelVersion := rows[0]["kernel_version"]
	platform := rows[0]["platform"]

	hostOS := fleet.OperatingSystem{Name: name, Arch: arch, KernelVersion: kernelVersion, Platform: platform}
	hostOS.Version = parseOSVersion(name, version, major, minor, patch, build)

	if err := ds.UpdateHostOperatingSystem(ctx, host.ID, hostOS); err != nil {
		return ctxerr.Wrap(ctx, err, "directIngestOSUnixLike update host operating system")
	}
	return nil
}

// parseOSVersion returns a point release string for an operating system. Parsing rules
// depend on available data, which varies between operating systems.
func parseOSVersion(name string, version string, major string, minor string, patch string, build string) string {
	var osVersion string
	if strings.Contains(strings.ToLower(name), "ubuntu") {
		// Ubuntu takes a different approach to updating patch IDs so we instead use
		// the version string provided after removing the code name.
		regx := regexp.MustCompile(`\(.*\)`)
		osVersion = strings.TrimSpace(regx.ReplaceAllString(version, ""))
	} else if major != "0" || minor != "0" || patch != "0" {
		osVersion = fmt.Sprintf("%s.%s.%s", major, minor, patch)
	} else {
		osVersion = build
	}
	osVersion = strings.Trim(osVersion, ".")

	return osVersion
}

func directIngestChromeProfiles(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string, failed bool) error {
	if failed {
		// assume the extension is not there
		return nil
	}

	mapping := make([]*fleet.HostDeviceMapping, 0, len(rows))
	for _, row := range rows {
		mapping = append(mapping, &fleet.HostDeviceMapping{
			HostID: host.ID,
			Email:  row["email"],
			Source: "google_chrome_profiles",
		})
	}
	return ds.ReplaceHostDeviceMapping(ctx, host.ID, mapping)
}

func directIngestBattery(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string, failed bool) error {
	if failed {
		level.Error(logger).Log("op", "directIngestBattery", "err", "failed")
		return nil
	}

	mapping := make([]*fleet.HostBattery, 0, len(rows))
	for _, row := range rows {
		cycleCount, err := strconv.ParseInt(EmptyToZero(row["cycle_count"]), 10, 64)
		if err != nil {
			return err
		}
		mapping = append(mapping, &fleet.HostBattery{
			HostID:       host.ID,
			SerialNumber: row["serial_number"],
			CycleCount:   int(cycleCount),
			// database type is VARCHAR(40) and since there isn't a
			// canonical list of strings we can get for health, we
			// truncate the value just in case.
			Health: fmt.Sprintf("%.40s", row["health"]),
		})
	}
	return ds.ReplaceHostBatteries(ctx, host.ID, mapping)
}

func directIngestWindowsUpdateHistory(
	ctx context.Context,
	logger log.Logger,
	host *fleet.Host,
	ds fleet.Datastore,
	rows []map[string]string,
	failed bool,
) error {
	if failed {
		level.Error(logger).Log("op", "directIngestWindowsUpdateHistory", "err", "failed")
		return nil
	}

	// The windows update history table will also contain entries for the Defender Antivirus. Unfortunately
	// there's no reliable way to differentiate between those entries and Cumulative OS updates.
	// Since each antivirus update will have the same KB ID, but different 'dates', to
	// avoid trying to insert duplicated data, we group by KB ID and then take the most 'out of
	// date' update in each group.

	uniq := make(map[uint]fleet.WindowsUpdate)
	for _, row := range rows {
		u, err := fleet.NewWindowsUpdate(row["title"], row["date"])
		if err != nil {
			level.Warn(logger).Log("op", "directIngestWindowsUpdateHistory", "skipped", err)
			continue
		}

		if v, ok := uniq[u.KBID]; !ok || v.MoreRecent(u) {
			uniq[u.KBID] = u
		}
	}

	var updates []fleet.WindowsUpdate
	for _, v := range uniq {
		updates = append(updates, v)
	}

	return ds.InsertWindowsUpdates(ctx, host.ID, updates)
}

func directIngestScheduledQueryStats(ctx context.Context, logger log.Logger, host *fleet.Host, task *async.Task, rows []map[string]string, failed bool) error {
	if failed {
		level.Error(logger).Log("op", "directIngestScheduledQueryStats", "err", "failed")
		return nil
	}

	packs := map[string][]fleet.ScheduledQueryStats{}
	for _, row := range rows {
		providedName := row["name"]
		if providedName == "" {
			level.Debug(logger).Log(
				"msg", "host reported scheduled query with empty name",
				"host", host.Hostname,
			)
			continue
		}
		delimiter := row["delimiter"]
		if delimiter == "" {
			level.Debug(logger).Log(
				"msg", "host reported scheduled query with empty delimiter",
				"host", host.Hostname,
			)
			continue
		}

		// Split with a limit of 2 in case query name includes the
		// delimiter. Not much we can do if pack name includes the
		// delimiter.
		trimmedName := strings.TrimPrefix(providedName, "pack"+delimiter)
		parts := strings.SplitN(trimmedName, delimiter, 2)
		if len(parts) != 2 {
			level.Debug(logger).Log(
				"msg", "could not split pack and query names",
				"host", host.Hostname,
				"name", providedName,
				"delimiter", delimiter,
			)
			continue
		}
		packName, scheduledName := parts[0], parts[1]

		stats := fleet.ScheduledQueryStats{
			ScheduledQueryName: scheduledName,
			PackName:           packName,
			AverageMemory:      cast.ToInt(row["average_memory"]),
			Denylisted:         cast.ToBool(row["denylisted"]),
			Executions:         cast.ToInt(row["executions"]),
			Interval:           cast.ToInt(row["interval"]),
			// Cast to int first to allow cast.ToTime to interpret the unix timestamp.
			LastExecuted: time.Unix(cast.ToInt64(row["last_executed"]), 0).UTC(),
			OutputSize:   cast.ToInt(row["output_size"]),
			SystemTime:   cast.ToInt(row["system_time"]),
			UserTime:     cast.ToInt(row["user_time"]),
			WallTime:     cast.ToInt(row["wall_time"]),
		}
		packs[packName] = append(packs[packName], stats)
	}

	packStats := []fleet.PackStats{}
	for packName, stats := range packs {
		packStats = append(
			packStats,
			fleet.PackStats{
				PackName:   packName,
				QueryStats: stats,
			},
		)
	}
	if err := task.RecordScheduledQueryStats(ctx, host.ID, packStats, time.Now()); err != nil {
		return ctxerr.Wrap(ctx, err, "record host pack stats")
	}

	return nil
}

func directIngestSoftware(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string, failed bool) error {
	if failed {
		level.Error(logger).Log("op", "directIngestSoftware", "err", "failed")
		return nil
	}

	var software []fleet.Software
	for _, row := range rows {
		name := row["name"]
		version := row["version"]
		source := row["source"]
		bundleIdentifier := row["bundle_identifier"]
		vendor := row["vendor"]

		if name == "" {
			level.Debug(logger).Log(
				"msg", "host reported software with empty name",
				"host", host.Hostname,
				"version", version,
				"source", source,
			)
			continue
		}
		if source == "" {
			level.Debug(logger).Log(
				"msg", "host reported software with empty name",
				"host", host.Hostname,
				"version", version,
				"name", name,
			)
			continue
		}

		var lastOpenedAt time.Time
		if lastOpenedRaw := row["last_opened_at"]; lastOpenedRaw != "" {
			if lastOpenedEpoch, err := strconv.ParseFloat(lastOpenedRaw, 64); err != nil {
				level.Debug(logger).Log(
					"msg", "host reported software with invalid last opened timestamp",
					"host", host.Hostname,
					"version", version,
					"name", name,
					"last_opened_at", lastOpenedRaw,
				)
			} else if lastOpenedEpoch > 0 {
				lastOpenedAt = time.Unix(int64(lastOpenedEpoch), 0).UTC()
			}
		}

		// Check whether the vendor is longer than the max allowed width and if so, truncate it.
		if utf8.RuneCountInString(vendor) >= fleet.SoftwareVendorMaxLength {
			vendor = fmt.Sprintf(fleet.SoftwareVendorMaxLengthFmt, vendor)
		}

		s := fleet.Software{
			Name:             name,
			Version:          version,
			Source:           source,
			BundleIdentifier: bundleIdentifier,

			Release: row["release"],
			Vendor:  vendor,
			Arch:    row["arch"],
		}
		if !lastOpenedAt.IsZero() {
			s.LastOpenedAt = &lastOpenedAt
		}
		software = append(software, s)
	}

	if err := ds.UpdateHostSoftware(ctx, host.ID, software); err != nil {
		return ctxerr.Wrap(ctx, err, "update host software")
	}

	return nil
}

func directIngestUsers(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string, failed bool) error {
	var users []fleet.HostUser
	for _, row := range rows {
		uid, err := strconv.Atoi(row["uid"])
		if err != nil {
			return fmt.Errorf("converting uid %s to int: %w", row["uid"], err)
		}
		username := row["username"]
		type_ := row["type"]
		groupname := row["groupname"]
		shell := row["shell"]
		u := fleet.HostUser{
			Uid:       uint(uid),
			Username:  username,
			Type:      type_,
			GroupName: groupname,
			Shell:     shell,
		}
		users = append(users, u)
	}
	if len(users) == 0 {
		return nil
	}
	if err := ds.SaveHostUsers(ctx, host.ID, users); err != nil {
		return ctxerr.Wrap(ctx, err, "update host users")
	}
	return nil
}

func directIngestDiskSpace(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string, failed bool) error {
	if failed {
		level.Error(logger).Log("op", "directIngestDiskSpace", "err", "failed")
		return nil
	}
	if len(rows) != 1 {
		logger.Log("component", "service", "method", "directIngestDiskSpace", "err",
			fmt.Sprintf("detail_query_disk_space expected single result got %d", len(rows)))
		return nil
	}

	gigsAvailable, err := strconv.ParseFloat(EmptyToZero(rows[0]["gigs_disk_space_available"]), 64)
	if err != nil {
		return err
	}
	percentAvailable, err := strconv.ParseFloat(EmptyToZero(rows[0]["percent_disk_space_available"]), 64)
	if err != nil {
		return err
	}

	return ds.SetOrUpdateHostDisksSpace(ctx, host.ID, gigsAvailable, percentAvailable)
}

func directIngestMDM(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string, failed bool) error {
	if len(rows) == 0 || failed {
		// assume the extension is not there
		return nil
	}
	if len(rows) > 1 {
		logger.Log("component", "service", "method", "ingestMDM", "warn",
			fmt.Sprintf("mdm expected single result got %d", len(rows)))
	}
	enrolledVal := rows[0]["enrolled"]
	if enrolledVal == "" {
		return ctxerr.Wrap(ctx, fmt.Errorf("missing mdm.enrolled value: %d", host.ID))
	}
	enrolled, err := strconv.ParseBool(enrolledVal)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "parsing enrolled")
	}
	installedFromDepVal := rows[0]["installed_from_dep"]
	installedFromDep := false
	if installedFromDepVal != "" {
		installedFromDep, err = strconv.ParseBool(installedFromDepVal)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "parsing installed_from_dep")
		}
	}

	return ds.SetOrUpdateMDMData(ctx, host.ID, enrolled, rows[0]["server_url"], installedFromDep)
}

func directIngestMunkiInfo(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string, failed bool) error {
	if len(rows) == 0 || failed {
		// assume the extension is not there
		return nil
	}
	if len(rows) > 1 {
		logger.Log("component", "service", "method", "ingestMunkiInfo", "warn",
			fmt.Sprintf("munki_info expected single result got %d", len(rows)))
	}

	errors, warnings := rows[0]["errors"], rows[0]["warnings"]
	errList, warnList := splitCleanSemicolonSeparated(errors), splitCleanSemicolonSeparated(warnings)
	return ds.SetOrUpdateMunkiInfo(ctx, host.ID, rows[0]["version"], errList, warnList)
}

func ingestKubequeryInfo(ctx context.Context, logger log.Logger, host *fleet.Host, rows []map[string]string) error {
	if len(rows) != 1 {
		return fmt.Errorf("kubernetes_info expected single result got: %d", len(rows))
	}

	host.Hostname = fmt.Sprintf("kubequery %s", rows[0]["cluster_name"])

	// These values are not provided by kubequery
	host.OsqueryVersion = "kubequery"
	host.Platform = "kubequery"
	return nil
}

func GetDetailQueries(fleetConfig config.FleetConfig, features *fleet.Features) map[string]DetailQuery {
	generatedMap := make(map[string]DetailQuery)
	for key, query := range hostDetailQueries {
		generatedMap[key] = query
	}
	for key, query := range extraDetailQueries {
		generatedMap[key] = query
	}

	if features != nil && features.EnableSoftwareInventory {
		generatedMap["software_macos"] = softwareMacOS
		generatedMap["software_linux"] = softwareLinux
		generatedMap["software_windows"] = softwareWindows
	}

	if features != nil && features.EnableHostUsers {
		generatedMap["users"] = usersQuery
	}

	if !fleetConfig.Vulnerabilities.DisableWinOSVulnerabilities {
		generatedMap["windows_update_history"] = windowsUpdateHistory
	}

	if fleetConfig.App.EnableScheduledQueryStats {
		generatedMap["scheduled_query_stats"] = scheduledQueryStats
	}

	for _, env := range os.Environ() {
		prefix := "FLEET_DANGEROUS_REPLACE_"
		if !strings.HasPrefix(env, prefix) {
			continue
		}
		if i := strings.Index(env, "="); i >= 0 {
			queryName := strings.ToLower(strings.TrimPrefix(env[:i], prefix))
			newQuery := env[i+1:]
			query := generatedMap[queryName]
			query.Query = newQuery
			generatedMap[queryName] = query
		}
	}

	return generatedMap
}

func splitCleanSemicolonSeparated(s string) []string {
	parts := strings.Split(s, ";")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}
	return cleaned
}
