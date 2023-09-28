package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/pubsub"
	"github.com/fleetdm/fleet/v4/server/service/osquery_utils"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/spf13/cast"
)

// osqueryError is the error returned to osquery agents.
type osqueryError struct {
	message     string
	nodeInvalid bool

	fleet.ErrorWithUUID
}

var _ fleet.ErrorUUIDer = (*osqueryError)(nil)

// Error implements the error interface.
func (e *osqueryError) Error() string {
	return e.message
}

// NodeInvalid returns whether the error returned to osquery
// should contain the node_invalid property.
func (e *osqueryError) NodeInvalid() bool {
	return e.nodeInvalid
}

func newOsqueryErrorWithInvalidNode(msg string) *osqueryError {
	return &osqueryError{
		message:     msg,
		nodeInvalid: true,
	}
}

func newOsqueryError(msg string) *osqueryError {
	return &osqueryError{
		message:     msg,
		nodeInvalid: false,
	}
}

func (svc *Service) AuthenticateHost(ctx context.Context, nodeKey string) (*fleet.Host, bool, error) {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	if nodeKey == "" {
		return nil, false, newOsqueryErrorWithInvalidNode("authentication error: missing node key")
	}

	host, err := svc.ds.LoadHostByNodeKey(ctx, nodeKey)
	switch {
	case err == nil:
		// OK
	case fleet.IsNotFound(err):
		return nil, false, newOsqueryErrorWithInvalidNode("authentication error: invalid node key: " + nodeKey)
	default:
		return nil, false, newOsqueryError("authentication error: " + err.Error())
	}

	// Update the "seen" time used to calculate online status. These updates are
	// batched for MySQL performance reasons. Because this is done
	// asynchronously, it is possible for the server to shut down before
	// updating the seen time for these hosts. This seems to be an acceptable
	// tradeoff as an online host will continue to check in and quickly be
	// marked online again.
	if err := svc.task.RecordHostLastSeen(ctx, host.ID); err != nil {
		logging.WithErr(ctx, ctxerr.Wrap(ctx, err, "record host last seen"))
	}
	host.SeenTime = svc.clock.Now()

	return host, svc.debugEnabledForHost(ctx, host.ID), nil
}

////////////////////////////////////////////////////////////////////////////////
// Enroll Agent
////////////////////////////////////////////////////////////////////////////////

type enrollAgentRequest struct {
	EnrollSecret   string                         `json:"enroll_secret"`
	HostIdentifier string                         `json:"host_identifier"`
	HostDetails    map[string](map[string]string) `json:"host_details"`
}

type enrollAgentResponse struct {
	NodeKey string `json:"node_key,omitempty"`
	Err     error  `json:"error,omitempty"`
}

func (r enrollAgentResponse) error() error { return r.Err }

func enrollAgentEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*enrollAgentRequest)
	nodeKey, err := svc.EnrollAgent(ctx, req.EnrollSecret, req.HostIdentifier, req.HostDetails)
	if err != nil {
		return enrollAgentResponse{Err: err}, nil
	}
	return enrollAgentResponse{NodeKey: nodeKey}, nil
}

func (svc *Service) EnrollAgent(ctx context.Context, enrollSecret, hostIdentifier string, hostDetails map[string](map[string]string)) (string, error) {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	logging.WithExtras(ctx, "hostIdentifier", hostIdentifier)

	secret, err := svc.ds.VerifyEnrollSecret(ctx, enrollSecret)
	if err != nil {
		return "", newOsqueryErrorWithInvalidNode("enroll failed: " + err.Error())
	}

	nodeKey, err := server.GenerateRandomText(svc.config.Osquery.NodeKeySize)
	if err != nil {
		return "", newOsqueryErrorWithInvalidNode("generate node key failed: " + err.Error())
	}

	hostIdentifier = getHostIdentifier(svc.logger, svc.config.Osquery.HostIdentifier, hostIdentifier, hostDetails)
	canEnroll, err := svc.enrollHostLimiter.CanEnrollNewHost(ctx)
	if err != nil {
		return "", newOsqueryErrorWithInvalidNode("can enroll host check failed: " + err.Error())
	}
	if !canEnroll {
		deviceCount := "unknown"
		if lic, _ := license.FromContext(ctx); lic != nil {
			deviceCount = strconv.Itoa(lic.DeviceCount)
		}
		return "", newOsqueryErrorWithInvalidNode(fmt.Sprintf("enroll host failed: maximum number of hosts reached: %s", deviceCount))
	}

	// the the device's uuid and serial from the system_info table provided with
	// the osquery enrollment
	var hardwareUUID, hardwareSerial string
	if r, ok := hostDetails["system_info"]; ok {
		hardwareUUID = r["uuid"]
		hardwareSerial = r["hardware_serial"]
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return "", newOsqueryErrorWithInvalidNode("app config load failed: " + err.Error())
	}

	host, err := svc.ds.EnrollHost(ctx, appConfig.MDM.EnabledAndConfigured, hostIdentifier, hardwareUUID, hardwareSerial, nodeKey, secret.TeamID, svc.config.Osquery.EnrollCooldown)
	if err != nil {
		return "", newOsqueryErrorWithInvalidNode("save enroll failed: " + err.Error())
	}

	features, err := svc.HostFeatures(ctx, host)
	if err != nil {
		return "", newOsqueryErrorWithInvalidNode("host features load failed: " + err.Error())
	}

	// Save enrollment details if provided
	detailQueries := osquery_utils.GetDetailQueries(ctx, svc.config, appConfig, features)
	save := false
	if r, ok := hostDetails["os_version"]; ok {
		err := detailQueries["os_version"].IngestFunc(ctx, svc.logger, host, []map[string]string{r})
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "Ingesting os_version")
		}
		save = true
	}
	if r, ok := hostDetails["osquery_info"]; ok {
		err := detailQueries["osquery_info"].IngestFunc(ctx, svc.logger, host, []map[string]string{r})
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "Ingesting osquery_info")
		}
		save = true
	}
	if r, ok := hostDetails["system_info"]; ok {
		err := detailQueries["system_info"].IngestFunc(ctx, svc.logger, host, []map[string]string{r})
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "Ingesting system_info")
		}
		save = true
	}

	if save {
		if appConfig.ServerSettings.DeferredSaveHost {
			go svc.serialUpdateHost(host)
		} else {
			if err := svc.ds.UpdateHost(ctx, host); err != nil {
				return "", ctxerr.Wrap(ctx, err, "save host in enroll agent")
			}
		}
	}

	return nodeKey, nil
}

