package service

import (
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/fleetdm/fleet/v4/ee/server/service/hostidentity/httpsig"
	"github.com/fleetdm/fleet/v4/server"
	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/agentws"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/pubsub"
	"github.com/fleetdm/fleet/v4/server/service/conditional_access_microsoft_proxy"
	"github.com/fleetdm/fleet/v4/server/service/contract"
	"github.com/fleetdm/fleet/v4/server/service/osquery_utils"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/spf13/cast"
	"golang.org/x/exp/slices"
)

func newOsqueryErrorWithInvalidNode(msg string) *OsqueryError {
	return NewOsqueryError(msg, true)
}

func newOsqueryError(msg string) *OsqueryError {
	return NewOsqueryError(msg, false)
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
		// Fall back to the orbit node key: with the WebSocket transport active,
		// orbit calls the distributed endpoints on behalf of osquery and only
		// has its own node key. Both keys resolve to the same host row, so
		// authorization is unchanged. The fallback runs only on an osquery-key
		// miss and only with the transport enabled, keeping legacy auth
		// semantics (and the single-query hot path) intact otherwise.
		if !svc.config.WebSocket.TransportEnabled {
			return nil, false, newOsqueryErrorWithInvalidNode("authentication error: invalid node key")
		}
		host, err = svc.ds.LoadHostByOrbitNodeKey(ctx, nodeKey)
		switch {
		case err == nil:
			// OK
		case fleet.IsNotFound(err):
			return nil, false, newOsqueryErrorWithInvalidNode("authentication error: invalid node key")
		case errors.Is(err, context.Canceled):
			return nil, false, err
		default:
			return nil, false, newOsqueryError("authentication error: " + err.Error())
		}
	case errors.Is(err, context.Canceled):
		// Most likely client disconnected, so we treat this as a client error.
		return nil, false, err
	default:
		return nil, false, newOsqueryError("authentication error: " + err.Error())
	}

	if *host.HasHostIdentityCert {
		err = httpsig.VerifyHostIdentity(ctx, svc.ds, host)
		if err != nil {
			osqueryError := newOsqueryError("authentication error: " + err.Error())
			osqueryError.StatusCode = http.StatusUnauthorized
			return nil, false, osqueryError
		}
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

func enrollAgentEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*contract.EnrollOsqueryAgentRequest)
	nodeKey, err := svc.EnrollOsquery(ctx, req.EnrollSecret, req.HostIdentifier, req.HostDetails)
	if err != nil {
		return contract.EnrollOsqueryAgentResponse{Err: err}, nil
	}
	return contract.EnrollOsqueryAgentResponse{NodeKey: nodeKey}, nil
}

func (svc *Service) EnrollOsquery(ctx context.Context, enrollSecret, hostIdentifier string, hostDetails map[string](map[string]string)) (string, error) {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	logging.WithLevel(logging.WithExtras(ctx, "hostIdentifier", hostIdentifier), slog.LevelInfo)

	secret, err := svc.ds.VerifyEnrollSecret(ctx, enrollSecret)
	if err != nil {
		return "", newOsqueryErrorWithInvalidNode("enroll failed: " + err.Error())
	}

	identityCert, err := svc.ds.GetHostIdentityCertByName(ctx, hostIdentifier)
	if err != nil && !fleet.IsNotFound(err) {
		return "", fleet.OrbitError{Message: fmt.Sprintf("loading certificate: %s", err.Error())}
	}

	// If an identity certificate exists for this host, make sure the request had an HTTP message signature with the matching certificate.
	hostIdentityCert, httpSigPresent := httpsig.FromContext(ctx)
	if identityCert != nil {
		if !httpSigPresent {
			return "", fleet.NewAuthFailedError("authentication error: missing HTTP signature")
		}
		if identityCert.SerialNumber != hostIdentityCert.SerialNumber {
			return "", fleet.NewAuthFailedError("authentication error: certificate serial number mismatch")
		}
	} else if httpSigPresent { // but we couldn't find cert in DB
		return "", fleet.NewAuthFailedError("authentication error: certificate matching HTTP message signature not found")
	}

	nodeKey, err := server.GenerateRandomText(svc.config.Osquery.NodeKeySize)
	if err != nil {
		return "", newOsqueryErrorWithInvalidNode("generate node key failed: " + err.Error())
	}

	hostIdentifier = getHostIdentifier(ctx, svc.logger, svc.config.Osquery.HostIdentifier, hostIdentifier, hostDetails)
	canEnroll, err := svc.enrollHostLimiter.CanEnrollNewHost(ctx)
	if err != nil {
		return "", newOsqueryErrorWithInvalidNode("can enroll host check failed: " + err.Error())
	}
	if !canEnroll {
		deviceCount := "unknown"
		if lic, _ := license.FromContext(ctx); lic != nil {
			deviceCount = strconv.Itoa(lic.GetDeviceCount())
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

	host, err := svc.ds.EnrollOsquery(ctx,
		fleet.WithEnrollOsqueryMDMEnabled(appConfig.MDM.EnabledAndConfigured),
		fleet.WithEnrollOsqueryHostID(hostIdentifier),
		fleet.WithEnrollOsqueryHardwareUUID(hardwareUUID),
		fleet.WithEnrollOsqueryHardwareSerial(hardwareSerial),
		fleet.WithEnrollOsqueryNodeKey(nodeKey),
		fleet.WithEnrollOsqueryTeamID(secret.TeamID),
		fleet.WithEnrollOsqueryCooldown(svc.config.Osquery.EnrollCooldown),
		fleet.WithEnrollOsqueryIdentityCert(identityCert),
	)
	if err != nil {
		return "", newOsqueryErrorWithInvalidNode("save enroll failed: " + err.Error())
	}

	features, err := svc.HostFeatures(ctx, host)
	if err != nil {
		return "", newOsqueryErrorWithInvalidNode("host features load failed: " + err.Error())
	}

	// Save enrollment details if provided
	detailQueries := osquery_utils.GetDetailQueries(
		ctx,
		svc.config,
		appConfig,
		features,
		osquery_utils.Integrations{
			ConditionalAccessMicrosoft: false, // here we are just using a few ingestion functions, so no need to set.
		}, nil, // Ok ... the following queries do not need the Team's MDM config
	)
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
			go svc.serialUpdateHost(ctx, host)
		} else {
			if err := svc.ds.UpdateHost(ctx, host); err != nil {
				return "", ctxerr.Wrap(ctx, err, "save host in enroll agent")
			}
		}
	}

	return nodeKey, nil
}

var counter = int64(0)

func (svc *Service) serialUpdateHost(ctx context.Context, host *fleet.Host) {
	newVal := atomic.AddInt64(&counter, 1)
	defer func() {
		atomic.AddInt64(&counter, -1)
	}()
	// Detach from request cancellation but preserve context values (e.g. OTEL trace),
	// then apply a timeout for this background operation.
	ctx, cancelFunc := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancelFunc()
	svc.logger.DebugContext(ctx, "serial update host background", "background", newVal)
	err := svc.ds.SerialUpdateHost(ctx, host)
	if err != nil {
		svc.logger.ErrorContext(ctx, "serial update host background error", "err", err)
	}
}

func getHostIdentifier(ctx context.Context, logger *slog.Logger, identifierOption, providedIdentifier string, details map[string](map[string]string)) string {
	switch identifierOption {
	case "provided":
		// Use the host identifier already provided in the request.
		return providedIdentifier

	case "instance":
		r, ok := details["osquery_info"]
		if !ok { //nolint:gocritic // ignore ifElseChain
			logger.InfoContext(ctx, "could not get host identifier",
				"reason", "missing osquery_info",
				"identifier", "instance",
			)
		} else if r["instance_id"] == "" {
			logger.InfoContext(ctx, "could not get host identifier",
				"reason", "missing instance_id in osquery_info",
				"identifier", "instance",
			)
		} else {
			return r["instance_id"]
		}

	case "uuid":
		r, ok := details["osquery_info"]
		if !ok { //nolint:gocritic // ignore ifElseChain
			logger.InfoContext(ctx, "could not get host identifier",
				"reason", "missing osquery_info",
				"identifier", "uuid",
			)
		} else if r["uuid"] == "" {
			logger.InfoContext(ctx, "could not get host identifier",
				"reason", "missing instance_id in osquery_info",
				"identifier", "uuid",
			)
		} else {
			return r["uuid"]
		}

	case "hostname":
		r, ok := details["system_info"]
		if !ok { //nolint:gocritic // ignore ifElseChain
			logger.InfoContext(ctx, "could not get host identifier",
				"reason", "missing system_info",
				"identifier", "hostname",
			)
		} else if r["hostname"] == "" {
			logger.InfoContext(ctx, "could not get host identifier",
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
	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		svc.logger.DebugContext(ctx, "getting app config for host debug", "host-id", id, "err", ctxerr.Wrap(ctx, err, "getting app config for host debug"))
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
	// ETag is the body-carried conditional-request validator (see the
	// GetClientConfigWithETag interface docs). nil means the agent did not
	// send the field and has not opted in; an empty string means the agent
	// opted in but holds no validator yet (its first request). The field is
	// decoded from the body even in header-auth mode, where only node_key is
	// ignored.
	ETag *string `json:"etag"`
}

func (r *getClientConfigRequest) hostNodeKey() string {
	return r.NodeKey
}

func (getClientConfigRequest) DecodeRequest(
	ctx context.Context,
	r *http.Request,
) (any, error) {
	req := new(getClientConfigRequest)
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		return nil, err
	}
	return req, nil
}

// configUnchangedBody is the constant response for an agent whose etag
// matches the current config: the reserved value "ok" tells the agent its
// config is current. It is never used as a real validator.
const configUnchangedBody = `{"etag":"ok"}`

type getClientConfigResponse struct {
	// Config is NOT populated on the live request path anymore: the endpoint
	// renders the pre-marshaled body via HijackRender below. Config and the
	// success branch of MarshalJSON exist only for tests and for UnmarshalJSON
	// (client-side decoding of a config response).
	Config      map[string]any `json:"-"`
	body        []byte
	notModified bool
	Err         error `json:"error,omitempty"`
}

func (r getClientConfigResponse) Error() error { return r.Err }

// MarshalJSON implements json.Marshaler.
//
// Osquery expects the response for configs to be at the
// top-level of the JSON response.
//
// On the live request path only the error branch is reachable (the platform
// encoder checks Error() before HijackRender, and HijackRender writes r.body
// directly, bypassing this method). The success branch serves tests that
// round-trip Config.
func (r getClientConfigResponse) MarshalJSON() ([]byte, error) {
	if r.Err != nil {
		return json.Marshal(struct {
			Error string `json:"error,omitempty"`
		}{Error: r.Err.Error()})
	}
	return marshalClientConfig(r.Config)
}

// UnmarshalJSON implements json.Unmarshaler.
//
// Osquery expects the response for configs to be at the
// top-level of the JSON response.
func (r *getClientConfigResponse) UnmarshalJSON(data []byte) error {
	r.Config = make(map[string]any)
	return json.Unmarshal(data, &r.Config)
}

func (r getClientConfigResponse) HijackRender(
	ctx context.Context,
	w http.ResponseWriter,
) {
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	body := r.body
	if r.notModified {
		body = []byte(configUnchangedBody)
	}
	if _, err := w.Write(body); err != nil {
		logging.WithErr(ctx, err)
	}
}

// marshalClientConfig serializes the config map to JSON using the same
// encoder settings as the existing jsonMarshal path (two-space indent,
// trailing newline from json.Encoder.Encode).
func marshalClientConfig(config map[string]any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(config); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// clientConfigETag computes the SHA-256 validator over the canonical
// (etag-less) config body. The value is opaque to agents and carried in the
// JSON bodies, not HTTP headers, so it uses bare hex — which also can never
// collide with the reserved "ok" value.
func clientConfigETag(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// clientConfigETagMatches reports whether the agent's body-carried etag
// matches the current validator. A nil clientETag means the agent did not
// opt in; an empty one is the opt-in signal from an agent with no stored
// validator. Neither can match, so the "unchanged" response is never sent
// to an agent without history.
func clientConfigETagMatches(clientETag *string, etag string) bool {
	return clientETag != nil && *clientETag != "" && *clientETag == etag
}

func getClientConfigEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getClientConfigRequest)

	// GetClientConfigWithETag may answer without building the config at all;
	// see its interface docs in server/fleet/service.go for the contract.
	result, err := svc.GetClientConfigWithETag(ctx, req.ETag)
	if err != nil {
		return getClientConfigResponse{Err: err}, nil
	}

	// Per-request diagnostics are debug-only because this endpoint is the
	// highest-volume route in Fleet; the Prometheus counters in
	// server/service/redis_config_etag carry the aggregate view.
	logging.WithLevel(ctx, slog.LevelDebug)
	logging.WithExtras(ctx, "etag_result", result.CacheStatus, "etag_mode", result.Mode)

	return getClientConfigResponse{
		body:        result.Body,
		notModified: result.NotModified,
	}, nil
}

func (svc *Service) getScheduledQueries(ctx context.Context, teamID *uint) (fleet.Queries, error) {
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load app config")
	}

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, newOsqueryError("internal error: missing host from request context")
	}

	queries, err := svc.ds.ListScheduledQueriesForAgents(ctx, teamID, &host.ID, appConfig.ServerSettings.QueryReportsDisabled)
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

// packConfigCacheKey returns a cache key for the pack config cache
// keyed by (teamID, queryReportsDisabled).
func packConfigCacheKey(teamID *uint, queryReportsDisabled bool) string {
	tid := "global"
	if teamID != nil {
		tid = fmt.Sprintf("%d", *teamID)
	}
	qrd := "0"
	if queryReportsDisabled {
		qrd = "1"
	}
	return "pack_config:" + tid + ":" + qrd
}

// getPackConfig returns the marshaled pack config JSON for the host. It uses
// a cache for hosts without legacy packs and without label-scoped queries,
// keyed by (teamID, queryReportsDisabled). The cache is nil when
// osquery.config_in_memory_cache is disabled, which makes every call
// build from the database.
//
// bypassTeamPackCache When true, the team-keyed packConfigCache is
// neither read NOR written. Per-host cache mode (label-scoped reports in the
// host's effective scope) requires this: the team-keyed cache stores ONE
// host's label-filtered render and serves it team-wide (#48702's documented
// limitation), so in label-scoped scopes its content is structurally wrong
// for other hosts — a per-host ETag derived from it would be poisoned by
// construction, invisible to every invalidation mechanism. This bypass
// prevents systematic cross-host wrongness; it is not defense against a rare
// race.
// packs are the host's legacy (2017) packs, which make the config host-specific,
// so its ETag must never reach the team-shared store.
func (svc *Service) getPackConfig(ctx context.Context, host *fleet.Host, packs []*fleet.Pack, bypassTeamPackCache bool) (raw json.RawMessage, err error) {
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "fetch app config")
	}
	queryReportsDisabled := appConfig.ServerSettings.QueryReportsDisabled

	// Fast path: if no legacy packs and no label-scoped queries, try the cached pack config.
	// The scheduled queries pack config is identical for all hosts in the
	// same team ONLY when no queries have label targeting. When labels are
	// involved, ListScheduledQueriesForAgents filters per host, so the
	// result varies per host and cannot be cached at the team level.
	useLegacyPacks := len(packs) > 0
	canUseCache := !useLegacyPacks && !bypassTeamPackCache && svc.packConfigCache != nil
	if canUseCache {
		// Check (with caching) whether any scheduled queries have label targeting.
		// This is cached separately from the pack config itself to avoid a DB
		// query on every request for the common case (no label-scoped queries).
		// Note: if labels are added to a query mid-cache, the stale "false" entry
		// lets the pack config cache serve the old team-wide result until the TTL
		// expires. This is the same staleness window as any other query change
		// (1 minute default) and is an accepted trade-off to avoid explicit
		// invalidation across the datastore/service boundary.
		labelCacheKey := "has_label_scoped:" + packConfigCacheKey(host.TeamID, queryReportsDisabled)
		if cached, found := svc.packConfigCache.Get(labelCacheKey); found {
			if hasLabels, ok := cached.(bool); ok && hasLabels {
				canUseCache = false
			}
		} else {
			hasLabelScoped, err := svc.ds.HasLabelScopedScheduledQueries(ctx, host.TeamID, queryReportsDisabled)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "check label-scoped scheduled queries")
			}
			svc.packConfigCache.SetDefault(labelCacheKey, hasLabelScoped)
			if hasLabelScoped {
				canUseCache = false
			}
		}
	}
	if canUseCache {
		cacheKey := packConfigCacheKey(host.TeamID, queryReportsDisabled)
		if cached, found := svc.packConfigCache.Get(cacheKey); found {
			// cached may be nil (negative cache: no queries for this team)
			// or a json.RawMessage with the marshaled pack config.
			cachedRaw, _ := cached.(json.RawMessage)
			return cachedRaw, nil
		}
	}

	// Cache miss, label-scoped queries present, or legacy packs: build pack config from DB.
	packConfig := fleet.Packs{}

	for _, pack := range packs {
		queries, err := svc.ds.ListScheduledQueriesInPack(ctx, pack.ID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "list scheduled queries in pack")
		}

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

		packConfig[pack.Name] = fleet.PackContent{
			Platform: pack.Platform,
			Queries:  configQueries,
		}
	}

	globalQueries, err := svc.getScheduledQueries(ctx, nil)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get global scheduled queries")
	}
	if len(globalQueries) > 0 {
		packConfig["Global"] = fleet.PackContent{
			Queries: globalQueries,
		}
	}

	if host.TeamID != nil {
		teamQueries, err := svc.getScheduledQueries(ctx, host.TeamID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get team scheduled queries")
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
			return nil, ctxerr.Wrap(ctx, err, "marshal pack config")
		}
		raw = json.RawMessage(packJSON)
	}

	// Cache the result (including empty) for future requests (only when safe
	// to cache: no legacy packs, no label-scoped queries in scope, and the
	// caller did not require a per-host-correct build).
	if canUseCache {
		cacheKey := packConfigCacheKey(host.TeamID, queryReportsDisabled)
		svc.packConfigCache.SetDefault(cacheKey, raw)
	}

	return raw, nil
}

// GetClientConfig always performs a full config build (it never consults the
// Redis ETag store). It remains the entry point for the launcher (gRPC)
// service. The osquery HTTP endpoint uses GetClientConfigWithETag instead.
func (svc *Service) GetClientConfig(ctx context.Context) (map[string]any, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, newOsqueryError("internal error: missing host from request context")
	}
	packs, err := svc.ds.ListPacksForHost(ctx, host.ID)
	if err != nil {
		return nil, newOsqueryError("internal error: list packs for host: " + err.Error())
	}
	return svc.buildClientConfig(ctx, packs, false)
}

