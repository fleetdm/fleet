# Configuration

- [Configuring the Fleet binary](#configuring-the-fleet-binary)
  - [High-level configuration overview](#high-level-configuration-overview)
  - [Commands](#commands)
  - [Options](#options)
- [Managing osquery configurations](#managing-osquery-configurations)
- [Running with systemd](#running-with-systemd)
- [Configuring single sign on](#configuring-single-sign-on)
  - [Identity Provider (IDP) configuration](#identity-provider-IDP-configuration)
  - [Fleet SSO configuration](#fleet-sso-configuration)
  - [Creating SSO users in Fleet](#creating-sso-users-in-fleet)
- [Feature flags](#feature-flags)

## Configuring the Fleet binary

For information on how to run the `fleet` binary, detailed usage information can be found by running `fleet --help`. This document is a more detailed version of the information presented in the help output text. If you prefer to use a CLI instead of a web browser, we hope that you like the binary interface to the Fleet application!

### High-level configuration overview

To get the most out of running the Fleet server, it is helpful to establish a mutual understanding of what the desired architecture looks like and what it's trying to accomplish.

Your Fleet server's two main purposes are:

- To serve as your [osquery TLS server](https://osquery.readthedocs.io/en/stable/deployment/remote/)
- To serve the Fleet web UI, which allows you to manage osquery configuration, query hosts, etc.

The Fleet server allows you persist configuration, manage users, etc. Thus, it needs a database. Fleet uses MySQL and requires you to supply configurations to connect to a MySQL server. It is also possible to configure connection to a MySQL replica in addition to the primary, to be used for reading only. Fleet also uses Redis to perform some more high-speed data access action throughout the lifecycle of the application (for example, distributed query result ingestion). Thus, Fleet also requires that you supply Redis connection configurations.

Fleet can scale to hundreds of thousands of devices with a single Redis instance, and is also compatible with Redis Cluster. Fleet does not support Redis Sentinel.

Since Fleet is a web application, when you run Fleet there are some other configurations that must be defined, such as:

- The TLS certificates that Fleet should use to terminate TLS.

When deploying Fleet, mitigate DoS attacks as you would when deploying any app.

Since Fleet is an osquery TLS server, you are also able to define configurations that can customize your experience there, such as:

- The destination of the osquery status and result logs on the local filesystem
- Various details about the refresh/check-in intervals for your hosts

### Commands

The `fleet` binary contains several "commands". Similarly to how `git` has many commands (`git status`, `git commit`, etc), the `fleet` binary accepts the following commands:

- `fleet prepare db`
- `fleet serve`
- `fleet version`
- `fleet config_dump`

### Options

#### How do you specify options?

In order of precedence, options can be specified via:

- A configuration file (in YAML format)
- Environment variables
- Command-line flags

For example, all of the following ways of launching Fleet are equivalent:

##### Using only CLI flags

```
/usr/bin/fleet serve \
--mysql_address=127.0.0.1:3306 \
--mysql_database=fleet \
--mysql_username=root \
--mysql_password=toor \
--redis_address=127.0.0.1:6379 \
--server_cert=/tmp/server.cert \
--server_key=/tmp/server.key \
--logging_json
```

##### Using only environment variables

```
FLEET_MYSQL_ADDRESS=127.0.0.1:3306 \
FLEET_MYSQL_DATABASE=fleet \
FLEET_MYSQL_USERNAME=root \
FLEET_MYSQL_PASSWORD=toor \
FLEET_REDIS_ADDRESS=127.0.0.1:6379 \
FLEET_SERVER_CERT=/tmp/server.cert \
FLEET_SERVER_KEY=/tmp/server.key \
FLEET_LOGGING_JSON=true \
/usr/bin/fleet serve
```

##### Using a config file

```
echo '
mysql:
  address: 127.0.0.1:3306
  database: fleet
  username: root
  password: toor
redis:
  address: 127.0.0.1:6379
server:
  cert: /tmp/server.cert
  key: /tmp/server.key
logging:
  json: true
' > /tmp/fleet.yml
fleet serve --config /tmp/fleet.yml
```

### What are the options?

Note that all option names can be converted consistently from flag name to environment variable and visa-versa. For example, the `--mysql_address` flag would be the `FLEET_MYSQL_ADDRESS`. Further, specifying the `mysql_address` option in the config would follow the pattern:

```
mysql:
  address: 127.0.0.1:3306
```

And `mysql_read_replica_address` would be:

```
mysql_read_replica:
  address: 127.0.0.1:3307
```

Basically, just capitalize the option and prepend `FLEET_` to it in order to get the environment variable. The conversion works the same the opposite way.

All duration-based settings accept valid time units of `s`, `m`, `h`.

#### MySQL

This section describes the configuration options for the primary - if you also want to setup a read replica, the options are the same, except that the yaml section is `mysql_read_replica`, and the flags have the `mysql_read_replica_` prefix instead of `mysql_` (the corresponding environment variables follow the same transformation). Note that there is no default value for `mysql_read_replica_address`, it must be set explicitly for fleet to use a read replica, and it is recommended in that case to set a non-zero value for `mysql_read_replica_conn_max_lifetime` as in some environments, the replica's address may dynamically change to point
from the primary to an actual distinct replica based on auto-scaling options, so existing idle connections need to be recycled
periodically.

##### mysql_address

The address of the MySQL server which Fleet should connect to. Include the hostname and port.

- Default value: `localhost:3306`
- Environment variable: `FLEET_MYSQL_ADDRESS`
- Config file format:

  ```
  mysql:
  	address: localhost:3306
  ```

##### mysql_database

The name of the MySQL database which Fleet will use.

- Default value: `fleet`
- Environment variable: `FLEET_MYSQL_DATABASE`
- Config file format:

  ```
  mysql:
  	database: fleet
  ```

##### mysql_username

The username to use when connecting to the MySQL instance.

- Default value: `fleet`
- Environment variable: `FLEET_MYSQL_USERNAME`
- Config file format:

  ```
  mysql:
  	username: fleet
  ```

##### mysql_password

The password to use when connecting to the MySQL instance.

- Default value: `fleet`
- Environment variable: `FLEET_MYSQL_PASSWORD`
- Config file format:

  ```
  mysql:
  	password: fleet
  ```

##### mysql_password_path

File path to a file that contains the password to use when connecting to the MySQL instance.

- Default value: `""`
- Environment variable: `FLEET_MYSQL_PASSWORD_PATH`
- Config file format:

  ```
  mysql:
  	password_path: '/run/secrets/fleetdm-mysql-password'
  ```

##### mysql_tls_ca

The path to a PEM encoded certificate of MYSQL's CA for client certificate authentication.

- Default value: none
- Environment variable: `FLEET_MYSQL_TLS_CA`
- Config file format:

  ```
  mysql:
  	tls_ca: /path/to/server-ca.pem
  ```

##### mysql_tls_cert

The path to a PEM encoded certificate use for tls authentication.

- Default value: none
- Environment variable: `FLEET_MYSQL_TLS_CERT`
- Config file format:

  ```
  mysql:
  	tls_cert: /path/to/certificate.pem
  ```

##### mysql_tls_key

The path to a PEM encoded private key use for tls authentication.

- Default value: none
- Environment variable: `FLEET_MYSQL_TLS_KEY`
- Config file format:

  ```
  mysql:
  	tls_key: /path/to/key.pem
  ```

##### mysql_tls_config

The tls value in a MYSQL DSN. Can be `true`,`false`,`skip-verify` or the CN value of the certificate.

- Default value: none
- Environment variable: `FLEET_MYSQL_TLS_CONFIG`
- Config file format:

  ```
  mysql:
  	tls_config: true
  ```

##### mysql_tls_server_name

The server name or IP address used by the client certificate.

- Default value: none
- Environment variable: `FLEET_MYSQL_TLS_SERVER_NAME`
- Config file format:

  ```
  mysql:
  	server_name: 127.0.0.1
  ```

##### mysql_max_open_conns

Maximum open connections to database

- Default value: 50
- Environment variable: `FLEET_MYSQL_MAX_OPEN_CONNS`
- Config file format:

  ```
  mysql:
  	max_open_conns: 50
  ```

##### mysql_max_idle_conns

Maximum idle connections to database. This value should be equal to or less than `mysql_max_open_conns`

- Default value: 50
- Environment variable: `FLEET_MYSQL_MAX_IDLE_CONNS`
- Config file format:

  ```
  mysql:
  	max_idle_conns: 50
  ```

##### mysql_conn_max_lifetime

Maximum amount of time, in seconds, a connection may be reused.

- Default value: 0 (Unlimited)
- Environment variable: `FLEET_MYSQL_CONN_MAX_LIFETIME`
- Config file format:

  ```
  mysql:
  	conn_max_lifetime: 50
  ```

#### Redis

Note that a TLS connection to a Redis instance can be tested by running the
`tlsconnect` Go program in `tools/redis-tests`, e.g. from the root of the repository:

```
$ go run ./tools/redis-tests/tlsconnect.go -addr <redis_address> -cacert <redis_tls_ca> -cert <redis_tls_cert> -key <redis_tls_key>
# run `go run ./tools/redis-tests/tlsconnect.go -h` for the full list of supported flags
```

By default, this will setup a Redis pool for that configuration and execute a
`PING` command with a TLS connection, printing any error it encounters.

##### redis_address

The address of the Redis server which Fleet should connect to. Include the hostname and port.

- Default value: `localhost:6379`
- Environment variable: `FLEET_REDIS_ADDRESS`
- Config file format:

  ```
  redis:
  	address: 127.0.0.1:7369
  ```

##### redis_password

The password to use when connecting to the Redis instance.

- Default value: `<empty>`
- Environment variable: `FLEET_REDIS_PASSWORD`
- Config file format:

  ```
  redis:
  	password: foobar
  ```

##### redis_database

The database to use when connecting to the Redis instance.

- Default value: `0`
- Environment variable: `FLEET_REDIS_DATABASE`
- Config file format:

  ```
  redis:
    database: 14
  ```

##### redis_use_tls

Use a TLS connection to the Redis server.

- Default value: `false`
- Environment variable: `FLEET_REDIS_USE_TLS`
- Config file format:

  ```
  redis:
    use_tls: true
  ```

##### redis_duplicate_results

Whether or not to duplicate Live Query results to another Redis channel named `LQDuplicate`. This is useful in a scenario that would involve shipping the Live Query results outside of Fleet, near-realtime.

- Default value: `false`
- Environment variable: `FLEET_REDIS_DUPLICATE_RESULTS`
- Config file format:

  ```
  redis:
    duplicate_results: true
  ```

##### redis_connect_timeout

Timeout for redis connection.

- Default value: 5s
- Environment variable: `FLEET_REDIS_CONNECT_TIMEOUT`
- Config file format:

  ```
  redis:
    connect_timeout: 10s
  ```

##### redis_keep_alive

Interval between keep alive probes.

- Default value: 10s
- Environment variable: `FLEET_REDIS_KEEP_ALIVE`
- Config file format:

  ```
  redis:
    keep_alive: 30s
  ```

##### redis_connect_retry_attempts

Maximum number of attempts to retry a failed connection to a redis node. Only
certain type of errors are retried, such as connection timeouts.

- Default value: 0 (no retry)
- Environment variable: `FLEET_REDIS_CONNECT_RETRY_ATTEMPTS`
- Config file format:

  ```
  redis:
    connect_retry_attempts: 2
  ```

##### redis_cluster_follow_redirections

Whether or not to automatically follow redirection errors received from the
Redis server. Applies only to Redis Cluster setups, ignored in standalone
Redis. In Redis Cluster, keys can be moved around to different nodes when the
cluster is unstable and reorganizing the data. With this configuration option
set to true, those (typically short and transient) redirection errors can be
handled transparently instead of ending in an error.

- Default value: false
- Environment variable: `FLEET_REDIS_CLUSTER_FOLLOW_REDIRECTIONS`
- Config file format:

  ```
  redis:
    cluster_follow_redirections: true
  ```

##### redis_cluster_read_from_replica

Whether or not to prefer reading from a replica when possible. Applies only
to Redis Cluster setups, ignored in standalone Redis.

- Default value: false
- Environment variable: `FLEET_REDIS_CLUSTER_READ_FROM_REPLICA`
- Config file format:

  ```
  redis:
    cluster_read_from_replica: true
  ```

##### redis_tls_cert

The path to a PEM-encoded certificate used for tls authentication.

- Default value: none
- Environment variable: `FLEET_REDIS_TLS_CERT`
- Config file format:

  ```
  redis:
  	tls_cert: /path/to/certificate.pem
  ```

##### redis_tls_key

The path to a PEM-encoded private key used for tls authentication.

- Default value: none
- Environment variable: `FLEET_REDIS_TLS_KEY`
- Config file format:

  ```
  redis:
  	tls_key: /path/to/key.pem
  ```

##### redis_tls_ca

The path to a PEM-encoded certificate of Redis' CA for client certificate authentication.

- Default value: none
- Environment variable: `FLEET_REDIS_TLS_CA`
- Config file format:

  ```
  redis:
  	tls_ca: /path/to/server-ca.pem
  ```

##### redis_tls_server_name

The server name or IP address used by the client certificate.

- Default value: none
- Environment variable: `FLEET_REDIS_TLS_SERVER_NAME`
- Config file format:

  ```
  redis:
  	server_name: 127.0.0.1
  ```

##### redis_tls_handshake_timeout

The timeout for the Redis TLS handshake part of the connection. A value of 0 means no timeout.

- Default value: 10s
- Environment variable: `FLEET_REDIS_TLS_HANDSHAKE_TIMEOUT`
- Config file format:

  ```
  redis:
  	tls_handshake_timeout: 10s
  ```

##### redis_max_idle_conns

Maximum idle connections to Redis. This value should be equal to or less than `redis_max_open_conns`.

- Default value: 3
- Environment variable: `FLEET_REDIS_MAX_IDLE_CONNS`
- Config file format:

  ```
  redis:
  	max_idle_conns: 50
  ```

##### redis_max_open_conns

Maximum open connections to Redis. A value of 0 means no limit.

- Default value: 0
- Environment variable: `FLEET_REDIS_MAX_OPEN_CONNS`
- Config file format:

  ```
  redis:
  	max_open_conns: 100
  ```

##### redis_conn_max_lifetime

Maximum amount of time a Redis connection may be reused. A value of 0 means no limit.

- Default value: 0 (Unlimited)
- Environment variable: `FLEET_REDIS_CONN_MAX_LIFETIME`
- Config file format:

  ```
  redis:
  	conn_max_lifetime: 30m
  ```

##### redis_idle_timeout

Maximum amount of time a Redis connection may stay idle. A value of 0 means no limit.

- Default value: 240s
- Environment variable: `FLEET_REDIS_IDLE_TIMEOUT`
- Config file format:

  ```
  redis:
  	idle_timeout: 5m
  ```

##### redis_conn_wait_timeout

Maximum amount of time to wait for a Redis connection if the max_open_conns
limit is reached. A value of 0 means no wait. This is ignored if Redis is not
running in cluster mode.

- Default value: 0
- Environment variable: `FLEET_REDIS_CONN_WAIT_TIMEOUT`
- Config file format:

  ```
  redis:
  	conn_wait_timeout: 1s
  ```

#### Server

##### server_address

The address to serve the Fleet webserver.

- Default value: `0.0.0.0:8080`
- Environment variable: `FLEET_SERVER_ADDRESS`
- Config file format:

  ```
  server:
  	address: 0.0.0.0:443
  ```

##### server_cert

The TLS cert to use when terminating TLS.

See [TLS certificate considerations](./01-Introduction.md#tls-certificate) for more information about certificates and Fleet.

- Default value: `./tools/osquery/fleet.crt`
- Environment variable: `FLEET_SERVER_CERT`
- Config file format:

  ```
  server:
  	cert: /tmp/fleet.crt
  ```

##### server_key

The TLS key to use when terminating TLS.

- Default value: `./tools/osquery/fleet.key`
- Environment variable: `FLEET_SERVER_KEY`
- Config file format:

  ```
  server:
  	key: /tmp/fleet.key
  ```

##### server_tls

Whether or not the server should be served over TLS.

- Default value: `true`
- Environment variable: `FLEET_SERVER_TLS`
- Config file format:

  ```
  server:
  	tls: false
  ```

##### server_tls_compatibility

Configures the TLS settings for compatibility with various user agents. Options are `modern` and `intermediate`. These correspond to the compatibility levels [defined by the Mozilla OpSec team](https://wiki.mozilla.org/index.php?title=Security/Server_Side_TLS&oldid=1229478) (updated July 24, 2020).

- Default value: `intermediate`
- Environment variable: `FLEET_SERVER_TLS_COMPATIBILITY`
- Config file format:

  ```
  server:
  	tls_compatibility: intermediate
  ```

##### server_url_prefix

Sets a URL prefix to use when serving the Fleet API and frontend. Prefixes should be in the form `/apps/fleet` (no trailing slash).

Note that some other configurations may need to be changed when modifying the URL prefix. In particular, URLs that are provided to osquery via flagfile, the configuration served by Fleet, the URL prefix used by `fleetctl`, and the redirect URL set with an SSO Identity Provider.

- Default value: Empty (no prefix set)
- Environment variable: `FLEET_SERVER_URL_PREFIX`
- Config file format:

  ```
  server:
  	url_prefix: /apps/fleet
  ```

##### server_keepalive

Controls the server side http keep alive property.

Turning off keepalives has helped reduce outstanding TCP connections in some deployments.

- Default value: true
- Environment variable: `FLEET_SERVER_KEEPALIVE`
- Config file format:

  ```
  server:
  	keepalive: true
  ```

#### Auth

##### auth_bcrypt_cost

The bcrypt cost to use when hashing user passwords.

- Default value: `12`
- Environment variable: `FLEET_AUTH_BCRYPT_COST`
- Config file format:

  ```
  auth:
  	bcrypt_cost: 14
  ```

##### auth_salt_key_size

The key size of the salt which is generated when hashing user passwords.

- Default value: `24`
- Environment variable: `FLEET_AUTH_SALT_KEY_SIZE`
- Config file format:

  ```
  auth:
  	salt_key_size: 36
  ```

#### App

##### app_token_key_size

Size of generated app tokens.

- Default value: `24`
- Environment variable: `FLEET_APP_TOKEN_KEY_SIZE`
- Config file format:

  ```
  app:
  	token_key_size: 36
  ```

##### app_invite_token_validity_period

How long invite tokens should be valid for.

- Default value: `5 days`
- Environment variable: `FLEET_APP_INVITE_TOKEN_VALIDITY_PERIOD`
- Config file format:

  ```
  app:
  	invite_token_validity_period: 1d
  ```

#### License

##### license_key

The license key provided to Fleet customers which provides access to Fleet Premium features.

- Default value: none
- Environment variable: `FLEET_LICENSE_KEY`
- Config file format:

  ```
  license:
    key: foobar
  ```

#### Session

##### session_key_size

The size of the session key.

- Default value: `64`
- Environment variable: `FLEET_SESSION_KEY_SIZE`
- Config file format:

  ```
  session:
  	key_size: 48
  ```

##### session_duration

The amount of time that a session should last for.

Valid time units are `s`, `m`, `h`.

- Default value: `24h` (24 hours)
- Environment variable: `FLEET_SESSION_DURATION`
- Config file format:

  ```
  session:
  	duration: 4h
  ```

#### Osquery

##### osquery_node_key_size

The size of the node key which is negotiated with `osqueryd` clients.

- Default value: `24`
- Environment variable: `FLEET_OSQUERY_NODE_KEY_SIZE`
- Config file format:

  ```
  osquery:
  	node_key_size: 36
  ```

##### osquery_host_identifier

The identifier to use when determining uniqueness of hosts.

Options are `provided` (default), `uuid`, `hostname`, or `instance`.

This setting works in combination with the `--host_identifier` flag in osquery. In most deployments, using `uuid` will be the best option. The flag defaults to `provided` -- preserving the existing behavior of Fleet's handling of host identifiers -- using the identifier provided by osquery. `instance`, `uuid`, and `hostname` correspond to the same meanings as for osquery's `--host_identifier` flag.

Users that have duplicate UUIDs in their environment can benefit from setting this flag to `instance`.

- Default value: `provided`
- Environment variable: `FLEET_OSQUERY_HOST_IDENTIFIER`
- Config file format:

  ```
  osquery:
  	host_identifier: uuid
  ```

##### osquery_enroll_cooldown

The cooldown period for host enrollment. If a host (uniquely identified by the `osquery_host_identifier` option) tries to enroll within this duration from the last enrollment, enroll will fail.

This flag can be used to control load on the database in scenarios in which many hosts are using the same identifier. Often configuring `osquery_host_identifier` to `instance` may be a better solution.

- Default value: `0` (off)
- Environment variable: `FLEET_OSQUERY_ENROLL_COOLDOWN`
- Config file format:

  ```
  osquery:
  	enroll_cooldown: 1m
  ```

##### osquery_label_update_interval

The interval at which Fleet will ask osquery agents to update their results for label queries.

Setting this to a higher value can reduce baseline load on the Fleet server in larger deployments.

Valid time units are `s`, `m`, `h`.

- Default value: `1h`
- Environment variable: `FLEET_OSQUERY_LABEL_UPDATE_INTERVAL`
- Config file format:

  ```
  osquery:
  	label_update_interval: 30m
  ```

##### osquery_policy_update_interval

The interval at which Fleet will ask osquery agents to update their results for policy queries.

Setting this to a higher value can reduce baseline load on the Fleet server in larger deployments.

Valid time units are `s`, `m`, `h`.

- Default value: `1h`
- Environment variable: `FLEET_OSQUERY_POLICY_UPDATE_INTERVAL`
- Config file format:

  ```
  osquery:
  	policy_update_interval: 30m
  ```

##### osquery_detail_update_interval

The interval at which Fleet will ask osquery agents to update host details (such as uptime, hostname, network interfaces, etc.)

Setting this to a higher value can reduce baseline load on the Fleet server in larger deployments.

Valid time units are `s`, `m`, `h`.

- Default value: `1h`
- Environment variable: `FLEET_OSQUERY_DETAIL_UPDATE_INTERVAL`
- Config file format:

  ```
  osquery:
  	detail_update_interval: 30m
  ```

##### osquery_status_log_plugin

Which log output plugin should be used for osquery status logs received from clients.

Options are `filesystem`, `firehose`, `kinesis`, `lambda`, `pubsub`, `kafkarest`, and `stdout`.

- Default value: `filesystem`
- Environment variable: `FLEET_OSQUERY_STATUS_LOG_PLUGIN`
- Config file format:

  ```
  osquery:
  	status_log_plugin: firehose
  ```

##### osquery_result_log_plugin

Which log output plugin should be used for osquery result logs received from clients.

Options are `filesystem`, `firehose`, `kinesis`, `lambda`, `pubsub`, `kafkarest`, and `stdout`.

- Default value: `filesystem`
- Environment variable: `FLEET_OSQUERY_RESULT_LOG_PLUGIN`
- Config file format:

  ```
  osquery:
  	result_log_plugin: firehose
  ```

##### osquery_max_jitter_percent

Given an update interval (label, or details), this will add up to the defined percentage in randomness to the interval.

The goal of this is to prevent all hosts from checking in with data at the same time.

So for example, if the label_update_interval is 1h, and this is set to 10. It'll add up a random number between 0 and 6 minutes
to the amount of time it takes for fleet to give the host the label queries.

- Default value: `10`
- Environment variable: `FLEET_OSQUERY_MAX_JITTER_PERCENT`
- Config file format:

  ```
  osquery:
  	max_jitter_percent: 10
  ```

##### osquery_enable_async_host_processing

**Experimental feature**. Enable asynchronous processing of hosts query results. Currently, only supported for label query execution results. This may improve performance and CPU usage of the fleet instances and MySQL database servers for setups with a large number of hosts (100 000+), while requiring more resources from Redis server(s). Using Redis Cluster is recommended to enable this mode.

- Default value: false
- Environment variable: `FLEET_OSQUERY_ENABLE_ASYNC_HOST_PROCESSING`
- Config file format:

  ```
  osquery:
  	enable_async_host_processing: true
  ```

##### osquery_async_host_collect_interval

Applies only when `osquery_enable_async_host_processing` is enabled. Sets the interval at which the host data will be collected into the database. Each fleet instance will attempt to do the collection at this interval (with some optional jitter added, see `osquery_async_host_collect_max_jitter_percent`), with only one succeeding to get the exclusive lock.

- Default value: 30s
- Environment variable: `FLEET_OSQUERY_ASYNC_HOST_COLLECT_INTERVAL`
- Config file format:

  ```
  osquery:
  	async_host_collect_interval: 1m
  ```

##### osquery_async_host_collect_max_jitter_percent

Applies only when `osquery_enable_async_host_processing` is enabled. A number interpreted as a percentage of `osquery_async_host_collect_interval` to add to (or remove from) the interval so that not all hosts try to do the collection at the same time.

- Default value: 10
- Environment variable: `FLEET_OSQUERY_ASYNC_HOST_COLLECT_MAX_JITTER_PERCENT`
- Config file format:

  ```
  osquery:
  	async_host_collect_max_jitter_percent: 5
  ```

##### osquery_async_host_collect_lock_timeout

Applies only when `osquery_enable_async_host_processing` is enabled. Timeout of the lock acquired by a fleet instance to collect host data into the database. If the collection runs for too long or the instance crashes unexpectedly, the lock will be automatically released after this duration and another fleet instance can proceed with the next collection.

- Default value: 1m
- Environment variable: `FLEET_OSQUERY_ASYNC_HOST_COLLECT_LOCK_TIMEOUT`
- Config file format:

  ```
  osquery:
  	async_host_collect_lock_timeout: 5m
  ```

##### osquery_async_host_collect_log_stats_interval

Applies only when `osquery_enable_async_host_processing` is enabled. Interval at which the host collection statistics are logged, 0 to disable logging of statistics. Note that logging is done at the "debug" level.

- Default value: 1m
- Environment variable: `FLEET_OSQUERY_ASYNC_HOST_COLLECT_LOG_STATS_INTERVAL`
- Config file format:

  ```
  osquery:
  	async_host_collect_log_stats_interval: 5m
  ```

##### osquery_async_host_insert_batch

Applies only when `osquery_enable_async_host_processing` is enabled. Size of the INSERT batch when collecting host data into the database.

- Default value: 2000
- Environment variable: `FLEET_OSQUERY_ASYNC_HOST_INSERT_BATCH`
- Config file format:

  ```
  osquery:
  	async_host_insert_batch: 1000
  ```

##### osquery_async_host_delete_batch

Applies only when `osquery_enable_async_host_processing` is enabled. Size of the DELETE batch when collecting host data into the database.

- Default value: 2000
- Environment variable: `FLEET_OSQUERY_ASYNC_HOST_DELETE_BATCH`
- Config file format:

  ```
  osquery:
  	async_host_delete_batch: 1000
  ```

##### osquery_async_host_update_batch

Applies only when `osquery_enable_async_host_processing` is enabled. Size of the UPDATE batch when collecting host data into the database.

- Default value: 1000
- Environment variable: `FLEET_OSQUERY_ASYNC_HOST_UPDATE_BATCH`
- Config file format:

  ```
  osquery:
  	async_host_update_batch: 500
  ```

##### osquery_async_host_redis_pop_count

Applies only when `osquery_enable_async_host_processing` is enabled. Maximum number of items to pop from a redis key at a time when collecting host data into the database.

- Default value: 1000
- Environment variable: `FLEET_OSQUERY_ASYNC_HOST_REDIS_POP_COUNT`
- Config file format:

  ```
  osquery:
  	async_host_redis_pop_count: 500
  ```

##### osquery_async_host_redis_scan_keys_count

Applies only when `osquery_enable_async_host_processing` is enabled. Order of magnitude (e.g. 10, 100, 1000, etc.) of keys to scan in a single SCAN request for keys to process when collecting host data into the database.

- Default value: 1000
- Environment variable: `FLEET_OSQUERY_ASYNC_HOST_REDIS_SCAN_KEYS_COUNT`
- Config file format:

  ```
  osquery:
  	async_host_redis_scan_keys_count: 100
  ```

#### Logging (Fleet server logging)

##### logging_debug

Whether or not to enable debug logging.

- Default value: `false`
- Environment variable: `FLEET_LOGGING_DEBUG`
- Config file format:

  ```
  logging:
  	debug: true
  ```

##### logging_json

Whether or not to log in JSON.

- Default value: `false`
- Environment variable: `FLEET_LOGGING_JSON`
- Config file format:

  ```
  logging:
  	json: true
  ```

##### logging_disable_banner

Whether or not to log the welcome banner.

- Default value: `false`
- Environment variable: `FLEET_LOGGING_DISABLE_BANNER`
- Config file format:

  ```
  logging:
  	disable_banner: true
  ```

##### logging_error_retention_period

The amount of time to keep an error. Unique instances of errors are stored temporarily to help
with troubleshooting, this setting controls that duration.

- Default value: 24h
- Environment variable: `FLEET_LOGGING_ERROR_RETENTION_PERIOD`
- Config file format:

  ```
  logging:
  	error_retention_period: 1h
  ```

#### Filesystem

##### filesystem_status_log_file

This flag only has effect if `osquery_status_log_plugin` is set to `filesystem` (the default value).

The path which osquery status logs will be logged to.

- Default value: `/tmp/osquery_status`
- Environment variable: `FLEET_FILESYSTEM_STATUS_LOG_FILE`
- Config file format:

  ```
  filesystem:
  	status_log_file: /var/log/osquery/status.log
  ```

##### filesystem_result_log_file

This flag only has effect if `osquery_result_log_plugin` is set to `filesystem` (the default value).

The path which osquery result logs will be logged to.

- Default value: `/tmp/osquery_result`
- Environment variable: `FLEET_FILESYSTEM_RESULT_LOG_FILE`
- Config file format:

  ```
  filesystem:
  	result_log_file: /var/log/osquery/result.log
  ```

##### filesystem_enable_log_rotation

This flag only has effect if `osquery_result_log_plugin` or `osquery_status_log_plugin` are set to `filesystem` (the default value).

This flag will cause the osquery result and status log files to be automatically
rotated when files reach a size of 500 Mb or an age of 28 days.

- Default value: `false`
- Environment variable: `FLEET_FILESYSTEM_ENABLE_LOG_ROTATION`
- Config file format:

  ```
  filesystem:
     enable_log_rotation: true
  ```

##### filesystem_enable_log_compression

This flag only has effect if `filesystem_enable_log_rotation` is set to `true`.

This flag will cause the rotated logs to be compressed with gzip.

- Default value: `false`
- Environment variable: `FLEET_FILESYSTEM_ENABLE_LOG_COMPRESSION`
- Config file format:

  ```
  filesystem:
     enable_log_compression: true
  ```

#### Firehose

##### firehose_region

This flag only has effect if `osquery_status_log_plugin` is set to `firehose`.

AWS region to use for Firehose connection

- Default value: none
- Environment variable: `FLEET_FIREHOSE_REGION`
- Config file format:

  ```
  firehose:
  	region: ca-central-1
  ```

##### firehose_access_key_id

This flag only has effect if `osquery_status_log_plugin` or `osquery_result_log_plugin` are set to `firehose`.

If `firehose_access_key_id` and `firehose_secret_access_key` are omitted, Fleet will try to use [AWS STS](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html) credentials.

AWS access key ID to use for Firehose authentication.

- Default value: none
- Environment variable: `FLEET_FIREHOSE_ACCESS_KEY_ID`
- Config file format:

  ```
  firehose:
  	access_key_id: AKIAIOSFODNN7EXAMPLE
  ```

##### firehose_secret_access_key

This flag only has effect if `osquery_status_log_plugin` or `osquery_result_log_plugin` are set to `firehose`.

AWS secret access key to use for Firehose authentication.

- Default value: none
- Environment variable: `FLEET_FIREHOSE_SECRET_ACCESS_KEY`
- Config file format:

  ```
  firehose:
  	secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  ```

##### firehose_sts_assume_role_arn

This flag only has effect if `osquery_status_log_plugin` or
`osquery_result_log_plugin` are set to `firehose`.

AWS STS role ARN to use for Firehose authentication.

- Default value: none
- Environment variable: `FLEET_FIREHOSE_STS_ASSUME_ROLE_ARN`
- Config file format:

  ```
  firehose:
  	sts_assume_role_arn: arn:aws:iam::1234567890:role/firehose-role
  ```

##### firehose_status_stream

This flag only has effect if `osquery_status_log_plugin` is set to `firehose`.

Name of the Firehose stream to write osquery status logs received from clients.

- Default value: none
- Environment variable: `FLEET_FIREHOSE_STATUS_STREAM`
- Config file format:

  ```
  firehose:
  	status_stream: osquery_status
  ```

The IAM role used to send to Firehose must allow the following permissions on
the stream listed:

- `firehose:DescribeDeliveryStream`
- `firehose:PutRecordBatch`

##### firehose_result_stream

This flag only has effect if `osquery_result_log_plugin` is set to `firehose`.

Name of the Firehose stream to write osquery result logs received from clients.

- Default value: none
- Environment variable: `FLEET_FIREHOSE_RESULT_STREAM`
- Config file format:

  ```
  firehose:
  	result_stream: osquery_result
  ```

The IAM role used to send to Firehose must allow the following permissions on
the stream listed:

- `firehose:DescribeDeliveryStream`
- `firehose:PutRecordBatch`

#### Kinesis

##### kinesis_region

This flag only has effect if `osquery_status_log_plugin` is set to `kinesis`.

AWS region to use for Kinesis connection

- Default value: none
- Environment variable: `FLEET_KINESIS_REGION`
- Config file format:

  ```
  kinesis:
  	region: ca-central-1
  ```

##### kinesis_access_key_id

This flag only has effect if `osquery_status_log_plugin` or
`osquery_result_log_plugin` are set to `kinesis`.

If `kinesis_access_key_id` and `kinesis_secret_access_key` are omitted, Fleet
will try to use
[AWS STS](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html)
credentials.

AWS access key ID to use for Kinesis authentication.

- Default value: none
- Environment variable: `FLEET_KINESIS_ACCESS_KEY_ID`
- Config file format:

  ```
  kinesis:
  	access_key_id: AKIAIOSFODNN7EXAMPLE
  ```

##### kinesis_secret_access_key

This flag only has effect if `osquery_status_log_plugin` or
`osquery_result_log_plugin` are set to `kinesis`.

AWS secret access key to use for Kinesis authentication.

- Default value: none
- Environment variable: `FLEET_KINESIS_SECRET_ACCESS_KEY`
- Config file format:

  ```
  kinesis:
  	secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  ```

##### kinesis_sts_assume_role_arn

This flag only has effect if `osquery_status_log_plugin` or
`osquery_result_log_plugin` are set to `kinesis`.

AWS STS role ARN to use for Kinesis authentication.

- Default value: none
- Environment variable: `FLEET_KINESIS_STS_ASSUME_ROLE_ARN`
- Config file format:

  ```
  kinesis:
  	sts_assume_role_arn: arn:aws:iam::1234567890:role/kinesis-role
  ```

##### kinesis_status_stream

This flag only has effect if `osquery_status_log_plugin` is set to `kinesis`.

Name of the Kinesis stream to write osquery status logs received from clients.

- Default value: none
- Environment variable: `FLEET_KINESIS_STATUS_STREAM`
- Config file format:

  ```
  kinesis:
  	status_stream: osquery_status
  ```

The IAM role used to send to Kinesis must allow the following permissions on
the stream listed:

- `kinesis:DescribeStream`
- `kinesis:PutRecords`

##### kinesis_result_stream

This flag only has effect if `osquery_result_log_plugin` is set to `kinesis`.

Name of the Kinesis stream to write osquery result logs received from clients.

- Default value: none
- Environment variable: `FLEET_KINESIS_RESULT_STREAM`
- Config file format:

  ```
  kinesis:
  	result_stream: osquery_result
  ```

The IAM role used to send to Kinesis must allow the following permissions on
the stream listed:

- `kinesis:DescribeStream`
- `kinesis:PutRecords`

#### Lambda

##### lambda_region

This flag only has effect if `osquery_status_log_plugin` is set to `lambda`.

AWS region to use for Lambda connection

- Default value: none
- Environment variable: `FLEET_LAMBDA_REGION`
- Config file format:

  ```
  lambda:
  	region: ca-central-1
  ```

##### lambda_access_key_id

This flag only has effect if `osquery_status_log_plugin` or
`osquery_result_log_plugin` are set to `lambda`.

If `lambda_access_key_id` and `lambda_secret_access_key` are omitted, Fleet
will try to use
[AWS STS](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html)
credentials.

AWS access key ID to use for Lambda authentication.

- Default value: none
- Environment variable: `FLEET_LAMBDA_ACCESS_KEY_ID`
- Config file format:

  ```
  lambda:
  	access_key_id: AKIAIOSFODNN7EXAMPLE
  ```

##### lambda_secret_access_key

This flag only has effect if `osquery_status_log_plugin` or
`osquery_result_log_plugin` are set to `lambda`.

AWS secret access key to use for Lambda authentication.

- Default value: none
- Environment variable: `FLEET_LAMBDA_SECRET_ACCESS_KEY`
- Config file format:

  ```
  lambda:
  	secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  ```

##### lambda_sts_assume_role_arn

This flag only has effect if `osquery_status_log_plugin` or
`osquery_result_log_plugin` are set to `lambda`.

AWS STS role ARN to use for Lambda authentication.

- Default value: none
- Environment variable: `FLEET_LAMBDA_STS_ASSUME_ROLE_ARN`
- Config file format:

  ```
  lambda:
  	sts_assume_role_arn: arn:aws:iam::1234567890:role/lambda-role
  ```

##### lambda_status_function

This flag only has effect if `osquery_status_log_plugin` is set to `lambda`.

Name of the Lambda function to write osquery status logs received from clients.

- Default value: none
- Environment variable: `FLEET_LAMBDA_STATUS_FUNCTION`
- Config file format:

  ```
  lambda:
  	status_function: statusFunction
  ```

The IAM role used to send to Lambda must allow the following permissions on
the function listed:

- `lambda:InvokeFunction`

##### lambda_result_function

This flag only has effect if `osquery_result_log_plugin` is set to `lambda`.

Name of the Lambda function to write osquery result logs received from clients.

- Default value: none
- Environment variable: `FLEET_LAMBDA_RESULT_FUNCTION`
- Config file format:

  ```
  lambda:
  	result_function: resultFunction
  ```

The IAM role used to send to Lambda must allow the following permissions on
the function listed:

- `lambda:InvokeFunction`

#### PubSub

##### pubsub_project

This flag only has effect if `osquery_status_log_plugin` is set to `pubsub`.

The identifier of the Google Cloud project containing the pubsub topics to
publish logs to.

Note that the pubsub plugin uses [Application Default Credentials (ADCs)](https://cloud.google.com/docs/authentication/production)
for authentication with the service.

- Default value: none
- Environment variable: `FLEET_PUBSUB_PROJECT`
- Config file format:

  ```
  pubsub:
    project: my-gcp-project
  ```

##### pubsub_result_topic

This flag only has effect if `osquery_status_log_plugin` is set to `pubsub`.

The identifier of the pubsub topic that client results will be published to.

- Default value: none
- Environment variable: `FLEET_PUBSUB_RESULT_TOPIC`
- Config file format:

  ```
  pubsub:
    result_topic: osquery_result
  ```

##### pubsub_status_topic

This flag only has effect if `osquery_status_log_plugin` is set to `pubsub`.

The identifier of the pubsub topic that osquery status logs will be published to.

- Default value: none
- Environment variable: `FLEET_PUBSUB_STATUS_TOPIC`
- Config file format:

  ```
  pubsub:
    status_topic: osquery_status
  ```

##### pubsub_add_attributes

This flag only has effect if `osquery_status_log_plugin` is set to `pubsub`.

Add Pub/Sub attributes to messages. When enabled, the plugin parses the osquery result
messages, and adds the following Pub/Sub message attributes:

- `name` - the `name` attribute from the message body
- `timestamp` - the `unixTime` attribute from the message body, converted to rfc3339 format
- Each decoration from the message

This feature is useful when combined with [subscription filters](https://cloud.google.com/pubsub/docs/filtering).

- Default value: false
- Environment variable: `FLEET_PUBSUB_ADD_ATTRIBUTES`
- Config file format:

  ```
  pubsub:
    status_topic: osquery_status
  ```

#### Kafka Rest Proxy Logging

##### kafkarest_proxyhost

This flag only has effect if `osquery_status_log_plugin` or `osquery_result_log_plugin` is set to `kafkarest`.

The URL of the host which to check for the topic existence and post messages to the specified topic.

- Default value: none
- Environment variable: `FLEET_KAFKAREST_PROXYHOST`
- Config file format:

  ```
  kafkarest:
    proxyhost: "https://localhost:8443"
  ```

##### kafkarest_status_topic

This flag only has effect if `osquery_status_log_plugin` is set to `kafkarest`.

The identifier of the kafka topic that osquery status logs will be published to.

- Default value: none
- Environment variable: `FLEET_KAFKAREST_STATUS_TOPIC`
- Config file format:

  ```
  kafkarest:
    status_topic: osquery_status
  ```

##### kafkarest_result_topic

This flag only has effect if `osquery_result_log_plugin` is set to `kafkarest`.

The identifier of the kafka topic that osquery status logs will be published to.

- Default value: none
- Environment variable: `FLEET_KAFKAREST_RESULT_TOPIC`
- Config file format:

  ```
  kafkarest:
    status_topic: osquery_result
  ```

##### kafkarest_timeout

This flag only has effect if `osquery_status_log_plugin` or `osquery_result_log_plugin` is set to `kafkarest`.

The timeout value for the http post attempt. Value is in units of seconds.

- Default value: 5
- Environment variable: `FLEET_KAFKAREST_TIMEOUT`
- Config file format:

  ```
  kafkarest:
    timeout: 5
  ```

#### S3 file carving backend

##### s3_bucket

Name of the S3 bucket to use to store file carves.

- Default value: none
- Environment variable: `FLEET_S3_BUCKET`
- Config file format:

  ```
  s3:
  	bucket: some-carve-bucket
  ```

##### s3_prefix

Prefix to prepend to carve objects.

All carve objects will also be prefixed by date and hour (UTC), making the resulting keys look like: `<prefix><year>/<month>/<day>/<hour>/<carve-name>`.

- Default value: none
- Environment variable: `FLEET_S3_PREFIX`
- Config file format:

  ```
  s3:
  	prefix: carves-go-here/
  ```

##### s3_access_key_id

AWS access key ID to use for S3 authentication.

If `s3_access_key_id` and `s3_secret_access_key` are omitted, Fleet will try to use
[the default credential provider chain](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials).

The IAM identity used in this context must be allowed to perform the following actions on the bucket: `s3:PutObject`, `s3:GetObject`, `s3:ListMultipartUploadParts`, `s3:ListBucket`, `s3:GetBucketLocation`.

- Default value: none
- Environment variable: `FLEET_S3_ACCESS_KEY_ID`
- Config file format:

  ```
  s3:
  	access_key_id: AKIAIOSFODNN7EXAMPLE
  ```

##### s3_secret_access_key

AWS secret access key to use for S3 authentication.

- Default value: none
- Environment variable: `FLEET_S3_SECRET_ACCESS_KEY`
- Config file format:

  ```
  s3:
  	secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  ```

##### s3_sts_assume_role_arn

AWS STS role ARN to use for S3 authentication.

- Default value: none
- Environment variable: `FLEET_S3_STS_ASSUME_ROLE_ARN`
- Config file format:

  ```
  s3:
  	sts_assume_role_arn: arn:aws:iam::1234567890:role/some-s3-role
  ```

##### s3_endpoint_url

AWS S3 Endpoint URL. Override when using a different S3 compatible object storage backend (such as Minio),
or running s3 locally with localstack. Leave this blank to use the default S3 service endpoint.

- Default value: none
- Environment variable: `FLEET_S3_ENDPOINT_URL`
- Config file format:

  ```
  s3:
  	endpoint_url: http://localhost:9000
  ```

##### s3_disable_ssl

AWS S3 Disable SSL. Useful for local testing.

- Default value: false
- Environment variable: `FLEET_S3_DISABLE_SSL`
- Config file format:

  ```
  s3:
  	disable_ssl: false
  ```

##### s3_force_s3_path_style

AWS S3 Force S3 Path Style. Set this to `true` to force the request to use path-style addressing,
i.e., `http://s3.amazonaws.com/BUCKET/KEY`. By default, the S3 client
will use virtual hosted bucket addressing when possible
(`http://BUCKET.s3.amazonaws.com/KEY`).

See [here](http://docs.aws.amazon.com/AmazonS3/latest/dev/VirtualHosting.html) for details.

- Default value: false
- Environment variable: `FLEET_S3_FORCE_S3_PATH_STYLE`
- Config file format:

  ```
  s3:
  	force_s3_path_style: false
  ```

##### s3_region

AWS S3 Region. Leave blank to enable region discovery.

Minio users must set this to any nonempty value (eg. `minio`), as Minio does not support region discovery.

- Default value:
- Environment variable: `FLEET_S3_REGION`
- Config file format:

  ```
  s3:
  	region: us-east-1
  ```

#### Upgrades

##### allow_missing_migations

If set then `fleet serve` will run even if there are database migrations missing.

- Default value: `false`
- Environment variable: `FLEET_UPGRADES_ALLOW_MISSING_MIGRATIONS`
- Config file format:

  ```
  upgrades:
    allow_missing_migrations: true
  ```

#### Vulnerabilities

##### databases_path

The path specified needs to exist and fleet needs to be able to read and write to and from it. This is the only mandatory configuration needed for vulnerability processing to work.

When `current_instance_checks` is set to `auto` (the default), Fleet instances will try to create the `databases_path` if it doesn't exist.

- Default value: none
- Environment variable: `FLEET_VULNERABILITIES_DATABASES_PATH`
- Config file format:

  ```
  vulnerabilities:
  	databases_path: /some/path
  ```

##### periodicity

How often vulnerabilities are checked.

- Default value: `1h`
- Environment variable: `FLEET_VULNERABILITIES_PERIODICITY`
- Config file format:

  ```
  vulnerabilities:
  	periodicity: 1h
  ```

##### cpe_database_url

URL to fetch the CPE dictionary database from. Some users want to control where fleet gets its database from. When Fleet sees this value defined, it downloads the file directly. It expects a file in the same format as can be found in https://github.com/fleetdm/nvd/releases. If this value is not defined, Fleet checks for the latest release in Github and only downloads it if needed.

- Default value: `""`
- Environment variable: `FLEET_VULNERABILITIES_CPE_DATABASE_URL`
- Config file format:

  ```
  vulnerabilities:
  	cpe_database_url: ""
  ```

##### cve_feed_prefix_url

Similarly to the CPE dictionary, we allow users to define where to get the CVE feeds from. In this case, the url should be a host that serves the files in the path /feeds/json/cve/1.1/. Fleet expects to find there all the JSON Feeds that can be found in https://nvd.nist.gov/vuln/data-feeds. When not defined, Fleet downloads from the nvd.nist.gov host.

- Default value: `""`
- Environment variable: `FLEET_VULNERABILITIES_CVE_FEED_PREFIX_URL`
- Config file format:

  ```
  vulnerabilities:
  	cve_database_url: ""
  ```

##### current_instance_checks

When running multiple instances of the Fleet server, by default, one of them dynamically takes the lead in vulnerability processing. This lead can change over time. Some Fleet users want to be able to define which deployment is doing this checking. If you wish to do this, you'll need to deploy your Fleet instances with this set explicitly to no and one of them set to yes.

- Default value: `auto`
- Environment variable: `FLEET_VULNERABILITIES_CURRENT_INSTANCE_CHECKS`
- Config file format:

  ```
  vulnerabilities:
  	current_instance_checks: yes
  ```

##### disable_data_sync

Fleet by default automatically downloads and keeps the different data streams needed to properly do vulnerability processing. In some setups, this behavior is not wanted, as access to outside resources might be blocked, or the data stream files might need review/audit before use.

In order to support vulnerability processing in such environments, we allow users to disable automatic sync of data streams with this configuration value.

To download the data streams, you can use `fleetctl vulnerability-data-stream --dir ./somedir`. The contents downloaded can then be reviewed, and finally uploaded to the defined `databases_path` in the fleet instance(s) doing the vulnerability processing.

- Default value: false
- Environment variable: `FLEET_VULNERABILITIES_DISABLE_DATA_SYNC`
- Config file format:

  ```
  vulnerabilities:
  	disable_data_sync: true
  ```

## Managing osquery configurations

We recommend that you use an infrastructure configuration management tool to manage these osquery configurations consistently across your environment. If you're unsure about what configuration management tools your organization uses, contact your company's system administrators. If you are evaluating new solutions for this problem, the founders of Fleet have successfully managed configurations in large production environments using [Chef](https://www.chef.io/chef/) and [Puppet](https://puppet.com/).

## Running with systemd

Once you've verified that you can run Fleet in your shell, you'll likely want to keep Fleet running in the background and after the server reboots. To do that we recommend using [systemd](https://coreos.com/os/docs/latest/getting-started-with-systemd.html).

Below is a sample unit file.

```

[Unit]
Description=Fleet
After=network.target

[Service]
LimitNOFILE=8192
ExecStart=/usr/local/bin/fleet serve \
  --mysql_address=127.0.0.1:3306 \
  --mysql_database=fleet \
  --mysql_username=root \
  --mysql_password=toor \
  --redis_address=127.0.0.1:6379 \
  --server_cert=/tmp/server.cert \
  --server_key=/tmp/server.key \
  --logging_json

[Install]
WantedBy=multi-user.target
```

Once you created the file, you need to move it to `/etc/systemd/system/fleet.service` and start the service.

```
sudo mv fleet.service /etc/systemd/system/fleet.service
sudo systemctl start fleet.service
sudo systemctl status fleet.service

sudo journalctl -u fleet.service -f
```

### Making changes

Sometimes you'll need to update the systemd unit file defining the service. To do that, first open /etc/systemd/system/fleet.service in a text editor, and make your modifications.

Then, run

```
sudo systemctl daemon-reload
sudo systemctl restart fleet.service
```

## Using a proxy

If you are in an enterprise environment where Fleet is behind a proxy and you would like to be able to retrieve Vulnerability data for [Vulnerability Processing](https://fleetdm.com/docs/using-fleet/vulnerability-processing#vulnerability-processing), it may be necessary to configure the proxy settings. Fleet automatically uses the `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` environment variables.

For example, to configure the proxy in a systemd service file:

```
[Service]
Environment="HTTP_PROXY=http(s)://PROXY_URL:PORT/"
Environment="HTTPS_PROXY=http(s)://PROXY_URL:PORT/"
Environment="NO_PROXY=localhost,127.0.0.1,::1"
```

After modifying the configuration you will need to reload and restart the Fleet service, as explained above.

## Configuring single sign on (SSO)

Fleet supports SAML single sign on capability.

Fleet supports both SP-initiated SAML login and IDP-initiated login, however IDP-initiated login must be enabled in the web interface's SAML single sign on options.

Fleet supports the SAML Web Browser SSO Profile using the HTTP Redirect Binding.

### Identity Provider (IDP) Configuration

Setting up the service provider (Fleet) with an identity provider generally requires the following information:

- _Assertion Consumer Service_ - This is the call back URL that the identity provider
  will use to send security assertions to Fleet. In Okta, this field is called _Single sign on URL_. On Google it is "ACS URL". The value that you supply will be a fully qualified URL consisting of your Fleet web address and the callback path `/api/v1/fleet/sso/callback`. For example, if your Fleet web address is https://fleet.example.com, then the value you would use in the identity provider configuration would be:

  ```
  https://fleet.example.com/api/v1/fleet/sso/callback
  ```

- _Entity ID_ - This value is an identifier that you choose. It identifies your Fleet instance as the service provider that issues authorization requests. The value must exactly match the Entity ID that you define in the Fleet SSO configuration.

- _Name ID Format_ - The value should be `urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress`. This may be shortened in the IDP setup to something like `email` or `EmailAddress`.

- _Subject Type (Application username in Okta)_ - `email`.

After supplying the above information, the IDP will generate an issuer URI and a metadata that will be used to configure Fleet as a service provider.

### Fleet SSO Configuration

A Fleet user must be assigned the Admin role to configure Fleet for SSO. In Fleet, SSO configuration settings are located in **Settings > Organization settings > SAML single sign on options**.

If your IDP supports dynamic configuration, like Okta, you only need to provide an _Identity Provider Name_ and _Entity ID_, then paste a link in the metadata URL field.

Otherwise, the following values are required:

- _Identity Provider Name_ - A human readable name of the IDP. This is rendered on the login page.

- _Entity ID_ - A URI that identifies your Fleet instance as the issuer of authorization
  requests (eg. `fleet.example.com`). This much match the _Entity ID_ configured with the IDP.

- _Issuer URI_ - This value is obtained from the IDP.

- _Metadata URL_ - This value is obtained from the IDP and is used by Fleet to
  issue authorization requests to the IDP.

- _Metadata_ - If the IDP does not provide a metadata URL, the metadata must
  be obtained from the IDP and entered. Note that the metadata URL is preferred if
  the IDP provides metadata in both forms.

#### Example Fleet SSO Configuration

![Example SSO Configuration](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/sso-setup.png)

### Creating SSO users in Fleet

When an admin creates a new user to Fleet, they may select the `Enable single sign on` option. The
SSO enabled users will not be able to sign in with a regular user ID and password.

It is strongly recommended that at least one admin user is set up to use the traditional password
based log in so that there is a fallback method for logging into Fleet in the event of SSO
configuration problems.

#### Okta IDP Configuration

![Example Okta IDP Configuration](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/okta-idp-setup.png)

> The names of the items required to configure an Identity Provider may vary from provider to provider and may not conform to the SAML spec.

> Individual users must also be setup on the IDP before they can sign in to Fleet.

#### Google Workspace IDP Configuration

Follow these steps to configure Fleet SSO with Google Workspace. This will require administrator permissions in Google Workspace.

1. Navigate to the [Web and Mobile Apps](https://admin.google.com/ac/apps/unified) section of the Google Workspace dashboard. Click _Add App -> Add custom SAML app_.

![The Google Workspace admin dashboard](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-1.png)

2. Enter `Fleet` for the _App name_ and click _Continue_.

![Adding a new app to Google workspace admin dashboard](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-2.png)

3. Click _Download Metadata_, saving the metadata to your computer. Copy the _SSO URL_. Click _Continue_.

![Download metadata and copy the SSO URL](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-3.png)

4. In Fleet, navigate to the _Organization Settings_ page. Configure the _SAML single sign on options_ section.

- Check the _Enable single sign on_ checkbox.
- For _Identity provider name_ use `Google`.
- For _Entity ID_, use a unique identifier such as `fleet.example.com`. Note that Google seems to error when the provided ID includes `https://`.
- For _Issuer URI_, paste the _SSO URL_ copied from step 3.
- For _Metadata_, paste the contents of the downloaded metadata XML from step 3.
- All other fields can be left blank.

Click _Update settings_ at the bottom of the page.

![Fleet's SAML single sign on options page](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-4.png)

5. In Google Workspace, configure the _Service provider details_.

- For _ACS URL_, use `https://<your_fleet_url>/api/v1/fleet/sso/callback` (eg. `https://fleet.example.com/api/v1/fleet/sso/callback`).
- For Entity ID, use **the same unique identifier from step 4** (eg. `fleet.example.com`).
- For _Name ID format_ choose `EMAIL`.
- For _Name ID_ choose `Basic Information > Primary email`.
- All other fields can be left blank.

Click _Continue_ at the bottom of the page.

![Configuring the service provider details in Google Workspace](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-5.png)

6. Click _Finish_.

![Finish configuring the new SAML app in Google Workspace](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-6.png)

7. Click the down arrow on the _User access_ section of the app details page.

![The new SAML app's details page in Google Workspace](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-7.png)

8. Check _ON for everyone_. Click _Save_.

![The new SAML app's service status page in Google Workspace](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-8.png)

9. Enable SSO for a test user and try logging in. Note that Google sometimes takes a long time to propagate the SSO configuration, and it can help to try logging in to Fleet with an Incognito/Private window in the browser.

## Feature flags

Fleet features are sometimes gated behind feature flags. This will usually be due to not-yet-stable APIs, or not-fully-tested performance characteristics.

Feature flags on the server are controlled by environment variables prefixed with `FLEET_BETA_`.
