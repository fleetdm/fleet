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
	"github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/async"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/spf13/cast"
)

// Some machines don't have a correctly set serial number, create a default ignore list
// https://github.com/fleetdm/fleet/issues/25993
var invalidHardwareSerialRegexp = regexp.MustCompile("(?i)(?:default|serial|string)")

type DetailQuery struct {
	// Description is an optional description of the query to be displayed in the
	// Host Vitals documentation https://fleetdm.com/docs/using-fleet/understanding-host-vitals
	Description string
	// Query is the SQL query string.
	Query string
	// QueryFunc is optionally used to dynamically build a query. If false is returned, then the query should be
	// ignored.
	QueryFunc func(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore) (string, bool)
	// Discovery is the SQL query that defines whether the query will run on the host or not.
	// If not set, Fleet makes sure the query will always run.
	Discovery string
	// Platforms is a list of platforms to run the query on. If this value is
	// empty, run on all platforms.
	Platforms []string
	// SoftwareOverrideMatch is a function that can be used to override a software
	// result. The function evaluates a software detail query result row and deletes
	// the result if the function returns true so the result of this detail query can be
	// used instead.
	SoftwareOverrideMatch func(row map[string]string) bool
	// SoftwareProcessResults is a function that can be used to process entries of the main
	// software query and append or modify data using results of additional queries.
	SoftwareProcessResults func(mainSoftwareResults []map[string]string, additionalSoftwareResults []map[string]string) []map[string]string
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
	-- Destination 0.0.0.0/0 or ::/0 (IPv6) is the default route on route tables.
    (r.destination = '0.0.0.0' OR r.destination = '::') AND r.netmask = 0
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
		// Fleet requires the DisplayVersion as well as the UBR (4th part of the version number) to
		// correctly map OS vulnerabilities to hosts. The UBR is not available in the os_version table.
		// The full version number is available in the `kernel_info` table, but there is a Win10 bug
		// which is reporting an incorrect build number (3rd part), so we query the Windows registry for the UBR
		// here instead.  To note, osquery 5.12.0 will have the UBR in the os_version table.

		// display_version is not available in some versions of
		// Windows (Server 2019). By including it using a JOIN it can
		// return no rows and the query will still succeed
		Query: `
		WITH display_version_table AS (
			SELECT data as display_version
			FROM registry
			WHERE path = 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\DisplayVersion'
		),
		ubr_table AS (
			SELECT data AS ubr
			FROM registry
			WHERE path ='HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\UBR'
		)
		SELECT
			os.name,
			COALESCE(d.display_version, '') AS display_version,
			COALESCE(CONCAT((SELECT version FROM os_version), '.', u.ubr), k.version) AS version
		FROM
			os_version os,
			kernel_info k
		LEFT JOIN
			display_version_table d
		LEFT JOIN
			ubr_table u`,
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
		Platforms: append(fleet.HostLinuxOSs, "darwin", "windows"), // not chrome
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
			if invalidHardwareSerialRegexp.Match([]byte(rows[0]["hardware_serial"])) {
				// If the serial number is a default (uninitialize) value, set to empty
				host.HardwareSerial = ""
			} else if rows[0]["hardware_serial"] != "-1" {
				// ignoring the default -1 serial. See: https://github.com/fleetdm/fleet/issues/19789
				host.HardwareSerial = rows[0]["hardware_serial"]
			}
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
		Platforms: append(fleet.HostLinuxOSs, "darwin", "windows"), // not chrome
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

	switch {
	case len(rows) == 0:
		level.Debug(logger).Log("err", "detail_query_network_interface did not find a private IP address")
		return nil
	case len(rows) > 1:
		level.Error(logger).Log("msg", fmt.Sprintf("detail_query_network_interface expected single result, got %d", len(rows)))
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
		// Entries also need to be filtered by their [Intune enrollmentState][1]
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
			-- old enrollments that aren't completely cleaned up may still be aronud
			-- in the registry so we want to make sure we return the one with an actual
			-- discovery URL set if there is one. LENGTH is used here to prefer those
			-- with actual URLs over empty string/null if there are multiple
					ORDER BY LENGTH(e.discovery_service_url) DESC
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
		// This query is used to determine battery health of macOS and Windows hosts
		// based on the cycle count, designed capacity, and max capacity of the battery.
		// The `health` column is ommitted due to a known osquery issue with M1 Macs
		// (https://github.com/fleetdm/fleet/issues/6763) and its absence on Windows.
		Query:            `SELECT serial_number, cycle_count, designed_capacity, max_capacity FROM battery`,
		Platforms:        []string{"windows", "darwin"},
		DirectIngestFunc: directIngestBattery,
		Discovery:        discoveryTable("battery"), // added to Windows in v5.12.1 (https://github.com/osquery/osquery/releases/tag/5.12.1)
	},
	"os_windows": {
		// This query is used to populate the `operating_systems` and `host_operating_system`
		// tables. Separately, the `hosts` table is populated via the `os_version` and
		// `os_version_windows` detail queries above.
		// See above description for the `os_version_windows` detail query.
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
	),
	ubr_table AS (
	SELECT data AS ubr
	FROM registry
	WHERE path ='HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\UBR'
	)
	SELECT
		os.name,
		os.platform,
		os.arch,
		k.version as kernel_version,
		COALESCE(CONCAT((SELECT version FROM os_version), '.', u.ubr), k.version) AS version,
		COALESCE(d.display_version, '') AS display_version
	FROM
		os_version os,
		kernel_info k
	LEFT JOIN
		display_version_table d
	LEFT JOIN
		ubr_table u`,
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
	"certificates_darwin": {
		Query: `
	SELECT
		ca, common_name, subject, issuer,
		key_algorithm, key_strength, key_usage, signing_algorithm,
		not_valid_after, not_valid_before,
		serial, sha1, "system" as source,
		path
	FROM
		certificates
	WHERE
		path = '/Library/Keychains/System.keychain'
	UNION
	SELECT
		ca, common_name, subject, issuer,
		key_algorithm, key_strength, key_usage, signing_algorithm,
		not_valid_after, not_valid_before,
		serial, sha1, "user" as source,
		path
	FROM
		certificates
	WHERE
		path LIKE '/Users/%/Library/Keychains/login.keychain-db';`,
		Platforms:        []string{"darwin"},
		DirectIngestFunc: directIngestHostCertificates,
	},
}

// mdmQueries are used by the Fleet server to compliment certain MDM
// features.
// They are only sent to the device when Fleet's MDM is on and properly
// configured
var mdmQueries = map[string]DetailQuery{
	"mdm_config_profiles_darwin": {
		Query:            `SELECT display_name, identifier, install_date FROM macos_profiles WHERE type = "Configuration";`,
		Platforms:        []string{"darwin"},
		DirectIngestFunc: directIngestMacOSProfiles,
		Discovery:        fmt.Sprintf(`SELECT 1 WHERE EXISTS (%s) AND NOT EXISTS (%s);`, discoveryTable("macos_profiles"), discoveryTable("macos_user_profiles")),
	},
	"mdm_config_profiles_darwin_with_user": {
		QueryFunc:        buildConfigProfilesMacOSQuery,
		Platforms:        []string{"darwin"},
		DirectIngestFunc: directIngestMacOSProfiles,
		Discovery: generateSQLForAllExists(
			discoveryTable("macos_profiles"),
			discoveryTable("macos_user_profiles"),
		),
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
	return fmt.Sprintf("SELECT 1 FROM osquery_registry WHERE active = true AND registry = 'table' AND name = '%s'", tableName)
}

func macOSBundleIDExistsQuery(appName string) string {
	return fmt.Sprintf("SELECT 1 FROM apps WHERE bundle_identifier = '%s' LIMIT 1", appName)
}

// generateSQLForAllExists generates a SQL query that returns
// 1 if all subqueries return 1, otherwise returns no rows.
// subqueries should be a list of SQL queries that return 1+ rows
// if a condition is met, otherwise returns no rows.
func generateSQLForAllExists(subqueries ...string) string {
	if len(subqueries) == 0 {
		return "SELECT 0 LIMIT 0" // Return no rows if no subqueries provided
	}

	// Generate EXISTS clause for each subquery
	var conditions []string
	for _, query := range subqueries {
		// Remove trailing semicolons from the query to ensure subqueries
		// are not terminated early (Issue #19401)
		sanitized := strings.TrimRight(strings.TrimSpace(query), ";")

		condition := fmt.Sprintf("EXISTS (%s)", sanitized)
		conditions = append(conditions, condition)
	}

	// Join all conditions with AND
	fullCondition := strings.Join(conditions, " AND ")

	// Build the final SQL query
	sql := fmt.Sprintf("SELECT 1 WHERE %s", fullCondition)
	return sql
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

// entraIDDetails holds the query and ingestion function for Microsoft "Conditional access" feature.
var entraIDDetails = DetailQuery{
	// The query ingests Entra's Device ID and User Principal Name of the account
	// that logged in to the device (using Company Portal.app with the Platform SSO extension).
	Query:            "SELECT * FROM app_sso_platform WHERE extension_identifier = 'com.microsoft.CompanyPortalMac.ssoextension' AND realm = 'KERBEROS.MICROSOFTONLINE.COM';",
	Discovery:        discoveryTable("app_sso_platform"),
	Platforms:        []string{"darwin"},
	DirectIngestFunc: directIngestEntraIDDetails,
}

var softwareMacOS = DetailQuery{
	// Note that we create the cached_users CTE (the WITH clause) in order to suggest to SQLite
	// that it generates the users once instead of once for each UNIONed query. We use CROSS JOIN to
	// ensure that the nested loops in the query generation are ordered correctly for the _extensions
	// tables that need a uid parameter. CROSS JOIN ensures that SQLite does not reorder the loop
	// nesting, which is important as described in https://youtu.be/hcn3HIcHAAo?t=77.
	//
	// Homebrew package casks are filtered to exclude those that have an associated .app bundle
	// as these are already included in the apps table.  Apps table software includes bundle_identifier
	// which is used in vulnerability scanning.
	Query: withCachedUsers(`WITH cached_users AS (%s)
SELECT
  COALESCE(NULLIF(display_name, ''), NULLIF(bundle_name, ''), NULLIF(bundle_executable, ''), TRIM(name, '.app') ) AS name,
  COALESCE(NULLIF(bundle_short_version, ''), bundle_version) AS version,
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
  '' AS bundle_identifier,
  '' AS extension_id,
  '' AS browser,
  'homebrew_packages' AS source,
  '' AS vendor,
  0 AS last_opened_at,
  path AS installed_path
FROM homebrew_packages
WHERE type = 'formula'
UNION
SELECT
  name AS name,
  version AS version,
  '' AS bundle_identifier,
  '' AS extension_id,
  '' AS browser,
  'homebrew_packages' AS source,
  '' AS vendor,
  0 AS last_opened_at,
  path AS installed_path
FROM homebrew_packages
WHERE type = 'cask'
AND NOT EXISTS (SELECT 1 FROM file WHERE file.path LIKE CONCAT(homebrew_packages.path, '/%%%%') AND file.path LIKE '%%.app%%' LIMIT 1);
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
	Platforms:            append(fleet.HostLinuxOSs, "darwin", "windows"), // not chrome
}

