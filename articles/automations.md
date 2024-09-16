# Automations

You can configure Fleet to trigger automations that reserve time in your end users' calendars (maintenance windows), send webhooks, or to create tickets.

To learn how to use Fleet's maintenance windows, head to this [article](https://fleetdm.com/announcements/fleet-in-your-calendar-introducing-maintenance-windows). 

## Activity automations

Activity automations are triggered when an activity happens in Fleet (queries, scripts, logins, etc). See a list of all activities [here](https://fleetdm.com/docs/using-fleet/audit-logs).

You can automatically send activites to a webhook URL or a [log destination](https://fleetdm.com/docs/configuration/fleet-server-configuration#external-activity-audit-logging).

## Policy automations

Policy automations are triggered if a policy is newly failing on at least one host.

> Note that a policy is "newly failing" if a host updated its response from "no response" to "failing" or from "passing" to "failing."

Fleet checks whether to trigger policy automations once per day by default.

For webhooks, if a policy is newly failing on more than one host during the same period, a separate webhook request is triggered for each host by default.

For tickets, a single ticket is created per newly failed policy (i.e., multiple tickets are not created if a policy is newly failing on more than one host during the same period).

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
