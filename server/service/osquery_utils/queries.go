package osquery_utils

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/publicip"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/async"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/spf13/cast"
)

type DetailQuery struct {
	// Query is the SQL query string.
	Query string
	// QueryFunc is optionally used to dynamically build a query.
	QueryFunc func(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore) string
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
					rows[0]["extra"],
				))
			}

			return nil
		},
	},
	"os_version_windows": {
		// display_version is not available in some versions of
		// Windows (Server 2019). By including it using a JOIN it can
		// return no rows and the query will still succeed
		Query: `
		WITH display_version_table AS (
			SELECT data as display_version
			FROM registry
			WHERE path = 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\DisplayVersion'
		)
		SELECT
			os.name,
			COALESCE(d.display_version, '') AS display_version,
			k.version
		FROM
			os_version os,
			kernel_info k
		LEFT JOIN
			display_version_table d`,
		Platforms: []string{"windows"},
		IngestFunc: func(ctx context.Context, logger log.Logger, host *fleet.Host, rows []map[string]string) error {
			if len(rows) != 1 {
				logger.Log("component", "service", "method", "IngestFunc", "err",
					fmt.Sprintf("detail_query_os_version_windows expected single result got %d", len(rows)))
				return nil
			}

			s := fmt.Sprintf("%s %s", rows[0]["name"], rows[0]["display_version"])
			// Shorten "Microsoft Windows" to "Windows" to facilitate display and sorting in UI
			s = strings.Replace(s, "Microsoft Windows", "Windows", 1)
			s = strings.TrimSpace(s)
			s += " " + rows[0]["version"]
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
       round((blocks_available * blocks_size * 10e-10),2) AS gigs_disk_space_available,
       round((blocks           * blocks_size * 10e-10),2) AS gigs_total_disk_space
FROM mounts WHERE path = '/' LIMIT 1;`,
		Platforms:        append(fleet.HostLinuxOSs, "darwin"),
		DirectIngestFunc: directIngestDiskSpace,
	},

	"disk_space_windows": {
		Query: `
SELECT ROUND((sum(free_space) * 100 * 10e-10) / (sum(size) * 10e-10)) AS percent_disk_space_available,
       ROUND(sum(free_space) * 10e-10) AS gigs_disk_space_available,
       ROUND(sum(size)       * 10e-10) AS gigs_total_disk_space
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

func ingestNetworkInterface(ctx context.Context, logger log.Logger, host *fleet.Host, rows []map[string]string) error {
	logger = log.With(logger,
		"component", "service",
		"method", "IngestFunc",
		"host", host.Hostname,
		"platform", host.Platform,
	)

	// Attempt to extract public IP from the HTTP request.
	// NOTE: We are executing the IP extraction first to not depend on the network
	// interface query succeeding and returning results.
	ipStr := publicip.FromContext(ctx)
	// First set host.PublicIP to empty to not hide an infrastructure change that
	// misses to set or sets an invalid value in the expected HTTP headers.
	host.PublicIP = ""
	if ipStr != "" {
		ip := net.ParseIP(ipStr)
		if ip != nil {
			host.PublicIP = ipStr
		} else {
			logger.Log("err", fmt.Sprintf("expected an IP address, got %s", ipStr))
		}
	}

	if len(rows) != 1 {
		logger.Log("err", fmt.Sprintf("detail_query_network_interface expected single result, got %d", len(rows)))
		return nil
	}

	host.PrimaryIP = rows[0]["address"]
	host.PrimaryMac = rows[0]["mac"]

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
	gigsTotal, err := strconv.ParseFloat(EmptyToZero(rows[0]["gigs_total_disk_space"]), 64)
	if err != nil {
		return err
	}

	return ds.SetOrUpdateHostDisksSpace(ctx, host.ID, gigsAvailable, percentAvailable, gigsTotal)
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
		// we get most of the MDM information for Windows from the
		// `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Enrollments\%%`
		// registry keys. A computer might many different folders under
		// that path, for different enrollments, so we need to group by
		// enrollment (key in this case) and try to grab the most
		// likely candiate to be an MDM solution.
		//
		// The best way I have found, is to filter by groups of entries
		// with an UPN value, and pick the first one.
		//
		// An example of a host having more than one entry: when
		// the `mdm_bridge` table is used, the `mdmlocalmanagement.dll`
		// registers an MDM with ProviderID = `Local_Management`
		//
		// Entries also need to be filtered by their enrollment status, described [here][1]
		//
		//   Member        Value  Description
		//   unknown       0      Device enrollment state is unknown
		//   enrolled      1      Device is Enrolled.
		//   pendingReset  2      Enrolled but it's enrolled via enrollment profile and the enrolled profile is different from the assigned profile.
		//   failed        3      Not enrolled and there is enrollment failure record.
		//   notContacted  4      Device is imported but not enrolled.
		//   blocked       5      Device is enrolled as userless, but is blocked from moving to user enrollment because the app failed to install.
		//
		// [1]: https://learn.microsoft.com/en-us/graph/api/resources/intune-shared-enrollmentstate
		Query: `
                    WITH registry_keys AS (
                        SELECT *
                        FROM registry
                        WHERE path LIKE 'HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Enrollments\%%'
                    ),
                    enrollment_info AS (
                        SELECT
                            MAX(CASE WHEN name = 'UPN' THEN data END) AS upn,
                            MAX(CASE WHEN name = 'DiscoveryServiceFullURL' THEN data END) AS discovery_service_url,
                            MAX(CASE WHEN name = 'ProviderID' THEN data END) AS provider_id,
                            MAX(CASE WHEN name = 'EnrollmentState' THEN data END) AS state,
                            MAX(CASE WHEN name = 'AADResourceID' THEN data END) AS aad_resource_id
                        FROM registry_keys
                        GROUP BY key
                    ),
                    installation_info AS (
                        SELECT data AS installation_type
                        FROM registry
                        WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\InstallationType'
                        LIMIT 1
                    )
                    SELECT
                        e.aad_resource_id,
                        e.discovery_service_url,
                        e.provider_id,
                        i.installation_type
                    FROM installation_info i
                    LEFT JOIN enrollment_info e ON e.upn IS NOT NULL
		    -- coalesce to 'unknown' and keep that state in the list
		    -- in order to account for hosts that might not have this
		    -- key, and servers
                    WHERE COALESCE(e.state, '0') IN ('0', '1', '2', '3')
                    LIMIT 1;
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
		//
		// DisplayVersion doesn't exist on all versions of Windows (Server 2019).
		// To prevent the query from failing in those cases, we join
		// the values in when they exist, alternatively the column is
		// just empty.
		Query: `
	WITH display_version_table AS (
		SELECT data as display_version
		FROM registry
		WHERE path = 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\DisplayVersion'
	)
	SELECT
		os.name,
		os.platform,
		os.arch,
		k.version as kernel_version,
		os.version,
		COALESCE(d.display_version, '') AS display_version
	FROM
		os_version os,
		kernel_info k
	LEFT JOIN
		display_version_table d`,
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
		os.extra,
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
		Query:            `SELECT * FROM orbit_info`,
		DirectIngestFunc: directIngestOrbitInfo,
		Platforms:        append(fleet.HostLinuxOSs, "darwin", "windows"),
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
		// Bitlocker is an optional component on Windows Server and
		// isn't guaranteed to be installed. If we try to query the
		// bitlocker_info table when the bitlocker component isn't
		// present, the query will crash and fail to report back to
		// the server. Before querying bitlocke_info, we check if it's
		// either:
		// 1. both an optional component, and installed.
		// OR
		// 2. not optional, meaning it's built into the OS
		Query: `
	WITH encrypted(enabled) AS (
		SELECT CASE WHEN
			NOT EXISTS(SELECT 1 FROM windows_optional_features WHERE name = 'BitLocker')
			OR
			(SELECT 1 FROM windows_optional_features WHERE name = 'BitLocker' AND state = 1)
		THEN (SELECT 1 FROM bitlocker_info WHERE drive_letter = 'C:' AND protection_status = 1)
	END)
	SELECT 1 FROM encrypted WHERE enabled IS NOT NULL`,
		Platforms:        []string{"windows"},
		DirectIngestFunc: directIngestDiskEncryption,
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
	"mdm_config_profiles_windows": {
		QueryFunc:        buildConfigProfilesWindowsQuery,
		Platforms:        []string{"windows"},
		DirectIngestFunc: directIngestWindowsProfiles,
		Discovery:        discoveryTable("mdm_bridge"),
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
	"mdm_device_id_windows": {
		Query:            `SELECT name, data FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Provisioning\OMADM\MDMDeviceID\DeviceClientId';`,
		Platforms:        []string{"windows"},
		DirectIngestFunc: directIngestMDMDeviceIDWindows,
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
  '' AS extension_id,
  '' AS browser,
  'apps' AS source,
  '' AS vendor,
  last_opened_time AS last_opened_at,
  path AS installed_path
FROM apps
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Python)' AS type,
  '' AS bundle_identifier,
  '' AS extension_id,
  '' AS browser,
  'python_packages' AS source,
  '' AS vendor,
  0 AS last_opened_at,
  path AS installed_path
FROM python_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Chrome)' AS type,
  '' AS bundle_identifier,
  identifier AS extension_id,
  browser_type AS browser,
  'chrome_extensions' AS source,
  '' AS vendor,
  0 AS last_opened_at,
  path AS installed_path
FROM cached_users CROSS JOIN chrome_extensions USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Firefox)' AS type,
  '' AS bundle_identifier,
  identifier AS extension_id,
  'firefox' AS browser,
  'firefox_addons' AS source,
  '' AS vendor,
  0 AS last_opened_at,
  path AS installed_path
