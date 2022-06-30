# Usage statistics

```
ℹ️  In Fleet 4.0, Usage statistics were introduced.
```

Fleet Device Management Inc. periodically collects information about your instance.

## What is included in usage statistics in Fleet?

- The usage data that Fleet collects includes the **installed Fleet version** and the **number of enrolled hosts** for your Fleet instance. Below is the JSON payload that is sent to Fleet Device Management Inc:

```json
{
  "anonymousIdentifier": "9pnzNmrES3mQG66UQtd29cYTiX2+fZ4CYxDvh495720=",
  "fleetVersion": "x.x.x",
  "licenseTier": "free",
  "numHostsEnrolled": 999,
  "numUsers": 999,
  "numTeams": 999,
  "numPolicies": 999,
  "numLabels": 999,
  "softwareInventoryEnabled": true,
  "vulnDetectionEnabled": true,
  "systemUsersEnabled": true,
  "hostStatusWebhookEnabled": true,
  "organization": "Fleet",
}
```

- For Fleet Free instances, usage statistics are anonymous. The "organization" property is reported as "unknown."

- Sending Usage statistics from your Fleet instance is optional and can be disabled.

## Why should we enable usage statistics?

- Help make Fleet better! Fleet has wide adoption, but limited avenues for quantifying this. We need a way of measuring whether the enhancements and new features we ship are actually working.

- Every time we ship a Fleet release without usage statistics, it's like launching a shiny, expensive new rocket into space without any way to find out what happens to it. Up until now, we've relied heavily on talking to users and working closely with customers and other community members. That's helped a lot! But it doesn't give us visibility into the problems other users might be having.

- Insights about Fleet version adoption helps the team be more efficient when planning upgrade guides, release notes, and future security notices for users running vulnerable software versions.

### Why does Fleet collect my Fleet version?

In the future, we can notify you about future upgrades to Fleet.

### Why does Fleet collect a count of the hosts I have enrolled to Fleet?

In the future, we can notify you about methods to improve performance of your Fleet. The performance improvements we suggest will depend on the number of hosts you have enrolled.

## Disable usage statistics

Users with the Admin role can disable usage statistics.

To disable usage statistics:

1. In the top navigation, navigate to **Settings > Organization settings**.

2. Scroll to the "Usage statistics" section.

3. Uncheck the "Enable usage statistics" checkbox and then select "Update settings."

<meta name="pageOrderInSection" value="1100">
