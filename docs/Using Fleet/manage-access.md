# Manage access

Users have different abilities depending on the access level they have.

## Roles

### Admin

Users with the admin role receive all permissions.

### Maintainer

Maintainers can manage most entities in Fleet, like queries, policies, and labels.
Unlike admins, maintainers cannot edit higher level settings like application configuration, teams or users.

### Observer

The observer role is a read-only role. It can access most entities in Fleet, like queries, policies, labels, application configuration, teams, etc.
They can also run queries configured with the `observer_can_run` flag set to `true`.

### Observer+

`Applies only to Fleet Premium`

Observer+ is an observer with the added ability to run *any* query.

### GitOps

`Applies only to Fleet Premium`

GitOps is a modern approach to Continuous Deployment (CD) that uses Git as the single source of truth for declarative infrastructure and application configurations.
GitOps is an API-only and write-only role that can be used on CI/CD pipelines.

## User permissions

| **Action**                                                                                                                                 | Observer | Observer+* | Maintainer | Admin | GitOps* |
| ------------------------------------------------------------------------------------------------------------------------------------------ | -------- | ---------- | ---------- | ----- | ------- |
| View all [activity](https://fleetdm.com/docs/using-fleet/rest-api#activities)                                                              | ✅       | ✅         | ✅         | ✅    |         |
| View all hosts                                                                                                                             | ✅       | ✅         | ✅         | ✅    |         |
| View a host by identifier                                                                                                                  | ✅       | ✅         | ✅         | ✅    | ✅      |
| Filter hosts using [labels](https://fleetdm.com/docs/using-fleet/rest-api#labels)                                                          | ✅       | ✅         | ✅         | ✅    |         |
| Target hosts using labels                                                                                                                  | ✅       | ✅         | ✅         | ✅    |         |
| Add/remove manual labels to/from hosts                                                                                                     |          |            | ✅         | ✅    | ✅      |
| Add and delete hosts                                                                                                                       |          |            | ✅         | ✅    |         |
| Transfer hosts between teams\*                                                                                                             |          |            | ✅         | ✅    | ✅      |
| Create, edit, and delete labels                                                                                                            |          |            | ✅         | ✅    | ✅      |
| View all software                                                                                                                          | ✅       | ✅         | ✅         | ✅    |         |
| Filter software by [vulnerabilities](https://fleetdm.com/docs/using-fleet/vulnerability-processing#vulnerability-processing)               | ✅       | ✅         | ✅         | ✅    |         |
| Filter hosts by software                                                                                                                   | ✅       | ✅         | ✅         | ✅    |         |
| Filter software by team\*                                                                                                                  | ✅       | ✅         | ✅         | ✅    |         |
| Manage [vulnerability automations](https://fleetdm.com/docs/using-fleet/automations#vulnerability-automations)                             |          |            |            | ✅    | ✅      |
| Run queries designated "**observer can run**" as live queries against all hosts                                                            | ✅       | ✅         | ✅         | ✅    |         |
| Run any query as [live query](https://fleetdm.com/docs/using-fleet/fleet-ui#run-a-query) against all hosts                                 |          | ✅         | ✅         | ✅    |         |
| Create, edit, and delete queries                                                                                                           |          |            | ✅         | ✅    | ✅      |
| View all queries and their reports                                                                                                         | ✅       | ✅         | ✅         | ✅    | ✅      |
| Manage [query automations](https://fleetdm.com/docs/using-fleet/fleet-ui#schedule-a-query)                                                 |          |            | ✅         | ✅    | ✅      |
| Create, edit, view, and delete packs                                                                                                       |          |            | ✅         | ✅    | ✅      |
| View all policies                                                                                                                          | ✅       | ✅         | ✅         | ✅    | ✅      |
| Run all policies                                                                                                                           |          | ✅         | ✅         | ✅    |         |
| Filter hosts using policies                                                                                                                | ✅       | ✅         | ✅         | ✅    |         |
| Create, edit, and delete policies for all hosts                                                                                            |          |            | ✅         | ✅    | ✅      |
| Create, edit, and delete policies for all hosts assigned to team\*                                                                         |          |            | ✅         | ✅    | ✅      |
| Manage [policy automations](https://fleetdm.com/docs/using-fleet/automations#policy-automations)                                           |          |            |            | ✅    | ✅      |
| Create, edit, view, and delete users                                                                                                       |          |            |            | ✅    |         |
| Add and remove team users\*                                                                                                                |          |            |            | ✅    | ✅      |
| Create, edit, and delete teams\*                                                                                                           |          |            |            | ✅    | ✅      |
| Create, edit, and delete [enroll secrets](https://fleetdm.com/docs/deploying/faq#when-do-i-need-to-deploy-a-new-enroll-secret-to-my-hosts) |          |            | ✅         | ✅    | ✅      |
| Create, edit, and delete [enroll secrets for teams](https://fleetdm.com/docs/using-fleet/rest-api#get-enroll-secrets-for-a-team)\*         |          |            | ✅         | ✅    |         |
| Read organization settings\**                                                                                                              | ✅       | ✅         | ✅         | ✅   | ✅      |
| Read Single Sign-On settings\**                                                                                                            |          |            |            | ✅    |         |
| Read SMTP settings\**                                                                                                                      |          |            |            | ✅    |         |
| Read osquery agent options\**                                                                                                              |          |            |            | ✅    |         |
| Edit [organization settings](https://fleetdm.com/docs/using-fleet/configuration-files#organization-settings)                               |          |            |            | ✅    | ✅      |
| Edit [agent options](https://fleetdm.com/docs/using-fleet/configuration-files#agent-options)                                               |          |            |            | ✅    | ✅      |
| Edit [agent options for hosts assigned to teams](https://fleetdm.com/docs/using-fleet/configuration-files#team-agent-options)\*            |          |            |            | ✅    | ✅      |
| Initiate [file carving](https://fleetdm.com/docs/using-fleet/rest-api#file-carving)                                                        |          |            | ✅         | ✅    |         |
| Retrieve contents from file carving                                                                                                        |          |            |            | ✅    |         |
| View Apple mobile device management (MDM) certificate information                                                                          |          |            |            | ✅    |         |
| View Apple business manager (BM) information                                                                                               |          |            |            | ✅    |         |
| Generate Apple mobile device management (MDM) certificate signing request (CSR)                                                            |          |            |            | ✅    |         |
| View disk encryption key for macOS and Windows hosts                                                                                       | ✅       | ✅         | ✅         | ✅    |         |
| Edit OS updates for macOS and Windows hosts                                                                                                |          |            | ✅         | ✅    | ✅      |
| Create, edit, resend and delete configuration profiles for macOS and Windows hosts                                                                  |          |            | ✅         | ✅    | ✅      |
| Execute MDM commands on macOS and Windows hosts\**                                                                                         |          |            | ✅         | ✅    |         |
| View results of MDM commands executed on macOS and Windows hosts\**                                                                        | ✅       | ✅         | ✅         | ✅    |         |
| Edit [MDM settings](https://fleetdm.com/docs/using-fleet/mdm-macos-settings)                                                               |          |            |            | ✅    | ✅      |
| Edit [MDM settings for teams](https://fleetdm.com/docs/using-fleet/mdm-macos-settings)                                                     |          |            |            | ✅    | ✅      |
| View all [MDM settings](https://fleetdm.com/docs/using-fleet/mdm-macos-settings)                                                           |          |            |            | ✅    | ✅      |
| Upload an EULA file for MDM automatic enrollment\*                                                                                         |          |            |            | ✅    |         |
| View/download MDM macOS setup assistant\*                                                                                                  |          |            | ✅         | ✅    |         |
| Edit/upload MDM macOS setup assistant\*                                                                                                    |          |            | ✅         | ✅    | ✅      |
| View metadata of MDM macOS bootstrap packages\*                                                                                            |          |            | ✅         | ✅    |         |
| Edit/upload MDM macOS bootstrap packages\*                                                                                                 |          |            | ✅         | ✅    | ✅      |
| Enable/disable MDM macOS setup end user authentication\*                                                                                   |          |            | ✅         | ✅    | ✅      |
| Run arbitrary scripts on hosts\*                                                                                                           |          |            | ✅         | ✅    |         |
| View saved scripts\*                                                                                                                       | ✅       | ✅         | ✅         | ✅    |         |
| Edit/upload saved scripts\*                                                                                                                |          |            | ✅         | ✅    | ✅      |
| Run saved scripts on hosts\*                                                                                                               | ✅       | ✅         | ✅         | ✅    |         |
| Lock, unlock, and wipe hosts\*                                                                                                             |          |            | ✅         | ✅    |         |

\* Applies only to Fleet Premium

\** Applies only to [Fleet REST API](https://fleetdm.com/docs/using-fleet/rest-api)

## Team user permissions

`Applies only to Fleet Premium`

Users in Fleet either have team access or global access.

Users with team access only have access to the [hosts](https://fleetdm.com/docs/using-fleet/rest-api#hosts), [software](https://fleetdm.com/docs/using-fleet/rest-api#software), and [policies](https://fleetdm.com/docs/using-fleet/rest-api#policies) assigned to
their team.

Users with global access have access to all
[hosts](https://fleetdm.com/docs/using-fleet/rest-api#hosts), [software](https://fleetdm.com/docs/using-fleet/rest-api#software), [queries](https://fleetdm.com/docs/using-fleet/rest-api#queries), and [policies](https://fleetdm.com/docs/using-fleet/rest-api#policies). Check out [the user permissions
table](#user-permissions) above for global user permissions.

Users can be assigned to multiple teams in Fleet.

Users with access to multiple teams can be assigned different roles for each team. For example, a user can be given access to the "Workstations" team and assigned the "Observer" role. This same user can be given access to the "Servers" team and assigned the "Maintainer" role.

| **Action**                                                                                                                       | Team observer | Team observer+ | Team maintainer | Team admin | Team GitOps |
| -------------------------------------------------------------------------------------------------------------------------------- | ------------- | -------------- | --------------- | ---------- | ----------- |
| View hosts                                                                                                                       | ✅            | ✅             | ✅              | ✅         |             |
| View a host by identifier                                                                                                        | ✅            | ✅             | ✅              | ✅         | ✅          |
| Filter hosts using [labels](https://fleetdm.com/docs/using-fleet/rest-api#labels)                                                | ✅            | ✅             | ✅              | ✅         |             |
| Target hosts using labels                                                                                                        | ✅            | ✅             | ✅              | ✅         |             |
| Add/remove manual labels to/from hosts                                                                                           |               |                | ✅              | ✅         | ✅          |
| Add and delete hosts                                                                                                             |               |                | ✅              | ✅         |             |
| Filter software by [vulnerabilities](https://fleetdm.com/docs/using-fleet/vulnerability-processing#vulnerability-processing)     | ✅            | ✅             | ✅              | ✅         |             |
| Filter hosts by software                                                                                                         | ✅            | ✅             | ✅              | ✅         |             |
| Filter software                                                                                                                  | ✅            | ✅             | ✅              | ✅         |             |
| Run queries designated "**observer can run**" as live queries against hosts                                                      | ✅            | ✅             | ✅              | ✅         |             |
| Run any query as [live query](https://fleetdm.com/docs/using-fleet/fleet-ui#run-a-query)                                         |               | ✅             | ✅              | ✅         |             |
| Create, edit, and delete only **self authored** queries                                                                          |               |                | ✅              | ✅         | ✅          |
| View team queries and their reports                                                                                              | ✅            | ✅             | ✅              | ✅         |             |
| View global (inherited) queries and their reports\**                                                                             | ✅            | ✅             | ✅              | ✅         |             |
| Manage [query automations](https://fleetdm.com/docs/using-fleet/fleet-ui#schedule-a-query)                                       |               |                | ✅              | ✅         | ✅          |
| View team policies                                                                                                               | ✅            | ✅             | ✅              | ✅         |             |
| Run team policies as a live policy                                                                                               |               | ✅             | ✅              | ✅         |             |
| View global (inherited) policies                                                                                                 | ✅            | ✅             | ✅              | ✅         |             |
| Run global (inherited) policies as a live policy                                                                                 |               | ✅             | ✅              | ✅         |             |
| Filter hosts using policies                                                                                                      | ✅            | ✅             | ✅              | ✅         |             |
| Create, edit, and delete team policies                                                                                           |               |                | ✅              | ✅         | ✅          |
| Manage [policy automations](https://fleetdm.com/docs/using-fleet/automations#policy-automations)                                 |               |                |                 | ✅         | ✅          |
| Add and remove team users                                                                                                        |               |                |                 | ✅         | ✅          |
| Edit team name                                                                                                                   |               |                |                 | ✅         | ✅          |
| Create, edit, and delete [team enroll secrets](https://fleetdm.com/docs/using-fleet/rest-api#get-enroll-secrets-for-a-team)      |               |                | ✅              | ✅         |             |
| Read organization settings\*                                                                                                     | ✅            | ✅             | ✅              | ✅         |             |
| Read agent options\*                                                                                                             | ✅            | ✅             | ✅              | ✅         |             |
| Edit [agent options](https://fleetdm.com/docs/using-fleet/configuration-files#agent-options)                                     |               |                |                 | ✅         | ✅          |
| Initiate [file carving](https://fleetdm.com/docs/using-fleet/rest-api#file-carving)                                              |               |                | ✅              | ✅         |             |
| View disk encryption key for macOS hosts                                                                                         | ✅            | ✅             | ✅              | ✅         |             |
| Edit OS updates for macOS and Windows hosts                                                                                                |          |            | ✅         | ✅    | ✅      |
| Create, edit, resend and delete configuration profiles for macOS and Windows hosts                                                        |               |                | ✅              | ✅         | ✅          |
| Execute MDM commands on macOS and Windows hosts*                                                                                 |               |                | ✅              | ✅         |             |
| View results of MDM commands executed on macOS and Windows hosts*                                                                | ✅            | ✅             | ✅              | ✅         |             |
| Edit [team MDM settings](https://fleetdm.com/docs/using-fleet/mdm-macos-settings)                                                |               |                |                 | ✅         | ✅          |
| View/download MDM macOS setup assistant                                                                                          |               |                | ✅              | ✅         |             |
| Edit/upload MDM macOS setup assistant                                                                                            |               |                | ✅              | ✅         | ✅          |
| View metadata of MDM macOS bootstrap packages                                                                                    |               |                | ✅              | ✅         |             |
| Edit/upload MDM macOS bootstrap packages                                                                                         |               |                | ✅              | ✅         | ✅          |
| Enable/disable MDM macOS setup end user authentication                                                                           |               |                | ✅              | ✅         | ✅          |
| Run arbitrary scripts on hosts                                                                                                   |               |                | ✅              | ✅         |             |
| View saved scripts                                                                                                               | ✅            | ✅             | ✅              | ✅         |             |
| Edit/upload saved scripts                                                                                                        |               |                | ✅              | ✅         |             |
| Run saved scripts on hosts                                                                                                       | ✅            | ✅             | ✅              | ✅         |             |
| View script details by host                                                                                                      | ✅            | ✅             | ✅              | ✅         |             |
| Lock, unlock, and wipe hosts                                                                                                     |               |                | ✅              | ✅         |             |


\* Applies only to [Fleet REST API](https://fleetdm.com/docs/using-fleet/rest-api)

\** Team-level users only see global query results for hosts on teams where they have access.

<meta name="pageOrderInSection" value="900">
<meta name="description" value="Learn about the different roles and permissions in Fleet.">
<meta name="navSection" value="The basics">
