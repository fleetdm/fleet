# Fleet usage statistics

Fleet Device Management Inc. periodically collects information about your instance.

> To disable usage statistics, [see here](#disable-usage-statistics).

## What is included in usage statistics in Fleet?

Below is the JSON payload that is sent to Fleet Device Management Inc:

```json
{
  "anonymousIdentifier": "9pnzNmrES3mQG66UQtd29cYTiX2+fZ4CYxDvh495720=",
  "fleetVersion": "x.x.x",
  "licenseTier": "free",
  "organization": "Fleet",
  "numHostsEnrolled": 999,
  "numUsers": 999,
  "numTeams": 999,
  "numPolicies": 999,
  "numQueries": 999,
  "numLabels": 999,
  "softwareInventoryEnabled": true,
  "vulnDetectionEnabled": true,
  "systemUsersEnabled": true,
  "hostsStatusWebHookEnabled": true,
  "mdmMacOsEnabled": true,
  "hostExpiryEnabled": true,
  "mdmWindowsEnabled": false,
  "liveQueryDisabled": false,
  "numWeeklyActiveUsers": 999,
  "numWeeklyPolicyViolationDaysActual": 999,
  "numWeeklyPolicyViolationDaysPossible": 999,
  "numSoftwareVersions": 999,
  "numHostSoftwares": 999,
  "numSoftwareTitles": 999,
  "numHostSoftwareInstalledPaths": 999,
  "numSoftwareCPEs": 999,
  "numSoftwareCVEs": 999,
  "numHostsNotResponding": 9,
  "aiFeaturesDisabled": true,
  "maintenanceWindowsEnabled": true,
  "maintenanceWindowsConfigured": true,
  "numHostsFleetDesktopEnabled": 999,
  "hostsEnrolledByOperatingSystem": {
    "darwin": [
      {
        "version": "macOS 12.3.1",
        "numEnrolled": 999
      },
      ...
    ],
    "windows": [
      {
        "version": "Microsoft Windows 10, version 21H2 (W)",
        "numEnrolled": 999
      },
      ...
    ],
    "ubuntu": [
      {
        "version": "Ubuntu 22.04 'Jammy Jellyfish' (LTS)",
        "numEnrolled": 999
      },
      ...
    ],
    "rhel": [
      {
        "version": "Red Hat Enterprise Linux 8.4.0",
        "numEnrolled": 999
      },
      ...
    ],
    "debian": [
      {
        "version": "Debian GNU/Linux 9.0.0",
        "numEnrolled": 999
      },
      ...
    ],
    "amzn": [
      {
        "version": "Amazon Linux 2.0.0",
        "numEnrolled": 999
      },
      ...
    ]
  },
  "hostsEnrolledByOrbitVersion": [
    {
      "version": "1.1.0",
      "numHosts": 999
    },
    ...
  ],
  "hostsEnrolledByOsqueryVersion": [
    {
      "version": "4.9.0",
      "numHosts": 999
    },
    ...
  ],
  "storedErrors": [
    {
      "count": 3,
      "loc": [
        "github.com/fleetdm/fleet/v4/server/example.example:12",
        "github.com/fleetdm/fleet/v4/server/example.example:130",
      ]
    },
    ...
  ]
}
```

Statistics contain no personal information about any particular device or person.

For Fleet Free instances, usage statistics are anonymous. The "organization" property is reported as "unknown."

Sending Usage statistics from your Fleet Free instance is optional and can be disabled.

Note: Usage statistics are not optional for Fleet Premium instances.

## Why should we enable usage statistics?

Help make Fleet better! Fleet has wide adoption, but limited avenues for quantifying this. We need a way of measuring whether the enhancements and new features we ship are actually working.

Every time we ship a Fleet release without usage statistics, it's like launching a shiny, expensive new rocket into space without any way to find out what happens to it. Up until now, we've relied heavily on talking to users and working closely with customers and other community members. That's helped a lot! But it doesn't give us visibility into the problems other users might be having.

Insights about Fleet version adoption helps the team be more efficient when planning upgrade guides, release notes, and future security notices for users running vulnerable software versions.

## Disable usage statistics

Users with the Admin role can disable usage statistics.

To disable usage statistics:

1. In the top navigation, navigate to **Settings > Organization settings**.

2. Scroll to the "Usage statistics" section.

3. Uncheck the "Enable usage statistics" checkbox and then select "Update settings."

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2024-08-13">
<meta name="articleTitle" value="Fleet usage statistics">
<meta name="description" value="Learn about Fleet's usage statistics and what information is collected.">
