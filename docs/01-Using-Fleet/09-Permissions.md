# Permissions

Users have different abilities depending on the access level they have.

Users with the Admin role receive all permissions.

## User permissions

```
ℹ️  In Fleet 4.0, the Observer, Maintainer, and Admin roles were introduced.
```

The following table depicts various permissions levels for each role.

| Action                                               | Observer | Maintainer | Admin |
| ---------------------------------------------------- | -------- | ---------- | ----- |
| Browse all hosts                                     | ✅       | ✅         | ✅    |
| Filter hosts using labels                            | ✅       | ✅         | ✅    |
| Browse all policies                                  | ✅       | ✅         | ✅    |
| Filter hosts using policies                          | ✅       | ✅         | ✅    |
| Target hosts using labels                            | ✅       | ✅         | ✅    |
| Run saved queries as live queries against all hosts  | ✅       | ✅         | ✅    |
| Run custom queries as live queries against all hosts |          | ✅         | ✅    |
| Enroll hosts                                         |          | ✅         | ✅    |
| Delete hosts                                         |          | ✅         | ✅    |
| Transfer hosts between teams\*                       |          | ✅         | ✅    |
| Create saved queries                                 |          | ✅         | ✅    |
| Edit saved queries                                   |          | ✅         | ✅    |
| Delete saved queries                                 |          | ✅         | ✅    |
| Schedule queries for all hosts                       |          | ✅         | ✅    |
| Schedule queries for all hosts assigned to a team\*  |          | ✅         | ✅    |
| Create packs                                         |          | ✅         | ✅    |
| Edit packs                                           |          | ✅         | ✅    |
| Delete packs                                         |          | ✅         | ✅    |
| Create labels                                        |          | ✅         | ✅    |
| Edit labels                                          |          | ✅         | ✅    |
| Delete labels                                        |          | ✅         | ✅    |
| Add policies for all hosts                           |          | ✅         | ✅    |
| Remove policies for all hosts                        |          | ✅         | ✅    |
| Create users                                         |          |            | ✅    |
| Edit users                                           |          |            | ✅    |
| Delete users                                         |          |            | ✅    |
| Edit organization settings                           |          |            | ✅    |
| Create enroll secrets                                |          | ✅         | ✅    |
| Edit enroll secrets                                  |          | ✅         | ✅    |
| Edit agent options                                   |          |            | ✅    |
| Edit agent options for hosts assigned to teams\*     |          |            | ✅    |
| Create teams\*                                       |          |            | ✅    |
| Edit teams\*                                         |          |            | ✅    |
| Add members to teams\*                               |          |            | ✅    |

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

| Action                                                       | Observer | Maintainer | Admin   |
| ------------------------------------------------------------ | -------- | ---------- | ------- |
| Browse hosts assigned to team                                | ✅       | ✅         | ✅       |
| Browse policies for hosts assigned to team                   | ✅       | ✅         | ✅       |
| Filter hosts assigned to team using policies                 | ✅       | ✅         | ✅       |
| Filter hosts assigned to team using labels                   | ✅       | ✅         | ✅       |
| Target hosts assigned to team using labels                   | ✅       | ✅         | ✅       |
| Run saved queries as live queries on hosts assigned to team  | ✅       | ✅         | ✅       |
| Run custom queries as live queries on hosts assigned to team |          | ✅         | ✅       |
| Enroll hosts to team                                         |          | ✅         | ✅       |
| Delete hosts assigned to team                                |          | ✅         | ✅       |
| Create queries                                               |          | ✅         | ✅       |
| Edit queries they authored                                   |          | ✅         | ✅       |
| Delete queries they authored                                 |          | ✅         | ✅       |
| Schedule queries for hosts assigned to team                  |          | ✅         | ✅       |
| Add policies for hosts assigned to team                      |          | ✅         | ✅       |
| Remove policies for hosts assigned to team                   |          | ✅         | ✅       |
| Edit users assigned to team                                  |          |            | ✅       |
| Remove users assigned to team                                |          |            | ✅       |
| Create team enroll secrets                                   |          | ✅         | ✅       |
| Edit team enroll secrets                                     |          | ✅         | ✅       |
| Edit agent options for hosts assigned to team                |          |            | ✅       |
