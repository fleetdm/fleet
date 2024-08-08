# Seamless MDM migrations to Fleet

Typical MDM migrations on macOS require end-user interaction and result in a window of time in which the device is unmanaged. This has consequences from users being kicked off wifi due to certificate profile removal, compliance issues with unmanaged devices, and incomplete migrations. These concerns leave some organizations stuck on outdated MDM solutions that are no longer meeting their needs. There is a better way.

For customers with eligible MDM deployments, migration to Fleet is possible with no gap in management and without involving the end-user.

## Requirements

Note: Deployments that do not meet these seamless migration requirements can still migrate with the [standard MDM migration process](https://fleetdm.com/docs/using-fleet/mdm-migration-guide).

* Customer controls the DNS used in the MDM server enrollment (eg. devices are enrolled to `*.customerowneddomain.com`, not `*.mdmvendor.com`).
* Customer has access to the Apple Push Notification Service (APNS) certificate/key and SCEP certificate/key, or access to the MDM server database to extract these values.

These requirements are easily met in self-hosted open-source MDM solutions, and may be met with commercial solutions when the customer is self-hosting or otherwise controls the DNS.

Seamless migration may still be possible with control of DNS along with a copy of the original Certificate Signing Request (CSR) for the APNS certificate. If you are in this situation, please reach out to the Fleet team.

### Why?

Apple allows changing most values in profiles delivered by MDM, but the `ServerURL`, `CheckinURL`, and `PushTopic` cannot be changed without re-enrollment (and user actions). Control of DNS and the certificates allows the MDM to be swapped out without changing these.

## High-level process

The process outline is simple:

1. Configure Fleet with the APNS & SCEP certificates/keys.
2. Import database records letting Fleet know about the devices to be migrated.
3. Install fleetd on the devices (through the existing MDM).
4. Update DNS records to point devices to the Fleet server.


```mermaid
---
title: Before migration
---
flowchart LR
subgraph macOS Device
  mdmclient[MDM client]
end
mdmclient -- Routed by DNS <br> (mdm.example.com)-->oldmdm
oldmdm[Existing MDM Server]
mdmclient ~~~ fleet
fleet[Fleet Server]
```

```mermaid
---
title: After migration
---
flowchart LR
subgraph macOS Device
  mdmclient[MDM client]
end
oldmdm[Existing MDM Server]
mdmclient ~~~ oldmdm
mdmclient -- Routed by DNS <br> (mdm.example.com)-->fleet
fleet[Fleet Server]
```

<meta name="category" value="guides">
<meta name="authorFullName" value="Zach Wasserman">
<meta name="authorGitHubUsername" value="zwass">
<meta name="publishedOn" value="2024-08-08">
<meta name="articleTitle" value="Seamless MDM migrations to Fleet">
<meta name="articleImageUrl" value="../website/assets/images/articles/sysadmin-diaries-1600x900@2x.png">
