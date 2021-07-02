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
| Target hosts using labels                            | ✅       | ✅         | ✅    |
| Run saved queries as live queries against all hosts  | ✅       | ✅         | ✅    |
| Run custom queries as live queries against all hosts |          | ✅         | ✅    |
| Enroll hosts                                         |          | ✅         | ✅    |
| Delete hosts                                         |          | ✅         | ✅    |
| Transfer hosts between teams\*                       |          | ✅         | ✅    |
| Create saved queries                                 |          | ✅         | ✅    |
| Edit saved queries                                   |          | ✅         | ✅    |
| Delete saved queries                                 |          | ✅         | ✅    |
| Create packs                                         |          | ✅         | ✅    |
| Edit packs                                           |          | ✅         | ✅    |
| Delete packs                                         |          | ✅         | ✅    |
| Create labels                                        |          | ✅         | ✅    |
| Edit labels                                          |          | ✅         | ✅    |
| Delete labels                                        |          | ✅         | ✅    |
| Create users                                         |          |            | ✅    |
| Edit users                                           |          |            | ✅    |
| Delete users                                         |          |            | ✅    |
| Edit organization settings                           |          |            | ✅    |
| Create enroll secrets                                |          |            | ✅    |
| Edit enroll secrets                                  |          |            | ✅    |
| Edit global level agent options                      |          |            | ✅    |
| Edit team level agent options\*                      |          |            | ✅    |
| Create teams\*                                       |          |            | ✅    |
| Edit teams\*                                         |          |            | ✅    |
| Add members to teams\*                               |          |            | ✅    |

\*Applies only to Fleet Basic

## Team member permissions

`Applies only to Fleet Basic`

```
ℹ️  In Fleet 4.0, the Teams feature was introduced.
```

Users either have global access to Fleet or team access to Fleet. Check out [the user permissions table](#user-permissions) above for global user permissions.

Users can be a member of multiple teams in Fleet.

Users that are members of multiple teams can be assigned different roles for each team. For example, a user can be given access to the "Workstations" team and assigned the "Observer" role. This same user can be given access to the "Servers" team and assigned the "Maintainer" role.

The following table depicts various permissions levels in a team.

| Action                                                       | Observer | Maintainer |
| ------------------------------------------------------------ | -------- | ---------- |
| Browse hosts assigned to team                                | ✅       | ✅         |
| Filter hosts assigned to team using labels                   | ✅       | ✅         |
| Target hosts assigned to team using labels                   | ✅       | ✅         |
| Run saved queries as live queries on hosts assigned to team  | ✅       | ✅         |
| Run custom queries as live queries on hosts assigned to team |          | ✅         |
| Enroll hosts to member team                                  |          | ✅         |
| Delete hosts belonging to member team                        |          | ✅         |