// buildClientConfig performs the full osquery config build: agent options +
// pack config + host intervals reconciliation.
//
// SIDE-EFFECT NOTICE Anything added to this function (or anything it
// calls) does NOT run when GetClientConfigWithETag serves a not-modified
// response from the Redis ETag short circuit. A side effect that must run on
// every config check-in belongs in GetClientConfigWithETag BEFORE its
// fast-path return, not here. (The existing UpdateHostOsqueryIntervals
// reconciliation below is safe to skip on a match: intervals only drift when
// the config content changes, and a matching etag proves the host already
// received the current config — the full response that delivered it
// performed the reconciliation. Agents echo the etag of the last config
// RECEIVED, not applied; a host stuck failing to apply a config surfaces
// that loudly on its own logs and refresh status, not on this endpoint.)
//
// bypassTeamPackCache must be true for per-host cache-mode builds — see the
// notice on getPackConfig.
func (svc *Service) buildClientConfig(ctx context.Context, packs []*fleet.Pack, bypassTeamPackCache bool) (config map[string]any, err error) {
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

	config = make(map[string]any)
	if baseConfig != nil {
		err = json.Unmarshal(baseConfig, &config)
		if err != nil {
			return nil, newOsqueryError("internal error: parse base configuration: " + err.Error())
		}
		if config == nil {
			// Unmarshaling the JSON literal `null` (e.g. agent options with
			// "config": null) sets the map to nil rather than leaving it empty.
			// Re-initialize so later assignments (e.g. config["packs"]) don't
			// panic with "assignment to entry in nil map".
			config = make(map[string]any)
		}
	}

	// With the WebSocket transport enabled, orbit points osquery at its own
	// distributed plugin on the command line. Fleet's default agent options
	// include `distributed_plugin: tls` as a config option, which osquery
	// applies at runtime and which would silently flip the host back to TLS
	// polling on its first config refresh — strip it. Agents get their
	// distributed plugin from the fleetd-managed command line either way.
	if svc.config.WebSocket.TransportEnabled {
		if opts, ok := config["options"].(map[string]any); ok {
			delete(opts, "distributed_plugin")
		}
	}

	packConfigJSON, err := svc.getPackConfig(ctx, host, packs, bypassTeamPackCache)
	if err != nil {
		return nil, newOsqueryError("internal error: build pack config: " + err.Error())
	}
	if packConfigJSON != nil {
		config["packs"] = packConfigJSON
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

// clientConfigETagScope returns the Redis ETag scope for a host: "global" for
// hosts with no team (fleet), "team:<id>" otherwise. Together with the host's
// platform this identifies the config representation — the rendered config is
// identical for every non-legacy-pack host in the same (team, platform) pair,
// which is the same fact the packConfigCache relies on.
func clientConfigETagScope(host *fleet.Host) string {
	if host.TeamID != nil {
		return fmt.Sprintf("team:%d", *host.TeamID)
	}
	return "global"
}

// GetClientConfigWithETag implements the ETag-aware config path; the contract
// is on fleet.OsqueryService and the design in server/service/redis_config_etag.
//
// The part to keep in mind while editing: on a short-circuit hit this returns
// before buildClientConfig runs, so nothing below is guaranteed to execute on a
// check-in. See the side-effect notice on buildClientConfig.
//
// Every failure mode degrades to a full build. Gate state that cannot be read
// is treated as bypass rather than guessed, because guessing "shared" would
// publish one host's config under a key its teammates read.
func (svc *Service) GetClientConfigWithETag(ctx context.Context, clientETag *string) (*fleet.ClientConfigResult, error) {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, newOsqueryError("internal error: missing host from request context")
	}

	// ESCAPE HATCH osquery.config_etags=false disables conditional
	// requests entirely: the agent's etag field is ignored (as if never
	// sent), the response never carries an "etag" key or the "unchanged"
	// body, and no etag store I/O happens — byte-identical to the
	// pre-feature behavior for every agent. Distinct from
	// osquery.redis_config_etags, which only disables the Redis short
	// circuit and leaves the protocol active.
	store := svc.configETagStore
	if !svc.config.Osquery.ConfigETags {
		clientETag = nil
		store = nil
	}
	scope := clientConfigETagScope(host)

	// Cache-mode selection from the two cached gate answers. Their loaders
	// (below) are the only DB load the short circuit machinery performs, at
	// most once per few minutes per cluster.
	// labelScopesUnknown is set when the deployment has no legacy packs but the
	// label-scope state could not be read or loaded. In that state the
	// deployment MAY have label-scoped reports, which makes the team-keyed
	// pack cache host-incorrect (see getPackConfig) — so the full build
	// below must bypass it even though the request stays in bypass mode (no
	// Redis record reads/writes with unknown state). Cost: one
	// pre-#48702-cost build for the few requests that hit gate errors or
	// leader-election contention. This branch is unreachable during a full
	// Redis outage — the legacy gate fails first and plain bypass (with the
	// team cache, i.e. exact baseline behavior) applies.
	labelScopesUnknown := false
	mode := fleet.ConfigETagModeOff
	if store != nil {
		mode = fleet.ConfigETagModeBypass
		legacyPresent, err := store.LegacyPacksPresent(ctx, svc.userPacksExist)
		switch {
		case errors.Is(err, fleet.ErrConfigETagGateLoading):
			// Another request on this instance is loading the gate state:
			// normal contention, not a fault. Bypass for this request
			// without waiting and without error logging (see the store's
			// leader-election docs).
		case err != nil:
			// FAIL OPEN: unknown gate state bypasses the short circuit —
			// costing performance, never correctness.
			svc.logConfigETagError(ctx, "config etag: legacy packs gate unavailable; bypassing short circuit", err)
		case !legacyPresent:
			scopes, err := store.LabelScopes(ctx, svc.labelScopedReportScopes)
			switch {
			case errors.Is(err, fleet.ErrConfigETagGateLoading):
				// normal contention: bypass silently, as above — but the
				// build must be per-host correct (see labelScopesUnknown).
				labelScopesUnknown = true
			case err != nil:
				svc.logConfigETagError(ctx, "config etag: label scope state unavailable; bypassing short circuit", err)
				labelScopesUnknown = true
			case scopes.PerHostMode(host.TeamID):
				mode = fleet.ConfigETagModeHost
			default:
				mode = fleet.ConfigETagModeShared
			}
		}
		// Bounded state log: once per Fleet container, on first observation.
		svc.configETagStateOnce.Do(func() {
			svc.logger.InfoContext(ctx, "config etag optimization state first observed",
				"component", "config-etag", "mode", mode, "scope", scope)
		})
	}

	// THE SHORT CIRCUIT One Redis MGET; zero database reads on a hit.
	// Gated on a non-empty client etag: an agent that did not opt in (nil)
	// or holds no validator yet ("") always gets a full build, and can never
	// be answered "unchanged". The store != nil guard is technically implied
	// (mode can only be shared/host when a store was selected above) but is
	// stated here so the invariant is local — for nilaway, and for anyone
	// who later reorders the mode selection.
	if store != nil && clientETag != nil && *clientETag != "" {
		switch mode {
		case fleet.ConfigETagModeShared:
			storedETag, valid, err := store.GetETagIfCurrent(ctx, scope, host.Platform)
			switch {
			case err != nil:
				// FAIL OPEN: fall through to the full build.
				svc.logConfigETagError(ctx, "config etag: redis read failed; falling back to full config build", err)
			case valid && clientConfigETagMatches(clientETag, storedETag):
				return &fleet.ClientConfigResult{
					ETag:        storedETag,
					NotModified: true,
					CacheStatus: fleet.ConfigETagStatusRedisNotModified,
					Mode:        mode,
				}, nil
			}
		case fleet.ConfigETagModeHost:
			// GetHostETagIfCurrent validates generation, stored scope, and stored
			// platform against the authenticated host context — a team
			// transfer or platform change reads as a miss.
			storedETag, valid, err := store.GetHostETagIfCurrent(ctx, host.ID, scope, host.Platform)
			switch {
			case err != nil:
				svc.logConfigETagError(ctx, "config etag: redis host read failed; falling back to full config build", err)
			case valid && clientConfigETagMatches(clientETag, storedETag):
				return &fleet.ClientConfigResult{
					ETag:        storedETag,
					NotModified: true,
					CacheStatus: fleet.ConfigETagStatusRedisHostNotModified,
					Mode:        mode,
				}, nil
			}
		}
		// miss / stale generation / validator mismatch: full build below.
	}

	// Full build. In per-host mode the team-keyed pack cache is BYPASSED:
	// its content is one host's label-filtered render served team-wide, so a
	// per-host record derived from it could bind this host to another host's
	// config — poisoning that no invalidation mechanism can see. The bypass
	// also applies when the label-scope state is unknown (labelScopesUnknown):
	// the deployment may have label-scoped reports, so the cached render may
	// be host-incorrect for this host. Shared mode and plain bypass keep the
	// pre-existing build path, in-memory caches and all.
	packs, err := svc.ds.ListPacksForHost(ctx, host.ID)
	if err != nil {
		return nil, newOsqueryError("internal error: list packs for host: " + err.Error())
	}
	usedLegacyPacks := len(packs) > 0

	config, err := svc.buildClientConfig(ctx, packs, mode == fleet.ConfigETagModeHost || labelScopesUnknown)
	if err != nil {
		return nil, err
	}
	body, err := marshalClientConfig(config)
	if err != nil {
		return nil, newOsqueryError("internal error: encode config: " + err.Error())
	}
	etag := clientConfigETag(body)

	// usedLegacyPacks is checked here, not just in mode selection, because the
	// legacy gate is cached for minutes and can be stale: if THIS build saw
	// legacy packs, its config is host-specific in ways even a per-host record
	// does not model, so it must never be published.
	if store != nil && !usedLegacyPacks {
		// Only shared/host modes publish; bypass and off never touch Redis, so
		// the absence of a case is the "nothing to publish" path.
		switch mode {
		case fleet.ConfigETagModeShared:
			stored, publishErr := store.SetIfNoFence(ctx, scope, host.Platform, etag)
			svc.recordETagPublish(ctx, stored, publishErr)
		case fleet.ConfigETagModeHost:
			stored, publishErr := store.SetHostIfNoFence(ctx, host.ID, scope, host.Platform, etag)
			svc.recordETagPublish(ctx, stored, publishErr)
		}
	}

	// Even without the short circuit, honor the validator against the
	// just-built body (this is the pre-existing bandwidth-only
	// naive-not-modified path: the config was built, but the response body
	// shrinks to the constant "unchanged" form).
	notModified := clientConfigETagMatches(clientETag, etag)
	cacheStatus := fleet.ConfigETagStatusFullMismatch
	switch {
	case notModified:
		cacheStatus = fleet.ConfigETagStatusNotModified
	case clientETag == nil || *clientETag == "":
		cacheStatus = fleet.ConfigETagStatusFullNoValidator
	}
	result := &fleet.ClientConfigResult{
		ETag:        etag,
		NotModified: notModified,
		CacheStatus: cacheStatus,
		Mode:        mode,
	}
	if !notModified {
		// An opted-in agent receives the config with the validator added under
		// the "etag" key; an agent that never sent the field receives the
		// canonical body. The validator is always computed over the etag-less
		// body — the representation the agent applies after stripping the key —
		// so the re-marshal happens after hashing.
		result.Body = body
		if clientETag != nil {
			config["etag"] = etag
			bodyWithETag, err := marshalClientConfig(config)
			if err != nil {
				return nil, newOsqueryError("internal error: encode config with etag: " + err.Error())
			}
			result.Body = bodyWithETag
		}
	}
	return result, nil
}

// userPacksExist is the loader for the legacy (2017) packs gate — the hard
// deployment-wide bypass. ListPacks (without IncludeSystemPacks) broadly
// matches packs whose pack_type is NULL or empty — deliberately wider than
// ListPacksForHost's strict `pack_type IS NULL`, because for this gate
// over-matching only costs the optimization while under-matching could let a
// host's stale etag match past a legacy pack change. Errors report as
// present (fail toward bypassing the optimization).
func (svc *Service) userPacksExist(ctx context.Context) (bool, error) {
	packs, err := svc.ds.ListPacks(ctx, fleet.PackListOptions{ListOptions: fleet.ListOptions{PerPage: 1}})
	if err != nil {
		return true, ctxerr.Wrap(ctx, err, "list user packs for config etag gate")
	}
	return len(packs) > 0, nil
}

// labelScopedReportScopes is the loader for the label-scope mode state: one
// deployment-level query returning which scopes (global, team IDs) contain
// label-scoped scheduled reports. Label-scoped reports make
// ListScheduledQueriesForAgents filter per host, so configs in those scopes
// are NOT identical across a (team, platform) pair and drift with label
// membership — hence per-host mode there.
func (svc *Service) labelScopedReportScopes(ctx context.Context) (fleet.ConfigETagLabelScopes, error) {
	scopes, err := svc.ds.LabelScopedScheduledQueryScopes(ctx)
	if err != nil {
		return fleet.ConfigETagLabelScopes{}, ctxerr.Wrap(ctx, err, "list label scoped report scopes for config etag mode")
	}
	return scopes, nil
}

// recordETagPublish logs the outcome of an ETag publication attempt as the
// etag_publish debug field. Publication failing is never visible to the agent
// — it only costs the optimization.
func (svc *Service) recordETagPublish(ctx context.Context, stored bool, err error) {
	switch {
	case err != nil:
		svc.logConfigETagError(ctx, "config etag: redis write failed", err)
		logging.WithExtras(ctx, "etag_publish", "error")
	case !stored:
		// Fence or quarantine suppression: normal after a recent mutation.
		logging.WithExtras(ctx, "etag_publish", "suppressed")
	default:
		logging.WithExtras(ctx, "etag_publish", "stored")
	}
}

// logConfigETagError logs config-ETag Redis/gate errors at most once per 30
// seconds per Fleet instance. The fast path fails open, so during a Redis
// outage every config request would otherwise emit an error line at check-in
// volume.
func (svc *Service) logConfigETagError(ctx context.Context, msg string, err error) {
	if svc.configETagErrLast == nil {
		svc.logger.ErrorContext(ctx, msg, "component", "config-etag", "err", err)
		return
	}
	const minInterval = 30 // seconds
	now := time.Now().Unix()
	last := svc.configETagErrLast.Load()
	if now-last >= minInterval && svc.configETagErrLast.CompareAndSwap(last, now) {
		svc.logger.ErrorContext(ctx, msg, "component", "config-etag", "err", err)
	}
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
		return nil, ctxerr.Wrap(ctx, err, "load app config")
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

func (r getDistributedQueriesResponse) Error() error { return r.Err }

// recordDistributedReadStats wraps the distributed/read endpoint to count
// requests per host in the agent WebSocket hub, split by request path:
// osqueryd's built-in tls plugin polls the /api/v1/... alias, orbit's
// WebSocket-driven client uses /api/osquery/... — the split makes hosts that
// are still polling visible on /debug/agentws.
func recordDistributedReadStats(
	hub *agentws.Hub,
	next func(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error),
) func(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	return func(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
		if host, ok := hostctx.FromContext(ctx); ok {
			path, _ := ctx.Value(kithttp.ContextKeyRequestPath).(string)
			hub.RecordDistributedRead(host.ID, strings.HasPrefix(path, "/api/v1/"))
		}
		return next(ctx, request, svc)
	}
}

func getDistributedQueriesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
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
		svc.logger.ErrorContext(ctx, "QueriesForHost", "err", err)
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
	for name, query := range queries {
		// there's a bug somewhere (Fleet, osquery or both?)
		// that causes hosts to check-in in a loop if you send
		// an empty query string.
		//
		// we previously fixed this for detail query overrides (see
		// #14286, #14296) but I'm also adding this here as a safeguard
		// for issues like #15524
		if query == "" {
			delete(queries, name)
			delete(discovery, name)
			continue
		}
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
	"mdm":         true,
	"mdm_windows": true,
}

// hostDetailQueryConfig holds pre-loaded configuration data needed for building and ingesting
// detail queries. Loading this once and passing it through avoids redundant database calls
// (AppConfig, HostFeatures, TeamMDMConfig, conditional access) on every detail query result,
// and also caches the resolved detail query map so it is built only once per request.
type hostDetailQueryConfig struct {
	appConfig     *fleet.AppConfig
	features      *fleet.Features
	detailQueries map[string]osquery_utils.DetailQuery
}

func (svc *Service) loadHostDetailQueryConfig(ctx context.Context, host *fleet.Host) (*hostDetailQueryConfig, error) {
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "read app config")
	}

	features, err := svc.HostFeatures(ctx, host)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "read host features")
	}

	var mdmTeamConfig *fleet.TeamMDM
	// LUKS key escrow needs no MDM, so Linux hosts need their fleet's config
	// even when no MDM platform is configured
	if appConfig != nil && host.TeamID != nil &&
		(appConfig.MDM.EnabledAndConfigured || appConfig.MDM.WindowsEnabledAndConfigured || host.FleetPlatform() == "linux") {
		mdmTeamConfig, err = svc.ds.TeamMDMConfig(ctx, *host.TeamID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "reading MDM Team Config")
		}
	}

	detailQueries := osquery_utils.GetDetailQueries(
		ctx,
		svc.config,
		appConfig,
		features,
		osquery_utils.Integrations{
			ConditionalAccessMicrosoft: svc.hostRequiresConditionalAccessMicrosoftIngestion(ctx, host),
		},
		mdmTeamConfig,
	)

	return &hostDetailQueryConfig{
		appConfig:     appConfig,
		features:      features,
		detailQueries: detailQueries,
	}, nil
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

	cfg, err := svc.loadHostDetailQueryConfig(ctx, host)
	if err != nil {
		return nil, nil, err
	}

	queries = make(map[string]string)
	discovery = make(map[string]string)

	for name, query := range cfg.detailQueries {
		if criticalQueriesOnly && !criticalDetailQueries[name] {
			continue
		}

		if query.RunsForPlatform(host.Platform) {
			queryName := hostDetailQueryPrefix + name

			if query.QueryFunc != nil && query.Query == "" {
				query, ok := query.QueryFunc(ctx, svc.logger, host, svc.ds)
				if !ok {
					continue
				}
				queries[queryName] = query
			} else {
				queries[queryName] = query.Query
			}

			discoveryQuery := query.Discovery
			if discoveryQuery == "" {
				discoveryQuery = alwaysTrueQuery
			}
			discovery[queryName] = discoveryQuery
		}
	}

	if cfg.features.AdditionalQueries == nil || criticalQueriesOnly {
		// No additional queries set
		return queries, discovery, nil
	}

	var additionalQueries map[string]string
	if err := json.Unmarshal(*cfg.features.AdditionalQueries, &additionalQueries); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "unmarshal additional queries")
	}

	for name, query := range additionalQueries {
		queryName := hostAdditionalQueryPrefix + name
		queries[queryName] = query
		discovery[queryName] = alwaysTrueQuery
	}

	return queries, discovery, nil
}

