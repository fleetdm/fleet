package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	kithttp "github.com/go-kit/kit/transport/http"

	"github.com/fleetdm/fleet/v4/server/fleet"

	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/pubsub"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

type osqueryError struct {
	message     string
	nodeInvalid bool
}

func (e osqueryError) Error() string {
	return e.message
}

func (e osqueryError) NodeInvalid() bool {
	return e.nodeInvalid
}

// Sometimes osquery gives us empty string where we expect an integer.
// We change the to "0" so it can be handled by the appropriate string to
// integer conversion function, as these will err on ""
func emptyToZero(val string) string {
	if val == "" {
		return "0"
	}
	return val
}

func (svc Service) AuthenticateHost(ctx context.Context, nodeKey string) (*fleet.Host, error) {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	logIPs(ctx)

	if nodeKey == "" {
		return nil, osqueryError{
			message:     "authentication error: missing node key",
			nodeInvalid: true,
		}
	}

	host, err := svc.ds.AuthenticateHost(nodeKey)
	if err != nil {
		switch err.(type) {
		case fleet.NotFoundError:
			return nil, osqueryError{
				message:     "authentication error: invalid node key: " + nodeKey,
				nodeInvalid: true,
			}
		default:
			return nil, osqueryError{
				message: "authentication error: " + err.Error(),
			}
		}
	}

	// Update the "seen" time used to calculate online status. These updates are
	// batched for MySQL performance reasons. Because this is done
	// asynchronously, it is possible for the server to shut down before
	// updating the seen time for these hosts. This seems to be an acceptable
	// tradeoff as an online host will continue to check in and quickly be
	// marked online again.
	svc.seenHostSet.addHostID(host.ID)
	host.SeenTime = svc.clock.Now()

	return host, nil
}

func (svc Service) EnrollAgent(ctx context.Context, enrollSecret, hostIdentifier string, hostDetails map[string](map[string]string)) (string, error) {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	logIPs(ctx, "hostIdentifier", hostIdentifier)

	secret, err := svc.ds.VerifyEnrollSecret(enrollSecret)
	if err != nil {
		return "", osqueryError{
			message:     "enroll failed: " + err.Error(),
			nodeInvalid: true,
		}
	}

	nodeKey, err := server.GenerateRandomText(svc.config.Osquery.NodeKeySize)
	if err != nil {
		return "", osqueryError{
			message:     "generate node key failed: " + err.Error(),
			nodeInvalid: true,
		}
	}

	hostIdentifier = getHostIdentifier(svc.logger, svc.config.Osquery.HostIdentifier, hostIdentifier, hostDetails)

	host, err := svc.ds.EnrollHost(hostIdentifier, nodeKey, secret.TeamID, svc.config.Osquery.EnrollCooldown)
	if err != nil {
		return "", osqueryError{message: "save enroll failed: " + err.Error(), nodeInvalid: true}
	}

	// Save enrollment details if provided
	save := false
	if r, ok := hostDetails["os_version"]; ok {
		detailQueries["os_version"].IngestFunc(svc.logger, host, []map[string]string{r})
		save = true
	}
	if r, ok := hostDetails["osquery_info"]; ok {
		detailQueries["osquery_info"].IngestFunc(svc.logger, host, []map[string]string{r})
		save = true
	}
	if r, ok := hostDetails["system_info"]; ok {
		detailQueries["system_info"].IngestFunc(svc.logger, host, []map[string]string{r})
		save = true
	}
	if save {
		if err := svc.ds.SaveHost(host); err != nil {
			return "", osqueryError{message: "saving host details: " + err.Error(), nodeInvalid: true}
		}
	}

	return host.NodeKey, nil
}

func getHostIdentifier(logger log.Logger, identifierOption, providedIdentifier string, details map[string](map[string]string)) string {
	switch identifierOption {
	case "provided":
		// Use the host identifier already provided in the request.
		return providedIdentifier

	case "instance":
		r, ok := details["osquery_info"]
		if !ok {
			level.Info(logger).Log(
				"msg", "could not get host identifier",
				"reason", "missing osquery_info",
				"identifier", "instance",
			)
		} else if r["instance_id"] == "" {
			level.Info(logger).Log(
				"msg", "could not get host identifier",
				"reason", "missing instance_id in osquery_info",
				"identifier", "instance",
			)
		} else {
			return r["instance_id"]
		}

	case "uuid":
		r, ok := details["osquery_info"]
		if !ok {
			level.Info(logger).Log(
				"msg", "could not get host identifier",
				"reason", "missing osquery_info",
				"identifier", "uuid",
			)
		} else if r["uuid"] == "" {
			level.Info(logger).Log(
				"msg", "could not get host identifier",
				"reason", "missing instance_id in osquery_info",
				"identifier", "uuid",
			)
		} else {
			return r["uuid"]
		}

	case "hostname":
		r, ok := details["system_info"]
		if !ok {
			level.Info(logger).Log(
				"msg", "could not get host identifier",
				"reason", "missing system_info",
				"identifier", "hostname",
			)
		} else if r["hostname"] == "" {
			level.Info(logger).Log(
				"msg", "could not get host identifier",
				"reason", "missing instance_id in system_info",
				"identifier", "hostname",
			)
		} else {
			return r["hostname"]
		}

	default:
		panic("Unknown option for host_identifier: " + identifierOption)
	}

	return providedIdentifier
}

