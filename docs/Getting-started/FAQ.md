# FAQs

*On this page*

- [Using Fleet](#using-fleet)
- [Deploying](#deploying)
- [Contributing](#contributing)

## Using Fleet

- [How can I switch to Fleet from Kolide Fleet?](#how-can-i-switch-to-fleet-from-kolide-fleet)
- [Has anyone stress tested Fleet? How many clients can the Fleet server handle?](#has-anyone-stress-tested-fleet-how-many-clients-can-the-fleet-server-handle)
- [Can I target my hosts using their enroll secrets?](#can-I-target-my-hosts-using-their-enroll-secrets)
- [How often do labels refresh? Is the refresh frequency configurable?](#how-often-do-labels-refresh-is-the-refresh-frequency-configurable)
- [How do I revoke the authorization tokens for a user?](#how-do-i-revoke-the-authorization-tokens-for-a-user)
- [How do I monitor the performance of my queries?](#how-do-i-monitor-the-performance-of-my-queries)
- [How do I monitor a Fleet server?](#how-do-i-monitor-a-fleet-server)
- [Why is the “Add User” button disabled?](#why-is-the-add-user-button-disabled)
- [Can I disable password-based authentication in the Fleet UI?](#can-i-disable-password-based-authentication-in-the-fleet-ui)
- [Where are my query results?](#where-are-my-query-results)
- [Why aren’t my live queries being logged?](#why-arent-my-live-queries-being-logged)
- [Can I use the Fleet API to fetch results from a scheduled query pack?](#can-i-use-the-fleet-api-to-fetch-results-from-a-scheduled-query-pack)
- [How do I automatically add hosts to packs when the hosts enroll to Fleet?](#how-do-i-automatically-add-hosts-to-packs-when-the-hosts-enroll-to-Fleet)
- [How do I automatically assign a host to a team when it enrolls with Fleet?](#how-do-i-automatically-assign-a-host-to-a-team-when-it-enrolls-with-fleet)
- [How do I resolve an "unknown column" error when upgrading Fleet?](#how-do-i-resolve-an-unknown-column-error-when-upgrading-fleet)
- [Why is my host not updating a policy's response.](#why-is-my-host-not-updating-a-policys-response)
- [What should I do if my computer is showing up as an offline host?](#what-should-i-do-if-my-computer-is-showing-up-as-an-offline-host)
- [How does Fleet deal with IP duplication?](#how-does-fleet-deal-with-ip-duplication)
- [Can Orbit run alongside osquery?](#can-orbit-run-alongside-osquery)
- [Can I disable auto updates for Orbit?](#can-i-disable-auto-updates-for-orbit)
- [Can I bundle osquery extensions into Orbit?](#can-i-bundle-osquery-extensions-into-orbit)
- [How does Fleet work with osquery extensions?](#how-does-fleet-work-with-osquery-extensions)
- [Why am I seeing "unknown certificate error" when adding hosts to my dev server?](#why-am-i-seeing-"unknown-certificate-error"-when-adding-hosts-to-my-dev-server)
- [Can I hide known vulnerabilities that I feel are insignificant?](#can-i-hide-known-vulnerabilities-that-i-feel-are-insignificant)
- [Can I create reports based on historical data in Fleet?](#can-i-create-reports-based-on-historical-data-in-fleet)
- [Why can't I run queries with `fleetctl` using a new API-only user?](#why-cant-i-run-queries-with-fleetctl-using-a-new-api-only-user)
- [Why am I getting an error about self-signed certificates when running `fleetctl preview`?](#why-am-i-getting-an-error-about-self-signed-certificates-when-running-fleetctl-preview)
- [Can I audit actions taken in Fleet?](#can-i-audit-actions-taken-in-fleet)
- [How often is the software inventory updated?](#how-often-is-the-software-inventory-updated)
- [Can I group results from multiple hosts?](#can-i-group-results-from-multiple-hosts)
- [Will updating fleetctl lead to loss of data in fleetctl preview?](will-updating-fleetctl-lead-to-loss-of-data-in-fleetctl-preview?)
- [How do I downgrade from Fleet Premium to Fleet Free?](how-do-i-downgrade-from-fleet-premium-to-fleet-free)
- [If I use a software orchestration tool (Ansible, Chef, Puppet, etc.) to manage agent options, do I have to apply the same options in the Fleet UI?](#if-i-use-a-software-orchestration-tool-ansible-chef-puppet-etc-to-manage-agent-options-do-i-have-to-apply-the-same-options-in-the-fleet-ui)
- [How can I uninstall Orbit/Fleet Desktop?](#how-can-i-uninstall-orbit-fleet-desktop)

### How can I switch to Fleet from Kolide Fleet?

To migrate to Fleet from Kolide Fleet, please follow the steps outlined in the [Upgrading Fleet section](../Deploying/Upgrading-Fleet.md) of the documentation.


### Has anyone stress tested Fleet? How many hosts can the Fleet server handle?

Fleet has been stress tested to 150,000 online hosts and 400,000 total enrolled hosts. Production deployments exist with over 100,000 hosts and numerous production deployments manage tens of thousands of hosts.

It’s standard deployment practice to have multiple Fleet servers behind a load balancer. However, typically the MySQL database is the performance bottleneck and a single Fleet server can handle tens of thousands of hosts.

### Can I target my hosts using their enroll secrets?

No, currently, there’s no way to retrieve the name of the enroll secret with a query. This means that there's no way to create a label using your hosts' enroll secrets and then use this label as a target for queries or query packs.

Typically folks will use some other unique identifier to create labels that distinguish each type of device. As a workaround, [Fleet's manual labels](../Using-Fleet/fleetctl-CLI.md#host-labels) provide a way to create groups of hosts without a query. These manual labels can then be used as targets for queries or query packs.

There is, however, a way to accomplish this even though the answer to the question remains "no": Teams. As of Fleet v4.0.0, you can group hosts in Teams either by enrolling them with a team specific secret, or by transferring hosts to a team. One the hosts you want to target are part of a team, you can create a query and target the team in question.

### How often do labels refresh? Is the refresh frequency configurable?

The update frequency for labels is configurable with the [—osquery_label_update_interval](../Deploying/Configuration.md#osquery-label-update-interval) flag (default 1 hour).

### How do I revoke the authorization tokens for a user?

Authorization tokens are revoked when the “require password reset” action is selected for that user. User-initiated password resets do not expire the existing tokens.

### How do I monitor the performance of my queries?

Fleet can live query the `osquery_schedule` table. Performing this live query allows you to get the performance data for your scheduled queries. Also consider scheduling a query to the `osquery_schedule` table to get these logs into your logging pipeline.

### How do I monitor a Fleet server?

Fleet provides standard interfaces for monitoring and alerting. See the [Monitoring Fleet](../Using-Fleet/Monitoring-Fleet.md) documentation for details.

### Why is the “Add User” button disabled?

The “Add User” button is disabled if SMTP (email) has not been configured for the Fleet server. Currently, there is no way to add new users without email capabilities.

One way to hack around this is to use a simulated mailserver like [Mailhog](https://github.com/mailhog/MailHog). You can retrieve the email that was “sent” in the Mailhog UI, and provide users with the invite URL manually.

### Can I disable password-based authentication in the Fleet UI?

Some folks like to enforce users with SAML SSO enabled to login only via the SSO and not via password.

There is no option in the Fleet UI for disabling password-based authentication.
However, users that have SSO enabled in Fleet will not be able to log in via password-based authentication.

If a user has SSO enabled, the Login page in the Fleet UI displays the “Email” and “Password” fields but on attempted password-based login, this user will receive an “Authentication failed” message.

### Where are my query results?

#### Live queries

Live query results (executed in the web UI or `fleetctl query`) are pushed directly to the UI where the query is running. The results never go to a file unless you as the user manually save them.

#### Scheduled queries

Scheduled query results (queries that are scheduled to run in Packs) are typically sent to the Fleet server, and will be available on the filesystem of the server at the path configurable by [`--osquery_result_log_file`](../Deploying/Configuration.md#osquery-result-log-file). This defaults to `/tmp/osquery_result`.

It is possible to configure osqueryd to log query results outside of Fleet. For results to go to Fleet, the `--logger_plugin` flag must be set to `tls`.

#### What are my options for storing the osquery logs?

Folks typically use Fleet to ship logs to data aggregation systems like Splunk, the ELK stack, and Graylog.

The [logger configuration options](../Deploying/Configuration.md#osquery-status-log-plugin) allow you to select the log output plugin. Using the log outputs you can route the logs to your chosen aggregation system.

#### Troubleshooting

Expecting results, but not seeing anything in the logs?

- Try scheduling a query that always returns results (eg. `SELECT * FROM time`).
- Check whether the query is scheduled in differential mode. If so, new results will only be logged when the result set changes.
- Ensure that the query is scheduled to run on the intended platforms, and that the tables queried are supported by those platforms.
- Use live query to `SELECT * FROM osquery_schedule` to check whether the query has been scheduled on the host.
- Look at the status logs provided by osquery. In a standard configuration these are available on the filesystem of the Fleet server at the path configurable by [`--filesystem_status_log_file`](../Deploying/Configuration.md#filesystem-status-log-file). This defaults to `/tmp/osquery_status`. The host will output a status log each time it executes the query.

### Why does the same query come back faster sometimes?

Don't worry, this behavior is expected; it's part of how osquery works.

Fleet and osquery work together by communicating with heartbeats. Depending on how close the next heartbeat is, Fleet might return results a few seconds faster or slower.
>By the way, to get around a phenomena called the "thundering herd problem", these heartbeats aren't exactly the same number of seconds apart each time. osquery implements a "splay", a few ± milliseconds that are added to or subtracted from the heartbeat interval to prevent these thundering herds. This helps prevent situations where many thousands of devices might unnecessarily attempt to communicate with the Fleet server at exactly the same time. (If you've ever used Socket.io, a similar phenomena can occur with that tool's automatic WebSocket reconnects.)

### What happens if I have a query on a team policy and I also have it scheduled to run separately?

Both queries will run as scheduled on applicable hosts. If there are any hosts that both the scheduled run and the policy apply to, they will be queried twice.

### Why aren’t my live queries being logged?

Live query results are never logged to the filesystem of the Fleet server. See [Where are my query results?](#where-are-my-query-results).

### Why does my query work locally with osquery but not in Fleet?

If you're seeing query results using `osqueryi` but not through Fleet, the most likely culprit is a permissions issue. Check out the [osquery docs](https://osquery.readthedocs.io/en/stable/deployment/process-auditing/#full-disk-access) for more details and instructions for setting up Full Disk Access. 

### Can I use the Fleet API to fetch results from a scheduled query pack?

You cannot. Scheduled query results are logged to whatever logging plugin you have configured and are not stored in the Fleet DB.

However, the Fleet API exposes a significant amount of host information via the [`api/v1/fleet/hosts`](../Using-Fleet/REST-API.md#list-hosts) and the [`api/v1/fleet/hosts/{id}`](../Using-Fleet/REST-API.md#get-host) API endpoints. The `api/v1/fleet/hosts` [can even be configured to return additional host information](https://github.com/fleetdm/fleet/blob/9fb9da31f5462fa7dda4819a114bbdbc0252c347/docs/1-Using-Fleet/2-fleetctl-CLI.md#fleet-configuration-options).

For example, let's say you want to retrieve a host's OS version, installed software, and kernel version:

Each host’s OS version is available using the `api/v1/fleet/hosts` API endpoint. [Check out the API documentation for this endpoint](../Using-Fleet/REST-API.md#list-hosts).

The ability to view each host’s installed software was released behind a feature flag in Fleet 3.11.0 and called Software inventory. [Check out the feature flag documentation for instructions on turning on Software inventory in Fleet](../Deploying/Configuration.md#feature-flags).

Once the Software inventory feature is turned on, a list of a specific host’s installed software is available using the `api/v1/fleet/hosts/{id}` endpoint. [Check out the documentation for this endpoint](../Using-Fleet/REST-API.md#get-host).

It’s possible in Fleet to retrieve each host’s kernel version, using the Fleet API, through `additional_queries`. The Fleet configuration options YAML file includes an `additional_queries` property that allows you to append custom query results to the host details returned by the `api/v1/fleet/hosts` endpoint. [Check out an example configuration file with the additional_queries field](../Using-Fleet/fleetctl-CLI.md#fleet-configuration-options).

### How do I automatically add hosts to packs when the hosts enroll to Fleet?

You can accomplish this by adding specific labels as targets of your pack. First, identify an already existing label or create a new label that will include the hosts you intend to enroll to Fleet. Next, add this label as a target of the pack in the Fleet UI.

When your hosts enroll to Fleet, they will become a member of the label and, because the label is a target of your pack, these hosts will automatically become targets of the pack.

You can also do this by setting the `targets` field in the [YAML configuration file](../Using-Fleet/fleetctl-CLI.md#query-packs) that manages the packs that are added to your Fleet instance.

### How do I automatically assign a host to a team when it enrolls with Fleet?

[Team enroll secrets](../Using-fleet/Adding-hosts.md#automatically-adding-hosts-to-a-team) allow you to automatically assign a host to a team.

### Why my host is not updating a policy's response.

The following are reasons why a host may not be updating a policy's response:

* The policy's query includes tables that are not compatible with this host's platform. For example, if your policy's query contains the [`apps` table](https://osquery.io/schema/5.0.1/#apps), which is only compatible on hosts running macOS, this policy will not update its response if this host is running Windows or Linux. 

* The policy's query includes invalid SQL syntax. If your policy's query includes invalid syntax, this policy will not update its response. You can check the syntax of your query by heading to the **Queries** page, selecting your query, and then selecting "Save."

### What should I do if my computer is showing up as an offline host?

If your device is showing up as an offline host in the Fleet instance, and you're sure that the computer has osquery running, we recommend trying the following:

* Try un-enrolling and re-enrolling the host. You can do this by uninstalling osquery on the host and then enrolling your device again using one of the [recommended methods](../Using-Fleet/Adding-hosts.md).
* Restart the `fleetctl preview` docker containers.
* Uninstall and reinstall Docker.

### Fleet preview fails with Invalid interpolation. What should I do?

If you tried running `fleetctl preview` and you get the following error:

```
fleetctl preview
Downloading dependencies into /root/.fleet/preview...
Pulling Docker dependencies...
Invalid interpolation format for "fleet01" option in service "services": "fleetdm/fleet:${FLEET_VERSION:-latest}"

Failed to run docker-compose
```

You are probably running an old version of Docker. You should download the installer for your platform from https://docs.docker.com/compose/install/

### How does Fleet deal with IP duplication?

Fleet relies on UUIDs so any overlap with host IP addresses should not cause a problem. The only time this might be an issue is if you are running a query that involves a specific IP address that exists in multiple locations as it might return multiple results - [Fleet's teams feature](../Using-Fleet/Teams.md) can be used to restrict queries to specific hosts.

### Can Orbit run alongside osquery?

Yes, Orbit can be run alongside osquery. The osquery instance that Orbit runs uses its own database directory that is stored within the Orbit directory.

### Can I disable auto-updates for Orbit?

Yes, auto-updates can be disabled by passing `--disable-updates` as a flag when running `fleetctl package` to generate your installer (easy) or by deploying a modified systemd file to your hosts (more complicated). We'd recommend the flag:

```
fleetctl package --fleetctl package --type=deb --fleet-url=https://localhost:8080 --enroll-secret=superRandomSecret --disable-updates
```
### Can I bundle osquery extensions into Orbit?

This isn't supported yet, but we're working on it! 

### What happens to osquery logs if my Fleet server or my logging destination is offline?

If Fleet can't send logs to the destination, it will return an error to osquery. This causes osquery to retry sending the logs. The logs will then be stored in osquery's internal buffer until they are sent successfully, or they get expired if the `buffered_log_max`(defaults to 1,000,000 logs) is exceeded. Check out the [Remote logging buffering section](https://osquery.readthedocs.io/en/latest/deployment/remote/#remote-logging-buffering) on the osquery docs for more on this behavior.

### How does Fleet work with osquery extensions?

Any extension table available in a host enrolled to Fleet can be queried by Fleet. Note that the "compatible with" message may show an error because it won't know your extension table, but the query will still work, and Fleet will gracefully ignore errors from any incompatible hosts.

### Why do I see "Unknown Certificate Error" when adding hosts to my dev server?

If you are using a self-signed certificate on `localhost`, add the  `--insecure` flag when building your installation packages:

```
fleetctl package --fleetctl package --type=deb --fleet-url=https://localhost:8080 --enroll-secret=superRandomSecret --insecure
```

### Can I hide known vulnerabilities that I feel are insignificant?

This isn't currently supported, but we're working on it! You can track that issue [here](https://github.com/fleetdm/fleet/issues/3152).

### Can I create reports based on historical data in Fleet?

Currently, Fleet only stores the current state of your hosts (when they last communicated with Fleet). The best way at the moment to maintain historical data would be to use the [REST API](../Using-Fleet/REST-API.md) or the [`fleetctl` CLI](../Using-Fleet/fleetctl-CLI.md) to retrieve it manually. Then save the data you need to your schedule. 

### When do I need fleetctl vs the REST API vs the Fleet UI?

[fleetctl](https://fleetdm.com/docs/using-fleet/fleetctl-cli) is great for users that like to do things in a terminal (like iTerm on a Mac). Lots of tech folks are real power users of the terminal. It is also helpful for automating things like deployments.

The [REST API](https://fleetdm.com/docs/using-fleet/rest-api) is somewhat similar to fleetctl, but it tends to be used more by other computer programs rather than human users (although humans can use it too). For example, our [Fleet UI](https://fleetdm.com/docs/using-fleet/rest-api) talks to the server via the REST API. Folks can also use the REST API if they want to build their own programs that talk to the Fleet server.

The [Fleet UI](https://fleetdm.com/docs/using-fleet/fleet-ui) is built for human users to make interfacing with the Fleet server user-friendly and visually appealing. It also makes things simpler and more accessible to a broader range of users. 

### Why can't I run queries with `fleetctl` using a new API-only user?

In versions prior to Fleet 4.13, a password reset is needed before a new API-only user can perform queries. You can find detailed instructions for setting that up [here](https://github.com/fleetdm/fleet/blob/a1eba3d5b945cb3339004dd1181526c137dc901c/docs/Using-Fleet/fleetctl-CLI.md#reset-the-password).

### Why am I getting an error about self-signed certificates when running `fleetctl preview`?

If you are trying to run `fleetctl preview` and seeing errors about self-signed certificates, the most likely culprit is that you're behind a corporate proxy server and need to [add the proxy settings to Docker](https://docs.docker.com/network/proxy/) so that the container created by `fleetctl preview` is able to connect properly. 

### Can I audit actions taken in Fleet?

The [REST API `activities` endpoint](../Using-Fleet/REST-API.md#activities) provides a full breakdown of actions taken on packs, queries, policies, and teams (Available in Fleet Premium) through the UI, the REST API, or `fleetctl`.  

### How often is the software inventory updated?

By default, Fleet will query hosts for software inventory hourly. If you'd like to set a different interval, you can update the [periodicity](../Deploying/Configuration.md#periodicity) in your vulnerabilities configuration. 

### Can I group results from multiple hosts?

There are a few ways you can go about getting counts of hosts that meet specific criteria using the REST API. You can use [`GET /api/v1/fleet/hosts`](../Using-Fleet/REST-API.md#list-hosts) or the [`fleetctl` CLI](../Using-Fleet/fleetctl-CLI.md#available-commands) to gather a list of all hosts and then work with that data however you'd like. For example, you could retrieve all hosts using `fleetctl get hosts` and then use `jq` to pull out the data you need. The following example would give you a count of hosts by their OS version:

```
$ fleetctl get hosts --json | jq '.spec .os_version' | sort | uniq -c

   1 "CentOS Stream 8.0.0"
   2 "Ubuntu 20.4.0"
   1 "macOS 11.5.2"
   1 "macOS 11.6.3"
   1 "macOS 12.1.0"
   3 "macOS 12.2.1"
   3 "macOS 12.3.0"
   6 "macOS 12.3.1"
```

### Will updating fleetctl lead to loss of data in fleetctl preview?

No, you won't experience data loss when you update fleetctl. Note that you can run `fleetctl preview --tag v#.#.#` if you want to run Preview on a previous version. Just replace # with the version numbers of interest.

### Can I disable usage statistics via the config file or a CLI flag?
Apart from an admin [disabling usage](https://fleetdm.com/docs/using-fleet/usage-statistics#disable-usage-statistics) statistics on the Fleet UI, you can edit your `fleet.yml` config file to disable usage statistics. Look for the `server_settings` in your `fleet.yml` and set `enable_analytics: false`. Do note there is no CLI flag option to disable usage statistics at this time.

### How do I downgrade from Fleet Premium to Fleet Free?

If you'd like to renew your Fleet Premium license key, please contact us [here](https://fleetdm.com/company/contact).

How to downgrade from Fleet Premium to Fleet Free:

First, back up your users and update all team-level users to global users:

1. Run the `fleetctl get user_roles > user_roles.yml` command. Save the `user_roles.yml` file so
   that, if you choose to upgrade later, you can restore user roles.
2. Head to the **Settings > Users** page in the Fleet UI.
3. For each user that has any team listed under the **Teams** column, select **Actions > Edit**,
   then select
   **Global user**, and then select **Save**. If a user shouldn't have global access, delete this user.

Next, move all team-level scheduled queries to the global level:
1. Head to the **Schedule** page in the Fleet UI.
2. For each scheduled query that belongs to a team, copy the name in the **Query** column, select
   **All teams** in the top dropdown, select **Schedule a query**, past the name in the **Select
   query** field, choose the frequency, and select **Schedule**.
3. Delete each scheduled query that belongs to a team because they will no longer run on any hosts
   following the downgrade process.

Next, move all team level policies to the global level:
1. Head to the **Policies** page in the Fleet UI.
2. For each policy that belongs to a team, copy the **Name**, **Description**, **Resolve**,
  and **Query**. Then, select **All teams** in the top dropdown, select **Add a policy**, select
  **create your own policy**, paste each item in the appropriate field, and select **Save**.
3. Delete each policy that belongs to a team because they will no longer run on any hosts
following the downgrade process.

Next, back up your teams:
1. Run the `fleetctl get teams > teams.yml` command. Save the `teams.yml` file so
that, if you choose to upgrade later, you can restore teams.
2. Head to the **Settings > Teams** page in the Fleet UI.
3. Delete all teams. This will move all hosts to the global level.

Lastly, remove your Fleet Premium license key:
1. Remove your license key from your Fleet configuration. Documentation on where the license key is
   located in your configuration is [here](https://fleetdm.com/docs/deploying/configuration#license).
2. Restart your Fleet server.

### If I use a software orchestration tool (Ansible, Chef, Puppet, etc.) to manage agent options, do I have to apply the same options in the Fleet UI?

No. The agent options set using your software orchestration tool will override the default agent options that appear in the **Settings > Organization settings > Global agent options** page. On this page, if you hit the **Save** button, the options that appear in the Fleet UI will override the agent options set using your software orchestration.

### How can I uninstall Orbit/Fleet Desktop?
To uninstall Orbit/Fleet Desktop, follow the below instructions for your Operating System.

#### MacOS
Run the Orbit [cleanup script](https://github.com/fleetdm/fleet/blob/main/orbit/tools/cleanup/cleanup_macos.sh)

#### Windows
Use the "Add or remove programs" dialog to remove Orbit.

#### Ubuntu
Run `sudo apt remove fleet-osquery -y`

#### CentOS
Run `sudo rpm -e fleet-osquery-X.Y.Z.x86_64`

## Deploying

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
- [What MySQL versions are supported?](#what-mysql-versions-are-supported)
- [What are the MySQL user access requirements?](#what-are-the-mysql-user-requirements)
- [Does Fleet support MySQL replication?](#does-fleet-support-mysql-replication)
- [What is duplicate enrollment and how do I fix it?](#what-is-duplicate-enrollment-and-how-do-i-fix-it)
- [How long are osquery enroll secrets valid?](#how-long-are-osquery-enroll-secrets-valid)
- [Should I use multiple enroll secrets?](#should-i-use-multiple-enroll-secrets)
- [How can enroll secrets be rotated?](#how-can-enroll-secrets-be-rotated)
- [What API endpoints should I expose to the public internet?](#what-api-endpoints-should-i-expose-to-the-public-internet)
- [Will my older version of Fleet work with Redis 6?](#will-my-older-version-of-fleet-work-with-redis-6)

### How do I get support for working with Fleet?

For bug reports, please use the [Github issue tracker](https://github.com/fleetdm/fleet/issues).

For questions and discussion, please join us in the #fleet channel of [osquery Slack](https://fleetdm.com/slack).

### Can multiple instances of the Fleet server be run behind a load-balancer?

Yes. Fleet scales horizontally out of the box as long as all of the Fleet servers are connected to the same MySQL and Redis instances.

Note that osquery logs will be distributed across the Fleet servers.

Read the [performance documentation](../Using-Fleet/Monitoring-Fleet.md#fleet-server-performance) for more.

### Why aren't my osquery agents connecting to Fleet?

This can be caused by a variety of problems. The best way to debug is usually to add `--verbose --tls_dump` to the arguments provided to `osqueryd` and look at the logs for the server communication.

#### Common problems

- `Connection refused`: The server is not running, or is not listening on the address specified. Is the server listening on an address that is available from the host running osquery? Do you have a load balancer that might be blocking connections? Try testing with `curl`.
- `No node key returned`: Typically this indicates that the osquery client sent an incorrect enroll secret that was rejected by the server. Check what osquery is sending by looking in the logs near this error.
- `certificate verify failed`: See [How do I fix "certificate verify failed" errors from osqueryd](#how-do-i-fix-certificate-verify-failed-errors-from-osqueryd).
- `bad record MAC`: When generating your certificate for your Fleet server, ensure you set the hostname to the FQDN or the IP of the server. This error is common when setting up Fleet servers and accepting defaults when generating certificates using `openssl`.

### How do I fix "certificate verify failed" errors from osqueryd?

Osquery requires that all communication between the agent and Fleet are over a secure TLS connection. For the safety of osquery deployments, there is no (convenient) way to circumvent this check.

- Try specifying the path to the full certificate chain used by the server using the `--tls_server_certs` flag in `osqueryd`. This is often unnecessary when using a certificate signed by an authority trusted by the system, but is mandatory when working with self-signed certificates. In all cases it can be a useful debugging step.
- Ensure that the CNAME or one of the Subject Alternate Names (SANs) on the certificate matches the address at which the server is being accessed. If osquery connects via `https://localhost:443`, but the certificate is for `https://fleet.example.com`, the verification will fail.
- Is Fleet behind a load-balancer? Ensure that if the load-balancer is terminating TLS, this is the certificate provided to osquery.
- Does the certificate verify with `curl`? Try `curl -v -X POST https://fleetserver:port/api/v1/osquery/enroll`.

### What do I need to do to change the Fleet server TLS certificate?

If the both the existing and new certificates verify with osquery's default root certificates (such as a certificate issued by a well-known Certificate Authority) and no certificate chain was deployed with osquery, there is no need to deploy a new certificate chain.

If osquery has been deployed with the full certificate chain (using `--tls_server_certs`), deploying a new certificate chain is necessary to allow for verification of the new certificate.

Deploying a certificate chain cannot be done centrally from Fleet.

### How do I use a proxy server with Fleet?

Seeing your proxy's requests fail with an error like `DEPTH_ZERO_SELF_SIGNED_CERT`)?
To get your proxy server's HTTP client to work with a local Fleet when using a self-signed cert, disable SSL / self-signed verification in the client.

The exact solution to this depends on the request client you are using. For example, when using Node.js ± Sails.js, you can work around this in the requests you're sending with `await sails.helpers.http.get()` by lifting your app with the `NODE_TLS_REJECT_UNAUTHORIZED` environment variable set to `0`:

```
NODE_TLS_REJECT_UNAUTHORIZED=0 sails console
```

### I'm only getting partial results from live queries

Redis has an internal buffer limit for pubsub that Fleet uses to communicate query results. If this buffer is filled, extra data is dropped. To fix this, we recommend disabling the buffer size limit. Most installs of Redis should have plenty of spare memory to not run into issues. More info about this limit can be found [here](https://redis.io/topics/clients#:~:text=Pub%2FSub%20clients%20have%20a,64%20megabyte%20per%2060%20second.) and [here](https://raw.githubusercontent.com/redis/redis/unstable/redis.conf) (search for client-output-buffer-limit).

We recommend a config like the following:

```
client-output-buffer-limit pubsub 0 0 60
```

### When do I need to deploy a new enroll secret to my hosts?

Osquery provides the enroll secret only during the enrollment process. Once a host is enrolled, the node key it receives remains valid for authentication independent from the enroll secret.

Currently enrolled hosts do not necessarily need enroll secrets updated, as the existing enrollment will continue to be valid as long as the host is not deleted from Fleet and the osquery store on the host remains valid. Any newly enrolling hosts must have the new secret.

Deploying a new enroll secret cannot be done centrally from Fleet.

### How do I migrate hosts from one Fleet server to another (eg. testing to production)?

Primarily, this would be done by changing the `--tls_hostname` and enroll secret to the values for the new server. In some circumstances (see [What do I need to do to change the Fleet server TLS certificate?](#what-do-i-need-to-do-to-change-the-fleet-server-tls-certificate)) it may be necessary to deploy a new certificate chain configured with `--tls_server_certs`.

These configurations cannot be managed centrally from Fleet.

### What do I do about "too many open files" errors?

This error usually indicates that the Fleet server has run out of file descriptors. Fix this by increasing the `ulimit` on the Fleet process. See the `LimitNOFILE` setting in the [example systemd unit file](../Deploying/Configuration.md#runing-with-systemd) for an example of how to do this with systemd.

Some deployments may benefit by setting the [`--server_keepalive`](../Deploying/Configuration.md#server-keepalive) flag to false.

This was also seen as a symptom of a different issue: if you're deploying on AWS on T type instances, there are different scenarios where the activity can increase and the instances will burst. If they run out of credits, then they'll stop processing leaving the file descriptors open.

### I upgraded my database, but Fleet is still running slowly. What could be going on?

This could be caused by a mismatched connection limit between the Fleet server and the MySQL server that prevents Fleet from fully utilizing the database. First [determine how many open connections your MySQL server supports](https://dev.mysql.com/doc/refman/8.0/en/too-many-connections.html). Now set the [`--mysql_max_open_conns`](../Deploying/Configuration.md#mysql-max-open-conns) and [`--mysql_max_idle_conns`](../Deploying/Configuration.md#mysql-max-idle-conns) flags appropriately.

### Why am I receiving a database connection error when attempting to "prepare" the database?

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

### Is Fleet available as a SaaS product?

No. Currently, Fleet is only available for self-hosting on premises or in the cloud.

### What MySQL versions are supported?

Fleet is tested with MySQL 5.7.21 and 8.0.28. Newer versions of MySQL 5.7 and MySQL 8 typically work well. AWS Aurora requires at least version 2.10.0. Please avoid using MariaDB or other MySQL variants that are not officially supported. Compatibility issues have been identified with MySQL variants and these may not be addressed in future Fleet releases.

### What are the MySQL user requirements?

The user `fleet prepare db` (via environment variable `FLEET_MYSQL_USERNAME` or command line flag `--mysql_username=<username>`) uses to interact with the database needs to be able to create, alter, and drop tables as well as the ability to create temporary tables.

### Does Fleet support MySQL replication?

You can deploy MySQL or Maria any way you want. We recommend using managed/hosted mysql so you don't have to think about it, but you can think about it more if you want. Read replicas are supported. You can read more about MySQL configuration [here](../Deploying/Configuration.md#my-sql).

### What is duplicate enrollment and how do I fix it?

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

Find more information about [host identifiers here](../Deploying/Configuration.md#osquery-host-identifier).

### How long are osquery enroll secrets valid?

Enroll secrets are valid until you delete them.

### Should I use multiple enroll secrets?

That is up to you! Some organizations have internal goals around rotating secrets. Having multiple secrets allows some of them to work at the same time the rotation is happening.
Another reason you might want to use multiple enroll secrets is to use a certain enroll secret to
auto-enroll hosts into a specific team (Fleet Premium).

### How can enroll secrets be rotated?

Rotating enroll secrets follows this process:

1. Add a new secret.
2. Transition existing clients to the new secret. Note that existing clients may not need to be
   updated, as the enroll secret is not used by already enrolled clients.
3. Remove the old secret.

To do this with `fleetctl` (assuming the existing secret is `oldsecret` and the new secret is `newsecret`):

Begin by retrieving the existing secret configuration:

```
$ fleetctl get enroll_secret
---
apiVersion: v1
kind: enroll_secret
spec:
  secrets:
  - created_at: "2021-11-17T00:39:50Z"
    secret: oldsecret
```

Apply the new configuration with both secrets:

```
$ echo '
---
apiVersion: v1
kind: enroll_secret
spec:
  secrets:
  - created_at: "2021-11-17T00:39:50Z"
    secret: oldsecret
  - secret: newsecret
' > secrets.yml
$ fleetctl apply -f secrets.yml
```

Now transition clients to using only the new secret. When the transition is completed, remove the
old secret:

```
$ echo '
---
apiVersion: v1
kind: enroll_secret
spec:
  secrets:
  - secret: newsecret
' > secrets.yml
$ fleetctl apply -f secrets.yml
```

At this point, the old secret will no longer be accepted for new enrollments and the rotation is
complete.

A similar process may be followed for rotating team-specific enroll secrets. For teams, the secrets
are managed in the team yaml.

### How do I resolve an "unknown column" error when upgrading Fleet?

The `unknown column` error typically occurs when the database migrations haven't been run during the upgrade process.

Check out the [documentation on running database migrations](../Deploying/Upgrading-Fleet.md#running-database-migrations) to resolve this issue.

### What API endpoints should I expose to the public internet?

If you would like to manage hosts that can travel outside your VPN or intranet we recommend only exposing the "/api/v1/osquery" endpoint to the public internet.

If you would like to use the fleetctl CLI from outside of your network, the following endpoints will also need to be exposed for `fleetctl`:

- /api/setup
- /api/v1/setup
- /api/osquery/*
- /api/latest/fleet/*
- /api/v1/fleet/*

### What is the minimum version of MySQL required by Fleet?

Fleet requires at least MySQL version 5.7.

### How do I migrate from Fleet Free to Fleet Premium?

To migrate from Fleet Free to Fleet Premium, once you get a Fleet license, set it as a parameter to `fleet serve` either as an environment variable using `FLEET_LICENSE_KEY` or in the Fleet's config file. See [here](../Deploying/Configuration.md#license) for more details. Note: You don't need to redeploy Fleet after the migration.

### Will my older version of Fleet work with Redis 6?

Most likely, yes! While we'd definitely recommend keeping Fleet up to date in order to take advantage of new features and bug patches, most legacy versions should work with Redis 6. Just keep in mind that we likely haven't tested your particular combination so that you may run into some unforeseen hiccups. 

## Contributing

- [Make errors](#make-errors)
  - [`dep: command not found`](#dep-command-not-found)
  - [`undefined: Asset`](#undefined-asset)
- [How do I connect to the MailHog simulated mail server?](#how-do-i-connect-to-the-mailhog-simulated-mail-server)
- [Adding hosts for testing](#adding-hosts-for-testing)


### Enrolling in multiple Fleet servers
Enrolling your device with more than one Fleet server is not currently possible.  Multiple install roots are useful for the development of Fleet itself but complex to maintain.  While this has some value for Fleet contributors, there is currently no active effort to add and maintain support for multiple enrollments from the same device.

### Make errors

#### `dep: command not found`

```
/bin/bash: dep: command not found
make: *** [.deps] Error 127
```

If you get the above error, you need to add `$GOPATH/bin` to your PATH. A quick fix is to run `export PATH=$GOPATH/bin:$PATH`.
See the Go language documentation for [workspaces](https://golang.org/doc/code.html#Workspaces) and [GOPATH](https://golang.org/doc/code.html#GOPATH) for more in-depth documentation.

#### `undefined: Asset`

```
server/fleet/emails.go:90:23: undefined: Asset
make: *** [fleet] Error 2
```

If you get an `undefined: Asset` error, it is likely because you did not run `make generate` before `make build`. See [Building Fleet](../Contributing/Building-Fleet.md) for additional documentation on compiling the `fleet` binary.

### Adding hosts for testing

The `osquery` directory contains a docker-compose.yml and additional configuration files to start containerized osquery agents.

To start osquery, first retrieve the "Enroll secret" from Fleet (by clicking the "Add New Host") button in the Fleet dashboard, or with `fleetctl get enroll-secret`).

```
cd tools/osquery
ENROLL_SECRET=<copy from fleet> docker-compose up
```
