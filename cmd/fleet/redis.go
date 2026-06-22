package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/cached_mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysqlredis"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
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

// initRedis brings up the Redis pool and the two datastore wrappers that
// depend on it: cached_mysql (in-memory caching layer over the datastore)
// and mysqlredis (Redis-backed host lookup and license-enforced host
// limit). Failures go through initFatal. Returns nil values on the
// failure path so the function is safe when initFatal does not terminate
// (e.g., tests using a recorder).
//
// The returned fleet.Datastore is the fully wrapped chain (mysqlredis →
// cached_mysql → input ds); the returned *mysqlredis.Datastore is the
// outermost wrapper, which a few callers need by concrete type.
func initRedis(
	ctx context.Context,
	cfg config.FleetConfig,
	license *fleet.LicenseInfo,
	ds fleet.Datastore,
	logger *slog.Logger,
	initFatal func(err error, msg string),
) (fleet.RedisPool, fleet.Datastore, *mysqlredis.Datastore) {
	if license == nil {
		initFatal(errors.New("license was nil"), "initialize Redis")
		return nil, nil, nil
	}

	// Validate cheap local config before dialing Redis: surfaces a
	// host-cache config error as itself, not as a connectivity failure,
	// and avoids opening a pool that would be discarded if initFatal is
	// swapped (e.g., a test recorder) and execution continues.
	if err := validateRedisConfig(cfg.Redis); err != nil {
		initFatal(err, "validate host cache configuration")
		return nil, nil, nil
	}

	redisPool, err := redis.NewPool(buildRedisPoolConfig(cfg.Redis))
	if err != nil {
		initFatal(err, "initialize Redis")
		return nil, nil, nil
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
	return redisPool, redisWrapperDS, redisWrapperDS
}