func (svc *Service) hostRequiresConditionalAccessMicrosoftIngestion(ctx context.Context, host *fleet.Host) bool {
	if host.Platform != "darwin" && host.Platform != "windows" {
		return false
	}

	conditionalAccessConfigured, conditionalAccessEnabledForTeam, err := svc.conditionalAccessConfiguredAndEnabledForTeam(ctx, host.TeamID)
	if err != nil {
		svc.logger.ErrorContext(ctx, "load conditional access configured and enabled, skipping ingestion",
			"host_id", host.ID,
			"err", err,
		)
		return false
	}

	return conditionalAccessConfigured && conditionalAccessEnabledForTeam
}

func (svc *Service) shouldUpdate(lastUpdated time.Time, interval time.Duration, hostID uint) bool {
	svc.jitterMu.RLock()
	jh := svc.jitterH[interval]
	svc.jitterMu.RUnlock()

	if jh == nil {
		svc.jitterMu.Lock()
		// Double-check after acquiring write lock.
		if svc.jitterH[interval] == nil {
			svc.jitterH[interval] = newJitterHashTable(int(int64(svc.config.Osquery.MaxJitterPercent) * int64(interval.Minutes()) / 100.0))
			svc.logger.DebugContext(context.TODO(), "jitter table created", "bucketCount", svc.jitterH[interval].bucketCount)
		}
		jh = svc.jitterH[interval]
		svc.jitterMu.Unlock()
	}

	jitter := jh.jitterForHost(hostID)
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

// dueHostsChunkSize bounds the number of host IDs per ListHostsLiteByIDs
// query when checking which hosts are due for a distributed read.
const dueHostsChunkSize = 1000

// ListHostIDsDueForDistributedRead returns the subset of hostIDs whose next
// distributed/read would include interval work or an unanswered live query
// campaign, keyed by host ID with the reason it is due. It reuses the read
// path's staleness gates (shouldUpdate, including the per-host jitter
// tables), so notification and read decisions agree by construction. IDs
// with no hosts row (deleted while their agent held a connection) are also
// returned, with AgentWSReasonHostNotFound, so the caller can drop them.
//
// The live query check makes the pub/sub wake-up a latency optimization only:
// a campaign whose one-shot wake-up was lost anywhere along the way is
// recovered within one interval check tick, and hosts stop being re-notified
// once they answer (answering clears their targeting in the store).
//
// Known limitation: with async task processing enabled, the label/policy
// reported-at timestamps may be fresher in Redis than the hosts table columns
// used here. This can only over-notify (one cheap empty read per tick until
// the async timestamps are flushed), never miss due work.
func (svc *Service) ListHostIDsDueForDistributedRead(ctx context.Context, hostIDs []uint) (map[uint]string, error) {
	// skipauth: internal caller (the per-instance interval check job), not a
	// user-facing endpoint.
	svc.authz.SkipAuthorization(ctx)

	// With no active campaigns (the common case) the per-host live query check
	// below is skipped entirely. Errors are non-fatal so interval-work
	// notification never depends on the live query store being reachable.
	activeCampaigns, err := svc.liveQueryStore.LoadActiveQueryNames()
	if err != nil {
		svc.logger.ErrorContext(ctx, "load active query names for distributed read due check", "err", err)
	}

	due := make(map[uint]string)
	for start := 0; start < len(hostIDs); start += dueHostsChunkSize {
		end := min(start+dueHostsChunkSize, len(hostIDs))
		hosts, err := svc.ds.ListHostsLiteByIDs(ctx, hostIDs[start:end])
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "list hosts due for distributed read")
		}
		found := make(map[uint]struct{}, len(hosts))
		for _, host := range hosts {
			found[host.ID] = struct{}{}
		}
		for _, id := range hostIDs[start:end] {
			if _, ok := found[id]; !ok {
				due[id] = fleet.AgentWSReasonHostNotFound
			}
		}
		for _, host := range hosts {
			if reason := svc.hostDueForDistributedRead(host); reason != "" {
				due[host.ID] = reason
				continue
			}
			// A host notified for interval work performs a full
			// distributed/read, which serves any live query targeting it
			// anyway, so only hosts with no interval work are checked.
			if len(activeCampaigns) > 0 {
				if reason := svc.hostDueForLiveQuery(ctx, host.ID); reason != "" {
					due[host.ID] = reason
				}
			}
		}
	}
	return due, nil
}

// hostDueForLiveQuery returns a live-<campaign ID> reason when an active live
// query campaign targets the host and it has not answered yet, or ""
// otherwise. Errors are logged and treated as not due: the check re-runs on
// the next interval check tick. Costs one Redis lookup per host per tick
// while a campaign is active — the same lookup a polling host's
// distributed/read performs today, at a lower frequency.
func (svc *Service) hostDueForLiveQuery(ctx context.Context, hostID uint) string {
	queries, err := svc.liveQueryStore.QueriesForHost(hostID)
	if err != nil {
		svc.logger.ErrorContext(ctx, "list live queries for distributed read due check",
			"host_id", hostID, "err", err)
		return ""
	}
	// The reason is informational only, so when several campaigns target the
	// host any one of them will do.
	for name := range queries {
		return fleet.AgentWSReasonLiveQueryName(name)
	}
	return ""
}

// hostDueForDistributedRead mirrors the gates of detailQueriesForHost,
// labelQueriesForHost and policyQueriesForHost: any single gate being due
// means the host's next distributed/read carries work. It returns the first
// due gate's reason ("" when none); the reason is informational only, so ties
// are not enumerated.
func (svc *Service) hostDueForDistributedRead(host *fleet.Host) string {
	switch {
	case host.RefetchRequested:
		return fleet.AgentWSReasonRefetch
	case host.RefetchCriticalQueriesUntil != nil && host.RefetchCriticalQueriesUntil.After(svc.clock.Now()):
		return fleet.AgentWSReasonRefetch
	case svc.shouldUpdate(host.DetailUpdatedAt, svc.config.Osquery.DetailUpdateInterval, host.ID):
		return fleet.AgentWSReasonDetail
	case svc.shouldUpdate(host.LabelUpdatedAt, svc.config.Osquery.LabelUpdateInterval, host.ID):
		return fleet.AgentWSReasonLabel
	case svc.shouldUpdate(host.PolicyUpdatedAt, svc.config.Osquery.PolicyUpdateInterval, host.ID):
		return fleet.AgentWSReasonPolicy
	default:
		return ""
	}
}

func (svc *Service) hostIsInSetupExperience(ctx context.Context, host *fleet.Host) (bool, error) {
	return fleet.HostIsInSetupExperience(ctx, svc.ds, host)
}

// discardOutOfScopePolicyResults removes, in place, the results for policies that are not in scope for the host.
//
// A host authenticates with its node key and fully controls the fleet_policy_query_<id> keys it submits, so a result is
// only trustworthy for a policy the host is actually assigned (by team, platform and label). Without this, any enrolled
// host could forge membership for policies it was never sent, including policies belonging to another fleet.
//
// The lookup is restricted to the reported IDs rather than loading the host's whole in-scope set, since every
// policy-reporting check-in pays for it.
//
// A host in setup experience is sent a subset of its in-scope policies, but it is checked against the full set: those
// policies are legitimately the host's, so a result for one of them is worth keeping even if setup experience had not
// asked for it yet.
// summarizePolicyResults splits policy results into failing, passing and did-not-execute policy
// IDs, sorted, so they can be logged readably: the results map holds *bool, which renders as
// pointer addresses.
func summarizePolicyResults(policyResults map[uint]*bool) (failing, passing, notExecuted []uint) {
	for policyID, result := range policyResults {
		switch {
		case result == nil:
			notExecuted = append(notExecuted, policyID)
		case *result:
			passing = append(passing, policyID)
		default:
			failing = append(failing, policyID)
		}
	}
	for _, ids := range [][]uint{failing, passing, notExecuted} {
		slices.Sort(ids)
	}
	return failing, passing, notExecuted
}

func (svc *Service) discardOutOfScopePolicyResults(ctx context.Context, host *fleet.Host, policyResults map[uint]*bool) error {
	candidateIDs := make([]uint, 0, len(policyResults))
	for policyID := range policyResults {
		candidateIDs = append(candidateIDs, policyID)
	}

	inScope, err := svc.ds.PolicyQueriesForHostFiltered(ctx, host, candidateIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "retrieve policy queries")
	}
	for policyID := range policyResults {
		if _, ok := inScope[fmt.Sprint(policyID)]; !ok {
			svc.logger.DebugContext(ctx, "discarding result for out-of-scope policy", "policyID", policyID, "hostID", host.ID)
			delete(policyResults, policyID)
		}
	}
	return nil
}

// cleanupOutOfScopePolicyMembership deletes the host's policy_membership rows
// for the given stale policies: policies with a stored row but no result in the
// host's incoming distributed write (as returned by RecordPolicyQueryExecutions).
// Fleet sends all in-scope policy queries together at the policy update interval
// and osquery reports a result (or an error) for each, so a stale policy is no
// longer in scope for the host (e.g. it changed teams, or fell out of the
// policy's platform or label scope). Such rows otherwise linger forever,
// inflating the failing policies counts computed from raw policy_membership
// (Fleet Desktop badge, host issues) even though the host's policy listing
// filters those policies out.
//
// The deletion is skipped for hosts in setup experience: they are sent a
// filtered subset of policy queries (see policyQueriesForHost), so their stale
// set is not meaningful. Under async policy processing this is a no-op, since
// the task layer buffers results in Redis and always reports no stale policies.
// Errors are logged and swallowed: this cleanup is best-effort and self-heals
// on the host's next policy reporting cycle.
func (svc *Service) cleanupOutOfScopePolicyMembership(ctx context.Context, host *fleet.Host, stalePolicyIDs []uint) {
	if len(stalePolicyIDs) == 0 {
		return
	}
	inSetupExperience, err := svc.hostIsInSetupExperience(ctx, host)
	if err != nil {
		logging.WithErr(ctx, err)
		return
	}
	if inSetupExperience {
		return
	}
	if err := svc.ds.ClearHostPolicyMembershipForPolicies(ctx, host.ID, stalePolicyIDs); err != nil {
		logging.WithErr(ctx, err)
		return
	}
	// Refresh the failing policies count now that stale rows are gone;
	// RecordPolicyQueryExecutions already updated it, but before the deletion.
	if err := svc.ds.UpdateHostIssuesFailingPoliciesForSingleHost(ctx, host.ID); err != nil {
		logging.WithErr(ctx, err)
	}
}

// policyQueriesForHost returns policy queries if it's the time to re-run policies on the given host.
// It returns (nil, true, nil) if the interval is so that policies should be executed on the host, but there are no policies
// assigned to such host.
func (svc *Service) policyQueriesForHost(ctx context.Context, host *fleet.Host) (policyQueries map[string]string, noPoliciesForHost bool, err error) {
	policyReportedAt := svc.task.GetHostPolicyReportedAt(ctx, host)
	if !svc.shouldUpdate(policyReportedAt, svc.config.Osquery.PolicyUpdateInterval, host.ID) && !host.RefetchRequested {
		return nil, false, nil
	}
	// This must come after the check above to avoid unnecessary queries to the database. Most
	// requests from live connected hosts will not reach this point
	hostRunningSetupExperience, err := svc.hostIsInSetupExperience(ctx, host)
	if err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "check if host is in setup experience")
	}
	if hostRunningSetupExperience {
		// During setup experience, run ONLY the policies that gate this host's pending setup-experience software, instead of the
		// host's whole (possibly large) team policy set. All other policies stay skipped so unrelated automations do not fire
		// mid-setup. The install itself is performed by setup experience, not by the policy automation (which is suppressed for
		// in-setup hosts in processSoftwareForNewlyFailingPolicies).
		hostUUID, err := fleet.HostUUIDForSetupExperience(host)
		if err != nil {
			return nil, false, ctxerr.Wrap(ctx, err, "get host uuid for setup experience policy queries")
		}
		policyIDs, err := svc.ds.GetSetupExperiencePolicyIDsForHost(ctx, hostUUID)
		if err != nil {
			return nil, false, ctxerr.Wrap(ctx, err, "get setup experience policy ids for host")
		}
		if len(policyIDs) == 0 {
			svc.logger.DebugContext(ctx, "skipping policy queries for host in setup experience (no policy-gated items)", "host_id", host.ID)
			return nil, false, nil
		}
		policyQueries, err = svc.ds.PolicyQueriesForHostFiltered(ctx, host, policyIDs)
		if err != nil {
			return nil, false, ctxerr.Wrap(ctx, err, "retrieve filtered setup experience policy queries")
		}
		// If a gated policy's platform/label scope excludes the host, it won't be returned here and won't run; setup experience
		// detects that and falls back to installing the item, so we don't flag the host as "no policies" (which would bump the
		// policy timestamp).
		return policyQueries, false, nil
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
	Stats    map[string]*fleet.Stats    `json:"stats"`
}

func (shim *submitDistributedQueryResultsRequestShim) hostNodeKey() string {
	return shim.NodeKey
}

// DecodeBody implements the bodyDecoder interface for custom request body
// decoding. This endpoint receives large payloads (distributed query results),
// making it susceptible to client read timeouts (poll.DeadlineExceededError).
// By implementing DecodeBody, we can classify those network errors as client errors.
func (shim *submitDistributedQueryResultsRequestShim) DecodeBody(_ context.Context, r io.Reader, _ url.Values, _ []*x509.Certificate) error {
	if err := json.NewDecoder(r).Decode(shim); err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			osqueryErr := NewOsqueryError("request body read error: "+err.Error(), false)
			osqueryErr.StatusCode = http.StatusRequestTimeout
			return osqueryErr
		}
		return err
	}
	return nil
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
		Stats:    shim.Stats,
	}, nil
}

type SubmitDistributedQueryResultsRequest struct {
	NodeKey  string                               `json:"node_key"`
	Results  fleet.OsqueryDistributedQueryResults `json:"queries"`
	Statuses map[string]fleet.OsqueryStatus       `json:"statuses"`
	Messages map[string]string                    `json:"messages"`
	Stats    map[string]*fleet.Stats              `json:"stats"`
}

type submitDistributedQueryResultsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r submitDistributedQueryResultsResponse) Error() error { return r.Err }

func submitDistributedQueryResultsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	shim := request.(*submitDistributedQueryResultsRequestShim)
	req, err := shim.toRequest(ctx)
	if err != nil {
		return submitDistributedQueryResultsResponse{Err: err}, nil
	}

	err = svc.SubmitDistributedQueryResults(ctx, req.Results, req.Statuses, req.Messages, req.Stats)
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
	stats map[string]*fleet.Stats,
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

	svc.maybeDebugHost(ctx, host, results, statuses, messages, stats)

	preProcessSoftwareResults(ctx, host, results, statuses, messages, osquery_utils.SoftwareOverrideQueries, svc.logger)

	// Lazy-load detail query config only when a detail result is present, to avoid
	// unnecessary HostFeatures/TeamMDMConfig/conditional access DB calls for payloads
	// that only contain label, policy, or live-query results.
	var detailConfig *hostDetailQueryConfig
	var detailConfigFailed bool

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
			logLevel := slog.LevelDebug
			// We'd like to log these as warning for troubleshooting and improving of distributed queries.
			// We have multiple feature requests filed to expose this information in the UI, including https://github.com/fleetdm/fleet/issues/18004
			if messages[query] == "distributed query is denylisted" {
				logLevel = slog.LevelWarn
			}
			svc.logger.Log(ctx, logLevel, "distributed query failed", "query", query, "message", messages[query], "hostID", host.ID)
		}
		queryStats := stats[query]

		// Lazy-load detail config on first detail query result.
		if detailConfig == nil && strings.HasPrefix(query, hostDetailQueryPrefix) {
			if detailConfigFailed {
				// Already failed to load detail config, skip all detail queries.
				continue
			}
			var err error
			detailConfig, err = svc.loadHostDetailQueryConfig(ctx, host)
			if err != nil {
				detailConfigFailed = true
				logging.WithErr(ctx, ctxerr.Wrap(ctx, err, "loading host detail query config"))
				continue
			}
		}

		ingestedDetailUpdated, ingestedAdditionalUpdated, err := svc.ingestQueryResults(
			ctx, query, host, rows, failed, messages, policyResults, labelResults, additionalResults, queryStats, detailConfig,
		)
		if err != nil {
			logging.WithErr(ctx, ctxerr.New(ctx, "error in query ingestion"))
			logging.WithExtras(ctx, "ingestion-err", err)
		}

		detailUpdated = detailUpdated || ingestedDetailUpdated
		additionalUpdated = additionalUpdated || ingestedAdditionalUpdated
	}

	// Load AppConfig separately for label/policy processing. detailConfig may be nil
	// (no detail queries in this check-in) or may have failed to load (soft failure).
	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting app config")
	}

	if len(labelResults) > 0 {
		// Force clear results for labels that do not apply to the host anymore.
		//
		// There could be a timing bug where:
		// 1. Host receives a "team label" query to run (distributed/read).
		// 2. Host is transferred to another team (all its label/policy membership are cleared).
		// 3. Fleet receives distributed/write corresponding to (1) which includes the result for
		//    the label of the old team.
		hostLabelQueries, err := svc.ds.LabelQueriesForHost(ctx, host)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "retrieve label queries")
		}
		for labelID := range labelResults {
			if _, ok := hostLabelQueries[fmt.Sprint(labelID)]; !ok {
				svc.logger.DebugContext(ctx, "clearing result for inapplicable label", "labelID", labelID, "hostID", host.ID)
				labelResults[labelID] = ptr.Bool(false)
			}
		}

		if err := svc.task.RecordLabelQueryExecutions(ctx, host, labelResults, svc.clock.Now(), ac.ServerSettings.DeferredSaveHost); err != nil {
			logging.WithErr(ctx, err)
		}
	}

	// Keep separate from the block below: this can empty policyResults, and an empty (rather than absent) result set
	// makes RecordPolicyQueryExecutions treat every stored policy_membership row for the host as stale.
	if len(policyResults) > 0 {
		failing, passing, notExecuted := summarizePolicyResults(policyResults)
		svc.logger.DebugContext(ctx, "received policy results",
			"host_id", host.ID,
			"host_platform", host.Platform,
			"team_id", ptr.ValOrZero(host.TeamID),
			"failing", failing,
			"passing", passing,
			"not_executed", notExecuted,
		)

		if err := svc.discardOutOfScopePolicyResults(ctx, host, policyResults); err != nil {
			// Drop this cycle's policy results instead of failing the whole write: the host reports them again on
			// its next check-in, whereas the detail and additional results from this same payload are only written
			// further down (SaveHostAdditional, UpdateHost) and returning here would discard them.
			logging.WithErr(ctx, ctxerr.Wrap(ctx, err, "discard out-of-scope policy results"))
			clear(policyResults)
		}
	}

	if len(policyResults) > 0 {
		// Compute flipping policies once for all consumers. This replaces up to 5 individual calls to
		// FlippingPoliciesForHost with a single database query.
		newFailing, newPassing, err := svc.ds.FlippingPoliciesForHost(ctx, host.ID, policyResults)
		if err != nil {
			logging.WithErr(ctx, err)
		}
		// Ensure newPassing is non-nil so RecordPolicyQueryExecutions can distinguish "pre-computed with zero results"
		// from "not pre-computed" (nil means compute it yourself).
		if newPassing == nil {
			newPassing = []uint{}
		}
		newFailingSet := make(map[uint]struct{}, len(newFailing))
		for _, id := range newFailing {
			newFailingSet[id] = struct{}{}
		}
		// The automations below act on transitions, not on the raw results, so this is the line to
		// check first when one of them doesn't fire for a policy that is reporting a failure.
		svc.logger.DebugContext(ctx, "computed policy transitions",
			"host_id", host.ID,
			"new_failing", newFailing,
			"new_passing", newPassing,
			"results_in_scope", len(policyResults),
		)

		if err := processCalendarPolicies(ctx, svc.ds, ac, host, policyResults, svc.logger); err != nil {
			logging.WithErr(ctx, err)
		}

		if err := svc.processScriptsForNewlyFailingPolicies(ctx, host.ID, host.TeamID, host.Platform, host.OrbitNodeKey, host.ScriptsEnabled, policyResults, newFailingSet); err != nil {
			logging.WithErr(ctx, err)
		}

		if host.Platform == "darwin" || host.Platform == "windows" {
			if err := svc.processConditionalAccessForNewlyFailingPolicies(ctx, host.ID, host.TeamID, host.OrbitNodeKey, host.Platform, policyResults); err != nil {
				logging.WithErr(ctx, err)
			}

			if err := svc.processProfileResendsForNewlyFailingPolicies(ctx, host, policyResults, newFailingSet); err != nil {
				logging.WithErr(ctx, err)
			}
		}

		if host.Platform == "darwin" && svc.EnterpriseOverrides != nil {
			// NOTE: if the installers for the policies here are not scoped to the host via labels, we update the policy status here to stop it from showing up as "failed" in the
			// host details.
			if err := svc.processVPPForNewlyFailingPolicies(ctx, host.ID, host.TeamID, host.Platform, policyResults, newFailingSet); err != nil {
				logging.WithErr(ctx, err)
			}
		}

		// setupExperienceHostUUID keys setup-experience rows (OsqueryHostID on Windows/Linux); on error it is empty, which
		// disables setup-experience automation suppression (matches no host).
		setupExperienceHostUUID, seuErr := fleet.HostUUIDForSetupExperience(host)
		if seuErr != nil {
			svc.logger.ErrorContext(ctx, "could not derive setup experience host UUID; setup-experience suppression disabled for this host",
				"err", seuErr, "host_id", host.ID, "platform", host.Platform)
			ctxerr.Handle(ctx, seuErr)
		}
		if err := svc.processSoftwareForNewlyFailingPolicies(ctx, host.ID, host.TeamID, host.Platform, host.OrbitNodeKey, setupExperienceHostUUID, policyResults, newFailingSet); err != nil {
			logging.WithErr(ctx, err)
		}

		// Filter policy results for webhooks using pre-computed flipping sets.
		var policyIDs []uint
		if globalPolicyAutomationsEnabled(ac.WebhookSettings, ac.Integrations) {
			policyIDs = append(policyIDs, ac.WebhookSettings.FailingPoliciesWebhook.PolicyIDs...)
		}

		teamID := uint(0)
		if host.TeamID != nil {
			teamID = *host.TeamID
		}
		team, err := svc.ds.TeamLite(ctx, teamID)
		if err != nil {
			logging.WithErr(ctx, err)
		} else if teamPolicyAutomationsEnabled(team.Config.WebhookSettings, team.Config.Integrations) {
			policyIDs = append(policyIDs, team.Config.WebhookSettings.FailingPoliciesWebhook.PolicyIDs...)
		}

		filteredResults := filterPolicyResults(policyResults, policyIDs)
		if len(filteredResults) > 0 {
			// Filter the pre-computed flipping results to only webhook-enabled policies.
			webhookFailing := filterByPolicyIDs(newFailing, filteredResults)
			webhookPassing := filterByPolicyIDs(newPassing, filteredResults)
			if len(webhookFailing) > 0 || len(webhookPassing) > 0 {
				// Register the flipped policies on a goroutine to not block the hosts on redis requests.
				go func() {
					if err := svc.registerFlippedPolicies(ctx, host.ID, host.Hostname, host.DisplayName(), webhookFailing, webhookPassing); err != nil {
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

		stalePolicyIDs, err := svc.task.RecordPolicyQueryExecutions(ctx, host, policyResults, svc.clock.Now(), ac.ServerSettings.DeferredSaveHost, newPassing)
		if err != nil {
			logging.WithErr(ctx, err)
		}
		svc.cleanupOutOfScopePolicyMembership(ctx, host, stalePolicyIDs)
	} else if hostWithoutPolicies {
		// RecordPolicyQueryExecutions called with results=nil will still update the host's policy_updated_at column.
		// The host was sent the "no policies" wildcard query, so no policies are in scope
		// for it and all of its stored policy_membership rows are stale.
		stalePolicyIDs, err := svc.task.RecordPolicyQueryExecutions(ctx, host, nil, svc.clock.Now(), ac.ServerSettings.DeferredSaveHost, []uint{})
		if err != nil {
			logging.WithErr(ctx, err)
		}
		svc.cleanupOutOfScopePolicyMembership(ctx, host, stalePolicyIDs)
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
	if refetchCriticalSet {
		svc.logger.DebugContext(ctx, "refetch critical status on submit distributed query results", "host_id", host.ID, "refetch_requested", refetchRequested, "refetch_critical_queries_until", host.RefetchCriticalQueriesUntil, "refetch_critical_cleared", refetchCriticalCleared)
	}

	if refetchRequested || detailUpdated || refetchCriticalCleared {
		if ac.ServerSettings.DeferredSaveHost {
			go svc.serialUpdateHost(ctx, host)
		} else {
			if err := svc.ds.UpdateHost(ctx, host); err != nil {
				logging.WithErr(ctx, err)
			}
		}
	}

	if detailUpdated && ac.MDM.EnabledAndConfigured && host.Platform == "darwin" && host.ComputerName != "" {
		if err := svc.ds.UpdateHostDeviceNameStatusFromReport(ctx, host.UUID, host.ComputerName); err != nil {
			logging.WithErr(ctx, err)
		}
	}

	if host.DiskEncryptionKeyEscrowed {
		if err := svc.NewActivity(
			ctx,
			nil,
			fleet.ActivityTypeEscrowedDiskEncryptionKey{
				HostID:          host.ID,
				HostDisplayName: host.DisplayName(),
			},
		); err != nil {
			svc.logger.ErrorContext(ctx, "record fleet disk encryption key escrowed activity",
				"err", err,
			)
		}
	}

	return nil
}

func processCalendarPolicies(
	ctx context.Context,
	ds fleet.Datastore,
	appConfig *fleet.AppConfig,
	host *fleet.Host,
	policyResults map[uint]*bool,
	logger *slog.Logger,
) error {
	if len(appConfig.Integrations.GoogleCalendar) == 0 || host.TeamID == nil {
		return nil
	}

	team, err := ds.TeamLite(ctx, *host.TeamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "load host team")
	}

	if team.Config.Integrations.GoogleCalendar == nil || !team.Config.Integrations.GoogleCalendar.Enable {
		return nil
	}

	hostCalendarEvent, calendarEvent, err := ds.GetHostCalendarEvent(ctx, host.ID)
	switch {
	case err == nil:
		if hostCalendarEvent.WebhookStatus != fleet.CalendarWebhookStatusPending {
			return nil
		}
	case fleet.IsNotFound(err):
		return nil
	default:
		return ctxerr.Wrap(ctx, err, "get host calendar event")
	}

	now := time.Now()
	if now.Before(calendarEvent.StartTime) {
		logger.WarnContext(ctx, "results came too early", "now", now, "start_time", calendarEvent.StartTime)
		if err = ds.UpdateHostCalendarWebhookStatus(context.Background(), host.ID, fleet.CalendarWebhookStatusError); err != nil {
			logger.ErrorContext(ctx, "mark webhook as errored early", "err", err)
		}
		return nil
	}

	//
	// TODO(lucas): Discuss.
	//
	const allowedTimeRelativeToEndTime = 5 * time.Minute // up to 5 minutes after the end_time to allow for short (0-time) event times

	if now.After(calendarEvent.EndTime.Add(allowedTimeRelativeToEndTime)) {
		logger.WarnContext(ctx, "results came too late", "now", now, "end_time", calendarEvent.EndTime)
		if err = ds.UpdateHostCalendarWebhookStatus(context.Background(), host.ID, fleet.CalendarWebhookStatusError); err != nil {
			logger.ErrorContext(ctx, "mark webhook as errored late", "err", err)
		}
		return nil
	}

	calendarPolicies, err := ds.GetCalendarPolicies(ctx, *host.TeamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get calendar policy ids")
	}
	if len(calendarPolicies) == 0 {
		return nil
	}

	failingCalendarPolicies := getFailingCalendarPolicies(policyResults, calendarPolicies)
	if len(failingCalendarPolicies) == 0 {
		return nil
	}

	go func() {
		retryStrategy := backoff.NewExponentialBackOff()
		retryStrategy.MaxElapsedTime = 30 * time.Minute
		err := backoff.Retry(
			func() error {
				if err := fleet.FireCalendarWebhook(
					team.Config.Integrations.GoogleCalendar.WebhookURL,
					host.ID, host.HardwareSerial, host.DisplayName(), failingCalendarPolicies, "",
					logger,
				); err != nil {
					var statusCoder kithttp.StatusCoder
					if errors.As(err, &statusCoder) && statusCoder.StatusCode() == http.StatusTooManyRequests {
						logger.DebugContext(ctx, "fire webhook", "err", err)
						if err := ds.UpdateHostCalendarWebhookStatus(
							context.Background(), host.ID, fleet.CalendarWebhookStatusRetry,
						); err != nil {
							logger.ErrorContext(ctx, "mark fired webhook as retry", "err", err)
						}
						return err
					}
					return backoff.Permanent(err)
				}
				return nil
			}, retryStrategy,
		)
		nextStatus := fleet.CalendarWebhookStatusSent
		if err != nil {
			logger.ErrorContext(ctx, "fire webhook", "err", err)
			nextStatus = fleet.CalendarWebhookStatusError
		}
		if err := ds.UpdateHostCalendarWebhookStatus(context.Background(), host.ID, nextStatus); err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("mark fired webhook as %v", nextStatus), "err", err)
		}
	}()

	return nil
}

func getFailingCalendarPolicies(policyResults map[uint]*bool, calendarPolicies []fleet.PolicyCalendarData) []fleet.PolicyCalendarData {
	var failingPolicies []fleet.PolicyCalendarData
	for _, calendarPolicy := range calendarPolicies {
		result, ok := policyResults[calendarPolicy.ID]
		if !ok || // ignore result of a policy that's not configured for calendar.
			result == nil { // ignore policies that failed to execute.
			continue
		}
		if !*result {
			failingPolicies = append(failingPolicies, calendarPolicy)
		}
	}
	return failingPolicies
}

// preProcessSoftwareResults will run pre-processing on the responses of the software queries.
// It will move the results from the software extra queries (e.g. software_vscode_extensions)
// into the main software query results (software_{macos|linux|windows}) as well as process
// any overrides that are set.
// We do this to not grow the main software queries and to ingest
// all software together (one direct ingest function for all software).
func preProcessSoftwareResults(
	ctx context.Context,
	host *fleet.Host,
	results fleet.OsqueryDistributedQueryResults,
	statuses map[string]fleet.OsqueryStatus,
	messages map[string]string,
	overrides map[string]osquery_utils.DetailQuery,
	logger *slog.Logger,
) {
	vsCodeExtensionsExtraQuery := hostDetailQueryPrefix + "software_vscode_extensions"
	preProcessSoftwareExtraResults(ctx, vsCodeExtensionsExtraQuery, host.ID, results, statuses, messages, osquery_utils.DetailQuery{}, logger)

	pythonPackagesExtraQuery := hostDetailQueryPrefix + "software_python_packages"
	preProcessSoftwareExtraResults(ctx, pythonPackagesExtraQuery, host.ID, results, statuses, messages, osquery_utils.DetailQuery{}, logger)
	pythonPackagesWithUsersExtraQuery := hostDetailQueryPrefix + "software_python_packages_with_users_dir"
	preProcessSoftwareExtraResults(ctx, pythonPackagesWithUsersExtraQuery, host.ID, results, statuses, messages, osquery_utils.DetailQuery{}, logger)

	fleetdPacmanPackagesExtraQuery := hostDetailQueryPrefix + "software_linux_fleetd_pacman"
	preProcessSoftwareExtraResults(ctx, fleetdPacmanPackagesExtraQuery, host.ID, results, statuses, messages, osquery_utils.DetailQuery{}, logger)

	jetbrainsPluginsExtraQuery := hostDetailQueryPrefix + "software_jetbrains_plugins"
	preProcessSoftwareExtraResults(ctx, jetbrainsPluginsExtraQuery, host.ID, results, statuses, messages, osquery_utils.DetailQuery{}, logger)

	adobePluginsExtraQuery := hostDetailQueryPrefix + "software_adobe_plugins"
	preProcessSoftwareExtraResults(ctx, adobePluginsExtraQuery, host.ID, results, statuses, messages, osquery_utils.DetailQuery{}, logger)

	goBinariesExtraQuery := hostDetailQueryPrefix + "software_go_binaries"
	preProcessSoftwareExtraResults(ctx, goBinariesExtraQuery, host.ID, results, statuses, messages, osquery_utils.DetailQuery{}, logger)

	for name, query := range overrides {
		fullQueryName := hostDetailQueryPrefix + "software_" + name
		preProcessSoftwareExtraResults(ctx, fullQueryName, host.ID, results, statuses, messages, query, logger)
	}

	// Filter out python packages that are also deb packages on ubuntu/debian
	pythonPackageFilter(host.Platform, results, statuses)

	updateFleetdVersion(host.Platform, results)
}

// updateFleetdVersion updates the version of the fleetd package using the orbit version from the orbit_info table for Linux hosts.
// We do this because orbit uses an auto-update mechanism which does not update the host's package manager database.
func updateFleetdVersion(hostPlatform string, results fleet.OsqueryDistributedQueryResults) {
	// Just update the versions for Linux.
	if !fleet.IsLinux(hostPlatform) {
		return
	}

	orbitInfoResults := results[hostDetailQueryPrefix+"orbit_info"]
	if len(orbitInfoResults) != 1 {
		return
	}
	orbitVersion := orbitInfoResults[0]["version"]
	if orbitVersion == "" {
		return
	}

	for _, row := range results[hostDetailQueryPrefix+"software_linux"] {
		if row["name"] != "fleet-osquery" {
			continue
		}
		row["version"] = orbitVersion
		break
	}
}

// pythonPackageFilter filters out duplicate python_packages that are installed under deb_packages on Ubuntu and Debian.
// python_packages not matching a Debian package names are updated to "python3-packagename" to match OVAL definitions.
func pythonPackageFilter(platform string, results fleet.OsqueryDistributedQueryResults, statuses map[string]fleet.OsqueryStatus) {
	const pythonPrefix = "python3-"
	const pythonSource = "python_packages"
	const debSource = "deb_packages"
	const rpmSource = "rpm_packages"
	const linuxSoftware = hostDetailQueryPrefix + "software_linux"

	// Return early if platform is not Ubuntu, Debian, or RHEL (Inc. Fedora)
	// We may need to add more platforms in the future
	if platform != "ubuntu" && platform != "debian" && platform != "rhel" {
		return
	}

	// Check the 'software_linux' result and status
	sw, ok := results[linuxSoftware]
	if !ok {
		return
	}
	if status, ok := statuses[linuxSoftware]; !ok || status != fleet.StatusOK {
		return
	}

	// Extract the Python and Debian packages from the software list for filtering
	// pre-allocating space for 40 packages based on number of package found in
	// a fresh ubuntu 24.04 install.
	// A python package name may appear multiple times (e.g. from multiple user directories),
	// so we track all indexes for each name.
	pythonPackages := make(map[string][]int, 40)
	debPackages := make(map[string]struct{}, 40)
	rpmPackages := make(map[string]struct{}, 60)

	// Track indexes of rows to remove
	indexesToRemove := []int{}

	for i, row := range sw {
		switch row["source"] {
		case pythonSource:
			loweredName := strings.ToLower(row["name"])
			pythonPackages[loweredName] = append(pythonPackages[loweredName], i)
			row["name"] = loweredName
		case debSource:
			// Only append python3 deb packages
			if strings.HasPrefix(row["name"], pythonPrefix) {
				debPackages[row["name"]] = struct{}{}
			}
		case rpmSource:
			if strings.HasPrefix(row["name"], pythonPrefix) {
				rpmPackages[row["name"]] = struct{}{}
			}
		}
	}

	// Return early if there are no Python packages to process
	if len(pythonPackages) == 0 {
		return
	}

	// Loop through pythonPackages map to identify any that should be removed
	for name, indexes := range pythonPackages {
		convertedName := pythonPrefix + name

		// Filter out Python packages that are also Debian or RPM packages
		if _, found := debPackages[convertedName]; found {
			indexesToRemove = append(indexesToRemove, indexes...)
		} else if _, found := rpmPackages[convertedName]; found {
			indexesToRemove = append(indexesToRemove, indexes...)
		} else {
			// Update remaining Python package names to match OVAL definitions
			for _, index := range indexes {
				sw[index]["name"] = convertedName
			}
		}
	}

	// Sort indexes to remove in descending order
	sort.Sort(sort.Reverse(sort.IntSlice(indexesToRemove)))

	// Remove rows from sw in descending order of indexes
	for _, index := range indexesToRemove {
		sw = append(sw[:index], sw[index+1:]...)
	}

	// Store the updated software result back in the results map
	results[linuxSoftware] = sw
}

func preProcessSoftwareExtraResults(
	ctx context.Context,
	softwareExtraQuery string,
	hostID uint,
	results fleet.OsqueryDistributedQueryResults,
	statuses map[string]fleet.OsqueryStatus,
	messages map[string]string,
	override osquery_utils.DetailQuery,
	logger *slog.Logger,
) {
	// We always remove the extra query and its results
	// in case the main or extra software query failed to execute.
	defer delete(results, softwareExtraQuery)

	status, ok := statuses[softwareExtraQuery]
	if !ok {
		return // query did not execute, e.g. the table does not exist.
	}
	failed := status != fleet.StatusOK
	if failed {
		// extra query executed but with errors, so we return without changing anything.
		logger.ErrorContext(ctx, "extra query executed with errors",
			"query", softwareExtraQuery,
			"message", messages[softwareExtraQuery],
			"hostID", hostID,
		)
		return
	}

	// Extract the results of the extra query.
	softwareExtraRows := results[softwareExtraQuery]
	if len(softwareExtraRows) == 0 {
		return
	}

	// Append the results of the extra query to the main query.
	for _, query := range []string{
		// Only one of these execute in each host.
		hostDetailQueryPrefix + "software_macos",
		hostDetailQueryPrefix + "software_windows",
		hostDetailQueryPrefix + "software_linux",
	} {
		if _, ok := results[query]; !ok {
			continue
		}
		if status, ok := statuses[query]; ok && status != fleet.StatusOK {
			// Do not append results if the main query failed to run.
			continue
		}
		if override.SoftwareProcessResults != nil {
			results[query] = override.SoftwareProcessResults(results[query], softwareExtraRows)
		} else {
			results[query] = removeOverrides(results[query], override)
			results[query] = append(results[query], softwareExtraRows...)
		}
		return
	}
}

func removeOverrides(rows []map[string]string, override osquery_utils.DetailQuery) []map[string]string {
	if override.SoftwareOverrideMatch != nil {
		rows = slices.DeleteFunc(rows, func(row map[string]string) bool {
			return override.SoftwareOverrideMatch(row)
		})
	}

	return rows
}

// globalPolicyAutomationsEnabled returns true if any of the global policy automations are enabled.
// globalPolicyAutomationsEnabled and teamPolicyAutomationsEnabled are effectively identical.
// We could not use Go generics because Go generics does not support accessing common struct fields right now.
// The umbrella Go issue tracking this: https://github.com/golang/go/issues/63940
func globalPolicyAutomationsEnabled(webhookSettings fleet.WebhookSettings, integrations fleet.Integrations) bool {
	if webhookSettings.FailingPoliciesWebhook.Enable {
		return true
	}
	for _, j := range integrations.Jira {
		if j.EnableFailingPolicies {
			return true
		}
	}
	for _, z := range integrations.Zendesk {
		if z.EnableFailingPolicies {
			return true
		}
	}
	return false
}

func teamPolicyAutomationsEnabled(webhookSettings fleet.TeamWebhookSettings, integrations fleet.TeamIntegrations) bool {
	if webhookSettings.FailingPoliciesWebhook.Enable {
		return true
	}
	for _, j := range integrations.Jira {
		if j.EnableFailingPolicies {
			return true
		}
	}
	for _, z := range integrations.Zendesk {
		if z.EnableFailingPolicies {
			return true
		}
	}
	return false
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
	stats *fleet.Stats,
	detailConfig *hostDetailQueryConfig,
) (bool, bool, error) {
	var detailUpdated, additionalUpdated bool

	// live queries we do want to ingest even if the query had issues, because we want to inform the user of these
	// issues
	// same applies to policies, since it's a 3 state result, one of them being failure, and labels take this state
	// into account as well

	var err error
	switch {
	case strings.HasPrefix(query, hostDistributedQueryPrefix):
		err = svc.ingestDistributedQuery(ctx, *host, query, rows, messages[query], stats)
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
		if detailConfig == nil { // safety net for NilAway linter
			return false, false, newOsqueryError("detail query config not loaded for query " + query)
		}
		trimmedQuery := strings.TrimPrefix(query, hostDetailQueryPrefix)
		var ingested bool
		ingested, err = svc.directIngestDetailQuery(ctx, host, trimmedQuery, rows, detailConfig)
		if !ingested && err == nil {
			err = svc.ingestDetailQuery(ctx, host, trimmedQuery, rows, detailConfig)
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

func (svc *Service) directIngestDetailQuery(ctx context.Context, host *fleet.Host, name string, rows []map[string]string, cfg *hostDetailQueryConfig) (ingested bool, err error) {
	query, ok := cfg.detailQueries[name]
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
func (svc *Service) ingestDistributedQuery(
	ctx context.Context, host fleet.Host, name string, rows []map[string]string, errMsg string, stats *fleet.Stats,
) error {
	trimmedQuery := strings.TrimPrefix(name, hostDistributedQueryPrefix)

	campaignID, err := strconv.Atoi(osquery_utils.EmptyToZero(trimmedQuery))
	if err != nil {
		return newOsqueryError("unable to parse campaign ID: " + trimmedQuery)
	}

	// Write the results to the pubsub store
	res := fleet.DistributedQueryResult{
		DistributedQueryCampaignID: uint(campaignID), //nolint:gosec // dismiss G115
		Host: fleet.ResultHostData{
			ID:          host.ID,
			Hostname:    host.Hostname,
			DisplayName: host.DisplayName(),
		},
		Rows:  rows,
		Stats: stats,
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
		campaign, err := svc.ds.DistributedQueryCampaign(ctx, uint(campaignID)) //nolint:gosec // dismiss G115
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
		results[uint(trimmedQueryNum)] = nil //nolint:gosec // dismiss G115
	} else {
		results[uint(trimmedQueryNum)] = ptr.Bool(len(rows) > 0) //nolint:gosec // dismiss G115
	}

	return nil
}

// ingestDetailQuery takes the results of a detail query and modifies the
// provided fleet.Host appropriately.
func (svc *Service) ingestDetailQuery(ctx context.Context, host *fleet.Host, name string, rows []map[string]string, cfg *hostDetailQueryConfig) error {
	query, ok := cfg.detailQueries[name]
	if !ok {
		return newOsqueryError("unknown detail query " + name)
	}

	if query.IngestFunc != nil {
		if err := query.IngestFunc(ctx, svc.logger, host, rows); err != nil {
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

// filterByPolicyIDs returns only the policy IDs from ids that are present in allowedResults and have a non-nil result
// (i.e., the policy actually executed). This matches the behavior of FlippingPoliciesForHost which ignores nil results.
func filterByPolicyIDs(ids []uint, allowedResults map[uint]*bool) []uint {
	var filtered []uint
	for _, id := range ids {
		if val, ok := allowedResults[id]; ok && val != nil {
			filtered = append(filtered, id)
		}
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

// continuousAutomationOnCooldown reports whether a continuous policy automation that
// last fired (queued an install) at lastFiredAt should be skipped on this run.
//
// Continuous automations re-fire on every failing policy result, not just on
// pass→fail transitions. A successful install requests a host vitals refetch, and a
// refetch makes policies re-run immediately (bypassing the policy update interval).
// If the policy keeps failing, that creates a tight install→refetch→re-run→install
// loop. We throttle continuous re-fires to at most once per policy update interval so
// a perpetually-failing policy retries on the next interval (~1h) instead of
// continuously. A zero lastFiredAt (no prior automation install) is never on cooldown.
func (svc *Service) continuousAutomationOnCooldown(lastFiredAt time.Time) bool {
	if lastFiredAt.IsZero() {
		return false
	}
	return svc.clock.Now().Sub(lastFiredAt) < svc.config.Osquery.PolicyUpdateInterval
}

// deferFleetInitiatedActivation reports whether fleet-initiated activities
// (policy-automation installs and scripts) should be enqueued without inline
// activation, leaving them to the fleet-initiated release cron to activate
// within the activity.fleet_initiated_release_per_minute budget.
func (svc *Service) deferFleetInitiatedActivation() bool {
	return svc.config.Activity.FleetInitiatedReleasePerMinute > 0
}

func (svc *Service) processSoftwareForNewlyFailingPolicies(
	ctx context.Context,
	hostID uint,
	hostTeamID *uint,
	hostPlatform string,
	hostOrbitNodeKey *string,
	setupExperienceHostUUID string,
	incomingPolicyResults map[uint]*bool,
	newFailingSet map[uint]struct{},
) error {
	if hostOrbitNodeKey == nil || *hostOrbitNodeKey == "" {
		// We do not want to queue software installations on vanilla osquery hosts.
		return nil
	}

	var policyTeamID uint
	if hostTeamID == nil {
		policyTeamID = fleet.PolicyNoTeamID
	} else {
		policyTeamID = *hostTeamID
	}

	// Filter out results that are not failures (we are only interested on failing policies,
	// we don't care about passing policies or policies that failed to execute).
	var incomingFailingPoliciesIDs []uint
	for policyID, policyResult := range incomingPolicyResults {
		if policyResult != nil && !*policyResult {
			incomingFailingPoliciesIDs = append(incomingFailingPoliciesIDs, policyID)
		}
	}
	if len(incomingFailingPoliciesIDs) == 0 {
		return nil
	}

	// Get policies with associated installers for the team.
	policiesWithInstaller, err := svc.ds.GetPoliciesWithAssociatedInstaller(ctx, policyTeamID, incomingFailingPoliciesIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "failed to get policies with installer")
	}
	if len(policiesWithInstaller) == 0 {
		return nil
	}

	// Filter to policies with installers that are newly failing, or that have
	// continuous_automations_enabled set (in which case every failing result
	// triggers an install, not just pass→fail transitions).
	var failingPoliciesWithInstaller []fleet.PolicySoftwareInstallerData
	for _, policyWithInstaller := range policiesWithInstaller {
		if _, ok := newFailingSet[policyWithInstaller.ID]; ok || policyWithInstaller.ContinuousAutomationsEnabled {
			failingPoliciesWithInstaller = append(failingPoliciesWithInstaller, policyWithInstaller)
		}
	}
	if len(failingPoliciesWithInstaller) == 0 {
		return nil
	}

	// Suppress the automation for any policy that gates one of this host's setup-experience items: while the host is in setup
	// experience, setup experience performs that install itself. Installing here too would double-install.
	if setupExperienceHostUUID != "" {
		gatedPolicyIDs, err := svc.ds.GetSetupExperiencePolicyIDsForHost(ctx, setupExperienceHostUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get setup experience policy ids for host")
		}
		if len(gatedPolicyIDs) > 0 {
			gatedSet := make(map[uint]struct{}, len(gatedPolicyIDs))
			for _, id := range gatedPolicyIDs {
				gatedSet[id] = struct{}{}
			}
			kept := failingPoliciesWithInstaller[:0]
			for _, p := range failingPoliciesWithInstaller {
				if _, gated := gatedSet[p.ID]; gated {
					svc.logger.DebugContext(ctx, "skipping policy automation install for host in setup experience; setup experience will install it",
						"host_id", hostID, "policy_id", p.ID)
					continue
				}
				kept = append(kept, p)
			}
			failingPoliciesWithInstaller = kept
			if len(failingPoliciesWithInstaller) == 0 {
				return nil
			}
		}
	}

	for _, failingPolicyWithInstaller := range failingPoliciesWithInstaller {
		policyID := failingPolicyWithInstaller.ID
		_, newlyFailing := newFailingSet[policyID]
		installerMetadata, err := svc.ds.GetSoftwareInstallerMetadataByID(ctx, failingPolicyWithInstaller.InstallerID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get software installer metadata by id")
		}
		softwareInstallerTitleID_ := uint(0)
		if installerMetadata.TitleID != nil {
			softwareInstallerTitleID_ = *installerMetadata.TitleID
		}
		logger := svc.logger.With(
			"host_id", hostID,
			"host_platform", hostPlatform,
			"policy_id", failingPolicyWithInstaller.ID,
			"software_installer_id", failingPolicyWithInstaller.InstallerID,
			"software_title_id", softwareInstallerTitleID_,
			"software_installer_platform", installerMetadata.Platform,
		)
		if fleet.PlatformFromHost(hostPlatform) != installerMetadata.Platform {
			logger.DebugContext(ctx, "installer platform does not match host platform")
			continue
		}
		scoped, err := svc.ds.IsSoftwareInstallerLabelScoped(ctx, failingPolicyWithInstaller.InstallerID, hostID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "checking if software installer is label scoped to host")
		}
		if !scoped {
			// NOTE: we update the policy status here to stop it from showing up as "failed" in the
			// host details.
			incomingPolicyResults[failingPolicyWithInstaller.ID] = nil
			logger.DebugContext(ctx, "not marking policy as failed since software is out of scope for host")
			continue
		}
		hostLastInstall, err := svc.ds.GetHostLastInstallData(ctx, hostID, installerMetadata.InstallerID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get host last install data")
		}
		// hostLastInstall.Status == nil can happen when a software is installed by Fleet and later removed.
		if hostLastInstall != nil && hostLastInstall.Status != nil &&
			*hostLastInstall.Status == fleet.SoftwareInstallPending {
			// There's a pending install for this host and installer,
			// thus we do not queue another install request.
			logger.DebugContext(ctx, "found pending install request for this host and installer",
				"pending_execution_id", hostLastInstall.ExecutionID,
			)
			continue
		}

		// Throttle continuous policy automation re-installs: if this policy fired only
		// because continuous_automations_enabled is set (not a pass→fail transition)
		// and we already queued a successful install within the policy update interval,
		// skip it. This prevents a tight install→refetch→re-run loop when the install
		// succeeds but never makes the policy pass. We only throttle on a successful
		// install because only success requests the refetch that drives the loop; failed
		// installs are retried via the dedicated retry path (see
		// shouldRetryPolicyAutomationSoftwareInstall). See continuousAutomationOnCooldown.
		if !newlyFailing && failingPolicyWithInstaller.ContinuousAutomationsEnabled &&
			hostLastInstall != nil && hostLastInstall.Status != nil &&
			*hostLastInstall.Status == fleet.SoftwareInstalled &&
			svc.continuousAutomationOnCooldown(hostLastInstall.UpdatedAt) {
			logger.InfoContext(ctx, "skipping continuous policy automation install; within policy update interval cooldown",
				"last_install_execution_id", hostLastInstall.ExecutionID,
				"last_install_at", hostLastInstall.UpdatedAt,
			)
			continue
		}

		// Don't attempt another install for this policy if the retry limit is reached.
		if svc.installFailureLimitReached(ctx, hostID, installerMetadata.InstallerID, policyID) {
			continue
		}

		// On a continuous re-fire (policy still failing), reset prior
		// attempt_number values for this host/policy to 0 so the new attempt
		// restarts the retry sequence at 1 instead of inheriting the cap from
		// the previous sequence. A no-op on pass→fail transitions (those rows
		// are already at 0 from the prior fail→pass reset).
		if failingPolicyWithInstaller.ContinuousAutomationsEnabled {
			if err := svc.ds.ResetPolicyAutomationRetryAttemptsForHost(ctx, hostID, []uint{policyID}); err != nil {
				return ctxerr.Wrap(ctx, err, "reset policy automation retry attempts for host")
			}
		}

		// NOTE(lucas): The user_id set in this software install will be NULL
		// so this means that when generating the activity for this action
		// (in SaveHostSoftwareInstallResult) the author will be set to Fleet.
		installUUID, err := svc.ds.InsertSoftwareInstallRequest(
			ctx, hostID,
			installerMetadata.InstallerID,
			fleet.HostSoftwareInstallOptions{
				SelfService:     false,
				PolicyID:        &policyID,
				DeferActivation: svc.deferFleetInitiatedActivation(),
			},
		)
		if err != nil {
			return ctxerr.Wrapf(ctx, err,
				"insert software install request: host_id=%d, software_installer_id=%d",
				hostID, installerMetadata.InstallerID,
			)
		}
		logger.DebugContext(ctx, "install request sent",
			"install_uuid", installUUID,
		)
	}
	return nil
}

func (svc *Service) processVPPForNewlyFailingPolicies(
	ctx context.Context,
	hostID uint,
	hostTeamID *uint,
	hostPlatform string,
	incomingPolicyResults map[uint]*bool,
	newFailingSet map[uint]struct{},
) error {
	var policyTeamID uint
	if hostTeamID == nil {
		policyTeamID = fleet.PolicyNoTeamID
	} else {
		policyTeamID = *hostTeamID
	}

	// Filter out results that are not failures (we are only interested on failing policies,
	// we don't care about passing policies or policies that failed to execute).
	var incomingFailingPoliciesIDs []uint
	for policyID, policyResult := range incomingPolicyResults {
		if policyResult != nil && !*policyResult {
			incomingFailingPoliciesIDs = append(incomingFailingPoliciesIDs, policyID)
		}
	}
	if len(incomingFailingPoliciesIDs) == 0 {
		return nil
	}

	// Get policies with associated VPP apps for the team.
	policiesWithVPP, err := svc.ds.GetPoliciesWithAssociatedVPP(ctx, policyTeamID, incomingFailingPoliciesIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "failed to get policies with installer")
	}
	if len(policiesWithVPP) == 0 {
		return nil
	}

	// Filter to policies with VPP apps that are newly failing, or that have
	// continuous_automations_enabled set (in which case every failing result
	// triggers an install, not just pass→fail transitions).
	//
	// An app can be added for several platforms and GetPoliciesWithAssociatedVPP filters on neither
	// the app's platform nor the host's, so a policy bound to the iOS build can arrive here for a
	// macOS host. Dropping those now rather than in the install loop lets the return below skip the
	// host, the token and three lookups. processSoftwareForNewlyFailingPolicies makes the same check.
	var failingPoliciesWithVPP []fleet.PolicyVPPData
	for _, policyWithVPP := range policiesWithVPP {
		if _, ok := newFailingSet[policyWithVPP.ID]; !ok && !policyWithVPP.ContinuousAutomationsEnabled {
			continue
		}
		if fleet.PlatformFromHost(hostPlatform) != string(policyWithVPP.Platform) {
			svc.logger.DebugContext(ctx, "app platform does not match host platform",
				"host_id", hostID,
				"policy_id", policyWithVPP.ID,
				"vpp_adam_id", policyWithVPP.AdamID,
				"vpp_platform", policyWithVPP.Platform,
			)
			continue
		}
		failingPoliciesWithVPP = append(failingPoliciesWithVPP, policyWithVPP)
	}
	if len(failingPoliciesWithVPP) == 0 {
		return nil
	}

	host, err := svc.ds.Host(ctx, hostID)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "failed to get host details")
	}
	vppToken, err := svc.EnterpriseOverrides.GetVPPTokenIfCanInstallVPPApps(ctx, true, host)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "host is not able to install VPP apps")
	}

	pendingAppInstalls, err := svc.ds.MapAdamIDsPendingInstall(ctx, hostID)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "failed to check pending VPP installs")
	}

	// The pending lookup only matches install commands that haven't been delivered yet, so it
	// misses an install that is awaiting verification or waiting behind one that is.
	queuedAppInstalls, err := svc.ds.MapAdamIDsQueuedInstalls(ctx, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "failed to check queued VPP installs")
	}

	// Apps successfully installed within the policy update interval are used to throttle
	// continuous policy automation re-installs (see continuousAutomationOnCooldown).
	recentAppInstalls, err := svc.ds.MapAdamIDsRecentlyVerifiedInstalls(ctx, hostID, int(svc.config.Osquery.PolicyUpdateInterval.Seconds()))
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "failed to check recent VPP installs")
	}

	// When two policies are bound to one app only the first queues an install, so sort to make that
	// choice stable. Sorted here rather than in GetPoliciesWithAssociatedVPP so it stays verifiable
	// without a live database.
	slices.SortFunc(failingPoliciesWithVPP, func(a, b fleet.PolicyVPPData) int {
		return cmp.Compare(a.ID, b.ID)
	})

	for _, failingPolicyWithVPP := range failingPoliciesWithVPP {
		policyID := failingPolicyWithVPP.ID
		_, newlyFailing := newFailingSet[policyID]
		logger := svc.logger.With(
			"host_id", hostID,
			"host_platform", hostPlatform,
			"policy_id", policyID,
			"vpp_adam_id", failingPolicyWithVPP.AdamID,
			"vpp_platform", failingPolicyWithVPP.Platform,
			"continuous_automations_enabled", failingPolicyWithVPP.ContinuousAutomationsEnabled,
		)

		vppMetadata, err := svc.ds.GetVPPAppMetadataByAdamIDPlatformTeamID(ctx, failingPolicyWithVPP.AdamID, failingPolicyWithVPP.Platform, host.TeamID)
		if err != nil {
			logger.ErrorContext(ctx, "failed to get VPP metadata",
				"err", err,
			)
			continue
		}

		scoped, err := svc.ds.IsVPPAppLabelScoped(ctx, vppMetadata.VPPAppTeam.AppTeamID, hostID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "checking if vpp app is label scoped to host")
		}

		if !scoped {
			// NOTE: we update the policy status here to stop it from showing up as "failed" in the
			// host details.
			incomingPolicyResults[failingPolicyWithVPP.ID] = nil
			logger.DebugContext(ctx, "not marking policy as failed since vpp app is out of scope for host")
			continue
		}

		if _, hasPendingInstall := pendingAppInstalls[failingPolicyWithVPP.AdamID]; hasPendingInstall {
			logger.DebugContext(ctx, "install of app is already pending")
			continue
		}

		// Also covers an install queued by an earlier policy in this run, which is why the successful
		// install below writes back into this map. Two policies can be bound to one app, since
		// policies.vpp_apps_teams_id is not unique, and the lookup was read once above.
		if _, hasQueuedInstall := queuedAppInstalls[failingPolicyWithVPP.AdamID]; hasQueuedInstall {
			logger.DebugContext(ctx, "install of app is already queued")
			continue
		}

		// Throttle continuous policy automation re-installs: if this policy fired only
		// because continuous_automations_enabled is set (not a pass→fail transition)
		// and the VPP app was successfully installed (verified) within the policy update
		// interval, skip it. A successful VPP install requests a host refetch, which
		// re-runs policies immediately; without this a perpetually-failing policy would
		// loop tightly.
		if _, recentlyInstalled := recentAppInstalls[failingPolicyWithVPP.AdamID]; !newlyFailing && failingPolicyWithVPP.ContinuousAutomationsEnabled && recentlyInstalled {
			logger.InfoContext(ctx, "skipping continuous policy automation vpp install; within policy update interval cooldown")
			continue
		}

		commandUUID, err := svc.EnterpriseOverrides.InstallVPPAppPostValidation(ctx, host, vppMetadata, vppToken, fleet.HostSoftwareInstallOptions{
			SelfService:     false,
			PolicyID:        &policyID,
			DeferActivation: svc.deferFleetInitiatedActivation(),
		})
		if err != nil {
			logger.ErrorContext(ctx, "failed to get install VPP app",
				"err", err,
			)
			continue
		}

		queuedAppInstalls[failingPolicyWithVPP.AdamID] = struct{}{}
		logger.DebugContext(ctx, "vpp install request sent", "command_uuid", commandUUID)
	}

	return nil
}