func (svc *Service) GetClientConfig(ctx context.Context) (map[string]interface{}, error) {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	logIPs(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, osqueryError{message: "internal error: missing host from request context"}
	}

	baseConfig, err := svc.AgentOptionsForHost(ctx, &host)
	if err != nil {
		return nil, osqueryError{message: "internal error: fetch base config: " + err.Error()}
	}

	var config map[string]interface{}
	err = json.Unmarshal(baseConfig, &config)
	if err != nil {
		return nil, osqueryError{message: "internal error: parse base configuration: " + err.Error()}
	}

	packs, err := svc.ds.ListPacksForHost(host.ID)
	if err != nil {
		return nil, osqueryError{message: "database error: " + err.Error()}
	}

	packConfig := fleet.Packs{}
	for _, pack := range packs {
		// first, we must figure out what queries are in this pack
		queries, err := svc.ds.ListScheduledQueriesInPack(pack.ID, fleet.ListOptions{})
		if err != nil {
			return nil, osqueryError{message: "database error: " + err.Error()}
		}

		// the serializable osquery config struct expects content in a
		// particular format, so we do the conversion here
		configQueries := fleet.Queries{}
		for _, query := range queries {
			queryContent := fleet.QueryContent{
				Query:    query.Query,
				Interval: query.Interval,
				Platform: query.Platform,
				Version:  query.Version,
				Removed:  query.Removed,
				Shard:    query.Shard,
				Denylist: query.Denylist,
			}

			if query.Removed != nil {
				queryContent.Removed = query.Removed
			}

			if query.Snapshot != nil && *query.Snapshot {
				queryContent.Snapshot = query.Snapshot
			}

			configQueries[query.Name] = queryContent
		}

		// finally, we add the pack to the client config struct with all of
		// the pack's queries
		packConfig[pack.Name] = fleet.PackContent{
			Platform: pack.Platform,
			Queries:  configQueries,
		}
	}

	if len(packConfig) > 0 {
		packJSON, err := json.Marshal(packConfig)
		if err != nil {
			return nil, osqueryError{message: "internal error: marshal pack JSON: " + err.Error()}
		}
		config["packs"] = json.RawMessage(packJSON)
	}

	// Save interval values if they have been updated.
	saveHost := false
	if options, ok := config["options"].(map[string]interface{}); ok {
		distributedIntervalVal, ok := options["distributed_interval"]
		distributedInterval, err := cast.ToUintE(distributedIntervalVal)
		if ok && err == nil && host.DistributedInterval != distributedInterval {
			host.DistributedInterval = distributedInterval
			saveHost = true
		}

		loggerTLSPeriodVal, ok := options["logger_tls_period"]
		loggerTLSPeriod, err := cast.ToUintE(loggerTLSPeriodVal)
		if ok && err == nil && host.LoggerTLSPeriod != loggerTLSPeriod {
			host.LoggerTLSPeriod = loggerTLSPeriod
			saveHost = true
		}

		// Note config_tls_refresh can only be set in the osquery flags (and has
		// also been deprecated in osquery for quite some time) so is ignored
		// here.
		configRefreshVal, ok := options["config_refresh"]
		configRefresh, err := cast.ToUintE(configRefreshVal)
		if ok && err == nil && host.ConfigTLSRefresh != configRefresh {
			host.ConfigTLSRefresh = configRefresh
			saveHost = true
		}
	}

	if saveHost {
		err := svc.ds.SaveHost(&host)
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

func (svc *Service) SubmitStatusLogs(ctx context.Context, logs []json.RawMessage) error {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	logIPs(ctx)

	if err := svc.osqueryLogWriter.Status.Write(ctx, logs); err != nil {
		return osqueryError{message: "error writing status logs: " + err.Error()}
	}
	return nil
}

func logIPs(ctx context.Context, extras ...interface{}) {
	remoteAddr, _ := ctx.Value(kithttp.ContextKeyRequestRemoteAddr).(string)
	xForwardedFor, _ := ctx.Value(kithttp.ContextKeyRequestXForwardedFor).(string)
	logging.WithDebugExtrasNoUser(ctx, append(extras, "ip_addr", remoteAddr, "x_for_ip_addr", xForwardedFor)...)
}

func (svc *Service) SubmitResultLogs(ctx context.Context, logs []json.RawMessage) error {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	logIPs(ctx)

	if err := svc.osqueryLogWriter.Result.Write(ctx, logs); err != nil {
		return osqueryError{message: "error writing result logs: " + err.Error()}
	}
	return nil
}

// hostLabelQueryPrefix is appended before the query name when a query is
// provided as a label query. This allows the results to be retrieved when
// osqueryd writes the distributed query results.
const hostLabelQueryPrefix = "fleet_label_query_"

// hostDetailQueryPrefix is appended before the query name when a query is
// provided as a detail query.
const hostDetailQueryPrefix = "fleet_detail_query_"

// hostAdditionalQueryPrefix is appended before the query name when a query is
// provided as an additional query (additional info for hosts to retrieve).
const hostAdditionalQueryPrefix = "fleet_additional_query_"

// hostDistributedQueryPrefix is appended before the query name when a query is
// run from a distributed query campaign
const hostDistributedQueryPrefix = "fleet_distributed_query_"

type detailQuery struct {
	Query string
	// Platforms is a list of platforms to run the query on. If this value is
	// empty, run on all platforms.
	Platforms  []string
	IngestFunc func(logger log.Logger, host *fleet.Host, rows []map[string]string) error
}

// runForPlatform determines whether this detail query should run on the given platform
func (q *detailQuery) runForPlatform(platform string) bool {
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
var detailQueries = map[string]detailQuery{
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
					interval, err := strconv.Atoi(emptyToZero(row["value"]))
					if err != nil {
						return errors.Wrap(err, "parsing distributed_interval")
					}
					host.DistributedInterval = uint(interval)

				case "config_tls_refresh":
					// Prior to osquery 2.4.6, the flag was
					// called `config_tls_refresh`.
					interval, err := strconv.Atoi(emptyToZero(row["value"]))
					if err != nil {
						return errors.Wrap(err, "parsing config_tls_refresh")
					}
					configTLSRefresh = uint(interval)
					configTLSRefreshSeen = true

				case "config_refresh":
					// After 2.4.6 `config_tls_refresh` was
					// aliased to `config_refresh`.
					interval, err := strconv.Atoi(emptyToZero(row["value"]))
					if err != nil {
						return errors.Wrap(err, "parsing config_refresh")
					}
					configRefresh = uint(interval)
					configRefreshSeen = true

				case "logger_tls_period":
					interval, err := strconv.Atoi(emptyToZero(row["value"]))
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
			host.Memory, err = strconv.ParseInt(emptyToZero(rows[0]["physical_memory"]), 10, 64)
			if err != nil {
				return err
			}
			host.Hostname = rows[0]["hostname"]
			host.UUID = rows[0]["uuid"]
			host.CPUType = rows[0]["cpu_type"]
			host.CPUSubtype = rows[0]["cpu_subtype"]
			host.CPUBrand = rows[0]["cpu_brand"]
			host.CPUPhysicalCores, err = strconv.Atoi(emptyToZero(rows[0]["cpu_physical_cores"]))
			if err != nil {
				return err
			}
			host.CPULogicalCores, err = strconv.Atoi(emptyToZero(rows[0]["cpu_logical_cores"]))
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

			uptimeSeconds, err := strconv.Atoi(emptyToZero(rows[0]["total_seconds"]))
			if err != nil {
				return err
			}
			host.Uptime = time.Duration(uptimeSeconds) * time.Second

			return nil
		},
	},
	"software_macos": {
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
	},
	"software_linux": {
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
	},
	"software_windows": {
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
	"users": {
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

// hostDetailQueries returns the map of queries that should be executed by
// osqueryd to fill in the host details
func (svc *Service) hostDetailQueries(host fleet.Host) (map[string]string, error) {
	queries := make(map[string]string)
	if host.DetailUpdatedAt.After(svc.clock.Now().Add(-svc.config.Osquery.DetailUpdateInterval)) && !host.RefetchRequested {
		// No need to update already fresh details
		return queries, nil
	}

	for name, query := range detailQueries {
		if query.runForPlatform(host.Platform) {
			if strings.HasPrefix(name, "software_") {
				// Feature flag this because of as-yet-untested performance
				// considerations.
				if os.Getenv("FLEET_BETA_SOFTWARE_INVENTORY") == "" {
					continue
				}
			}
			queries[hostDetailQueryPrefix+name] = query.Query
		}
	}

	// Get additional queries
	config, err := svc.ds.AppConfig()
	if err != nil {
		return nil, osqueryError{message: "get additional queries: " + err.Error()}
	}
	if config.AdditionalQueries == nil {
		// No additional queries set
		return queries, nil
	}

	var additionalQueries map[string]string
	if err := json.Unmarshal(*config.AdditionalQueries, &additionalQueries); err != nil {
		return nil, osqueryError{message: "unmarshal additional queries: " + err.Error()}
	}

	for name, query := range additionalQueries {
		queries[hostAdditionalQueryPrefix+name] = query
	}

	return queries, nil
}

func (svc *Service) GetDistributedQueries(ctx context.Context) (map[string]string, uint, error) {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	logIPs(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, 0, osqueryError{message: "internal error: missing host from request context"}
	}

	queries, err := svc.hostDetailQueries(host)
	if err != nil {
		return nil, 0, err
	}

	// Retrieve the label queries that should be updated
	cutoff := svc.clock.Now().Add(-svc.config.Osquery.LabelUpdateInterval)
	labelQueries, err := svc.ds.LabelQueriesForHost(&host, cutoff)
	if err != nil {
		return nil, 0, osqueryError{message: "retrieving label queries: " + err.Error()}
	}

	for name, query := range labelQueries {
		queries[hostLabelQueryPrefix+name] = query
	}

	liveQueries, err := svc.liveQueryStore.QueriesForHost(host.ID)
	if err != nil {
		return nil, 0, osqueryError{message: "retrieve live queries: " + err.Error()}
	}

	for name, query := range liveQueries {
		queries[hostDistributedQueryPrefix+name] = query
	}

	accelerate := uint(0)
	if host.Hostname == "" || host.Platform == "" {
		// Assume this host is just enrolling, and accelerate checkins
		// (to allow for platform restricted labels to run quickly
		// after platform is retrieved from details)
		accelerate = 10
	}

	return queries, accelerate, nil
}

// ingestDetailQuery takes the results of a detail query and modifies the
// provided fleet.Host appropriately.
func (svc *Service) ingestDetailQuery(host *fleet.Host, name string, rows []map[string]string) error {
	trimmedQuery := strings.TrimPrefix(name, hostDetailQueryPrefix)
	query, ok := detailQueries[trimmedQuery]
	if !ok {
		return osqueryError{message: "unknown detail query " + trimmedQuery}
	}

	err := query.IngestFunc(svc.logger, host, rows)
	if err != nil {
		return osqueryError{
			message: fmt.Sprintf("ingesting query %s: %s", name, err.Error()),
		}
	}

	// Refetch is no longer needed after ingesting details.
	host.RefetchRequested = false

	return nil
}

// ingestLabelQuery records the results of label queries run by a host
func (svc *Service) ingestLabelQuery(host fleet.Host, query string, rows []map[string]string, results map[uint]bool) error {
	trimmedQuery := strings.TrimPrefix(query, hostLabelQueryPrefix)
	trimmedQueryNum, err := strconv.Atoi(emptyToZero(trimmedQuery))
	if err != nil {
		return errors.Wrap(err, "converting query from string to int")
	}
	// A label query matches if there is at least one result for that
	// query. We must also store negative results.
	results[uint(trimmedQueryNum)] = len(rows) > 0
	return nil
}

// ingestDistributedQuery takes the results of a distributed query and modifies the
// provided fleet.Host appropriately.
func (svc *Service) ingestDistributedQuery(host fleet.Host, name string, rows []map[string]string, failed bool, errMsg string) error {
	trimmedQuery := strings.TrimPrefix(name, hostDistributedQueryPrefix)

	campaignID, err := strconv.Atoi(emptyToZero(trimmedQuery))
	if err != nil {
		return osqueryError{message: "unable to parse campaign ID: " + trimmedQuery}
	}

	// Write the results to the pubsub store
	res := fleet.DistributedQueryResult{
		DistributedQueryCampaignID: uint(campaignID),
		Host:                       host,
		Rows:                       rows,
	}
	if failed {
		res.Error = &errMsg
	}

	err = svc.resultStore.WriteResult(res)
	if err != nil {
		nErr, ok := err.(pubsub.Error)
		if !ok || !nErr.NoSubscriber() {
			return osqueryError{message: "writing results: " + err.Error()}
		}

		// If there are no subscribers, the campaign is "orphaned"
		// and should be closed so that we don't continue trying to
		// execute that query when we can't write to any subscriber
		campaign, err := svc.ds.DistributedQueryCampaign(uint(campaignID))
		if err != nil {
			if err := svc.liveQueryStore.StopQuery(strconv.Itoa(int(campaignID))); err != nil {
				return osqueryError{message: "stop orphaned campaign after load failure: " + err.Error()}
			}
			return osqueryError{message: "loading orphaned campaign: " + err.Error()}
		}

		if campaign.CreatedAt.After(svc.clock.Now().Add(-5 * time.Second)) {
			// Give the client 5 seconds to connect before considering the
			// campaign orphaned
			return osqueryError{message: "campaign waiting for listener (please retry)"}
		}

		if campaign.Status != fleet.QueryComplete {
			campaign.Status = fleet.QueryComplete
			if err := svc.ds.SaveDistributedQueryCampaign(campaign); err != nil {
				return osqueryError{message: "closing orphaned campaign: " + err.Error()}
			}
		}

		if err := svc.liveQueryStore.StopQuery(strconv.Itoa(int(campaignID))); err != nil {
			return osqueryError{message: "stopping orphaned campaign: " + err.Error()}
		}

		// No need to record query completion in this case
		return nil
	}

	err = svc.liveQueryStore.QueryCompletedByHost(strconv.Itoa(int(campaignID)), host.ID)
	if err != nil {
		return osqueryError{message: "record query completion: " + err.Error()}
	}

	return nil
}

func (svc *Service) SubmitDistributedQueryResults(ctx context.Context, results fleet.OsqueryDistributedQueryResults, statuses map[string]fleet.OsqueryStatus, messages map[string]string) error {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	logIPs(ctx)

	host, ok := hostctx.FromContext(ctx)

	if !ok {
		return osqueryError{message: "internal error: missing host from request context"}
	}

	// Check for label queries and if so, load host additional. If we don't do
	// this, we will end up unintentionally dropping any existing host
	// additional info.
	for query := range results {
		if strings.HasPrefix(query, hostLabelQueryPrefix) {
			fullHost, err := svc.ds.Host(host.ID)
			if err != nil {
				return osqueryError{message: "internal error: load host additional: " + err.Error()}
			}
			host = *fullHost
			break
		}
	}

	var err error
	detailUpdated := false // Whether detail or additional was updated
	additionalResults := make(fleet.OsqueryDistributedQueryResults)
	labelResults := map[uint]bool{}
	for query, rows := range results {
		switch {
		case strings.HasPrefix(query, hostDetailQueryPrefix):
			err = svc.ingestDetailQuery(&host, query, rows)
			detailUpdated = true
		case strings.HasPrefix(query, hostAdditionalQueryPrefix):
			name := strings.TrimPrefix(query, hostAdditionalQueryPrefix)
			additionalResults[name] = rows
			detailUpdated = true
		case strings.HasPrefix(query, hostLabelQueryPrefix):
			err = svc.ingestLabelQuery(host, query, rows, labelResults)
		case strings.HasPrefix(query, hostDistributedQueryPrefix):
			// osquery docs say any nonzero (string) value for
			// status indicates a query error
			status, ok := statuses[query]
			failed := (ok && status != fleet.StatusOK)
			err = svc.ingestDistributedQuery(host, query, rows, failed, messages[query])
		default:
			err = osqueryError{message: "unknown query prefix: " + query}
		}

		if err != nil {
			return osqueryError{message: "failed to ingest result: " + err.Error()}
		}
	}

	if len(labelResults) > 0 {
		host.Modified = true
		host.LabelUpdatedAt = svc.clock.Now()
		err = svc.ds.RecordLabelQueryExecutions(&host, labelResults, svc.clock.Now())
		if err != nil {
			return osqueryError{message: "failed to save labels: " + err.Error()}
		}
	}

	if detailUpdated {
		host.Modified = true
		host.DetailUpdatedAt = svc.clock.Now()
		additionalJSON, err := json.Marshal(additionalResults)
		if err != nil {
			return osqueryError{message: "failed to marshal additional: " + err.Error()}
		}
		additional := json.RawMessage(additionalJSON)
		host.Additional = &additional
	}

	if host.Modified {
		err = svc.ds.SaveHost(&host)
		if err != nil {
			return osqueryError{message: "failed to update host details: " + err.Error()}
		}
	}

	return nil
}