var counter = int64(0)

func (svc *Service) serialUpdateHost(host *fleet.Host) {
	newVal := atomic.AddInt64(&counter, 1)
	defer func() {
		atomic.AddInt64(&counter, -1)
	}()
	level.Debug(svc.logger).Log("background", newVal)

	ctx, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFunc()
	err := svc.ds.SerialUpdateHost(ctx, host)
	if err != nil {
		level.Error(svc.logger).Log("background-err", err)
	}
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

func (svc *Service) debugEnabledForHost(ctx context.Context, id uint) bool {
	hlogger := log.With(svc.logger, "host-id", id)
	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		level.Debug(hlogger).Log("err", ctxerr.Wrap(ctx, err, "getting app config for host debug"))
		return false
	}

	for _, hostID := range ac.ServerSettings.DebugHostIDs {
		if hostID == id {
			return true
		}
	}
	return false
}

////////////////////////////////////////////////////////////////////////////////
// Get Client Config
////////////////////////////////////////////////////////////////////////////////

type getClientConfigRequest struct {
	NodeKey string `json:"node_key"`
}

func (r *getClientConfigRequest) hostNodeKey() string {
	return r.NodeKey
}

type getClientConfigResponse struct {
	Config map[string]interface{}
	Err    error `json:"error,omitempty"`
}

func (r getClientConfigResponse) error() error { return r.Err }

// MarshalJSON implements json.Marshaler.
//
// Osquery expects the response for configs to be at the
// top-level of the JSON response.
func (r getClientConfigResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Config)
}

// UnmarshalJSON implements json.Unmarshaler.
//
// Osquery expects the response for configs to be at the
// top-level of the JSON response.
func (r *getClientConfigResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &r.Config)
}

func getClientConfigEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	config, err := svc.GetClientConfig(ctx)
	if err != nil {
		return getClientConfigResponse{Err: err}, nil
	}

	return getClientConfigResponse{
		Config: config,
	}, nil
}

func (svc *Service) getScheduledQueries(ctx context.Context, teamID *uint) (fleet.Queries, error) {
	queries, err := svc.ds.ListScheduledQueriesForAgents(ctx, teamID)
	if err != nil {
		return nil, err
	}

	if len(queries) == 0 {
		return nil, nil
	}

	config := make(fleet.Queries, len(queries))
	for _, query := range queries {
		config[query.Name] = query.ToQueryContent()
	}

	return config, nil
}

