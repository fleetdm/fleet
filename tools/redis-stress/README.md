# redis-stress

Cluster-aware Redis stress tool with two modes.

Both modes use Fleet's own `redis.NewPool` (`server/datastore/redis`), so
cluster topology, redirection handling, and connection routing match what the
real Fleet server does in production.

## Modes

### `write` — steady SET-only load

Fill a Redis instance (standalone or cluster) at a configurable rate. Each
worker writes keys on its own ticker; useful for "occupy the cluster with
ongoing writes while I observe something else" or seeding a dataset.

This mode is the cluster-aware successor to the original `tools/redis-stress`
tool. The old subcommand-less invocation (`redis-stress -addr=X -wait=10m`)
still works — when the first arg starts with `-`, the dispatcher routes to
`write`. The legacy flags `-wait`, `-debug`, and `-index-start` are kept for
backward compatibility.

```sh
go run ./tools/redis-stress write \
  -addr 127.0.0.1:7001 \
  -workers 5 \
  -rate 100 \
  -duration 1m
```

| Flag | Default | Purpose |
|---|---|---|
| `-addr` | `127.0.0.1:6379` | Redis address (cluster startup node OK; cluster auto-detected) |
| `-workers` | `1` | Concurrent SET workers |
| `-rate` | `1` | SETs per worker per second (fractional OK) |
| `-duration` | `10m` | Total run time |
| `-key-prefix` | `stress_write_` | Key prefix |
| `-key-ttl` | `10m` | Per-key expiration |
| `-index-start` | `0` | Starting value of each worker's per-key counter (legacy) |
| `-debug` | `false` | Log every SET (legacy) |
| `-wait` | — | Alias for `-duration` (legacy) |
| `-cluster-follow-redirects` | `true` | `ClusterFollowRedirections` (cluster only) |
| `-cluster-read-from-replica` | `false` | `ClusterReadFromReplica` (cluster only) |

### `race` — SET-then-GET race detection

Each worker repeatedly does, on fresh pool connections:

```
conn1 := pool.Get(); conn1.Do("SET", k, v, "PX", ttl); conn1.Close()
conn2 := pool.Get(); conn2.Do("GET", k);                conn2.Close()
```

and counts any `GET` that returns `nil` immediately after a successful `SET`
on the same key. This mirrors how Fleet's
`server/service/redis_key_value.RedisKeyValue` does its `Set` / `Get` —
fresh connection per call.

```sh
# default — cluster mode, reads through ConfigureDoer (i.e., not explicitly
# routed to replicas; same path Fleet uses today)
go run ./tools/redis-stress race -addr 127.0.0.1:7001 -workers 50 -iterations 2000

# explicit replica reads — useful for testing what a deployment with an
# external read-from-replica router (proxy, ElastiCache reader endpoint,
# Redis-Enterprise R/W split) would expose
go run ./tools/redis-stress race -addr 127.0.0.1:7001 \
  -cluster-read-from-replica \
  -explicit-readonly
```

| Flag | Default | Purpose |
|---|---|---|
| `-addr` | `127.0.0.1:7001` | Redis cluster startup node |
| `-workers` | `50` | Concurrent SET-then-GET workers |
| `-iterations` | `1000` | Iterations per worker |
| `-ttl` | `4m` | PX expiration on SET |
| `-key-prefix` | `stress_race_` | Key prefix |
| `-explicit-readonly` | `false` | Wrap the GET conn with `redis.ReadOnlyConn` so it's routed to a replica when the pool has `ClusterReadFromReplica=true`. Without this flag, both SET and GET go to primary. |
| `-cluster-follow-redirects` | `true` | `ClusterFollowRedirections` |
| `-cluster-read-from-replica` | `true` | `ClusterReadFromReplica` |

#### Output

```
================ summary ================
elapsed:           11.2s
sets:              100000 (errors 0)
gets:              100000 (errors 0)
nil-after-set:     0  ← the bug
stale-after-set:   0
ops/sec:           17858.2
```

`nil-after-set` is the metric to watch. Any non-zero value means the cluster
served a `GET` for a key the same code had just `SET` and gotten an OK
acknowledgement for. The tool exits with status 1 when this happens.

`stale-after-set` should be zero in normal operation (the key namespace is
worker-and-iteration-specific) and is included as a defense against
unexpected key collisions.

## Bringing up a local Redis cluster

The repo includes a 6-node Redis Cluster compose file. From the repo root:

```sh
docker compose -f docker-compose.yml -f docker-compose-redis-cluster.yml up -d \
  redis-cluster-1 redis-cluster-2 redis-cluster-3 \
  redis-cluster-4 redis-cluster-5 redis-cluster-6 \
  redis-cluster-setup
```

Verify the cluster came up:

```sh
docker exec fleet-redis-cluster-1-1 redis-cli -p 7001 cluster info | grep cluster_state
# cluster_state:ok
```

On macOS, host-to-cluster networking requires
[`docker-mac-net-connect`](https://github.com/chipmk/docker-mac-net-connect):

```sh
brew install chipmk/tap/docker-mac-net-connect
sudo brew services start chipmk/tap/docker-mac-net-connect
```

Without it, only port-forwarded nodes are reachable and any cluster redirect
times out.

## Forcing a replica to lag

To verify the race detector works mechanically (or to simulate a customer
deployment where reads can fall behind writes), pause one of the replica
nodes mid-test:

```sh
docker pause fleet-redis-cluster-4-1
go run ./tools/redis-stress race \
  -addr 127.0.0.1:7001 -explicit-readonly -workers 20 -iterations 500
docker unpause fleet-redis-cluster-4-1
```

With `-explicit-readonly` *and* a paused replica, you should see non-zero
`nil-after-set` events for keys whose slot's replica was the paused one.
