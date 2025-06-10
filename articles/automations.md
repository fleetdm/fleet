# Automations

You can configure Fleet to automatically reserve time in your end users' calendars (maintenance
windows), trigger or send query results to webhooks, or create tickets.

To learn how to use Fleet's maintenance windows, head to this [article](https://fleetdm.com/announcements/fleet-in-your-calendar-introducing-maintenance-windows). 

## Activity automations

Activity automations are triggered when an activity happens in Fleet (queries, scripts, logins, etc). See our [Audit logs documentation](https://fleetdm.com/docs/using-fleet/audit-logs) for a list of all activity types.

You can automatically send activites to a webhook URL or a [log destination](https://fleetdm.com/docs/configuration/fleet-server-configuration#external-activity-audit-logging).

## Policy automations

Policy automations are triggered if a policy is newly failing on at least one host.

> Note that a policy is "newly failing" if a host updated its response from "no response" to "failing" or from "passing" to "failing."

Fleet checks whether to trigger policy automations once per day by default.

For webhooks, if a policy is newly failing on more than one host during the same period, a separate webhook request is triggered for each host by default.

For tickets, a single ticket is created per newly failed policy (i.e., multiple tickets are not
created if a policy is newly failing on more than one host during the same period).

## Query automations

Query automations let you send data gathered from macOS, Windows, and Linux hosts to a log
destination. Data is sent according to a query's interval.

### Webhook

Results from scheduled queries can be written to an arbitrary external webhook of your choosing.
First, follow the [configuration docs](https://fleetdm.com/docs/deploying/configuration#webhook).
Then in the UI:

1. Navigate to the **Queries** page, select the relevant team, and click **Manage automations**
2. In the modal that opens, confirm that you see "Log destination: Webhook", and when you hover over
   "Webhook", you see "Each time a query runs, the data is sent via webhook to:
   <target_result_url>"
3. Select the queries that you want to send data to this webhook
4. Click **Save**

Results from the selected scheduled queries will be sent to the configured results URL. *Not configurable per-query.*

### Amazon Kinesis Data Firehose

See [the log destination guide](https://fleetdm.com/guides/log-destinations#amazon-kinesis-data-firehose)

## Vulnerability automations

Vulnerability automations are triggered if Fleet detects a new vulnerability (CVE) on at least one host. 

> Note that Fleet treats a CVE as "new" if it was published within the preceding 30 days by default. This setting can be changed through the [`recent_vulnerability_max_age` configuration option](https://fleetdm.com/docs/deploying/configuration#recent-vulnerability-max-age).

Fleet checks whether to trigger vulnerability automations once per hour by default. This period can be changed through the [`vulnerabilities_periodicity` configuration option](https://fleetdm.com/docs/deploying/configuration#periodicity). 

Once a CVE has been detected on any host, automations are not triggered if the CVE is detected on other hosts in subsequent periods. If the CVE has been remediated on all hosts, an automation may be triggered if the CVE is detected subsequently so long as the CVE is treated as "new" by Fleet. 

For webhooks, if a new CVE is detected on more than one host during the same period that the initial detection occurred, a separate webhook request is triggered for each host by default.

## Host status automations

Host status automations send a webhook request if a configured percentage of hosts have not checked in to Fleet for a configured number of days.

Fleet sends these webhook requests once per day by default.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2024-07-03">
<meta name="articleTitle" value="Automations">
<meta name="description" value="Configure Fleet automations to trigger webhooks or create tickets in Jira and Zendesk for vulnerability, policy, and host status events.">
