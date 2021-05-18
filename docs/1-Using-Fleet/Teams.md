# Teams

`Applies only to Fleet Basic`

```
ℹ️  In Fleet 4.0, Teams were introduced.
```

In Fleet, you can group hosts together in a team.

With hosts segmented into exclusive teams, you can apply specific queries, packs, and agent options to each team.

For example, you might create a team for each type of system in your organization. You can name the teams `Workstations`, `Workstations - sandbox`, `Servers`, and `Servers - sandbox`.

> A popular pattern is to end a team’s name with “- sandbox”, then you can use this to test new queries and configuration with staging hosts or volunteers acting as canaries.

Then you can:

- Enroll hosts to one team using team specific enroll secrets

- Apply unique agent options to each team

- Schedule queries that target one or more teams

- Run live queries against one or more teams

- Grant users access to one or more teams
