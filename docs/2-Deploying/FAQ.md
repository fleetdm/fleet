# Deployment FAQ

- [How do I get support for working with Fleet?](#how-do-i-get-support-for-working-with-fleet)
- [Can multiple instances of the Fleet server be run behind a load-balancer?](#can-multiple-instances-of-the-fleet-server-be-run-behind-a-load-balancer)
- [Why aren't my osquery agents connecting to Fleet?](#why-arent-my-osquery-agents-connecting-to-fleet)
- [How do I fix "certificate verify failed" errors from osqueryd?](#how-do-i-fix-certificate-verify-failed-errors-from-osqueryd)
- [What do I need to do to change the Fleet server TLS certificate?](#what-do-i-need-to-do-to-change-the-fleet-server-tls-certificate)
- [When do I need to deploy a new enroll secret to my hosts?](#when-do-i-need-to-deploy-a-new-enroll-secret-to-my-hosts)
- [How do I migrate hosts from one Fleet server to another (eg. testing to production)?](#how-do-i-migrate-hosts-from-one-fleet-server-to-another-eg-testing-to-production)
- [What do I do about "too many open files" errors?](#what-do-i-do-about-too-many-open-files-errors)
- [I upgraded my database, but Fleet is still running slowly. What could be going on?](#i-upgraded-my-database-but-fleet-is-still-running-slowly-what-could-be-going-on)
- [Why am I receiving a database connection error when attempting to "prepare" the database?](#why-am-i-receiving-a-database-connection-error-when-attempting-to-prepare-the-database)
- [Is Fleet available as a SaaS product?](#is-fleet-available-as-a-saas-product)
- [Is Fleet compatible with X flavor of MySQL?](#is-fleet-compatible-with-x-flavor-of-mysql)

## How do I get support for working with Fleet?

For bug reports, please use the [Github issue tracker](https://github.com/fleetdm/fleet/issues).

For questions and discussion, please join us in the #fleet channel of [osquery Slack](https://osquery.slack.com/join/shared_invite/zt-h29zm0gk-s2DBtGUTW4CFel0f0IjTEw#/).

## Can multiple instances of the Fleet server be run behind a load-balancer?

Yes. Fleet scales horizontally out of the box as long as all of the Fleet servers are connected to the same MySQL and Redis instances.

Note that osquery logs will be distributed across the Fleet servers.

Read the [performance documentation](../1-Using-Fleet/6-Monitoring-Fleet.md#fleet-server-performance) for more.

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
- Does the certificate verify with `curl`? Try `curl -v -X POST https://fleetserver:port/api/v1/osquery/enroll`.

## What do I need to do to change the Fleet server TLS certificate?

If the both the existing and new certificates verify with osquery's default root certificates (such as a certificate issued by a well-known Certificate Authority) and no certificate chain was deployed with osquery, there is no need to deploy a new certificate chain.

If osquery has been deployed with the full certificate chain (using `--tls_server_certs`), deploying a new certificate chain is necessary to allow for verification of the new certificate.

Deploying a certificate chain cannot be done centrally from Fleet.

## When do I need to deploy a new enroll secret to my hosts?

Osquery provides the enroll secret only during the enrollment process. Once a host is enrolled, the node key it receives remains valid for authentication independent from the enroll secret.

Currently enrolled hosts do not necessarily need enroll secrets updated, as the existing enrollment will continue to be valid as long as the host is not deleted from Fleet and the osquery store on the host remains valid. Any newly enrolling hosts must have the new secret.

Deploying a new enroll secret cannot be done centrally from Fleet.

## How do I migrate hosts from one Fleet server to another (eg. testing to production)?

Primarily, this would be done by changing the `--tls_hostname` and enroll secret to the values for the new server. In some circumstances (see [What do I need to do to change the Fleet server TLS certificate?](#what-do-i-need-to-do-to-change-the-fleet-server-tls-certificate)) it may be necessary to deploy a new certificate chain configured with `--tls_server_certs`.

These configurations cannot be managed centrally from Fleet.

## What do I do about "too many open files" errors?

This error usually indicates that the Fleet server has run out of file descriptors. Fix this by increasing the `ulimit` on the Fleet process. See the `LimitNOFILE` setting in the [example systemd unit file](./2-Configuration.md#runing-with-systemd) for an example of how to do this with systemd.

Some deployments may benefit by setting the [`--server_keepalive`](./2-Configuration.md#server_keepalive) flag to false.

This was also seen as a symptom of a different issue: if you're deploying on AWS on T type instances, there are different scenarios where the activity can increase and the instances will burst. If they run out of credits, then they'll stop processing leaving the file descriptors open.

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

## Is Fleet compatible with X flavor of MySQL?

Fleet is built to run on MySQL 5.7 or above. However, particularly with AWS Aurora, we recommend 2.10.0 and above, as we've seen issues with anything below that.