func (svc *Service) processProfileResendsForNewlyFailingPolicies(
	ctx context.Context,
	host *fleet.Host,
	incomingPolicyResults map[uint]*bool,
	newFailingSet map[uint]struct{},
) error {
	// While it's gated outside, we gate in here as well to avoid future callers not gating.
	if host.Platform != "darwin" && host.Platform != "windows" {
		return nil
	}

	var policyTeamID uint
	if host.TeamID == nil {
		policyTeamID = fleet.PolicyNoTeamID
	} else {
		policyTeamID = *host.TeamID
	}

	// Only trigger resend on pass->fail or fresh failures.
	var newlyFailingPolicyIDs []uint
	for policyID, policyResult := range incomingPolicyResults {
		if policyResult == nil || *policyResult {
			continue
		}
		if _, newlyFailing := newFailingSet[policyID]; !newlyFailing {
			continue
		}
		newlyFailingPolicyIDs = append(newlyFailingPolicyIDs, policyID)
	}
	if len(newlyFailingPolicyIDs) == 0 {
		return nil
	}

	policiesWithProfile, err := svc.ds.GetPoliciesWithAssociatedProfile(ctx, policyTeamID, newlyFailingPolicyIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "failed to get policies with associated profile")
	}
	svc.logger.DebugContext(ctx, "looked up profiles to resend for newly failing policies",
		"host_id", host.ID,
		"team_id", policyTeamID,
		"newly_failing", newlyFailingPolicyIDs,
		"with_profile", len(policiesWithProfile),
	)
	if len(policiesWithProfile) == 0 {
		return nil
	}

	for _, profile := range policiesWithProfile {
		var reported bool
		onError := func(innerErr error, rejected bool) {
			reported = true
			if rejected {
				svc.logger.DebugContext(ctx, "skipping resend of MDM profile for host",
					"host_id", host.ID,
					"host_platform", host.Platform,
					"policy_id", profile.PolicyID,
					"profile_uuid", profile.ProfileUUID,
					"err", innerErr,
				)
				return
			}
			svc.logger.ErrorContext(ctx, "failed to resend MDM profile for host",
				"host_id", host.ID,
				"policy_id", profile.PolicyID,
				"profile_uuid", profile.ProfileUUID,
				"err", innerErr,
			)
		}
		svc.logger.DebugContext(ctx, "attempting resend of MDM profile for newly failing policy",
			"host_id", host.ID,
			"host_uuid", host.UUID,
			"policy_id", profile.PolicyID,
			"policy_name", profile.PolicyName,
			"profile_uuid", profile.ProfileUUID,
			"profile_name", profile.ProfileName,
		)
		checkAndResendHostMDMProfile(ctx, svc, host, onError, profile.ProfileUUID, profile.ProfileName, &checkAndResendPolicyArgs{
			PolicyID:   profile.PolicyID,
			PolicyName: profile.PolicyName,
		})
		if !reported {
			// Nothing went to onError, so the profile is queued for the profile schedule to pick up
			// and the activity is recorded.
			svc.logger.DebugContext(ctx, "queued MDM profile for resend",
				"host_id", host.ID,
				"policy_id", profile.PolicyID,
				"profile_uuid", profile.ProfileUUID,
			)
		}
	}

	return nil
}

