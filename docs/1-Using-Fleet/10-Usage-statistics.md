# Usage statistics

```
ℹ️  In Fleet 4.0, Usage statistics were introduced.
```

Fleet Device Management Inc. periodically collects anonymous information about your instance.

### What is included in usage statistics in Fleet?

- The usage data that Fleet collects includes the **installed Fleet version** and the **number of enrolled hosts** for your Fleet instance. Below is an example JSON payload that is sent to Fleet Device Management Inc:

```json
{
  "anonymous_identifier": 1,
  "fleet_version": "x.x.x",
  "hosts_enrolled_count": 12345
}
```

- All statistics are anonymous and contain no personal information about any particular device, organization, or person.

- Sending Usage statistics from your Fleet instance is optional and can be disabled.

### Why should we enabled usage statistics?

- Fleet has wide adoption, but limited avenues for quantifying this. We need a way of measuring whether the enhancements and new features we ship are actually working.

- Every time we ship a Fleet release without usage statistics, it's like launching a shiny, expensive new rocket into space without any way to find out what happens to it. Up until now, we've relied heavily on talking to users and working closely with customers and other community members. That's helped a lot! But it doesn't give us visibility into the problems other users might be having.

- Insights about Fleet version adoption helps the team be more efficient when planning upgrade guides, release notes, and future security notices for users running vulnerable software versions.

#### Why does Fleet collect my Fleet version?

In the future, we can notify you about future upgrades to Fleet.

#### Why does Fleet collect a count of the hosts I have enrolled to Fleet?

In the future, we can notify you about methods to improve performance of your Fleet. The performance improvements we suggest will depend on the number of hosts you have enrolled.

### Disable usage statistics

Users with the Admin role can disabled usage statistics.

To disabled usage statistics:

1. In the top navigation, navigate to **Settings > Organization settings**.

2. Scroll to the "Usage statistics" section.

3. Uncheck the "Enable usage statistics" checkbox and then select "Update settings."