var softwareLinux = DetailQuery{
	Query: withCachedUsers(`WITH cached_users AS (%s)
SELECT
  name AS name,
  version AS version,
  '' AS extension_id,
  '' AS browser,
  'deb_packages' AS source,
  '' AS release,
  '' AS vendor,
  '' AS arch,
  '' AS installed_path
FROM deb_packages
WHERE status LIKE '%% ok installed'
UNION
SELECT
  package AS name,
  version AS version,
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
  identifier AS extension_id,
  'firefox' AS browser,
  'firefox_addons' AS source,
  '' AS release,
  '' AS vendor,
  '' AS arch,
  path AS installed_path
FROM cached_users CROSS JOIN firefox_addons USING (uid);
`),
	Platforms:        fleet.HostLinuxOSs,
	DirectIngestFunc: directIngestSoftware,
}

var softwareWindows = DetailQuery{
	Query: withCachedUsers(`WITH cached_users AS (%s)
SELECT
  name AS name,
  version AS version,
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

// In osquery versions < 5.16.0 use the original python_packages query, as the cross join on
// users is not supported. We're *not* using VERSION_COMPARE() here and below because that function
// doesn't exist in osquery version < 5.11.0, causing discovery queries to fail for those versions.
// See #27126 for more information.
var softwarePythonPackages = DetailQuery{
	Description: "Prior to osquery version 5.16.0, the python_packages table did not search user directories.",
	Query: `
		SELECT
		  name AS name,
		  version AS version,
		  '' AS extension_id,
		  '' AS browser,
		  'python_packages' AS source,
		  '' AS vendor,
		  path AS installed_path
		FROM python_packages
	`,
	Platforms: append(fleet.HostLinuxOSs, "darwin", "windows"),
	Discovery: `SELECT 1 FROM (
    SELECT
    CAST(SUBSTR(version, 1, INSTR(version, '.') - 1) AS INTEGER) major,
    CAST(SUBSTR(version, INSTR(version, '.') + 1, INSTR(SUBSTR(version, INSTR(version, '.') + 1), '.') - 1) AS INTEGER) minor from osquery_info
) AS version_parts WHERE major < 5 OR (major = 5 AND minor < 16)`,
}

// In osquery versions >= 5.16.0 the python_packages table was modified to allow for a
// cross join on users so that user directories could be searched for python packages
var softwarePythonPackagesWithUsersDir = DetailQuery{
	Description: "As of osquery version 5.16.0, the python_packages table searches user directories with support from a cross join on users. See https://fleetdm.com/guides/osquery-consider-joining-against-the-users-table.",
	Query: withCachedUsers(`WITH cached_users AS (%s)
		SELECT
		  name AS name,
		  version AS version,
		  '' AS extension_id,
		  '' AS browser,
		  'python_packages' AS source,
		  '' AS vendor,
		  path AS installed_path
		FROM cached_users CROSS JOIN python_packages USING (uid)
	`),
	Platforms: append(fleet.HostLinuxOSs, "darwin", "windows"),
	Discovery: `SELECT 1 FROM (
    SELECT
    CAST(substr(version, 1, instr(version, '.') - 1) AS INTEGER) major,
    CAST(substr(version, instr(version, '.') + 1, instr(substr(version, instr(version, '.') + 1), '.') - 1) AS INTEGER) minor from osquery_info
) AS version_parts WHERE major > 5 OR (major = 5 AND minor >= 16)`,
}

var softwareChrome = DetailQuery{
	Query: `SELECT
  name AS name,
  version AS version,
  identifier AS extension_id,
  browser_type AS browser,
  'chrome_extensions' AS source,
  '' AS vendor,
  '' AS installed_path
FROM chrome_extensions`,
	Platforms:        []string{"chrome"},
	DirectIngestFunc: directIngestSoftware,
}

// SoftwareOverrideQueries are used to override software detail query results.  These DetailQueries
// must include a `SoftwareOverrideMatch` function that returns true if the software row should be
// overridden with the results of `Query`.
// Software queries expect specific columns to be present.  Reference the
// software_{macos|windows|linux} queries for the expected columns.
var SoftwareOverrideQueries = map[string]DetailQuery{
	// macos_firefox differentiates between Firefox and Firefox ESR by checking the RemotingName value in the
	// application.ini file. If the RemotingName is 'firefox-esr', the name is set to 'Firefox ESR.app'.
	//
	// NOTE(lucas): This could be re-written to use SoftwareProcessResults so that this query doesn't need to match
	// the columns of the main softwareMacOS query.
	"macos_firefox": {
		Description: "A software override query[^1] to differentiate between Firefox and Firefox ESR on macOS. Requires `fleetd`",
		Query: `
			WITH app_paths AS (
				SELECT path
				FROM apps
				WHERE bundle_identifier = 'org.mozilla.firefox'
			),
			remoting_name AS (
				SELECT value, path
				FROM parse_ini
				WHERE key = 'RemotingName'
				AND path IN (SELECT CONCAT(path, '/Contents/Resources/application.ini') FROM app_paths)
			)
			SELECT
				CASE
					WHEN remoting_name.value = 'firefox-esr' THEN 'Firefox ESR.app'
					ELSE 'Firefox.app'
				END AS name,
				COALESCE(NULLIF(apps.bundle_short_version, ''), apps.bundle_version) AS version,
				apps.bundle_identifier AS bundle_identifier,
				'' AS extension_id,
				'' AS browser,
				'apps' AS source,
				'' AS vendor,
				apps.last_opened_time AS last_opened_at,
				apps.path AS installed_path
			FROM apps
			LEFT JOIN remoting_name ON apps.path = REPLACE(remoting_name.path, '/Contents/Resources/application.ini', '')
			WHERE apps.bundle_identifier = 'org.mozilla.firefox'`,
		Platforms: []string{"darwin"},
		Discovery: generateSQLForAllExists(
			macOSBundleIDExistsQuery("org.mozilla.firefox"),
			discoveryTable("parse_ini"),
		),
		SoftwareOverrideMatch: func(row map[string]string) bool {
			return row["bundle_identifier"] == "org.mozilla.firefox"
		},
	},
	// macos_codesign collects code signature information of apps on a separate query for two reasons:
	//   - codesign is a fleetd table (not part of osquery core).
	//   - Avoid growing the main `software_macos` query
	//     (having big queries can cause performance issues or be denylisted).
	"macos_codesign": {
		Query: `
		SELECT c.*
		FROM apps a
		JOIN codesign c ON a.path = c.path
	`,
		Description: "A software override query[^1] to append codesign information to macOS software entries. Requires `fleetd`",
		Platforms:   []string{"darwin"},
		Discovery:   discoveryTable("codesign"),
		SoftwareProcessResults: func(mainSoftwareResults, codesignResults []map[string]string) []map[string]string {
			type codesignResultRow struct {
				teamIdentifier string
				cdhashSHA256   string
			}

			codesignInformation := make(map[string]codesignResultRow) // path -> team_identifier
			for _, codesignResult := range codesignResults {
				var cdhashSha256 string
				if hash, ok := codesignResult["cdhash_sha256"]; ok {
					cdhashSha256 = hash
				}

				codesignInformation[codesignResult["path"]] = codesignResultRow{
					teamIdentifier: codesignResult["team_identifier"],
					cdhashSHA256:   cdhashSha256,
				}
			}
			if len(codesignInformation) == 0 {
				return mainSoftwareResults
			}

			for _, result := range mainSoftwareResults {
				codesignInfo, ok := codesignInformation[result["installed_path"]]
				if !ok {
					// No codesign information for this application.
					continue
				}
				result["team_identifier"] = codesignInfo.teamIdentifier
				result["cdhash_sha256"] = codesignInfo.cdhashSHA256
			}

			return mainSoftwareResults
		},
	},
	// windows_last_opened_at collects last opened at information from prefetch files on Windows
	// hosts. Joining this within the main software query is not performant enough to do on the
	// device (resulted in denylisted queries during testing), so we do it on the server instead.
	"windows_last_opened_at": {
		Query: `
		SELECT
		  MAX(last_run_time) AS last_opened_at,
		  REGEX_MATCH(accessed_files, "VOLUME[^\\]+([^,]+"||filename||")", 1) AS executable_path
		FROM prefetch
		GROUP BY executable_path
	`,
		Description: "A software override query[^1] to append last_opened_at information to Windows software entries.",
		Platforms:   []string{"windows"},
		SoftwareProcessResults: func(mainSoftwareResults, prefetchResults []map[string]string) []map[string]string {
			if len(prefetchResults) == 0 {
				return mainSoftwareResults
			}

			skipInstallPaths := map[string]bool{
				"":                      true,
				"\\":                    true,
				"\\windows\\system32\\": true,
			}

			for _, result := range mainSoftwareResults {
				if result["source"] != "programs" {
					// Only override last_opened_at for software from the programs source.
					continue
				}

				installPath := strings.ToLower(result["installed_path"])
				// Remove the drive letter from the path
				volume, path, found := strings.Cut(installPath, ":")
				if !found || len(volume) != 1 || len(path) == 0 {
					continue
				}

				// Skip some install paths where binaries live but can't be well correlated to programs
				if skipInstallPaths[path] {
					fmt.Println("Skipping install path:", path)
					continue
				}

				for _, prefetchResult := range prefetchResults {
					prefetchPath := strings.ToLower(prefetchResult["executable_path"])
					if strings.Contains(prefetchPath, path) {
						fmt.Println("found prefetch for install path:", path, "->", prefetchPath, prefetchResult["last_opened_at"])
						// Some programs will have multiple prefetch entries matching their install
						// path, so we compare the last_opened_at values and keep the more recent.
						result["last_opened_at"] = maxString(result["last_opened_at"], prefetchResult["last_opened_at"])
					}
				}
				fmt.Println("Final last_opened_at for", result["installed_path"], ":", result["last_opened_at"])
			}

			return mainSoftwareResults
		},
	},
}

// Convert the strings to integers and return the larger one.
func maxString(a, b string) string {
	intA, _ := strconv.Atoi(a)
	intB, _ := strconv.Atoi(b)
	if intA > intB {
		return a
	}
	return b
}

var usersQuery = DetailQuery{
	// Note we use the cached_groups CTE (`WITH` clause) here to suggest to SQLite that it generate
	// the `groups` table only once. Without doing this, on some Windows systems (Domain Controllers)
	// with many user accounts and groups, this query could be very expensive as the `groups` table
	// was generated once for each user.
	Query:            usersQueryStr,
	Platforms:        append(fleet.HostLinuxOSs, "darwin", "windows"),
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
		KernelVersion: rows[0]["version"],
		Platform:      rows[0]["platform"],
		Version:       rows[0]["version"],
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

// directIngestBattery ingests battery data from a host on a Windows or macOS platform
// and calculates the battery health based on cycle count and capacity.
// Due to a known osquery issue with M1 Macs (https://github.com/fleetdm/fleet/issues/6763)
// and the ommission of the `health` column on Windows, we are not leveraging the `health`
// column in the query and instead aligning the definition of battery health between
// macOS and Windows.
func directIngestBattery(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
	mapping := make([]*fleet.HostBattery, 0, len(rows))

	for _, row := range rows {
		health, cycleCount, err := generateBatteryHealth(row, logger)
		if err != nil {
			level.Error(logger).Log("op", "directIngestBattery", "hostID", host.ID, "err", err)
		}

		mapping = append(mapping, &fleet.HostBattery{
			HostID:       host.ID,
			SerialNumber: row["serial_number"],
			CycleCount:   cycleCount,
			Health:       health,
		})
	}

	return ds.ReplaceHostBatteries(ctx, host.ID, mapping)
}

const (
	batteryStatusUnknown      = "Unknown"
	batteryStatusDegraded     = "Service recommended"
	batteryStatusGood         = "Normal"
	batteryDegradedThreshold  = 80
	batteryDegradedCycleCount = 1000
)

// generateBatteryHealth calculates the battery health based on the cycle count and capacity.
func generateBatteryHealth(row map[string]string, logger log.Logger) (string, int, error) {
	designedCapacity := row["designed_capacity"]
	maxCapacity := row["max_capacity"]
	cycleCount := row["cycle_count"]

	count, err := strconv.Atoi(EmptyToZero(cycleCount))
	if err != nil {
		level.Error(logger).Log("op", "generateBatteryHealth", "err", err)
		// If we can't parse the cycle count, we'll assume it's 0
		// and continue with the rest of the battery health check.
		count = 0
	}

	if count >= batteryDegradedCycleCount {
		return batteryStatusDegraded, count, nil
	}

	if designedCapacity == "" || maxCapacity == "" {
		return batteryStatusUnknown, count, fmt.Errorf("missing battery capacity values, designed: %s, max: %s", designedCapacity, maxCapacity)
	}

	designed, err := strconv.ParseInt(designedCapacity, 10, 64)
	if err != nil {
		return batteryStatusUnknown, count, fmt.Errorf("failed to parse designed capacity: %s", designedCapacity)
	}

	maxCapInt, err := strconv.ParseInt(maxCapacity, 10, 64)
	if err != nil {
		return batteryStatusUnknown, count, fmt.Errorf("failed to parse max capacity: %s", maxCapacity)
	}

	health := float64(maxCapInt) / float64(designed) * 100

	if health < batteryDegradedThreshold {
		return batteryStatusDegraded, count, nil
	}

	return batteryStatusGood, count, nil
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

func directIngestEntraIDDetails(
	ctx context.Context,
	logger log.Logger,
	host *fleet.Host,
	ds fleet.Datastore,
	rows []map[string]string,
) error {
	if len(rows) == 0 {
		// Device maybe hasn't logged in to Entra ID yet.
		return nil
	}
	row := rows[0]

	deviceID := row["device_id"]
	if deviceID == "" {
		return ctxerr.New(ctx, "empty Entra ID device_id")
	}
	userPrincipalName := row["user_principal_name"]
	// userPrincipalName can be empty on macOS workstations with e.g. two accounts:
	// one logged in to Entra and the other one not logged in.
	// While the second one is logged in, it would report the same Device ID but empty user principal name.

	if err := ds.CreateHostConditionalAccessStatus(ctx, host.ID, deviceID, userPrincipalName); err != nil {
		return ctxerr.Wrap(ctx, err, "failed to create host conditional access status")
	}
	return nil
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

		if shouldRemoveSoftware(host, s) {
			continue
		}

		software = append(software, *s)

		installedPath := strings.TrimSpace(row["installed_path"])
		if installedPath != "" &&
			// NOTE: osquery is sometimes incorrectly returning the value "null" for some install paths.
			// Thus, we explicitly ignore such value here.
			strings.ToLower(installedPath) != "null" {
			truncateString := func(str string, length int) string {
				runes := []rune(str)
				if len(runes) > length {
					return string(runes[:length])
				}
				return str
			}
			teamIdentifier := truncateString(row["team_identifier"], fleet.SoftwareTeamIdentifierMaxLength)
			var cdhashSHA256 string
			if hash, ok := row["cdhash_sha256"]; ok {
				cdhashSHA256 = hash
			}
			key := fmt.Sprintf(
				"%s%s%s%s%s%s%s",
				installedPath, fleet.SoftwareFieldSeparator, teamIdentifier, fleet.SoftwareFieldSeparator, cdhashSHA256, fleet.SoftwareFieldSeparator, s.ToUniqueStr(),
			)
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
		if row["uid"] == "" {
			// Under certain circumstances, broken users can come back with empty UIDs. We don't want them
			continue
		}
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
			Uid:       uint(uid), // nolint:gosec // dismiss G115
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

	if host.RefetchCriticalQueriesUntil != nil {
		level.Debug(logger).Log("msg", "ingesting Windows mdm data during refetch critical queries window", "host_id", host.ID,
			"data", fmt.Sprintf("%+v", rows))
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

	mdmSolutionName := deduceMDMNameWindows(data)
	if !enrolled && mdmSolutionName != fleet.WellKnownMDMFleet && host.RefetchCriticalQueriesUntil != nil {
		// the host was unenrolled from a non-Fleet MDM solution, and the refetch
		// critical queries timestamp was set, so clear it.
		host.RefetchCriticalQueriesUntil = nil
	}

	return ds.SetOrUpdateMDMData(ctx,
		host.ID,
		isServer,
		enrolled,
		serverURL,
		automatic,
		mdmSolutionName,
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
	return ds.SetOrUpdateHostDiskEncryptionKey(ctx, host, base64Key, "", decryptable)
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

	return ds.SetOrUpdateHostDiskEncryptionKey(ctx, host, base64Key, "", decryptable)
}

func buildConfigProfilesMacOSQuery(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore) (string, bool) {
	query := `
		SELECT
			display_name,
			identifier,
			install_date
		FROM
			macos_profiles
		WHERE
			type = "Configuration"`

	username, err := ds.GetNanoMDMUserEnrollmentUsername(ctx, host.UUID)
	if err != nil {
		logger.Log("component", "service",
			"method", "QueryFunc - macos config profiles", "err", err)
		return "", false
	}

	if username != "" {
		query += fmt.Sprintf(`
			UNION

			SELECT
				display_name,
				identifier,
				install_date
			FROM
				macos_user_profiles
			WHERE
				username = %q AND
				type = "Configuration"`, username)
	}
	return query, true
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

var luksVerifyQuery = DetailQuery{
	Platforms: fleet.HostLinuxOSs,
	Discovery: fmt.Sprintf(
		`SELECT 1 WHERE EXISTS (%s) AND EXISTS (%s);`,
		discoveryTable("lsblk"),
		discoveryTable("cryptsetup_luks_salt"),
	),
	QueryFunc: func(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore) (string, bool) {
		if host.OrbitNodeKey == nil || *host.OrbitNodeKey == "" || !host.IsLUKSSupported() {
			return "", false
		}

		if _, err := ds.GetHostDiskEncryptionKey(ctx, host.ID); err != nil {
			if fleet.IsNotFound(err) {
				return "", false
			}
		}

		// Returns the key_slot and salt of the LUKS block device where '/' is mounted.
		query := `
		WITH RECURSIVE
		devices AS (
			SELECT
				MAX(CASE WHEN key = 'path' THEN value ELSE NULL END) AS path,
				MAX(CASE WHEN key = 'kname' THEN value ELSE NULL END) AS kname,
				MAX(CASE WHEN key = 'pkname' THEN value ELSE NULL END) AS pkname,
				MAX(CASE WHEN key = 'fstype' THEN value ELSE NULL END) AS fstype,
				MAX(CASE WHEN key = 'mountpoint' THEN value ELSE NULL END) as mountpoint
			FROM lsblk
			GROUP BY parent
		HAVING path <> '' AND fstype <> ''
		),
		root_mount AS (
			SELECT
				path,
				kname,
				-- if '/' is mounted in a LUKS FS, then we don't need to transverse the tree
				CASE WHEN fstype = 'crypto_LUKS' THEN NULL ELSE pkname END as pkname,
				fstype
			FROM devices
			WHERE mountpoint = '/'
		),
		luks_h(path, kname, pkname, fstype) AS (
			SELECT
				rtm.path,
				rtm.kname,
				rtm.pkname,
				rtm.fstype
			FROM root_mount rtm
			UNION
		   SELECT
				dv.path,
				dv.kname,
				dv.pkname,
				dv.fstype
		   FROM devices dv
		   JOIN luks_h ON dv.kname=luks_h.pkname
		)
		SELECT salt, key_slot
		FROM cryptsetup_luks_salt
		WHERE device = (SELECT path FROM luks_h WHERE fstype = 'crypto_LUKS' LIMIT 1)`
		return query, true
	},
}

// We need to define the ingest function inline like this because we need access to the server private key
var luksVerifyQueryIngester = func(decrypter func(string) (string, error)) func(
	ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
	return func(
		ctx context.Context,
		logger log.Logger,
		host *fleet.Host,
		ds fleet.Datastore,
		rows []map[string]string,
	) error {
		if len(rows) == 0 || host == nil || !host.IsLUKSSupported() {
			return nil
		}

		dek, err := ds.GetHostDiskEncryptionKey(ctx, host.ID)
		if err != nil {
			if fleet.IsNotFound(err) {
				level.Error(logger).Log(
					"component", "service",
					"method", "luksVerifyQueryIngester",
					"msg", "unexpected missing LUKS2 disk encryption key",
					"err", err,
				)
				return nil
			}
			level.Error(logger).Log(
				"component", "service",
				"method", "luksVerifyQueryIngester",
				"msg", "unexpected error",
				"err", err,
			)
			return err
		}
		if dek == nil || dek.Base64EncryptedSalt == "" || dek.KeySlot == nil {
			return nil
		}

		storedSalt, err := decrypter(dek.Base64EncryptedSalt)
		if err != nil {
			level.Debug(logger).Log(
				"component", "service",
				"method", "luksVerifyQueryIngester",
				"host", host.ID,
				"err", err,
			)
			return err
		}
		storedKeySlot := fmt.Sprintf("%d", *dek.KeySlot)

		var entryFound bool
		for _, row := range rows {
			hostSalt, okSalt := row["salt"]
			hostKeySlot, okKeySlot := row["key_slot"]

			if !okSalt || !okKeySlot {
				level.Error(logger).Log(
					"component", "service",
					"method", "luksVerifyQueryIngester",
					"host", host.ID,
					"err", "luks_verify expected some salt and a key_slot",
				)
				continue
			}
			if hostSalt == storedSalt && hostKeySlot == storedKeySlot {
				entryFound = true
				break
			}
		}
		if !entryFound {
			level.Info(logger).Log(
				"component", "service",
				"method", "luksVerifyQueryIngester",
				"host", host.ID,
				"msg", "LUKS key do not match, deleting",
			)
			return ds.DeleteLUKSData(ctx, host.ID, *dek.KeySlot)
		}
		return nil
	}
}

//go:generate go run gen_queries_doc.go "../../../docs/Contributing/product-groups/orchestration/understanding-host-vitals.md"

type Integrations struct {
	ConditionalAccessMicrosoft bool
}

func GetDetailQueries(
	ctx context.Context,
	fleetConfig config.FleetConfig,
	appConfig *fleet.AppConfig,
	features *fleet.Features,
	integrations Integrations,
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
		generatedMap["software_python_packages"] = softwarePythonPackages
		generatedMap["software_python_packages_with_users_dir"] = softwarePythonPackagesWithUsersDir
		generatedMap["software_vscode_extensions"] = softwareVSCodeExtensions

		for key, query := range SoftwareOverrideQueries {
			generatedMap["software_"+key] = query
		}
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

	if integrations.ConditionalAccessMicrosoft {
		generatedMap["conditional_access_microsoft_device_id"] = entraIDDetails
	}

	if appConfig != nil && appConfig.MDM.EnableDiskEncryption.Value {
		luksVerifyQuery.DirectIngestFunc = luksVerifyQueryIngester(func(privateKey string) func(string) (string, error) {
			return func(encrypted string) (string, error) {
				return mdm.DecodeAndDecrypt(encrypted, privateKey)
			}
		}(fleetConfig.Server.PrivateKey))
		generatedMap["luks_verify"] = luksVerifyQuery
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
) (string, bool) {
	var sb strings.Builder
	sb.WriteString("<SyncBody>")
	gotProfiles := false
	err := microsoft_mdm.LoopOverExpectedHostProfiles(ctx, ds, host, func(profile *fleet.ExpectedMDMProfile, hash, locURI, data string) {
		// Per the [docs][1], to `<Get>` configurations you must
		// replace `/Policy/Config/` with `Policy/Result/`
		// [1]: https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-configuration-service-provider
		// Note that the trailing slash is important because there is also /Policy/ConfigOperations
		// for ADMX policy ingestion which does not have a corresponding Result URI.
		locURI = strings.Replace(locURI, "/Policy/Config/", "/Policy/Result/", 1)
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
		return "", false
	}
	if !gotProfiles {
		level.Debug(logger).Log(
			"component", "service",
			"method", "QueryFunc - windows config profiles",
			"msg", "host doesn't have profiles to check",
			"host_id", host.ID,
		)
		return "", false
	}
	sb.WriteString("</SyncBody>")
	return fmt.Sprintf("SELECT raw_mdm_command_output FROM mdm_bridge WHERE mdm_command_input = '%s';", sb.String()), true
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
	return microsoft_mdm.VerifyHostMDMProfiles(ctx, logger, ds, host, rawResponse)
}

var rxExtractUsernameFromHostCertPath = regexp.MustCompile(`^/Users/([^/]+)/Library/Keychains/login\.keychain\-db$`)

func directIngestHostCertificates(
	ctx context.Context,
	logger log.Logger,
	host *fleet.Host,
	ds fleet.Datastore,
	rows []map[string]string,
) error {
	if len(rows) == 0 {
		// if there are no results, it probably may indicate a problem so we log it
		level.Debug(logger).Log("component", "service", "method", "directIngestHostCertificates", "msg", "no rows returned", "host_id", host.ID)
		return nil
	}

	certs := make([]*fleet.HostCertificateRecord, 0, len(rows))
	for _, row := range rows {
		csum, err := hex.DecodeString(row["sha1"])
		if err != nil {
			level.Error(logger).Log("component", "service", "method", "directIngestHostCertificates", "msg", "decoding sha1", "err", err)
			continue
		}
		subject, err := fleet.ExtractDetailsFromOsqueryDistinguishedName(row["subject"])
		if err != nil {
			level.Error(logger).Log("component", "service", "method", "directIngestHostCertificates", "msg", "extracting subject details", "err", err)
			continue
		}
		issuer, err := fleet.ExtractDetailsFromOsqueryDistinguishedName(row["issuer"])
		if err != nil {
			level.Error(logger).Log("component", "service", "method", "directIngestHostCertificates", "msg", "extracting issuer details", "err", err)
			continue
		}
		source := fleet.HostCertificateSource(row["source"])
		if !source.IsValid() {
			// should never happen as the source is hard-coded in the query
			level.Error(logger).Log("component", "service", "method", "directIngestHostCertificates", "msg", "invalid certificate source", "err", fmt.Errorf("invalid source %s", row["source"]))
			continue
		}

		var username string
		if source == fleet.UserHostCertificate {
			// extract the username from the keychain path
			matches := rxExtractUsernameFromHostCertPath.FindStringSubmatch(row["path"])
			if len(matches) > 1 {
				username = matches[1]
			} else {
				// if we cannot extract the username, we log it but continue
				level.Error(logger).Log("component", "service", "method", "directIngestHostCertificates", "msg", "could not extract username from path", "err", fmt.Errorf("no username found in path %q", row["path"]))
			}
		}

		certs = append(certs, &fleet.HostCertificateRecord{
			HostID:                    host.ID,
			SHA1Sum:                   csum,
			NotValidAfter:             time.Unix(cast.ToInt64(row["not_valid_after"]), 0).UTC(),
			NotValidBefore:            time.Unix(cast.ToInt64(row["not_valid_before"]), 0).UTC(),
			CertificateAuthority:      cast.ToBool(row["ca"]),
			CommonName:                row["common_name"],
			KeyAlgorithm:              row["key_algorithm"],
			KeyStrength:               cast.ToInt(row["key_strength"]),
			KeyUsage:                  row["key_usage"],
			Serial:                    row["serial"],
			SigningAlgorithm:          row["signing_algorithm"],
			SubjectCountry:            subject.Country,
			SubjectOrganizationalUnit: subject.OrganizationalUnit,
			SubjectOrganization:       subject.Organization,
			SubjectCommonName:         subject.CommonName,
			IssuerCountry:             issuer.Country,
			IssuerOrganizationalUnit:  issuer.OrganizationalUnit,
			IssuerOrganization:        issuer.Organization,
			IssuerCommonName:          issuer.CommonName,
			Source:                    source,
			Username:                  username,
		})
	}

	if len(certs) == 0 {
		// don't overwrite existing certs if we were unable to parse any new ones
		return nil
	}

	return ds.UpdateHostCertificates(ctx, host.ID, host.UUID, certs)
}
