package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/osquery_utils"
	"github.com/throttled/throttled/v2"

	"github.com/fleetdm/fleet/v4/server/fleet"

	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/pubsub"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
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

var counter = int64(0)

func (svc Service) AuthenticateHost(ctx context.Context, nodeKey string) (*fleet.Host, bool, error) {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	if nodeKey == "" {
		return nil, false, osqueryError{
			message:     "authentication error: missing node key",
			nodeInvalid: true,
		}
	}

	host, err := svc.ds.AuthenticateHost(ctx, nodeKey)
	if err != nil {
		root := ctxerr.Cause(err)
		switch root.(type) {
		case fleet.NotFoundError:
			return nil, false, osqueryError{
				message:     "authentication error: invalid node key: " + nodeKey,
				nodeInvalid: true,
			}
		default:
			return nil, false, osqueryError{
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

	return host, svc.debugEnabledForHost(ctx, host), nil
}

func (svc Service) debugEnabledForHost(ctx context.Context, host *fleet.Host) bool {
	hlogger := log.With(svc.logger, "host-id", host.ID)
	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		level.Debug(hlogger).Log("err", ctxerr.Wrap(ctx, err, "getting app config for host debug"))
		return false
	}

	doDebug := false
	for _, hostID := range ac.ServerSettings.DebugHostIDs {
		if host.ID == hostID {
			doDebug = true
			break
		}
	}
	return doDebug
}

func (svc Service) EnrollAgent(ctx context.Context, enrollSecret, hostIdentifier string, hostDetails map[string](map[string]string)) (string, error) {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	logging.WithExtras(ctx, "hostIdentifier", hostIdentifier)

	secret, err := svc.ds.VerifyEnrollSecret(ctx, enrollSecret)
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

	host, err := svc.ds.EnrollHost(ctx, hostIdentifier, nodeKey, secret.TeamID, svc.config.Osquery.EnrollCooldown)
	if err != nil {
		return "", osqueryError{message: "save enroll failed: " + err.Error(), nodeInvalid: true}
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return "", osqueryError{message: "save enroll failed: " + err.Error(), nodeInvalid: true}
	}
	// Save enrollment details if provided
	detailQueries := osquery_utils.GetDetailQueries(appConfig)
	save := false
	if r, ok := hostDetails["os_version"]; ok {
		err := detailQueries["os_version"].IngestFunc(svc.logger, host, []map[string]string{r})
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "Ingesting os_version")
		}
		save = true
	}
	if r, ok := hostDetails["osquery_info"]; ok {
		err := detailQueries["osquery_info"].IngestFunc(svc.logger, host, []map[string]string{r})
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "Ingesting osquery_info")
		}
		save = true
	}
	if r, ok := hostDetails["system_info"]; ok {
		err := detailQueries["system_info"].IngestFunc(svc.logger, host, []map[string]string{r})
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "Ingesting system_info")
		}
		save = true
	}
	if save {
		if appConfig.ServerSettings.DeferredSaveHost {
			go svc.serialSaveHost(host)
		} else {
			err = svc.ds.SaveHost(ctx, host)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "save host in enroll agent")
			}
		}
	}

	return nodeKey, nil
}

func (svc Service) serialSaveHost(host *fleet.Host) {
	newVal := atomic.AddInt64(&counter, 1)
	defer func() {
		atomic.AddInt64(&counter, -1)
	}()
	level.Debug(svc.logger).Log("background", newVal)

	ctx, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFunc()
	err := svc.ds.SerialSaveHost(ctx, host)
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

func (svc *Service) GetClientConfig(ctx context.Context) (map[string]interface{}, error) {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, osqueryError{message: "internal error: missing host from request context"}
	}

	baseConfig, err := svc.AgentOptionsForHost(ctx, &host)
	if err != nil {
		return nil, osqueryError{message: "internal error: fetch base config: " + err.Error()}
	}

	config := make(map[string]interface{})

	if baseConfig != nil {
		err = json.Unmarshal(baseConfig, &config)
		if err != nil {
			return nil, osqueryError{message: "internal error: parse base configuration: " + err.Error()}
		}
	}

	packs, err := svc.ds.ListPacksForHost(ctx, host.ID)
	if err != nil {
		return nil, osqueryError{message: "database error: " + err.Error()}
	}

	packConfig := fleet.Packs{}
	for _, pack := range packs {
		// first, we must figure out what queries are in this pack
		queries, err := svc.ds.ListScheduledQueriesInPack(ctx, pack.ID, fleet.ListOptions{})
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
		appConfig, err := svc.ds.AppConfig(ctx)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get app config on get client config")
		}
		if appConfig.ServerSettings.DeferredSaveHost {
			go svc.serialSaveHost(&host)
		} else {
			err = svc.ds.SaveHost(ctx, &host)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "save host in get client config")
			}
		}
	}

	return config, nil
}

