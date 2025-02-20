# Seamless macOS MDM migration

> NOTE: Please contact Fleet here https://fleetdm.com/contact or reach out to the Fleet Customer Success team if you are a current Fleet customer for consultation when considering this migration path. We'd love to help!

![Seamless macOS MDM migrations to Fleet](../website/assets/images/articles/seamless-mdm-migration-1600x900@2x.png)

Migrating macOS devices between Mobile Device Management (MDM) solutions is often fraught with challenges, including potential gaps in device management, user disruption, and compliance issues. Traditional MDM migrations typically require end-user interaction and leave devices unmanaged for a period, leading to problems like Wi-Fi disconnections due to certificate profile removal and incomplete migrations. These challenges can force organizations to stay with outdated MDM solutions that no longer meet their needs. But there’s a better way.

Seamless MDM migrations are now possible, allowing organizations to transition their macOS devices to Fleet without any downtime or end-user involvement. By leveraging Fleet, you can ensure that your devices remain fully managed and compliant throughout the migration process. This means no more gaps in management, no user disruptions, and a smoother path to a more modern and effective MDM solution.

This guide will walk you through the entire process of migrating your MDM deployment to Fleet. You’ll start by understanding the specific requirements for a seamless migration, followed by configuring Fleet with the necessary certificates and database records. The guide will then take you through the process of installing Fleet’s agent (`fleetd`) on your devices, updating domain (DNS) records to redirect devices to the Fleet server, and finally, decommissioning your old MDM server.

Throughout the guide, you’ll find practical advice and best practices to ensure a smooth transition with minimal risk. By the end, you’ll be equipped with the knowledge and tools to execute a seamless MDM migration to Fleet, ensuring that your organization’s devices are securely managed without the typical headaches associated with a traditional MDM switch.

## Requirements

