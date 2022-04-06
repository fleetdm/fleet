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
| Browse all hosts                                     | ✅       | ✅         | ✅    |
| Filter hosts using labels                            | ✅       | ✅         | ✅    |
| Target hosts using labels                            | ✅       | ✅         | ✅    |
| Enroll hosts                                         |          | ✅         | ✅    |
| Delete hosts                                         |          | ✅         | ✅    |
| Transfer hosts between teams\*                       |          | ✅         | ✅    |
| Create, edit, and delete labels                      |          | ✅         | ✅    |
| Browse all software                                  | ✅       | ✅         | ✅    |
| Filter software by vulnerabilities                   | ✅       | ✅         | ✅    |
| Filter hosts by software                             | ✅       | ✅         | ✅    |
| Filter software by team*                             | ✅       | ✅         | ✅    |
| Manage vulnerability automations         |          | ✅         | ✅    |
| Run saved queries as live queries against all hosts  | ✅       | ✅         | ✅    |
| Run custom queries as live queries against all hosts |          | ✅         | ✅    |
| Create, edit, and delete queries                     |          | ✅         | ✅    |
| Create, edit, and delete scheduled queries for all hosts |          | ✅         | ✅    |
| Create, edit, and delete scheduled queries for all hosts assigned to a team\*  |          | ✅         | ✅    |
| Create, edit, and delete packs                       |          | ✅         | ✅    |
| View all policies                                  | ✅       | ✅         | ✅    |
| Filter hosts using policies                          | ✅       | ✅         | ✅    |
| Create, edit, and delete policies for all hosts      |          | ✅         | ✅    |
| Create, edit, and delete policies for all hosts assigned to team\*     |          | ✅         | ✅    |
| Manage failing policy automations for all hosts      |          | ✅         | ✅    |
| Create, edit, and delete users                       |          |            | ✅    |
| Add and remove team members\*                        |          |            | ✅    |
| Create, edit, and delete teams\*                     |          |            | ✅    |
| Create, edit, and delete enroll secrets              |          | ✅         | ✅    |
| Create, edit, and delete enroll secrets for teams\*  |          | ✅         | ✅    |
| Edit organization settings                           |          |            | ✅    |
| Edit agent options                                   |          |            | ✅    |
| Edit agent options for hosts assigned to teams\*    |          |            | ✅    |




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

The following table depicts various permissions levels in a team.

| **Action**                                                   | Observer | Maintainer | Admin   |
| ------------------------------------------------------------ | -------- | ---------- | ------- |
| Browse hosts assigned to team                                | ✅       | ✅         | ✅       |
| Filter hosts assigned to team using labels                   | ✅       | ✅         | ✅       |
| Target hosts assigned to team using labels                   | ✅       | ✅         | ✅       |
| Enroll hosts to team                                         |          | ✅         | ✅       |
| Delete hosts assigned to team                                |          | ✅         | ✅       |
| Filter software by vulnerabilities                           | ✅       | ✅         | ✅       |
| Filter hosts by software                                     | ✅       | ✅         | ✅       |
| Filter software by team\*                                    | ✅       | ✅         | ✅       |
| Run saved queries as live queries on hosts assigned to team  | ✅       | ✅         | ✅       |
| Run custom queries as live queries on hosts assigned to team |          | ✅         | ✅       |
| Create, edit, and delete queries _they authored only_        |          | ✅         | ✅       |
| Create, edit, and delete schedule queries for all hosts assigned to team |          | ✅         | ✅       |
| Browse policies for hosts assigned to team                   | ✅       | ✅         | ✅       |
| Browse global (inherited) policies                           | ✅       | ✅         | ✅       |
| Filter hosts assigned to team using policies                 | ✅       | ✅         | ✅       |
| Create, edit, and delete policies for all hosts assigned to team |          | ✅         | ✅       |
| Add and remove team members                                  |          |            | ✅       |
| Edit team name                                               |          |            | ✅       |
| Create, edit, and delete team enroll secrets                 |          | ✅         | ✅       |
| Edit agent options for hosts assigned to team                |          |            | ✅       |


<meta name="pageOrderInSection" value="900">
