# Permissions

Users have different abilities depending on the access level they have.

Users with the Admin role receive all permissions.

## User permissions

```
ℹ️  In Fleet 4.0, the Observer, Maintainer, and Admin roles were introduced.
```

The following table depicts various permissions levels for each role.

| **Action**                                           | Observer | Maintainer | Admin |
| ---------------------------------------------------- | -------- | ---------- | ----- |
| View activity                                        | ✅       | ✅         | ✅    |
| View all hosts                                       | ✅       | ✅         | ✅    |
| Filter hosts using labels                            | ✅       | ✅         | ✅    |
| Target hosts using labels                            | ✅       | ✅         | ✅    |
| Enroll hosts                                         |          | ✅         | ✅    |
| Delete hosts                                         |          | ✅         | ✅    |
| Transfer hosts between teams\*                       |          | ✅         | ✅    |
| Create, edit, and delete labels                      |          | ✅         | ✅    |
| View all software                                    | ✅       | ✅         | ✅    |
| Filter software by vulnerabilities                   | ✅       | ✅         | ✅    |
| Filter hosts by software                             | ✅       | ✅         | ✅    |
| Filter software by team*                             | ✅       | ✅         | ✅    |
| Manage vulnerability automations                     |          | ✅         | ✅    |
| Run only designated, _observer can run_ ,queries as live queries against all hosts  | ✅       | ✅         | ✅    |
| Run any query as live query against all hosts        |          | ✅         | ✅    |
| Create, edit, and delete queries                     |          | ✅         | ✅    |
| View all queries                                     | ✅       | ✅         | ✅    |
| Create, edit, and delete scheduled queries for all hosts |          | ✅         | ✅    |
| Create, edit, and delete scheduled queries for all hosts assigned to a team\*  |          | ✅         | ✅    |
| Create, edit, view, and delete packs                       |          | ✅         | ✅    |
| View all policies                                    | ✅       | ✅         | ✅    |
| Filter hosts using policies                          | ✅       | ✅         | ✅    |
| Create, edit, and delete policies for all hosts      |          | ✅         | ✅    |
| Create, edit, and delete policies for all hosts assigned to team\*     |          | ✅         | ✅    |
| Manage failing policy automations for all hosts      |          | ✅         | ✅    |
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

```
ℹ️  In Fleet 4.0, the Teams feature was introduced.
```

Users either have global access or team access in Fleet. Users with global access can observe and act on all hosts in Fleet. Check out [the user permissions table](#user-permissions) above for global user permissions.

Users with team access can only observe and act on hosts that are assigned to their team.

Users can be a member of multiple teams in Fleet.

Users that are members of multiple teams can be assigned different roles for each team. For example, a user can be given access to the "Workstations" team and assigned the "Observer" role. This same user can be given access to the "Servers" team and assigned the "Maintainer" role.

The following table depicts permissions levels in a team.

All hosts, software, policies, etc. are exclusive to one team. The following permissions outline actions associated with a team.

| **Action**                                                   | Team observer | Team maintainer | Team admin   |
| ------------------------------------------------------------ | -------- | ---------- | ------- |
| View hosts                                                   | ✅       | ✅         | ✅       |
| Filter hosts using labels                                    | ✅       | ✅         | ✅       |
| Target hosts using labels                                    | ✅       | ✅         | ✅       |
| Enroll hosts to team                                         |          | ✅         | ✅       |
| Delete hosts                                                 |          | ✅         | ✅       |
| Filter software by vulnerabilities                           | ✅       | ✅         | ✅       |
| Filter hosts by software                                     | ✅       | ✅         | ✅       |
| Filter software\*                                            | ✅       | ✅         | ✅       |
| Run saved queries as live queries on hosts                   | ✅       | ✅         | ✅       |
| Run custom queries as live queries on hosts                  |          | ✅         | ✅       |
| Create, edit, and delete queries _self authored only_        |          | ✅         | ✅       |
| Create, edit, and delete schedule queries hosts              |          | ✅         | ✅       |
| View policies for hosts                                      | ✅       | ✅         | ✅       |
| View global (inherited) policies                             | ✅       | ✅         | ✅       |
| Filter hosts assigned to team using policies                 | ✅       | ✅         | ✅       |
| Create, edit, and delete policies for hosts                  |          | ✅         | ✅       |
| Add and remove team members                                  |          |            | ✅       |
| Edit team name                                               |          |            | ✅       |
| Create, edit, and delete team enroll secrets                 |          | ✅         | ✅       |
| Edit host agent options                                      |          |            | ✅       |


<meta name="pageOrderInSection" value="900">