FROM cached_users CROSS JOIN firefox_addons USING (uid)
UNION
SELECT
  name As name,
  version AS version,
  'Browser plugin (Safari)' AS type,
  '' AS bundle_identifier,
  '' AS extension_id,
  '' AS browser,
  'safari_extensions' AS source,
  '' AS vendor,
  0 AS last_opened_at,
  path AS installed_path
FROM cached_users CROSS JOIN safari_extensions USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Homebrew)' AS type,
  '' AS bundle_identifier,
  '' AS extension_id,
  '' AS browser,
  'homebrew_packages' AS source,
  '' AS vendor,
  0 AS last_opened_at,
  path AS installed_path
FROM homebrew_packages;
`),
	Platforms:        []string{"darwin"},
	DirectIngestFunc: directIngestSoftware,
}

// softwareVSCodeExtensions collects VSCode extensions on a separate query for two reasons:
//   - vscode_extensions is not available in osquery < 5.11.0.
//   - Avoid growing the main `software_{macos|windows|linux}` queries
//     (having big queries can cause performance issues or be denylisted).
var softwareVSCodeExtensions = DetailQuery{
	Query: withCachedUsers(`WITH cached_users AS (%s)
SELECT
  name,
  version,
  'IDE extension (VS Code)' AS type,
  '' AS bundle_identifier,
  uuid AS extension_id,
  '' AS browser,
  'vscode_extensions' AS source,
  publisher AS vendor,
  '' AS last_opened_at,
  path AS installed_path
