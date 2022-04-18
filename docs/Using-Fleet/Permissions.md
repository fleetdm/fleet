# Permissions

Users have different abilities depending on the access level they have.

Users with the Admin role receive all permissions.

## User permissions

| **Action**                                           | Observer | Maintainer | Admin |
| ---------------------------------------------------- | -------- | ---------- | ----- |
| View all activity                                        | ✅       | ✅         | ✅    |
| View all hosts                                       | ✅       | ✅         | ✅    |
| Filter hosts using labels                            | ✅       | ✅         | ✅    |
| Target hosts using labels                            | ✅       | ✅         | ✅    |
| Add and delete hosts                                         |          | ✅         | ✅    |
| Transfer hosts between teams\*                       |          | ✅         | ✅    |
| Create, edit, and delete labels                      |          | ✅         | ✅    |
| View all software                                    | ✅       | ✅         | ✅    |
| Filter software by vulnerabilities                   | ✅       | ✅         | ✅    |
| Filter hosts by software                             | ✅       | ✅         | ✅    |
| Filter software by team*                             | ✅       | ✅         | ✅    |
| Manage vulnerability automations                     |          |           | ✅    |
| Run only designated, _observer can run_ ,queries as live queries against all hosts  | ✅       | ✅         | ✅    |
| Run any query as live query against all hosts        |          | ✅         | ✅    |
| Create, edit, and delete queries                     |          | ✅         | ✅    |
| View all queries                                     | ✅       | ✅         | ✅    |
| Add, edit, and remove queries from all schedules  |          | ✅         | ✅    |
| Create, edit, view, and delete packs                       |          | ✅         | ✅    |
| View all policies                                    | ✅       | ✅         | ✅    |
| Filter hosts using policies                          | ✅       | ✅         | ✅    |
| Create, edit, and delete policies for all hosts      |          | ✅         | ✅    |
| Create, edit, and delete policies for all hosts assigned to team\*     |          | ✅         | ✅    |
| Manage policy automations      |          |           | ✅    |
| Create, edit, view, and delete users                       |          |            | ✅    |
| Add and remove team members\*                        |          |            | ✅    |
| Create, edit, and delete teams\*                     |          |            | ✅    |
| Create, edit, and delete enroll secrets              |          | ✅         | ✅    |
| Create, edit, and delete enroll secrets for teams\*  |          | ✅         | ✅    |
| Edit organization settings                           |          |            | ✅    |
| Edit agent options                                   |          |            | ✅    |
| Edit agent options for hosts assigned to teams\*     |          |            | ✅    |




\*Applies only to Fleet Premium

## Team member permissions

`Applies only to Fleet Premium`

Users in Fleet either have team access or global access. 

Users with team access only have access to the hosts, software, schedules, and policies assigned to
their team.

Users with global access have access to all
hosts, software, queries, schedules, and policies. Check out [the user permissions
table](#user-permissions) above for global user permissions.

Users can be a member of multiple teams in Fleet.

Users that are members of multiple teams can be assigned different roles for each team. For example, a user can be given access to the "Workstations" team and assigned the "Observer" role. This same user can be given access to the "Servers" team and assigned the "Maintainer" role.

| **Action**                                                   | Team observer | Team maintainer | Team admin   |
| ------------------------------------------------------------ | -------- | ---------- | ------- |
| View hosts                                                   | ✅       | ✅         | ✅       |
| Filter hosts using labels                                    | ✅       | ✅         | ✅       |
| Target hosts using labels                                    | ✅       | ✅         | ✅       |
| Add and delete hosts                                         |          | ✅         | ✅       |
| Filter software by vulnerabilities                           | ✅       | ✅         | ✅       |
| Filter hosts by software                                     | ✅       | ✅         | ✅       |
| Filter software                                              | ✅       | ✅         | ✅       |
| Run only designated, _observer can run_ ,queries as live queries against all hosts  | ✅       | ✅         | ✅    |
| Run any query as live query        |          | ✅         | ✅    |
| Create, edit, and delete only _self authored_ queries        |          | ✅         | ✅       |
| Add, edit, and remove queries from the schedule                    |          | ✅         | ✅       |
| View policies                                     | ✅       | ✅         | ✅       |
| View global (inherited) policies                             | ✅       | ✅         | ✅       |
| Filter hosts using policies                 | ✅       | ✅         | ✅       |
| Create, edit, and delete policies                  |          | ✅         | ✅       |
| Add and remove team members                                  |          |            | ✅       |
| Edit team name                                               |          |            | ✅       |
| Create, edit, and delete team enroll secrets                 |          | ✅         | ✅       |
| Edit agent options                                      |          |            | ✅       |


<meta name="pageOrderInSection" value="900">
