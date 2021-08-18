# Configuration

- [Configuring the Fleet binary](#configuring-the-fleet-binary)
  - [High-level configuration overview](#high-level-configuration-overview)
  - [Commands](#commands)
  - [Options](#options)
- [Managing osquery configurations](#managing-osquery-configurations)
- [Running with systemd](#running-with-systemd)
- [Configuring Single Sign On](#configuring-single-sign-on)
  - [Identity Provider (IDP) Configuration](#identity-provider-IDP-configuration)
  - [Fleet SSO Configuration](#fleet-sso-configuration)
  - [Creating SSO Users in Fleet](#creating-sso-users-in-fleet)
- [Feature flags](#feature-flags)

## Configuring the Fleet binary

For information on how to run the `fleet` binary, detailed usage information can be found by running `fleet --help`. This document is a more detailed version of the information presented in the help output text. If you prefer to use a CLI instead of a web browser, we hope that you like the binary interface to the Fleet application!

### High-level configuration overview

To get the most out of running the Fleet server, it is helpful to establish a mutual understanding of what the desired architecture looks like and what it's trying to accomplish.

Your Fleet server's two main purposes are:

- To serve as your [osquery TLS server](https://osquery.readthedocs.io/en/stable/deployment/remote/)
- To serve the Fleet web UI, which allows you to manage osquery configuration, query hosts, etc.

The Fleet server allows you persist configuration, manage users, etc. Thus, it needs a database. Fleet uses MySQL and requires you to supply configurations to connect to a MySQL server. Fleet also uses Redis to perform some more high-speed data access action throughout the lifecycle of the application (for example, distributed query result ingestion). Thus, Fleet also requires that you supply Redis connection configurations.

> Fleet does not support Redis Cluster or Redis Sentinel. Fleet can scale to hundreds of thousands of devices with a single Redis instance.

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

#### What are the options?

Note that all option names can be converted consistently from flag name to environment variable and visa-versa. For example, the `--mysql_address` flag would be the `FLEET_MYSQL_ADDRESS`. Further, specifying the `mysql_address` option in the config would follow the pattern:

```
mysql:
  address: 127.0.0.1:3306
```

Basically, just capitalize the option and prepend `FLEET_` to it in order to get the environment variable. The conversion works the same the opposite way.

All duration-based settings accept valid time units of `s`, `m`, `h`.

##### MySQL

###### `mysql_address`

The address of the MySQL server which Fleet should connect to. Include the hostname and port.

- Default value: `localhost:3306`
- Environment variable: `FLEET_MYSQL_ADDRESS`
- Config file format:

  ```
  mysql:
  	address: localhost:3306
  ```

###### `mysql_database`

The name of the MySQL database which Fleet will use.

- Default value: `fleet`
- Environment variable: `FLEET_MYSQL_DATABASE`
- Config file format:

  ```
  mysql:
  	database: fleet
  ```

###### `mysql_username`

The username to use when connecting to the MySQL instance.

- Default value: `fleet`
- Environment variable: `FLEET_MYSQL_USERNAME`
- Config file format:

  ```
  mysql:
  	username: fleet
  ```

###### `mysql_password`

The password to use when connecting to the MySQL instance.

- Default value: `fleet`
- Environment variable: `FLEET_MYSQL_PASSWORD`
- Config file format:

  ```
  mysql:
  	password: fleet
  ```

##### `mysql_password_path`

File path to a file that contains the password to use when connecting to the MySQL instance.

- Default value: `""`
- Environment variable: `FLEET_MYSQL_PASSWORD_PATH`
- Config file format:

  ```
  mysql:
  	password_path: '/run/secrets/fleetdm-mysql-password'
  ```

##### `mysql_tls_ca`

The path to a PEM encoded certificate of MYSQL's CA for client certificate authentication.

- Default value: none
- Environment variable: `FLEET_MYSQL_TLS_CA`
- Config file format:

  ```
  mysql:
  	tls_ca: /path/to/server-ca.pem
  ```

###### `mysql_tls_cert`

The path to a PEM encoded certificate use for tls authentication.

- Default value: none
- Environment variable: `FLEET_MYSQL_TLS_CERT`
- Config file format:

  ```
  mysql:
  	tls_cert: /path/to/certificate.pem
  ```

###### `mysql_tls_key`

The path to a PEM encoded private key use for tls authentication.

- Default value: none
- Environment variable: `FLEET_MYSQL_TLS_KEY`
- Config file format:

  ```
  mysql:
  	tls_key: /path/to/key.pem
  ```

###### `mysql_tls_config`

The tls value in a MYSQL DSN. Can be `true`,`false`,`skip-verify` or the CN value of the certificate.

- Default value: none
- Environment variable: `FLEET_MYSQL_TLS_CONFIG`
- Config file format:

  ```
  mysql:
  	tls_config: true
  ```

###### `mysql_tls_server_name`

The server name or IP address used by the client certificate.

- Default value: none
- Environment variable: `FLEET_MYSQL_TLS_SERVER_NAME`
- Config file format:

  ```
  mysql:
  	servername: 127.0.0.1
  ```

###### `mysql_max_open_conns`

Maximum open connections to database

- Default value: 50
- Environment variable: `FLEET_MYSQL_MAX_OPEN_CONNS`
- Config file format:

  ```
  mysql:
  	max_open_conns: 50
  ```

###### `mysql_max_idle_conns`

Maximum idle connections to database. This value should be equal to or less than `mysql_max_open_conns`

- Default value: 50
- Environment variable: `FLEET_MYSQL_MAX_IDLE_CONNS`
- Config file format:

  ```
  mysql:
  	max_idle_conns: 50
  ```

###### `conn_max_lifetime`

Maximum amount of time, in seconds, a connection may be reused.

- Default value: 0 (Unlimited)
- Environment variable: `FLEET_MYSQL_CONN_MAX_LIFETIME`
- Config file format:

  ```
  mysql:
  	conn_max_lifetime: 50
  ```

##### Redis

###### `redis_address`

The address of the Redis server which Fleet should connect to. Include the hostname and port.

- Default value: `localhost:6379`
- Environment variable: `FLEET_REDIS_ADDRESS`
- Config file format:

  ```
  redis:
  	address: 127.0.0.1:7369
  ```

###### `redis_password`

The password to use when connecting to the Redis instance.

- Default value: `<empty>`
- Environment variable: `FLEET_REDIS_PASSWORD`
- Config file format:

  ```
  redis:
  	password: foobar
  ```

###### `redis_database`

The database to use when connecting to the Redis instance.

- Default value: `0`
- Environment variable: `FLEET_REDIS_DATABASE`
- Config file format:

  ```
  redis:
    database: 14
  ```

###### `redis_duplicate_results`

Whether or not to duplicate Live Query results to another Redis channel named `LQDuplicate`. This is useful in a scenario that would involve shipping the Live Query results outside of Fleet, near-realtime.

- Default value: `false`
- Environment variable: `FLEET_REDIS_DUPLICATE_RESULTS`
- Config file format:

  ```
  redis:
    duplicate_results: true
  ```

##### Server

###### `server_address`

The address to serve the Fleet webserver.

- Default value: `0.0.0.0:8080`
- Environment variable: `FLEET_SERVER_ADDRESS`
- Config file format:

  ```
  server:
  	address: 0.0.0.0:443
  ```

###### `server_cert`

The TLS cert to use when terminating TLS.

See [TLS certificate considerations](./1-Installation.md#tls-certificate-considerations) for more information about certificates and Fleet.

- Default value: `./tools/osquery/fleet.crt`
- Environment variable: `FLEET_SERVER_CERT`
- Config file format:

  ```
  server:
  	cert: /tmp/fleet.crt
  ```

###### `server_key`

The TLS key to use when terminating TLS.

- Default value: `./tools/osquery/fleet.key`
- Environment variable: `FLEET_SERVER_KEY`
- Config file format:

  ```
  server:
  	key: /tmp/fleet.key
  ```

###### `server_tls`

Whether or not the server should be served over TLS.

- Default value: `true`
- Environment variable: `FLEET_SERVER_TLS`
- Config file format:

  ```
  server:
  	tls: false
  ```

###### `server_tls_compatibility`

Configures the TLS settings for compatibility with various user agents. Options are `modern` and `intermediate`. These correspond to the compatibility levels [defined by the Mozilla OpSec team](https://wiki.mozilla.org/index.php?title=Security/Server_Side_TLS&oldid=1229478) (updated July 24, 2020).

- Default value: `intermediate`
- Environment variable: `FLEET_SERVER_TLS_COMPATIBILITY`
- Config file format:

  ```
  server:
  	tls_compatibility: intermediate
  ```

###### `server_url_prefix`

Sets a URL prefix to use when serving the Fleet API and frontend. Prefixes should be in the form `/apps/fleet` (no trailing slash).

Note that some other configurations may need to be changed when modifying the URL prefix. In particular, URLs that are provided to osquery via flagfile, the configuration served by Fleet, the URL prefix used by `fleetctl`, and the redirect URL set with an SSO Identity Provider.

- Default value: Empty (no prefix set)
- Environment variable: `FLEET_SERVER_URL_PREFIX`
- Config file format:

  ```
  server:
  	url_prefix: /apps/fleet
  ```

###### `server_keepalive`

Controls the server side http keep alive property.

Turning off keepalives has helped reduce outstanding TCP connections in some deployments.

- Default value: true
- Environment variable: `FLEET_SERVER_KEEPALIVE`
- Config file format:

  ```
  server:
  	keepalive: true
  ```

##### Auth

##### `auth_bcrypt_cost`

The bcrypt cost to use when hashing user passwords.

- Default value: `12`
- Environment variable: `FLEET_AUTH_BCRYPT_COST`
- Config file format:

  ```
  auth:
  	bcrypt_cost: 14
  ```

###### `auth_salt_key_size`

The key size of the salt which is generated when hashing user passwords.

- Default value: `24`
- Environment variable: `FLEET_AUTH_SALT_KEY_SIZE`
- Config file format:

  ```
  auth:
  	salt_key_size: 36
  ```

##### App

###### `app_token_key_size`

Size of generated app tokens.

- Default value: `24`
- Environment variable: `FLEET_APP_TOKEN_KEY_SIZE`
- Config file format:

  ```
  app:
  	token_key_size: 36
  ```

###### `app_invite_token_validity_period`

How long invite tokens should be valid for.

- Default value: `5 days`
- Environment variable: `FLEET_APP_INVITE_TOKEN_VALIDITY_PERIOD`
- Config file format:

  ```
  app:
  	invite_token_validity_period: 1d
  ```

##### License

###### `license_key`

The license key provided to Fleet customers which provides access to Fleet Basic features.

- Default value: none
- Environment variable: `FLEET_LICENSE_KEY`
- Config file format:

  ```
  license:
    key: foobar
  ```

##### Session

###### `session_key_size`

The size of the session key.

- Default value: `64`
- Environment variable: `FLEET_SESSION_KEY_SIZE`
- Config file format:

  ```
  session:
  	key_size: 48
  ```

###### `session_duration`

The amount of time that a session should last for.

Valid time units are `s`, `m`, `h`.

- Default value: `4 hours`
- Environment variable: `FLEET_SESSION_DURATION`
- Config file format:

  ```
  session:
  	duration: 24h
  ```

##### Osquery

###### `osquery_node_key_size`

The size of the node key which is negotiated with `osqueryd` clients.

- Default value: `24`
- Environment variable: `FLEET_OSQUERY_NODE_KEY_SIZE`
- Config file format:

  ```
  osquery:
  	node_key_size: 36
  ```

###### `osquery_host_identifier`

The identifier to use when determining uniqueness of hosts.

Options are `provided` (default), `uuid`, `hostname`, or `instance`.

This setting works in combination with the `--host_identifier` flag in osquery. In most deployments, using `instance` will be the best option. The flag defaults to `provided` -- preserving the existing behavior of Fleet's handling of host identifiers -- using the identifier provided by osquery. `instance`, `uuid`, and `hostname` correspond to the same meanings as for osquery's `--host_identifier` flag.

Users that have duplicate UUIDs in their environment can benefit from setting this flag to `instance`.

- Default value: `provided`
- Environment variable: `FLEET_OSQUERY_HOST_IDENTIFIER`
- Config file format:

  ```
  osquery:
  	host_identifier: uuid
  ```

###### `osquery_enroll_cooldown`

The cooldown period for host enrollment. If a host (uniquely identified by the `osquery_host_identifier` option) tries to enroll within this duration from the last enrollment, enroll will fail.

This flag can be used to control load on the database in scenarios in which many hosts are using the same identifier. Often configuring `osquery_host_identifier` to `instance` may be a better solution.

- Default value: `0` (off)
- Environment variable: `FLEET_OSQUERY_ENROLL_COOLDOWN`
- Config file format:

  ```
  osquery:
  	enroll_cooldown: 1m
  ```

###### `osquery_label_update_interval`

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

###### `osquery_detail_update_interval`

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

###### `osquery_status_log_plugin`

Which log output plugin should be used for osquery status logs received from clients.

Options are `filesystem`, `firehose`, `kinesis`, `lambda`, `pubsub`, and `stdout`.

- Default value: `filesystem`
- Environment variable: `FLEET_OSQUERY_STATUS_LOG_PLUGIN`
- Config file format:

  ```
  osquery:
  	status_log_plugin: firehose
  ```

###### `osquery_result_log_plugin`

Which log output plugin should be used for osquery result logs received from clients.

Options are `filesystem`, `firehose`, `kinesis`, `lambda`, `pubsub`, and `stdout`.

- Default value: `filesystem`
- Environment variable: `FLEET_OSQUERY_RESULT_LOG_PLUGIN`
- Config file format:

  ```
  osquery:
  	result_log_plugin: firehose
  ```

##### Logging (Fleet server logging)

###### `logging_debug`

Whether or not to enable debug logging.

- Default value: `false`
- Environment variable: `FLEET_LOGGING_DEBUG`
- Config file format:

  ```
  logging:
  	debug: true
  ```

###### `logging_json`

Whether or not to log in JSON.

- Default value: `false`
- Environment variable: `FLEET_LOGGING_JSON`
- Config file format:

  ```
  logging:
  	json: true
  ```

###### `logging_disable_banner`

Whether or not to log the welcome banner.

- Default value: `false`
- Environment variable: `FLEET_LOGGING_DISABLE_BANNER`
- Config file format:

  ```
  logging:
  	disable_banner: true
  ```

##### Filesystem

###### `filesystem_status_log_file`

This flag only has effect if `osquery_status_log_plugin` is set to `filesystem` (the default value).

The path which osquery status logs will be logged to.

- Default value: `/tmp/osquery_status`
- Environment variable: `FLEET_FILESYSTEM_STATUS_LOG_FILE`
- Config file format:

  ```
  filesystem:
  	status_log_file: /var/log/osquery/status.log
  ```

###### `filesystem_result_log_file`

This flag only has effect if `osquery_result_log_plugin` is set to `filesystem` (the default value).

The path which osquery result logs will be logged to.

- Default value: `/tmp/osquery_result`
- Environment variable: `FLEET_FILESYSTEM_RESULT_LOG_FILE`
- Config file format:

  ```
  filesystem:
  	result_log_file: /var/log/osquery/result.log
  ```

###### `filesystem_enable_log_rotation`

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

###### `filesystem_enable_log_compression`

This flag only has effect if `filesystem_enable_log_rotation` is set to `true`.

This flag will cause the rotated logs to be compressed with gzip.

- Default value: `false`
- Environment variable: `FLEET_FILESYSTEM_ENABLE_LOG_COMPRESSION`
- Config file format:

  ```
  filesystem:
     enable_log_compression: true
  ```

##### Firehose

###### `firehose_region`

This flag only has effect if `osquery_status_log_plugin` is set to `firehose`.

AWS region to use for Firehose connection

- Default value: none
- Environment variable: `FLEET_FIREHOSE_REGION`
- Config file format:

  ```
  firehose:
  	region: ca-central-1
  ```

###### `firehose_access_key_id`

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

###### `firehose_secret_access_key`

This flag only has effect if `osquery_status_log_plugin` or `osquery_result_log_plugin` are set to `firehose`.

AWS secret access key to use for Firehose authentication.

- Default value: none
- Environment variable: `FLEET_FIREHOSE_SECRET_ACCESS_KEY`
- Config file format:

  ```
  firehose:
  	secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  ```

###### `firehose_sts_assume_role_arn`

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

###### `firehose_status_stream`

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

###### `firehose_result_stream`

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

##### Kinesis

###### `kinesis_region`

This flag only has effect if `osquery_status_log_plugin` is set to `kinesis`.

AWS region to use for Kinesis connection

- Default value: none
- Environment variable: `FLEET_KINESIS_REGION`
- Config file format:

  ```
  kinesis:
  	region: ca-central-1
  ```

###### `kinesis_access_key_id`

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

###### `kinesis_secret_access_key`

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

###### `kinesis_sts_assume_role_arn`

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

###### `kinesis_status_stream`

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

###### `kinesis_result_stream`

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

##### Lambda

###### `lambda_region`

This flag only has effect if `osquery_status_log_plugin` is set to `lambda`.

AWS region to use for Lambda connection

- Default value: none
- Environment variable: `FLEET_LAMBDA_REGION`
- Config file format:

  ```
  lambda:
  	region: ca-central-1
  ```

###### `lambda_access_key_id`

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

###### `lambda_secret_access_key`

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

###### `lambda_sts_assume_role_arn`

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

###### `lambda_status_function`

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

###### `lambda_result_function`

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

##### PubSub

###### `pubsub_project`

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

###### `pubsub_result_topic`

This flag only has effect if `osquery_status_log_plugin` is set to `pubsub`.

The identifier of the pubsub topic that client results will be published to.

- Default value: none
- Environment variable: `FLEET_PUBSUB_RESULT_TOPIC`
- Config file format:

  ```
  pubsub:
    result_topic: osquery_result
  ```

###### `pubsub_status_topic`

This flag only has effect if `osquery_status_log_plugin` is set to `pubsub`.

The identifier of the pubsub topic that osquery status logs will be published to.

- Default value: none
- Environment variable: `FLEET_PUBSUB_STATUS_TOPIC`
- Config file format:

  ```
  pubsub:
    status_topic: osquery_status
  ```

###### `pubsub_add_attributes`

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

##### S3 file carving backend

###### `s3_bucket`

Name of the S3 bucket to use to store file carves.

- Default value: none
- Environment variable: `FLEET_S3_BUCKET`
- Config file format:

  ```
  s3:
  	bucket: some-carve-bucket
  ```

###### `s3_prefix`

Prefix to prepend to carve objects.

All carve objects will also be prefixed by date and hour (UTC), making the resulting keys look like: `<prefix><year>/<month>/<day>/<hour>/<carve-name>`.

- Default value: none
- Environment variable: `FLEET_S3_PREFIX`
- Config file format:

  ```
  s3:
  	prefix: carves-go-here/
  ```

###### `s3_access_key_id`

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

###### `s3_secret_access_key`

AWS secret access key to use for S3 authentication.

- Default value: none
- Environment variable: `FLEET_S3_SECRET_ACCESS_KEY`
- Config file format:

  ```
  s3:
  	secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  ```

###### `s3_sts_assume_role_arn`

AWS STS role ARN to use for S3 authentication.

- Default value: none
- Environment variable: `FLEET_S3_STS_ASSUME_ROLE_ARN`
- Config file format:

  ```
  s3:
  	sts_assume_role_arn: arn:aws:iam::1234567890:role/some-s3-role
  ```

##### Vulnerabilities

###### `databases_path`

The path specified needs to exist and fleet needs to be able to read and write to and from it. This is the only mandatory configuration needed for vulnerability processing to work.

- Default value: none
- Environment variable: `FLEET_VULNERABILITIES_DATABASES_PATH`
- Config file format:

  ```
  vulnerabilities:
  	databases_path: /some/path
  ```

###### `periodicity`

How often vulnerabilities are checked.

- Default value: `1hr`
- Environment variable: `FLEET_VULNERABILITIES_PERIODICITY`
- Config file format:

  ```
  vulnerabilities:
  	periodicity: 1hr
  ```

###### `cpe_database_url`

URL to fetch the CPE dictionary database from. Some users want to control where fleet gets its database from. When Fleet sees this value defined, it downloads the file directly. It expects a file in the same format as can be found in https://github.com/fleetdm/nvd/releases. If this value is not defined, Fleet checks for the latest release in Github and only downloads it if needed.

- Default value: `""`
- Environment variable: `FLEET_VULNERABILITIES_CPE_DATABASE_URL`
- Config file format:

  ```
  vulnerabilities:
  	cpe_database_url: ""
  ```

###### `cve_feed_prefix_url`

Similarly to the CPE dictionary, we allow users to define where to get the CVE feeds from. In this case, the url should be a host that serves the files in the path /feeds/json/cve/1.1/. Fleet expects to find there all the JSON Feeds that can be found in https://nvd.nist.gov/vuln/data-feeds. When not defined, Fleet downloads from the nvd.nist.gov host.

- Default value: `""`
- Environment variable: `FLEET_VULNERABILITIES_CVE_FEED_PREFIX_URL`
- Config file format:

  ```
  vulnerabilities:
  	cve_database_url: ""
  ```

###### `current_instance_checks`

When running multiple instances of the Fleet server, by default, one of them dynamically takes the lead in vulnerability processing. This lead can change over time. Some Fleet users want to be able to define which deployment is doing this checking. If you wish to do this, you'll need to deploy your Fleet instances with this set explicitly to no and one of them set to yes.

- Default value: `auto`
- Environment variable: `FLEET_VULNERABILITIES_CURRENT_INSTANCE_CHECKS`
- Config file format:

  ```
  vulnerabilities:
  	current_instance_checks: yes
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

## Configuring Single Sign On

Fleet supports SAML single sign on capability.

Fleet supports both SP-initiated SAML login and IDP-initiated login, however IDP-initiated login must be enabled in the web interface's SAML Single Sign On Options.

Fleet supports the SAML Web Browser SSO Profile using the HTTP Redirect Binding.

### Identity Provider (IDP) Configuration

Setting up the connected application (Fleet) with an identity provider generally requires the following information:

- _Assertion Consumer Service_ - This is the call back URL that the identity provider
  will use to send security assertions to Fleet. In Okta, this field is called _Single sign on URL_. The value that you supply will be a fully qualified URL
  consisting of your Fleet web address and the callback path `/api/v1/fleet/sso/callback`. For example,
  if your Fleet web address is https://fleet.acme.org, then the value you would
  use in the identity provider configuration would be:

  ```
  https://fleet.acme.org/api/v1/fleet/sso/callback
  ```

- _Entity ID_ - This value is a URI that you define. It identifies your Fleet instance as the service provider that issues authorization requests. The value must exactly match the Entity ID that you define in the Fleet SSO configuration.

- _Name ID Format_ - The value should be `urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress`. This may be shortened in the IDP setup to something like `email` or `EmailAddress`.

- _Subject Type (Application username in Okta)_ - `email`.

After supplying the above information, the IDP will generate an issuer URI and a metadata URL that will be used to configure Fleet as a service provider.

  #### Example Okta IDP Configuration

  ![Example Okta IDP Configuration](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/okta-idp-setup.png)

> The names of the items required to configure an Identity Provider may vary from provider to provider and may not conform to the SAML spec. 

> Individual users must also be setup on the IDP before they can sign in to Fleet. 

### Fleet SSO Configuration

A Fleet user must be assigned the Admin role to configure Fleet for SSO. In Fleet, SSO configuration settings are located in **Settings > Organization settings > SAML Single Sign On Options**. 

If your IDP supports dynamic configuration, like Okta, you only need to provide an _Identity Provider Name_ and _Entity ID_, then paste a link in the metadata URL field. 

Otherwise, the following values are required:

- _Identity Provider Name_ - A human friendly name of the IDP.

- _Entity ID_ - A URI that identifies your Fleet instance as the issuer of authorization
  requests. Assuming your company name is Acme, an example might be `fleet.acme.org` although
  the value could be anything as long as it is unique to Fleet as a service provider
  and matches the entity provider value used in the IDP configuration.

- _Issuer URI_ - This value is obtained from the IDP.

- _Metadata URL_ - This value is obtained from the IDP and is used by Fleet to
  issue authorization requests to the IDP.

- _Metadata_ - If the IDP does not provide a metadata URL, the metadata must
  be obtained from the IDP and entered. Note that the metadata URL is preferred if
  the IDP provides metadata in both forms.

#### Example Fleet SSO Configuration

![Example SSO Configuration](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/sso-setup.png)

### Creating SSO users in Fleet

When an admin creates a new user to Fleet, they may select the `Enable Single Sign On` option. The
SSO enabled users will not be able to sign in with a regular user ID and password. 

It is strongly recommended that at least one admin user is set up to use the traditional password
based log in so that there is a fallback method for logging into Fleet in the event of SSO
configuration problems.

## Feature flags

Fleet features are sometimes gated behind feature flags. This will usually be due to not-yet-stable APIs, or not-fully-tested performance characteristics.

Feature flags on the server are controlled by environment variables prefixed with `FLEET_BETA_`.

#### Software inventory

Enable by setting the environment variable `FLEET_BETA_SOFTWARE_INVENTORY=1`.

When enabled, Fleet will store a "software inventory" for hosts, updated along with the other host vitals. Note that it will take some time for the data to be available after setting this flag (it will be updated when the host details are next updated, configurable by [--osquery_detail_update_interval](#osquery_detail_update_interval)).

This is currently feature flagged because we would like to evaluate the performance characteristics on larger deployments.

To read more about the software inventory feature, [check out the Fleet 3.11.0 release blog post](https://medium.com/fleetdm/fleet-3-11-0-released-with-software-inventory-25d5a1efe19c).