FROM cached_users CROSS JOIN vscode_extensions USING (uid)`),
	Platforms: append(fleet.HostLinuxOSs, "darwin", "windows"),
	Discovery: discoveryTable("vscode_extensions"),
	// Has no IngestFunc, DirectIngestFunc or DirectTaskIngestFunc because
	// the results of this query are appended to the results of the other software queries.
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
  '' AS extension_id,
  '' AS browser,
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
  '' AS extension_id,
  '' AS browser,
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
  '' AS extension_id,
  '' AS browser,
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
  '' AS extension_id,
  '' AS browser,
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
  identifier AS extension_id,
  browser_type AS browser,
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
  identifier AS extension_id,
  'firefox' AS browser,
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
  'Package (Python)' AS type,
  '' AS extension_id,
  '' AS browser,
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
  '' AS extension_id,
  '' AS browser,
  'programs' AS source,
  publisher AS vendor,
  install_location AS installed_path
FROM programs
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Python)' AS type,
  '' AS extension_id,
  '' AS browser,
  'python_packages' AS source,
  '' AS vendor,
  path AS installed_path
FROM python_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (IE)' AS type,
  '' AS extension_id,
  '' AS browser,
  'ie_extensions' AS source,
  '' AS vendor,
  path AS installed_path
FROM ie_extensions
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Chrome)' AS type,
  identifier AS extension_id,
  browser_type AS browser,
  'chrome_extensions' AS source,
  '' AS vendor,
  path AS installed_path
FROM cached_users CROSS JOIN chrome_extensions USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Firefox)' AS type,
  identifier AS extension_id,
  'firefox' AS browser,
  'firefox_addons' AS source,
  '' AS vendor,
  path AS installed_path
FROM cached_users CROSS JOIN firefox_addons USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Chocolatey)' AS type,
  '' AS extension_id,
  '' AS browser,
  'chocolatey_packages' AS source,
  '' AS vendor,
  path AS installed_path
FROM chocolatey_packages
`),
	Platforms:        []string{"windows"},
	DirectIngestFunc: directIngestSoftware,
}

