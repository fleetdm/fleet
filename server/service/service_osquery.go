package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	hostctx "github.com/kolide/kolide-ose/server/contexts/host"
	"github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
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

func (svc service) AuthenticateHost(ctx context.Context, nodeKey string) (*kolide.Host, error) {
	if nodeKey == "" {
		return nil, osqueryError{
			message:     "authentication error: missing node key",
			nodeInvalid: true,
		}
	}
	host, err := svc.ds.AuthenticateHost(nodeKey)
	if err != nil {
		return nil, osqueryError{
			message:     "authentication error: " + err.Error(),
			nodeInvalid: true,
		}
	}
	return host, nil
}

func (svc service) EnrollAgent(ctx context.Context, enrollSecret, hostIdentifier string) (string, error) {
	if enrollSecret != svc.config.Osquery.EnrollSecret {
		return "", osqueryError{message: "invalid enroll secret", nodeInvalid: true}
	}

	host, err := svc.ds.EnrollHost(hostIdentifier, "", "", "", svc.config.Osquery.NodeKeySize)
	if err != nil {
		return "", osqueryError{message: "enrollment failed: " + err.Error(), nodeInvalid: true}
	}

	return host.NodeKey, nil
}

func (svc service) GetClientConfig(ctx context.Context) (*kolide.OsqueryConfig, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, osqueryError{message: "internal error: missing host from request context"}
	}

	config := &kolide.OsqueryConfig{
		Options: kolide.OsqueryOptions{
			PackDelimiter:      "/",
			DisableDistributed: false,
		},
		Packs: kolide.Packs{},
	}

	packs, err := svc.ListPacksForHost(ctx, host.ID)
	if err != nil {
		return nil, osqueryError{message: "database error: " + err.Error()}
	}

	for _, pack := range packs {
		// first, we must figure out what queries are in this pack
		queries, err := svc.ds.ListQueriesInPack(pack)
		if err != nil {
			return nil, osqueryError{message: "database error: " + err.Error()}
		}

		// the serializable osquery config struct expects content in a
		// particular format, so we do the conversion here
		configQueries := kolide.Queries{}
		for _, query := range queries {
			configQueries[query.Name] = kolide.QueryContent{
				Query:    query.Query,
				Interval: query.Interval,
				Platform: query.Platform,
				Version:  query.Version,
				Snapshot: query.Snapshot,
			}
		}

		// finally, we add the pack to the client config struct with all of
		// the packs queries
		config.Packs[pack.Name] = kolide.PackContent{
			Platform: pack.Platform,
			Queries:  configQueries,
		}
	}

	return config, nil
}

func (svc service) SubmitStatusLogs(ctx context.Context, logs []kolide.OsqueryStatusLog) error {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return osqueryError{message: "internal error: missing host from request context"}
	}

	for _, log := range logs {
		err := json.NewEncoder(svc.osqueryStatusLogWriter).Encode(log)
		if err != nil {
			return errors.NewFromError(err, http.StatusInternalServerError, "error writing status log")
		}
	}

	err := svc.ds.MarkHostSeen(&host, svc.clock.Now())
	if err != nil {
		return osqueryError{message: "failed to update host seen: " + err.Error()}
	}

	return nil
}

func (svc service) SubmitResultLogs(ctx context.Context, logs []kolide.OsqueryResultLog) error {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return osqueryError{message: "internal error: missing host from request context"}
	}

	for _, log := range logs {
		err := json.NewEncoder(svc.osqueryResultLogWriter).Encode(log)
		if err != nil {
			return errors.NewFromError(err, http.StatusInternalServerError, "error writing result log")
		}
	}

	err := svc.ds.MarkHostSeen(&host, svc.clock.Now())
	if err != nil {
		return osqueryError{message: "failed to update host seen: " + err.Error()}
	}

	return nil
}

// hostLabelQueryPrefix is appended before the query name when a query is
// provided as a label query. This allows the results to be retrieved when
// osqueryd writes the distributed query results.
const hostLabelQueryPrefix = "kolide_label_query_"

// hostDetailQueryPrefix is appended before the query name when a query is
// provided as a detail query.
const hostDetailQueryPrefix = "kolide_detail_query_"