func (svc *Service) processScriptsForNewlyFailingPolicies(
	ctx context.Context,
	hostID uint,
	hostTeamID *uint,
	hostPlatform string,
	hostOrbitNodeKey *string,
	hostScriptsEnabled *bool,
	incomingPolicyResults map[uint]*bool,
	newFailingSet map[uint]struct{},
) error {
	if hostOrbitNodeKey == nil || *hostOrbitNodeKey == "" {
		return nil // vanilla osquery hosts can't run scripts
	}
	// not logging here to avoid spamming logs on every policy failure for every no-scripts host even if the policy
	// doesn't have a script attached
	if hostScriptsEnabled != nil && !*hostScriptsEnabled {
		return nil
	}

	// Bail if scripts are disabled globally
	cfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return err
	}
	if cfg.ServerSettings.ScriptsDisabled {
		return nil
	}

	var policyTeamID uint
	if hostTeamID == nil {
		policyTeamID = fleet.PolicyNoTeamID
	} else {
		policyTeamID = *hostTeamID
	}

	// Filter out results that are not failures (we are only interested on failing policies,
	// we don't care about passing policies or policies that failed to execute).
	var incomingFailingPoliciesIDs []uint
	for policyID, policyResult := range incomingPolicyResults {
		if policyResult != nil && !*policyResult {
			incomingFailingPoliciesIDs = append(incomingFailingPoliciesIDs, policyID)
		}
	}
	if len(incomingFailingPoliciesIDs) == 0 {
		return nil
	}

	// Get policies with associated scripts for the team.
	policiesWithScript, err := svc.ds.GetPoliciesWithAssociatedScript(ctx, policyTeamID, incomingFailingPoliciesIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "failed to get policies with script")
	}
	if len(policiesWithScript) == 0 {
		return nil
	}

	// Filter to policies with scripts that are newly failing, or that have
	// continuous_automations_enabled set (in which case every failing result
	// triggers a script run, not just pass→fail transitions).
	var failingPoliciesWithScript []fleet.PolicyScriptData
	for _, policyWithScript := range policiesWithScript {
		if _, ok := newFailingSet[policyWithScript.ID]; ok || policyWithScript.ContinuousAutomationsEnabled {
			failingPoliciesWithScript = append(failingPoliciesWithScript, policyWithScript)
		}
	}
	if len(failingPoliciesWithScript) == 0 {
		return nil
	}

	for _, failingPolicyWithScript := range failingPoliciesWithScript {
		policyID := failingPolicyWithScript.ID

		scriptMetadata, err := svc.ds.Script(ctx, failingPolicyWithScript.ScriptID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get script metadata by id")
		}
		logger := svc.logger.With(
			"host_id", hostID,
			"host_platform", hostPlatform,
			"policy_id", policyID,
			"script_id", failingPolicyWithScript.ScriptID,
			"script_name", scriptMetadata.Name,
		)

		allScriptsExecutionPending, err := svc.ds.ListPendingHostScriptExecutions(ctx, hostID, false)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "list host pending script executions")
		}
		if len(allScriptsExecutionPending) > maxPendingScripts {
			logger.WarnContext(ctx, "too many scripts pending for host")
			return nil
		}

		// skip incompatible scripts
		hostPlatform := fleet.PlatformFromHost(hostPlatform)
		if (hostPlatform == "windows" && strings.HasSuffix(scriptMetadata.Name, ".sh")) ||
			(hostPlatform != "windows" && strings.HasSuffix(scriptMetadata.Name, ".ps1")) {
			logger.InfoContext(ctx, "script type does not match host platform")
			continue
		}

		// skip different-team scripts
		var scriptTeamID uint
		if scriptMetadata.TeamID != nil {
			scriptTeamID = *scriptMetadata.TeamID
		}
		if policyTeamID != scriptTeamID { // this should not happen
			logger.ErrorContext(ctx, "script team does not match host team")
			continue
		}

		scriptIsAlreadyPending, err := svc.ds.IsExecutionPendingForHost(ctx, hostID, scriptMetadata.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "check whether script is pending execution")
		}
		if scriptIsAlreadyPending {
			logger.DebugContext(ctx, "script is already pending on host")
			continue
		}

		// On a continuous re-fire (policy still failing), reset prior
		// attempt_number values for this host/policy to 0 so the new attempt
		// restarts the retry sequence at 1 instead of inheriting the cap from
		// the previous sequence. A no-op on pass→fail transitions (those rows
		// are already at 0 from the prior fail→pass reset).
		if failingPolicyWithScript.ContinuousAutomationsEnabled {
			if err := svc.ds.ResetPolicyAutomationRetryAttemptsForHost(ctx, hostID, []uint{policyID}); err != nil {
				return ctxerr.Wrap(ctx, err, "reset policy automation retry attempts for host")
			}
		}

		contents, err := svc.ds.GetScriptContents(ctx, scriptMetadata.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get script contents")
		}
		runScriptRequest := fleet.HostScriptRequestPayload{
			HostID:          hostID,
			ScriptContents:  string(contents),
			ScriptContentID: scriptMetadata.ScriptContentID,
			ScriptID:        &scriptMetadata.ID,
			TeamID:          policyTeamID,
			PolicyID:        &policyID,
			DeferActivation: svc.deferFleetInitiatedActivation(),
			// no user ID as scripts are executed by Fleet
		}

		scriptResult, err := svc.ds.NewHostScriptExecutionRequest(ctx, &runScriptRequest)
		if err != nil {
			return ctxerr.Wrapf(ctx, err,
				"insert script run request; host_id=%d, script_id=%d",
				hostID, scriptMetadata.ID,
			)
		}

		logger.DebugContext(ctx, "script run request sent",
			"execution_id", scriptResult.ExecutionID,
		)
	}

	return nil
}

func (svc *Service) conditionalAccessConfiguredAndEnabledForTeam(ctx context.Context, hostTeamID *uint) (configured bool, enabledForTeam bool, err error) {
	// Conditional access is a Fleet Premium feature. Gate on the current license
	// tier so that an integration left over from a previous Premium license
	// (e.g. after a downgrade or expiry) doesn't keep the feature active.
	if !license.IsPremium(ctx) {
		return false, false, nil
	}

	// Check if the integration is fully configured.
	integration, err := svc.ds.ConditionalAccessMicrosoftGet(ctx)
	if err != nil {
		if fleet.IsNotFound(err) {
			return false, false, nil
		}
		return false, false, ctxerr.Wrap(ctx, err, "failed to load the integration")
	}
	if !integration.SetupDone {
		return false, false, nil
	}

	if hostTeamID == nil {
		// Configuration for "No team" is stored in the main appconfig.
		cfg, err := svc.ds.AppConfig(ctx)
		if err != nil {
			return false, false, ctxerr.Wrap(ctx, err, "failed to load appconfig")
		}
		var conditionalAccessEnabled bool
		if cfg.Integrations.ConditionalAccessEnabled.Set {
			conditionalAccessEnabled = cfg.Integrations.ConditionalAccessEnabled.Value
		}
		return true, conditionalAccessEnabled, nil
	}

	// Host belongs to a team, thus we load the team configuration.
	team, err := svc.ds.TeamLite(ctx, *hostTeamID)
	if err != nil {
		return false, false, ctxerr.Wrap(ctx, err, "failed to load team config")
	}
	var teamConditionalAccessEnabled bool
	if team.Config.Integrations.ConditionalAccessEnabled.Set {
		teamConditionalAccessEnabled = team.Config.Integrations.ConditionalAccessEnabled.Value
	}
	return true, teamConditionalAccessEnabled, nil
}

func (svc *Service) processConditionalAccessForNewlyFailingPolicies(
	ctx context.Context,
	hostID uint,
	hostTeamID *uint,
	hostOrbitNodeKey *string,
	hostPlatform string,
	incomingPolicyResults map[uint]*bool,
) error {
	if hostOrbitNodeKey == nil || *hostOrbitNodeKey == "" {
		// Vanilla osquery hosts cannot do conditional access.
		return nil
	}

	configured, enabledForTeam, err := svc.conditionalAccessConfiguredAndEnabledForTeam(ctx, hostTeamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "failed to check for conditional access configuration")
	}

	if !configured || !enabledForTeam {
		// Nothing to do, feature not configured or not enabled for this host's team.
		return nil
	}

	hostConditionalAccessStatus, err := svc.ds.LoadHostConditionalAccessStatus(ctx, hostID)
	if err != nil {
		if fleet.IsNotFound(err) {
			// Nothing to do because Fleet hasn't ingested the Entra's "Device ID" or
			// "User Principal Name" from the device yet (we cannot perform any actions
			// for the host on Entra without it).
			return nil
		}
		return ctxerr.Wrap(ctx, err, "failed to load host conditional access status")
	}

	var policyTeamID uint
	if hostTeamID == nil {
		policyTeamID = fleet.PolicyNoTeamID
	} else {
		policyTeamID = *hostTeamID
	}

	var mdmEnrolled bool
	hostMDM, err := svc.ds.GetHostMDM(ctx, hostID)
	if err != nil {
		// If GetHostMDM returns not found then it means that
		// the host may not be MDM enrolled yet.
		if !fleet.IsNotFound(err) {
			return ctxerr.Wrap(ctx, err, "failed to get host mdm")
		}
	} else {
		mdmEnrolled = hostMDM.Enrolled
	}

	// Get policies configured for conditional access.
	conditionalAccessPolicyIDs, err := svc.ds.GetPoliciesForConditionalAccess(ctx, policyTeamID, hostPlatform)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "failed to get policies with conditional access")
	}

	hostIsCompliantInFleet := true
	conditionalAccessPolicyIDsSet := make(map[uint]struct{}, len(conditionalAccessPolicyIDs))
	for _, policyID := range conditionalAccessPolicyIDs {
		conditionalAccessPolicyIDsSet[policyID] = struct{}{}
	}
	var failingCAIDs []uint
	for incomingPolicyID, incomingPolicyResult := range incomingPolicyResults {
		if _, ok := conditionalAccessPolicyIDsSet[incomingPolicyID]; !ok {
			// Ignore results for policies that are not for conditional access.
			continue
		}
		if incomingPolicyResult != nil && !*incomingPolicyResult {
			failingCAIDs = append(failingCAIDs, incomingPolicyID)
			hostIsCompliantInFleet = false
		}
	}

	if hostConditionalAccessStatus.Managed != nil && mdmEnrolled == *hostConditionalAccessStatus.Managed &&
		hostConditionalAccessStatus.Compliant != nil && hostIsCompliantInFleet == *hostConditionalAccessStatus.Compliant {
		// Nothing to do, nothing has changed.
		return nil
	}

	svc.setHostConditionalAccessAsync(hostID, hostPlatform, hostConditionalAccessStatus, mdmEnrolled, hostIsCompliantInFleet, failingCAIDs)

	return nil
}

func (svc *Service) setHostConditionalAccessAsync(
	hostID uint,
	hostPlatform string,
	hostConditionalAccessStatus *fleet.HostConditionalAccessStatus,
	managed bool,
	compliant bool,
	failingPolicyIDs []uint,
) {
	go func() {
		logger := svc.logger.With(
			"host_id", hostID,
			"platform", hostPlatform,
			"managed", managed,
			"compliant", compliant,
		)
		start := time.Now()
		if err := svc.setHostConditionalAccess(hostID, hostPlatform, hostConditionalAccessStatus, managed, compliant, failingPolicyIDs); err != nil {
			logger.ErrorContext(context.TODO(), "set host conditional access", "took", time.Since(start), "err", err)
		}
		logger.DebugContext(context.TODO(), "set host conditional access", "took", time.Since(start))
	}()
}

