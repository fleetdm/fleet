https://github.com/fleetdm/fleet/pull/4818/files# Using Fleet FAQ

- [What do I need to do to switch from Kolide Fleet to FleetDM Fleet?](#waht-do-i-need-to-do-to-switch-from-kolide-fleet-to-fleetdm-fleet)
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
- [Why my host is not updating a policy's response.](#why-my-host-is-not-updating-a-policys-response)
- [What should I do if my computer is showing up as an offline host?](#what-should-i-do-if-my-computer-is-showing-up-as-an-offline-host)
- [How does Fleet deal with IP duplication?](#how-does-fleet-deal-with-ip-duplication)
- [Can Orbit run alongside osquery?](#can-orbit-run-alongside-osquery)
- [Can I disable auto updates for Orbit?](#can-i-disable-auto-updates-for-orbit)
- [Can I bundle osquery extensions into Orbit?](#can-i-bundle-osquery-extensions-into-orbit)
- [How does Fleet work with osquery extensions?](#how-does-fleet-work-with-osquery-extensions)
- [Why am I seeing "unknown certificate error" when adding hosts to my dev server?](#why-am-i-seeing-"unknown-certificate-error"-when-adding-hosts-to-my-dev-server)
- [Can I hide known vulnerabilities that I feel are insignificant?](#can-i-hide-known-vulnerabilities-that-i-feel-are-insignificant)
- [Can I create reports based on historical data in Fleet?](#can-i-create-reports-based-on-historical-data-in-fleet)

## What do I need to do to switch from Kolide Fleet to FleetDM Fleet?

The upgrade from kolide/fleet to fleetdm/fleet works the same as any minor version upgrade has in the past.

Minor version upgrades in Kolide Fleet often included database migrations and the recommendation to back up the database before migrating. The same goes for the new Fleet.

To migrate from `kolide/fleet` to the new Fleet, please follow the steps outlined in the [Upgrading Fleet section](../Deploying/Upgrading-Fleet.md) of the documentation.

## Has anyone stress tested Fleet? How many hosts can the Fleet server handle?

Fleet has been stress tested to 150,000 online hosts and 400,000 total enrolled hosts. Production deployments exist with over 100,000 hosts and numerous production deployments manage tens of thousands of hosts.

It’s standard deployment practice to have multiple Fleet servers behind a load balancer. However, typically the MySQL database is the performance bottleneck and a single Fleet server can handle tens of thousands of hosts.

## Can I target my hosts using their enroll secrets?

No, currently, there’s no way to retrieve the name of the enroll secret with a query. This means that there's no way to create a label using your hosts' enroll secrets and then use this label as a target for queries or query packs.

Typically folks will use some other unique identifier to create labels that distinguish each type of device. As a workaround, [Fleet's manual labels](./fleetctl-CLI.md#host-labels) provide a way to create groups of hosts without a query. These manual labels can then be used as targets for queries or query packs.

There is, however, a way to accomplish this even though the answer to the question remains "no": Teams. As of Fleet v4.0.0, you can group hosts in Teams either by enrolling them with a team specific secret, or by transferring hosts to a team. One the hosts you want to target are part of a team, you can create a query and target the team in question.

## How often do labels refresh? Is the refresh frequency configurable?

The update frequency for labels is configurable with the [—osquery_label_update_interval](../Deploying/Configuration.md#osquery-label-update-interval) flag (default 1 hour).

## How do I revoke the authorization tokens for a user?

Authorization tokens are revoked when the “require password reset” action is selected for that user. User-initiated password resets do not expire the existing tokens.

## How do I monitor the performance of my queries?

Fleet can live query the `osquery_schedule` table. Performing this live query allows you to get the performance data for your scheduled queries. Also consider scheduling a query to the `osquery_schedule` table to get these logs into your logging pipeline.

## How do I monitor a Fleet server?

Fleet provides standard interfaces for monitoring and alerting. See the [Monitoring Fleet](./Monitoring-Fleet.md) documentation for details.

## Why is the “Add User” button disabled?

The “Add User” button is disabled if SMTP (email) has not been configured for the Fleet server. Currently, there is no way to add new users without email capabilities.

One way to hack around this is to use a simulated mailserver like [Mailhog](https://github.com/mailhog/MailHog). You can retrieve the email that was “sent” in the Mailhog UI, and provide users with the invite URL manually.

## Can I disable password-based authentication in the Fleet UI?

Some folks like to enforce users with SAML SSO enabled to login only via the SSO and not via password.

There is no option in the Fleet UI for disabling password-based authentication.
However, users that have SSO enabled in Fleet will not be able to log in via password-based authentication.

If a user has SSO enabled, the Login page in the Fleet UI displays the “Email” and “Password” fields but on attempted password-based login, this user will receive an “Authentication failed” message.

## Where are my query results?

### Live queries

Live query results (executed in the web UI or `fleetctl query`) are pushed directly to the UI where the query is running. The results never go to a file unless you as the user manually save them.

### Scheduled queries

Scheduled query results (queries that are scheduled to run in Packs) are typically sent to the Fleet server, and will be available on the filesystem of the server at the path configurable by [`--osquery_result_log_file`](../Deploying/Configuration.md#osquery-result-log-file). This defaults to `/tmp/osquery_result`.

It is possible to configure osqueryd to log query results outside of Fleet. For results to go to Fleet, the `--logger_plugin` flag must be set to `tls`.

### What are my options for storing the osquery logs?

Folks typically use Fleet to ship logs to data aggregation systems like Splunk, the ELK stack, and Graylog.

The [logger configuration options](../Deploying/Configuration.md#osquery-status-log-plugin) allow you to select the log output plugin. Using the log outputs you can route the logs to your chosen aggregation system.

### Troubleshooting

Expecting results, but not seeing anything in the logs?

- Try scheduling a query that always returns results (eg. `SELECT * FROM time`).
- Check whether the query is scheduled in differential mode. If so, new results will only be logged when the result set changes.
- Ensure that the query is scheduled to run on the intended platforms, and that the tables queried are supported by those platforms.
- Use live query to `SELECT * FROM osquery_schedule` to check whether the query has been scheduled on the host.
- Look at the status logs provided by osquery. In a standard configuration these are available on the filesystem of the Fleet server at the path configurable by [`--filesystem_status_log_file`](../Deploying/Configuration.md#filesystem-status-log-file). This defaults to `/tmp/osquery_status`. The host will output a status log each time it executes the query.

## Why does the same query come back faster sometimes?

Don't worry, this behavior is expected; it's part of how osquery works.

Fleet and osquery work together by communicating with heartbeats. Depending on how close the next heartbeat is, Fleet might return results a few seconds faster or slower.
>By the way, to get around a phenomena called the "thundering herd problem", these heartbeats aren't exactly the same number of seconds apart each time. osquery implements a "splay", a few ± milliseconds that are added to or subtracted from the heartbeat interval to prevent these thundering herds. This helps prevent situations where many thousands of devices might unnecessarily attempt to communicate with the Fleet server at exactly the same time. (If you've ever used Socket.io, a similar phenomena can occur with that tool's automatic WebSocket reconnects.)

## What happens if I have a query on a team policy and I also have it scheduled to run separately?

Both queries will run as scheduled on applicable hosts. If there are any hosts that both the scheduled run and the policy apply to, they will be queried twice.

## Why aren’t my live queries being logged?

Live query results are never logged to the filesystem of the Fleet server. See [Where are my query results?](#where-are-my-query-results).

## Can I use the Fleet API to fetch results from a scheduled query pack?

You cannot. Scheduled query results are logged to whatever logging plugin you have configured and are not stored in the Fleet DB.

However, the Fleet API exposes a significant amount of host information via the [`api/v1/fleet/hosts`](./REST-API.md#list-hosts) and the [`api/v1/fleet/hosts/{id}`](./REST-API.md#get-host) API endpoints. The `api/v1/fleet/hosts` [can even be configured to return additional host information](https://github.com/fleetdm/fleet/blob/9fb9da31f5462fa7dda4819a114bbdbc0252c347/docs/1-Using-Fleet/2-fleetctl-CLI.md#fleet-configuration-options).

As an example, let's say you want to retrieve a host's OS version, installed software, and kernel version:

Each host’s OS version is available using the `api/v1/fleet/hosts` API endpoint. [Check out the API documentation for this endpoint](./REST-API.md#list-hosts).

The ability to view each host’s installed software was released behind a feature flag in Fleet 3.11.0 and called Software inventory. [Check out the feature flag documentation for instructions on turning on Software inventory in Fleet](../Deploying/Configuration.md#feature-flags).

Once the Software inventory feature is turned on, a list of a specific host’s installed software is available using the `api/v1/fleet/hosts/{id}` endpoint. [Check out the documentation for this endpoint](./REST-API.md#get-host).

It’s possible in Fleet to retrieve each host’s kernel version, using the Fleet API, through `additional_queries`. The Fleet configuration options yaml file includes an `additional_queries` property that allows you to append custom query results to the host details returned by the `api/v1/fleet/hosts` endpoint. [Check out an example configuration file with the additional_queries field](./fleetctl-CLI.md#fleet-configuration-options).

## How do I automatically add hosts to packs when the hosts enroll to Fleet?

You can accomplish this by adding specific labels as targets of your pack. First, identify an already existing label or create a new label that will include the hosts you intend to enroll to Fleet. Next, add this label as a target of the pack in the Fleet UI.

When your hosts enroll to Fleet, they will become a member of the label and, because the label is a target of your pack, these hosts will automatically become targets of the pack.

You can also do this by setting the `targets` field in the [YAML configuration file](./fleetctl-CLI.md#query-packs) that manages the packs that are added to your Fleet instance.

## How do I automatically assign a host to a team when it enrolls with Fleet?

[Team enroll secrets](./Teams.md#enroll-hosts-to-a-team) allow you to automatically assign a host to a team.

## Why my host is not updating a policy's response.

The following are reasons why a host may not be updating a policy's response:

* The policy's query includes tables that are not compatible with this host's platform. For example, if your policy's query contains the [`apps` table](https://osquery.io/schema/5.0.1/#apps), which is only compatible on hosts running macOS, this policy will not update its response if this host is running Windows or Linux. 

* The policy's query includes invalid SQL syntax. If your policy's query includes invalid syntax, this policy will not update its response. You can check the syntax of your query by heading to the **Queries** page, selecting your query, and then selecting "Save."

## What should I do if my computer is showing up as an offline host?

If your device is showing up as an offline host in the Fleet instance, and you're sure that the computer has osquery running, we recommend trying the following:

* Try un-enrolling and re-enrolling the host. You can do this by uninstalling osquery on the host and then enrolling your device again using one of the [recommended methods](./Adding-hosts.md).
* Restart the `fleetctl preview` docker containers.
* Uninstall and reinstall Docker.

## Fleet preview fails with Invalid interpolation. What should I do?

If you tried running fleet preview and you get the following error:

```
fleetctl preview
Downloading dependencies into /root/.fleet/preview...
Pulling Docker dependencies...
Invalid interpolation format for "fleet01" option in service "services": "fleetdm/fleet:${FLEET_VERSION:-latest}"

Failed to run docker-compose
```

You are probably running an old version of Docker. You should download the installer for your platform from https://docs.docker.com/compose/install/

## How does Fleet deal with IP duplication?

Fleet relies on UUIDs so any overlap with host IP addresses should not cause a problem. The only time this might be an issue is if you are running a query that involves a specific IP address that exists in multiple locations as it might return multiple results - [Fleet's teams feature](./Teams.md) can be used to restrict queries to specific hosts.

## Can Orbit run alongside osquery?

Yes, Orbit can be run alongside osquery. The osquery instance that Orbit runs uses its own database directory that is stored within the Orbit directory.

## Can I disable auto-updates for Orbit?

Yes, auto-updates can be disabled either by passing `--disable-updates` as a flag when running `fleetctl package` to generate your installer (easy) or by deploying a modified systemd file to your hosts (more complicated). We'd recommend the flag:

```
fleetctl package --fleetctl package --type=deb --fleet-url=https://localhost:8080 --enroll-secret=superRandomSecret --disable-updates
```
## Can I bundle osquery extensions into Orbit?

This isn't supported yet, but we're working on it! 

## What happens to osquery logs if my Fleet server or my logging destination is offline?

If Fleet can't send logs to the destination, it will return an error to osquery. This causes osquery to retry sending the logs. The logs will then be stored in osquery's internal buffer until they are sent successfully, or they get expired if the `buffered_log_max`(defaults to 1,000,000 logs) is exceeded. Check out the [Remote logging buffering section](https://osquery.readthedocs.io/en/latest/deployment/remote/#remote-logging-buffering) on the osquery docs for more on this behavior.

## How does Fleet work with osquery extensions?

Any extension table available in a host enrolled to Fleet can be queried by Fleet. Note that the "compatible with" message may show an error because it won't know your extension table, but the query will still work, and Fleet will gracefully ignore errors from any incompatible hosts.

## Why do I see "Unknown Certificate Error" when adding hosts to my dev server?

If you are using a self-signed certificate on `localhost`, add the  `--insecure` flag when building your installation packages:

```
fleetctl package --fleetctl package --type=deb --fleet-url=https://localhost:8080 --enroll-secret=superRandomSecret --insecure
```

## Can I hide known vulnerabilities that I feel are insignificant?

This isn't currently supported, but we're working on it! You can track that issue [here](https://github.com/fleetdm/fleet/issues/3152).

## Can I create reports based on historical data in Fleet?

Currently, Fleet only stores the current state of your hosts (when they last communicated with Fleet). The best way at the moment to maintain historical data would be to use the [REST API](./REST-API.md) or the [`fleetctl` CLI](./fleetctl-CLI.md) to retrieve it manually. Then save the data you need to your schedule. 

## When do I need fleetctl vs the REST API vs the Fleet UI?

[fleetctl](https://fleetdm.com/docs/using-fleet/fleetctl-cli) is great for users that like to do things in a terminal (like iTerm on a Mac). Lots of tech folks are real power users of the terminal. It is also helpful for automating things like deployments.

The [REST API](https://fleetdm.com/docs/using-fleet/rest-api) is somewhat similar to fleetctl, but it tends to be used more by other computer programs rather than human users (although humans can use it too). For example, our [Fleet UI](https://fleetdm.com/docs/using-fleet/rest-api) talks to the server via the REST API. Folks can also use the REST API if they want to build their own programs that talk to the Fleet server.

The [Fleet UI](https://fleetdm.com/docs/using-fleet/fleet-ui) is built for human users to make interfacing with the Fleet server user-friendly and visually appealing. It also makes things simpler and more accessible to a broader range of users. 