// hostDistributedQueryPrefix is appended before the query name when a query is
// run from a distributed query campaign
const hostDistributedQueryPrefix = "kolide_distributed_query_"

// detailQueries defines the detail queries that should be run on the host, as
// well as how the results of those queries should be ingested into the
// kolide.Host data model. This map should not be modified at runtime.
var detailQueries = map[string]struct {
	Query      string
	IngestFunc func(host *kolide.Host, rows []map[string]string) error
}{
	"osquery_info": {
		Query: "select * from osquery_info limit 1",
		IngestFunc: func(host *kolide.Host, rows []map[string]string) error {
			if len(rows) != 1 {
				return osqueryError{
					message: fmt.Sprintf("expected 1 row but got %d", len(rows)),
				}
			}

			host.Platform = rows[0]["build_platform"]
			host.OsqueryVersion = rows[0]["version"]

			return nil
		},
	},
	"system_info": {
		Query: "select * from system_info limit 1",
		IngestFunc: func(host *kolide.Host, rows []map[string]string) error {
			if len(rows) != 1 {
				return osqueryError{
					message: fmt.Sprintf("expected 1 row but got %d", len(rows)),
				}
			}

			var err error
			host.PhysicalMemory, err = strconv.Atoi(rows[0]["physical_memory"])
			if err != nil {
				return err
			}
			host.HostName = rows[0]["hostname"]
			host.UUID = rows[0]["uuid"]

			return nil
		},
	},
	"os_version": {
		Query: "select * from os_version limit 1",
		IngestFunc: func(host *kolide.Host, rows []map[string]string) error {
			if len(rows) != 1 {
				return osqueryError{
					message: fmt.Sprintf("expected 1 row but got %d", len(rows)),
				}
			}

			host.OSVersion = fmt.Sprintf(
				"%s %s.%s.%s",
				rows[0]["name"],
				rows[0]["major"],
				rows[0]["minor"],
				rows[0]["patch"],
			)

			return nil
		},
	},
	"uptime": {
		Query: "select * from uptime limit 1",
		IngestFunc: func(host *kolide.Host, rows []map[string]string) error {
			if len(rows) != 1 {
				return osqueryError{
					message: fmt.Sprintf("expected 1 row but got %d", len(rows)),
				}
			}

			uptimeSeconds, err := strconv.Atoi(rows[0]["total_seconds"])
			if err != nil {
				return err
			}
			host.Uptime = time.Duration(uptimeSeconds) * time.Second

			return nil
		},
	},
	"network_interface": {
		Query: `select * from interface_details id join interface_addresses ia
                        on ia.interface = id.interface where broadcast != ""
                        order by (ibytes + obytes) desc limit 1`,
		IngestFunc: func(host *kolide.Host, rows []map[string]string) error {
			if len(rows) != 1 {
				return osqueryError{
					message: fmt.Sprintf("expected 1 row but got %d", len(rows)),
				}
			}

			host.PrimaryMAC = rows[0]["mac"]
			host.PrimaryIP = rows[0]["address"]

			return nil
		},
	},
}

// detailUpdateInterval determines how often the detail queries should be
// updated
const detailUpdateInterval = 1 * time.Hour

// hostDetailQueries returns the map of queries that should be executed by
// osqueryd to fill in the host details
func (svc service) hostDetailQueries(host kolide.Host) map[string]string {
	queries := make(map[string]string)
	if host.DetailUpdateTime.After(svc.clock.Now().Add(-detailUpdateInterval)) {
		// No need to update already fresh details
		return queries
	}

	for name, query := range detailQueries {
		queries[hostDetailQueryPrefix+name] = query.Query
	}
	return queries
}

func (svc service) GetDistributedQueries(ctx context.Context) (map[string]string, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, osqueryError{message: "internal error: missing host from request context"}
	}

	queries := svc.hostDetailQueries(host)

	// Retrieve the label queries that should be updated
	cutoff := svc.clock.Now().Add(-svc.config.Osquery.LabelUpdateInterval)
	labelQueries, err := svc.ds.LabelQueriesForHost(&host, cutoff)
	if err != nil {
		return nil, err
	}

	for name, query := range labelQueries {
		queries[hostLabelQueryPrefix+name] = query
	}

	distributedQueries, err := svc.ds.DistributedQueriesForHost(&host)
	if err != nil {
		return nil, osqueryError{message: "retrieving query campaigns: " + err.Error()}
	}

	for id, query := range distributedQueries {
		queries[hostDistributedQueryPrefix+strconv.Itoa(int(id))] = query
	}

	return queries, nil
}

