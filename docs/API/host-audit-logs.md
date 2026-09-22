# Host audit logs

_Available in Fleet Premium._

Fleet can send a webhook request every time an activity linked to one of a fleet's hosts is created. Use this to send a fleet's host activities to your SIEM or trigger automations, without receiving activities for every fleet.

To see a host's activities in the Fleet UI, go to the host's details page and select **Activity > Past**.

This webhook sends the same payload format as [Global audit logs](./global-audit-logs.md), filtered to activities linked to the fleet's hosts. See that page for the full list of activity types and their fields.

## Configure

Host audit logs are configured per fleet, using the `host_activities_webhook` object under `webhook_settings`. Configure it from the Hosts page in the Fleet UI, the [fleets API](./rest-api.md#update-fleet), or [Fleet's GitOps YAML](https://fleetdm.com/docs/configuration/yaml-files#host-activities-webhook):

```yaml
name: Workstations
team_settings:
  webhook_settings:
    host_activities_webhook:
      enable_host_activities_webhook: true
      destination_url: https://example.org/webhook_handler
```

This can also be configured for "Unassigned" hosts, in `unassigned.yml`.

## Limitations

MDM command results, shown via **Show MDM commands** on the host details page, are not activities and don't trigger this webhook.

<meta name="title" value="Host audit logs">
<meta name="pageOrderInSection" value="90">
<meta name="description" value="Learn how to receive a webhook for activities linked to a fleet's hosts.">
