Configuring The Fleet Binary
=============================

For information on how to run the `fleet` binary, detailed usage information can be found by running `fleet --help`. This document is a more detailed version of the information presented in the help output text. If you prefer to use a CLI instead of a web browser, we hope that you like the binary interface to the Fleet application!

## High-level configuration overview

To get the most out of running the Fleet server, it is helpful to establish a mutual understanding of what the desired architecture looks like and what it's trying to accomplish.

Your Fleet server's two main purposes are:

- To serve as your [osquery TLS server](https://osquery.readthedocs.io/en/stable/deployment/remote/)
- To serve the [Fleet web application](https://kolide.com/fleet), which allows you to manage osquery configuration, query hosts, perform interesting analytics, etc.

The Fleet server allows you persist configuration, manage users, etc. Thus, it needs a database. Fleet uses MySQL and requires you to supply configurations to connect to a MySQL server. Fleet also uses Redis to perform some more high-speed data access action throughout the lifecycle of the application (for example, distributed query result ingestion). Thus, Fleet also requires that you supply Redis connention configurations.

Since Fleet is a web application, when you run Fleet there are some other configurations that are worth defining, such as:

- The TLS certificates that Fleet should use to terminate TLS.
- The [JWT](https://jwt.io/) Key which is used to sign and verify session tokens.

Since Fleet is an osquery TLS server, you are also able to define configurations that can customize your experience there, such as:

- The destination of the osquery status and result logs on the local filesystem
- Various details about the refresh/check-in intervals for your hosts

## Commands

The `fleet` binary contains several "commands". Similarly to how `git` has many commands (`git status`, `git commit`, etc), the `fleet` binary accepts the following commands:

- `fleet prepare db`
- `fleet serve`
- `fleet version`
- `fleet config_dump`

## Options

### How do you specify options?

In order of precedence, options can be specified via:

- A configuration file (in YAML format)
- Environment variables
- Command-line flags

For example, all of the following ways of launching Fleet are equivalent:

#### Using only CLI flags

```
$ /usr/bin/fleet serve \
  --mysql_address=127.0.0.1:3306 \
  --mysql_database=kolide \
  --mysql_username=root \
  --mysql_password=toor \
  --redis_address=127.0.0.1:6379 \
  --server_cert=/tmp/server.cert \
  --server_key=/tmp/server.key \
  --logging_json \
  --auth_jwt_key=changeme
```

#### Using only environment variables

```
$ KOLIDE_MYSQL_ADDRESS=127.0.0.1:3306 \
  KOLIDE_MYSQL_DATABASE=kolide \
  KOLIDE_MYSQL_USERNAME=root \
  KOLIDE_MYSQL_PASSWORD=toor \
  KOLIDE_REDIS_ADDRESS=127.0.0.1:6379 \
  KOLIDE_SERVER_CERT=/tmp/server.cert \
  KOLIDE_SERVER_KEY=/tmp/server.key \
  KOLIDE_LOGGING_JSON=true \
  KOLIDE_AUTH_JWT_KEY=changeme \
  /usr/bin/fleet serve
```

#### Using a config file

```
$ echo '
mysql:
  address: 127.0.0.1:3306
  database: kolide
  username: root
  password: toor
redis:
  address: 127.0.0.1:6379
server:
  cert: /tmp/server.cert
  key: /tmp/server.key
logging:
  json: true
auth:
  jwt_key: changeme
' > /tmp/kolide.yml
$ fleet serve --config /tmp/kolide.yml
```

### What are the options?

Note that all option names can be converted consistently from flag name to environment variable and visa-versa. For example, the `--mysql_address` flag would be the `KOLIDE_MYSQL_ADDRESS`. Further, specifying the `mysql_address` option in the config would follow the pattern:

```
mysql:
  address: 127.0.0.1:3306
```


Basically, just capitalize the option and prepend `KOLIDE_` to it in order to get the environment variable. The conversion works the same the opposite way.

#### MySQL

##### `mysql_address`

The address of the MySQL server which Fleet should connect to. Include the hostname and port.

- Default value: `localhost:3306`
- Environment variable: `KOLIDE_MYSQL_ADDRESS`
- Config file format:

	```
	mysql:
		address: localhost:3306
	```

##### `mysql_database`

The name of the MySQL database which Fleet will use.

- Default value: `kolide`
- Environment variable: `KOLIDE_MYSQL_DATABASE`
- Config file format:

	```
	mysql:
		database: kolide
	```

##### `mysql_username`

The username to use when connecting to the MySQL instance.

- Default value: `kolide`
- Environment variable: `KOLIDE_MYSQL_USERNAME`
- Config file format:

	```
	mysql:
		username: kolide
	```

##### `mysql_password`

The password to use when connecting to the MySQL instance.

- Default value: `kolide`
- Environment variable: `KOLIDE_MYSQL_PASSWORD`
- Config file format:

	```
	mysql:
		password: kolide
	```

##### `mysql_tls_ca`

The path to a PEM encoded certificate of MYSQL's CA for client certificate authentication.

- Default value: none
- Environment variable: `KOLIDE_MYSQL_TLS_CA`
- Config file format:

	```
	mysql:
		tls_ca: /path/to/server-ca.pem
	```

##### `mysql_tls_cert`

The path to a PEM encoded certificate use for tls authentication.

- Default value: none
- Environment variable: `KOLIDE_MYSQL_TLS_CERT`
- Config file format:

	```
	mysql:
		tls_cert: /path/to/certificate.pem
	```

##### `mysql_tls_key`

The path to a PEM encoded private key use for tls authentication.

- Default value: none
- Environment variable: `KOLIDE_MYSQL_TLS_KEY`
- Config file format:

	```
	mysql:
		tls_key: /path/to/key.pem
	```

##### `mysql_tls_config`

The tls value in a MYSQL DSN. Can be `true`,`false`,`skip-verify` or the CN value of the certificate.

- Default value: none
- Environment variable: `KOLIDE_MYSQL_TLS_CONFIG`
- Config file format:

	```
	mysql:
		tls_config: true
	```

##### `mysql_tls_server_name`

The server name or IP address used by the client certificate.

- Default value: none
- Environment variable: `KOLIDE_MYSQL_TLS_SERVER_NAME`
- Config file format:

	```
	mysql:
		servername: 127.0.0.1
	```

##### `mysql_max_open_conns`

Maximum open connections to database

- Default value: 50
- Environment variable: `KOLIDE_MYSQL_MAX_OPEN_CONNS`
- Config file format:

	```
	mysql:
		max_open_conns: 50
	```

##### `mysql_max_idle_conns`

Maximum idle connections to database. This value should be equal to or less than `mysql_max_open_conns`

- Default value: 50
- Environment variable: `KOLIDE_MYSQL_MAX_IDLE_CONNS`
- Config file format:

	```
	mysql:
		max_idle_conns: 50
	```

#### Redis

##### `redis_address`

The address of the Redis server which Fleet should connect to. Include the hostname and port.

- Default value: `localhost:6379`
- Environment variable: `KOLIDE_REDIS_ADDRESS`
- Config file format:

	```
	redis:
		address: 127.0.0.1:7369
	```

##### `redis_password`

The password to use when connecting to the Redis instance.

- Default value: `<empty>`
- Environment variable: `KOLIDE_REDIS_PASSWORD`
- Config file format:

	```
	redis:
		password: foobar
	```

#### Server

##### `server_address`

The address to serve the Fleet webserver.

- Default value: `0.0.0.0:8080`
- Environment variable: `KOLIDE_SERVER_ADDRESS`
- Config file format:

	```
	server:
		address: 0.0.0.0:443
	```

##### `server_cert`

The TLS cert to use when terminating TLS.

- Default value: `./tools/osquery/kolide.crt`
- Environment variable: `KOLIDE_SERVER_CERT`
- Config file format:

	```
	server:
		cert: /tmp/kolide.crt
	```


##### `server_key`

The TLS key to use when terminating TLS.

- Default value: `./tools/osquery/kolide.key`
- Environment variable: `KOLIDE_SERVER_KEY`
- Config file format:

	```
	server:
		key: /tmp/kolide.key
	```


##### `server_tls`

Whether or not the server should be served over TLS.

- Default value: `true`
- Environment variable: `KOLIDE_SERVER_TLS`
- Config file format:

	```
	server:
		tls: false
	```

##### `server_tls_compatibility`

Configures the TLS settings for compatibility with various user agents. Options are `modern`, `intermediate`, and `old`. These correspond to the compatibility levels [defined by the Mozilla OpSec team](https://wiki.mozilla.org/Security/Server_Side_TLS)

- Default value: `modern`
- Environment variable: `KOLIDE_SERVER_TLS_COMPATIBILITY`
- Config file format:

	```
	server:
		tls_compatibility: intermediate
	```


#### Auth

##### `auth_jwt_key`

The [JWT](https://jwt.io/) key to use when signing and validating session keys. If this value is not specified the Fleet server will fail to start and a randomly generated key will be provided for use.

- Default value: None
- Environment variable: `KOLIDE_AUTH_JWT_KEY`
- Config file format:

	```
	auth:
		jwt_key: JVnKw7CaUdJjZwYAqDgUHVYP
	```

#####	`auth_bcrypt_cost`

The bcrypt cost to use when hashing user passwords.

- Default value: `12`
- Environment variable:	`KOLIDE_AUTH_BCRYT_COST`
- Config file format:

	```
	auth:
		bcrypt_cost: 14
	```

##### `auth_salt_key_size`

The key size of the salt which is generated when hashing user passwords.

- Default value: `24`
- Environment variable: `KOLIDE_AUTH_SALT_KEY_SIZE`
- Config file format:

	```
	auth:
		salt_key_size: 36
	```

#### App

##### `app_token_key_size`

Size of generated app tokens.

- Default value: `24`
- Environment variable: `KOLIDE_APP_TOKEN_KEY_SIZE`
- Config file format:

	```
	app:
		token_key_size: 36
	```

##### `app_invite_token_validity_period`

How long invite tokens should be valid for.

- Default value: `5 days`
- Environment variable: `KOLIDE_APP_TOKEN_VALIDITY_PERIOD`
- Config file format:

	```
	app:
		invite_token_validity_period: 1d
	```

#### Session

##### `session_key_size`

The size of the session key.

- Default value: `64`
- Environment variable: `KOLIDE_SESSION_KEY_SIZE`
- Config file format:

	```
	session:
		key_size: 48
	```

##### `session_duration`

The amount of time that a session should last for.

- Default value: `90 days`
- Environment variable: `KOLIDE_SESSION_DURATION`
- Config file format:

	```
	session:
		duration: 30d
	```

#### Osquery

##### `osquery_node_key_size`

The size of the node key which is negotiated with `osqueryd` clients.

- Default value: `24`
- Environment variable:	`KOLIDE_OSQUERY_NODE_KEY_SIZE`
- Config file format:

	```
	osquery:
		node_key_size: 36
	```

##### `osquery_status_log_file`

The path which osquery status logs will be logged to.

- Default value: `/tmp/osquery_status`
- Environment variable: `KOLIDE_OSQUERY_STATUS_LOG_FILE`
- Config file format:

	```
	osquery:
		status_log_file: /var/log/osquery/status.log
	```

##### `osquery_result_log_file`

The path which osquery result logs will be logged to.

- Default value: `/tmp/osquery_result`
- Environment variable: `KOLIDE_OSQUERY_RESULT_LOG_FILE`
- Config file format:

	```
	osquery:
		result_log_file: /var/log/osquery/result.log
	```

##### `osquery_label_update_interval`

The interval at which Fleet will ask osquery agents to update their results for label queries.

- Default value: `1h`
- Environment variable: `KOLIDE_OSQUERY_LABEL_UPDATE_INTERVAL`
- Config file format:

	```
	osquery:
		label_query_update_interval: 30m
	```

##### `osquery_enable_log_rotation`

This flag will cause the osquery result and status log files to be automatically
rotated when files reach a size of 500 Mb or an age of 28 days.

- Default value: `false`
- Environment variable: `KOLIDE_OSQUERY_ENABLE_LOG_ROTATION`
- Config file format:

  ```
  osquery:
     enable_log_rotation: true
  ```

#### Logging

##### `logging_debug`

Whether or not to enable debug logging.

- Default value: `false`
- Environment variable: `KOLIDE_LOGGING_DEBUG`
- Config file format:

	```
	logging:
		debug: true
	```

##### `logging_json`

Whether or not to log in JSON.

- Default value: `false`
- Environment variable: `KOLIDE_LOGGING_JSON`
- Config file format:

	```
	logging:
		json: true
	```

##### `logging_disable_banner`

Whether or not to log the welcome banner.

- Default value: `false`
- Environment variable: `KOLIDE_LOGGING_DISABLE_BANNER`
- Config file format:

	```
	logging:
		diable_banner: true
	```
