# Permissions

Users have different abilities depending on the access level they have.

Global users with the Admin role receive all permissions.

## Global user permissions

```
ℹ️  In Fleet 4.0, the Admin, Maintainer, and Observer roles were introduced.
```

The following table depicts various permissions levels at the global level.

| Action                                               | Observer | Maintainer | Admin |
| ---------------------------------------------------- | -------- | ---------- | ------|
| Browse all hosts                                     | ✅       | ✅          | ✅    |
| Filter hosts using labels                            | ✅       | ✅          | ✅    |
| Target hosts using labels                            | ✅       | ✅          | ✅    |
| Run saved queries as live queries against all hosts  | ✅       | ✅          | ✅    |
| Run custom queries as live queries against all hosts |          | ✅          | ✅    |
| Enroll hosts                                         |          | ✅          | ✅    |
| Delete hosts                                         |          | ✅          | ✅    |
| Create saved queries                                 |          | ✅          | ✅    |
| Edit saved queries                                   |          | ✅          | ✅    |
| Delete saved queries                                 |          | ✅          | ✅    |
| Create packs                                         |          | ✅          | ✅    |
| Edit packs                                           |          | ✅          | ✅    |
| Delete packs                                         |          | ✅          | ✅    |
| Create labels                                        |          | ✅          | ✅    |
| Edit labels                                          |          | ✅          | ✅    |
| Delete labels                                        |          | ✅          | ✅    |
| Create users                                         |          |            | ✅    |
| Edit users                                           |          |            | ✅    |
| Delete users                                         |          |            | ✅    |
| Edit organization settings                           |          |            | ✅    |
| Edit global level agent options                      |          |            | ✅    |
| Edit team level agent options *                      |          |            | ✅    |
| Create teams *                                       |          |            | ✅    |
| Edit teams *                                         |          |            | ✅    |
| Add members to teams *                               |          |            | ✅    |

*Available in Fleet Basic

## Team member permissions

`Applies to Fleet Basic`

```
ℹ️  In Fleet 4.0, Teams were introduced.
```

Global users with the Admin role receive all permissions.

The following table depicts various permissions levels in a team.

| Action                                                       | Observer | Maintainer |
| ------------------------------------------------------------ | -------- | ---------- |
| Browse hosts assigned to team                                | ✅       | ✅          |
| Filter hosts assigned to team using labels                   | ✅       | ✅          |
| Target hosts assigned to team using labels                   | ✅       | ✅          |
| Run saved queries as live queries on hosts assigned to team  | ✅       | ✅          |
| Run custom queries as live queries on hosts assigned to team |          | ✅          |
| Enroll hosts to member team                                  |          | ✅          |
| Delete hosts from member team                                |          | ✅          |
| Create saved queries                                         |          | ✅          |
| Edit saved queries                                           |          | ✅          |