// conditionalAccessSetWaitTime is the interval to check for message status.
// It's a global variable to be set in tests.
var conditionalAccessSetWaitTime = 10 * time.Second

func (svc *Service) setHostConditionalAccess(
	hostID uint,
	hostPlatform string,
	hostConditionalAccessStatus *fleet.HostConditionalAccessStatus,
	managed bool,
	compliant bool,
	failingPolicyIDs []uint,
) error {
	ctx := context.Background()

	integration, err := svc.ds.ConditionalAccessMicrosoftGet(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get integration")
	}
	logger := svc.logger.With(
		"host_id", hostID,
		"platform", hostPlatform,
		"managed", managed,
		"compliant", compliant,
	)
	logger.DebugContext(ctx, "set compliance status")

	// Currently, only macOS and Windows are supported.
	osName := "macOS" // "macOS" is what Entra requires for darwin hosts.
	if hostPlatform == "windows" {
		osName = "windows"
	}

	response, err := svc.conditionalAccessMicrosoftProxy.SetComplianceStatus(ctx,
		integration.TenantID,
		integration.ProxyServerSecret,

		hostConditionalAccessStatus.DeviceID,
		hostConditionalAccessStatus.UserPrincipalName,

		managed,
		hostConditionalAccessStatus.DisplayName,
		osName,
		hostConditionalAccessStatus.OSVersion,
		compliant,
		time.Now().UTC(),
	)
	if err != nil {
		recordConditionalAccessFailureActivity(ctx, svc.activitySvc, hostID, failingPolicyIDs, err, logger)
		return ctxerr.Wrap(ctx, err, "failed to set compliance status")
	}

	//
	// The macOS API is asynchronous, the Windows API is not.
	// So we only need to retrieve the status of the "request" for macOS hosts.
	//

	if hostPlatform == "darwin" {
		const (
			timeout = 1 * time.Minute
		)
		logger.DebugContext(ctx, "set compliance status message sent")
		startTime := time.Now()
		for range time.Tick(conditionalAccessSetWaitTime) {
			if time.Since(startTime) > timeout {
				// No failure activity is recorded here. SetComplianceStatus
				// succeeded (we have a MessageID), so the push was accepted by
				// the remote provider; we just could not confirm completion
				// within the expected window. Recording a
				// failed_automation_conditional_access here would
				// misrepresent an in-flight async operation as a rejection.
				return ctxerr.Errorf(ctx, "timeout waiting for message after %s", time.Since(startTime))
			}
			logger.DebugContext(ctx, "get compliance status message wait")
			messageStatus, err := svc.conditionalAccessMicrosoftProxy.GetMessageStatus(ctx,
				integration.TenantID, integration.ProxyServerSecret, response.MessageID,
			)
			if err != nil {
				// Retry again in case of network or transient errors.
				logger.InfoContext(ctx, "get message status, retrying", "err", err)
				continue
			}
			if messageStatus.Status == conditional_access_microsoft_proxy.MessageStatusCompleted {
				logger.DebugContext(ctx, "set device compliance status completed",
					"took", time.Since(startTime),
				)
				break
			}
			detail := ""
			if messageStatus.Detail != nil {
				detail = *messageStatus.Detail
			}
			logger.InfoContext(ctx, "get message status, retrying",
				"status", messageStatus.Status,
				"detail", detail,
			)
		}
	}

	if err := svc.ds.SetHostConditionalAccessStatus(ctx, hostID, managed, compliant); err != nil {
		return ctxerr.Wrap(ctx, err, "set conditional access status on datastore")
	}

	if !compliant {
		// The host was pushed non-compliant, which blocks single sign-on. The
		// push has been accepted (and, for macOS, confirmed) at this point.
		recordSingleSignOnBlockedActivity(ctx, svc.activitySvc, hostID, failingPolicyIDs, logger)
	}

	return nil
}

// recordConditionalAccessFailureActivity records a
// failed_automation_conditional_access activity for the given host when
// a compliance push to the remote provider fails. One activity is recorded per
// failing conditional-access policy (policies the host is currently failing,
// not all CA policies configured for the team), capturing the remote status
// code and response body when available. Failures to record are logged and
// swallowed so they don't mask the original error.
func recordConditionalAccessFailureActivity(
	ctx context.Context,
	newActivitySvc activity_api.NewActivityService,
	hostID uint,
	policyIDs []uint,
	err error,
	logger *slog.Logger,
) {
	if len(policyIDs) == 0 {
		return
	}

	var statusCode int
	if sc, ok := errors.AsType[interface {
		error
		StatusCode() int
	}](err); ok {
		statusCode = sc.StatusCode()
	}

	errResponse := ""
	if b, ok := errors.AsType[interface {
		error
		Body() string
	}](err); ok {
		errResponse = b.Body()
	}
	if errResponse == "" {
		// network-level failures (e.g. connection refused) have no server
		// response; fall back to the error message.
		errResponse = err.Error()
	}
	for _, policyID := range policyIDs {
		if actErr := newActivitySvc.NewActivity(ctx, nil, fleet.ActivityTypeFailedAutomationConditionalAccess{
			PolicyID:      policyID,
			HostIDList:    []uint{hostID},
			StatusCode:    statusCode,
			ErrorResponse: errResponse,
		}); actErr != nil {
			logger.WarnContext(ctx, "failed to record conditional access policy automation failure activity",
				"policy_id", policyID, "host_id", hostID, "err", actErr)
		}
	}
}

// recordSingleSignOnBlockedActivity records a
// ran_automation_conditional_access activity for the given host once its
// non-compliant status has been successfully pushed to the remote provider,
// blocking single sign-on. One activity is recorded per conditional-access
// policy the host is failing. Failures to record are logged and swallowed so
// they don't affect the compliance push.
func recordSingleSignOnBlockedActivity(
	ctx context.Context,
	newActivitySvc activity_api.NewActivityService,
	hostID uint,
	policyIDs []uint,
	logger *slog.Logger,
) {
	if newActivitySvc == nil || len(policyIDs) == 0 {
		return
	}
	for _, policyID := range policyIDs {
		if actErr := newActivitySvc.NewActivity(ctx, nil, fleet.ActivityTypeRanAutomationConditionalAccess{
			PolicyID:   policyID,
			HostIDList: []uint{hostID},
		}); actErr != nil {
			logger.WarnContext(ctx, "failed to record single sign-on blocked policy automation activity",
				"policy_id", policyID, "host_id", hostID, "err", actErr)
		}
	}
}

func (svc *Service) maybeDebugHost(
	ctx context.Context,
	host *fleet.Host,
	results fleet.OsqueryDistributedQueryResults,
	statuses map[string]fleet.OsqueryStatus,
	messages map[string]string,
	stats map[string]*fleet.Stats,
) {
	if svc.debugEnabledForHost(ctx, host.ID) {
		hlogger := svc.logger.With("host-id", host.ID)

		logJSON(ctx, hlogger, host, "host")
		logJSON(ctx, hlogger, results, "results")
		logJSON(ctx, hlogger, statuses, "statuses")
		logJSON(ctx, hlogger, messages, "messages")
		logJSON(ctx, hlogger, stats, "stats")
	}
}

////////////////////////////////////////////////////////////////////////////////
// Submit Logs
////////////////////////////////////////////////////////////////////////////////

type submitLogsRequest struct {
	NodeKey string            `json:"node_key"`
	LogType string            `json:"log_type"`
	Data    []json.RawMessage `json:"data"`
}

func (r *submitLogsRequest) hostNodeKey() string {
	return r.NodeKey
}

type submitLogsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r submitLogsResponse) Error() error { return r.Err }

func submitLogsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*submitLogsRequest)

	var err error
	switch req.LogType {
	case "status":
		err = svc.SubmitStatusLogs(ctx, req.Data)
		if err != nil {
			break
		}

	case "result":
		logging.WithExtras(ctx, "results", len(req.Data))

		// We currently return errors to osqueryd if there are any issues submitting results
		// to the configured external destinations.
		if err = svc.SubmitResultLogs(ctx, req.Data); err != nil {
			break
		}

	default:
		err = newOsqueryError("unknown log type: " + req.LogType)
	}

	return submitLogsResponse{Err: err}, nil
}

// preProcessOsqueryResults will attempt to unmarshal `osqueryResults` and will return:
//   - `unmarshaledResults` with each result unmarshaled to `fleet.ScheduledQueryResult`s, where if an item is `nil` it means the corresponding
//     `osqueryResults` item could not be unmarshaled.
//   - queriesDBData has the corresponding DB query to each unmarshalled result in `osqueryResults`.
//
// Results are resolved to their DB query regardless of queryReportsDisabled, because the caller
// needs the query IDs to check them against the host's schedule either way. queryReportsDisabled
// only suppresses injecting `query_id` into the raw logs, to keep the payload reaching the logging
// destination unchanged for deployments that disable reports.
// maxDistinctQueryNamesPerSubmission bounds how many distinct query names a
// single result submission will resolve. It sits far above any realistic host
// schedule (global plus one team's scheduled queries) and exists only to cap the
// work a compromised/malicious host can force, since the request body size is
// unbounded in header-auth mode. A var so tests can force the cap cheaply.
var maxDistinctQueryNamesPerSubmission = 10000

func (svc *Service) preProcessOsqueryResults(
	ctx context.Context,
	osqueryResults []json.RawMessage,
	queryReportsDisabled bool,
) (unmarshaledResults []*fleet.ScheduledQueryResult, queriesDBData map[string]*fleet.Query) {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	lograw := func(raw json.RawMessage) string {
		logr := raw
		if len(raw) >= 64 {
			logr = raw[:64]
		}
		return string(logr)
	}

	for _, raw := range osqueryResults {
		var result *fleet.ScheduledQueryResult
		if err := json.Unmarshal(raw, &result); err != nil {
			svc.logger.DebugContext(ctx, "unmarshalling result", "err", err, "result", lograw(raw))
			// Note that if err != nil we have two scenarios:
			// 	- result == nil: which means the result could not be unmarshalled, e.g. not JSON.
			//	- result != nil: which means that the result was (partially) unmarshalled but some specific
			// 	field could not be unmarshalled.
			//
			// In both scenarios we want to add `result` to `unmarshaledResults`.
		} else if result != nil && result.QueryName == "" {
			// If the unmarshaled result doesn't have a "name" field then we ignore the result.
			svc.logger.DebugContext(ctx, "missing name field", "result", lograw(raw))
			result = nil
		}
		unmarshaledResults = append(unmarshaledResults, result)
	}

	queriesDBData = make(map[string]*fleet.Query)

	// A host controls the result names it sends, and each name needs its query
	// looked up. Parse every distinct name once and resolve them all in a single
	// batch lookup, so a submission carrying many (or many repeated non-existent)
	// names costs one query instead of one round-trip per result entry.
	type parsedName struct {
		scope fleet.TeamScopedQueryName
		ok    bool
		// capped marks a name left unresolved because the submission hit the
		// distinct-name cap, as opposed to one Fleet does not know. The two must
		// stay distinguishable: unknown names pass through, capped ones drop.
		capped bool
	}
	parsedByRaw := make(map[string]parsedName)
	var toResolve []fleet.TeamScopedQueryName
	seenScope := make(map[string]struct{})
	var cappedNames int
	for _, queryResult := range unmarshaledResults {
		if queryResult == nil {
			continue
		}
		if _, done := parsedByRaw[queryResult.QueryName]; done {
			continue
		}
		teamID, queryName, err := getQueryNameAndTeamIDFromResult(queryResult.QueryName)
		if errors.Is(err, fleet.ErrLegacyQueryPack) {
			// Legacy query. Cannot be stored and cannot infer team ID, but still
			// used by some customers.
			parsedByRaw[queryResult.QueryName] = parsedName{}
			continue
		}
		if err != nil {
			svc.logger.DebugContext(ctx, "querying name and team ID from result", "err", err)
			parsedByRaw[queryResult.QueryName] = parsedName{}
			continue
		}
		scope := fleet.TeamScopedQueryName{TeamID: teamID, Name: queryName}
		parsedByRaw[queryResult.QueryName] = parsedName{scope: scope, ok: true}
		if _, dup := seenScope[scope.Key()]; !dup {
			// Bound the number of distinct names resolved per submission,
			// independent of any request body-size limit (which does not apply in
			// header-auth mode). A real host's schedule is far below this; names
			// past the cap are treated as unresolved, so results still stream to
			// the log destination but skip report attribution.
			if len(toResolve) >= maxDistinctQueryNamesPerSubmission {
				cappedNames++
				parsedByRaw[queryResult.QueryName] = parsedName{capped: true}
				continue
			}
			seenScope[scope.Key()] = struct{}{}
			toResolve = append(toResolve, scope)
		}
	}
	// Count the names actually dropped rather than comparing against the cap, so
	// a submission that lands exactly on it does not report a breach it didn't
	// cause.
	if cappedNames > 0 {
		var hostID uint
		if host, ok := hostctx.FromContext(ctx); ok && host != nil {
			hostID = host.ID
		}
		svc.logger.WarnContext(ctx, "osquery result submission exceeded distinct query name cap; excess names left unresolved",
			"host_id", hostID, "cap", maxDistinctQueryNamesPerSubmission, "unresolved", cappedNames)
	}

	resolved, err := svc.ds.QueriesByName(ctx, toResolve)
	if err != nil {
		// Keep whatever resolved before the failure, so one failing chunk doesn't
		// leave the whole submission unresolved. Names still unresolved here are
		// treated as unknown to Fleet, which passes their results through to the
		// log destination without a schedule check.
		svc.logger.ErrorContext(ctx, "batch loading queries by name", "err", err)
		if resolved == nil {
			resolved = map[string]*fleet.Query{}
		}
	}

	for i, queryResult := range unmarshaledResults {
		if queryResult == nil {
			// These are results that could not be unmarshaled.
			continue
		}
		parsed := parsedByRaw[queryResult.QueryName]
		if parsed.capped {
			// Fail closed. A name Fleet declined to resolve must not inherit the
			// pass-through that names Fleet genuinely doesn't know get below,
			// otherwise filling the cap with junk would launder results for a
			// real query past the host's schedule check.
			unmarshaledResults[i] = nil
			continue
		}
		if !parsed.ok {
			continue
		}
		existingQuery, foundQuery := resolved[parsed.scope.Key()]
		if !foundQuery {
			// Name does not exist on this team.
			continue
		}
		queriesDBData[queryResult.QueryName] = existingQuery

		if queryReportsDisabled {
			continue
		}

		updatedResult, err := addQueryIDToLogResult(ctx, osqueryResults[i], existingQuery.ID)
		if err != nil {
			svc.logger.DebugContext(ctx, "inserting query id into query result", "err", err, "query_id", existingQuery.ID)
			continue
		}

		// Set the updated query results if we find query ID. This is used one level up by the logger
		osqueryResults[i] = updatedResult
	}
	return unmarshaledResults, queriesDBData
}

func addQueryIDToLogResult(ctx context.Context, logResult json.RawMessage, queryID uint) (json.RawMessage, error) {
	var query map[string]json.RawMessage
	if err := json.Unmarshal(logResult, &query); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "unable to unmarshal query result to insert query id")
	}

	query["query_id"] = json.RawMessage(strconv.FormatUint(uint64(queryID), 10))
	newResult, err := json.Marshal(query)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "unable to marshal query result with query id")
	}
	return newResult, nil
}

func (svc *Service) SubmitStatusLogs(ctx context.Context, logs []json.RawMessage) error {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	if err := svc.osqueryLogWriter.Status.Write(ctx, logs); err != nil {
		osqueryErr := newOsqueryError("error writing status logs: " + err.Error())
		// Attempting to write a large amount of data is the most likely explanation for this error.
		osqueryErr.StatusCode = http.StatusRequestEntityTooLarge
		return osqueryErr
	}
	return nil
}

func (svc *Service) SubmitResultLogs(ctx context.Context, logs []json.RawMessage) error {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	//
	// We do not return errors to osqueryd when processing results because
	// otherwise the results will never clear from its local DB and
	// will keep retrying forever.
	//
	// We do return errors if we fail to write to the external logging destination,
	// so that the logs are not lost and osquery retries on its next log interval.
	//

	var queryReportsDisabled bool
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		svc.logger.ErrorContext(ctx, "getting app config", "err", err)
		// If we fail to load the app config we assume the flag to be disabled so that
		// results are not stored as reports in that scenario. The schedule check below
		// still runs, since it does not depend on the app config.
		queryReportsDisabled = true
	} else {
		queryReportsDisabled = appConfig.ServerSettings.QueryReportsDisabled
	}

	unmarshaledResults, queriesDBData := svc.preProcessOsqueryResults(ctx, logs, queryReportsDisabled)

	// A host is the only source of its own results, so Fleet cannot tell truthful rows
	// from forged ones. What it can require is that it asked this host for them, which
	// happens here so that results for queries missing from the host's schedule reach
	// neither a report nor a log destination. Query reports being disabled removes the
	// report destination but not the log one, so the check applies either way.
	svc.dropResultsNotScheduledForHost(ctx, unmarshaledResults, queriesDBData)

	if !queryReportsDisabled {
		maxQueryReportRows := appConfig.ServerSettings.GetQueryReportCap()
		svc.saveResultLogsToQueryReports(ctx, unmarshaledResults, queriesDBData, maxQueryReportRows)
	}

	var filteredLogs []json.RawMessage
	for i, unmarshaledResult := range unmarshaledResults {
		if unmarshaledResult == nil {
			// Ignore results that could not be unmarshaled, and those dropped above for not
			// being on the host's schedule.
			continue
		}

		if queryReportsDisabled {
			// If query_reports_disabled=true we write the logs to the logging destination without
			// any processing beyond the schedule check above.
			//
			// If a query was recently configured with automations_enabled = 0 we may still write
			// the results for it here. Eventually the query will be removed from the host schedule
			// and thus Fleet won't receive any further results anymore.
			filteredLogs = append(filteredLogs, logs[i])
			continue
		}

		dbQuery, ok := queriesDBData[unmarshaledResult.QueryName]
		if !ok {
			// If Fleet doesn't know of the query we write the logs to the logging destination
			// without any extra processing. This is to support osquery nodes that load their
			// config from elsewhere (e.g. using `--config_plugin=filesystem`).
			//
			// If a query was configured from Fleet but was recently removed, we may still write
			// the results for it here. Eventually the query will be removed from the host schedule
			// and thus Fleet won't receive any further results anymore.
			filteredLogs = append(filteredLogs, logs[i])
			continue
		}

		if !dbQuery.AutomationsEnabled {
			// Ignore results for queries that have automations disabled.
			continue
		}

		filteredLogs = append(filteredLogs, logs[i])
	}

	if len(filteredLogs) == 0 {
		return nil
	}

	if err := svc.osqueryLogWriter.Result.Write(ctx, filteredLogs); err != nil {
		osqueryErr := newOsqueryError(
			"error writing result logs " +
				"(if the logging destination is down, you can reduce frequency/size of osquery logs by " +
				"increasing logger_tls_period and decreasing logger_tls_max_lines): " + err.Error(),
		)
		// Attempting to write a large amount of data is the most likely explanation for this error.
		osqueryErr.StatusCode = http.StatusRequestEntityTooLarge
		return osqueryErr
	}
	return nil
}