func (svc *Service) SubmitStatusLogs(ctx context.Context, logs []json.RawMessage) error {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	if err := svc.osqueryLogWriter.Status.Write(ctx, logs); err != nil {
		return osqueryError{message: "error writing status logs: " + err.Error()}
	}
	return nil
}

func (svc *Service) SubmitResultLogs(ctx context.Context, logs []json.RawMessage) error {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

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

// hostPolicyQueryPrefix is appended before the query name when a query is
// provided as a policy query. This allows the results to be retrieved when
// osqueryd writes the distributed query results.
const hostPolicyQueryPrefix = "fleet_policy_query_"

// hostDistributedQueryPrefix is appended before the query name when a query is
// run from a distributed query campaign
const hostDistributedQueryPrefix = "fleet_distributed_query_"

// detailQueriesForHost returns the map of detail+additional queries that should be executed by
// osqueryd to fill in the host details.
func (svc *Service) detailQueriesForHost(ctx context.Context, host fleet.Host) (map[string]string, bool, error) {
	shouldUpdate, rateLimited, err := svc.shouldUpdate(ctx,
		host.RefetchRequested,
		host.DetailUpdatedAt,
		svc.config.Osquery.DetailUpdateInterval,
		host.DistributedInterval,
		"detailQueries",
	)
	if err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "should update for detail queries")
	}
	if !shouldUpdate {
		return nil, rateLimited, nil
	}

	config, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "read app config")
	}

	queries := make(map[string]string)
	detailQueries := osquery_utils.GetDetailQueries(config)
	for name, query := range detailQueries {
		if query.RunsForPlatform(host.Platform) {
			queries[hostDetailQueryPrefix+name] = query.Query
		}
	}

	if config.HostSettings.AdditionalQueries == nil {
		// No additional queries set
		return queries, false, nil
	}

	var additionalQueries map[string]string
	if err := json.Unmarshal(*config.HostSettings.AdditionalQueries, &additionalQueries); err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "unmarshal additional queries")
	}

	for name, query := range additionalQueries {
		queries[hostAdditionalQueryPrefix+name] = query
	}

	return queries, false, nil
}

// shouldUpdate returns whether a query group should be sent to a host or not.
//
// queriesName must to be one of "detailQueries", "policyQueries" and "labelQueries",
// otherwise shouldUpdate panics.
func (svc *Service) shouldUpdate(
	ctx context.Context,
	refetchRequested bool,
	lastUpdated time.Time,
	queriesInterval time.Duration,
	distributedIntervalSecs uint,
	queriesName string,
) (update bool, rateLimited bool, err error) {
	if refetchRequested {
		return true, false, nil
	}
	var jitter time.Duration
	if svc.config.Osquery.MaxJitterPercent > 0 {
		maxJitter := time.Duration(svc.config.Osquery.MaxJitterPercent) * queriesInterval / time.Duration(100.0)
		randDuration, err := rand.Int(rand.Reader, big.NewInt(int64(maxJitter)))
		if err == nil {
			jitter = time.Duration(randDuration.Int64())
		}
	}
	cutoff := svc.clock.Now().Add(-(queriesInterval + jitter))
	if lastUpdated.After(cutoff) {
		return false, false, nil
	}

	config, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return false, false, ctxerr.Wrap(ctx, err, "read app config")
	}
	if config.HostSettings.RateLimitDisabled {
		return true, false, nil
	}

	rateLimit, ok := queriesRateLimit[queriesName]
	if !ok {
		panic(fmt.Sprintf("unknown rate limit config: %s", queriesName))
	}

	rate, err := svc.updateRateLimit(ctx, rateLimit, queriesName, queriesInterval)
	if err != nil {
		return false, false, ctxerr.Wrap(ctx, err)
	}

	// Log if rate-limiting operating is taking considerable time (e.g. due to Redis performance).
	defer func(start time.Time) {
		if took := time.Since(start); took > 2*time.Second {
			level.Info(svc.logger).Log("msg", "high-rate-limiting-latency", "took", took)
		}
	}(time.Now())

	limiter, err := throttled.NewGCRARateLimiter(
		svc.limitStore,
		throttled.RateQuota{
			MaxRate:  throttled.PerSec(rate),
			MaxBurst: rate,
		},
	)
	if err != nil {
		// If rate-limiting fails we log the error and "allow the request".
		level.Error(svc.logger).Log("op", "NewGCRARateLimiter", "err", err)
		return true, false, nil
	}
	limited, _, err := limiter.RateLimit(queriesName, 1)
	if err != nil {
		// If rate-limiting fails we log the error and "allow the request".
		level.Error(svc.logger).Log("op", "RateLimit", "err", err)
		return true, false, nil
	}

	return !limited, limited, nil
}

