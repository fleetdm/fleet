# Using Fleet FAQ
- [Has anyone stress tested Fleet? How many clients can the Fleet server handle?](#has-anyone-stress-tested-fleet-how-many-clients-can-the-fleet-server-handle)
- [How often do labels refresh? Is the refresh frequency configurable?](#how-often-do-labels-refresh-is-the-refresh-frequency-configurable)
- [How do I revoke the authorization tokens for a user?](#how-do-i-revoke-the-authorization-tokens-for-a-user)
- [How do I monitor the performance of my queries?](#how-do-i-monitor-the-performance-of-my-queries)
- [How do I monitor a Fleet server?](#how-do-i-monitor-a-fleet-server)
- [Why is the “Add User” button disabled?](#why-is-the-add-user-button-disabled)
- [Where are my query results?](#where-are-my-query-results)
- [Why aren’t my live queries being logged?](#why-arent-my-live-queries-being-logged)

## Has anyone stress tested Fleet? How many clients can the Fleet server handle?

Fleet has been stress tested to 150,000 online hosts and 400,000 total enrolled hosts. There are numerous production deployments in the thousands, in the tens of thousands of hosts range, and there are production deployments in the high tens of thousands of hosts range. 

It’s standard deployment practice to have multiple Fleet servers behind a load balancer. However, typically the MySQL database is the bottleneck and an individual Fleet server can handle tens of thousands of hosts.

## How often do labels refresh? Is the refresh frequency configurable?

The update frequency for labels is configurable with the [—osquery_label_update_interval](https://github.com/fleetdm/fleet/blob/master/docs/2-Deployment/2-Configuration.md#osquery_label_update_interval) flag (default 1 hour).

## How do I revoke the authorization tokens for a user?

Authorization tokens are revoked when the “require password reset” action is selected for that user. User-initiated password resets do not expire the existing tokens.

## How do I monitor the performance of my queries?

Fleet can live query the `osquery_schedule` table. Performing this live query allows you to get the performance data for your scheduled queries. Also consider scheduling a query to the `osquery_schedule` table to get these logs into your logging pipeline.

## How do I monitor a Fleet server?

Fleet provides standard interfaces for monitoring and alerting. See the [Monitoring Fleet](./5-Monitoring-Fleet.md) documentation for details.


## Why is the “Add User” button disabled?

The “Add User” button is disabled if SMTP (email) has not been configured for the Fleet server. Currently, there is no way to add new users without email capabilities.

One way to hack around this is to use a simulated mailserver like [Mailhog](https://github.com/mailhog/MailHog). You can retrieve the email that was “sent” in the Mailhog UI, and provide users with the invite URL manually.

## Where are my query results?

### Live Queries

Live query results (executed in the web UI or `fleetctl query`) are pushed directly to the UI where the query is running. The results never go to a file unless you as the user manually save them.

### Scheduled Queries

Scheduled query results (queries that are scheduled to run in Packs) are typically sent to the Fleet server, and will be available on the filesystem of the server at the path configurable by [`--osquery_result_log_file`](../2-Deployment/2-Configuration.md#osquery_result_log_file). This defaults to `/tmp/osquery_result`.

It is possible to configure osqueryd to log query results outside of Fleet. For results to go to Fleet, the `--logger_plugin` flag must be set to `tls`.

### What are my options for storing the osquery logs?

Folks typically use Fleet to ship logs to data aggregation systems like Splunk, the ELK stack, and Graylog. 

The [logger configuration options](https://github.com/fleetdm/fleet/blob/master/docs/2-Deployment/2-Configuration.md#osquery_status_log_plugin) allow you to select the log output plugin. Using the log outputs you can route the logs to your chosen aggregation system.

### Troubleshooting

Expecting results, but not seeing anything in the logs?

- Try scheduling a query that always returns results (eg. `SELECT * FROM time`).
- Check whether the query is scheduled in differential mode. If so, new results will only be logged when the result set changes.
- Ensure that the query is scheduled to run on the intended platforms, and that the tables queried are supported by those platforms.
- Use live query to `SELECT * FROM osquery_schedule` to check whether the query has been scheduled on the host.
- Look at the status logs provided by osquery. In a standard configuration these are available on the filesystem of the Fleet server at the path configurable by [`--filesystem_status_log_file`](../2-Deployment/2-Configuration.md#filesystem_status_log_file). This defaults to `/tmp/osquery_status`. The host will output a status log each time it executes the query.

## Why aren’t my live queries being logged?

Live query results are never logged to the filesystem of the Fleet server. See [Where are my query results?](#where-are-my-query-results).