func (svc *Service) GetClientConfig(ctx context.Context) (map[string]interface{}, error) {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, newOsqueryError("internal error: missing host from request context")
	}

	baseConfig, err := svc.AgentOptionsForHost(ctx, host.TeamID, host.Platform)
	if err != nil {
		return nil, newOsqueryError("internal error: fetch base config: " + err.Error())
	}

	config := make(map[string]interface{})
	if baseConfig != nil {
		err = json.Unmarshal(baseConfig, &config)
		if err != nil {
			return nil, newOsqueryError("internal error: parse base configuration: " + err.Error())
		}
	}

	packConfig := fleet.Packs{}

	packs, err := svc.ds.ListPacksForHost(ctx, host.ID)
	if err != nil {
		return nil, newOsqueryError("database error: " + err.Error())
	}
	for _, pack := range packs {
		// first, we must figure out what queries are in this pack
		queries, err := svc.ds.ListScheduledQueriesInPack(ctx, pack.ID)
		if err != nil {
			return nil, newOsqueryError("database error: " + err.Error())
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

	globalQueries, err := svc.getScheduledQueries(ctx, nil)
	if err != nil {
		return nil, newOsqueryError("database error: " + err.Error())
	}
	if len(globalQueries) > 0 {
		packConfig["Global"] = fleet.PackContent{
			Queries: globalQueries,
		}
	}

	if host.TeamID != nil {
		teamQueries, err := svc.getScheduledQueries(ctx, host.TeamID)
		if err != nil {
			return nil, newOsqueryError("database error: " + err.Error())
		}
		if len(teamQueries) > 0 {
			packName := fmt.Sprintf("team-%d", *host.TeamID)
			packConfig[packName] = fleet.PackContent{
				Queries: teamQueries,
			}
		}
	}

	if len(packConfig) > 0 {
		packJSON, err := json.Marshal(packConfig)
		if err != nil {
			return nil, newOsqueryError("internal error: marshal pack JSON: " + err.Error())
		}
		config["packs"] = json.RawMessage(packJSON)
	}

	// Save interval values if they have been updated.
	intervalsModified := false
	intervals := fleet.HostOsqueryIntervals{
		DistributedInterval: host.DistributedInterval,
		ConfigTLSRefresh:    host.ConfigTLSRefresh,
		LoggerTLSPeriod:     host.LoggerTLSPeriod,
	}
	if options, ok := config["options"].(map[string]interface{}); ok {
		distributedIntervalVal, ok := options["distributed_interval"]
		distributedInterval, err := cast.ToUintE(distributedIntervalVal)
		if ok && err == nil && intervals.DistributedInterval != distributedInterval {
			intervals.DistributedInterval = distributedInterval
			intervalsModified = true
		}

		loggerTLSPeriodVal, ok := options["logger_tls_period"]
		loggerTLSPeriod, err := cast.ToUintE(loggerTLSPeriodVal)
		if ok && err == nil && intervals.LoggerTLSPeriod != loggerTLSPeriod {
			intervals.LoggerTLSPeriod = loggerTLSPeriod
			intervalsModified = true
		}

		// Note config_tls_refresh can only be set in the osquery flags (and has
		// also been deprecated in osquery for quite some time) so is ignored
		// here.
		configRefreshVal, ok := options["config_refresh"]
		configRefresh, err := cast.ToUintE(configRefreshVal)
		if ok && err == nil && intervals.ConfigTLSRefresh != configRefresh {
			intervals.ConfigTLSRefresh = configRefresh
			intervalsModified = true
		}
	}

	// We are not doing deferred update host like in other places because the intervals
	// are not modified often.
	if intervalsModified {
		if err := svc.ds.UpdateHostOsqueryIntervals(ctx, host.ID, intervals); err != nil {
			return nil, newOsqueryError("internal error: update host intervals: " + err.Error())
		}
	}

	return config, nil
}

// AgentOptionsForHost gets the agent options for the provided host.
// The host information should be used for filtering based on team, platform, etc.
func (svc *Service) AgentOptionsForHost(ctx context.Context, hostTeamID *uint, hostPlatform string) (json.RawMessage, error) {
	// Team agent options have priority over global options.
	if hostTeamID != nil {
		teamAgentOptions, err := svc.ds.TeamAgentOptions(ctx, *hostTeamID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "load team agent options for host")
		}

		if teamAgentOptions != nil && len(*teamAgentOptions) > 0 {
			var options fleet.AgentOptions
			if err := json.Unmarshal(*teamAgentOptions, &options); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "unmarshal team agent options")
			}
			return options.ForPlatform(hostPlatform), nil
		}
	}
	// Otherwise return the appropriate override for global options.
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load global agent options")
	}
	var options fleet.AgentOptions
	if appConfig.AgentOptions != nil {
		if err := json.Unmarshal(*appConfig.AgentOptions, &options); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "unmarshal global agent options")
		}
	}
	return options.ForPlatform(hostPlatform), nil
}

////////////////////////////////////////////////////////////////////////////////
// Get Distributed Queries
////////////////////////////////////////////////////////////////////////////////

type getDistributedQueriesRequest struct {
	NodeKey string `json:"node_key"`
}

func (r *getDistributedQueriesRequest) hostNodeKey() string {
	return r.NodeKey
}

type getDistributedQueriesResponse struct {
	Queries    map[string]string `json:"queries"`
	Discovery  map[string]string `json:"discovery"`
	Accelerate uint              `json:"accelerate,omitempty"`
	Err        error             `json:"error,omitempty"`
}

func (r getDistributedQueriesResponse) error() error { return r.Err }

func getDistributedQueriesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	queries, discovery, accelerate, err := svc.GetDistributedQueries(ctx)
	if err != nil {
		return getDistributedQueriesResponse{Err: err}, nil
	}
	return getDistributedQueriesResponse{
		Queries:    queries,
		Discovery:  discovery,
		Accelerate: accelerate,
	}, nil
}

func (svc *Service) GetDistributedQueries(ctx context.Context) (queries map[string]string, discovery map[string]string, accelerate uint, err error) {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, nil, 0, newOsqueryError("internal error: missing host from request context")
	}

	queries = make(map[string]string)
	discovery = make(map[string]string)

	detailQueries, detailDiscovery, err := svc.detailQueriesForHost(ctx, host)
	if err != nil {
		return nil, nil, 0, newOsqueryError(err.Error())
	}
	for name, query := range detailQueries {
		queries[name] = query
	}
	for name, query := range detailDiscovery {
		discovery[name] = query
	}

	labelQueries, err := svc.labelQueriesForHost(ctx, host)
	if err != nil {
		return nil, nil, 0, newOsqueryError(err.Error())
	}
	for name, query := range labelQueries {
		queries[hostLabelQueryPrefix+name] = query
	}

	if liveQueries, err := svc.liveQueryStore.QueriesForHost(host.ID); err != nil {
		// If the live query store fails to fetch queries we still want the hosts
		// to receive all the other queries (details, policies, labels, etc.),
		// thus we just log the error.
		level.Error(svc.logger).Log("op", "QueriesForHost", "err", err)
	} else {
		for name, query := range liveQueries {
			queries[hostDistributedQueryPrefix+name] = query
		}
	}

	policyQueries, noPolicies, err := svc.policyQueriesForHost(ctx, host)
	if err != nil {
		return nil, nil, 0, newOsqueryError(err.Error())
	}
	for name, query := range policyQueries {
		queries[hostPolicyQueryPrefix+name] = query
	}
	if noPolicies {
		// This is only set when it's time to re-run policies on the host,
		// but the host doesn't have any policies assigned.
		queries[hostNoPoliciesWildcard] = alwaysTrueQuery
	}

	accelerate = uint(0)
	if host.Hostname == "" || host.Platform == "" {
		// Assume this host is just enrolling, and accelerate checkins
		// (to allow for platform restricted labels to run quickly
		// after platform is retrieved from details)
		accelerate = 10
	}

	// The way osquery's distributed "discovery" queries work is:
	// If len(discovery) > 0, then only those queries that have a "discovery"
	// query and return more than one row are executed on the host.
	//
	// Thus, we set the alwaysTrueQuery for all queries, except for those where we set
	// an explicit discovery query (e.g. orbit_info, google_chrome_profiles).
	for name := range queries {
		discoveryQuery := discovery[name]
		if discoveryQuery == "" {
			discoveryQuery = alwaysTrueQuery
		}
		discovery[name] = discoveryQuery
	}

	return queries, discovery, accelerate, nil
}

