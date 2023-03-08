# Permissions

Users have different abilities depending on the access level they have.

Users with the Admin role receive all permissions.

## User permissions

| **Action**                                                                                                                                 | Observer | Maintainer | Admin |
| ------------------------------------------------------------------------------------------------------------------------------------------ | -------- | ---------- | ----- |
| View all [activity](https://fleetdm.com/docs/using-fleet/rest-api#activities)                                                              | ✅       | ✅         | ✅    |
| View all hosts                                                                                                                             | ✅       | ✅         | ✅    |
| Filter hosts using [labels](https://fleetdm.com/docs/using-fleet/rest-api#labels)                                                          | ✅       | ✅         | ✅    |
| Target hosts using labels                                                                                                                  | ✅       | ✅         | ✅    |
| Add and delete hosts                                                                                                                       |          | ✅         | ✅    |
| Transfer hosts between teams\*                                                                                                             |          | ✅         | ✅    |
| Create, edit, and delete labels                                                                                                            |          | ✅         | ✅    |
| View all software                                                                                                                          | ✅       | ✅         | ✅    |
| Filter software by [vulnerabilities](https://fleetdm.com/docs/using-fleet/vulnerability-processing#vulnerability-processing)               | ✅       | ✅         | ✅    |
| Filter hosts by software                                                                                                                   | ✅       | ✅         | ✅    |
| Filter software by team\*                                                                                                                  | ✅       | ✅         | ✅    |
| Manage [vulnerability automations](https://fleetdm.com/docs/using-fleet/automations#vulnerability-automations)                             |          |            | ✅    |
| Run only designated, **observer can run** ,queries as live queries against all hosts                                                       | ✅       | ✅         | ✅    |
| Run any query as [live query](https://fleetdm.com/docs/using-fleet/fleet-ui#run-a-query) against all hosts                                 |          | ✅         | ✅    |
| Create, edit, and delete queries                                                                                                           |          | ✅         | ✅    |
| View all queries                                                                                                                           | ✅       | ✅         | ✅    |
| Add, edit, and remove queries from all schedules                                                                                           |          | ✅         | ✅    |
| Create, edit, view, and delete packs                                                                                                       |          | ✅         | ✅    |
| View all policies                                                                                                                          | ✅       | ✅         | ✅    |
| Filter hosts using policies                                                                                                                | ✅       | ✅         | ✅    |
| Create, edit, and delete policies for all hosts                                                                                            |          | ✅         | ✅    |
| Create, edit, and delete policies for all hosts assigned to team\*                                                                         |          | ✅         | ✅    |
| Manage [policy automations](https://fleetdm.com/docs/using-fleet/automations#policy-automations)                                           |          |            | ✅    |
| Create, edit, view, and delete users                                                                                                       |          |            | ✅    |
| Add and remove team members\*                                                                                                              |          |            | ✅    |
| Create, edit, and delete teams\*                                                                                                           |          |            | ✅    |
| Create, edit, and delete [enroll secrets](https://fleetdm.com/docs/deploying/faq#when-do-i-need-to-deploy-a-new-enroll-secret-to-my-hosts) |          | ✅         | ✅    |
| Create, edit, and delete [enroll secrets for teams](https://fleetdm.com/docs/using-fleet/rest-api#get-enroll-secrets-for-a-team)\*         |          | ✅         | ✅    |
| Edit [organization settings](https://fleetdm.com/docs/using-fleet/configuration-files#organization-settings)                               |          |            | ✅    |
| Edit [agent options](https://fleetdm.com/docs/using-fleet/configuration-files#agent-options)                                               |          |            | ✅    |
| Edit [agent options for hosts assigned to teams](https://fleetdm.com/docs/using-fleet/configuration-files#team-agent-options)\*            |          |            | ✅    |
| Initiate [file carving](https://fleetdm.com/docs/using-fleet/rest-api#file-carving)                                                        |          | ✅         | ✅    |
| Retrieve contents from file carving                                                                                                        |          |            | ✅    |
| View Apple mobile device management (MDM) certificate information                                                                          |          |            | ✅    |
| View Apple business manager (BM) information                                                                                               |          |            | ✅    |
| Generate Apple mobile device management (MDM) certificate signing request (CSR)                                                            |          |            | ✅    |
| View disk encryption key for macOS hosts enrolled in Fleet's MDM                                                                           | ✅       | ✅         | ✅    |
| Create edit and delete configuration profiles for macOS hosts enrolled in Fleet's MDM                                                      |         | ✅         | ✅    |

\*Applies only to Fleet Premium

## Team member permissions

`Applies only to Fleet Premium`

Users in Fleet either have team access or global access.

Users with team access only have access to the [hosts](https://fleetdm.com/docs/using-fleet/rest-api#hosts), [software](https://fleetdm.com/docs/using-fleet/rest-api#software), [schedules](https://fleetdm.com/docs/using-fleet/fleet-ui#schedule-a-query) , and [policies](https://fleetdm.com/docs/using-fleet/rest-api#policies) assigned to
their team.

Users with global access have access to all
[hosts](https://fleetdm.com/docs/using-fleet/rest-api#hosts), [software](https://fleetdm.com/docs/using-fleet/rest-api#software), [queries](https://fleetdm.com/docs/using-fleet/rest-api#queries), [schedules](https://fleetdm.com/docs/using-fleet/fleet-ui#schedule-a-query) , and [policies](https://fleetdm.com/docs/using-fleet/rest-api#policies). Check out [the user permissions
table](#user-permissions) above for global user permissions.

Users can be a member of multiple teams in Fleet.

Users that are members of multiple teams can be assigned different roles for each team. For example, a user can be given access to the "Workstations" team and assigned the "Observer" role. This same user can be given access to the "Servers" team and assigned the "Maintainer" role.

| **Action**                                                                                                                       | Team observer | Team maintainer | Team admin |
| -------------------------------------------------------------------------------------------------------------------------------- | ------------- | --------------- | ---------- |
| View hosts                                                                                                                       | ✅            | ✅              | ✅         |
| Filter hosts using [labels](https://fleetdm.com/docs/using-fleet/rest-api#labels)                                                | ✅            | ✅              | ✅         |
| Target hosts using labels                                                                                                        | ✅            | ✅              | ✅         |
| Add and delete hosts                                                                                                             |               | ✅              | ✅         |
| Filter software by [vulnerabilities](<(https://fleetdm.com/docs/using-fleet/vulnerability-processing#vulnerability-processing)>) | ✅            | ✅              | ✅         |
| Filter hosts by software                                                                                                         | ✅            | ✅              | ✅         |
| Filter software                                                                                                                  | ✅            | ✅              | ✅         |
| Run only designated, **observer can run** ,queries as live queries against all hosts                                             | ✅            | ✅              | ✅         |
| Run any query as [live query](https://fleetdm.com/docs/using-fleet/fleet-ui#run-a-query)                                         |               | ✅              | ✅         |
| Create, edit, and delete only **self authored** queries                                                                          |               | ✅              | ✅         |
| Add, edit, and remove queries from the schedule                                                                                  |               | ✅              | ✅         |
| View policies                                                                                                                    | ✅            | ✅              | ✅         |
| View global (inherited) policies                                                                                                 | ✅            | ✅              | ✅         |
| Filter hosts using policies                                                                                                      | ✅            | ✅              | ✅         |
| Create, edit, and delete policies                                                                                                |               | ✅              | ✅         |
| Manage [policy automations](https://fleetdm.com/docs/using-fleet/automations#policy-automations)                                 |               |                 | ✅         |
| Add and remove team members                                                                                                      |               |                 | ✅         |
| Edit team name                                                                                                                   |               |                 | ✅         |
| Create, edit, and delete [team enroll secrets](https://fleetdm.com/docs/using-fleet/rest-api#get-enroll-secrets-for-a-team)      |               | ✅              | ✅         |
| Edit [agent options](https://fleetdm.com/docs/using-fleet/configuration-files#agent-options)                                     |               |                 | ✅         |
| Initiate [file carving](https://fleetdm.com/docs/using-fleet/rest-api#file-carving)                                              |               | ✅              | ✅         |
| View disk encryption key for macOS hosts enrolled in Fleet's MDM                                                                 | ✅            | ✅              | ✅         |
| Create edit and delete configuration profiles for macOS hosts enrolled in Fleet's MDM                                            |               | ✅              | ✅         |

<meta name="pageOrderInSection" value="900">
