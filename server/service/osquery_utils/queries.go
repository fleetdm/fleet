package osquery_utils

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/spf13/cast"
)

type DetailQuery struct {
	Query string
	// Platforms is a list of platforms to run the query on. If this value is
	// empty, run on all platforms.
	Platforms []string
	// IngestFunc translates a query result into an update to the host struct,
	// around data that lives on the hosts table.
	IngestFunc func(logger log.Logger, host *fleet.Host, rows []map[string]string) error
	// DirectIngestFunc gathers results from a query and directly works with the datastore to
	// persist them. This is usually used for host data that is stored in a separate table.
	DirectIngestFunc func(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string, failed bool) error
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

// detailQueries defines the detail queries that should be run on the host, as
// well as how the results of those queries should be ingested into the
// fleet.Host data model. This map should not be modified at runtime.
var detailQueries = map[string]DetailQuery{
	"network_interface": {
		Query: `select address, mac
                        from interface_details id join interface_addresses ia
                               on ia.interface = id.interface where length(mac) > 0
                               order by (ibytes + obytes) desc`,
		IngestFunc: func(logger log.Logger, host *fleet.Host, rows []map[string]string) (err error) {
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
			return nil
		},
	},
	"os_version": {
		Query: "select * from os_version limit 1",
		IngestFunc: func(logger log.Logger, host *fleet.Host, rows []map[string]string) error {
			if len(rows) != 1 {
				logger.Log("component", "service", "method", "IngestFunc", "err",
					fmt.Sprintf("detail_query_os_version expected single result got %d", len(rows)))
				return nil
			}

			host.OSVersion = fmt.Sprintf(
				"%s %s.%s.%s",
				rows[0]["name"],
				rows[0]["major"],
				rows[0]["minor"],
				rows[0]["patch"],
			)
			host.OSVersion = strings.Trim(host.OSVersion, ".")

			if build, ok := rows[0]["build"]; ok {
				host.Build = build
			}

			host.Platform = rows[0]["platform"]
			host.PlatformLike = rows[0]["platform_like"]
			host.CodeName = rows[0]["code_name"]

			// On centos6 there is an osquery bug that leaves
			// platform empty. Here we workaround.
			if host.Platform == "" &&
				strings.Contains(strings.ToLower(rows[0]["name"]), "centos") {
				host.Platform = "centos"
			}

			return nil
		},
	},
	"osquery_flags": {
		// Collect the interval info (used for online status
		// calculation) from the osquery flags. We typically control
		// distributed_interval (but it's not required), and typically
		// do not control config_tls_refresh.
		Query: `select name, value from osquery_flags where name in ("distributed_interval", "config_tls_refresh", "config_refresh", "logger_tls_period")`,
		IngestFunc: func(logger log.Logger, host *fleet.Host, rows []map[string]string) error {
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
		IngestFunc: func(logger log.Logger, host *fleet.Host, rows []map[string]string) error {
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
		IngestFunc: func(logger log.Logger, host *fleet.Host, rows []map[string]string) error {
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
		IngestFunc: func(logger log.Logger, host *fleet.Host, rows []map[string]string) error {
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
	"scheduled_query_stats": {
		Query: `
			SELECT *,
				(SELECT value from osquery_flags where name = 'pack_delimiter') AS delimiter
			FROM osquery_schedule`,
		DirectIngestFunc: directIngestScheduledQueryStats,
	},
	"disk_space_unix": {
		Query: `
SELECT (blocks_available * 100 / blocks) AS percent_disk_space_available,
       round((blocks_available * blocks_size *10e-10),2) AS gigs_disk_space_available
FROM mounts WHERE path = '/' LIMIT 1;`,
		Platforms:  append(fleet.HostLinuxOSs, "darwin"),
		IngestFunc: ingestDiskSpace,
	},
	"disk_space_windows": {
		Query: `
SELECT ROUND((sum(free_space) * 100 * 10e-10) / (sum(size) * 10e-10)) AS percent_disk_space_available,
       ROUND(sum(free_space) * 10e-10) AS gigs_disk_space_available
FROM logical_drives WHERE file_system = 'NTFS' LIMIT 1;`,
		Platforms:  []string{"windows"},
		IngestFunc: ingestDiskSpace,
	},
	"mdm": {
		Query:            `select enrolled, server_url, installed_from_dep from mdm;`,
		DirectIngestFunc: directIngestMDM,
	},
	"munki_info": {
		Query:            `select version from munki_info;`,
		DirectIngestFunc: directIngestMunkiInfo,
	},
	"google_chrome_profiles": {
		Query:            `SELECT email FROM google_chrome_profiles WHERE NOT ephemeral`,
		DirectIngestFunc: directIngestChromeProfiles,
	},
}

var softwareMacOS = DetailQuery{
	// Note that we create the cached_users CTE (the WITH clause) in order to suggest to SQLite
	// that it generates the users once instead of once for each UNIONed query. We use CROSS JOIN to
	// ensure that the nested loops in the query generation are ordered correctly for the _extensions
	// tables that need a uid parameter. CROSS JOIN ensures that SQLite does not reorder the loop
	// nesting, which is important as described in https://youtu.be/hcn3HIcHAAo?t=77.
	Query: `
WITH cached_users AS (SELECT * FROM users)
SELECT
  name AS name,
  bundle_short_version AS version,
  'Application (macOS)' AS type,
  bundle_identifier AS bundle_identifier,
  'apps' AS source
FROM apps
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Python)' AS type,
  '' AS bundle_identifier,
  'python_packages' AS source
FROM python_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Chrome)' AS type,
  '' AS bundle_identifier,
  'chrome_extensions' AS source
FROM cached_users CROSS JOIN chrome_extensions USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Firefox)' AS type,
  '' AS bundle_identifier,
  'firefox_addons' AS source
FROM cached_users CROSS JOIN firefox_addons USING (uid)
UNION
SELECT
  name As name,
  version AS version,
  'Browser plugin (Safari)' AS type,
  '' AS bundle_identifier,
  'safari_extensions' AS source
FROM cached_users CROSS JOIN safari_extensions USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Atom)' AS type,
  '' AS bundle_identifier,
  'atom_packages' AS source
FROM cached_users CROSS JOIN atom_packages USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Homebrew)' AS type,
  '' AS bundle_identifier,
  'homebrew_packages' AS source
FROM homebrew_packages;
`,
	Platforms:        []string{"darwin"},
	DirectIngestFunc: directIngestSoftware,
}

var softwareLinux = DetailQuery{
	Query: `
WITH cached_users AS (SELECT * FROM users)
SELECT
  name AS name,
  version AS version,
  'Package (deb)' AS type,
  'deb_packages' AS source
FROM deb_packages
UNION
SELECT
  package AS name,
  version AS version,
  'Package (Portage)' AS type,
  'portage_packages' AS source
FROM portage_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Package (RPM)' AS type,
  'rpm_packages' AS source
FROM rpm_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Package (NPM)' AS type,
  'npm_packages' AS source
FROM npm_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Chrome)' AS type,
  'chrome_extensions' AS source
FROM cached_users CROSS JOIN chrome_extensions USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Firefox)' AS type,
  'firefox_addons' AS source
FROM cached_users CROSS JOIN firefox_addons USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Atom)' AS type,
  'atom_packages' AS source
FROM users CROSS JOIN atom_packages USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Python)' AS type,
  'python_packages' AS source
FROM python_packages;
`,
	Platforms:        fleet.HostLinuxOSs,
	DirectIngestFunc: directIngestSoftware,
}

var softwareWindows = DetailQuery{
	Query: `
WITH cached_users AS (SELECT * FROM users)
SELECT
  name AS name,
  version AS version,
  'Program (Windows)' AS type,
  'programs' AS source
FROM programs
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Python)' AS type,
  'python_packages' AS source
FROM python_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (IE)' AS type,
  'ie_extensions' AS source
FROM ie_extensions
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Chrome)' AS type,
  'chrome_extensions' AS source
FROM cached_users CROSS JOIN chrome_extensions USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Firefox)' AS type,
  'firefox_addons' AS source
FROM cached_users CROSS JOIN firefox_addons USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Chocolatey)' AS type,
  'chocolatey_packages' AS source
FROM chocolatey_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Atom)' AS type,
  'atom_packages' AS source
FROM cached_users CROSS JOIN atom_packages USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Python)' AS type,
  'python_packages' AS source
FROM python_packages;
`,
	Platforms:        []string{"windows"},
	DirectIngestFunc: directIngestSoftware,
}

var usersQuery = DetailQuery{
	// Note we use the cached_groups CTE (`WITH` clause) here to suggest to SQLite that it generate
	// the `groups` table only once. Without doing this, on some Windows systems (Domain Controllers)
	// with many user accounts and groups, this query could be very expensive as the `groups` table
	// was generated once for each user.
	Query: `
WITH cached_groups AS (select * from groups)
SELECT uid, username, type, groupname, shell
FROM users LEFT JOIN cached_groups USING (gid)
WHERE type <> 'special' AND shell NOT LIKE '%/false' AND shell NOT LIKE '%/nologin' AND shell NOT LIKE '%/shutdown' AND shell NOT LIKE '%/halt' AND username NOT LIKE '%$' AND username NOT LIKE '\_%' ESCAPE '\' AND NOT (username = 'sync' AND shell ='/bin/sync')`,
	DirectIngestFunc: directIngestUsers,
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

func directIngestScheduledQueryStats(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string, failed bool) error {
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
	if err := ds.SaveHostPackStats(ctx, host.ID, packStats); err != nil {
		return ctxerr.Wrap(ctx, err, "save host pack stats")
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
		s := fleet.Software{
			Name:             name,
			Version:          version,
			Source:           source,
			BundleIdentifier: bundleIdentifier,
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
	if err := ds.SaveHostUsers(ctx, host.ID, users); err != nil {
		return ctxerr.Wrap(ctx, err, "update host users")
	}
	return nil
}

func ingestDiskSpace(logger log.Logger, host *fleet.Host, rows []map[string]string) error {
	if len(rows) != 1 {
		logger.Log("component", "service", "method", "ingestDiskSpace", "err",
			fmt.Sprintf("detail_query_disk_space expected single result got %d", len(rows)))
		return nil
	}

	var err error
	host.GigsDiskSpaceAvailable, err = strconv.ParseFloat(EmptyToZero(rows[0]["gigs_disk_space_available"]), 64)
	if err != nil {
		return err
	}
	host.PercentDiskSpaceAvailable, err = strconv.ParseFloat(EmptyToZero(rows[0]["percent_disk_space_available"]), 64)
	if err != nil {
		return err
	}
	return nil
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
	if !enrolled {
		// A row with enrolled=false and all other columns empty is a host with the osquery
		// MDM table extensions installed (e.g. Orbit) but MDM unconfigured/disabled.
		return nil
	}
	installedFromDep, err := strconv.ParseBool(rows[0]["installed_from_dep"])
	if err != nil {
		return ctxerr.Wrap(ctx, err, "parsing installed_from_dep")
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

	return ds.SetOrUpdateMunkiVersion(ctx, host.ID, rows[0]["version"])
}

func GetDetailQueries(ac *fleet.AppConfig) map[string]DetailQuery {
	generatedMap := make(map[string]DetailQuery)
	for key, query := range detailQueries {
		generatedMap[key] = query
	}

	if ac != nil && ac.HostSettings.EnableSoftwareInventory {
		generatedMap["software_macos"] = softwareMacOS
		generatedMap["software_linux"] = softwareLinux
		generatedMap["software_windows"] = softwareWindows
	}

	if ac != nil && ac.HostSettings.EnableHostUsers {
		generatedMap["users"] = usersQuery
	}

	return generatedMap
}
