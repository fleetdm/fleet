# Deployment FAQ

- [How do I get support for working with Fleet?](#how-do-i-get-support-for-working-with-fleet)
- [Can multiple instances of the Fleet server be run behind a load-balancer?](#can-multiple-instances-of-the-fleet-server-be-run-behind-a-load-balancer)
- [Why aren't my osquery agents connecting to Fleet?](#why-arent-my-osquery-agents-connecting-to-fleet)
- [How do I fix "certificate verify failed" errors from osqueryd?](#how-do-i-fix-certificate-verify-failed-errors-from-osqueryd)
- [What do I do about "too many open files" errors?](#what-do-i-do-about-too-many-open-files-errors)
- [I upgraded my database, but Fleet is still running slowly. What could be going on?](#i-upgraded-my-database-but-fleet-is-still-running-slowly-what-could-be-going-on)
- [Why am I receiving a database connection error when attempting to "prepare" the database?](#why-am-i-receiving-a-database-connection-error-when-attempting-to-prepare-the-database)
- [Is Fleet available as a SaaS product?](#is-fleet-available-as-a-saas-product)

## How do I get support for working with Fleet?

For bug reports, please use the [Github issue tracker](https://github.com/fleetdm/fleet/issues).

For questions and discussion, please join us in the #fleet channel of [osquery Slack](https://osquery.slack.com/join/shared_invite/zt-h29zm0gk-s2DBtGUTW4CFel0f0IjTEw#/).

## Can multiple instances of the Fleet server be run behind a load-balancer?

Yes. Fleet scales horizontally out of the box as long as all of the Fleet servers are connected to the same MySQL and Redis instances.

Note that osquery logs will be distributed across the Fleet servers.

Read the [performance documentation](../1-Using-Fleet/5-Monitoring-Fleet.md#fleet-server-performance) for more.

## Why aren't my osquery agents connecting to Fleet?

This can be caused by a variety of problems. The best way to debug is usually to add `--verbose --tls_dump` to the arguments provided to `osqueryd` and look at the logs for the server communication.

### Common problems

- `Connection refused`: The server is not running, or is not listening on the address specified. Is the server listening on an address that is available from the host running osquery? Do you have a load balancer that might be blocking connections? Try testing with `curl`.
- `No node key returned`: Typically this indicates that the osquery client sent an incorrect enroll secret that was rejected by the server. Check what osquery is sending by looking in the logs near this error.
- `certificate verify failed`: See [How do I fix "certificate verify failed" errors from osqueryd](#how-do-i-fix-certificate-verify-failed-errors-from-osqueryd).
- `bad record MAC`: When generating your certificate for your Fleet server, ensure you set the hostname to the FQDN or the IP of the server. This error is common when setting up Fleet servers and accepting defaults when generating certificates using `openssl`.

## How do I fix "certificate verify failed" errors from osqueryd?

Osquery requires that all communication between the agent and Fleet are over a secure TLS connection. For the safety of osquery deployments, there is no (convenient) way to circumvent this check.

- Try specifying the path to the full certificate chain used by the server using the `--tls_server_certs` flag in `osqueryd`. This is often unnecessary when using a certificate signed by an authority trusted by the system, but is mandatory when working with self-signed certificates. In all cases it can be a useful debugging step.
- Ensure that the CNAME or one of the Subject Alternate Names (SANs) on the certificate matches the address at which the server is being accessed. If osquery connects via `https://localhost:443`, but the certificate is for `https://fleet.example.com`, the verification will fail.
- Is Fleet behind a load-balancer? Ensure that if the load-balancer is terminating TLS, this is the certificate provided to osquery.
- Does the certificate verify with `curl`? Try `curl -v -X POST https://kolideserver:port/api/v1/osquery/enroll`.

## What do I do about "too many open files" errors?

This error usually indicates that the Fleet server has run out of file descriptors. Fix this by increasing the `ulimit` on the Fleet process. See the `LimitNOFILE` setting in the [example systemd unit file](./2-Configuration.md#runing-with-systemd) for an example of how to do this with systemd.

## I upgraded my database, but Fleet is still running slowly. What could be going on?

This could be caused by a mismatched connection limit between the Fleet server and the MySQL server that prevents Fleet from fully utilizing the database. First [determine how many open connections your MySQL server supports](https://dev.mysql.com/doc/refman/8.0/en/too-many-connections.html). Now set the [`--mysql_max_open_conns`](./2-Configuration.md#mysql_max_open_conns) and [`--mysql_max_idle_conns`](./2-Configuration.md#mysql_max_idle_conns) flags appropriately.

## Why am I receiving a database connection error when attempting to "prepare" the database?

First, check if you have a version of MySQL installed that is at least 5.7. Then, make sure that you currently have a MySQL server running.

The next step is to make sure the credentials for the database match what is expected. Test your ability to connect to the database with `mysql -u<username> -h<hostname_or_ip> -P<port> -D<database_name> -p`.

If you're successful connecting to the database and still receive a database connection error, you may need to specify your database credentials when running `fleet prepare db`. It's encouraged to put your database credentials in environment variables or a config file.

```
fleet prepare db \
    --mysql_address=<database_address> \
    --mysql_database=<database_name> \
    --mysql_username=<username> \
    --mysql_password=<database_password>
```

## Is Fleet available as a SaaS product?

No. Currently, Fleet is only available for self-hosting on premises or in the cloud.
