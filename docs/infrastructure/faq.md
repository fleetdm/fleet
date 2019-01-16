# FAQ for using/operating Fleet

## Can multiple instances of the Fleet server be run behind a load-balancer?

Yes. Fleet scales horizontally out of the box as long as all of the Fleet servers are connected to the same MySQL and Redis instances.

Note that osquery logs will be distributed across the Fleet servers.

## Where are my query results?

### Live Queries

Live query results (executed in the web UI or `fleetctl query`) are pushed directly to the UI where the query is running. The results never go to a file unless you as the user manually save them.

### Scheduled Queries

Scheduled query results (queries that are scheduled to run in Packs) are typically sent to the Fleet server, and will be available on the filesystem of the server at the path configurable by [`--osquery_result_log_file`](./configuring-the-fleet-binary.md#osquery_result_log_file). This defaults to `/tmp/osquery_result`.

It is possible to configure osqueryd to log query results outside of Fleet. For results to go to Fleet, the `--logger_plugin` flag must be set to `tls`.

### Troubleshooting

Expecting results, but not seeing anything in the logs?

- Try scheduling a query that always returns results (eg. `SELECT * FROM time`).
- Check whether the query is scheduled in differential mode. If so, new results will only be logged when the result set changes.
- Ensure that the query is scheduled to run on the intended platforms, and that the tables queried are supported by those platforms.
- Look at the status logs provided by osquery. These are available on the filesystem of the Fleet server at the path configurable by [`--osquery_status_log_file`](./configuring-the-fleet-binary.md#osquery_status_log_file). This defaults to `/tmp/osquery_status`.

## Why aren’t my live queries being logged?

Live query results are never logged to the filesystem of the Fleet server. See [Where are my query results?](#where-are-my-query-results).

## Why aren't my osquery agents connecting to Fleet?

This can be caused by a variety of problems. The best way to debug is usually to add `--verbose --tls_dump` to the arguments provided to `osqueryd` and look at the logs for the server communication.

### Common problems

- `Connection refused`: The server is not running, or is not listening on the address specified. Is the server listening on an address that is available from the host running osquery? Do you have a load balancer that might be blocking connections? Try testing with `curl`.
- `No node key returned`: Typically this indicates that the osquery client sent an incorrect enroll secret that was rejected by the server. Check what osquery is sending by looking in the logs near this error.
- `certificate verify failed`: See [How do I fix "certificate verify failed" errors from osqueryd](#how-do-i-fix-certificate-verify-failed-errors-from-osqueryd).

## How do I fix "certificate verify failed" errors from osqueryd?

Osquery requires that all communication between the agent and Fleet are over a secure TLS connection. For the safety of osquery deployments, there is no (convenient) way to circumvent this check.

- Try specifying the path to the full certificate chain used by the server using the `--tls_server_certs` flag in `osqueryd`. This is often unnecessary when using a certificate signed by an authority trusted by the system, but is mandatory when working with self-signed certificates. In all cases it can be a useful debugging step.
- Ensure that the CNAME on the certificate matches the address at which the server is being accessed. If I try connect osquery via `https://localhost:443`, but my certificate is for `https://fleet.example.com`, the verification will fail.
- Is Fleet behind a load-balancer? Ensure that if the load-balancer is terminating TLS that this is the certificate provided to osquery.
- Does the certificate verify with `curl`? Try `curl -v -X POST https://kolideserver:port/api/v1/osquery/enroll`.

## What do I do about "too many open files" errors?

This error usually indicates that the Fleet server has run out of file descriptors. Fix this by increasing the `ulimit` on the Fleet process. See the `LimitNOFILE` setting in the [example systemd unit file](./systemd.md) for an example of how to do this with systemd.

## I upgraded my database, but Fleet is still running slowly. What could be going on?

This could be caused by a mismatched connection limit between the Fleet server and the MySQL server that prevents Fleet from fully utilizing the database. First [determine how many open connections your MySQL server supports](https://dev.mysql.com/doc/refman/8.0/en/too-many-connections.html). Now set the [`--mysql_max_open_conns`](./configuring-the-fleet-binary.md#mysql_max_open_conns) and [`--mysql_max_idle_conns`](./configuring-the-fleet-binary.md#mysql_max_idle_conns) flags appropriately.

## How do I monitor a Fleet server?

Fleet provides a `/healthz` endpoint. If you query it with `curl` it will return an HTTP Status code. `200 OK` means everything is alright. `500 Internal Server Error` means Fleet is having trouble communicating with MySQL or Redis. Check the Fleet logs for additional details.

The `/metrics` endpoint exposes data ready to be ingested by Prometheus.

## Why is the "Add User" button disabled?

The "Add User" button is disabled if SMTP (email) has not been configured for the Fleet server. Currently, there is no way to add new users without email capabilities.

One way to hack around this is to use a simulated mailserver like [Mailhog](https://github.com/mailhog/MailHog). You can retrieve the email that was "sent" in the Mailhog UI, and provide users with the invite URL manually.

## Is Fleet available as a SaaS product?

Kolide does not host a SaaS version of Fleet. We offer [Kolide Cloud](https://kolide.com) which is a separate product providing the capabilities of Fleet along with alerting and insights, all hosted in a secure SaaS platform.

## How do I get support for working with Fleet?

For bug reports, please use the [Github issue tracker](https://github.com/kolide/fleet/issues).

For questions and discussion, please join us in the #kolide channel of [osquery Slack](https://osquery-slack.herokuapp.com/).