// rateForOnline attempts to calculate a rate that allows all the online
// hosts to perform their update over a single interval.
func rateForOnline(onlineCount int, interval time.Duration) int {
	rate := onlineCount/int(interval.Seconds()) + 1
	const minRate = 10
	if rate < minRate {
		rate = minRate
	}
	return rate
}

// updateRateLimit updates the rate every 5 seconds depending on number of online hosts.
func (svc *Service) updateRateLimit(
	ctx context.Context,
	rateLimit *rateLimit,
	queriesName string,
	queriesInterval time.Duration,
) (int, error) {
	rateLimit.Lock()
	defer rateLimit.Unlock()

	if time.Since(rateLimit.lastCheck) < 5*time.Second {
		return rateLimit.rate, nil
	}

	rateLimit.lastCheck = time.Now()

	totalCount, err := svc.ds.PlainCountHosts(ctx)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "get online host counts for queries")
	}

	rateLimit.rate = rateForOnline(totalCount, queriesInterval)

	level.Info(svc.logger).Log(
		"queries", queriesName,
		"interval", queriesInterval,
		"online", totalCount,
		"rate", rateLimit.rate,
	)
	return rateLimit.rate, nil
}

func (svc *Service) labelQueriesForHost(ctx context.Context, host *fleet.Host) (map[string]string, bool, error) {
	labelReportedAt := svc.task.GetHostLabelReportedAt(ctx, host)
	shouldUpdate, rateLimited, err := svc.shouldUpdate(ctx,
		host.RefetchRequested,
		labelReportedAt,
		svc.config.Osquery.LabelUpdateInterval,
		host.DistributedInterval,
		"labelQueries",
	)
	if err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "should update for label queries")
	}
	if !shouldUpdate {
		return nil, rateLimited, nil
	}
	labelQueries, err := svc.ds.LabelQueriesForHost(ctx, host)
	if err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "retrieve label queries")
	}
	return labelQueries, false, nil
}

func (svc *Service) policyQueriesForHost(ctx context.Context, host *fleet.Host) (map[string]string, bool, error) {
	shouldUpdate, rateLimited, err := svc.shouldUpdate(ctx,
		host.RefetchRequested,
		host.PolicyUpdatedAt,
		svc.config.Osquery.PolicyUpdateInterval,
		host.DistributedInterval,
		"policyQueries",
	)
	if err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "should update for policy queries")
	}
	if !shouldUpdate {
		return nil, rateLimited, nil
	}
	policyQueries, err := svc.ds.PolicyQueriesForHost(ctx, host)
	if err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "retrieve policy queries")
	}
	return policyQueries, false, nil
}

