# Automations

You can configure automations in Fleet to send a webhook request if a certain condition is met.

[Vulnerability automations](#policy-automations) send a webhook request if a new vulnerability (CVE) is
detected on at least one host.

[Policy automations](#policy-automations) send a webhook request if a policy is newly failing on at
least one host.

[Host status automations](#host-status-automations) send a webhook request if a configured
percentage of hosts have not checked in to Fleet for a configured number of days.

## Vulnerability automations

Vulnerability automations send a webhook request if a new vulnerability (CVE) is
found on at least one host.

> Note that a CVE is "new" if it was published to the national vulnerability (NVD) database within
> the last 2 days.

Fleet sends these webhook requests once every hour. If two new vulnerabilities are detected
within the hour, two
webhook requests are sent. This interval can be updated with the [`vulnerabilities_periodicity` configuration option](../Deploying/Configuration.md#periodicity).

Example webhook payload:

```
POST https://server.com/example
```

```json
{
  "timestamp": "0000-00-00T00:00:00Z",
  "vulnerability": {
    "cve": "CVE-2014-9471",
    "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2014-9471",
    "hosts_affected": [
      {
        "id": 1,
        "hostname": "macbook-1",
        "url": "https://fleet.example.com/hosts/1"
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

## Policy automations

Policy automations send a webhook request if a policy is newly failing on at
least one host. 

> Note that a policy is "newly failing" if a host updated its response from "no response" to "failing"
> or from "passing" to "failing." 

Fleet sends these webhook requests once per day. If two policies are newly failing
within the day, two webhook requests are sent. This interval can be updated with the `webhook_settings.interval`
configuration option using the [`config` yaml document](./configuration-files/README.md#organization-settings) and the `fleetctl apply` command.

Example webhook payload:

```
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

To enable policy automations, navigate to **Policies > Manage automations** in the Fleet UI.

## Host status automations

Host status automations send a webhook request if a configured
percentage of hosts have not checked in to Fleet for a configured number of days.

Fleet sends these webhook requests once per day. This interval can be updated with the `webhook_settings.interval`
configuration option using the [`config` yaml document](./configuration-files/README.md#organization-settings) and the `fleetctl apply` command.

Example webhook payload:

```
POST https://server.com/example
```

```json
{
  "text": "More than X% of your hosts have not checked into Fleet           
           for more than X days. You’ve been sent this message  
           because the Host status webhook is enabeld in your Fleet 
           instance.",
  "data": {
    "unseen_hosts": 1,
    "total_hosts": 2,
    "days_unseen": 3,
  }
}
```

To enable and configure host status automations, navigate to **Settings > Organization settings > Host
status webhook** in the Fleet UI.

<meta name="pageOrderInSection" value="1300">