package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/cached_mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/etag_invalidate"
	"github.com/fleetdm/fleet/v4/server/datastore/mysqlredis"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/redis_config_etag"
)

// buildRedisPoolConfig translates the Fleet Redis config into the redis
// package's PoolConfig. The address has its "redis://" scheme stripped so
// providers that publish a full URI (e.g. Render's managed Redis) work
// without a separate config knob.
func buildRedisPoolConfig(cfg config.RedisConfig) redis.PoolConfig {
	return redis.PoolConfig{
		// Strip the Redis URI scheme if it's present. Scheme docs are at:
		// https://www.iana.org/assignments/uri-schemes/uri-schemes.xhtml
		// In the future, we could support the full Redis URI if needed
		// (including username, password, database, etc.)
		Server:                    strings.TrimPrefix(cfg.Address, "redis://"),
		Username:                  cfg.Username,
		Password:                  cfg.Password,
		Database:                  cfg.Database,
		UseTLS:                    cfg.UseTLS,
		Region:                    cfg.Region,
		CacheName:                 cfg.CacheName,
		StsAssumeRoleArn:          cfg.StsAssumeRoleArn,
		StsExternalID:             cfg.StsExternalID,
		ConnTimeout:               cfg.ConnectTimeout,
		KeepAlive:                 cfg.KeepAlive,
		ConnectRetryAttempts:      cfg.ConnectRetryAttempts,
		ClusterFollowRedirections: cfg.ClusterFollowRedirections,
		ClusterReadFromReplica:    cfg.ClusterReadFromReplica,
		TLSCert:                   cfg.TLSCert,
		TLSKey:                    cfg.TLSKey,
		TLSCA:                     cfg.TLSCA,
		TLSServerName:             cfg.TLSServerName,
		TLSHandshakeTimeout:       cfg.TLSHandshakeTimeout,
		MaxIdleConns:              cfg.MaxIdleConns,
		MaxOpenConns:              cfg.MaxOpenConns,
		ConnMaxLifetime:           cfg.ConnMaxLifetime,
		IdleTimeout:               cfg.IdleTimeout,
		ConnWaitTimeout:           cfg.ConnWaitTimeout,
		WriteTimeout:              cfg.WriteTimeout,
		ReadTimeout:               cfg.ReadTimeout,
	}
}

// validateRedisConfig returns a non-nil error when the Redis host-cache
// configuration is inconsistent (enabled without a positive TTL). It
// encodes the boot/refuse-to-boot rule for the cache configuration so the
// decision can be unit-tested without spinning up Redis.
func validateRedisConfig(cfg config.RedisConfig) error {
	if cfg.HostCacheEnabled && cfg.HostCacheTTL <= 0 {
		return fmt.Errorf("redis.host_cache_ttl must be > 0 when redis.host_cache_enabled is true (got %s)", cfg.HostCacheTTL)
	}
	return nil
}

// effectiveRedisConfigETags resolves osquery.redis_config_etags against its
// prerequisite: the Redis short circuit cannot be enabled while conditional
// config requests (osquery.config_etags) are disabled — with the protocol
// off, agents' etags are ignored, so the short circuit could never match and
// its Redis traffic would be pure waste. serve.go warns when the flag is set
// but gated off.
func effectiveRedisConfigETags(cfg config.FleetConfig) bool {
	return cfg.Osquery.ConfigETags && cfg.Osquery.RedisConfigETags
}