> Deployments that do not meet these seamless migration requirements can still migrate with the [standard MDM migration process](https://fleetdm.com/docs/using-fleet/mdm-migration-guide).

* Customer owns the domain (DNS) used in the MDM enrollment profile (e.g. devices are enrolled to `*.customerowneddomain.com`, not `*.mdmvendor.com`).
* Customer has access to the Apple Push Notification Service (APNS) certificate/key and SCEP certificate/key, or access to the MDM server database to extract these values.

These requirements are easily met in self-hosted open-source MDM solutions and may be met with commercial solutions when the customer is self-hosting or otherwise controls the DNS.

Seamless migration may still be possible with control of DNS along with a copy of the original Certificate Signing Request (CSR) for the APNS certificate. If you are in this situation, please reach out to the Fleet team.

### Why?

Apple allows changing most values in profiles delivered by MDM, but the `ServerURL`, `CheckinURL`, and `PushTopic` cannot be changed without re-enrollment (and user actions). Control of DNS and the certificates allows the MDM to be swapped out without changing these.

## High-level process

1. Configure Fleet with the APNS & SCEP certificates/keys, path redirects, and SCEP renewal.
2. Import database records letting Fleet know about the devices to be migrated.
3. Configure controls (profiles, updates, etc.) in Fleet.
4. Install `fleetd` on the devices (through the existing MDM).
5. Update domain (DNS) records to point devices to the Fleet server.
6. Decommission the old server.

It is recommended to follow the entire process on a staging/test MDM instance and devices, then repeat for the production instance and devices.

[![Before migration](https://mermaid.ink/img/pako:eNpVUctuwjAQ_BVrT62URIaEvFRxqNKeSivBrZiDiTeJpdhGxqFQBN9eA23VXvY1o9lZ7RFqIxBKCMOQaSddjyV5xMZYJEq2ljtpNNNXtOnNR91x68jLnOntsPbwpiOK128LUuFO1sg0IUqoupeo3XJWzcitXDGNWjD9i5EwJHMzOBRkfSDV64I8rO2U3HlChHuuNj1GtVH3YTg1vfBTpm95-bSXWyd1Sy7qC7Q7tKu_wufzmTQ9orsY9mn5fIk_TAhAoVVcCn_z8WKXgetQIYPSlwIbPvSOAdMnT-WDM4uDrqF0dsAAho3gDivJ_eUKyob3Wz_dcP1uzL8eyiPsoRzTOBrHySim2SQtaBbAAco4S6NxTmmeZcUoLiZJfArg8ypAo5SOKC3iNM-LNMmTJAAU0hk7u32pNrqRrXdmzdB23xtPX3Gkloc?type=png)](https://mermaid.live/edit#pako:eNpVUctuwjAQ_BVrT62URIaEvFRxqNKeSivBrZiDiTeJpdhGxqFQBN9eA23VXvY1o9lZ7RFqIxBKCMOQaSddjyV5xMZYJEq2ljtpNNNXtOnNR91x68jLnOntsPbwpiOK128LUuFO1sg0IUqoupeo3XJWzcitXDGNWjD9i5EwJHMzOBRkfSDV64I8rO2U3HlChHuuNj1GtVH3YTg1vfBTpm95-bSXWyd1Sy7qC7Q7tKu_wufzmTQ9orsY9mn5fIk_TAhAoVVcCn_z8WKXgetQIYPSlwIbPvSOAdMnT-WDM4uDrqF0dsAAho3gDivJ_eUKyob3Wz_dcP1uzL8eyiPsoRzTOBrHySim2SQtaBbAAco4S6NxTmmeZcUoLiZJfArg8ypAo5SOKC3iNM-LNMmTJAAU0hk7u32pNrqRrXdmzdB23xtPX3Gkloc)

[![After migration](https://mermaid.ink/img/pako:eNpVUcFuwjAM_ZXIu2xSW7XQdaWakCYxTmOT4DayQ0jcNqJJUEgZDMG3L6Vs2g5JbL9n-9k5AjcCoYAwDKl20jVYkKfSoSVKVpY5aTTVF7BszCevmXXkZU71tl15eFMTxfjbgkxwJzlSTYgSijcStVvOJjPSmx9UoxZUm0Z4ePm8l1sndUU6xgLtDq1n_CaS8_lMeurfaBiSuWkdCrI6kMnrgjyu7JjcekKEe6Y2DUbcqLswHJcNousE-2c57e6fLhCAQquYFH7kYyeXgqtRIYXCm42sakch6AHB7Hrmt9NhJWu2eI2vGF9X1rR-okvWzXQ6pUD1yVdnrTOLg-ZQONtiAO1GMIcTyfyyFBR9Gdgw_W7MPx-KI-yhSPI8GgzTJE2T-GGU5UkABygGeRz5k8SDJL8fpGmcnQL4ulSIo8zH49Ewy_NRluZpGgAK6Yyd9R_LjS5l5aV5xVV9bXn6BriRpdY?type=png)](https://mermaid.live/edit#pako:eNpVUcFuwjAM_ZXIu2xSW7XQdaWakCYxTmOT4DayQ0jcNqJJUEgZDMG3L6Vs2g5JbL9n-9k5AjcCoYAwDKl20jVYkKfSoSVKVpY5aTTVF7BszCevmXXkZU71tl15eFMTxfjbgkxwJzlSTYgSijcStVvOJjPSmx9UoxZUm0Z4ePm8l1sndUU6xgLtDq1n_CaS8_lMeurfaBiSuWkdCrI6kMnrgjyu7JjcekKEe6Y2DUbcqLswHJcNousE-2c57e6fLhCAQquYFH7kYyeXgqtRIYXCm42sakch6AHB7Hrmt9NhJWu2eI2vGF9X1rR-okvWzXQ6pUD1yVdnrTOLg-ZQONtiAO1GMIcTyfyyFBR9Gdgw_W7MPx-KI-yhSPI8GgzTJE2T-GGU5UkABygGeRz5k8SDJL8fpGmcnQL4ulSIo8zH49Ewy_NRluZpGgAK6Yyd9R_LjS5l5aV5xVV9bXn6BriRpdY)

### 1. Configure Fleet

The Fleet server must be configured with the APNS & SCEP certificates/keys copied from the existing server. This is done via manual modification of the Fleet database and configurations. The Fleet team will perform this configuration on Fleet Cloud instances and can advise how to do it on self-hosted Fleet instances.

In most cases, the paths (portion of the URL after the domain name) used in the enrollment profile `ServerURL`, `CheckInURL` and SCEP URL will differ from those used by Fleet. The Fleet Server load balancer must be configured to redirect the MDM client via HTTP 3xx redirects.

[Apple's documentation](https://developer.apple.com/documentation/devicemanagement/implementing_device_management/sending_mdm_commands_to_a_device?language=objc) states:

> MDM follows HTTP 3xx redirections without user interaction. However, it doesn’t save the URL given by HTTP 301 (Moved Permanently) redirections. Each transaction begins at the URL the MDM payload specifies.

Therefore, redirects must remain as long as migrated devices are enrolled.

For a typical MicroMDM to Fleet migration, the following redirects are used:

| From (MicroMDM path) | To (Fleet path) |
| -------------------- | --------------- |
| /mdm/checkin         | /mdm/apple/mdm  |
| /mdm/connect         | /mdm/apple/mdm  |
| /scep                | /mdm/apple/scep |

SCEP certificate renewals need special handling for migrated devices. This is configured (by, or with guidance from the Fleet team) in the server using the [`FLEET_SILENT_MIGRATION_ENROLLMENT_PROFILE` environment variable](https://github.com/fleetdm/fleet/pull/20063). When configured, migrated devices receive an enrollment profile with matching keys when SCEP renewal comes due (migrated devices reject the typical profile Fleet sends because it includes the new server URL).

### 2. Import database records

The Fleet server is made aware of the devices that will be migrated by inserting records into the database. The Fleet team will perform this operation in Fleet Cloud and can advise for self-hosted instances.

For MicroMDM, a [migration script](https://github.com/fleetdm/fleet/pull/18151) has been made that will generate the necessary SQL statements from the MicroMDM database.

For other MDM solutions, please work with the Fleet team to generate the appropriate records.

### 3. Configure controls

Next, configure the controls that will be applied to migrated devices. Use the Teams features in Fleet Premium to apply different configurations to different devices.

In particular,

* [Configuration profiles](https://fleetdm.com/docs/using-fleet/mdm-custom-os-settings#custom-os-settings)
* [OS updates](https://fleetdm.com/docs/using-fleet/mdm-os-updates)
* [Disk encryption](https://fleetdm.com/docs/using-fleet/mdm-disk-encryption)

When the device checks in after migration, Fleet will send the full set of configuration profiles configured for that device's team. Any profiles with identifiers matching existing profiles on the device will be updated in place.

Fleet will not send commands to remove profiles that have not been configured in Fleet. Either remove these profiles before migration in the existing MDM before migration or use `fleetctl` or the Fleet API to send an MDM command to remove any undesired profiles.

OS update configurations will apply automatically after the device is migrated.

As of Fleet 4.55, disk encryption keys will automatically be re-escrowed after migration the next time the user logs into their device.

### 4. Install `fleetd`

Install `fleetd` on the devices to migrate. Devices with `fleetd` installed will begin to show up in the Fleet UI (with profiles in a "Pending" state).

Generate `.pkg` packages following the [standard enrollment documentation](https://fleetdm.com/docs/using-fleet/enroll-hosts). Install the package using the existing MDM or any other management tool.

Devices are automatically assigned to Teams in Fleet based on the package they are provided, so be sure to distribute packages that assign devices to teams with the relevant configurations.

### 5. Update DNS

Devices are now communicating with the Fleet server via the `fleetd` agent. They have not yet migrated MDM servers.

Ensure the Fleet server load balancer can terminate HTTPS using the existing server hostname. This typically involves issuing a certificate [with AWS ACM](https://docs.aws.amazon.com/acm/latest/userguide/gs-acm-request-public.html). In Fleet Cloud, the Fleet team will ask the customer team to update a DNS record for verification so that AWS can issue the certificate.

Now the customer updates DNS to point the existing domain to the Fleet server load balancer. This typically involves setting a `CNAME` record with the hostname of the load balancer (eg. `mdm.example.com -> fleet-cloud-alb-1723349272.us-east-2.elb.amazonaws.com`).

Devices will begin checking in with the Fleet server and receiving new configurations.

### 6. Decommission the old server

At this point, the migration is complete. The old server can be decommissioned.

Keep a database backup of the old server on hand in case it is ever needed for reference or recovery.

## Gradual migration

In the process described, when we update DNS all of the devices are migrated immediately. To minimize risk, it is often desired to gradually migrate devices.

Fleet has created a [migration proxy](https://github.com/fleetdm/fleet/tree/main/tools/mdm/migration/mdmproxy) that can be used to gradually migrate specific devices and/or a percentage of devices. This allows a staged migration with progressively more devices migrated.

## Conclusion

Seamless MDM migrations on macOS are not just possible but are a significant step forward in maintaining a secure and compliant environment without disrupting end users. By following this guide, you can transition from your existing MDM solution to Fleet smoothly, keeping your devices managed and secure throughout the process. If you encounter any challenges, the Fleet team is ready to assist you, ensuring your migration is successful.

For organizations ready to take control of their MDM strategy, this seamless migration process is an opportunity to upgrade to a modern, flexible, and secure management solution. We encourage you to reach out for support or further explore the robust features Fleet offers to enhance your device management capabilities.

<meta name="category" value="guides">
<meta name="authorFullName" value="Zach Wasserman">
<meta name="authorGitHubUsername" value="zwass">
<meta name="publishedOn" value="2024-08-08">
<meta name="articleTitle" value="Seamless MDM migrations to Fleet">
<meta name="articleImageUrl" value="../website/assets/images/articles/seamless-mdm-migration-1600x900@2x.png">
<meta name="description" value="This guide provides a process for seamlessly migrating macOS devices from an existing MDM solution to Fleet.">