// dropResultsNotScheduledForHost sets to nil the results whose query Fleet knows but did
// not put on the submitting host's schedule. Entries are nilled in place rather than
// removed because the caller pairs them positionally with the raw logs.
func (svc *Service) dropResultsNotScheduledForHost(
	ctx context.Context,
	unmarshaledResults []*fleet.ScheduledQueryResult,
	queriesDBData map[string]*fleet.Query,
) {
	if len(queriesDBData) == 0 {
		return
	}

	// Neither failure below stops the loop: they leave the schedule empty, which makes it
	// drop every result that resolved to a Fleet query. With no host or no schedule, no
	// result can be shown to have been asked for.
	var hostID uint
	var scheduledQueryIDs []uint
	// ok is true for a nil host, so check both.
	if host, ok := hostctx.FromContext(ctx); !ok || host == nil {
		svc.logger.ErrorContext(ctx, "getting host from context")
	} else {
		hostID = host.ID
		var err error
		if scheduledQueryIDs, err = svc.ds.QueriesPerHost(ctx, host.ID, host.TeamID); err != nil {
			svc.logger.ErrorContext(ctx, "getting queries scheduled for host", "err", err, "host_id", host.ID)
		}
	}

	scheduled := make(map[uint]struct{}, len(scheduledQueryIDs))
	for _, queryID := range scheduledQueryIDs {
		scheduled[queryID] = struct{}{}
	}

	for i, result := range unmarshaledResults {
		if result == nil {
			continue
		}
		dbQuery, ok := queriesDBData[result.QueryName]
		if !ok {
			// Fleet doesn't know this query, so it has no schedule to check it against. Those
			// results are passed through to support osquery nodes configured outside of Fleet.
			continue
		}
		if _, ok := scheduled[dbQuery.ID]; !ok {
			// The query is not on the host's schedule (no interval, another team, or scoped to
			// labels the host is not a member of), so the results are either forged or stale
			// from before a scoping change.
			svc.logger.DebugContext(ctx, "ignoring results for query not scheduled for host",
				"query_id", dbQuery.ID, "host_id", hostID)
			unmarshaledResults[i] = nil
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// Query Reports
////////////////////////////////////////////////////////////////////////////////

func (svc *Service) saveResultLogsToQueryReports(
	ctx context.Context,
	unmarshaledResults []*fleet.ScheduledQueryResult,
	queriesDBData map[string]*fleet.Query,
	maxQueryReportRows int,
) {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		svc.logger.ErrorContext(ctx, "getting host from context")
		return
	}

	// Transform results that are in "event format" to "snapshot format".
	// This is needed to support query reports for hosts that are configured with `--logger_snapshot_event_type=true`
	// in their agent options.
	unmarshaledResultsFiltered := transformEventFormatToSnapshotFormat(unmarshaledResults)

	// Filter results to only the most recent for each query.
	unmarshaledResultsFiltered = getMostRecentResults(unmarshaledResultsFiltered)

	// Batch fetch query result counts from Redis for all queries
	var queryResultCounts map[uint]int
	if svc.liveQueryStore != nil {
		queryIDs := make([]uint, 0, len(queriesDBData))
		for _, dbQuery := range queriesDBData {
			queryIDs = append(queryIDs, dbQuery.ID)
		}
		var err error
		queryResultCounts, err = svc.liveQueryStore.GetQueryResultsCounts(queryIDs)
		if err != nil {
			svc.logger.ErrorContext(ctx, "get result counts for queries", "err", err)
			return
		}
	}

	// Track rows added per query for batched Redis increment
	rowsAddedByQuery := make(map[uint]int)

	for _, result := range unmarshaledResultsFiltered {
		dbQuery, ok := queriesDBData[result.QueryName]
		if !ok {
			// Means the query does not exist with such name anymore. Thus we ignore its result.
			continue
		}

		if dbQuery.DiscardData || dbQuery.Logging != fleet.LoggingSnapshot {
			// Ignore result if query is marked as discard data or if logging is not snapshot
			continue
		}

		hostTeamID := uint(0)
		if host.TeamID != nil {
			hostTeamID = *host.TeamID
		}
		if dbQuery.TeamID != nil && *dbQuery.TeamID != hostTeamID {
			// The host was transferred to another team/global so we ignore the incoming results
			// of this query that belong to a different team.
			continue
		}

		// Check Redis counter for approximate count (fast, distributed check).
		if queryResultCounts != nil {
			if count := queryResultCounts[dbQuery.ID]; count > maxQueryReportRows {
				continue
			}
		}

		var rowsAdded int
		var err error
		if rowsAdded, err = svc.overwriteResultRows(ctx, result, dbQuery.ID, host.ID, maxQueryReportRows); err != nil {
			svc.logger.ErrorContext(ctx, "overwrite results", "err", err, "query_id", dbQuery.ID, "host_id", host.ID)
			continue
		}

		// Track rows added for batched Redis increment
		rowsAddedByQuery[dbQuery.ID] += rowsAdded
	}

	// Batch increment Redis counters after all successful inserts
	if svc.liveQueryStore != nil && len(rowsAddedByQuery) > 0 {
		if err := svc.liveQueryStore.IncrQueryResultsCounts(rowsAddedByQuery); err != nil {
			// Log but don't fail - the inserts succeeded, counter is just a heuristic
			svc.logger.DebugContext(ctx, "incr query results counts in redis", "err", err)
		}
	}
}

// transformEventFormatToSnapshotFormat transforms results that are in "event format" to "snapshot format".
// This is needed to support query reports for hosts that are configured with `--logger_snapshot_event_type=true`
// in their agent options.
//
// "Snapshot format" contains all of the result rows of the same query on one entry with the "snapshot" field, example:
//
//	[
//		{
//			"snapshot":[
//				{"class":"9","model":"AppleUSBVHCIBCE Root Hub Simulation","model_id":"8007","protocol":"","removable":"0","serial":"0","subclass":"255","usb_address":"","usb_port":"","vendor":"Apple Inc.","vendor_id":"05ac","version":"0.0"},
//				{"class":"9","model":"AppleUSBXHCI Root Hub Simulation","model_id":"8007","protocol":"","removable":"0","serial":"0","subclass":"255","usb_address":"","usb_port":"","vendor":"Apple Inc.","vendor_id":"05ac","version":"0.0"}
//			],
//			"action":"snapshot",
//			"name":"pack/Global/All USB devices",
//			"hostIdentifier":"F5B29579-E946-46A2-BB0F-7A8D1E304940",
//			"calendarTime":"Wed Jan 29 22:17:17 2025 UTC",
//			"unixTime":1738189037,
//			"epoch":0,
//			"counter":0,
//			"numerics":false,
//			"decorations":{"host_uuid":"F5B29579-E946-46A2-BB0F-7A8D1E304940","hostname":"foobar.local"}
//		}
//	]
//
// "Event format" will split result rows of the same query into two separate entries each with its own "columns" field, example with same data as above:
//
//	[
//		{
//			"name":"pack/Global/All USB devices",
//			"hostIdentifier":"F5B29579-E946-46A2-BB0F-7A8D1E304940",
//			"calendarTime":"Wed Jan 29 12:32:54 2025 UTC",
//			"unixTime":1738153974,
//			"epoch":0,
//			"counter":0,
//			"numerics":false,
//			"decorations":{"host_uuid":"F5B29579-E946-46A2-BB0F-7A8D1E304940","hostname":"foobar.local"},
//			"columns": {
//				"class":"9",
//				"model":"AppleUSBVHCIBCE Root Hub Simulation",
//				"model_id":"8007",
//				"protocol":"",
//				"removable":"0",
//				"serial":"0",
//				"subclass":"255",
//				"usb_address":"",
//				"usb_port":"",
//				"vendor":"Apple Inc.",
//				"vendor_id":"05ac",
//				"version":"0.0"
//			},
//			"action":"snapshot"
//		},
//		{
//			"name":"pack/Global/All USB devices",
//			"hostIdentifier":"F5B29579-E946-46A2-BB0F-7A8D1E304940",
//			"calendarTime":"Wed Jan 29 12:32:54 2025 UTC",
//			"unixTime":1738153974,
//			"epoch":0,
//			"counter":0,
//			"numerics":false,
//			"decorations":{"host_uuid":"F5B29579-E946-46A2-BB0F-7A8D1E304940","hostname":"foobar.local"},
//			"columns":{
//				"class":"9",
//				"model":"AppleUSBXHCI Root Hub Simulation",
//				"model_id":"8007",
//				"protocol":"",
//				"removable":"0",
//				"serial":"0",
//				"subclass":"255",
//				"usb_address":"",
//				"usb_port":"",
//				"vendor":"Apple Inc.",
//				"vendor_id":"05ac",
//				"version":"0.0"
//			},
//			"action":"snapshot"
//		}
//	]
func transformEventFormatToSnapshotFormat(results []*fleet.ScheduledQueryResult) []*fleet.ScheduledQueryResult {
	isEventFormat := func(result *fleet.ScheduledQueryResult) bool {
		return result != nil && result.Action == "snapshot" && len(result.Snapshot) == 0 && len(result.Columns) > 0
	}

	resultsInEventFormat := make(map[string]*fleet.ScheduledQueryResult)
	for _, result := range results {
		if !isEventFormat(result) {
			continue
		}
		allResults, ok := resultsInEventFormat[result.QueryName]
		if !ok {
			// All snapshot results in "event format" for the same query have the same `hostIdentifier` and `unixTime`.
			resultsInEventFormat[result.QueryName] = &fleet.ScheduledQueryResult{
				QueryName:     result.QueryName,
				OsqueryHostID: result.OsqueryHostID,
				Snapshot:      []*json.RawMessage{&result.Columns},
				UnixTime:      result.UnixTime,
			}
		} else {
			resultsInEventFormat[allResults.QueryName].Snapshot = append(resultsInEventFormat[allResults.QueryName].Snapshot, &result.Columns)
		}
	}

	if len(resultsInEventFormat) == 0 {
		return results
	}

	replaced := make(map[string]struct{})
	var filteredResults []*fleet.ScheduledQueryResult
	for _, result := range results {
		if isEventFormat(result) {
			if _, ok := replaced[result.QueryName]; !ok {
				filteredResults = append(filteredResults, resultsInEventFormat[result.QueryName])
				replaced[result.QueryName] = struct{}{}
			}
			continue
		}
		filteredResults = append(filteredResults, result)
	}
	return filteredResults
}

// overwriteResultRows deletes existing and inserts the new results for a query and host.
//
// The "snapshot" array in a ScheduledQueryResult can contain multiple rows.
// Each row is saved as a separate ScheduledQueryResultRow, i.e. a result could contain
// many USB Devices or a result could contain all user accounts on a host.
func (svc *Service) overwriteResultRows(ctx context.Context, result *fleet.ScheduledQueryResult, queryID, hostID uint, maxQueryReportRows int) (int, error) {
	fetchTime := time.Now()

	rows := make([]*fleet.ScheduledQueryResultRow, 0, len(result.Snapshot))

	// If the snapshot is empty, we still want to save a row with a null value
	// to capture LastFetched.
	if len(result.Snapshot) == 0 {
		rows = append(rows, &fleet.ScheduledQueryResultRow{
			QueryID:     queryID,
			HostID:      hostID,
			Data:        nil,
			LastFetched: fetchTime,
		})
	}

	for _, snapshotItem := range result.Snapshot {
		row := &fleet.ScheduledQueryResultRow{
			QueryID:     queryID,
			HostID:      hostID,
			Data:        snapshotItem,
			LastFetched: fetchTime,
		}
		rows = append(rows, row)
	}

	var rowsAdded int
	var err error
	if rowsAdded, err = svc.ds.OverwriteQueryResultRows(ctx, rows, maxQueryReportRows); err != nil {
		return rowsAdded, ctxerr.Wrap(ctx, err, "overwriting query result rows")
	}
	// If we only inserted an error row, don't count it against the limit.
	if len(result.Snapshot) == 0 {
		rowsAdded--
	}
	return rowsAdded, nil
}

// getMostRecentResults returns only the most recent result per query.
// Osquery can send multiple results for the same query (ie. if an agent loses
// network connectivity it will cache multiple results).  Query Reports only
// save the most recent result for a given query.
func getMostRecentResults(results []*fleet.ScheduledQueryResult) []*fleet.ScheduledQueryResult {
	// Use a map to track the most recent entry for each unique QueryName
	latestResults := make(map[string]*fleet.ScheduledQueryResult)

	for _, result := range results {
		if result == nil {
			// This is a result that failed to unmarshal.
			continue
		}
		if existing, ok := latestResults[result.QueryName]; ok {
			// Compare the UnixTime time and update the map if the current result is more recent
			if result.UnixTime > existing.UnixTime {
				latestResults[result.QueryName] = result
			}
		} else {
			latestResults[result.QueryName] = result
		}
	}

	// Convert the map back to a slice
	var filteredResults []*fleet.ScheduledQueryResult
	for _, v := range latestResults {
		filteredResults = append(filteredResults, v)
	}

	return filteredResults
}

// findPackDelimiterString attempts to find the `pack_delimiter` string in the scheduled
// query name reported by osquery (note that `pack_delimiter` can contain multiple characters).
//
// The expected format for s is "pack<pack_delimiter>{Global|team-<team_id>}<pack_delimiter><query_name>"
//
// Returns "" if it failed to parse the pack_delimiter.

var (
	dcounter = regexp.MustCompile(`(Global)|(team-\d+)`)
	pattern  = regexp.MustCompile(`^(.*)(?:(Global)|(team-\d+))`)
)

func findPackDelimiterString(scheduledQueryName string) string {
	scheduledQueryName = scheduledQueryName[4:] // always starts with "pack"

	count := dcounter.FindAllString(scheduledQueryName, -1)

	// If Global or team-<team_id> does not appear, then the
	// pack_delimiter is invalid.
	if len(count) == 0 {
		return ""
	}

	if len(count) == 1 {
		matches := pattern.FindStringSubmatch(scheduledQueryName)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	// Handle edge cases where "Global" or "team-<team_id>"" appears multiple times in the query
	// name. Regex is not pre-compiled, so it is a less performant operation.
	// Go's regexp doesn't support backreferences so we have to perform some manual work.
	if len(count) > 1 {
		for l := 1; l < len(scheduledQueryName); l++ {
			sep := scheduledQueryName[:l]
			rest := scheduledQueryName[l:]
			pattern := fmt.Sprintf(`^(?:(Global)|(team-\d+))%s.+`, regexp.QuoteMeta(sep))
			matched, _ := regexp.MatchString(pattern, rest)
			if matched {
				return sep
			}
		}
	}

	return ""
}

// getQueryNameAndTeamIDFromResult attempts to parse the scheduled query name reported by osquery.
//
// The expected format of query names managed by Fleet is:
// "pack<pack_delimiter>{Global|team-<team_id>}<pack_delimiter><query_name>"
func getQueryNameAndTeamIDFromResult(path string) (*uint, string, error) {
	if !strings.HasPrefix(path, "pack") || len(path) <= 4 {
		return nil, "", fmt.Errorf("unknown format: %q", path)
	}

	sep := findPackDelimiterString(path)
	if sep == "" {
		// If a pack_delimiter could not be parsed we return an error.
		//
		// 2017/legacy packs with the format "pack/<Pack name>/<Query name> are
		// considered unknown format (they are not considered global or team
		// scheduled queries).

		// We can't infer the team from this and it can't be stored, but it's still valid
		if strings.HasPrefix(path, "pack/") && strings.Count(path, "/") == 2 {
			return nil, "", fleet.ErrLegacyQueryPack
		}

		// Truly unknown
		return nil, "", fmt.Errorf("unknown format: %q", path)
	}

	// For pattern: pack/Global/Name
	globalPattern := "pack" + sep + "Global" + sep
	if strings.HasPrefix(path, globalPattern) {
		name := strings.TrimPrefix(path, globalPattern)
		if name == "" {
			return nil, "", fmt.Errorf("parsing query name: %s", path)
		}
		return nil, strings.TrimPrefix(path, globalPattern), nil
	}

	// For pattern: pack/team-<ID>/Name
	teamPattern := "pack" + sep + "team-"
	if strings.HasPrefix(path, teamPattern) {
		teamIDAndRest := strings.TrimPrefix(path, teamPattern)
		teamIDAndQueryNameParts := strings.SplitN(teamIDAndRest, sep, 2)
		if len(teamIDAndQueryNameParts) != 2 {
			return nil, "", fmt.Errorf("parsing team number part: %s", path)
		}
		if teamIDAndQueryNameParts[1] == "" {
			return nil, "", fmt.Errorf("parsing query name: %s", path)
		}
		teamNumberUint, err := strconv.ParseUint(teamIDAndQueryNameParts[0], 10, 32)
		if err != nil {
			return nil, "", fmt.Errorf("parsing team number: %w", err)
		}
		teamNumber := uint(teamNumberUint)
		return &teamNumber, teamIDAndQueryNameParts[1], nil
	}

	// If none of the above patterns match, return error
	return nil, "", fmt.Errorf("unknown format: %q", path)
}

// Yara rules

func (svc *Service) YaraRuleByName(ctx context.Context, name string) (*fleet.YaraRule, error) {
	return svc.ds.YaraRuleByName(ctx, name)
}

type getYaraRequest struct {
	NodeKey string `json:"node_key"`
	Name    string `url:"name"`
}

func (r *getYaraRequest) hostNodeKey() string {
	return r.NodeKey
}

type getYaraResponse struct {
	Err     error `json:"error,omitempty"`
	Content string
}

func (r getYaraResponse) Error() error { return r.Err }

func (r getYaraResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(r.Content))
}

func getYaraEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	r := request.(*getYaraRequest)
	rule, err := svc.YaraRuleByName(ctx, r.Name)
	if err != nil {
		return getYaraResponse{Err: err}, nil
	}
	return getYaraResponse{Content: rule.Contents}, nil
}