func (svc *Service) GetDistributedQueries(ctx context.Context) (map[string]string, uint, error) {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, 0, osqueryError{message: "internal error: missing host from request context"}
	}

	queries := make(map[string]string)

	detailQueries, detailRateLimited, err := svc.detailQueriesForHost(ctx, host)
	if err != nil {
		return nil, 0, osqueryError{message: err.Error()}
	}
	for name, query := range detailQueries {
		queries[name] = query
	}

	labelQueries, labelRateLimited, err := svc.labelQueriesForHost(ctx, &host)
	if err != nil {
		return nil, 0, osqueryError{message: err.Error()}
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

	policyQueries, policyRateLimited, err := svc.policyQueriesForHost(ctx, &host)
	if err != nil {
		return nil, 0, osqueryError{message: err.Error()}
	}
	for name, query := range policyQueries {
		queries[hostPolicyQueryPrefix+name] = query
	}

	accelerate := uint(0)
	// Assume this host is just enrolling, and accelerate checkins
	// (to allow for platform restricted labels to run quickly
	// after platform is retrieved from details)
	if host.Hostname == "" || host.Platform == "" ||
		// If any of the detail, label or policies hit a rate limit we also enable accelerate.
		(detailRateLimited || labelRateLimited || policyRateLimited) {
		accelerate = 10
	}

	return queries, accelerate, nil
}

