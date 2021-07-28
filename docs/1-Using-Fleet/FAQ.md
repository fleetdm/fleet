# Using Fleet FAQ

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
- [How do I resolve an "unknown column" error when upgrading Fleet?](#how-do-i-resolve-an-unknown-column-error-when-upgrading-fleet)

## What do I need to do to switch from Kolide Fleet to FleetDM Fleet?

The upgrade from kolide/fleet to fleetdm/fleet works the same as any minor version upgrade has in the past.

Minor version upgrades in Kolide Fleet often included database migrations and the recommendation to back up the database before migrating. The same goes for FleetDM Fleet versions.

To migrate from Kolide Fleet to FleetDM Fleet, please follow the steps outlined in the [Updating Fleet section](./8-Updating-Fleet.md) of the documentation.

## Has anyone stress tested Fleet? How many clients can the Fleet server handle?

Fleet has been stress tested to 150,000 online hosts and 400,000 total enrolled hosts. Production deployments exist with over 100,000 hosts and numerous production deployments manage tens of thousands of hosts.

It’s standard deployment practice to have multiple Fleet servers behind a load balancer. However, typically the MySQL database is the performance bottleneck and a single Fleet server can handle tens of thousands of hosts.

## Can I target my hosts using their enroll secrets?

No, currently, there’s no way to retrieve the name of the enroll secret with a query. This means that there's no way to create a label using your hosts' enroll secrets and then use this label as a target for queries or query packs.

Typically folks will use some other unique identifier to create labels that distinguish each type of device. As a workaround, [Fleet's manual labels](./2-fleetctl-CLI.md#host-labels) provide a way to create groups of hosts without a query. These manual labels can then be used as targets for queries or query packs.

There is, however, a way to accomplish this even though the answer to the question remains "no": Teams. As of Fleet v4.0.0, you can group hosts in Teams either by enrolling them with a team specific secret, or by transferring hosts to a team. One the hosts you want to target are part of a team, you can create a query and target the team in question.

## How often do labels refresh? Is the refresh frequency configurable?

The update frequency for labels is configurable with the [—osquery_label_update_interval](../2-Deploying/2-Configuration.md#osquery_label_update_interval) flag (default 1 hour).

## How do I revoke the authorization tokens for a user?

Authorization tokens are revoked when the “require password reset” action is selected for that user. User-initiated password resets do not expire the existing tokens.

## How do I monitor the performance of my queries?

Fleet can live query the `osquery_schedule` table. Performing this live query allows you to get the performance data for your scheduled queries. Also consider scheduling a query to the `osquery_schedule` table to get these logs into your logging pipeline.

## How do I monitor a Fleet server?

Fleet provides standard interfaces for monitoring and alerting. See the [Monitoring Fleet](./6-Monitoring-Fleet.md) documentation for details.

## Why is the “Add User” button disabled?

The “Add User” button is disabled if SMTP (email) has not been configured for the Fleet server. Currently, there is no way to add new users without email capabilities.

One way to hack around this is to use a simulated mailserver like [Mailhog](https://github.com/mailhog/MailHog). You can retrieve the email that was “sent” in the Mailhog UI, and provide users with the invite URL manually.

## Can I disable password-based authentication in the Fleet UI?

Some folks like to enforce users with SAML SSO enabled to login only via the SSO and not via password.

There is no option in the Fleet UI for disabling password-based authentication.
However, users that have SSO enabled in Fleet will not be able to log in via password-based authentication.

If a user has SSO enabled, the Login page in the Fleet UI displays the “Email” and “Password” fields but on attempted password-based login, this user will receive an “Authentication failed” message.

## Where are my query results?

### Live Queries

Live query results (executed in the web UI or `fleetctl query`) are pushed directly to the UI where the query is running. The results never go to a file unless you as the user manually save them.

### Scheduled Queries

Scheduled query results (queries that are scheduled to run in Packs) are typically sent to the Fleet server, and will be available on the filesystem of the server at the path configurable by [`--osquery_result_log_file`](../2-Deploying/2-Configuration.md#osquery_result_log_file). This defaults to `/tmp/osquery_result`.

It is possible to configure osqueryd to log query results outside of Fleet. For results to go to Fleet, the `--logger_plugin` flag must be set to `tls`.

### What are my options for storing the osquery logs?

Folks typically use Fleet to ship logs to data aggregation systems like Splunk, the ELK stack, and Graylog.

The [logger configuration options](../2-Deploying/2-Configuration.md#osquery_status_log_plugin) allow you to select the log output plugin. Using the log outputs you can route the logs to your chosen aggregation system.

### Troubleshooting

Expecting results, but not seeing anything in the logs?

- Try scheduling a query that always returns results (eg. `SELECT * FROM time`).
- Check whether the query is scheduled in differential mode. If so, new results will only be logged when the result set changes.
- Ensure that the query is scheduled to run on the intended platforms, and that the tables queried are supported by those platforms.
- Use live query to `SELECT * FROM osquery_schedule` to check whether the query has been scheduled on the host.
- Look at the status logs provided by osquery. In a standard configuration these are available on the filesystem of the Fleet server at the path configurable by [`--filesystem_status_log_file`](../2-Deploying/2-Configuration.md#filesystem_status_log_file). This defaults to `/tmp/osquery_status`. The host will output a status log each time it executes the query.

## Why aren’t my live queries being logged?

Live query results are never logged to the filesystem of the Fleet server. See [Where are my query results?](#where-are-my-query-results).

## Can I use the Fleet API to fetch results from a scheduled query pack?

You cannot. Scheduled query results are logged to whatever logging plugin you have configured and are not stored in the Fleet DB.

However, the Fleet API exposes a significant amount of host information via the [`api/v1/fleet/hosts`](./3-REST-API.md#list-hosts) and the [`api/v1/fleet/hosts/{id}`](./3-REST-API.md#get-host) API endpoints. The `api/v1/fleet/hosts` [can even be configured to return additional host information](https://github.com/fleetdm/fleet/blob/9fb9da31f5462fa7dda4819a114bbdbc0252c347/docs/1-Using-Fleet/2-fleetctl-CLI.md#fleet-configuration-options).

As an example, let's say you want to retrieve a host's OS version, installed software, and kernel version:

Each host’s OS version is available using the `api/v1/fleet/hosts` API endpoint. [Check out the API documentation for this endpoint](./3-REST-API.md#list-hosts).

The ability to view each host’s installed software was released behind a feature flag in Fleet 3.11.0 and called Software inventory. [Check out the feature flag documentation for instructions on turning on Software inventory in Fleet](../2-Deploying/2-Configuration.md#feature-flags).

Once the Software inventory feature is turned on, a list of a specific host’s installed software is available using the `api/v1/fleet/hosts/{id}` endpoint. [Check out the documentation for this endpoint](./3-REST-API.md#get-host).

It’s possible in Fleet to retrieve each host’s kernel version, using the Fleet API, through `additional_queries`. The Fleet configuration options yaml file includes an `additional_queries` property that allows you to append custom query results to the host details returned by the `api/v1/fleet/hosts` endpoint. [Check out an example configuration file with the additional_queries field](./2-fleetctl-CLI.md#fleet-configuration-options).

## How do I automatically add hosts to packs when the hosts enroll to Fleet?

You can accomplish this by adding specific labels as targets of your pack. First, identify an already existing label or create a new label that will include the hosts you intend to enroll to Fleet. Next, add this label as a target of the pack in the Fleet UI.

When your hosts enroll to Fleet, they will become a member of the label and, because the label is a target of your pack, these hosts will automatically become targets of the pack.

You can also do this by setting the `targets` field in the [YAML configuration file](./2-fleetctl-CLI.md#query-packs) that manages the packs that are added to your Fleet instance.

## How do I resolve an "unknown column" error when upgrading Fleet?

The `unknown column` error typically occurs when the database migrations haven't been run during the upgrade process.

Check out the [documentation on running database migrations](./8-Updating-Fleet.md#running-database-migrations) to resolve this issue.
