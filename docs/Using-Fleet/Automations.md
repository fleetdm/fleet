# Automations

You can configure automations in Fleet to send a webhook request if a certain condition is met.

[Policy automations](#policy-automations) allow you to receive a list of hosts that recently failed a policy.

[Host status automations](#host-status-automations) allow you to receive a notification when a portion of your hosts go offline.

## Policy automations

Policy automations are triggered once per day, for each policy, if one or more hosts has recently
failed a policy.

Policy automations can be turned on or off for each policy.

For each policy, Fleet will send a webhook request with a list of the hosts that recently failed the
policy. 

Once per day, the Fleet server updates this list of hosts. The hosts that failed after the last
attempted webhook request are added to the list. The hosts that were included in the last successful
webhook request are removed from the list.

To enable policy automations, navigate to **Policies > Manage automations** in the Fleet UI.

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

## Host status automations

Host status automations are triggered once per day if a configured percentage of hosts has not
checked in to Fleet for a configured number of days.

To enable and configure host status automations, navigate to **Settings > Organization settings > Host
status webhook** in the Fleet UI.

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

<meta name="pageRank" value="13">