package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/service/osquery_utils"
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

	appConfig, err := svc.ds.AppConfig()
	if err != nil {
		return "", osqueryError{message: "save enroll failed: " + err.Error(), nodeInvalid: true}
	}
	// Save enrollment details if provided
	detailQueries := osquery_utils.GetDetailQueries(appConfig)
	save := false
	if r, ok := hostDetails["os_version"]; ok {
		err := detailQueries["os_version"].IngestFunc(svc.logger, host, []map[string]string{r})
		if err != nil {
			return "", errors.Wrap(err, "Ingesting os_version")
		}
		save = true
	}
	if r, ok := hostDetails["osquery_info"]; ok {
		err := detailQueries["osquery_info"].IngestFunc(svc.logger, host, []map[string]string{r})
		if err != nil {
			return "", errors.Wrap(err, "Ingesting osquery_info")
		}
		save = true
	}
	if r, ok := hostDetails["system_info"]; ok {
		err := detailQueries["system_info"].IngestFunc(svc.logger, host, []map[string]string{r})
		if err != nil {
			return "", errors.Wrap(err, "Ingesting system_info")
		}
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
	logging.WithLevel(
		logging.WithExtras(
			logging.WithNoUser(ctx),
			append(extras, "ip_addr", remoteAddr, "x_for_ip_addr", xForwardedFor)...),
		level.Debug,
	)
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

// hostDetailQueries returns the map of queries that should be executed by
// osqueryd to fill in the host details
func (svc *Service) hostDetailQueries(host fleet.Host) (map[string]string, error) {
	queries := make(map[string]string)
	if host.DetailUpdatedAt.After(svc.clock.Now().Add(-svc.config.Osquery.DetailUpdateInterval)) && !host.RefetchRequested {
		// No need to update already fresh details
		return queries, nil
	}
	config, err := svc.ds.AppConfig()
	if err != nil {
		return nil, osqueryError{message: "get additional queries: " + err.Error()}
	}

	detailQueries := osquery_utils.GetDetailQueries(config)
	for name, query := range detailQueries {
		if query.RunsForPlatform(host.Platform) {
			queries[hostDetailQueryPrefix+name] = query.Query
		}
	}

	// Get additional queries
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

	config, err := svc.ds.AppConfig()
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

	// Refetch is no longer needed after ingesting details.
	host.RefetchRequested = false

	return nil
}

// ingestLabelQuery records the results of label queries run by a host
func (svc *Service) ingestLabelQuery(host fleet.Host, query string, rows []map[string]string, results map[uint]bool) error {
	trimmedQuery := strings.TrimPrefix(query, hostLabelQueryPrefix)
	trimmedQueryNum, err := strconv.Atoi(osquery_utils.EmptyToZero(trimmedQuery))
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

	// Check for host details queries and if so, load host additional.
	// If we don't do this, we will end up unintentionally dropping
	// any existing host additional info.
	for query := range results {
		if strings.HasPrefix(query, hostDetailQueryPrefix) {
			fullHost, err := svc.ds.Host(host.ID)
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
			failed := ok && status != fleet.StatusOK
			err = svc.ingestDistributedQuery(host, query, rows, failed, messages[query])
		default:
			err = osqueryError{message: "unknown query prefix: " + query}
		}

		if err != nil {
			logging.WithExtras(ctx, "ingestion-err", err)
		}
	}

	if len(labelResults) > 0 {
		host.Modified = true
		host.LabelUpdatedAt = svc.clock.Now()
		err = svc.ds.RecordLabelQueryExecutions(&host, labelResults, svc.clock.Now())
		if err != nil {
			logging.WithErr(ctx, err)
		}
	}

	if detailUpdated {
		host.Modified = true
		host.DetailUpdatedAt = svc.clock.Now()
		additionalJSON, err := json.Marshal(additionalResults)
		if err != nil {
			logging.WithErr(ctx, err)
		} else {
			additional := json.RawMessage(additionalJSON)
			host.Additional = &additional
		}
	}

	if host.Modified {
		err = svc.ds.SaveHost(&host)
		if err != nil {
			logging.WithErr(ctx, err)
		}
	}

	return nil
}