// ingestDetailQuery takes the results of a detail query and modifies the
// provided fleet.Host appropriately.
func (svc *Service) ingestDetailQuery(ctx context.Context, host *fleet.Host, name string, rows []map[string]string) error {
	trimmedQuery := strings.TrimPrefix(name, hostDetailQueryPrefix)

	config, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return osqueryError{message: "ingest detail query: " + err.Error()}
	}

	detailQueries := osquery_utils.GetDetailQueries(config)
	query, ok := detailQueries[trimmedQuery]
	if !ok {
		return osqueryError{message: "unknown detail query " + trimmedQuery}
	}

	err = query.IngestFunc(svc.logger, host, rows)
	if err != nil {
		return osqueryError{
			message: fmt.Sprintf("ingesting query %s: %s", name, err.Error()),
		}
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

// ingestDistributedQuery takes the results of a distributed query and modifies the
// provided fleet.Host appropriately.
func (svc *Service) ingestDistributedQuery(ctx context.Context, host fleet.Host, name string, rows []map[string]string, failed bool, errMsg string) error {
	trimmedQuery := strings.TrimPrefix(name, hostDistributedQueryPrefix)

	campaignID, err := strconv.Atoi(osquery_utils.EmptyToZero(trimmedQuery))
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
		var pse pubsub.Error
		ok := errors.As(err, &pse)
		if !ok || !pse.NoSubscriber() {
			return osqueryError{message: "writing results: " + err.Error()}
		}

		// If there are no subscribers, the campaign is "orphaned"
		// and should be closed so that we don't continue trying to
		// execute that query when we can't write to any subscriber
		campaign, err := svc.ds.DistributedQueryCampaign(ctx, uint(campaignID))
		if err != nil {
			if err := svc.liveQueryStore.StopQuery(strconv.Itoa(campaignID)); err != nil {
				return osqueryError{message: "stop orphaned campaign after load failure: " + err.Error()}
			}
			return osqueryError{message: "loading orphaned campaign: " + err.Error()}
		}

		if campaign.CreatedAt.After(svc.clock.Now().Add(-1 * time.Minute)) {
			// Give the client a minute to connect before considering the
			// campaign orphaned
			return osqueryError{message: "campaign waiting for listener (please retry)"}
		}

		if campaign.Status != fleet.QueryComplete {
			campaign.Status = fleet.QueryComplete
			if err := svc.ds.SaveDistributedQueryCampaign(ctx, campaign); err != nil {
				return osqueryError{message: "closing orphaned campaign: " + err.Error()}
			}
		}

		if err := svc.liveQueryStore.StopQuery(strconv.Itoa(campaignID)); err != nil {
			return osqueryError{message: "stopping orphaned campaign: " + err.Error()}
		}

		// No need to record query completion in this case
		return osqueryError{message: "campaign stopped"}
	}

	err = svc.liveQueryStore.QueryCompletedByHost(strconv.Itoa(campaignID), host.ID)
	if err != nil {
		return osqueryError{message: "record query completion: " + err.Error()}
	}

	return nil
}

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
		return osqueryError{message: "internal error: missing host from request context"}
	}

	// Check for host details queries and if so, load host additional.
	// If we don't do this, we will end up unintentionally dropping
	// any existing host additional info.
	for query := range results {
		if strings.HasPrefix(query, hostDetailQueryPrefix) {
			fullHost, err := svc.ds.Host(ctx, host.ID, true)
			if err != nil {
				// leave this error return here, we don't want to drop host additionals
				// if we can't get a host, everything is lost
				return osqueryError{message: "internal error: load host additional: " + err.Error()}
			}
			host = *fullHost
			break
		}
	}

	var err error
	detailUpdated := false // Whether detail or additional was updated
	additionalResults := make(fleet.OsqueryDistributedQueryResults)
	additionalUpdated := false
	labelResults := map[uint]*bool{}
	policyResults := map[uint]*bool{}
	for query, rows := range results {
		// osquery docs say any nonzero (string) value for status indicates a query error
		status, ok := statuses[query]
		failed := ok && status != fleet.StatusOK
		switch {
		case strings.HasPrefix(query, hostDetailQueryPrefix):
			err = svc.ingestDetailQuery(ctx, &host, query, rows)
			detailUpdated = true
		case strings.HasPrefix(query, hostAdditionalQueryPrefix):
			name := strings.TrimPrefix(query, hostAdditionalQueryPrefix)
			additionalResults[name] = rows
			additionalUpdated = true
		case strings.HasPrefix(query, hostLabelQueryPrefix):
			err = ingestMembershipQuery(hostLabelQueryPrefix, query, rows, labelResults, failed)
		case strings.HasPrefix(query, hostPolicyQueryPrefix):
			err = ingestMembershipQuery(hostPolicyQueryPrefix, query, rows, policyResults, failed)
		case strings.HasPrefix(query, hostDistributedQueryPrefix):
			err = svc.ingestDistributedQuery(ctx, host, query, rows, failed, messages[query])

		default:
			err = osqueryError{message: "unknown query prefix: " + query}
		}

		if err != nil {
			logging.WithErr(ctx, ctxerr.New(ctx, "error in live query ingestion"))
			logging.WithExtras(ctx, "ingestion-err", err)
		}
	}

	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting app config")
	}

	if len(labelResults) > 0 {
		if ac.ServerSettings.DeferredSaveHost {
			if err := svc.ds.RecordLabelQueryExecutions(ctx, &host, labelResults, svc.clock.Now(), true); err != nil {
				logging.WithErr(ctx, err)
			}
		} else {
			if err := svc.task.RecordLabelQueryExecutions(ctx, &host, labelResults, svc.clock.Now()); err != nil {
				logging.WithErr(ctx, err)
			}
		}
	}

	if len(policyResults) > 0 {
		host.PolicyUpdatedAt = svc.clock.Now()
		err = svc.ds.RecordPolicyQueryExecutions(ctx, &host, policyResults, svc.clock.Now(), ac.ServerSettings.DeferredSaveHost)
		if err != nil {
			logging.WithErr(ctx, err)
		}
	}

	if detailUpdated || additionalUpdated {
		host.Modified = true
		host.DetailUpdatedAt = svc.clock.Now()
	}

	if additionalUpdated {
		additionalJSON, err := json.Marshal(additionalResults)
		if err != nil {
			logging.WithErr(ctx, err)
		} else {
			additional := json.RawMessage(additionalJSON)
			host.Additional = &additional
		}
	}

	svc.maybeDebugHost(ctx, host, results, statuses, messages)

	if host.RefetchRequested {
		host.RefetchRequested = false
		host.Modified = true
	}

	if host.Modified {
		appConfig, err := svc.ds.AppConfig(ctx)
		if err != nil {
			logging.WithErr(ctx, err)
		} else {
			if appConfig.ServerSettings.DeferredSaveHost {
				go svc.serialSaveHost(&host)
			} else {
				err = svc.ds.SaveHost(ctx, &host)
				if err != nil {
					logging.WithErr(ctx, err)
				}
			}
		}
	}

	return nil
}

func (svc *Service) maybeDebugHost(
	ctx context.Context,
	host fleet.Host,
	results fleet.OsqueryDistributedQueryResults,
	statuses map[string]fleet.OsqueryStatus,
	messages map[string]string,
) {
	if svc.debugEnabledForHost(ctx, &host) {
		hlogger := log.With(svc.logger, "host-id", host.ID)

		logJSON(hlogger, host, "host")
		logJSON(hlogger, results, "results")
		logJSON(hlogger, statuses, "statuses")
		logJSON(hlogger, messages, "messages")
	}
}
