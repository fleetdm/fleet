# Fleet server configuration

## Configuring the Fleet binary

For information on how to run the `fleet` binary, find detailed usage information by running `fleet --help`. This document is a more detailed version of the data presented in the help output text. If you prefer to use a CLI instead of a web browser, we hope  you like the binary interface of the Fleet application!

### High-level configuration overview

In order to get the most out of running the Fleet server, it is helpful to establish a mutual understanding of what the desired architecture looks like and what it's trying to accomplish.

Your Fleet server's two main purposes are:

- To serve as your [osquery TLS server](https://osquery.readthedocs.io/en/stable/deployment/remote/)
- To serve the Fleet web UI, which allows you to manage osquery configuration, query hosts, etc.

The Fleet server allows you to persist configuration, manage users, etc. Thus, it needs a database. Fleet uses MySQL and requires you to supply configurations to connect to a MySQL server. It is also possible to configure your connection to a MySQL replica in addition to the primary. This is for reading only. Fleet also uses Redis to perform more high-speed data access action throughout the applications lifecycle (for example, distributed query result ingestion). Thus, Fleet also requires that you supply Redis connection configurations.

Fleet can scale to hundreds of thousands of devices with a single Redis instance and is also compatible with Redis Cluster. Fleet does not support Redis Sentinel.

Since Fleet is a web application, when you run it there are other configurations that must be defined, such as:

- The TLS certificates that Fleet should use to terminate TLS.

When deploying Fleet, mitigate DoS attacks as you would when deploying any app.

Since Fleet is an osquery TLS server, you are also able to define configurations that can customize your experience there, such as:

- The destination of the osquery status and result logs on the local filesystem
- Various details about the refresh/check-in intervals for your hosts

### Options

#### How do you specify options?

You can specify options in the order of precedence via

1. a configuration file (in YAML format)
2. environment variables
3. command-line flags

For example, all of the following ways of launching Fleet are equivalent:

##### 1. Using a YAML config file

```sh
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

For more information on using YAML configuration files with fleet, please see the [configuration files](https://fleetdm.com/docs/using-fleet/configuration-files) documentation.

##### 2. Using only environment variables

```sh
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

##### 3. Using only CLI flags

```sh
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


### What are the options?

Note that all option names can be converted consistently from flag name to environment variable and visa-versa. For example, the `--mysql_address` flag would be the `FLEET_MYSQL_ADDRESS`. Further, specifying the `mysql_address` option in the config would follow the pattern:

```yaml
mysql:
  address: 127.0.0.1:3306
```

And `mysql_read_replica_address` would be:

```yaml
mysql_read_replica:
  address: 127.0.0.1:3307
```

Basically, just capitalize the option and prepend `FLEET_` to it to get the environment variable. The conversion works the same the opposite way.

All duration-based settings accept valid time units of `s`, `m`, `h`.

#### MySQL

This section describes the configuration options for the primary. Suppose you also want to set up a read replica. In that case the options are the same, except that the YAML section is `mysql_read_replica`, and the flags have the `mysql_read_replica_` prefix instead of `mysql_` (the corresponding environment variables follow the same transformation). Note that there is no default value for `mysql_read_replica_address`, it must be set explicitly for Fleet to use a read replica, and it is recommended in that case to set a non-zero value for `mysql_read_replica_conn_max_lifetime` as in some environments, the replica's address may dynamically change to point
from the primary to an actual distinct replica based on auto-scaling options, so existing idle connections need to be recycled
periodically.

##### mysql_address

For the address of the MySQL server that Fleet should connect to, include the hostname and port.

- Default value: `localhost:3306`
- Environment variable: `FLEET_MYSQL_ADDRESS`
- Config file format:
  ```yaml
  mysql:
    address: localhost:3306
  ```

##### mysql_database

This is the name of the MySQL database which Fleet will use.

- Default value: `fleet`
- Environment variable: `FLEET_MYSQL_DATABASE`
- Config file format:
  ```yaml
  mysql:
    database: fleet
  ```

##### mysql_username

The username to use when connecting to the MySQL instance.

- Default value: `fleet`
- Environment variable: `FLEET_MYSQL_USERNAME`
- Config file format:
  ```yaml
  mysql:
    username: fleet
  ```

##### mysql_password

The password to use when connecting to the MySQL instance.

- Default value: `fleet`
- Environment variable: `FLEET_MYSQL_PASSWORD`
- Config file format:
  ```yaml
  mysql:
    password: fleet
  ```

##### mysql_password_path

File path to a file that contains the password to use when connecting to the MySQL instance.

- Default value: `""`
- Environment variable: `FLEET_MYSQL_PASSWORD_PATH`
- Config file format:
  ```yaml
  mysql:
    password_path: '/run/secrets/fleetdm-mysql-password'
  ```

##### mysql_tls_ca

The path to a PEM encoded certificate of MYSQL's CA for client certificate authentication.

- Default value: none
- Environment variable: `FLEET_MYSQL_TLS_CA`
- Config file format:
  ```yaml
  mysql:
    tls_ca: /path/to/server-ca.pem
  ```

##### mysql_tls_cert

The path to a PEM encoded certificate is used for TLS authentication.

- Default value: none
- Environment variable: `FLEET_MYSQL_TLS_CERT`
- Config file format:
  ```yaml
  mysql:
    tls_cert: /path/to/certificate.pem
  ```

##### mysql_tls_key

The path to a PEM encoded private key used for TLS authentication.

- Default value: none
- Environment variable: `FLEET_MYSQL_TLS_KEY`
- Config file format:
  ```yaml
  mysql:
    tls_key: /path/to/key.pem
  ```

##### mysql_tls_config

The TLS value in an MYSQL DSN. Can be `true`,`false`,`skip-verify`, or the CN value of the certificate.

- Default value: none
- Environment variable: `FLEET_MYSQL_TLS_CONFIG`
- Config file format:
  ```yaml
  mysql:
    tls_config: true
  ```

##### mysql_tls_server_name

This is the server name or IP address used by the client certificate.

- Default value: none
- Environment variable: `FLEET_MYSQL_TLS_SERVER_NAME`
- Config file format:
  ```yaml
  mysql:
    server_name: 127.0.0.1
  ```

##### mysql_max_open_conns

The maximum open connections to the database.

- Default value: 50
- Environment variable: `FLEET_MYSQL_MAX_OPEN_CONNS`
- Config file format:
  ```yaml
  mysql:
    max_open_conns: 50
  ```

##### mysql_max_idle_conns

The maximum idle connections to the database. This value should be equal to or less than `mysql_max_open_conns`.

- Default value: 50
- Environment variable: `FLEET_MYSQL_MAX_IDLE_CONNS`
- Config file format:
  ```yaml
  mysql:
    max_idle_conns: 50
  ```

##### mysql_conn_max_lifetime

The maximum amount of time, in seconds, a connection may be reused.

- Default value: 0 (Unlimited)
- Environment variable: `FLEET_MYSQL_CONN_MAX_LIFETIME`
- Config file format:
  ```yaml
  mysql:
    conn_max_lifetime: 50
  ```

##### mysql_sql_mode

Sets the connection `sql_mode`. See [MySQL Reference](https://dev.mysql.com/doc/refman/5.7/en/sql-mode.html) for more details.
This setting should not usually be used.

- Default value: `""`
- Environment variable: `FLEET_MYSQL_SQL_MODE`
- Config file format:
  ```yaml
  mysql:
    sql_mode: ANSI
  ```

##### Example YAML

```yaml
mysql:
  address: localhost:3306
  database: fleet
  password: fleet
  max_open_conns: 50
  max_idle_conns: 50
  conn_max_lifetime: 50
```

#### Redis

Note that to test a TLS connection to a Redis instance, run the
`tlsconnect` Go program in `tools/redis-tests`, e.g., from the root of the repository:

```sh
$ go run ./tools/redis-tests/tlsconnect.go -addr <redis_address> -cacert <redis_tls_ca> -cert <redis_tls_cert> -key <redis_tls_key>
# run `go run ./tools/redis-tests/tlsconnect.go -h` for the full list of supported flags
```

By default, this will set up a Redis pool for that configuration and execute a
`PING` command with a TLS connection, printing any error it encounters.

##### redis_address

For the address of the Redis server that Fleet should connect to, include the hostname and port.

- Default value: `localhost:6379`
- Environment variable: `FLEET_REDIS_ADDRESS`
- Config file format:
  ```yaml
  redis:
    address: 127.0.0.1:7369
  ```

##### redis_username

The username to use when connecting to the Redis instance.

- Default value: `<empty>`
- Environment variable: `FLEET_REDIS_USERNAME`
- Config file format:
  ```yaml
  redis:
    username: foobar
  ```

##### redis_password

The password to use when connecting to the Redis instance.

- Default value: `<empty>`
- Environment variable: `FLEET_REDIS_PASSWORD`
- Config file format:
  ```yaml
  redis:
    password: foobar
  ```

##### redis_database

The database to use when connecting to the Redis instance.

- Default value: `0`
- Environment variable: `FLEET_REDIS_DATABASE`
- Config file format:
  ```yaml
  redis:
    database: 14
  ```

##### redis_use_tls

Use a TLS connection to the Redis server.

- Default value: `false`
- Environment variable: `FLEET_REDIS_USE_TLS`
- Config file format:
  ```yaml
  redis:
    use_tls: true
  ```

##### redis_duplicate_results

Whether or not to duplicate Live Query results to another Redis channel named `LQDuplicate`. This is useful in a scenario involving shipping the Live Query results outside of Fleet, near real-time.

- Default value: `false`
- Environment variable: `FLEET_REDIS_DUPLICATE_RESULTS`
- Config file format:
  ```yaml
  redis:
    duplicate_results: true
  ```

##### redis_connect_timeout

Timeout for redis connection.

- Default value: 5s
- Environment variable: `FLEET_REDIS_CONNECT_TIMEOUT`
- Config file format:
  ```yaml
  redis:
    connect_timeout: 10s
  ```

##### redis_keep_alive

The interval between keep-alive probes.

- Default value: 10s
- Environment variable: `FLEET_REDIS_KEEP_ALIVE`
- Config file format:
  ```yaml
  redis:
    keep_alive: 30s
  ```

##### redis_connect_retry_attempts

The maximum number of attempts to retry a failed connection to a Redis node. Only
certain types of errors are retried, such as connection timeouts.

- Default value: 0 (no retry)
- Environment variable: `FLEET_REDIS_CONNECT_RETRY_ATTEMPTS`
- Config file format:
  ```yaml
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
  ```yaml
  redis:
    cluster_follow_redirections: true
  ```

##### redis_cluster_read_from_replica

Whether or not to prefer reading from a replica when possible. Applies only
to Redis Cluster setups, ignored in standalone Redis.

- Default value: false
- Environment variable: `FLEET_REDIS_CLUSTER_READ_FROM_REPLICA`
- Config file format:
  ```yaml
  redis:
    cluster_read_from_replica: true
  ```

##### redis_tls_cert

This is the path to a PEM-encoded certificate used for TLS authentication.

- Default value: none
- Environment variable: `FLEET_REDIS_TLS_CERT`
- Config file format:
  ```yaml
  redis:
    tls_cert: /path/to/certificate.pem
  ```

##### redis_tls_key

This is the path to a PEM-encoded private key used for TLS authentication.

- Default value: none
- Environment variable: `FLEET_REDIS_TLS_KEY`
- Config file format:
  ```yaml
  redis:
    tls_key: /path/to/key.pem
  ```

##### redis_tls_ca

This is the path to a PEM-encoded certificate of Redis' CA for client certificate authentication.

- Default value: none
- Environment variable: `FLEET_REDIS_TLS_CA`
- Config file format:
  ```yaml
  redis:
    tls_ca: /path/to/server-ca.pem
  ```

##### redis_tls_server_name

The server name or IP address used by the client certificate.

- Default value: none
- Environment variable: `FLEET_REDIS_TLS_SERVER_NAME`
- Config file format:
  ```yaml
  redis:
    tls_server_name: 127.0.0.1
  ```

##### redis_tls_handshake_timeout

The timeout for the Redis TLS handshake part of the connection. A value of 0 means no timeout.

- Default value: 10s
- Environment variable: `FLEET_REDIS_TLS_HANDSHAKE_TIMEOUT`
- Config file format:
  ```yaml
  redis:
    tls_handshake_timeout: 10s
  ```

##### redis_max_idle_conns

The maximum idle connections to Redis. This value should be equal to or less than `redis_max_open_conns`.

- Default value: 3
- Environment variable: `FLEET_REDIS_MAX_IDLE_CONNS`
- Config file format:
  ```yaml
  redis:
    max_idle_conns: 50
  ```

##### redis_max_open_conns

The maximum open connections to Redis. A value of 0 means no limit.

- Default value: 0
- Environment variable: `FLEET_REDIS_MAX_OPEN_CONNS`
- Config file format:
  ```yaml
  redis:
    max_open_conns: 100
  ```

##### redis_conn_max_lifetime

The maximum time a Redis connection may be reused. A value of 0 means no limit.

- Default value: 0 (Unlimited)
- Environment variable: `FLEET_REDIS_CONN_MAX_LIFETIME`
- Config file format:
  ```yaml
  redis:
    conn_max_lifetime: 30m
  ```

##### redis_idle_timeout

The maximum time a Redis connection may stay idle. A value of 0 means no limit.

- Default value: 240s
- Environment variable: `FLEET_REDIS_IDLE_TIMEOUT`
- Config file format:
  ```yaml
  redis:
    idle_timeout: 5m
  ```

##### redis_conn_wait_timeout

The maximum time to wait for a Redis connection if the max_open_conns
limit is reached. A value of 0 means no wait. This is ignored if Redis is not
running in cluster mode.

- Default value: 0
- Environment variable: `FLEET_REDIS_CONN_WAIT_TIMEOUT`
- Config file format:
  ```yaml
  redis:
    conn_wait_timeout: 1s
  ```

##### redis_read_timeout

The maximum time to wait to receive a response from a Redis server.
A value of 0 means no timeout.

- Default value: 10s
- Environment variable: `FLEET_REDIS_READ_TIMEOUT`
- Config file format:
  ```yaml
  redis:
    read_timeout: 5s
  ```

##### redis_write_timeout

The maximum time to wait to send a command to a Redis server.
A value of 0 means no timeout.

- Default value: 10s
- Environment variable: `FLEET_REDIS_WRITE_TIMEOUT`
- Config file format:
  ```yaml
  redis:
    write_timeout: 5s
  ```

##### Example YAML

```yaml
redis:
  address: localhost:7369
  password: foobar
  database: 14
  connect_timeout: 10s
  connect_retry_attempts: 2
```

### Server

##### server_address

The address to serve the Fleet webserver.

- Default value: `0.0.0.0:8080`
- Environment variable: `FLEET_SERVER_ADDRESS`
- Config file format:
  ```yaml
  server:
    address: 0.0.0.0:443
  ```

##### server_cert

The TLS cert to use when terminating TLS.

See [TLS certificate considerations](https://fleetdm.com/docs/deploying/introduction#tls-certificate) for more information about certificates and Fleet.

- Default value: `./tools/osquery/fleet.crt`
- Environment variable: `FLEET_SERVER_CERT`
- Config file format:
  ```yaml
  server:
    cert: /tmp/fleet.crt
  ```

##### server_key

The TLS key to use when terminating TLS.

- Default value: `./tools/osquery/fleet.key`
- Environment variable: `FLEET_SERVER_KEY`
- Config file format:
  ```yaml
  server:
    key: /tmp/fleet.key
  ```

##### server_tls

Whether or not the server should be served over TLS.

- Default value: `true`
- Environment variable: `FLEET_SERVER_TLS`
- Config file format:
  ```yaml
  server:
    tls: false
  ```

##### server_tls_compatibility

Configures the TLS settings for compatibility with various user agents. Options are `modern` and `intermediate`. These correspond to the compatibility levels [defined by the Mozilla OpSec team](https://wiki.mozilla.org/index.php?title=Security/Server_Side_TLS&oldid=1229478) (updated July 24, 2020).

- Default value: `intermediate`
- Environment variable: `FLEET_SERVER_TLS_COMPATIBILITY`
- Config file format:
  ```yaml
  server:
    tls_compatibility: intermediate
  ```

##### server_url_prefix

Sets a URL prefix to use when serving the Fleet API and frontend. Prefixes should be in the form `/apps/fleet` (no trailing slash).

Note that some other configurations may need to be changed when modifying the URL prefix. In particular, URLs that are provided to osquery via flagfile, the configuration served by Fleet, the URL prefix used by `fleetctl`, and the redirect URL set with an identity provider.

- Default value: Empty (no prefix set)
- Environment variable: `FLEET_SERVER_URL_PREFIX`
- Config file format:
  ```yaml
  server:
    url_prefix: /apps/fleet
  ```

##### server_keepalive

Controls the server side http keep alive property.

Turning off keepalives has helped reduce outstanding TCP connections in some deployments.

- Default value: true
- Environment variable: `FLEET_SERVER_KEEPALIVE`
- Config file format:
  ```yaml
  server:
    keepalive: true
  ```

##### server_websockets_allow_unsafe_origin

Controls the servers websocket origin check. If your Fleet server is behind a reverse proxy,
the Origin header may not reflect the client's true origin. In this case, you might need to 
disable the origin header (by setting this configuration to `true`) 
check or configure your reverse proxy to forward the correct Origin header.

Setting to true will disable the origin check.

- Default value: false
- Environment variable: `FLEET_SERVER_WEBSOCKETS_ALLOW_UNSAFE_ORIGIN`
- Config file format:
  ```yaml
  server:
    websockets_allow_unsafe_origin: true
  ```

##### Example YAML

```yaml
server:
  address: 0.0.0.0:443
  password: foobar
  cert: /tmp/fleet.crt
  key: /tmp/fleet.key
  invite_token_validity_period: 1d
```

#### Auth

##### auth_bcrypt_cost

The bcrypt cost to use when hashing user passwords.

- Default value: `12`
- Environment variable: `FLEET_AUTH_BCRYPT_COST`
- Config file format:
  ```yaml
  auth:
    bcrypt_cost: 14
  ```

##### auth_salt_key_size

The key size of the salt which is generated when hashing user passwords.

- Default value: `24`
- Environment variable: `FLEET_AUTH_SALT_KEY_SIZE`
- Config file format:
  ```yaml
  auth:
    salt_key_size: 36
  ```

##### Example YAML

```yaml
auth:
  bcrypt_cost: 14
  salt_key_size: 36
```

#### App

##### app_token_key_size

Size of generated app tokens.

- Default value: `24`
- Environment variable: `FLEET_APP_TOKEN_KEY_SIZE`
- Config file format:
  ```yaml
  app:
    token_key_size: 36
  ```

##### app_invite_token_validity_period

How long invite tokens should be valid for.

- Default value: `5 days`
- Environment variable: `FLEET_APP_INVITE_TOKEN_VALIDITY_PERIOD`
- Config file format:
  ```yaml
  app:
    invite_token_validity_period: 1d
  ```

##### app_enable_scheduled_query_stats

Determines whether Fleet gets scheduled query statistics from hosts or not.

- Default value: `true`
- Environment variable: `FLEET_APP_ENABLE_SCHEDULED_QUERY_STATS`
- Config file format:
  ```yaml
  app:
    enable_scheduled_query_stats: true
  ```

##### Example YAML

```yaml
app:
  token_key_size: 36
  salt_key_size: 36
  invite_token_validity_period: 1d
```

#### License

##### license_key

The license key provided to Fleet customers which provides access to Fleet Premium features.

- Default value: none
- Environment variable: `FLEET_LICENSE_KEY`
- Config file format:
  ```yaml
  license:
    key: foobar
  ```

##### license_enforce_host_limit

Whether Fleet should enforce the host limit of the license, if true, attempting to enroll new hosts when the limit is reached will fail.

- Default value: `false`
- Environment variable: `FLEET_LICENSE_ENFORCE_HOST_LIMIT`
- Config file format:
  ```yaml
  license:
    enforce_host_limit: true
  ```

##### Example YAML

```yaml
license:
  key: foobar
  enforce_host_limit: false
```

#### Session

##### session_key_size

The size of the session key.

- Default value: `64`
- Environment variable: `FLEET_SESSION_KEY_SIZE`
- Config file format:
  ```yaml
  session:
    key_size: 48
  ```

##### session_duration

This is the amount of time that a session should last. Whenever a user logs in, the time is reset to the specified, or default, duration.

Valid time units are `s`, `m`, `h`.

- Default value: `5d` (5 days)
- Environment variable: `FLEET_SESSION_DURATION`
- Config file format:
  ```yaml
  session:
    duration: 4h
  ```

##### Example YAML

```yaml
session:
  duration: 4h
```

#### Osquery

##### osquery_node_key_size

The size of the node key which is negotiated with `osqueryd` clients.

- Default value: `24`
- Environment variable: `FLEET_OSQUERY_NODE_KEY_SIZE`
- Config file format:
  ```yaml
  osquery:
    node_key_size: 36
  ```

##### osquery_host_identifier

The identifier to use when determining uniqueness of hosts.

Options are `provided` (default), `uuid`, `hostname`, or `instance`.

This setting works in combination with the `--host_identifier` flag in osquery. In most deployments, using `uuid` will be the best option. The flag defaults to `provided` -- preserving the existing behavior of Fleet's handling of host identifiers -- using the identifier provided by osquery. `instance`, `uuid`, and `hostname` correspond to the same meanings as osquery's `--host_identifier` flag.

Users that have duplicate UUIDs in their environment can benefit from setting this flag to `instance`.

> If you are enrolling your hosts using Fleet generated packages, it is reccommended to use `uuid` as your indentifier. This prevents potential issues with duplicate host enrollments.

- Default value: `provided`
- Environment variable: `FLEET_OSQUERY_HOST_IDENTIFIER`
- Config file format:
  ```yaml
  osquery:
    host_identifier: uuid
  ```

##### osquery_enroll_cooldown

The cooldown period for host enrollment. If a host (uniquely identified by the `osquery_host_identifier` option) tries to enroll within this duration from the last enrollment, enroll will fail.

This flag can be used to control load on the database in scenarios in which many hosts are using the same identifier. Often configuring `osquery_host_identifier` to `instance` may be a better solution.

- Default value: `0` (off)
- Environment variable: `FLEET_OSQUERY_ENROLL_COOLDOWN`
- Config file format:
  ```yaml
  osquery:
    enroll_cooldown: 1m
  ```

##### osquery_label_update_interval

The interval at which Fleet will ask osquery agents to update their results for label queries.

Setting this to a higher value can reduce baseline load on the Fleet server in larger deployments.

> Setting this to a lower value can increase baseline load significantly and cause performance issues or even outages. Proceed with caution.

Valid time units are `s`, `m`, `h`.

- Default value: `1h`
- Environment variable: `FLEET_OSQUERY_LABEL_UPDATE_INTERVAL`
- Config file format:
  ```yaml
  osquery:
    label_update_interval: 90m
  ```

##### osquery_policy_update_interval

The interval at which Fleet will ask osquery agents to update their results for policy queries.

Setting this to a higher value can reduce baseline load on the Fleet server in larger deployments.

> Setting this to a lower value can increase baseline load significantly and cause performance issues or even outages. Proceed with caution.

Valid time units are `s`, `m`, `h`.

- Default value: `1h`
- Environment variable: `FLEET_OSQUERY_POLICY_UPDATE_INTERVAL`
- Config file format:
  ```yaml
  osquery:
    policy_update_interval: 90m
  ```

##### osquery_detail_update_interval

The interval at which Fleet will ask osquery agents to update host details (such as uptime, hostname, network interfaces, etc.)

Setting this to a higher value can reduce baseline load on the Fleet server in larger deployments.

> Setting this to a lower value can increase baseline load significantly and cause performance issues or even outages. Proceed with caution.

Valid time units are `s`, `m`, `h`.

- Default value: `1h`
- Environment variable: `FLEET_OSQUERY_DETAIL_UPDATE_INTERVAL`
- Config file format:
  ```yaml
  osquery:
    detail_update_interval: 90m
  ```

##### osquery_status_log_plugin

This is the log output plugin that should be used for osquery status logs received from clients. Check out the [reference documentation for log destinations](https://fleetdm.com/docs/using-fleet/log-destinations).


Options are `filesystem`, `firehose`, `kinesis`, `lambda`, `pubsub`, `kafkarest`, and `stdout`.

- Default value: `filesystem`
- Environment variable: `FLEET_OSQUERY_STATUS_LOG_PLUGIN`
- Config file format:
  ```yaml
  osquery:
    status_log_plugin: firehose
  ```

##### osquery_result_log_plugin

This is the log output plugin that should be used for osquery result logs received from clients. Check out the [reference documentation for log destinations](https://fleetdm.com/docs/using-fleet/log-destinations).

Options are `filesystem`, `firehose`, `kinesis`, `lambda`, `pubsub`, `kafkarest`, and `stdout`.

- Default value: `filesystem`
- Environment variable: `FLEET_OSQUERY_RESULT_LOG_PLUGIN`
- Config file format:
  ```yaml
  osquery:
    result_log_plugin: firehose
  ```

##### osquery_max_jitter_percent

Given an update interval (label, or details), this will add up to the defined percentage in randomness to the interval.

The goal of this is to prevent all hosts from checking in with data at the same time.

So for example, if the label_update_interval is 1h, and this is set to 10. It'll add up a random number between 0 and 6 minutes
to the amount of time it takes for Fleet to give the host the label queries.

- Default value: `10`
- Environment variable: `FLEET_OSQUERY_MAX_JITTER_PERCENT`
- Config file format:
  ```yaml
  osquery:
    max_jitter_percent: 10
  ```

##### osquery_enable_async_host_processing

**Experimental feature**. Enable asynchronous processing of hosts' query results. Currently, asyncronous processing is only supported for label query execution, policy membership results, hosts' last seen timestamp, and hosts' scheduled query statistics. This may improve the performance and CPU usage of the Fleet instances and MySQL database servers for setups with a large number of hosts while requiring more resources from Redis server(s).

Note that currently, if both the failing policies webhook *and* this `osquery.enable_async_host_processing` option are set, some failing policies webhooks could be missing (some transitions from succeeding to failing or vice-versa could happen without triggering a webhook request).

It can be set to a single boolean value ("true" or "false"), which controls all async host processing tasks, or it can be set for specific async tasks using a syntax similar to an URL query string or parameters in a Data Source Name (DSN) string, e.g., "label_membership=true&policy_membership=true". When using the per-task syntax, omitted tasks get the default value. The supported async task names are:

* `label_membership` for updating the hosts' label query execution;
* `policy_membership` for updating the hosts' policy membership results;
* `host_last_seen` for updating the hosts' last seen timestamp.
* `scheduled_query_stats` for saving the hosts' scheduled query statistics.

- Default value: false
- Environment variable: `FLEET_OSQUERY_ENABLE_ASYNC_HOST_PROCESSING`
- Config file format:
  ```yaml
  osquery:
    enable_async_host_processing: true
  ```

 > Fleet tested this option for `policy_membership=true` in [this issue](https://github.com/fleetdm/fleet/issues/12697) and found that it does not impact the performance or behavior of the app.

##### osquery_async_host_collect_interval

Applies only when `osquery_enable_async_host_processing` is enabled. Sets the interval at which the host data will be collected into the database. Each Fleet instance will attempt to do the collection at this interval (with some optional jitter added, see `osquery_async_host_collect_max_jitter_percent`), with only one succeeding to get the exclusive lock.

It can be set to a single duration value (e.g., "30s"), which defines the interval for all async host processing tasks, or it can be set for specific async tasks using a syntax similar to an URL query string or parameters in a Data Source Name (DSN) string, e.g., "label_membership=10s&policy_membership=1m". When using the per-task syntax, omitted tasks get the default value. See [osquery_enable_async_host_processing](#osquery_enable_async_host_processing) for the supported async task names.

- Default value: 30s
- Environment variable: `FLEET_OSQUERY_ASYNC_HOST_COLLECT_INTERVAL`
- Config file format:
  ```yaml
  osquery:
    async_host_collect_interval: 1m
  ```

##### osquery_async_host_collect_max_jitter_percent

Applies only when `osquery_enable_async_host_processing` is enabled. A number interpreted as a percentage of `osquery_async_host_collect_interval` to add to (or remove from) the interval so that not all hosts try to do the collection at the same time.

- Default value: 10
- Environment variable: `FLEET_OSQUERY_ASYNC_HOST_COLLECT_MAX_JITTER_PERCENT`
- Config file format:
  ```yaml
  osquery:
    async_host_collect_max_jitter_percent: 5
  ```

##### osquery_async_host_collect_lock_timeout

Applies only when `osquery_enable_async_host_processing` is enabled. Timeout of the lock acquired by a Fleet instance to collect host data into the database. If the collection runs for too long or the instance crashes unexpectedly, the lock will be automatically released after this duration and another Fleet instance can proceed with the next collection.

It can be set to a single duration value (e.g., "1m"), which defines the lock timeout for all async host processing tasks, or it can be set for specific async tasks using a syntax similar to an URL query string or parameters in a Data Source Name (DSN) string, e.g., "label_membership=2m&policy_membership=5m". When using the per-task syntax, omitted tasks get the default value. See [osquery_enable_async_host_processing](#osquery_enable_async_host_processing) for the supported async task names.

- Default value: 1m
- Environment variable: `FLEET_OSQUERY_ASYNC_HOST_COLLECT_LOCK_TIMEOUT`
- Config file format:
  ```yaml
  osquery:
    async_host_collect_lock_timeout: 5m
  ```

##### osquery_async_host_collect_log_stats_interval

Applies only when `osquery_enable_async_host_processing` is enabled. Interval at which the host collection statistics are logged, 0 to disable logging of statistics. Note that logging is done at the "debug" level.

- Default value: 1m
- Environment variable: `FLEET_OSQUERY_ASYNC_HOST_COLLECT_LOG_STATS_INTERVAL`
- Config file format:
  ```yaml
  osquery:
    async_host_collect_log_stats_interval: 5m
  ```

##### osquery_async_host_insert_batch

Applies only when `osquery_enable_async_host_processing` is enabled. Size of the INSERT batch when collecting host data into the database.

- Default value: 2000
- Environment variable: `FLEET_OSQUERY_ASYNC_HOST_INSERT_BATCH`
- Config file format:
  ```yaml
  osquery:
    async_host_insert_batch: 1000
  ```

##### osquery_async_host_delete_batch

Applies only when `osquery_enable_async_host_processing` is enabled. Size of the DELETE batch when collecting host data into the database.

- Default value: 2000
- Environment variable: `FLEET_OSQUERY_ASYNC_HOST_DELETE_BATCH`
- Config file format:
  ```yaml
  osquery:
    async_host_delete_batch: 1000
  ```

##### osquery_async_host_update_batch

Applies only when `osquery_enable_async_host_processing` is enabled. Size of the UPDATE batch when collecting host data into the database.

- Default value: 1000
- Environment variable: `FLEET_OSQUERY_ASYNC_HOST_UPDATE_BATCH`
- Config file format:
  ```yaml
  osquery:
    async_host_update_batch: 500
  ```

##### osquery_async_host_redis_pop_count

Applies only when `osquery_enable_async_host_processing` is enabled. Maximum number of items to pop from a redis key at a time when collecting host data into the database.

- Default value: 1000
- Environment variable: `FLEET_OSQUERY_ASYNC_HOST_REDIS_POP_COUNT`
- Config file format:
  ```yaml
  osquery:
    async_host_redis_pop_count: 500
  ```

##### osquery_async_host_redis_scan_keys_count

Applies only when `osquery_enable_async_host_processing` is enabled. Order of magnitude (e.g., 10, 100, 1000, etc.) of set members to scan in a single ZSCAN/SSCAN request for items to process when collecting host data into the database.

- Default value: 1000
- Environment variable: `FLEET_OSQUERY_ASYNC_HOST_REDIS_SCAN_KEYS_COUNT`
- Config file format:
  ```yaml
  osquery:
    async_host_redis_scan_keys_count: 100
  ```

##### osquery_min_software_last_opened_at_diff

The minimum time difference between the software's "last opened at" timestamp reported by osquery and the last timestamp saved for that software on that host helps minimize the number of updates required when a host reports its installed software information, resulting in less load on the database. If there is no existing timestamp for the software on that host (or if the software was not installed on that host previously), the new timestamp is automatically saved.

- Default value: 1h
- Environment variable: `FLEET_OSQUERY_MIN_SOFTWARE_LAST_OPENED_AT_DIFF`
- Config file format:
  ```yaml
  osquery:
    min_software_last_opened_at_diff: 4h
  ```

##### Example YAML

```yaml
osquery:
  host_identifier: uuid
  policy_update_interval: 30m
  duration: 4h
  status_log_plugin: firehose
  result_log_plugin: firehose
```
#### External Activity Audit Logging

> Applies only to Fleet Premium. Acitivity information is available for all Fleet instances using the [Activities API](https://fleetdm.com/docs/using-fleet/rest-api#activities).

Stream Fleet user activities to logs using Fleet's logging plugins. The audit events are logged in an asynchronous fashion. It can take up to 5 minutes for an event to be logged.

##### activity_enable_audit_log

This enables/disables the log output for audit events.
See the `activity_audit_log_plugin` option below that specifies the logging destination.

- Default value: `false`
- Environment variable: `FLEET_ACTIVITY_ENABLE_AUDIT_LOG`
- Config file format:
  ```yaml
  activity:
    enable_audit_log: true
  ```

##### activity_audit_log_plugin

This is the log output plugin that should be used for audit logs.
This flag only has effect if `activity_enable_audit_log` is set to `true`.

Each plugin has additional configuration options. Please see the configuration section linked below for your logging plugin.

Options are [`filesystem`](#filesystem), [`firehose`](#firehose), [`kinesis`](#kinesis), [`lambda`](#lambda), [`pubsub`](#pubsub), [`kafkarest`](#kafka-rest-proxy-logging), and `stdout` (no additional configuration needed).

- Default value: `filesystem`
- Environment variable: `FLEET_ACTIVITY_AUDIT_LOG_PLUGIN`
- Config file format:
  ```yaml
  activity:
    audit_log_plugin: firehose
  ```

#### Logging (Fleet server logging)

##### logging_debug

Whether or not to enable debug logging.

- Default value: `false`
- Environment variable: `FLEET_LOGGING_DEBUG`
- Config file format:
  ```yaml
  logging:
    debug: true
  ```

##### logging_json

Whether or not to log in JSON.

- Default value: `false`
- Environment variable: `FLEET_LOGGING_JSON`
- Config file format:
  ```yaml
  logging:
    json: true
  ```

##### logging_disable_banner

Whether or not to log the welcome banner.

- Default value: `false`
- Environment variable: `FLEET_LOGGING_DISABLE_BANNER`
- Config file format:
  ```yaml
  logging:
    disable_banner: true
  ```

##### logging_error_retention_period

The amount of time to keep an error. Unique instances of errors are stored temporarily to help
with troubleshooting, this setting controls that duration. Set to 0 to keep them without expiration,
and a negative value to disable storage of errors in Redis.

- Default value: 24h
- Environment variable: `FLEET_LOGGING_ERROR_RETENTION_PERIOD`
- Config file format:
  ```yaml
  logging:
    error_retention_period: 1h
  ```

##### Example YAML

```yaml
logging:
  disable_banner: true
  policy_update_interval: 30m
  error_retention_period: 1h
```
#### Filesystem

##### filesystem_status_log_file

This flag only has effect if `osquery_status_log_plugin` is set to `filesystem` (the default value).

The path which osquery status logs will be logged to.

- Default value: `/tmp/osquery_status`
- Environment variable: `FLEET_FILESYSTEM_STATUS_LOG_FILE`
- Config file format:
  ```yaml
  filesystem:
    status_log_file: /var/log/osquery/status.log
  ```

##### filesystem_result_log_file

This flag only has effect if `osquery_result_log_plugin` is set to `filesystem` (the default value).

The path which osquery result logs will be logged to.

- Default value: `/tmp/osquery_result`
- Environment variable: `FLEET_FILESYSTEM_RESULT_LOG_FILE`
- Config file format:
  ```yaml
  filesystem:
    result_log_file: /var/log/osquery/result.log
  ```

##### filesystem_audit_log_file

This flag only has effect if `activity_audit_log_plugin` is set to `filesystem` (the default value) and if `activity_enable_audit_log` is set to `true`.

The path which audit logs will be logged to.

- Default value: `/tmp/audit`
- Environment variable: `FLEET_FILESYSTEM_AUDIT_LOG_FILE`
- Config file format:
  ```yaml
  filesystem:
    audit_log_file: /var/log/fleet/audit.log
  ```

##### filesystem_enable_log_rotation

This flag only has effect if one of the following is true:
- `osquery_result_log_plugin` or `osquery_status_log_plugin` are set to `filesystem` (the default value).
- `activity_audit_log_plugin` is set to `filesystem` and `activity_enable_audit_log` is set to `true`.

This flag will cause the osquery result and status log files to be automatically
rotated when files reach a size of 500 Mb or an age of 28 days.

- Default value: `false`
- Environment variable: `FLEET_FILESYSTEM_ENABLE_LOG_ROTATION`
- Config file format:
  ```yaml
  filesystem:
     enable_log_rotation: true
  ```

##### filesystem_enable_log_compression

This flag only has effect if `filesystem_enable_log_rotation` is set to `true`.

This flag will cause the rotated logs to be compressed with gzip.

- Default value: `false`
- Environment variable: `FLEET_FILESYSTEM_ENABLE_LOG_COMPRESSION`
- Config file format:
  ```yaml
  filesystem:
     enable_log_compression: true
  ```

##### filesystem_max_size

This flag only has effect if `filesystem_enable_log_rotation` is set to `true`.

Sets the maximum size in megabytes of log files before it gets rotated.

- Default value: `500`
- Environment variable: `FLEET_FILESYSTEM_MAX_SIZE`
- Config file format:
  ```yaml
  filesystem:
     max_size: 100
  ```

##### filesystem_max_age

This flag only has effect if `filesystem_enable_log_rotation` is set to `true`.

Sets the maximum age in days to retain old log files before deletion. Setting this
to zero will retain all logs.

- Default value: `28`
- Environment variable: `FLEET_FILESYSTEM_MAX_AGE`
- Config file format:
  ```yaml
  filesystem:
     max_age: 0
  ```

##### filesystem_max_backups

This flag only has effect if `filesystem_enable_log_rotation` is set to `true`.

Sets the maximum number of old files to retain before deletion. Setting this
to zero will retain all logs. _Note_ max_age may still cause them to be deleted.

- Default value: `3`
- Environment variable: `FLEET_FILESYSTEM_MAX_BACKUPS`
- Config file format:
  ```yaml
  filesystem:
     max_backups: 0
  ```

##### Example YAML

```yaml
osquery:
  osquery_status_log_plugin: filesystem
  osquery_result_log_plugin: filesystem
filesystem:
  status_log_file: /var/log/osquery/status.log
  result_log_file: /var/log/osquery/result.log
  enable_log_rotation: true
```

#### Firehose

##### firehose_region

This flag only has effect if one of the following is true:
- `osquery_result_log_plugin` or `osquery_status_log_plugin` are set to `firehose`.
- `activity_audit_log_plugin` is set to `firehose` and `activity_enable_audit_log` is set to `true`.

AWS region to use for Firehose connection.

- Default value: none
- Environment variable: `FLEET_FIREHOSE_REGION`
- Config file format:
  ```yaml
  firehose:
    region: ca-central-1
  ```

##### firehose_access_key_id

This flag only has effect if one of the following is true:
- `osquery_result_log_plugin` or `osquery_status_log_plugin` are set to `firehose`.
- `activity_audit_log_plugin` is set to `firehose` and `activity_enable_audit_log` is set to `true`.

If `firehose_access_key_id` and `firehose_secret_access_key` are omitted, Fleet will try to use [AWS STS](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html) credentials.

AWS access key ID to use for Firehose authentication.

- Default value: none
- Environment variable: `FLEET_FIREHOSE_ACCESS_KEY_ID`
- Config file format:
  ```yaml
  firehose:
    access_key_id: AKIAIOSFODNN7EXAMPLE
  ```

##### firehose_secret_access_key

This flag only has effect if one of the following is true:
- `osquery_result_log_plugin` or `osquery_status_log_plugin` are set to `firehose`.
- `activity_audit_log_plugin` is set to `firehose` and `activity_enable_audit_log` is set to `true`.

AWS secret access key to use for Firehose authentication.

- Default value: none
- Environment variable: `FLEET_FIREHOSE_SECRET_ACCESS_KEY`
- Config file format:
  ```yaml
  firehose:
    secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  ```

##### firehose_sts_assume_role_arn

This flag only has effect if one of the following is true:
- `osquery_result_log_plugin` or `osquery_status_log_plugin` are set to `firehose`.
- `activity_audit_log_plugin` is set to `firehose` and `activity_enable_audit_log` is set to `true`.

AWS STS role ARN to use for Firehose authentication.

- Default value: none
- Environment variable: `FLEET_FIREHOSE_STS_ASSUME_ROLE_ARN`
- Config file format:
  ```yaml
  firehose:
    sts_assume_role_arn: arn:aws:iam::1234567890:role/firehose-role
  ```

##### firehose_status_stream

This flag only has effect if `osquery_status_log_plugin` is set to `firehose`.

Name of the Firehose stream to write osquery status logs received from clients.

- Default value: none
- Environment variable: `FLEET_FIREHOSE_STATUS_STREAM`
- Config file format:
  ```yaml
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
  ```yaml
  firehose:
    result_stream: osquery_result
  ```

The IAM role used to send to Firehose must allow the following permissions on
the stream listed:

- `firehose:DescribeDeliveryStream`
- `firehose:PutRecordBatch`

##### firehose_audit_stream

This flag only has effect if `activity_audit_log_plugin` is set to `firehose`.

Name of the Firehose stream to audit logs.

- Default value: none
- Environment variable: `FLEET_FIREHOSE_AUDIT_STREAM`
- Config file format:
  ```yaml
  firehose:
    audit_stream: fleet_audit
  ```

The IAM role used to send to Firehose must allow the following permissions on
the stream listed:

- `firehose:DescribeDeliveryStream`
- `firehose:PutRecordBatch`


##### Example YAML

```yaml
osquery:
  osquery_status_log_plugin: firehose
  osquery_result_log_plugin: firehose
firehose:
  region: ca-central-1
  access_key_id: AKIAIOSFODNN7EXAMPLE
  secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  sts_assume_role_arn: arn:aws:iam::1234567890:role/firehose-role
  status_stream: osquery_status
  result_stream: osquery_result
```

#### Kinesis

##### kinesis_region

This flag only has effect if one of the following is true:
- `osquery_result_log_plugin` or `osquery_status_log_plugin` are set to `kinesis`.
- `activity_audit_log_plugin` is set to `kinesis` and `activity_enable_audit_log` is set to `true`.

AWS region to use for Kinesis connection

- Default value: none
- Environment variable: `FLEET_KINESIS_REGION`
- Config file format:
  ```yaml
  kinesis:
    region: ca-central-1
  ```

##### kinesis_access_key_id

This flag only has effect if one of the following is true:
- `osquery_result_log_plugin` or `osquery_status_log_plugin` are set to `kinesis`.
- `activity_audit_log_plugin` is set to `kinesis` and `activity_enable_audit_log` is set to `true`.

If `kinesis_access_key_id` and `kinesis_secret_access_key` are omitted, Fleet
will try to use
[AWS STS](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html)
credentials.

AWS access key ID to use for Kinesis authentication.

- Default value: none
- Environment variable: `FLEET_KINESIS_ACCESS_KEY_ID`
- Config file format:
  ```yaml
  kinesis:
    access_key_id: AKIAIOSFODNN7EXAMPLE
  ```

##### kinesis_secret_access_key

This flag only has effect if one of the following is true:
- `osquery_result_log_plugin` or `osquery_status_log_plugin` are set to `kinesis`.
- `activity_audit_log_plugin` is set to `kinesis` and `activity_enable_audit_log` is set to `true`.

AWS secret access key to use for Kinesis authentication.

- Default value: none
- Environment variable: `FLEET_KINESIS_SECRET_ACCESS_KEY`
- Config file format:
  ```yaml
  kinesis:
    secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  ```

##### kinesis_sts_assume_role_arn

This flag only has effect if one of the following is true:
- `osquery_result_log_plugin` or `osquery_status_log_plugin` are set to `kinesis`.
- `activity_audit_log_plugin` is set to `kinesis` and `activity_enable_audit_log` is set to `true`.

AWS STS role ARN to use for Kinesis authentication.

- Default value: none
- Environment variable: `FLEET_KINESIS_STS_ASSUME_ROLE_ARN`
- Config file format:
  ```yaml
  kinesis:
    sts_assume_role_arn: arn:aws:iam::1234567890:role/kinesis-role
  ```

##### kinesis_status_stream

This flag only has effect if `osquery_status_log_plugin` is set to `kinesis`.

Name of the Kinesis stream to write osquery status logs received from clients.

- Default value: none
- Environment variable: `FLEET_KINESIS_STATUS_STREAM`
- Config file format:
  ```yaml
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
  ```yaml
  kinesis:
    result_stream: osquery_result
  ```

The IAM role used to send to Kinesis must allow the following permissions on
the stream listed:

- `kinesis:DescribeStream`
- `kinesis:PutRecords`

##### kinesis_audit_stream

This flag only has effect if `activity_audit_log_plugin` is set to `kinesis`.

Name of the Kinesis stream to write audit logs.

- Default value: none
- Environment variable: `FLEET_KINESIS_AUDIT_STREAM`
- Config file format:
  ```yaml
  kinesis:
    audit_stream: fleet_audit
  ```

The IAM role used to send to Kinesis must allow the following permissions on
the stream listed:

- `kinesis:DescribeStream`
- `kinesis:PutRecords`

##### Example YAML

```yaml
osquery:
  osquery_status_log_plugin: kinesis
  osquery_result_log_plugin: kinesis
kinesis:
  region: ca-central-1
  access_key_id: AKIAIOSFODNN7EXAMPLE
  secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  sts_assume_role_arn: arn:aws:iam::1234567890:role/firehose-role
  status_stream: osquery_status
  result_stream: osquery_result
```

#### Lambda

##### lambda_region

This flag only has effect if one of the following is true:
- `osquery_result_log_plugin` or `osquery_status_log_plugin` are set to `lambda`.
- `activity_audit_log_plugin` is set to `lambda` and `activity_enable_audit_log` is set to `true`.

AWS region to use for Lambda connection.

- Default value: none
- Environment variable: `FLEET_LAMBDA_REGION`
- Config file format:
  ```yaml
  lambda:
    region: ca-central-1
  ```

##### lambda_access_key_id

This flag only has effect if one of the following is true:
- `osquery_result_log_plugin` or `osquery_status_log_plugin` are set to `lambda`.
- `activity_audit_log_plugin` is set to `lambda` and `activity_enable_audit_log` is set to `true`.

If `lambda_access_key_id` and `lambda_secret_access_key` are omitted, Fleet
will try to use
[AWS STS](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html)
credentials.

AWS access key ID to use for Lambda authentication.

- Default value: none
- Environment variable: `FLEET_LAMBDA_ACCESS_KEY_ID`
- Config file format:
  ```yaml
  lambda:
    access_key_id: AKIAIOSFODNN7EXAMPLE
  ```

##### lambda_secret_access_key

This flag only has effect if one of the following is true:
- `osquery_result_log_plugin` or `osquery_status_log_plugin` are set to `lambda`.
- `activity_audit_log_plugin` is set to `lambda` and `activity_enable_audit_log` is set to `true`.

AWS secret access key to use for Lambda authentication.

- Default value: none
- Environment variable: `FLEET_LAMBDA_SECRET_ACCESS_KEY`
- Config file format:
  ```yaml
  lambda:
    secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  ```

##### lambda_sts_assume_role_arn

This flag only has effect if one of the following is true:
- `osquery_result_log_plugin` or `osquery_status_log_plugin` are set to `lambda`.
- `activity_audit_log_plugin` is set to `lambda` and `activity_enable_audit_log` is set to `true`.

AWS STS role ARN to use for Lambda authentication.

- Default value: none
- Environment variable: `FLEET_LAMBDA_STS_ASSUME_ROLE_ARN`
- Config file format:
  ```yaml
  lambda:
    sts_assume_role_arn: arn:aws:iam::1234567890:role/lambda-role
  ```

##### lambda_status_function

This flag only has effect if `osquery_status_log_plugin` is set to `lambda`.

Name of the Lambda function to write osquery status logs received from clients.

- Default value: none
- Environment variable: `FLEET_LAMBDA_STATUS_FUNCTION`
- Config file format:
  ```yaml
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
  ```yaml
  lambda:
    result_function: resultFunction
  ```

The IAM role used to send to Lambda must allow the following permissions on
the function listed:

- `lambda:InvokeFunction`

##### lambda_audit_function

This flag only has effect if `activity_audit_log_plugin` is set to `lambda`.

Name of the Lambda function to write audit logs.

- Default value: none
- Environment variable: `FLEET_LAMBDA_AUDIT_FUNCTION`
- Config file format:
  ```yaml
  lambda:
    audit_function: auditFunction
  ```

The IAM role used to send to Lambda must allow the following permissions on
the function listed:

- `lambda:InvokeFunction`

##### Example YAML

```yaml
osquery:
  osquery_status_log_plugin: lambda
  osquery_result_log_plugin: lambda
lambda:
  region: ca-central-1
  access_key_id: AKIAIOSFODNN7EXAMPLE
  secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  sts_assume_role_arn: arn:aws:iam::1234567890:role/firehose-role
  status_function: statusFunction
  result_function: resultFunction
```

#### PubSub

##### pubsub_project

This flag only has effect if one of the following is true:
- `osquery_result_log_plugin` or `osquery_status_log_plugin` are set to `pubsub`.
- `activity_audit_log_plugin` is set to `pubsub` and `activity_enable_audit_log` is set to `true`.

The identifier of the Google Cloud project containing the pubsub topics to
publish logs to.

Note that the pubsub plugin uses [Application Default Credentials (ADCs)](https://cloud.google.com/docs/authentication/production)
for authentication with the service.

- Default value: none
- Environment variable: `FLEET_PUBSUB_PROJECT`
- Config file format:
  ```yaml
  pubsub:
    project: my-gcp-project
  ```

##### pubsub_result_topic

This flag only has effect if `osquery_result_log_plugin` is set to `pubsub`.

The identifier of the pubsub topic that client results will be published to.

- Default value: none
- Environment variable: `FLEET_PUBSUB_RESULT_TOPIC`
- Config file format:
  ```yaml
  pubsub:
    result_topic: osquery_result
  ```

##### pubsub_status_topic

This flag only has effect if `osquery_status_log_plugin` is set to `pubsub`.

The identifier of the pubsub topic that osquery status logs will be published to.

- Default value: none
- Environment variable: `FLEET_PUBSUB_STATUS_TOPIC`
- Config file format:
  ```yaml
  pubsub:
    status_topic: osquery_status
  ```

##### pubsub_audit_topic

This flag only has effect if `osquery_audit_log_plugin` is set to `pubsub`.

The identifier of the pubsub topic that client results will be published to.

- Default value: none
- Environment variable: `FLEET_PUBSUB_AUDIT_TOPIC`
- Config file format:
  ```yaml
  pubsub:
    audit_topic: fleet_audit
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
  ```yaml
  pubsub:
    add_attributes: true
  ```

##### Example YAML

```yaml
osquery:
  osquery_status_log_plugin: pubsub
  osquery_result_log_plugin: pubsub
pubsub:
  project: my-gcp-project
  result_topic: osquery_result
  status_topic: osquery_status
  sts_assume_role_arn: arn:aws:iam::1234567890:role/firehose-role
  status_function: statusFunction
  result_function: resultFunction
```

#### Kafka REST Proxy logging

##### kafkarest_proxyhost

This flag only has effect if one of the following is true:
- `osquery_result_log_plugin` or `osquery_status_log_plugin` are set to `kafkarest`.
- `activity_audit_log_plugin` is set to `kafkarest` and `activity_enable_audit_log` is set to `true`.

The URL of the host which to check for the topic existence and post messages to the specified topic.

- Default value: none
- Environment variable: `FLEET_KAFKAREST_PROXYHOST`
- Config file format:
  ```yaml
  kafkarest:
    proxyhost: "https://localhost:8443"
  ```

##### kafkarest_status_topic

This flag only has effect if `osquery_status_log_plugin` is set to `kafkarest`.

The identifier of the kafka topic that osquery status logs will be published to.

- Default value: none
- Environment variable: `FLEET_KAFKAREST_STATUS_TOPIC`
- Config file format:
  ```yaml
  kafkarest:
    status_topic: osquery_status
  ```

##### kafkarest_result_topic

This flag only has effect if `osquery_result_log_plugin` is set to `kafkarest`.

The identifier of the kafka topic that osquery result logs will be published to.

- Default value: none
- Environment variable: `FLEET_KAFKAREST_RESULT_TOPIC`
- Config file format:
  ```yaml
  kafkarest:
    result_topic: osquery_result
  ```

##### kafkarest_audit_topic

This flag only has effect if `osquery_audit_log_plugin` is set to `kafkarest`.

The identifier of the kafka topic that audit logs will be published to.

- Default value: none
- Environment variable: `FLEET_KAFKAREST_AUDIT_TOPIC`
- Config file format:
  ```yaml
  kafkarest:
    audit_topic: fleet_audit
  ```

##### kafkarest_timeout

This flag only has effect if one of the following is true:
- `osquery_result_log_plugin` or `osquery_status_log_plugin` are set to `kafkarest`.
- `activity_audit_log_plugin` is set to `kafkarest` and `activity_enable_audit_log` is set to `true`.

The timeout value for the http post attempt. Value is in units of seconds.

- Default value: 5
- Environment variable: `FLEET_KAFKAREST_TIMEOUT`
- Config file format:
  ```yaml
  kafkarest:
    timeout: 5
  ```

##### kafkarest_content_type_value

This flag only has effect if one of the following is true:
- `osquery_result_log_plugin` or `osquery_status_log_plugin` are set to `kafkarest`.
- `activity_audit_log_plugin` is set to `kafkarest` and `activity_enable_audit_log` is set to `true`.

The value of the Content-Type header to use in Kafka REST Proxy API calls. More information about available versions
can be found [here](https://docs.confluent.io/platform/current/kafka-rest/api.html#content-types). _Note: only JSON format is supported_

- Default value: application/vnd.kafka.json.v1+json
- Environment variable: `FLEET_KAFKAREST_CONTENT_TYPE_VALUE`
- Config file format:
  ```yaml
  kafkarest:
    content_type_value: application/vnd.kafka.json.v2+json
  ```

##### Example YAML

```yaml
osquery:
  osquery_status_log_plugin: kafkarest
  osquery_result_log_plugin: kafkarest
kafkarest:
  proxyhost: "https://localhost:8443"
  result_topic: osquery_result
  status_topic: osquery_status
```

#### Email backend

By default, the SMTP backend is enabled and no additional configuration is required on the server settings. You can configure
SMTP through the [Fleet console UI](https://fleetdm.com/docs/using-fleet/configuration-files#smtp-settings). However, you can also
configure Fleet to use AWS SES natively rather than through SMTP.

##### backend

Enable SES support for Fleet. You must also configure the ses configurations such as `ses.source_arn`

````yaml
email:
  backend: ses
````

#### SES

The following configurations only have an effect if SES email backend is enabled `FLEET_EMAIL_BACKEND=ses`.

##### ses_region

This flag only has effect if `email.backend` or `FLEET_EMAIL_BACKEND` is set to `ses`.

AWS region to use for SES connection.

- Default value: none
- Environment variable: `FLEET_SES_REGION`
- Config file format:
  ```yaml
  ses:
    region: us-east-2
  ```

##### ses_access_key_id

This flag only has effect if `email.backend` or `FLEET_EMAIL_BACKEND` is set to `ses`.

If `ses_access_key_id` and `ses_secret_access_key` are omitted, Fleet
will try to use
[AWS STS](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html)
credentials.

AWS access key ID to use for Lambda authentication.

- Default value: none
- Environment variable: `FLEET_SES_ACCESS_KEY_ID`
- Config file format:
  ```yaml
  ses:
    access_key_id: AKIAIOSFODNN7EXAMPLE
  ```

##### ses_secret_access_key

This flag only has effect if `email.backend` or `FLEET_EMAIL_BACKEND` is set to `ses`.

If `ses_access_key_id` and `ses_secret_access_key` are omitted, Fleet
will try to use
[AWS STS](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html)
credentials.

AWS secret access key to use for SES authentication.

- Default value: none
- Environment variable: `FLEET_SES_SECRET_ACCESS_KEY`
- Config file format:
  ```yaml
  ses:
    secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  ```

##### ses_sts_assume_role_arn

This flag only has effect if `email.backend` or `FLEET_EMAIL_BACKEND` is set to `ses`.

AWS STS role ARN to use for SES authentication.

- Default value: none
- Environment variable: `FLEET_SES_STS_ASSUME_ROLE_ARN`
- Config file format:
  ```yaml
  ses:
    sts_assume_role_arn: arn:aws:iam::1234567890:role/ses-role
  ```

##### ses_source_arn

This flag only has effect if `email.backend` or `FLEET_EMAIL_BACKEND` is set to `ses`. This configuration **is
required** when using the SES email backend.

The ARN of the identity that is associated with the sending authorization policy that permits you to send
for the email address specified in the Source parameter of SendRawEmail.

- Default value: none
- Environment variable: `FLEET_SES_SOURCE_ARN`
- Config file format:
  ```yaml
  ses:
    sts_assume_role_arn: arn:aws:iam::1234567890:role/ses-role
  ```

#### S3 file carving backend

##### s3_bucket

Name of the S3 bucket to use to store file carves.

- Default value: none
- Environment variable: `FLEET_S3_BUCKET`
- Config file format:
  ```yaml
  s3:
    bucket: some-carve-bucket
  ```

##### s3_prefix

Prefix to prepend to carve objects.

All carve objects will also be prefixed by date and hour (UTC), making the resulting keys look like: `<prefix><year>/<month>/<day>/<hour>/<carve-name>`.

- Default value: none
- Environment variable: `FLEET_S3_PREFIX`
- Config file format:
  ```yaml
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
  ```yaml
  s3:
    access_key_id: AKIAIOSFODNN7EXAMPLE
  ```

##### s3_secret_access_key

AWS secret access key to use for S3 authentication.

- Default value: none
- Environment variable: `FLEET_S3_SECRET_ACCESS_KEY`
- Config file format:
  ```yaml
  s3:
    secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  ```

##### s3_sts_assume_role_arn

AWS STS role ARN to use for S3 authentication.

- Default value: none
- Environment variable: `FLEET_S3_STS_ASSUME_ROLE_ARN`
- Config file format:
  ```yaml
  s3:
    sts_assume_role_arn: arn:aws:iam::1234567890:role/some-s3-role
  ```

##### s3_endpoint_url

AWS S3 Endpoint URL. Override when using a different S3 compatible object storage backend (such as Minio),
or running s3 locally with localstack. Leave this blank to use the default S3 service endpoint.

- Default value: none
- Environment variable: `FLEET_S3_ENDPOINT_URL`
- Config file format:
  ```yaml
  s3:
    endpoint_url: http://localhost:9000
  ```

##### s3_disable_ssl

AWS S3 Disable SSL. Useful for local testing.

- Default value: false
- Environment variable: `FLEET_S3_DISABLE_SSL`
- Config file format:
  ```yaml
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
  ```yaml
  s3:
    force_s3_path_style: false
  ```

##### s3_region

AWS S3 Region. Leave blank to enable region discovery.

Minio users must set this to any nonempty value (eg. `minio`), as Minio does not support region discovery.

- Default value:
- Environment variable: `FLEET_S3_REGION`
- Config file format:
  ```yaml
  s3:
    region: us-east-1
  ```

##### Example YAML

```yaml
s3:
  bucket: some-carve-bucket
  prefix: carves-go-here/
  access_key_id: AKIAIOSFODNN7EXAMPLE
  secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  sts_assume_role_arn: arn:aws:iam::1234567890:role/some-s3-role
  region: us-east-1
```

#### Upgrades

##### allow_missing_migrations

If set then `fleet serve` will run even if there are database migrations missing.

- Default value: `false`
- Environment variable: `FLEET_UPGRADES_ALLOW_MISSING_MIGRATIONS`
- Config file format:
  ```yaml
  upgrades:
    allow_missing_migrations: true
  ```

#### Vulnerabilities

##### databases_path

The path specified needs to exist and Fleet needs to be able to read and write to and from it. This is the only mandatory configuration needed for vulnerability processing to work.

When `current_instance_checks` is set to `auto` (the default), Fleet instances will try to create the `databases_path` if it doesn't exist.

- Default value: `/tmp/vulndbs`
- Environment variable: `FLEET_VULNERABILITIES_DATABASES_PATH`
- Config file format:
  ```yaml
  vulnerabilities:
    databases_path: /some/path
  ```

##### periodicity

How often vulnerabilities are checked. This is also the interval at which the counts of hosts per software is calculated.

- Default value: `1h`
- Environment variable: `FLEET_VULNERABILITIES_PERIODICITY`
- Config file format:
  ```yaml
  vulnerabilities:
    periodicity: 1h
  ```

##### cpe_database_url

You can fetch the CPE dictionary database from this URL. Some users want to control where Fleet gets its database.
When Fleet sees this value defined, it downloads the file directly.
It expects a file in the same format that can be found in https://github.com/fleetdm/nvd/releases.
If this value is not defined, Fleet checks for the latest release in Github and only downloads it if needed.

- Default value: `""`
- Environment variable: `FLEET_VULNERABILITIES_CPE_DATABASE_URL`
- Config file format:
  ```yaml
  vulnerabilities:
    cpe_database_url: ""
  ```

##### cpe_translations_url

You can fetch the CPE translations from this URL.
Translations are used when matching software to CPE entries in the CPE database that would otherwise be missed for various reasons.
When Fleet sees this value defined, it downloads the file directly.
It expects a file in the same format that can be found in https://github.com/fleetdm/nvd/releases.
If this value is not defined, Fleet checks for the latest release in Github and only downloads it if needed.

- Default value: `""`
- Environment variable: `FLEET_VULNERABILITIES_CPE_TRANSLATIONS_URL`
- Config file format:
  ```yaml
  vulnerabilities:
    cpe_translations_url: ""
  ```

##### cve_feed_prefix_url

Like the CPE dictionary, we allow users to define where to get the CVE feeds.
In this case, the URL should be a host that serves the files in the path /feeds/json/cve/1.1/.
Fleet expects to find all the JSON Feeds that can be found in https://nvd.nist.gov/vuln/data-feeds.
When not defined, Fleet downloads from the nvd.nist.gov host.

- Default value: `""`
- Environment variable: `FLEET_VULNERABILITIES_CVE_FEED_PREFIX_URL`
- Config file format:
  ```yaml
  vulnerabilities:
    cve_feed_prefix_url: ""
  ```

##### current_instance_checks

When running multiple instances of the Fleet server, by default, one of them dynamically takes the lead in vulnerability processing. This lead can change over time. Some Fleet users want to be able to define which deployment is doing this checking. If you wish to do this, you'll need to deploy your Fleet instances with this set explicitly to no and one of them set to yes.

- Default value: `auto`
- Environment variable: `FLEET_VULNERABILITIES_CURRENT_INSTANCE_CHECKS`
- Config file format:
  ```yaml
  vulnerabilities:
    current_instance_checks: yes
  ```

##### disable_schedule

To externally manage running vulnerability processing set the value to `true` and then run `fleet vuln_processing` using external
tools like crontab.

- Default value: `false`
- Environment variable: `FLEET_VULNERABILITIES_DISABLE_SCHEDULE`
- Config file format:
  ```yaml
  vulnerabilities:
    disable_schedule: false
  ```

##### disable_data_sync

Fleet by default automatically downloads and keeps the different data streams needed to properly do vulnerability processing. In some setups, this behavior is not wanted, as access to outside resources might be blocked, or the data stream files might need review/audit before use.

In order to support vulnerability processing in such environments, we allow users to disable automatic sync of data streams with this configuration value.

To download the data streams, you can use `fleetctl vulnerability-data-stream --dir ./somedir`. The contents downloaded can then be reviewed, and finally uploaded to the defined `databases_path` in the fleet instance(s) doing the vulnerability processing.

- Default value: false
- Environment variable: `FLEET_VULNERABILITIES_DISABLE_DATA_SYNC`
- Config file format:
  ```yaml
  vulnerabilities:
    disable_data_sync: true
  ```

##### recent_vulnerability_max_age

Maximum age of a vulnerability (a CVE) to be considered "recent". The age is calculated based on the published date of the CVE in the [National Vulnerability Database](https://nvd.nist.gov/) (NVD). Recent vulnerabilities play a special role in Fleet's [automations](https://fleetdm.com/docs/using-fleet/automations), as they are reported when discovered on a host if the vulnerabilities webhook or a vulnerability integration is enabled.

- Default value: `720h` (30 days)
- Environment variable: `FLEET_VULNERABILITIES_RECENT_VULNERABILITY_MAX_AGE`
- Config file format:
  ```yaml
  vulnerabilities:
       recent_vulnerability_max_age: 48h
  ```

##### disable_win_os_vulnerabilities

If using osquery 5.4 or later, Fleet by default will fetch and store all applied Windows updates and use that for detecting Windows
vulnerabilities — which might be a writing-intensive process (depending on the number of Windows hosts
in your Fleet). Setting this to true will cause Fleet to skip both processes.

- Default value: false
- Environment variable: `FLEET_VULNERABILITIES_DISABLE_WIN_OS_VULNERABILITIES`
- Config file format:
  ```yaml
  vulnerabilities:
    disable_win_os_vulnerabilities: true
  ```

##### Example YAML

```yaml
vulnerabilities:
  databases_path: /some/path
  current_instance_checks: yes
  disable_data_sync: true
```

#### GeoIP

##### database_path

The path to a valid Maxmind GeoIP database (mmdb). Support exists for the country & city versions of the database. If city database is supplied
then Fleet will attempt to resolve the location via the city lookup, otherwise it defaults to the country lookup. The IP address used
to determine location is extracted via HTTP headers in the following order: `True-Client-IP`, `X-Real-IP`, and finally `X-FORWARDED-FOR` [headers](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Forwarded-For)
on the Fleet web server.

You can get a copy of the
[Geolite2](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data?lang=en) database for free by
[creating an account](https://www.maxmind.com/en/geolite2/signup?lang=en) on the MaxMind website,
navigating to the [download page](https://www.maxmind.com/en/accounts/current/geoip/downloads),
and downloading the GZIP archive. Decompress it and place the mmdb file somewhere fleet can access.

It is also possible to automatically keep the database up to date, see the
[documentation](https://dev.maxmind.com/geoip/updating-databases?lang=en) from MaxMind.

GeoIP databases can find what general area a device is from, but not the exact location.
They work by collecting which IP addresses ISPs use for different cities and countries and
packaging them up into a list mapping IP address to city.

You've likely seen services use GeoIP databases if they redirect you to a site specific
to your country. e.g. Google will redirect you to [google.ca](https://google.ca) if you visit from Canada
or Mouser will change to your local currency if you view an electronic component.

This can be useful for your fleet install if you want to tell if a device is somewhere it shouldn't
be. If a desktop machine located at a site in New York suddenly appears in London, then you can tell
that something is wrong. It can also help you differentiate machines if they have similar names,
e.g. if you have two computers "John's MacBook Pro".

While it can be a useful tool, an unexpected result could be an error in the database, a user
connecting via a mobile network which uses the same IP address for a wide area, or a user visiting
family. Checking on the location of devices too often could be invasive to employees who are keeping
work devices on them for e.g. oncall responsibilities.

- Default value: none
- Environment variable: `FLEET_GEOIP_DATABASE_PATH`
- Config file format:
  ```yaml
  geoip:
    database_path: /some/path/to/geolite2.mmdb
  ```

#### Sentry

##### DSN

If set, then `Fleet serve` will capture errors and panics and push them to Sentry.

- Default value: `""`
- Environment variable: `FLEET_SENTRY_DSN`
- Config file format:
  ```yaml
  sentry:
    dsn: "https://somedsnprovidedby.sentry.com/"
  ```


#### Prometheus

##### basic_auth.username

This is the username to use for HTTP Basic Auth on the `/metrics` endpoint.

If `basic_auth.username` is not set, then:
  - If `basic_auth.disable` is not set then the Prometheus `/metrics` endpoint is disabled.
  - If `basic_auth.disable` is set then the Prometheus `/metrics` endpoint is enabled but without HTTP Basic Auth.

- Default value: `""`
- Environment variable: `FLEET_PROMETHEUS_BASIC_AUTH_USERNAME`
- Config file format:
  ```yaml
  prometheus:
    basic_auth:
      username: "foo"
  ```

##### basic_auth.password

This is the password to use for HTTP Basic Auth on the `/metrics` endpoint.

If `basic_auth.password` is not set, then:
  - If `basic_auth.disable` is not set then the Prometheus `/metrics` endpoint is disabled.
  - If `basic_auth.disable` is set then the Prometheus `/metrics` endpoint is enabled but without HTTP Basic Auth.

- Default value: `""`
- Environment variable: `FLEET_PROMETHEUS_BASIC_AUTH_PASSWORD`
- Config file format:
  ```yaml
  prometheus:
    basic_auth:
      password: "bar"
  ```

##### basic_auth.disable

This allows running the Prometheus endpoint `/metrics` without HTTP Basic Auth.

If both `basic_auth.username` and `basic_auth.password` are set, then this setting is ignored.

- Default value: false
- Environment variable: `FLEET_PROMETHEUS_BASIC_AUTH_DISABLE`
- Config file format:
  ```yaml
  prometheus:
    basic_auth:
      disable: true
  ```

#### Packaging

These configurations control how Fleet interacts with the
packaging server (coming soon).  These features are currently only intended to be used within
Fleet sandbox, but this is subject to change.

##### packaging_global_enroll_secret

This is the enroll secret for adding hosts to the global scope. If this value is
set, the server won't allow changes to the enroll secret via the config
endpoints.

This value should be treated as a secret. We recommend using a
cryptographically secure pseudo random string. For example, using `openssl`:

```sh
openssl rand -base64 24
```

This config only takes effect if you don't have a global enroll secret already
stored in your database.

- Default value: `""`
- Environment variable: `FLEET_PACKAGING_GLOBAL_ENROLL_SECRET`
- Config file format:
  ```yaml
  packaging:
    global_enroll_secret: "xyz"
  ```

##### packaging_s3_bucket

This is the name of the S3 bucket to store pre-built Fleetd installers.

- Default value: ""
- Environment variable: `FLEET_PACKAGING_S3_BUCKET`
- Config file format:
  ```yaml
  packaging:
    s3:
      bucket: some-bucket
  ```

##### packaging_s3_prefix

This is the prefix to prepend when searching for installers.

- Default value: ""
- Environment variable: `FLEET_PACKAGING_S3_PREFIX`
- Config file format:
  ```yaml
  packaging:
    s3:
      prefix:
        installers-go-here/
  ```

##### packaging_s3_access_key_id

This is the AWS access key ID for S3 authentication.

If `s3_access_key_id` and `s3_secret_access_key` are omitted, Fleet will try to use
[the default credential provider chain](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials).

The IAM identity used in this context must be allowed to perform the following actions on the bucket: `s3:GetObject`, `s3:ListBucket`.

- Default value: ""
- Environment variable: `FLEET_PACKAGING_S3_ACCESS_KEY_ID`
- Config file format:
  ```yaml
  packaging:
    s3:
      access_key_id: AKIAIOSFODNN7EXAMPLE
  ```

##### packaging_s3_secret_access_key

This is the AWS secret access key for S3 authentication.

- Default value: ""
- Environment variable: `FLEET_PACKAGING_S3_SECRET_ACCESS_KEY`
- Config file format:
  ```yaml
  packaging:
    s3:
      secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  ```

##### packaging_s3_sts_assume_role_arn

This is the AWS STS role ARN for S3 authentication.

- Default value: ""
- Environment variable: `FLEET_PACKAGING_S3_STS_ASSUME_ROLE_ARN`
- Config file format:
  ```yaml
  packaging:
    s3:
      sts_assume_role_arn: arn:aws:iam::1234567890:role/some-s3-role
  ```

##### packaging_s3_endpoint_url

This is the AWS S3 Endpoint URL. Override when using a different S3 compatible object storage backend (such as Minio)
or running S3 locally with LocalStack. Leave this blank to use the default AWS S3 service endpoint.

- Default value: ""
- Environment variable: `FLEET_PACKAGING_S3_ENDPOINT_URL`
- Config file format:
  ```yaml
  packaging:
    s3:
      endpoint_url: http://localhost:9000
  ```

##### packaging_s3_disable_ssl

This is the AWS S3 Disable SSL. It's useful for local testing.

- Default value: false
- Environment variable: `FLEET_PACKAGING_S3_DISABLE_SSL`
- Config file format:
  ```yaml
  packaging:
    s3:
      disable_ssl: false
  ```

##### packaging_s3_force_s3_path_style

This is the AWS S3 Force S3 Path Style. Set this to `true` to force the request to use path-style addressing
(e.g., `http://s3.amazonaws.com/BUCKET/KEY`). By default, the S3 client
will use virtual hosted bucket addressing when possible
(`http://BUCKET.s3.amazonaws.com/KEY`).

See the [Virtual hosting of buckets doc](http://docs.aws.amazon.com/AmazonS3/latest/dev/VirtualHosting.html) for details.

- Default value: false
- Environment variable: `FLEET_PACKAGING_S3_FORCE_S3_PATH_STYLE`
- Config file format:
  ```yaml
  packaging:
    s3:
      force_s3_path_style: false
  ```

##### packaging_s3_region

This is the AWS S3 Region. Leave it blank to enable region discovery.

Minio users must set this to any non-empty value (e.g., `minio`), as Minio does not support region discovery.

- Default value: ""
- Environment variable: `FLEET_PACKAGING_S3_REGION`
- Config file format:
  ```yaml
  packaging:
    s3:
      region: us-east-1
  ```

##### Example YAML

```yaml
packaging:
  s3:
    bucket: some-bucket
    prefix: installers-go-here/
    access_key_id: AKIAIOSFODNN7EXAMPLE
    secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
    sts_assume_role_arn: arn:aws:iam::1234567890:role/some-s3-role
    region: us-east-1
```

## Mobile device management (MDM)

> MDM features require some endpoints to be publicly accessible outside your VPN or intranet, for more details see [What API endpoints should I expose to the public internet?](https://fleetdm.com/docs/get-started/faq#what-api-endpoints-should-i-expose-to-the-public-internet)

This section is a reference for the configuration required to turn on MDM features in production.

If you're a Fleet contributor and you'd like to turn on MDM features in a local environment, see the guided instructions [here](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Testing-and-local-development.md#mdm-setup-and-testing).

##### mdm.apple_apns_cert_bytes

The content of the Apple Push Notification service (APNs) certificate. An X.509 certificate, PEM-encoded. Typically generated via `fleetctl generate mdm-apple`.

- Default value: ""
- Environment variable: `FLEET_MDM_APPLE_APNS_CERT_BYTES`
- Config file format:
  ```yaml
  mdm:
    apple_apns_cert_bytes: |
      -----BEGIN CERTIFICATE-----
      ... PEM-encoded content ...
      -----END CERTIFICATE-----
  ```

##### mdm.apple_apns_key_bytes

The content of the PEM-encoded private key for the Apple Push Notification service (APNs). Typically generated via `fleetctl generate mdm-apple`.

- Default value: ""
- Environment variable: `FLEET_MDM_APPLE_APNS_KEY_BYTES`
- Config file format:
  ```yaml
  mdm:
    apple_apns_key_bytes: |
      -----BEGIN RSA PRIVATE KEY-----
      ... PEM-encoded content ...
      -----END RSA PRIVATE KEY-----
  ```

##### mdm.apple_scep_cert_bytes

The content of the Simple Certificate Enrollment Protocol (SCEP) certificate. An X.509 certificate, PEM-encoded. Typically generated via `fleetctl generate mdm-apple`.

- Default value: ""
- Environment variable: `FLEET_MDM_APPLE_SCEP_CERT_BYTES`
- Config file format:
  ```yaml
  mdm:
    apple_scep_cert_bytes: |
      -----BEGIN CERTIFICATE-----
      ... PEM-encoded content ...
      -----END CERTIFICATE-----
  ```

##### mdm.apple_scep_key_bytes

The content of the PEM-encoded private key for the Simple Certificate Enrollment Protocol (SCEP). Typically generated via `fleetctl generate mdm-apple`.

- Default value: ""
- Environment variable: `FLEET_MDM_APPLE_SCEP_KEY_BYTES`
- Config file format:
  ```yaml
  mdm:
    apple_scep_key_bytes: |
      -----BEGIN RSA PRIVATE KEY-----
      ... PEM-encoded content ...
      -----END RSA PRIVATE KEY-----
  ```

##### mdm.apple_scep_challenge

An alphanumeric secret for the Simple Certificate Enrollment Protocol (SCEP). Define a unique, static secret 32 characters in length and only include alphanumeric characters.

> SCEP is commonly applied to a number of certificate use cases. Notably, Mobile Device Management (MDM) systems like Microsoft Intune and Apple MDM use SCEP for PKI certificate enrollment.

- Default value: ""
- Environment variable: `FLEET_MDM_APPLE_SCEP_CHALLENGE`
- Config file format:
  ```yaml
  mdm:
    apple_scep_challenge: scepchallenge
  ```

##### mdm.apple_scep_signer_validity_days

The number of days the signed SCEP client certificates will be valid.

- Default value: 365
- Environment variable: `FLEET_MDM_APPLE_SCEP_SIGNER_VALIDITY_DAYS`
- Config file format:
  ```yaml
  mdm:
    apple_scep_signer_validity_days: 100
  ```

##### mdm.apple_scep_signer_allow_renewal_days

The number of days allowed to renew SCEP certificates.

- Default value: 14
- Environment variable: `FLEET_MDM_APPLE_SCEP_SIGNER_ALLOW_RENEWAL_DAYS`
- Config file format:
  ```yaml
  mdm:
    apple_scep_signer_allow_renewal_days: 30
  ```

##### mdm.apple_bm_server_token_bytes

This is the content of the Apple Business Manager encrypted server token downloaded from Apple Business Manager.

- Default value: ""
- Environment variable: `FLEET_MDM_APPLE_BM_SERVER_TOKEN_BYTES`
- Config file format:
  ```yaml
  mdm:
    apple_bm_server_token_bytes: |
      Content-Type: application/pkcs7-mime; name="smime.p7m"; smime-type=enveloped-data
      Content-Transfer-Encoding: base64
      ... rest of content ...
  ```

##### mdm.apple_bm_cert_bytes

This is the content of the Apple Business Manager certificate. The certificate is a PEM-encoded X.509 certificate that's typically generated via `fleetctl generate mdm-apple-bm`.

- Default value: ""
- Environment variable: `FLEET_MDM_APPLE_BM_CERT_BYTES`
- Config file format:
  ```yaml
  mdm:
    apple_bm_cert_bytes: |
      -----BEGIN CERTIFICATE-----
      ... PEM-encoded content ...
      -----END CERTIFICATE-----
  ```

##### mdm.apple_bm_key_bytes

This is the content of the PEM-encoded private key for the Apple Business Manager. It's typically generated via `fleetctl generate mdm-apple-bm`.

- Default value: ""
- Environment variable: `FLEET_MDM_APPLE_BM_KEY_BYTES`
- Config file format:
  ```yaml
  mdm:
    apple_bm_key_bytes: |
      -----BEGIN RSA PRIVATE KEY-----
      ... PEM-encoded content ...
      -----END RSA PRIVATE KEY-----
  ```

##### mdm.apple_dep_sync_periodicity

The duration between DEP device syncing (fetching and setting of DEP profiles). Only relevant if Apple Business Manager (ABM) is configured.

- Default value: 1m
- Environment variable: `FLEET_MDM_APPLE_DEP_SYNC_PERIODICITY`
- Config file format:
  ```yaml
  mdm:
    apple_dep_sync_periodicity: 10m
  ```
##### mdm.windows_wstep_identity_cert_bytes
> Windows MDM features are not ready for production and are currently in development. These features are disabled by default.

The content of the Windows WSTEP identity certificate. An X.509 certificate, PEM-encoded.
- Default value: ""
- Environment variable: `FLEET_MDM_WINDOWS_WSTEP_IDENTITY_CERT_BYTES`
- Config file format: 
  ```
  mdm:
   windows_wstep_identity_cert_bytes: |
      -----BEGIN CERTIFICATE-----
      ... PEM-encoded content ...
      -----END CERTIFICATE-----
  ```



##### mdm.windows_wstep_identity_key_bytes
> Windows MDM features are not ready for production and are currently in development. These features are disabled by default.

The content of the Windows WSTEP identity key. An RSA private key, PEM-encoded.
- Default value: ""
- Environment variable: `FLEET_MDM_WINDOWS_WSTEP_IDENTITY_KEY_BYTES` 
- Config file format:  
  ```
  mdm:
    windows_wstep_identity_key_bytes: |
      -----BEGIN RSA PRIVATE KEY-----
      ... PEM-encoded content ...
      -----END RSA PRIVATE KEY-----
  ```


## Managing osquery configurations

We recommend that you use an infrastructure configuration management tool to manage these osquery configurations consistently across your environment. If you're unsure about what configuration management tools your organization uses, contact your company's system administrators. If you are evaluating new solutions for this problem, the founders of Fleet have successfully managed configurations in large production environments using [Chef](https://www.chef.io/chef/) and [Puppet](https://puppet.com/).

<h2 id="running-with-systemd">Running with systemd</h2>

This content was moved to [Systemd](http://fleetdm.com/docs/deploy/system-d) on Sept 6th, 2023.

<h2 id="using-a-proxy">Using a proxy</h2>

This content was moved to [Proxies](http://fleetdm.com/docs/deploy/proxies) on Sept 6th, 2023.

<h2 id="configuring-single-sign-on-sso">Configuring single sign-on (SSO)</h2>

This content was moved to [Single sign-on (SSO)](http://fleetdm.com/deploy/single-sign-on-sso) on Sept 6th, 2023.

<h2 id="public-ips-of-devices">Public IPs of devices</h2>

This content was moved to [Public IPs](http://fleetdm.com/docs/deploy/public-ips) on Sept 6th, 2023.


<meta name="pageOrderInSection" value="100">
<meta name="description" value="This page includes resources for configuring the Fleet binary, managing osquery configurations, and running with systemd.">
<meta name="navSection" value="Fleet server">