const alwaysTrueQuery = "SELECT 1"

// list of detail queries that are returned when only the critical queries
// should be returned (due to RefetchCriticalQueriesUntil timestamp being set).
var criticalDetailQueries = map[string]bool{
	"mdm": true,
}

// detailQueriesForHost returns the map of detail+additional queries that should be executed by
// osqueryd to fill in the host details.
func (svc *Service) detailQueriesForHost(ctx context.Context, host *fleet.Host) (queries map[string]string, discovery map[string]string, err error) {
	var criticalQueriesOnly bool
	if !svc.shouldUpdate(host.DetailUpdatedAt, svc.config.Osquery.DetailUpdateInterval, host.ID) && !host.RefetchRequested {
		// would not return anything, check if critical queries should be returned
		if host.RefetchCriticalQueriesUntil != nil && host.RefetchCriticalQueriesUntil.After(svc.clock.Now()) {
			// return only those critical queries
			criticalQueriesOnly = true
		} else {
			return nil, nil, nil
		}
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "read app config")
	}

	features, err := svc.HostFeatures(ctx, host)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "read host features")
	}

	queries = make(map[string]string)
	discovery = make(map[string]string)

	detailQueries := osquery_utils.GetDetailQueries(ctx, svc.config, appConfig, features)
	for name, query := range detailQueries {
		if criticalQueriesOnly && !criticalDetailQueries[name] {
			continue
		}

		if query.RunsForPlatform(host.Platform) {
			queryName := hostDetailQueryPrefix + name
			queries[queryName] = query.Query
			discoveryQuery := query.Discovery
			if discoveryQuery == "" {
				discoveryQuery = alwaysTrueQuery
			}
			discovery[queryName] = discoveryQuery
		}
	}

	if features.AdditionalQueries == nil || criticalQueriesOnly {
		// No additional queries set
		return queries, discovery, nil
	}

	var additionalQueries map[string]string
	if err := json.Unmarshal(*features.AdditionalQueries, &additionalQueries); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "unmarshal additional queries")
	}

	for name, query := range additionalQueries {
		queryName := hostAdditionalQueryPrefix + name
		queries[queryName] = query
		discovery[queryName] = alwaysTrueQuery
	}

	return queries, discovery, nil
}

func (svc *Service) shouldUpdate(lastUpdated time.Time, interval time.Duration, hostID uint) bool {
	svc.jitterMu.Lock()
	defer svc.jitterMu.Unlock()

	if svc.jitterH[interval] == nil {
		svc.jitterH[interval] = newJitterHashTable(int(int64(svc.config.Osquery.MaxJitterPercent) * int64(interval.Minutes()) / 100.0))
		level.Debug(svc.logger).Log("jitter", "created", "bucketCount", svc.jitterH[interval].bucketCount)
	}

	jitter := svc.jitterH[interval].jitterForHost(hostID)
	cutoff := svc.clock.Now().Add(-(interval + jitter))
	return lastUpdated.Before(cutoff)
}

func (svc *Service) labelQueriesForHost(ctx context.Context, host *fleet.Host) (map[string]string, error) {
	labelReportedAt := svc.task.GetHostLabelReportedAt(ctx, host)
	if !svc.shouldUpdate(labelReportedAt, svc.config.Osquery.LabelUpdateInterval, host.ID) && !host.RefetchRequested {
		return nil, nil
	}
	labelQueries, err := svc.ds.LabelQueriesForHost(ctx, host)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "retrieve label queries")
	}
	return labelQueries, nil
}

// policyQueriesForHost returns policy queries if it's the time to re-run policies on the given host.
// It returns (nil, true, nil) if the interval is so that policies should be executed on the host, but there are no policies
// assigned to such host.
func (svc *Service) policyQueriesForHost(ctx context.Context, host *fleet.Host) (policyQueries map[string]string, noPoliciesForHost bool, err error) {
	policyReportedAt := svc.task.GetHostPolicyReportedAt(ctx, host)
	if !svc.shouldUpdate(policyReportedAt, svc.config.Osquery.PolicyUpdateInterval, host.ID) && !host.RefetchRequested {
		return nil, false, nil
	}
	policyQueries, err = svc.ds.PolicyQueriesForHost(ctx, host)
	if err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "retrieve policy queries")
	}
	if len(policyQueries) == 0 {
		return nil, true, nil
	}
	return policyQueries, false, nil
}

////////////////////////////////////////////////////////////////////////////////
// Write Distributed Query Results
////////////////////////////////////////////////////////////////////////////////

