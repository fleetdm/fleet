package osquery_utils

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/publicip"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
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
	DirectIngestFunc func(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error
	// DirectTaskIngestFunc is similar to DirectIngestFunc except that it uses a task to
	// ingest the results. This is for ingestion that can be either sync or async.
	// DirectIngestFunc must not be set if this is set.
	DirectTaskIngestFunc func(ctx context.Context, logger log.Logger, host *fleet.Host, task *async.Task, rows []map[string]string) error
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

// networkInterfaceQuery is the query to use to ingest a host's "Primary IP" and "Primary MAC".
//
// "Primary IP"/"Primary MAC" is the IP/MAC of the interface the system uses when it originates traffic to the default route.
//
// The following was used to determine private IPs:
// https://cs.opensource.google/go/go/+/refs/tags/go1.20.1:src/net/ip.go;l=131-148;drc=c53390b078b4d3b18e3aca8970d4b31d4d82cce1
//
// NOTE: We cannot use `in_cidr_block` because it's available since osquery 5.3.0, so we use
// rudimentary split and string matching for IPv4 and and regex_match for IPv6.
const networkInterfaceQuery = `SELECT
    ia.address,
    id.mac
FROM
    interface_addresses ia
    JOIN interface_details id ON id.interface = ia.interface
	-- On Unix ia.interface is the name of the interface,
	-- whereas on Windows ia.interface is the IP of the interface.
    JOIN routes r ON %s
WHERE
	-- Destination 0.0.0.0/0 is the default route on route tables.
    r.destination = '0.0.0.0' AND r.netmask = 0
	-- Type of route is "gateway" for Unix, "remote" for Windows.
    AND r.type = '%s'
	-- We are only interested on private IPs (some devices have their Public IP as Primary IP too).
    AND (
		-- Private IPv4 addresses.
		inet_aton(ia.address) IS NOT NULL AND (
			split(ia.address, '.', 0) = '10'
			OR (split(ia.address, '.', 0) = '172' AND (CAST(split(ia.address, '.', 1) AS INTEGER) & 0xf0) = 16)
			OR (split(ia.address, '.', 0) = '192' AND split(ia.address, '.', 1) = '168')
		)
		-- Private IPv6 addresses start with 'fc' or 'fd'.
		OR (inet_aton(ia.address) IS NULL AND regex_match(lower(ia.address), '^f[cd][0-9a-f][0-9a-f]:[0-9a-f:]+', 0) IS NOT NULL)
	)
ORDER BY
    r.metric ASC,
	-- Prefer IPv4 addresses over IPv6 addresses if their route have the same metric.
	inet_aton(ia.address) IS NOT NULL DESC
LIMIT 1;`

// hostDetailQueries defines the detail queries that should be run on the host, as
// well as how the results of those queries should be ingested into the
// fleet.Host data model (via IngestFunc).
//
// This map should not be modified at runtime.
var hostDetailQueries = map[string]DetailQuery{
	"network_interface_unix": {
		Query:      fmt.Sprintf(networkInterfaceQuery, "r.interface = ia.interface", "gateway"),
		Platforms:  append(fleet.HostLinuxOSs, "darwin"),
		IngestFunc: ingestNetworkInterface,
	},
	"network_interface_windows": {
		Query:      fmt.Sprintf(networkInterfaceQuery, "r.interface = ia.address", "remote"),
		Platforms:  []string{"windows"},
		IngestFunc: ingestNetworkInterface,
	},
	"network_interface_chrome": {
		Query:      `SELECT ipv4 AS address, mac FROM network_interfaces LIMIT 1`,
		Platforms:  []string{"chrome"},
		IngestFunc: ingestNetworkInterface,
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
		Query: `
	SELECT
		os.name,
		os.version
	FROM
		os_version os`,
		Platforms: []string{"windows"},
		IngestFunc: func(ctx context.Context, logger log.Logger, host *fleet.Host, rows []map[string]string) error {
			if len(rows) != 1 {
				logger.Log("component", "service", "method", "IngestFunc", "err",
					fmt.Sprintf("detail_query_os_version_windows expected single result got %d", len(rows)))
				return nil
			}

			version := rows[0]["version"]
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

func isPublicIP(ip net.IP) bool {
	return !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsPrivate()
}

func ingestNetworkInterface(ctx context.Context, logger log.Logger, host *fleet.Host, rows []map[string]string) error {
	logger = log.With(logger,
		"component", "service",
		"method", "IngestFunc",
		"host", host.Hostname,
		"platform", host.Platform,
	)

	if len(rows) != 1 {
		logger.Log("err", fmt.Sprintf("detail_query_network_interface expected single result, got %d", len(rows)))
		return nil
	}

	host.PrimaryIP = rows[0]["address"]
	host.PrimaryMac = rows[0]["mac"]

	// Attempt to extract public IP from the HTTP request.
	ipStr := publicip.FromContext(ctx)
	ip := net.ParseIP(ipStr)
	if ip != nil {
		if isPublicIP(ip) {
			host.PublicIP = ipStr
		} else {
			level.Debug(logger).Log("err", "IP is not public, ignoring", "ip", ipStr)
			host.PublicIP = ""
		}
	} else {
		logger.Log("err", fmt.Sprintf("expected an IP address, got %s", ipStr))
	}

	return nil
}

func directIngestDiskSpace(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
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

const usesMacOSDiskEncryptionQuery = `SELECT 1 FROM disk_encryption WHERE user_uuid IS NOT "" AND filevault_status = 'on' LIMIT 1`

// extraDetailQueries defines extra detail queries that should be run on the host, as
// well as how the results of those queries should be ingested into the hosts related tables
// (via DirectIngestFunc).
//
// This map should not be modified at runtime.
var extraDetailQueries = map[string]DetailQuery{
	"mdm": {
		Query:            `select enrolled, server_url, installed_from_dep, payload_identifier from mdm;`,
		DirectIngestFunc: directIngestMDMMac,
		Platforms:        []string{"darwin"},
		Discovery:        discoveryTable("mdm"),
	},
	"mdm_windows": {
		Query: `
			SELECT * FROM (
				SELECT "provider_id" AS "key", data as "value" FROM registry
				WHERE path LIKE 'HKEY_LOCAL_MACHINE\Software\Microsoft\Enrollments\%\ProviderID'
				LIMIT 1
			)
			UNION ALL
			SELECT * FROM (
				SELECT "discovery_service_url" AS "key", data as "value" FROM registry
				WHERE path LIKE 'HKEY_LOCAL_MACHINE\Software\Microsoft\Enrollments\%\DiscoveryServiceFullURL'
				LIMIT 1
			)
			UNION ALL
			SELECT * FROM (
				SELECT "is_federated" AS "key", data as "value" FROM registry 
				WHERE path LIKE 'HKEY_LOCAL_MACHINE\Software\Microsoft\Enrollments\%\IsFederated'
				LIMIT 1
			)
			UNION ALL
			SELECT * FROM (
				SELECT "installation_type" AS "key", data as "value" FROM registry
				WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\InstallationType'
				LIMIT 1
			)
			;
		`,
		DirectIngestFunc: directIngestMDMWindows,
		Platforms:        []string{"windows"},
	},
	"munki_info": {
		Query:            `select version, errors, warnings from munki_info;`,
		DirectIngestFunc: directIngestMunkiInfo,
		Platforms:        []string{"darwin"},
		Discovery:        discoveryTable("munki_info"),
	},
	// On ChromeOS, the `users` table returns only the user signed into the primary chrome profile.
	"chromeos_profile_user_info": {
		Query:            `SELECT email FROM users`,
		DirectIngestFunc: directIngestChromeProfiles,
		Platforms:        []string{"chrome"},
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
	SELECT
		os.name,
		os.platform,
		os.arch,
		k.version as kernel_version,
		os.version
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
	"os_chrome": {
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
		os.version AS kernel_version
	FROM
		os_version os`,
		Platforms:        []string{"chrome"},
		DirectIngestFunc: directIngestOSUnixLike,
	},
	"orbit_info": {
		Query:            `SELECT version FROM orbit_info`,
		DirectIngestFunc: directIngestOrbitInfo,
		Discovery:        discoveryTable("orbit_info"),
	},
	"disk_encryption_darwin": {
		Query:            usesMacOSDiskEncryptionQuery,
		Platforms:        []string{"darwin"},
		DirectIngestFunc: directIngestDiskEncryption,
		// the "disk_encryption" table doesn't need a Discovery query as it is an official
		// osquery table on darwin and linux, it is always present.
	},
	"disk_encryption_linux": {
		// This query doesn't do any filtering as we've seen what's possibly an osquery bug because it's returning bad
		// results if we filter further, so we'll do the filtering in Go.
		Query:            `SELECT de.encrypted, m.path FROM disk_encryption de JOIN mounts m ON m.device_alias = de.name;`,
		Platforms:        fleet.HostLinuxOSs,
		DirectIngestFunc: directIngestDiskEncryptionLinux,
		// the "disk_encryption" table doesn't need a Discovery query as it is an official
		// osquery table on darwin and linux, it is always present.
	},
	"disk_encryption_windows": {
		Query:            `SELECT 1 FROM bitlocker_info WHERE drive_letter = 'C:' AND protection_status = 1;`,
		Platforms:        []string{"windows"},
		DirectIngestFunc: directIngestDiskEncryption,
		// the "bitlocker_info" table doesn't need a Discovery query as it is an official
		// osquery table on windows, it is always present.
	},
}

// mdmQueries are used by the Fleet server to compliment certain MDM
// features.
// They are only sent to the device when Fleet's MDM is on and properly
// configured
var mdmQueries = map[string]DetailQuery{
	"mdm_config_profiles_darwin": {
		Query:            `SELECT display_name, identifier, install_date FROM macos_profiles where type = "Configuration";`,
		Platforms:        []string{"darwin"},
		DirectIngestFunc: directIngestMacOSProfiles,
		Discovery:        discoveryTable("macos_profiles"),
	},
	// There are two mutually-exclusive queries used to read the FileVaultPRK depending on which
	// extension tables are discovered on the agent. The preferred query uses the newer custom
	// `filevault_prk` extension table rather than the macadmins `file_lines` table. It is preferred
	// because the `file_lines` implementation uses bufio.ScanLines which drops end of line
	// characters.
	//
	// Both queries depend on the same pre-requisites:
	//
	// 1. FileVault must be enabled with a personal recovery key.
	// 2. The "FileVault Recovery Key Escrow" profile must be configured
	//    in the host.
	//
	// This file is safe to access and well [documented by Apple][1]:
	//
	// > If FileVault is enabled after this payload is installed on the system,
	// > the FileVault PRK will be encrypted with the specified certificate,
	// > wrapped with a CMS envelope and stored at /var/db/FileVaultPRK.dat. The
	// > encrypted data will be made available to the MDM server as part of the
	// > SecurityInfo command.
	// >
	// > Alternatively, if a site uses its own administration
	// > software, it can extract the PRK from the foregoing
	// > location at any time.
	//
	// [1]: https://developer.apple.com/documentation/devicemanagement/fderecoverykeyescrow
	"mdm_disk_encryption_key_file_lines_darwin": {
		Query: fmt.Sprintf(`
	WITH 
		de AS (SELECT IFNULL((%s), 0) as encrypted),
		fl AS (SELECT line FROM file_lines WHERE path = '/var/db/FileVaultPRK.dat')
	SELECT encrypted, hex(line) as hex_line FROM de LEFT JOIN fl;`, usesMacOSDiskEncryptionQuery),
		Platforms:        []string{"darwin"},
		DirectIngestFunc: directIngestDiskEncryptionKeyFileLinesDarwin,
		Discovery:        fmt.Sprintf(`SELECT 1 WHERE EXISTS (%s) AND NOT EXISTS (%s);`, strings.Trim(discoveryTable("file_lines"), ";"), strings.Trim(discoveryTable("filevault_prk"), ";")),
	},
	"mdm_disk_encryption_key_file_darwin": {
		Query: fmt.Sprintf(`
	WITH
		de AS (SELECT IFNULL((%s), 0) as encrypted),
		fv AS (SELECT base64_encrypted as filevault_key FROM filevault_prk)
	SELECT encrypted, filevault_key FROM de LEFT JOIN fv;`, usesMacOSDiskEncryptionQuery),
		Platforms:        []string{"darwin"},
		DirectIngestFunc: directIngestDiskEncryptionKeyFileDarwin,
		Discovery:        discoveryTable("filevault_prk"),
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
  COALESCE(NULLIF(bundle_short_version, ''), bundle_version) AS version,
  'Application (macOS)' AS type,
  bundle_identifier AS bundle_identifier,
  'apps' AS source,
  last_opened_time AS last_opened_at,
  path AS installed_path
FROM apps
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Python)' AS type,
  '' AS bundle_identifier,
  'python_packages' AS source,
  0 AS last_opened_at,
  path AS installed_path
FROM python_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Chrome)' AS type,
  '' AS bundle_identifier,
  'chrome_extensions' AS source,
  0 AS last_opened_at,
  path AS installed_path
FROM cached_users CROSS JOIN chrome_extensions USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Firefox)' AS type,
  '' AS bundle_identifier,
  'firefox_addons' AS source,
  0 AS last_opened_at,
  path AS installed_path
FROM cached_users CROSS JOIN firefox_addons USING (uid)
UNION
SELECT
  name As name,
  version AS version,
  'Browser plugin (Safari)' AS type,
  '' AS bundle_identifier,
  'safari_extensions' AS source,
  0 AS last_opened_at,
  path AS installed_path
FROM cached_users CROSS JOIN safari_extensions USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Atom)' AS type,
  '' AS bundle_identifier,
  'atom_packages' AS source,
  0 AS last_opened_at,
  path AS installed_path
FROM cached_users CROSS JOIN atom_packages USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Homebrew)' AS type,
  '' AS bundle_identifier,
  'homebrew_packages' AS source,
  0 AS last_opened_at,
  path AS installed_path
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
  '' AS arch,
  '' AS installed_path
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
  '' AS arch,
  '' AS installed_path
FROM portage_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Package (RPM)' AS type,
  'rpm_packages' AS source,
  release AS release,
  vendor AS vendor,
  arch AS arch,
  '' AS installed_path
FROM rpm_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Package (NPM)' AS type,
  'npm_packages' AS source,
  '' AS release,
  '' AS vendor,
  '' AS arch,
  path AS installed_path
FROM npm_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Chrome)' AS type,
  'chrome_extensions' AS source,
  '' AS release,
  '' AS vendor,
  '' AS arch,
  path AS installed_path
FROM cached_users CROSS JOIN chrome_extensions USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Firefox)' AS type,
  'firefox_addons' AS source,
  '' AS release,
  '' AS vendor,
  '' AS arch,
  path AS installed_path
FROM cached_users CROSS JOIN firefox_addons USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Atom)' AS type,
  'atom_packages' AS source,
  '' AS release,
  '' AS vendor,
  '' AS arch,
  path AS installed_path
FROM cached_users CROSS JOIN atom_packages USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Python)' AS type,
  'python_packages' AS source,
  '' AS release,
  '' AS vendor,
  '' AS arch,
  path AS installed_path
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
  publisher AS vendor,
  install_location AS installed_path
FROM programs
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Python)' AS type,
  'python_packages' AS source,
  '' AS vendor,
  path AS installed_path
FROM python_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (IE)' AS type,
  'ie_extensions' AS source,
  '' AS vendor,
  path AS installed_path
FROM ie_extensions
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Chrome)' AS type,
  'chrome_extensions' AS source,
  '' AS vendor,
  path AS installed_path
FROM cached_users CROSS JOIN chrome_extensions USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Firefox)' AS type,
  'firefox_addons' AS source,
  '' AS vendor,
  path AS installed_path
FROM cached_users CROSS JOIN firefox_addons USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Chocolatey)' AS type,
  'chocolatey_packages' AS source,
  '' AS vendor,
  path AS installed_path
FROM chocolatey_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Atom)' AS type,
  'atom_packages' AS source,
  '' AS vendor,
  path AS installed_path
FROM cached_users CROSS JOIN atom_packages USING (uid);
`),
	Platforms:        []string{"windows"},
	DirectIngestFunc: directIngestSoftware,
}

var softwareChrome = DetailQuery{
	Query: `SELECT
  name AS name,
  version AS version,
  'Browser plugin (Chrome)' AS type,
  'chrome_extensions' AS source,
  '' AS vendor,
  path AS installed_path
FROM chrome_extensions`,
	Platforms:        []string{"chrome"},
	DirectIngestFunc: directIngestSoftware,
}

var usersQuery = DetailQuery{
	// Note we use the cached_groups CTE (`WITH` clause) here to suggest to SQLite that it generate
	// the `groups` table only once. Without doing this, on some Windows systems (Domain Controllers)
	// with many user accounts and groups, this query could be very expensive as the `groups` table
	// was generated once for each user.
	Query:            usersQueryStr,
	Platforms:        []string{"linux", "darwin", "windows"},
	DirectIngestFunc: directIngestUsers,
}

var usersQueryChrome = DetailQuery{
	Query:            `SELECT uid, username, email FROM users`,
	Platforms:        []string{"chrome"},
	DirectIngestFunc: directIngestUsers,
}

// directIngestOrbitInfo ingests data from the orbit_info extension table.
func directIngestOrbitInfo(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
	if len(rows) != 1 {
		return ctxerr.Errorf(ctx, "directIngestOrbitInfo invalid number of rows: %d", len(rows))
	}
	version := rows[0]["version"]
	if err := ds.SetOrUpdateHostOrbitInfo(ctx, host.ID, version); err != nil {
		return ctxerr.Wrap(ctx, err, "directIngestOrbitInfo update host orbit info")
	}

	return nil
}

// directIngestOSWindows ingests selected operating system data from a host on a Windows platform
func directIngestOSWindows(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
	if len(rows) != 1 {
		return ctxerr.Errorf(ctx, "directIngestOSWindows invalid number of rows: %d", len(rows))
	}

	hostOS := fleet.OperatingSystem{
		Name:          rows[0]["name"],
		Arch:          rows[0]["arch"],
		KernelVersion: rows[0]["kernel_version"],
		Platform:      rows[0]["platform"],
	}

	version := rows[0]["version"]
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
// (e.g., darwin, Linux or ChromeOS)
func directIngestOSUnixLike(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
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
	switch {
	case strings.Contains(strings.ToLower(name), "ubuntu"):
		// Ubuntu takes a different approach to updating patch IDs so we instead use
		// the version string provided after removing the code name.
		regx := regexp.MustCompile(`\(.*\)`)
		osVersion = strings.TrimSpace(regx.ReplaceAllString(version, ""))
	case strings.Contains(strings.ToLower(name), "chrome"):
		osVersion = version
	case major != "0" || minor != "0" || patch != "0":
		osVersion = fmt.Sprintf("%s.%s.%s", major, minor, patch)
	default:
		osVersion = build
	}

	osVersion = strings.Trim(osVersion, ".")

	return osVersion
}

func directIngestChromeProfiles(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
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

func directIngestBattery(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
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
) error {
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

func directIngestScheduledQueryStats(ctx context.Context, logger log.Logger, host *fleet.Host, task *async.Task, rows []map[string]string) error {
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
	if err := task.RecordScheduledQueryStats(ctx, host.TeamID, host.ID, packStats, time.Now()); err != nil {
		return ctxerr.Wrap(ctx, err, "record host pack stats")
	}

	return nil
}

func directIngestSoftware(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
	var software []fleet.Software
	sPaths := map[string]struct{}{}

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

		installedPath := strings.TrimSpace(row["installed_path"])
		if installedPath != "" &&
			// NOTE: osquery is sometimes incorrectly returning the value "null" for some install paths.
			// Thus, we explicitly ignore such value here.
			strings.ToLower(installedPath) != "null" {
			key := fmt.Sprintf("%s%s%s", installedPath, fleet.SoftwareFieldSeparator, s.ToUniqueStr())
			sPaths[key] = struct{}{}
		}
	}

	result, err := ds.UpdateHostSoftware(ctx, host.ID, software)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update host software")
	}

	if err := ds.UpdateHostSoftwareInstalledPaths(ctx, host.ID, sPaths, result); err != nil {
		return ctxerr.Wrap(ctx, err, "update software installed path")
	}

	return nil
}

func directIngestUsers(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
	var users []fleet.HostUser
	for _, row := range rows {
		uid, err := strconv.Atoi(row["uid"])
		if err != nil {
			// Chrome returns uids that are much larger than a 32 bit int, ignore this.
			if host.Platform == "chrome" {
				uid = 0
			} else {
				return fmt.Errorf("converting uid %s to int: %w", row["uid"], err)
			}
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

func directIngestMDMMac(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
	if len(rows) == 0 {
		logger.Log("component", "service", "method", "ingestMDM", "warn",
			fmt.Sprintf("mdm expected single result got %d", len(rows)))
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

	mdmSolutionName := deduceMDMNameMacOS(rows[0])
	if !enrolled && installedFromDep && mdmSolutionName != fleet.WellKnownMDMFleet && host.RefetchCriticalQueriesUntil != nil {
		// the host was unenrolled from a non-Fleet DEP MDM solution, and the
		// refetch critical queries timestamp was set, so clear it.
		host.RefetchCriticalQueriesUntil = nil
	}

	serverURL, err := url.Parse(rows[0]["server_url"])
	if err != nil {
		return ctxerr.Wrap(ctx, err, "parsing server_url")
	}
	// strip any query parameters from the URL
	serverURL.RawQuery = ""

	return ds.SetOrUpdateMDMData(ctx,
		host.ID,
		false,
		enrolled,
		serverURL.String(),
		installedFromDep,
		mdmSolutionName,
	)
}

func deduceMDMNameMacOS(row map[string]string) string {
	// If the PayloadIdentifier is Fleet's MDM then use Fleet as name of the MDM solution.
	// (For Fleet MDM we cannot use the URL because Fleet can be deployed On-Prem.)
	if payloadIdentifier := row["payload_identifier"]; payloadIdentifier == apple_mdm.FleetPayloadIdentifier {
		return fleet.WellKnownMDMFleet
	}
	return fleet.MDMNameFromServerURL(row["server_url"])
}

func deduceMDMNameWindows(data map[string]string) string {
	serverURL := data["discovery_service_url"]
	if serverURL == "" {
		return ""
	}
	if name := data["provider_id"]; name != "" {
		return name
	}
	return fleet.MDMNameFromServerURL(serverURL)
}

func directIngestMDMWindows(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
	data := make(map[string]string, len(rows))
	for _, r := range rows {
		data[r["key"]] = r["value"]
	}
	var enrolled bool
	var automatic bool
	serverURL := data["discovery_service_url"]
	if serverURL != "" {
		enrolled = true
		if isFederated := data["is_federated"]; isFederated == "1" {
			// NOTE: We intentionally nest this condition to eliminate `enrolled == false && automatic == true`
			// as a possible status for Windows hosts (which would be otherwise be categorized as
			// "Pending"). Currently, the "Pending" status is supported only for macOS hosts.
			automatic = true
		}
	}
	isServer := strings.Contains(strings.ToLower(data["installation_type"]), "server")

	return ds.SetOrUpdateMDMData(ctx,
		host.ID,
		isServer,
		enrolled,
		serverURL,
		automatic,
		deduceMDMNameWindows(data),
	)
}

func directIngestMunkiInfo(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
	if len(rows) == 0 {
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

func directIngestDiskEncryptionLinux(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
	encrypted := false
	for _, row := range rows {
		if row["path"] == "/" && row["encrypted"] == "1" {
			encrypted = true
			break
		}
	}

	return ds.SetOrUpdateHostDisksEncryption(ctx, host.ID, encrypted)
}

func directIngestDiskEncryption(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
	encrypted := len(rows) > 0
	return ds.SetOrUpdateHostDisksEncryption(ctx, host.ID, encrypted)
}

// directIngestDiskEncryptionKeyFileDarwin ingests the FileVault key from the `filevault_prk`
// extension table. It is the preferred method when a host has the extension table available.
func directIngestDiskEncryptionKeyFileDarwin(
	ctx context.Context,
	logger log.Logger,
	host *fleet.Host,
	ds fleet.Datastore,
	rows []map[string]string,
) error {
	if len(rows) == 0 {
		// assume the extension is not there
		level.Debug(logger).Log(
			"component", "service",
			"method", "directIngestDiskEncryptionKeyFileDarwin",
			"msg", "no rows or failed",
			"host", host.Hostname,
		)
		return nil
	}

	if len(rows) > 1 {
		level.Debug(logger).Log(
			"component", "service",
			"method", "directIngestDiskEncryptionKeyFileDarwin",
			"msg", fmt.Sprintf("filevault_prk should have a single row, but got %d", len(rows)),
			"host", host.Hostname,
		)
	}

	if rows[0]["encrypted"] != "1" {
		level.Debug(logger).Log(
			"component", "service",
			"method", "directIngestDiskEncryptionKeyFileDarwin",
			"msg", "host does not use disk encryption",
			"host", host.Hostname,
		)
		return nil
	}

	// it's okay if the key comes empty, this can happen and if the disk is
	// encrypted it means we need to reset the encryption key
	return ds.SetOrUpdateHostDiskEncryptionKey(ctx, host.ID, rows[0]["filevault_key"])
}

// directIngestDiskEncryptionKeyFileLinesDarwin ingests the FileVault key from the `file_lines`
// extension table. It is the fallback method in cases where the preferred `filevault_prk` extension
// table is not available on the host.
func directIngestDiskEncryptionKeyFileLinesDarwin(
	ctx context.Context,
	logger log.Logger,
	host *fleet.Host,
	ds fleet.Datastore,
	rows []map[string]string,
) error {
	if len(rows) == 0 {
		// assume the extension is not there
		level.Debug(logger).Log(
			"component", "service",
			"method", "directIngestDiskEncryptionKeyFileLinesDarwin",
			"msg", "no rows or failed",
			"host", host.Hostname,
		)
		return nil
	}

	var hexLines []string
	for _, row := range rows {
		if row["encrypted"] != "1" {
			level.Debug(logger).Log(
				"component", "service",
				"method", "directIngestDiskEncryptionKeyDarwin",
				"msg", "host does not use disk encryption",
				"host", host.Hostname,
			)
			return nil
		}
		hexLines = append(hexLines, row["hex_line"])
	}
	// We concatenate the lines in Go rather than using SQL `group_concat` because the order in
	// which SQL appends the lines is not deterministic, nor guaranteed to be the right order.
	// We assume that hexadecimal 0A (i.e. new line) was the delimiter used to split all lines;
	// however, there are edge cases where this will not be true. It is a known limitation
	// with the `file_lines` extension table and its reliance on bufio.ScanLines that carriage
	// returns will be lost if the source file contains hexadecimal 0D0A (i.e. carriage
	// return preceding new line). In such cases, the stored key will be incorrect.
	b, err := hex.DecodeString(strings.Join(hexLines, "0A"))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "decoding hex string")
	}

	// it's okay if the key comes empty, this can happen and if the disk is
	// encrypted it means we need to reset the encryption key
	return ds.SetOrUpdateHostDiskEncryptionKey(ctx, host.ID, base64.StdEncoding.EncodeToString(b))
}

func directIngestMacOSProfiles(
	ctx context.Context,
	logger log.Logger,
	host *fleet.Host,
	ds fleet.Datastore,
	rows []map[string]string,
) error {
	if len(rows) == 0 {
		// assume the extension is not there
		level.Debug(logger).Log(
			"component", "service",
			"method", "directIngestMacOSProfiles",
			"msg", "no rows or failed",
			"host", host.Hostname,
		)
		return nil
	}

	mapping := make([]*fleet.HostMacOSProfile, 0, len(rows))
	for _, row := range rows {
		installDate, err := time.Parse("2006-01-02 15:04:05 -0700", row["install_date"])
		if err != nil {
			return err
		}
		mapping = append(mapping, &fleet.HostMacOSProfile{
			DisplayName: row["display_name"],
			Identifier:  row["identifier"],
			InstallDate: installDate,
		})
	}

	return ds.UpdateVerificationHostMacOSProfiles(ctx, host, mapping)
}

// go:generate go run gen_queries_doc.go "../../../docs/Using Fleet/Understanding-host-vitals.md"

func GetDetailQueries(
	ctx context.Context,
	fleetConfig config.FleetConfig,
	appConfig *fleet.AppConfig,
	features *fleet.Features,
) map[string]DetailQuery {
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
		generatedMap["software_chrome"] = softwareChrome
	}

	if features != nil && features.EnableHostUsers {
		generatedMap["users"] = usersQuery
		generatedMap["users_chrome"] = usersQueryChrome
	}

	if !fleetConfig.Vulnerabilities.DisableWinOSVulnerabilities {
		generatedMap["windows_update_history"] = windowsUpdateHistory
	}

	if fleetConfig.App.EnableScheduledQueryStats {
		generatedMap["scheduled_query_stats"] = scheduledQueryStats
	}

	if appConfig != nil && appConfig.MDM.EnabledAndConfigured {
		for key, query := range mdmQueries {
			generatedMap[key] = query
		}
	}

	if features != nil {
		var unknownQueries []string

		for name, override := range features.DetailQueryOverrides {
			query, ok := generatedMap[name]
			if !ok {
				unknownQueries = append(unknownQueries, name)
				continue
			}
			if override == nil {
				delete(generatedMap, name)
			} else {
				query.Query = *override
				generatedMap[name] = query
			}
		}

		if len(unknownQueries) > 0 {
			logging.WithErr(ctx, ctxerr.New(ctx, fmt.Sprintf("detail_query_overrides: unknown queries: %s", strings.Join(unknownQueries, ","))))
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