// initRedis brings up the Redis pool and the datastore wrappers that depend
// on it: cached_mysql (in-memory caching layer over the datastore),
// mysqlredis (Redis-backed host lookup and license-enforced host limit), and
// etag_invalidate (osquery config ETag invalidation hooks). Failures go
// through initFatal. Returns nil values on the failure path so the function
// is safe when initFatal does not terminate (e.g., tests using a recorder).
//
// The returned fleet.Datastore is the fully wrapped chain (etag_invalidate →
// mysqlredis → cached_mysql → input ds when the config ETag feature is
// effectively enabled; without the etag_invalidate wrapper otherwise); the
// returned *mysqlredis.Datastore is the inner wrapper, which a few callers
// need by concrete type. The returned ConfigETagStore is the Redis-backed
// osquery config ETag store, nil when the feature is effectively disabled.
//
// ██ NO ETAG REDIS I/O WHEN DISABLED ██ The store and the etag_invalidate
// write hooks are wired ONLY when effectiveRedisConfigETags is true — with
// osquery.redis_config_etags off (directly, or gated off by
// osquery.config_etags=false) no config ETag Redis code runs at all: no
// reads, no writes, no invalidation traffic. Coherence across flag flips is
// instead guaranteed by a startup generation bump: every boot that enables
// the feature invalidates all stored records first, so nothing written under
// an earlier configuration (including a window with the hooks off) can ever
// validate. The bump also arms the write fence, so the cache warms a few
// minutes after (each instance of) an enabling boot — a bounded cold start
// during rolling deploys, traded for zero Redis traffic while disabled.
func initRedis(
	ctx context.Context,
	cfg config.FleetConfig,
	license *fleet.LicenseInfo,
	ds fleet.Datastore,
	logger *slog.Logger,
	initFatal func(err error, msg string),
) (fleet.RedisPool, fleet.Datastore, *mysqlredis.Datastore, fleet.ConfigETagStore) {
	if license == nil {
		initFatal(errors.New("license was nil"), "initialize Redis")
		return nil, nil, nil, nil
	}

	// Validate cheap local config before dialing Redis: surfaces a
	// host-cache config error as itself, not as a connectivity failure,
	// and avoids opening a pool that would be discarded if initFatal is
	// swapped (e.g., a test recorder) and execution continues.
	if err := validateRedisConfig(cfg.Redis); err != nil {
		initFatal(err, "validate host cache configuration")
		return nil, nil, nil, nil
	}

	redisPool, err := redis.NewPool(buildRedisPoolConfig(cfg.Redis))
	if err != nil {
		initFatal(err, "initialize Redis")
		return nil, nil, nil, nil
	}
	logger.InfoContext(ctx, "redis initialized", "component", "redis", "mode", redisPool.Mode())

	wrappedDS := cached_mysql.New(ds)

	var dsOpts []mysqlredis.Option
	if license.DeviceCount > 0 && cfg.License.EnforceHostLimit {
		dsOpts = append(dsOpts, mysqlredis.WithEnforcedHostLimit(license.DeviceCount))
	}
	if cfg.Redis.HostCacheEnabled {
		dsOpts = append(dsOpts, mysqlredis.WithHostCache(cfg.Redis.HostCacheTTL))
		logger.InfoContext(ctx, "host lookup redis cache enabled",
			"component", "mysqlredis", "ttl", cfg.Redis.HostCacheTTL)
	}

	redisWrapperDS := mysqlredis.New(wrappedDS, redisPool, dsOpts...)

	// Config ETag store + invalidation hooks, wired ONLY when the short
	// circuit is effectively enabled (see the NO ETAG REDIS I/O notice
	// above). The etag_invalidate wrapper is OUTERMOST so it sees every
	// config-affecting write regardless of the inner caching layers.
	if !effectiveRedisConfigETags(cfg) {
		return redisPool, redisWrapperDS, redisWrapperDS, nil
	}

	configETagStore := redis_config_etag.New(redisPool, logger.With("component", "config-etag"))

	// Startup generation bump: invalidate every stored record before this
	// instance starts serving, so nothing written under an earlier
	// configuration — including a window when the feature (and therefore the
	// invalidation hooks) was disabled — can ever validate. If the bump
	// fails, the short circuit stays OFF for this boot (fail open costs
	// performance, never correctness): stale records cannot be read because
	// the store is never injected, and the next enabling boot retries.
	if err := configETagStore.Invalidate(ctx); err != nil {
		logger.ErrorContext(ctx, "config etag: startup generation bump failed; short circuit disabled for this boot",
			"component", "config-etag", "err", err)
		return redisPool, redisWrapperDS, redisWrapperDS, nil
	}

	etagDS := etag_invalidate.New(redisWrapperDS, configETagStore, logger.With("component", "etag-invalidate"))

	return redisPool, etagDS, redisWrapperDS, configETagStore
}