// When a distributed query has no results, the JSON schema is
// inconsistent, so we use this shim and massage into a consistent
// schema. For example (simplified from actual osqueryd 1.8.2 output):
// {
//
//	"queries": {
//	  "query_with_no_results": "", // <- Note string instead of array
//	  "query_with_results": [{"foo":"bar","baz":"bang"}]
//	 },
//
// "node_key":"IGXCXknWQ1baTa8TZ6rF3kAPZ4\/aTsui"
// }
type submitDistributedQueryResultsRequestShim struct {
	NodeKey  string                     `json:"node_key"`
	Results  map[string]json.RawMessage `json:"queries"`
	Statuses map[string]interface{}     `json:"statuses"`
	Messages map[string]string          `json:"messages"`
}

func (shim *submitDistributedQueryResultsRequestShim) hostNodeKey() string {
	return shim.NodeKey
}

func (shim *submitDistributedQueryResultsRequestShim) toRequest(ctx context.Context) (*SubmitDistributedQueryResultsRequest, error) {
	results := fleet.OsqueryDistributedQueryResults{}
	for query, raw := range shim.Results {
		queryResults := []map[string]string{}
		// No need to handle error because the empty array is what we
		// want if there was an error parsing the JSON (the error
		// indicates that osquery sent us incosistently schemaed JSON)
		_ = json.Unmarshal(raw, &queryResults)
		results[query] = queryResults
	}

	// Statuses were represented by strings in osquery < 3.0 and now
	// integers in osquery > 3.0. Massage to string for compatibility with
	// the service definition.
	statuses := map[string]fleet.OsqueryStatus{}
	for query, status := range shim.Statuses {
		switch s := status.(type) {
		case string:
			sint, err := strconv.Atoi(s)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "parse status to int")
			}
			statuses[query] = fleet.OsqueryStatus(sint)
		case float64:
			statuses[query] = fleet.OsqueryStatus(s)
		default:
			return nil, ctxerr.Errorf(ctx, "query status should be string or number, got %T", s)
		}
	}

	return &SubmitDistributedQueryResultsRequest{
		NodeKey:  shim.NodeKey,
		Results:  results,
		Statuses: statuses,
		Messages: shim.Messages,
	}, nil
}

type SubmitDistributedQueryResultsRequest struct {
	NodeKey  string                               `json:"node_key"`
	Results  fleet.OsqueryDistributedQueryResults `json:"queries"`
	Statuses map[string]fleet.OsqueryStatus       `json:"statuses"`
	Messages map[string]string                    `json:"messages"`
}

type submitDistributedQueryResultsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r submitDistributedQueryResultsResponse) error() error { return r.Err }

func submitDistributedQueryResultsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	shim := request.(*submitDistributedQueryResultsRequestShim)
	req, err := shim.toRequest(ctx)
	if err != nil {
		return submitDistributedQueryResultsResponse{Err: err}, nil
	}

	err = svc.SubmitDistributedQueryResults(ctx, req.Results, req.Statuses, req.Messages)
	if err != nil {
		return submitDistributedQueryResultsResponse{Err: err}, nil
	}
	return submitDistributedQueryResultsResponse{}, nil
}

const (
	// hostLabelQueryPrefix is appended before the query name when a query is
	// provided as a label query. This allows the results to be retrieved when
	// osqueryd writes the distributed query results.
	hostLabelQueryPrefix = "fleet_label_query_"

	// hostDetailQueryPrefix is appended before the query name when a query is
	// provided as a detail query.
	hostDetailQueryPrefix = "fleet_detail_query_"

	// hostAdditionalQueryPrefix is appended before the query name when a query is
	// provided as an additional query (additional info for hosts to retrieve).
	hostAdditionalQueryPrefix = "fleet_additional_query_"

	// hostPolicyQueryPrefix is appended before the query name when a query is
	// provided as a policy query. This allows the results to be retrieved when
	// osqueryd writes the distributed query results.
	hostPolicyQueryPrefix = "fleet_policy_query_"

	// hostNoPoliciesWildcard is a query sent to hosts when it's time to run policy
	// queries on a host, but such host does not have any policies assigned.
	// When Fleet receives results from such query then it will update the host's
	// policy_updated_at column.
	//
	// This is used to prevent hosts without policies assigned to continuously
	// perform lookups in the policies table on every check in.
	hostNoPoliciesWildcard = "fleet_no_policies_wildcard"

	// hostDistributedQueryPrefix is appended before the query name when a query is
	// run from a distributed query campaign
	hostDistributedQueryPrefix = "fleet_distributed_query_"
)