// ingestDetailQuery takes the results of a detail query and modifies the
// provided kolide.Host appropriately.
func (svc service) ingestDetailQuery(host *kolide.Host, name string, rows []map[string]string) error {
	trimmedQuery := strings.TrimPrefix(name, hostDetailQueryPrefix)
	query, ok := detailQueries[trimmedQuery]
	if !ok {
		return osqueryError{message: "unknown detail query " + trimmedQuery}
	}

	err := query.IngestFunc(host, rows)
	if err != nil {
		return osqueryError{
			message: fmt.Sprintf("ingesting query %s: %s", name, err.Error()),
		}
	}
	return nil
}

// ingestLabelQuery records the results of label queries run by a host
func (svc service) ingestLabelQuery(host kolide.Host, query string, rows []map[string]string, results map[string]bool) error {
	trimmedQuery := strings.TrimPrefix(query, hostLabelQueryPrefix)
	// A label query matches if there is at least one result for that
	// query. We must also store negative results.
	results[trimmedQuery] = len(rows) > 0
	return nil
}

// ingestDistributedQuery takes the results of a distributed query and modifies the
// provided kolide.Host appropriately.
func (svc service) ingestDistributedQuery(host kolide.Host, name string, rows []map[string]string) error {
	trimmedQuery := strings.TrimPrefix(name, hostDistributedQueryPrefix)

	campaignID, err := strconv.Atoi(trimmedQuery)
	if err != nil {
		return osqueryError{message: "unable to parse campaign ID: " + trimmedQuery}
	}

	// Write the results to the pubsub store
	res := kolide.DistributedQueryResult{
		DistributedQueryCampaignID: uint(campaignID),
		Host: host,
		Rows: rows,
	}

	err = svc.resultStore.WriteResult(res)
	if err != nil {
		return osqueryError{message: "writing results: " + err.Error()}
	}

	// Record execution of the query
	exec := &kolide.DistributedQueryExecution{
		HostID: host.ID,
		DistributedQueryCampaignID: uint(campaignID),
		Status: kolide.ExecutionSucceeded,
	}

	_, err = svc.ds.NewDistributedQueryExecution(exec)
	if err != nil {
		return osqueryError{message: "recording execution: " + err.Error()}
	}

	return nil
}

func (svc service) SubmitDistributedQueryResults(ctx context.Context, results kolide.OsqueryDistributedQueryResults) error {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return osqueryError{message: "internal error: missing host from request context"}
	}

	err := svc.ds.MarkHostSeen(&host, svc.clock.Now())
	if err != nil {
		return osqueryError{message: "failed to update host seen: " + err.Error()}
	}

	labelResults := map[string]bool{}
	for query, rows := range results {
		switch {
		case strings.HasPrefix(query, hostDetailQueryPrefix):
			err = svc.ingestDetailQuery(&host, query, rows)

		case strings.HasPrefix(query, hostLabelQueryPrefix):
			err = svc.ingestLabelQuery(host, query, rows, labelResults)

		case strings.HasPrefix(query, hostDistributedQueryPrefix):
			err = svc.ingestDistributedQuery(host, query, rows)

		default:
			err = osqueryError{message: "unknown query prefix: " + query}
		}

		if err != nil {
			return osqueryError{message: "failed to ingest result: " + err.Error()}
		}

	}

	if len(labelResults) > 0 {
		err = svc.ds.RecordLabelQueryExecutions(&host, labelResults, svc.clock.Now())
		if err != nil {
			return osqueryError{message: "failed to save labels: " + err.Error()}
		}
	}

	host.DetailUpdateTime = svc.clock.Now()
	err = svc.ds.SaveHost(&host)
	if err != nil {
		return osqueryError{message: "failed to update host details: " + err.Error()}
	}

	return nil
}
