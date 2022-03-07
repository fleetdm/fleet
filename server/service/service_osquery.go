package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/osquery_utils"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
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

	host, err := svc.ds.LoadHostByNodeKey(ctx, nodeKey)
	switch {
	case err == nil:
		// OK
	case fleet.IsNotFound(err):
		return nil, false, osqueryError{
			message:     "authentication error: invalid node key: " + nodeKey,
			nodeInvalid: true,
		}
	default:
		return nil, false, osqueryError{
			message: "authentication error: " + err.Error(),
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

	return host, svc.debugEnabledForHost(ctx, host.ID), nil
}

func (svc Service) debugEnabledForHost(ctx context.Context, id uint) bool {
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
		return "", osqueryError{message: "app config load failed: " + err.Error(), nodeInvalid: true}
	}

	// Save enrollment details if provided
	detailQueries := osquery_utils.GetDetailQueries(appConfig, svc.config)
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
			go svc.serialUpdateHost(host)
		} else {
			if err := svc.ds.UpdateHost(ctx, host); err != nil {
				return "", ctxerr.Wrap(ctx, err, "save host in enroll agent")
			}
		}
	}

	return nodeKey, nil
}

func (svc Service) serialUpdateHost(host *fleet.Host) {
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

// jitterHashTable implements a data structure that allows a fleet to generate a static jitter value
// that is properly balanced. Balance in this context means that hosts would be distributed uniformly
// across the total jitter time so there are no spikes.
// The way this structure works is as follows:
// Given an amount of buckets, we want to place hosts in buckets evenly. So we don't want bucket 0 to
// have 1000 hosts, and all the other buckets 0. If there were 1000 buckets, and 1000 hosts, we should
// end up with 1 per bucket.
// The total amount of online hosts is unknown, so first it assumes that amount of buckets >= amount
// of total hosts (maxCapacity of 1 per bucket). Once we have more hosts than buckets, then we
// increase the maxCapacity by 1 for all buckets, and start placing hosts.
// Hosts that have been placed in a bucket remain in that bucket for as long as the fleet instance is
// running.
// The preferred bucket for a host is the one at (host id % bucketCount). If that bucket is full, the
// next one will be tried. If all buckets are full, then capacity gets increased and the bucket
// selection process restarts.
// Once a bucket is found, the index for the bucket (going from 0 to bucketCount) will be the amount of
// minutes added to the host check in time.
// For example: at a 1hr interval, and the default 10% max jitter percent. That allows hosts to
// distribute within 6 minutes around the hour mark. We would have 6 buckets in that case.
// In the worst possible case that all hosts start at the same time, max jitter percent can be set to
// 100, and this method will distribute hosts evenly.
// The main caveat of this approach is that it works at the fleet instance. So depending on what
// instance gets chosen by the load balancer, the jitter might be different. However, load tests have
// shown that the distribution in practice is pretty balance even when all hosts try to check in at
// the same time.
type jitterHashTable struct {
	mu          sync.Mutex
	maxCapacity int
	bucketCount int
	buckets     map[int]int
	cache       map[uint]time.Duration
}

func newJitterHashTable(bucketCount int) *jitterHashTable {
	if bucketCount == 0 {
		bucketCount = 1
	}
	return &jitterHashTable{
		maxCapacity: 1,
		bucketCount: bucketCount,
		buckets:     make(map[int]int),
		cache:       make(map[uint]time.Duration),
	}
}

func (jh *jitterHashTable) jitterForHost(hostID uint) time.Duration {
	// if no jitter is configured just return 0
	if jh.bucketCount <= 1 {
		return 0
	}

	jh.mu.Lock()
	if jitter, ok := jh.cache[hostID]; ok {
		jh.mu.Unlock()
		return jitter
	}

	for i := 0; i < jh.bucketCount; i++ {
		possibleBucket := (int(hostID) + i) % jh.bucketCount

		// if the next bucket has capacity, great!
		if jh.buckets[possibleBucket] < jh.maxCapacity {
			jh.buckets[possibleBucket]++
			jitter := time.Duration(possibleBucket) * time.Minute
			jh.cache[hostID] = jitter

			jh.mu.Unlock()
			return jitter
		}
	}

	// otherwise, bump the capacity and restart the process
	jh.maxCapacity++

	jh.mu.Unlock()
	return jh.jitterForHost(hostID)
}