func (svc *Service) SubmitDistributedQueryResults(
	ctx context.Context,
	results fleet.OsqueryDistributedQueryResults,
	statuses map[string]fleet.OsqueryStatus,
	messages map[string]string,
) error {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return newOsqueryError("internal error: missing host from request context")
	}

	detailUpdated := false
	additionalResults := make(fleet.OsqueryDistributedQueryResults)
	additionalUpdated := false
	labelResults := map[uint]*bool{}
	policyResults := map[uint]*bool{}
	refetchCriticalSet := host.RefetchCriticalQueriesUntil != nil

	svc.maybeDebugHost(ctx, host, results, statuses, messages)

	var hostWithoutPolicies bool
	for query, rows := range results {
		// When receiving this query in the results, we will update the host's
		// policy_updated_at column.
		if query == hostNoPoliciesWildcard {
			hostWithoutPolicies = true
			continue
		}

		// osquery docs say any nonzero (string) value for status indicates a query error
		status, ok := statuses[query]
		failed := ok && status != fleet.StatusOK
		if failed && messages[query] != "" && !noSuchTableRegexp.MatchString(messages[query]) {
			ll := level.Debug(svc.logger)
			// We'd like to log these as error for troubleshooting and improving of distributed queries.
			if messages[query] == "distributed query is denylisted" {
				ll = level.Error(svc.logger)
			}
			ll.Log("query", query, "message", messages[query], "hostID", host.ID)
		}

		ingestedDetailUpdated, ingestedAdditionalUpdated, err := svc.ingestQueryResults(
			ctx, query, host, rows, failed, messages, policyResults, labelResults, additionalResults,
		)
		if err != nil {
			logging.WithErr(ctx, ctxerr.New(ctx, "error in query ingestion"))
			logging.WithExtras(ctx, "ingestion-err", err)
		}

		detailUpdated = detailUpdated || ingestedDetailUpdated
		additionalUpdated = additionalUpdated || ingestedAdditionalUpdated
	}

	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting app config")
	}

	if len(labelResults) > 0 {
		if err := svc.task.RecordLabelQueryExecutions(ctx, host, labelResults, svc.clock.Now(), ac.ServerSettings.DeferredSaveHost); err != nil {
			logging.WithErr(ctx, err)
		}
	}

	if len(policyResults) > 0 {

		// filter policy results for webhooks
		var policyIDs []uint
		if ac.WebhookSettings.FailingPoliciesWebhook.Enable {
			policyIDs = append(policyIDs, ac.WebhookSettings.FailingPoliciesWebhook.PolicyIDs...)
		}

		if host.TeamID != nil {
			team, err := svc.ds.Team(ctx, *host.TeamID)
			if err != nil {
				logging.WithErr(ctx, err)
			} else {
				if team.Config.WebhookSettings.FailingPoliciesWebhook.Enable {
					policyIDs = append(policyIDs, team.Config.WebhookSettings.FailingPoliciesWebhook.PolicyIDs...)
				}
			}
		}

		filteredResults := filterPolicyResults(policyResults, policyIDs)
		if len(filteredResults) > 0 {
			if failingPolicies, passingPolicies, err := svc.ds.FlippingPoliciesForHost(ctx, host.ID, filteredResults); err != nil {
				logging.WithErr(ctx, err)
			} else {
				// Register the flipped policies on a goroutine to not block the hosts on redis requests.
				go func() {
					if err := svc.registerFlippedPolicies(ctx, host.ID, host.Hostname, host.DisplayName(), failingPolicies, passingPolicies); err != nil {
						logging.WithErr(ctx, err)
					}
				}()
			}
		}
		// NOTE(mna): currently, failing policies webhook wouldn't see the new
		// flipped policies on the next run if async processing is enabled and the
		// collection has not been done yet (not persisted in mysql). Should
		// FlippingPoliciesForHost take pending redis data into consideration, or
		// maybe we should impose restrictions between async collection interval
		// and policy update interval?

		if err := svc.task.RecordPolicyQueryExecutions(ctx, host, policyResults, svc.clock.Now(), ac.ServerSettings.DeferredSaveHost); err != nil {
			logging.WithErr(ctx, err)
		}
	} else {
		if hostWithoutPolicies {
			// RecordPolicyQueryExecutions called with results=nil will still update the host's policy_updated_at column.
			if err := svc.task.RecordPolicyQueryExecutions(ctx, host, nil, svc.clock.Now(), ac.ServerSettings.DeferredSaveHost); err != nil {
				logging.WithErr(ctx, err)
			}
		}
	}

	if additionalUpdated {
		additionalJSON, err := json.Marshal(additionalResults)
		if err != nil {
			logging.WithErr(ctx, err)
		} else {
			additional := json.RawMessage(additionalJSON)
			if err := svc.ds.SaveHostAdditional(ctx, host.ID, &additional); err != nil {
				logging.WithErr(ctx, err)
			}
		}
	}

	if detailUpdated {
		host.DetailUpdatedAt = svc.clock.Now()
	}

	refetchRequested := host.RefetchRequested
	if refetchRequested {
		host.RefetchRequested = false
	}
	refetchCriticalCleared := refetchCriticalSet && host.RefetchCriticalQueriesUntil == nil

	if refetchRequested || detailUpdated || refetchCriticalCleared {
		appConfig, err := svc.ds.AppConfig(ctx)
		if err != nil {
			logging.WithErr(ctx, err)
		} else {
			if appConfig.ServerSettings.DeferredSaveHost {
				go svc.serialUpdateHost(host)
			} else {
				if err := svc.ds.UpdateHost(ctx, host); err != nil {
					logging.WithErr(ctx, err)
				}
			}
		}
	}

	return nil
}

