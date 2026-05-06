# Role-based access

Users have different abilities depending on the access level they have.

## Roles

### Admin

Users with the admin role receive all permissions.

### Maintainer

Maintainers can manage most entities in Fleet, like queries, policies, and labels.

Unlike admins, maintainers cannot edit higher level settings like application configuration, fleets or users.

### Technician

`Applies only to Fleet Premium`

Technicians have the ability to run scripts, view their results, and install/uninstall software.

### Observer

The observer role is a read-only role. It can access most entities in Fleet, like queries, policies, labels, application configuration, fleets, etc.

They can also run reports configured with the `observer_can_run` flag set to `true`.

### Observer+

`Applies only to Fleet Premium`

Observer+ is an observer with the added ability to run *any* report.

### GitOps

`Applies only to Fleet Premium`

GitOps is a modern approach to Continuous Deployment (CD) that uses Git as the single source of truth for declarative infrastructure and application configurations.
GitOps is an API-only and write-only role that can be used on CI/CD pipelines.

## User permissions

| **Action**                                                                                                                                 | Observer | Observer+* | Technician* | Maintainer | Admin | GitOps* |
| ------------------------------------------------------------------------------------------------------------------------------------------ | :------: | :--------: | :--------: | :--------: | :---: | :-----: |
| View all [activity](https://fleetdm.com/docs/using-fleet/rest-api#activities)                                                              | ✅       | ✅         | ✅         | ✅         | ✅    |         |
| Cancel [hosts' upcoming activity](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-upcoming-activity)                                  |          |            |            | ✅         | ✅    |         |
| Manage [activity automations](https://fleetdm.com/docs/using-fleet/audit-logs)                                                             |          |            |            |            | ✅    | ✅      |
| View all hosts                                                                                                                             | ✅       | ✅         | ✅         | ✅         | ✅    |         |
| View a host by identifier                                                                                                                  | ✅       | ✅         | ✅         | ✅         | ✅    | ✅      |
| Filter hosts using [labels](https://fleetdm.com/docs/using-fleet/rest-api#labels)                                                          | ✅       | ✅         | ✅         | ✅         | ✅    |         |
| Target hosts using labels                                                                                                                  | ✅       | ✅         | ✅         | ✅         | ✅    |         |
| Add/remove manual labels to/from hosts                                                                                                     |          |            | ✅         | ✅         | ✅    | ✅      |
| Add and delete hosts                                                                                                                       |          |            |            | ✅         | ✅    |         |
| Transfer hosts between fleets\*                                                                                                            |          |            | ✅         | ✅         | ✅    | ✅      |
| Add user information from IdP to hosts\*                                                                                                   |          |            |            | ✅         | ✅    |         |
| Create, edit, and delete labels                                                                                                            |          |            | ✅         | ✅         | ✅    | ✅      |
| View all software                                                                                                                          | ✅       | ✅         | ✅         | ✅         | ✅    |         |
| Add, edit, and delete software                                                                                                             |          |            |            | ✅         | ✅    | ✅      |
| Download added software                                                                                                                    |          |            | ✅         | ✅         | ✅    |         |
| Install/uninstall software on hosts                                                                                                        |          |            | ✅         | ✅         | ✅    |         |
| Filter software by [vulnerabilities](https://fleetdm.com/docs/using-fleet/vulnerability-processing#vulnerability-processing)               | ✅       | ✅         | ✅         | ✅         | ✅    |         |
| Filter hosts by software                                                                                                                   | ✅       | ✅         | ✅         | ✅         | ✅    |         |
| Filter software by fleet\*                                                                                                                  | ✅       | ✅         | ✅         | ✅         | ✅    |         |
| Manage [vulnerability automations](https://fleetdm.com/docs/using-fleet/automations#vulnerability-automations)                             |          |            |            |            | ✅    | ✅      |
| Run queries designated "**observer can run**" as live queries against all hosts                                                            | ✅       | ✅         | ✅         | ✅         | ✅    |         |
| Run any query as [live query](https://fleetdm.com/docs/using-fleet/fleet-ui#run-a-query) against all hosts                                 |          | ✅         | ✅         | ✅         | ✅    |         |
| Create, edit, and delete queries                                                                                                           |          |            |            | ✅         | ✅    | ✅      |
| View all queries and their reports                                                                                                         | ✅       | ✅         | ✅         | ✅         | ✅    | ✅      |
| Manage [query automations](https://fleetdm.com/docs/using-fleet/fleet-ui#schedule-a-query)                                                 |          |            |            | ✅         | ✅    | ✅      |
| Create, edit, view, and delete packs                                                                                                       |          |            |            | ✅         | ✅    | ✅      |
| View all policies                                                                                                                          | ✅       | ✅         | ✅         | ✅         | ✅    | ✅      |
| Run all policies                                                                                                                           |          | ✅         | ✅         | ✅         | ✅    |         |
| Filter hosts using policies                                                                                                                | ✅       | ✅         | ✅         | ✅         | ✅    |         |
| Create, edit, and delete policies for all hosts                                                                                            |          |            |            | ✅         | ✅    | ✅      |
| Create, edit, and delete policies for all hosts in a fleet\*                                                                         |          |            |            | ✅         | ✅    | ✅      |
| Edit "All fleets" policy automations                                                                                               |          |            |            |            | ✅    | ✅      |
| Edit any fleet's policy automations: calendar events, install software, and run script\*                                                          |          |            |            | ✅         | ✅    | ✅      |
| Edit any fleet's policy automations: other workflows (tickets and webhooks)\*                                                                     |          |            |            |            | ✅    | ✅      |
| Edit "Unassigned" policy automations                                                                                                          |          |            |            |            | ✅    | ✅      |
| View users\**                                                                                                                              | ✅       | ✅        | ✅         |  ✅        | ✅    |         |
| Create, edit, view, and delete users                                                                                                       |          |            |            |            | ✅    |         |
| Add and remove a fleet's users\*                                                                                                                |          |            |            |            | ✅    | ✅      |
| Create, edit, and delete fleets\*                                                                                                           |          |            |            |            | ✅    | ✅      |
| Create, edit, and delete [enroll secrets](https://fleetdm.com/docs/deploying/faq#when-do-i-need-to-deploy-a-new-enroll-secret-to-my-hosts) |          |            |            | ✅         | ✅    | ✅      |
| Create, edit, and delete [enroll secrets for a fleet](https://fleetdm.com/docs/using-fleet/rest-api#get-enroll-secrets-for-a-team)\*         |          |            |            | ✅         | ✅    |         |
| Read organization settings\**                                                                                                              | ✅       | ✅         | ✅         | ✅         | ✅    | ✅      |
| Read Single Sign-On settings\**                                                                                                            |          |            |            |            | ✅    |         |
| Read SMTP settings\**                                                                                                                      |          |            |            |            | ✅    |         |
| Read osquery agent options\**                                                                                                              |          |            |            |            | ✅    |         |
| Edit organization settings                                                                                                                 |          |            |            |            | ✅    | ✅      |
| Edit agent options                                                                                                                         |          |            |            |            | ✅    | ✅      |
| Edit agent options for hosts in a fleet\*                                                                                           |          |            |            |            | ✅    | ✅      |
| Initiate [file carving](https://fleetdm.com/docs/using-fleet/rest-api#file-carving)                                                        |          |            |            | ✅         | ✅    |         |
| Retrieve contents from file carving                                                                                                        |          |            |            |            | ✅    |         |
| Create Apple Push Certificates service (APNs) certificate signing request (CSR)                                                            |          |            |            |            | ✅    |         |
| View, edit, and delete APNs certificate                                                                                                    |          |            |            |            | ✅    |         |
| View, edit, and delete Apple Business (AB) connections                                                                            |          |            |            |            | ✅    |         |
| View, edit, and delete Volume Purchasing Program (VPP) connections                                                                         |          |            |            |            | ✅    |         |
| Connect Android Enterprise                                                                                                                 |          |            |            |            | ✅    |         |
| View disk encryption key for macOS, Windows, and Linux hosts                                                                               | ✅       | ✅         | ✅         | ✅        | ✅    |         |
| View Recovery Lock password for macOS hosts                                                                                                | ✅       | ✅         | ✅         | ✅        | ✅    |         |
| Rotate Recovery Lock password for macOS hosts                                                                                              |          |            |            | ✅         | ✅    |         |
| Edit OS updates for macOS, Windows, iOS, and iPadOS hosts                                                                                  |          |            |            |            | ✅    | ✅      |
| Create, edit, resend and delete configuration profiles for Apple (macOS/iOS/iPadOS), Windows, and Android hosts                            |          |            |            | ✅         | ✅    | ✅      |
| View and resend configuration profiles for Apple (macOS/iOS/iPadOS), Windows, and Android hosts                                            |          |            | ✅         | ✅         | ✅    | ✅      |
| Execute MDM commands on macOS and Windows hosts\**                                                                                         |          |            |            | ✅         | ✅    | ✅      |
| View results of MDM commands executed on macOS and Windows hosts\**                                                                        | ✅       | ✅         | ✅         | ✅         | ✅    |         |
| Edit [OS settings](https://fleetdm.com/docs/rest-api/rest-api#os-settings)                                                                 |          |            |            | ✅         | ✅    | ✅      |
| View all [OS settings](https://fleetdm.com/docs/rest-api/rest-api#os-settings)                                                             |          |            | ✅         | ✅         | ✅    | ✅      |
| Edit [setup experience](https://fleetdm.com/guides/setup-experience)\*                                                                     |          |            |            | ✅         | ✅    | ✅      |
| Add and edit identity provider for IdP authentication, end user license agreement (EULA), and end user migration workflow\*           |          |            |            |            | ✅    |         |
| Add and edit certificate authorities (CA)\*                                                                                                |          |            |            |            | ✅    | ✅      |
| View certificate authorities (CA)\*                                                                                                        |          |            |            | ✅         | ✅    | ✅      |
| View certificate authority secrets (CA)\*                                                                                                  |          |            |            |            | ✅    | ✅      |
| Request certificates (CA)\*                                                                                                                |          |            |            |            | ✅    | ✅      |
| Run scripts on hosts                                                                                                                       |          |            | ✅         | ✅         | ✅    |         |
| Schedule scripts on hosts                                                                                                                  |          |            |            | ✅         | ✅    |         |
| View saved scripts\*                                                                                                                       | ✅       | ✅         | ✅         | ✅         | ✅    |         |
| Edit/upload saved scripts\*                                                                                                                |          |            |            | ✅         | ✅    | ✅      |
| Lock, unlock, and wipe hosts\*                                                                                                             |          |            |            | ✅         | ✅    |         |
| Turn off MDM for specific hosts                                                                                                            |          |            |            | ✅         | ✅    |         |
| Configure Microsoft Entra conditional access integration                                                                                   |          |            |            |            | ✅    |         |
| Add Microsoft Entra tenant                                                                                                                 |          |            |            |            | ✅    |         |
| View [custom variables](https://fleetdm.com/docs/rest-api/rest-api#list-custom-variables)                                                  | ✅       | ✅         | ✅         | ✅         | ✅    |         |
| Create, edit, and delete custom variables                                                                                                  | ✅       | ✅         | ✅         | ✅         | ✅    |         |

\* Applies only to Fleet Premium

\** Applies only to [Fleet REST API](https://fleetdm.com/docs/using-fleet/rest-api)

## Fleet-level user permissions

`Applies only to Fleet Premium`

Users in Fleet either have global access or access to specific fleets.

Users with access to specific fleets only have access to data for their assigned fleet.

Global users can access data for all fleets. Check out [the user permissions
table](#user-permissions) above for global user permissions.

Users can be assigned to multiple fleets, and can have different roles for each fleet. For example, a user can be given access to the "💻 Workstations" fleet and assigned the "Observer" role. This same user can be given access to the "📱🔐 Personal mobile devices" fleet and assigned the "Maintainer" role.

| **Action**                                                                                                                       | Observer | Observer+ | Technician | Maintainer | Admin | GitOps |
| -------------------------------------------------------------------------------------------------------------------------------- | :-----------: | :------------: | :-------------: | :-------------: | :--------: | :---------: |
| View hosts                                                                                                                       | ✅            | ✅             | ✅              | ✅              | ✅         |             |
| View a host by identifier                                                                                                        | ✅            | ✅             | ✅              | ✅              | ✅         | ✅          |
| Filter hosts using [labels](https://fleetdm.com/docs/using-fleet/rest-api#labels)                                                | ✅            | ✅             | ✅              | ✅              | ✅         |             |
| Target hosts using labels                                                                                                        | ✅            | ✅             | ✅              | ✅              | ✅         |             |
| View hosts' [past](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-past-activity) and [upcoming](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-upcoming-activity) activity | ✅            | ✅             | ✅              | ✅              | ✅         |             |
| Cancel hosts' [upcoming](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-upcoming-activity) activity                        |               |                |                 | ✅              | ✅         |             |
| Add/remove manual labels to/from hosts                                                                                           |               |                | ✅              | ✅              | ✅         | ✅          |
| Create and edit self-authored labels                                                                                             |               |                | ✅              | ✅              | ✅         | ✅          |
| Add and delete hosts                                                                                                             |               |                |                 | ✅              | ✅         |             |
| Transfer hosts between fleets\*                                                                                                  |               |                | ✅              | ✅              | ✅         | ✅          |
| View software                                                                                                                    | ✅            | ✅             | ✅              | ✅              | ✅         |             |
| Add, edit, and delete software                                                                                                   |               |                |                 | ✅              | ✅         | ✅          |
| Download added software                                                                                                          |               |                | ✅              | ✅              | ✅         |             |
| Install/uninstall software on hosts                                                                                              |               |                | ✅              | ✅              | ✅         |             |
| Filter software by [vulnerabilities](https://fleetdm.com/docs/using-fleet/vulnerability-processing#vulnerability-processing)     | ✅            | ✅             | ✅              | ✅              | ✅         |             |
| Filter hosts by software                                                                                                         | ✅            | ✅             | ✅              | ✅              | ✅         |             |
| Filter software                                                                                                                  | ✅            | ✅             | ✅              | ✅              | ✅         |             |
| Run queries designated "**observer can run**" as live queries against hosts                                                      | ✅            | ✅             | ✅              | ✅              | ✅         |             |
| Run any query as [live query](https://fleetdm.com/docs/using-fleet/fleet-ui#run-a-query)                                         |               | ✅             | ✅              | ✅              | ✅         |             |
| Create, edit, and delete self-authored queries                                                                                   |               |                |                 | ✅              | ✅         | ✅          |
| View the fleet's queries and their reports                                                                                              | ✅            | ✅             | ✅              | ✅              | ✅         |             |
| View global (inherited) queries and their reports\**                                                                             | ✅            | ✅             | ✅              | ✅              | ✅         |             |
| Manage [query automations](https://fleetdm.com/docs/using-fleet/fleet-ui#schedule-a-query)                                       |               |                |                 | ✅              | ✅         | ✅          |
| View the fleet's policies                                                                                                               | ✅            | ✅             | ✅              | ✅              | ✅         |             |
| Run the fleet's policies as a live policy                                                                                               |               | ✅             | ✅              | ✅              | ✅         |             |
| View global (inherited) policies                                                                                                 | ✅            | ✅             | ✅              | ✅              | ✅         |             |
| Run global (inherited) policies as a live policy                                                                                 |               | ✅             | ✅              | ✅              | ✅         |             |
| Filter hosts using policies                                                                                                      | ✅            | ✅             | ✅              | ✅              | ✅         |             |
| Create, edit, and delete fleet-level policies                                                                                           |               |                |                 | ✅              | ✅         | ✅          |
| Edit fleet-level policy automations: calendar events, install software, and run script                                                  |               |                |                 | ✅              | ✅         | ✅          |
| Edit fleet-level policy automations: other workflows (tickets and webhooks)                                                             |               |                |                 |                 | ✅         | ✅          |
| Add and remove fleet-level users                                                                                                        |               |                |                 |                 | ✅         | ✅          |
| Edit the fleet's name                                                                                                                   |               |                |                 |                 | ✅         | ✅          |
| Create, edit, and delete a [fleet's enroll secrets](https://fleetdm.com/docs/using-fleet/rest-api#get-enroll-secrets-for-a-team)      |               |                |                 | ✅              | ✅         |             |
| Read organization settings\*                                                                                                     | ✅            | ✅             | ✅              | ✅              | ✅         | ✅          |
| Read agent options\*                                                                                                             | ✅            | ✅             | ✅              | ✅              | ✅         |             |
| Edit agent options                                                                                                               |               |                |                 |                 | ✅         | ✅          |
| Initiate [file carving](https://fleetdm.com/docs/using-fleet/rest-api#file-carving)                                              |               |                |                 | ✅              | ✅         |             |
| View disk encryption key for macOS hosts                                                                                         | ✅            | ✅             | ✅              | ✅              | ✅         |             |
| View Recovery Lock password for macOS hosts                                                                                      | ✅            | ✅             | ✅              | ✅              | ✅         |             |
| Rotate Recovery Lock password for macOS hosts                                                                                    |               |                |                 | ✅              | ✅         |             |
| Edit OS updates for macOS, Windows, iOS, and iPadOS hosts                                                                        |               |                |                 |                 | ✅         | ✅          |
| Create, edit, resend and delete configuration profiles for Apple (macOS/iOS/iPadOS), Windows, and Android hosts                  |               |                |                 | ✅              | ✅         | ✅          |
| View and resend configuration profiles for Apple (macOS/iOS/iPadOS), Windows, and Android hosts                                  |               |                | ✅              | ✅              | ✅         | ✅          |
| Execute MDM commands on macOS and Windows hosts*                                                                                 |               |                |                 | ✅              | ✅         |             |
| View results of MDM commands executed on macOS and Windows hosts*                                                                | ✅            | ✅             | ✅              | ✅              | ✅         |             |
| Edit a fleet's [OS settings](https://fleetdm.com/docs/rest-api/rest-api#os-settings)                                                  |               |                |                 | ✅              | ✅         | ✅          |
| Edit [setup experience](https://fleetdm.com/guides/setup-experience)\*                                                           |               |                |                 | ✅              | ✅         | ✅          |
| Run scripts on hosts                                                                                                             |               |                | ✅              | ✅              | ✅         |             |
| Schedule scripts on hosts                                                                                                        |               |                |                 | ✅              | ✅         |             |
| View saved scripts                                                                                                               | ✅            | ✅             | ✅              | ✅              | ✅         |             |
| Edit/upload saved scripts                                                                                                        |               |                |                 | ✅              | ✅         |             |
| View script details by host                                                                                                      | ✅            | ✅             | ✅              | ✅              | ✅         |             |
| Lock, unlock, and wipe hosts                                                                                                     |               |                |                 | ✅              | ✅         |             |
| Turn off MDM for specific hosts                                                                                                  |               |                |                 | ✅              | ✅         |             |
| View certificate authorities (CA)                                                                                                |               |                |                 | ✅              | ✅         | ✅          |
| View [custom variables](https://fleetdm.com/docs/rest-api/rest-api#list-custom-variables)                                        | ✅            | ✅             | ✅             | ✅              | ✅         |             |

\* Applies only to [Fleet REST API](https://fleetdm.com/docs/using-fleet/rest-api)

\** Fleet-level users only see global report results for hosts on fleets where they have access.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2024-10-31">
<meta name="articleTitle" value="Role-based access">
<meta name="description" value="Learn about the different roles and permissions in Fleet.">
