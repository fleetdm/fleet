package osquery_utils

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

type DetailQuery struct {
	Query string
	// Platforms is a list of platforms to run the query on. If this value is
	// empty, run on all platforms.
	Platforms  []string
	IngestFunc func(logger log.Logger, host *fleet.Host, rows []map[string]string) error
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
					//IPv6
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
						return errors.Wrap(err, "parsing distributed_interval")
					}
					host.DistributedInterval = uint(interval)

				case "config_tls_refresh":
					// Prior to osquery 2.4.6, the flag was
					// called `config_tls_refresh`.
					interval, err := strconv.Atoi(EmptyToZero(row["value"]))
					if err != nil {
						return errors.Wrap(err, "parsing config_tls_refresh")
					}
					configTLSRefresh = uint(interval)
					configTLSRefreshSeen = true

				case "config_refresh":
					// After 2.4.6 `config_tls_refresh` was
					// aliased to `config_refresh`.
					interval, err := strconv.Atoi(EmptyToZero(row["value"]))
					if err != nil {
						return errors.Wrap(err, "parsing config_refresh")
					}
					configRefresh = uint(interval)
					configRefreshSeen = true

				case "logger_tls_period":
					interval, err := strconv.Atoi(EmptyToZero(row["value"]))
					if err != nil {
						return errors.Wrap(err, "parsing logger_tls_period")
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
			FROM osquery_schedule
`,
		IngestFunc: func(logger log.Logger, host *fleet.Host, rows []map[string]string) error {
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

			host.PackStats = []fleet.PackStats{}
			for packName, stats := range packs {
				host.PackStats = append(
					host.PackStats,
					fleet.PackStats{
						PackName:   packName,
						QueryStats: stats,
					},
				)
			}

			return nil
		},
	},
	"disk_space_unix": {
		Query: `
SELECT (blocks_available * 100 / blocks) AS percent_disk_space_available, 
       round((blocks_available * blocks_size *10e-10),2) AS gigs_disk_space_available 
FROM mounts WHERE path = '/' LIMIT 1;`,
		Platforms:  []string{"darwin", "linux", "rhel", "ubuntu", "centos"},
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
}

var softwareMacOS = DetailQuery{
	Query: `
SELECT
  name AS name,
  bundle_short_version AS version,
  'Application (macOS)' AS type,
  'apps' AS source
FROM apps
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
  'Browser plugin (Chrome)' AS type,
  'chrome_extensions' AS source
FROM chrome_extensions
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Firefox)' AS type,
  'firefox_addons' AS source
FROM firefox_addons
UNION
SELECT
  name As name,
  version AS version,
  'Browser plugin (Safari)' AS type,
  'safari_extensions' AS source
FROM safari_extensions
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Homebrew)' AS type,
  'homebrew_packages' AS source
FROM homebrew_packages;
`,
	Platforms:  []string{"darwin"},
	IngestFunc: ingestSoftware,
}

var softwareLinux = DetailQuery{
	Query: `
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
  'Package (Atom)' AS type,
  'atom_packages' AS source
FROM atom_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Python)' AS type,
  'python_packages' AS source
FROM python_packages;
`,
	Platforms:  []string{"linux", "rhel", "ubuntu", "centos"},
	IngestFunc: ingestSoftware,
}

var softwareWindows = DetailQuery{
	Query: `
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
FROM chrome_extensions
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Firefox)' AS type,
  'firefox_addons' AS source
FROM firefox_addons
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
FROM atom_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Python)' AS type,
  'python_packages' AS source
FROM python_packages;
`,
	Platforms:  []string{"windows"},
	IngestFunc: ingestSoftware,
}

var usersQuery = DetailQuery{
	Query: `SELECT uid, username, type, groupname FROM users u JOIN groups g ON g.gid=u.gid;`,
	IngestFunc: func(logger log.Logger, host *fleet.Host, rows []map[string]string) error {
		var users []fleet.HostUser
		for _, row := range rows {
			uid, err := strconv.Atoi(row["uid"])
			if err != nil {
				return errors.Wrapf(err, "converting uid %s to int", row["uid"])
			}
			username := row["username"]
			type_ := row["type"]
			groupname := row["groupname"]
			u := fleet.HostUser{
				Uid:       uint(uid),
				Username:  username,
				Type:      type_,
				GroupName: groupname,
			}
			users = append(users, u)
		}
		host.Users = users

		return nil
	},
}

func ingestSoftware(logger log.Logger, host *fleet.Host, rows []map[string]string) error {
	software := fleet.HostSoftware{Modified: true}

	for _, row := range rows {
		name := row["name"]
		version := row["version"]
		source := row["source"]
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
		s := fleet.Software{Name: name, Version: version, Source: source}
		software.Software = append(software.Software, s)
	}

	host.HostSoftware = software

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

func GetDetailQueries(ac *fleet.AppConfig) map[string]DetailQuery {
	generatedMap := make(map[string]DetailQuery)
	for key, query := range detailQueries {
		generatedMap[key] = query
	}

	softwareInventory := ac != nil && ac.HostSettings.EnableSoftwareInventory
	if os.Getenv("FLEET_BETA_SOFTWARE_INVENTORY") != "" || softwareInventory {
		generatedMap["software_macos"] = softwareMacOS
		generatedMap["software_linux"] = softwareLinux
		generatedMap["software_windows"] = softwareWindows
	}

	if ac != nil && ac.HostSettings.EnableHostUsers {
		generatedMap["users"] = usersQuery
	}

	return generatedMap
}
