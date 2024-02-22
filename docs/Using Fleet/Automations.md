# Automations

You can configure Fleet to trigger an automation if a certain condition is met. Automations in Fleet can be configured to send a webhook request to a specified URL or to create a ticket in Jira or Zendesk.

[Vulnerability automations](#vulnerability-automations) are triggered if a new vulnerability (CVE) is
detected on at least one host.

[Policy automations](#policy-automations) are triggered if a policy is newly failing on at
least one host.

[Host status automations](#host-status-automations) are triggered if a configured
percentage of hosts have not checked in to Fleet for a configured number of days.

## Vulnerability automations

Vulnerability automations are triggered if Fleet detects a new vulnerability (CVE) on at least one host. 

> Note that a CVE is treated as "new" by Fleet if it was published to the national vulnerability database (NVD) within the preceding 30 days by default. This setting can be changed through the [`recent_vulnerability_max_age` configuration option](https://fleetdm.com/docs/deploying/configuration#recent-vulnerability-max-age).

Fleet can be configured either to send a webhook request or to create a ticket in Jira or Zendesk. Fleet checks whether to trigger vulnerability automations once per hour by default. This period can be changed through the [`vulnerabilities_periodicity` configuration option](https://fleetdm.com/docs/deploying/configuration#periodicity). 

Once a CVE has been detected on any host, automations are not triggered if the CVE is detected on other hosts in subsequent periods. If the CVE has been remediated on all hosts, an automation may be triggered if the CVE is detected subsequently so long as the CVE is treated as "new" by Fleet. 

For webhook automations, if a new CVE is detected on more than one host during the same period that the initial detection occurred, a separate webhook request is triggered for each host by default. This behavior can be configured instead to group hosts into batched webhook requests through the [`host_batch_size` configuration option](https://fleetdm.com/docs/using-fleet/configuration-files#webhook-settings-vulnerabilities-webhook-host-batch-size). 

Example webhook payload:

```http
POST https://server.com/example
```

```json
{
  "timestamp": "0000-00-00T00:00:00Z",
  "vulnerability": {
    "cve": "CVE-2014-9471",
    "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2014-9471",
    "epss_probability": 0.7, // Premium feature only
    "cvss_score": 5.7, // Premium feature only
    "cisa_known_exploit": true, // Premium feature only
    "cve_published": "2020-10-28T00:00:00Z", // Premium feature only
    "cve_description": "The parse_datetime function in GNU coreutils allows remote attackers to cause a denial of service (crash) or possibly execute arbitrary code via a crafted date string, as demonstrated by the \"--date=TZ=\"123\"345\" @1\" string to the touch or date command.", // Premium feature only
    "hosts_affected": [
      {
        "id": 1,
        "hostname": "macbook-1",
        "url": "https://fleet.example.com/hosts/1",
        "software_installed_paths": ["/usr/lib/some-path"],
      },
      {
        "id": 2,
        "hostname": "macbook-2",
        "url": "https://fleet.example.com/hosts/2"
      }
    ]
  }
}
```


For ticket automations, only one ticket per CVE is created even if a CVE is detected on multiple hosts.

Follow the steps below to configure Jira or Zendesk as a ticket destination:

1. In the top bar of the Fleet UI, select your avatar and then **Settings**.
2. Select **Integrations > Add integration**.
3. Under **Ticket destination** select **Jira** or select **Zendesk**.
4. Enter your ticket destination's credentials.
5. In the top bar, select **Software > Manage automations**.
6. Select **Enable vulnerability automations** and choose **Ticket**.
7. Under **Ticket destination**, select your ticket destination and select **Save**.

## Policy automations

Policy automations are triggered if a policy is newly failing on at least one host. Policy automations are triggered separately for each failing policy.

> Note that a policy is "newly failing" if a host updated its response from "no response" to "failing" or from "passing" to "failing."

Fleet can be configured either to send a webhook request or to create a ticket in Jira or Zendesk. Fleet checks whether to trigger policy automations once per day by default. This interval can be updated with the `webhook_settings.interval` configuration option using the [`config` YAML document](https://fleetdm.com/docs/using-fleet/configuration-files#organization-settings) and the `fleetctl apply` command. Note that this interval currently configures both host status and failing policy automations. This interval applies to both creating tickets for failing policies as well as webhooks requests.

For webhooks automations, if a policy is newly failing on more than one host during the same period, a separate webhook request is triggered for each host by default. This behavior can be configured instead to group hosts into batched webhook requests through the [`host_batch_size` configuration option](https://fleetdm.com/docs/using-fleet/configuration-files#webhook-settings-failing-policies-webhook-host-batch-size).

Example webhook payload:

```http
POST https://server.com/example
```

```json
{
  "timestamp": "0000-00-00T00:00:00Z",
  "policy": {
    "id": 1,
      "name": "Is Gatekeeper enabled?",
      "query": "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
      "description": "Checks if gatekeeper is enabled on macOS devices.",
      "author_id": 1,
      "author_name": "John",
      "author_email": "john@example.com",
      "resolution": "Turn on Gatekeeper feature in System Preferences.",
      "passing_host_count": 2000,
      "failing_host_count": 300
  },
  "hosts": [
    {
      "id": 1,
      "hostname": "macbook-1",
      "url": "https://fleet.example.com/hosts/1"
    },
    {
      "id": 2,
      "hostname": "macbbook-2",
      "url": "https://fleet.example.com/hosts/2"
    }
  ]
}
```

For ticket automations, a single ticket is created per newly failed policy (i.e., multiple tickets are not created if a policy is newly failing on more than one host during the same period).

Follow the steps below to configure Jira or Zendesk as a ticket destination:

1. In the top bar of the Fleet UI, select your avatar and then **Settings**.
2. Select **Integrations > Add integration**.
3. Under **Ticket destination** select **Jira** or select **Zendesk**.
4. Enter your ticket destination's credentials.
5. In the top bar, select **Policies > Manage automations**.
6. Select **Enable policy automations**, check the policies you'd like to listen to, and choose **Ticket**.
7. Under **Ticket destination**, select your ticket destination and select **Save**.

## Host status automations

Host status automations send a webhook request if a configured percentage of hosts have not checked in to Fleet for a configured number of days. This can be customized [globally](https://fleetdm.com/docs/configuration/configuration-files#organization-settingss) or [per-team](https://fleetdm.com/docs/configuration/configuration-files#teams).

Fleet sends these webhook requests once per day by default. This interval can be updated with the `webhook_settings.interval` [configuration option](https://fleetdm.com/docs/configuration/configuration-files#organization-settings).  Note that this interval currently configures both host status and failing policy automations.

Example webhook payload:

```http
POST https://server.com/example
```

```json
{
  "text": "More than X% of your hosts have not checked into Fleet
           for more than X days. You’ve been sent this message
           because the Host status webhook is enabeld in your Fleet
           instance.",
  "data": {
    "unseen_hosts": 3,
    "total_hosts": 12,
    "days_unseen": 3,
    "team_id": 123,
    "host_ids": [1, 2, 3]
  }
}
```

To enable and configure host status automations, navigate to **Settings > Organization settings > Host
status webhook** in the Fleet UI.

<meta name="pageOrderInSection" value="1300">
<meta name="description" value="Configure Fleet automations to trigger webhooks or create tickets in Jira and Zendesk for vulnerability, policy, and host status events.">
<meta name="navSection" value="Vuln management">