func (svc *Service) ingestQueryResults(
	ctx context.Context,
	query string,
	host *fleet.Host,
	rows []map[string]string,
	failed bool,
	messages map[string]string,
	policyResults map[uint]*bool,
	labelResults map[uint]*bool,
	additionalResults fleet.OsqueryDistributedQueryResults,
) (bool, bool, error) {
	var detailUpdated, additionalUpdated bool

	// live queries we do want to ingest even if the query had issues, because we want to inform the user of these
	// issues
	// same applies to policies, since it's a 3 state result, one of them being failure, and labels take this state
	// into account as well

	var err error
	switch {
	case strings.HasPrefix(query, hostDistributedQueryPrefix):
		err = svc.ingestDistributedQuery(ctx, *host, query, rows, messages[query])
	case strings.HasPrefix(query, hostPolicyQueryPrefix):
		err = ingestMembershipQuery(hostPolicyQueryPrefix, query, rows, policyResults, failed)
	case strings.HasPrefix(query, hostLabelQueryPrefix):
		err = ingestMembershipQuery(hostLabelQueryPrefix, query, rows, labelResults, failed)
	}

	if failed {
		// if a query failed, and it might be a detailed query or host additional, don't even try to ingest it
		return false, false, err
	}

	switch {
	case strings.HasPrefix(query, hostDetailQueryPrefix):
		trimmedQuery := strings.TrimPrefix(query, hostDetailQueryPrefix)
		var ingested bool
		ingested, err = svc.directIngestDetailQuery(ctx, host, trimmedQuery, rows)
		if !ingested && err == nil {
			err = svc.ingestDetailQuery(ctx, host, trimmedQuery, rows)
			// No err != nil check here because ingestDetailQuery could have updated
			// successfully some values of host.
			detailUpdated = true
		}
	case strings.HasPrefix(query, hostAdditionalQueryPrefix):
		name := strings.TrimPrefix(query, hostAdditionalQueryPrefix)
		additionalResults[name] = rows
		additionalUpdated = true
	}

	return detailUpdated, additionalUpdated, err
}

var noSuchTableRegexp = regexp.MustCompile(`^no such table: \S+$`)

func (svc *Service) directIngestDetailQuery(ctx context.Context, host *fleet.Host, name string, rows []map[string]string) (ingested bool, err error) {
	features, err := svc.HostFeatures(ctx, host)
	if err != nil {
		return false, newOsqueryError("ingest detail query: " + err.Error())
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return false, newOsqueryError("ingest detail query: " + err.Error())
	}

	detailQueries := osquery_utils.GetDetailQueries(ctx, svc.config, appConfig, features)
	query, ok := detailQueries[name]
	if !ok {
		return false, newOsqueryError("unknown detail query " + name)
	}
	if query.DirectIngestFunc != nil {
		err = query.DirectIngestFunc(ctx, svc.logger, host, svc.ds, rows)
		if err != nil {
			return false, newOsqueryError(fmt.Sprintf("ingesting query %s: %s", name, err.Error()))
		}
		return true, nil
	} else if query.DirectTaskIngestFunc != nil {
		err = query.DirectTaskIngestFunc(ctx, svc.logger, host, svc.task, rows)
		if err != nil {
			return false, newOsqueryError(fmt.Sprintf("ingesting query %s: %s", name, err.Error()))
		}
		return true, nil
	}
	return false, nil
}

// ingestDistributedQuery takes the results of a distributed query and modifies the
// provided fleet.Host appropriately.
func (svc *Service) ingestDistributedQuery(ctx context.Context, host fleet.Host, name string, rows []map[string]string, errMsg string) error {
	trimmedQuery := strings.TrimPrefix(name, hostDistributedQueryPrefix)

	campaignID, err := strconv.Atoi(osquery_utils.EmptyToZero(trimmedQuery))
	if err != nil {
		return newOsqueryError("unable to parse campaign ID: " + trimmedQuery)
	}

	// Write the results to the pubsub store
	res := fleet.DistributedQueryResult{
		DistributedQueryCampaignID: uint(campaignID),
		Host: fleet.ResultHostData{
			ID:          host.ID,
			Hostname:    host.Hostname,
			DisplayName: host.DisplayName(),
		},
		Rows: rows,
	}
	if errMsg != "" {
		res.Error = &errMsg
	}

	err = svc.resultStore.WriteResult(res)
	if err != nil {
		var pse pubsub.Error
		ok := errors.As(err, &pse)
		if !ok || !pse.NoSubscriber() {
			return newOsqueryError("writing results: " + err.Error())
		}

		// If there are no subscribers, the campaign is "orphaned"
		// and should be closed so that we don't continue trying to
		// execute that query when we can't write to any subscriber
		campaign, err := svc.ds.DistributedQueryCampaign(ctx, uint(campaignID))
		if err != nil {
			if err := svc.liveQueryStore.StopQuery(strconv.Itoa(campaignID)); err != nil {
				return newOsqueryError("stop orphaned campaign after load failure: " + err.Error())
			}
			return newOsqueryError("loading orphaned campaign: " + err.Error())
		}

		if campaign.CreatedAt.After(svc.clock.Now().Add(-1 * time.Minute)) {
			// Give the client a minute to connect before considering the
			// campaign orphaned.
			//
			// Live queries work in two stages (asynchronous):
			// 	1. The campaign is created by a client. So the target devices checking in
			// 	will start receiving the query corresponding to the campaign.
			//	2. The client (UI/fleetctl) starts listenting for query results.
			//
			// This expected error can happen if:
			//	A. A device checked in and sent results back in between steps (1) and (2).
			// 	B. The client stopped listening in (2) and devices continue to send results back.
			return newOsqueryError(fmt.Sprintf("campaignID=%d waiting for listener", campaignID))
		}

		if campaign.Status != fleet.QueryComplete {
			campaign.Status = fleet.QueryComplete
			if err := svc.ds.SaveDistributedQueryCampaign(ctx, campaign); err != nil {
				return newOsqueryError("closing orphaned campaign: " + err.Error())
			}
		}

		if err := svc.liveQueryStore.StopQuery(strconv.Itoa(campaignID)); err != nil {
			return newOsqueryError("stopping orphaned campaign: " + err.Error())
		}

		// No need to record query completion in this case
		return newOsqueryError(fmt.Sprintf("campaignID=%d stopped", campaignID))
	}

	err = svc.liveQueryStore.QueryCompletedByHost(strconv.Itoa(campaignID), host.ID)
	if err != nil {
		return newOsqueryError("record query completion: " + err.Error())
	}

	return nil
}