var softwareChrome = DetailQuery{
	Query: `SELECT
  name AS name,
  version AS version,
  identifier AS extension_id,
  browser_type AS browser,
  'Browser plugin (Chrome)' AS type,
  'chrome_extensions' AS source,
  '' AS vendor,
  '' AS installed_path
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
	var desktopVersion sql.NullString
	desktopVersion.String, desktopVersion.Valid = rows[0]["desktop_version"]
	var scriptsEnabled sql.NullBool
	scriptsEnabledStr, ok := rows[0]["scripts_enabled"]
	if ok {
		scriptsEnabled.Bool = scriptsEnabledStr == "1"
		scriptsEnabled.Valid = true
	}
	if err := ds.SetOrUpdateHostOrbitInfo(ctx, host.ID, version, desktopVersion, scriptsEnabled); err != nil {
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
		Version:       rows[0]["kernel_version"],
	}

	displayVersion := rows[0]["display_version"]
	if displayVersion != "" {
		hostOS.Name += " " + displayVersion
		hostOS.DisplayVersion = displayVersion
	}

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
	extra := rows[0]["extra"]
	arch := rows[0]["arch"]
	kernelVersion := rows[0]["kernel_version"]
	platform := rows[0]["platform"]

	hostOS := fleet.OperatingSystem{Name: name, Arch: arch, KernelVersion: kernelVersion, Platform: platform}
	hostOS.Version = parseOSVersion(name, version, major, minor, patch, build, extra)

	if err := ds.UpdateHostOperatingSystem(ctx, host.ID, hostOS); err != nil {
		return ctxerr.Wrap(ctx, err, "directIngestOSUnixLike update host operating system")
	}
	return nil
}

// parseOSVersion returns a point release string for an operating system. Parsing rules
// depend on available data, which varies between operating systems.
func parseOSVersion(name string, version string, major string, minor string, patch string, build string, extra string) string {
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

	// extra is the Apple Rapid Security Response version
	if extra != "" {
		osVersion = fmt.Sprintf("%s %s", osVersion, strings.TrimSpace(extra))
	}

	return osVersion
}

func directIngestChromeProfiles(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
	mapping := make([]*fleet.HostDeviceMapping, 0, len(rows))
	for _, row := range rows {
		mapping = append(mapping, &fleet.HostDeviceMapping{
			HostID: host.ID,
			Email:  row["email"],
			Source: fleet.DeviceMappingGoogleChromeProfiles,
		})
	}
	return ds.ReplaceHostDeviceMapping(ctx, host.ID, mapping, fleet.DeviceMappingGoogleChromeProfiles)
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
			// If the update failed to parse then we log a debug error and ignore it.
			// E.g. we've seen KB updates with titles like "Logitech - Image - 1.4.40.0".
			level.Debug(logger).Log("op", "directIngestWindowsUpdateHistory", "skipped", err)
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

		// Do not save stats without executions so that we do not overwrite existing stats.
		// It is normal for host to have no executions when the query just got scheduled.
		executions := cast.ToUint64(row["executions"])
		if executions == 0 {
			level.Debug(logger).Log(
				"msg", "host reported scheduled query with no executions",
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

		// Handle rare case when wall_time_ms is missing (for osquery < 5.3.0)
		wallTimeMs := cast.ToUint64(row["wall_time_ms"])
		if wallTimeMs == 0 {
			wallTime := cast.ToUint64(row["wall_time"])
			if wallTime != 0 {
				wallTimeMs = wallTime * 1000
			}
		}
		stats := fleet.ScheduledQueryStats{
			ScheduledQueryName: scheduledName,
			PackName:           packName,
			AverageMemory:      cast.ToUint64(row["average_memory"]),
			Denylisted:         cast.ToBool(row["denylisted"]),
			Executions:         executions,
			Interval:           cast.ToInt(row["interval"]),
			// Cast to int first to allow cast.ToTime to interpret the unix timestamp.
			LastExecuted: time.Unix(cast.ToInt64(row["last_executed"]), 0).UTC(),
			OutputSize:   cast.ToUint64(row["output_size"]),
			SystemTime:   cast.ToUint64(row["system_time"]),
			UserTime:     cast.ToUint64(row["user_time"]),
			WallTimeMs:   wallTimeMs,
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
		// Attempt to parse the last_opened_at and emit a debug log if it fails.
		if _, err := fleet.ParseSoftwareLastOpenedAtRowValue(row["last_opened_at"]); err != nil {
			level.Debug(logger).Log(
				"msg", "host reported software with invalid last opened timestamp",
				"host_id", host.ID,
				"row", fmt.Sprintf("%+v", row),
			)
		}

		s, err := fleet.SoftwareFromOsqueryRow(
			row["name"],
			row["version"],
			row["source"],
			row["vendor"],
			row["installed_path"],
			row["release"],
			row["arch"],
			row["bundle_identifier"],
			row["extension_id"],
			row["browser"],
			row["last_opened_at"],
		)
		if err != nil {
			level.Debug(logger).Log(
				"msg", "failed to parse software row",
				"host_id", host.ID,
				"row", fmt.Sprintf("%+v", row),
				"err", err,
			)
			continue
		}

		sanitizeSoftware(host, s, logger)

		if shouldRemoveSoftware(host, s) {
			continue
		}

		software = append(software, *s)

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

var (
	macOSMSTeamsVersion = regexp.MustCompile(`(\d).00.(\d)(\d+)`)
	citrixName          = regexp.MustCompile(`Citrix Workspace [0-9]+`)
)

// sanitizeSoftware performs any sanitization required to the ingested software fields.
//
// Some fields are reported with known incorrect values and we need to fix them before using them.
func sanitizeSoftware(h *fleet.Host, s *fleet.Software, logger log.Logger) {
	softwareSanitizers := []struct {
		checkSoftware  func(*fleet.Host, *fleet.Software) bool
		mutateSoftware func(*fleet.Software)
	}{
		// "Microsoft Teams" on macOS defines the `bundle_short_version` (CFBundleShortVersionString) in a different
		// unexpected version format. Thus here we transform the version string to the expected format
		// (see https://learn.microsoft.com/en-us/officeupdates/teams-app-versioning).
		// E.g. `bundle_short_version` comes with `1.00.622155` and instead it should be transformed
		// to `1.6.00.22155` || s.Name == "Microsoft Teams (work or school).app".

		// Note: in December 2023, Microsoft released "New Teams" for MacOS. This new version of
		// Teams uses a completely different versioning scheme, which is documented at the URL
		// above. Existing versions of Teams on MacOS were renamed to "Microsoft Teams Classic" and still use
		// the same versioning scheme discussed above.
		{
			checkSoftware: func(h *fleet.Host, s *fleet.Software) bool {
				return h.Platform == "darwin" && (s.Name == "Microsoft Teams.app" || s.Name == "Microsoft Teams classic.app")
			},
			mutateSoftware: func(s *fleet.Software) {
				if matches := macOSMSTeamsVersion.FindStringSubmatch(s.Version); len(matches) > 0 {
					s.Version = fmt.Sprintf("%s.%s.00.%s", matches[1], matches[2], matches[3])
				}
			},
		},
		// In the Windows Registry, Cloudflare WARP defines its major version with the last two digits, e.g. `23.9.248.0`.
		// On NVD, the vulnerabilities are reported using the full year, e.g. `2023.9.248.0`.
		{
			checkSoftware: func(h *fleet.Host, s *fleet.Software) bool {
				return h.Platform == "windows" && s.Name == "Cloudflare WARP" && s.Source == "programs"
			},
			mutateSoftware: func(s *fleet.Software) {
				// Perform some sanity check on the version before mutating it.
				parts := strings.Split(s.Version, ".")
				if len(parts) <= 1 {
					level.Debug(logger).Log("msg", "failed to parse software version", "name", s.Name, "version", s.Version)
					return
				}
				_, err := strconv.Atoi(parts[0])
				if err != nil {
					level.Debug(logger).Log("msg", "failed to parse software version", "name", s.Name, "version", s.Version, "err", err)
					return
				}
				// In case Cloudflare starts returning the full year.
				if len(parts[0]) == 4 {
					return
				}
				s.Version = "20" + s.Version // Cloudflare WARP was released on 2019.
			},
		},
		{
			checkSoftware: func(h *fleet.Host, s *fleet.Software) bool {
				return citrixName.Match([]byte(s.Name)) || s.Name == "Citrix Workspace.app"
			},
			mutateSoftware: func(s *fleet.Software) {
				parts := strings.Split(s.Version, ".")
				if len(parts) <= 1 {
					level.Debug(logger).Log("msg", "failed to parse software version", "name", s.Name, "version", s.Version)
					return
				}

				if len(parts[0]) > 2 {
					// then the versioning is correct, so no need to change
					return
				}

				part1, err := strconv.Atoi(parts[0])
				if err != nil {
					level.Debug(logger).Log("msg", "failed to parse software version", "name", s.Name, "version", s.Version, "err", err)
					return
				}

				part2, err := strconv.Atoi(parts[1])
				if err != nil {
					level.Debug(logger).Log("msg", "failed to parse software version", "name", s.Name, "version", s.Version, "err", err)
					return
				}

				newFirstPart := part1*100 + part2
				newFirstStr := strconv.Itoa(newFirstPart)
				newParts := []string{newFirstStr}
				newParts = append(newParts, parts[2:]...)
				s.Version = strings.Join(newParts, ".")
			},
		},
	}

	for _, softwareSanitizer := range softwareSanitizers {
		if softwareSanitizer.checkSoftware(h, s) {
			softwareSanitizer.mutateSoftware(s)
			return
		}
	}
}

// shouldRemoveSoftware returns whether or not we should remove the given Software item from this
// host's software list.
func shouldRemoveSoftware(h *fleet.Host, s *fleet.Software) bool {
	// Parallels is a common VM software for MacOS. Parallels makes the VM's applications
	// visible in the host as MacOS applications, which leads to confusing output (e.g. a MacOS
	// host reporting that it has Notepad installed when this is just an app from the Windows VM
	// under Parallels). We want to filter out those "applications" to avoid confusion.
	return h.Platform == "darwin" && strings.HasPrefix(s.BundleIdentifier, "com.parallels.winapp")
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

	if host.RefetchCriticalQueriesUntil != nil {
		level.Debug(logger).Log("msg", "ingesting macos mdm data during refetch critical queries window", "host_id", host.ID,
			"data", fmt.Sprintf("%+v", rows))
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

	// if the MDM solution is Fleet, we need to extract the enrollment reference from the URL and
	// upsert host emails based on the MDM IdP account associated with the enrollment reference
	var fleetEnrollRef string
	if mdmSolutionName == fleet.WellKnownMDMFleet {
		fleetEnrollRef = serverURL.Query().Get(mobileconfig.FleetEnrollReferenceKey)
		if fleetEnrollRef == "" {
			// TODO: We have some inconsistencies where we use enroll_reference sometimes and
			// enrollment_reference other times. It really should be the same everywhere, but
			// it seems to be working now because the values are matching where they need to match.
			// We should clean this up at some point, but for now we'll just check both.
			fleetEnrollRef = serverURL.Query().Get("enrollment_reference")
		}
		if fleetEnrollRef != "" {
			if err := ds.SetOrUpdateHostEmailsFromMdmIdpAccounts(ctx, host.ID, fleetEnrollRef); err != nil {
				return ctxerr.Wrap(ctx, err, "updating host emails from mdm idp accounts")
			}
		}
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
		fleetEnrollRef,
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

	if name := data["provider_id"]; name == fleet.WellKnownMDMFleet {
		return name
	}

	return fleet.MDMNameFromServerURL(serverURL)
}

func directIngestMDMWindows(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
	if len(rows) == 0 {
		// no mdm information in the registry
		return ds.SetOrUpdateMDMData(ctx, host.ID, false, false, "", false, "", "")
	}
	if len(rows) > 1 {
		logger.Log("component", "service", "method", "directIngestMDMWindows", "warn",
			fmt.Sprintf("mdm expected single result got %d", len(rows)))
		// assume the extension is not there
		return nil
	}

	data := rows[0]
	var enrolled bool
	var automatic bool
	serverURL := data["discovery_service_url"]
	if serverURL != "" {
		enrolled = true
		if data["aad_resource_id"] != "" {
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
		"",
	)
}

func directIngestMunkiInfo(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
	if len(rows) == 0 {
		// munki is not there, and we need to mark it deleted if it was there before
		return ds.SetOrUpdateMunkiInfo(ctx, host.ID, "", []string{}, []string{})
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

	// at this point we know that the disk is encrypted, if the key is
	// empty then the disk is not decryptable. For example an user might
	// have removed the `/var/db/FileVaultPRK.dat` or the computer might
	// have been encrypted without FV escrow enabled.
	var decryptable *bool
	base64Key := rows[0]["filevault_key"]
	if base64Key == "" {
		decryptable = ptr.Bool(false)
	}
	return ds.SetOrUpdateHostDiskEncryptionKey(ctx, host.ID, base64Key, "", decryptable)
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

	// at this point we know that the disk is encrypted, if the key is
	// empty then the disk is not decryptable. For example an user might
	// have removed the `/var/db/FileVaultPRK.dat` or the computer might
	// have been encrypted without FV escrow enabled.
	var decryptable *bool
	base64Key := base64.StdEncoding.EncodeToString(b)
	if base64Key == "" {
		decryptable = ptr.Bool(false)
	}

	return ds.SetOrUpdateHostDiskEncryptionKey(ctx, host.ID, base64Key, "", decryptable)
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

	installed := make(map[string]*fleet.HostMacOSProfile, len(rows))
	for _, row := range rows {
		installDate, err := time.Parse("2006-01-02 15:04:05 -0700", row["install_date"])
		if err != nil {
			return err
		}
		if installDate.IsZero() {
			// this should never happen, but if it does, we should log it
			level.Debug(logger).Log(
				"component", "service",
				"method", "directIngestMacOSProfiles",
				"msg", "profile install date is zero value",
				"host", host.Hostname,
			)
		}
		if _, ok := installed[row["identifier"]]; ok {
			// this should never happen, but if it does, we should log it
			level.Debug(logger).Log(
				"component", "service",
				"method", "directIngestMacOSProfiles",
				"msg", "duplicate profile identifier",
				"host", host.Hostname,
				"identifier", row["identifier"],
			)
		}
		installed[row["identifier"]] = &fleet.HostMacOSProfile{
			DisplayName: row["display_name"],
			Identifier:  row["identifier"],
			InstallDate: installDate,
		}
	}
	return apple_mdm.VerifyHostMDMProfiles(ctx, ds, host, installed)
}

func directIngestMDMDeviceIDWindows(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
	if len(rows) == 0 {
		// this registry key is only going to be present if the device is enrolled to mdm so assume that mdm is turned off
		return nil
	}

	if len(rows) > 1 {
		return ctxerr.Errorf(ctx, "directIngestMDMDeviceIDWindows invalid number of rows: %d", len(rows))
	}
	return ds.UpdateMDMWindowsEnrollmentsHostUUID(ctx, host.UUID, rows[0]["data"])
}

//go:generate go run gen_queries_doc.go "../../../docs/Using Fleet/Understanding-host-vitals.md"

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
		generatedMap["software_vscode_extensions"] = softwareVSCodeExtensions
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

	if appConfig != nil && (appConfig.MDM.EnabledAndConfigured || appConfig.MDM.WindowsEnabledAndConfigured) {
		for key, query := range mdmQueries {
			if slices.Equal(query.Platforms, []string{"windows"}) && !appConfig.MDM.WindowsEnabledAndConfigured {
				continue
			}
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
			if override == nil || *override == "" {
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

func buildConfigProfilesWindowsQuery(
	ctx context.Context,
	logger log.Logger,
	host *fleet.Host,
	ds fleet.Datastore,
) string {
	var sb strings.Builder
	sb.WriteString("<SyncBody>")
	gotProfiles := false
	err := microsoft_mdm.LoopHostMDMLocURIs(ctx, ds, host, func(profile *fleet.ExpectedMDMProfile, hash, locURI, data string) {
		// Per the [docs][1], to `<Get>` configurations you must
		// replace `/Policy/Config` with `Policy/Result`
		// [1]: https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-configuration-service-provider
		locURI = strings.Replace(locURI, "/Policy/Config", "/Policy/Result", 1)
		sb.WriteString(
			// NOTE: intentionally building the xml as a one-liner
			// to prevent any errors in the query.
			fmt.Sprintf(
				"<Get><CmdID>%s</CmdID><Item><Target><LocURI>%s</LocURI></Target></Item></Get>",
				hash,
				locURI,
			))
		gotProfiles = true
	})
	if err != nil {
		logger.Log(
			"component", "service",
			"method", "QueryFunc - windows config profiles",
			"err", err,
		)
		return ""
	}
	if !gotProfiles {
		logger.Log(
			"component", "service",
			"method", "QueryFunc - windows config profiles",
			"info", "host doesn't have profiles to check",
		)
		return ""
	}
	sb.WriteString("</SyncBody>")
	return fmt.Sprintf("SELECT raw_mdm_command_output FROM mdm_bridge WHERE mdm_command_input = '%s';", sb.String())
}

func directIngestWindowsProfiles(
	ctx context.Context,
	logger log.Logger,
	host *fleet.Host,
	ds fleet.Datastore,
	rows []map[string]string,
) error {
	if len(rows) == 0 {
		return nil
	}

	if len(rows) > 1 {
		return ctxerr.Errorf(ctx, "directIngestWindowsProfiles invalid number of rows: %d", len(rows))
	}

	rawResponse := []byte(rows[0]["raw_mdm_command_output"])
	if len(rawResponse) == 0 {
		return ctxerr.Errorf(ctx, "directIngestWindowsProfiles host %s got an empty SyncML response", host.UUID)
	}
	return microsoft_mdm.VerifyHostMDMProfiles(ctx, ds, host, rawResponse)
}
