# Deployment FAQ

- [How do I get support for working with Fleet?](#how-do-i-get-support-for-working-with-fleet)
- [Can multiple instances of the Fleet server be run behind a load-balancer?](#can-multiple-instances-of-the-fleet-server-be-run-behind-a-load-balancer)
- [Why aren't my osquery agents connecting to Fleet?](#why-arent-my-osquery-agents-connecting-to-fleet)
- [How do I fix "certificate verify failed" errors from osqueryd?](#how-do-i-fix-certificate-verify-failed-errors-from-osqueryd)
- [What do I need to do to change the Fleet server TLS certificate?](#what-do-i-need-to-do-to-change-the-fleet-server-tls-certificate)
- [How do I migrate hosts from one Fleet server to another (eg. testing to production)?](#how-do-i-migrate-hosts-from-one-fleet-server-to-another-eg-testing-to-production)
- [What do I do about "too many open files" errors?](#what-do-i-do-about-too-many-open-files-errors)
- [Can I skip versions when updating Fleet to the latest version?](#can-i-skip-versions-when-updating-to-the-latest-version)
- [I upgraded my database, but Fleet is still running slowly. What could be going on?](#i-upgraded-my-database-but-fleet-is-still-running-slowly-what-could-be-going-on)
- [Why am I receiving a database connection error when attempting to "prepare" the database?](#why-am-i-receiving-a-database-connection-error-when-attempting-to-prepare-the-database)
- [Is Fleet available as a SaaS product?](#is-fleet-available-as-a-saas-product)
- [What MySQL versions are supported?](#what-mysql-versions-are-supported)
- [What are the MySQL user access requirements?](#what-are-the-mysql-user-requirements)
- [Does Fleet support MySQL replication?](#does-fleet-support-mysql-replication)
- [What is duplicate enrollment and how do I fix it?](#what-is-duplicate-enrollment-and-how-do-i-fix-it)
- [What API endpoints should I expose to the public internet?](#what-api-endpoints-should-i-expose-to-the-public-internet)
- [What Redis versions are supported?](#what-redis-versions-are-supported)
- [Will my older version of Fleet work with Redis 6?](#will-my-older-version-of-fleet-work-with-redis-6)

## How do I get support for working with Fleet?

For bug reports, please use the [Github issue tracker](https://github.com/fleetdm/fleet/issues).

For questions and discussion, please join us in the #fleet channel of [osquery Slack](https://fleetdm.com/slack).

## Can multiple instances of the Fleet server be run behind a load-balancer?

Yes. Fleet scales horizontally out of the box as long as all of the Fleet servers are connected to the same MySQL and Redis instances.

Note that osquery logs will be distributed across the Fleet servers.

Read the [performance documentation](https://fleetdm.com/docs/using-fleet/monitoring-fleet#fleet-server-performance) for more.

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

## How do I use a proxy server with Fleet?

Seeing your proxy's requests fail with an error like `DEPTH_ZERO_SELF_SIGNED_CERT`)?
To get your proxy server's HTTP client to work with a local Fleet when using a self-signed cert, disable SSL / self-signed verification in the client.

The exact solution to this depends on the request client you are using. For example, when using Node.js ± Sails.js, you can work around this in the requests you're sending with `await sails.helpers.http.get()` by lifting your app with the `NODE_TLS_REJECT_UNAUTHORIZED` environment variable set to `0`:

```
NODE_TLS_REJECT_UNAUTHORIZED=0 sails console
```

## I'm only getting partial results from live queries

Redis has an internal buffer limit for pubsub that Fleet uses to communicate query results. If this buffer is filled, extra data is dropped. To fix this, we recommend disabling the buffer size limit. Most installs of Redis should have plenty of spare memory to not run into issues. More info about this limit can be found [here](https://redis.io/topics/clients#:~:text=Pub%2FSub%20clients%20have%20a,64%20megabyte%20per%2060%20second.) and [here](https://raw.githubusercontent.com/redis/redis/unstable/redis.conf) (search for client-output-buffer-limit).

We recommend a config like the following:

```
client-output-buffer-limit pubsub 0 0 60
```

## How do I migrate hosts from one Fleet server to another (eg. testing to production)?

Primarily, this would be done by changing the `--tls_hostname` and enroll secret to the values for the new server. In some circumstances (see [What do I need to do to change the Fleet server TLS certificate?](#what-do-i-need-to-do-to-change-the-fleet-server-tls-certificate)) it may be necessary to deploy a new certificate chain configured with `--tls_server_certs`.

These configurations cannot be managed centrally from Fleet.

## What do I do about "too many open files" errors?

This error usually indicates that the Fleet server has run out of file descriptors. Fix this by increasing the `ulimit` on the Fleet process. See the `LimitNOFILE` setting in the [example systemd unit file](https://fleetdm.com/docs/deploying/configuration#runing-with-systemd) for an example of how to do this with systemd.

Some deployments may benefit by setting the [`--server_keepalive`](https://fleetdm.com/docs/deploying/configuration#server-keepalive) flag to false.

This was also seen as a symptom of a different issue: if you're deploying on AWS on T type instances, there are different scenarios where the activity can increase and the instances will burst. If they run out of credits, then they'll stop processing leaving the file descriptors open.

## Can I skip versions when updating Fleet to the latest version?

Absolutely! If you're updating from the current major release of Fleet (v4), you can install the [latest version](https://github.com/fleetdm/fleet/releases/latest) without upgrading to each minor version along the way. Just make sure to back up your database in case anything odd does pop up!

If you're updating from an older version (we'll use Fleet v3 as an example), it's best to take some stops along the way:

1. Back up your database. 
2. Upgrade to the last release of of v3 - [3.13.0](https://github.com/fleetdm/fleet/releases/tag/3.13.0).
3. Migrate the database.
4. Test
5. Check the release post for [v4.0.0 ](https://github.com/fleetdm/fleet/releases/tag/v4.0.0) to see the breaking changes and get Fleet ready for v4.
6. Upgrade to v4.0.0.
7. Migrate the database.
8. Test
9. Upgrade to the [current release](https://github.com/fleetdm/fleet/releases/latest).
10. One last migration.
11. Test again for good measure. 

Taking it a bit slower on major releases gives you an opportunity to better track down where any issues may have been introduced. 

## I upgraded my database, but Fleet is still running slowly. What could be going on?

This could be caused by a mismatched connection limit between the Fleet server and the MySQL server that prevents Fleet from fully utilizing the database. First [determine how many open connections your MySQL server supports](https://dev.mysql.com/doc/refman/8.0/en/too-many-connections.html). Now set the [`--mysql_max_open_conns`](https://fleetdm.com/docs/deploying/configuration#mysql-max-open-conns) and [`--mysql_max_idle_conns`](https://fleetdm.com/docs/deploying/configuration#mysql-max-idle-conns) flags appropriately.

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

Yes! Please sign up for the [Fleet Cloud Beta](https://kqphpqst851.typeform.com/to/yoo5smT9).

## What MySQL versions are supported?

Fleet is tested with MySQL 5.7.21 and 8.0.28. Newer versions of MySQL 5.7 and MySQL 8 typically work well. AWS Aurora requires at least version 2.10.0. Please avoid using MariaDB or other MySQL variants that are not officially supported. Compatibility issues have been identified with MySQL variants and these may not be addressed in future Fleet releases.

## What are the MySQL user requirements?

The user `fleet prepare db` (via environment variable `FLEET_MYSQL_USERNAME` or command line flag `--mysql_username=<username>`) uses to interact with the database needs to be able to create, alter, and drop tables as well as the ability to create temporary tables.

## Does Fleet support MySQL replication?

You can deploy MySQL or Maria any way you want. We recommend using managed/hosted mysql so you don't have to think about it, but you can think about it more if you want. Read replicas are supported. You can read more about MySQL configuration [here](https://fleetdm.com/docs/deploying/configuration#mysql).

## What is duplicate enrollment and how do I fix it?

Duplicate host enrollment is when more than one host enrolls in Fleet using the same identifier
(hardware UUID or osquery generated UUID).

Typically, this is caused by cloning a VM Image with an already enrolled
osquery client, which results in duplicate osquery generated UUIDs. To resolve this issue, it is
advised to configure `--osquery_host_identifier=uuid` (which will use the hardware UUID), and then
delete the associated host in the Fleet UI.

In rare instances, VM Hypervisors have been seen to duplicate hardware UUIDs. When this happens,
using `--osquery_host_identifier=uuid` will not resolve the duplicate enrollment problem. Sometimes
the problem can be resolved by setting `--osquery_host_identifier=instance` (which will use the
osquery generated UUID), and then delete the associated host in the Fleet UI.

Find more information about [host identifiers here](https://fleetdm.com/docs/deploying/configuration#osquery-host-identifier).

## How do I resolve an "unknown column" error when upgrading Fleet?

The `unknown column` error typically occurs when the database migrations haven't been run during the upgrade process.

Check out the [documentation on running database migrations](https://fleetdm.com/docs/deploying/upgrading-fleet#running-database-migrations) to resolve this issue.

## What API endpoints should I expose to the public internet?

If you would like to manage hosts that can travel outside your VPN or intranet we recommend only exposing the osquery endpoints to the public internet:

- `/api/osquery`
- `/api/v1/osquery`

If you are using Fleet Desktop and want it to work on remote devices, the bare minimum API to expose is `/api/latest/fleet/device/*/desktop`. This minimal endpoint will only provide the number of failing policies. 

For full Fleet Desktop functionality, `/api/fleet/orbit/*` and`/api/fleet/device/ping` must also be exposed.

If you would like to use the fleetctl CLI from outside of your network, the following endpoints will also need to be exposed for `fleetctl`:

- `/api/setup`
- `/api/v1/setup`
- `/api/latest/fleet/*`
- `/api/v1/fleet/*`

**IN PROGRESS** If you would like to use Fleet MDM, the following endpoints need to be exposed:

- `/mdm/apple/scep` to allow hosts to obtain a SCEP certificate.
- `/mdm/apple/mdm` to allow hosts to reach the server using the MDM protocol.
- `/api/mdm/apple/enroll` to allow DEP enrolled devices to get an enrollment profile.
- `/api/*/fleet/device/*/mdm/apple/manual_enrollment_profile` to allow manually enrolled devices to
  download an enrollment profile.

> The `/mdm/apple/scep` and `/mdm/apple/mdm` endpoints are outside of the `/api` path because they
> are not RESTful, and are not intended for use by API clients or browsers. 

## What is the minimum version of MySQL required by Fleet?

Fleet requires at least MySQL version 5.7.

## How do I migrate from Fleet Free to Fleet Premium?

To migrate from Fleet Free to Fleet Premium, once you get a Fleet license, set it as a parameter to `fleet serve` either as an environment variable using `FLEET_LICENSE_KEY` or in the Fleet's config file. See [here](https://fleetdm.com/docs/deploying/configuration#license) for more details. Note: You don't need to redeploy Fleet after the migration.

## What Redis versions are supported?
Fleet is tested with Redis 5.0.14 and 6.2.7. Any version Redis after version 5 will typically work well.

## Will my older version of Fleet work with Redis 6?

Most likely, yes! While we'd definitely recommend keeping Fleet up to date in order to take advantage of new features and bug patches, most legacy versions should work with Redis 6. Just keep in mind that we likely haven't tested your particular combination so that you may run into some unforeseen hiccups. 