// ingestMembershipQuery records the results of label queries run by a host
func ingestMembershipQuery(
	prefix string,
	query string,
	rows []map[string]string,
	results map[uint]*bool,
	failed bool,
) error {
	trimmedQuery := strings.TrimPrefix(query, prefix)
	trimmedQueryNum, err := strconv.Atoi(osquery_utils.EmptyToZero(trimmedQuery))
	if err != nil {
		return fmt.Errorf("converting query from string to int: %w", err)
	}
	// A label/policy query matches if there is at least one result for that
	// query. We must also store negative results.
	if failed {
		results[uint(trimmedQueryNum)] = nil
	} else {
		results[uint(trimmedQueryNum)] = ptr.Bool(len(rows) > 0)
	}

	return nil
}

// ingestDetailQuery takes the results of a detail query and modifies the
// provided fleet.Host appropriately.
func (svc *Service) ingestDetailQuery(ctx context.Context, host *fleet.Host, name string, rows []map[string]string) error {
	features, err := svc.HostFeatures(ctx, host)
	if err != nil {
		return newOsqueryError("ingest detail query: " + err.Error())
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return newOsqueryError("ingest detail query: " + err.Error())
	}

	detailQueries := osquery_utils.GetDetailQueries(ctx, svc.config, appConfig, features)
	query, ok := detailQueries[name]
	if !ok {
		return newOsqueryError("unknown detail query " + name)
	}

	if query.IngestFunc != nil {
		err = query.IngestFunc(ctx, svc.logger, host, rows)
		if err != nil {
			return newOsqueryError(fmt.Sprintf("ingesting query %s: %s", name, err.Error()))
		}
	}

	return nil
}

// filterPolicyResults filters out policies that aren't configured for webhook automation.
func filterPolicyResults(incoming map[uint]*bool, webhookPolicies []uint) map[uint]*bool {
	wp := make(map[uint]struct{})
	for _, policyID := range webhookPolicies {
		wp[policyID] = struct{}{}
	}
	filtered := make(map[uint]*bool)
	for policyID, passes := range incoming {
		if _, ok := wp[policyID]; !ok {
			continue
		}
		filtered[policyID] = passes
	}
	return filtered
}

func (svc *Service) registerFlippedPolicies(ctx context.Context, hostID uint, hostname, displayName string, newFailing, newPassing []uint) error {
	host := fleet.PolicySetHost{
		ID:          hostID,
		Hostname:    hostname,
		DisplayName: displayName,
	}
	for _, policyID := range newFailing {
		if err := svc.failingPolicySet.AddHost(policyID, host); err != nil {
			return err
		}
	}
	for _, policyID := range newPassing {
		if err := svc.failingPolicySet.RemoveHosts(policyID, []fleet.PolicySetHost{host}); err != nil {
			return err
		}
	}
	return nil
}

func (svc *Service) maybeDebugHost(
	ctx context.Context,
	host *fleet.Host,
	results fleet.OsqueryDistributedQueryResults,
	statuses map[string]fleet.OsqueryStatus,
	messages map[string]string,
) {
	if svc.debugEnabledForHost(ctx, host.ID) {
		hlogger := log.With(svc.logger, "host-id", host.ID)

		logJSON(hlogger, host, "host")
		logJSON(hlogger, results, "results")
		logJSON(hlogger, statuses, "statuses")
		logJSON(hlogger, messages, "messages")
	}
}

////////////////////////////////////////////////////////////////////////////////
// Submit Logs
////////////////////////////////////////////////////////////////////////////////

type submitLogsRequest struct {
	NodeKey string          `json:"node_key"`
	LogType string          `json:"log_type"`
	Data    json.RawMessage `json:"data"`
}

func (r *submitLogsRequest) hostNodeKey() string {
	return r.NodeKey
}

type submitLogsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r submitLogsResponse) error() error { return r.Err }

func submitLogsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*submitLogsRequest)

	var err error
	switch req.LogType {
	case "status":
		var statuses []json.RawMessage
		if err := json.Unmarshal(req.Data, &statuses); err != nil {
			err = newOsqueryError("unmarshalling status logs: " + err.Error())
			break
		}

		err = svc.SubmitStatusLogs(ctx, statuses)
		if err != nil {
			break
		}

	case "result":
		var results []json.RawMessage
		if err := json.Unmarshal(req.Data, &results); err != nil {
			err = newOsqueryError("unmarshalling result logs: " + err.Error())
			break
		}
		err = svc.SubmitResultLogs(ctx, results)
		if err != nil {
			break
		}

	default:
		err = newOsqueryError("unknown log type: " + req.LogType)
	}

	return submitLogsResponse{Err: err}, nil
}

func (svc *Service) SubmitStatusLogs(ctx context.Context, logs []json.RawMessage) error {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	if err := svc.osqueryLogWriter.Status.Write(ctx, logs); err != nil {
		return newOsqueryError("error writing status logs: " + err.Error())
	}
	return nil
}

func (svc *Service) SubmitResultLogs(ctx context.Context, logs []json.RawMessage) error {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	if err := svc.osqueryLogWriter.Result.Write(ctx, logs); err != nil {
		return newOsqueryError("error writing result logs: " + err.Error())
	}
	return nil
}
