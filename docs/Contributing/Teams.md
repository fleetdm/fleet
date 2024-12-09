# Teams

Teams are a _premium_ feature in Fleet and allow grouping of hosts inside a team, distinct configuration settings specific to that team, limiting user read-write authorizations to specific teams, etc. See [the Teams guide](https://fleetdm.com/guides/teams) for more information on what is a team and how its recommended best-practices.

## How teams are implemented

At a basic level, teams are created and stored in the `teams` table. Other entities that may belong to one or many teams simply have a relationship to the relevant `team.id`, e.g. a host may belong to only one team, and so there is a `host.team_id` column in the `hosts` table that points to the team the host belongs to (or is `NULL` if it doesn't belong to a team, more on that later). Similarly, a user may be authorized to access multiple teams, and so there is a `user_teams` relationship table that maps a given `user.id` to zero or many `team.id`.

In addition to the teams created in the `teams` table, there is also a concept of "No team" and "All teams". Those are not actual entries stored in `teams`, they use special values to represent them.

For most features of Fleet, we use the "No team" concept more than the "All teams" one - that is, if for example you create a script, it is associated with a team or "No team", "All teams" is not an option.

In this case, the backing database table will have both a _nullable_ `team_id` column to reference the ID of the team, and a _non-nullable_ `global_or_team_id` column that is either `0` for "No team" or the same value as the `team_id` column for real teams. An example of such a table is the `scripts` table which stores saved scripts (and scripts belong to a team). The reason for the two columns approach is this:

* This allows to use `global_or_team_id` as part of a `UNIQUE` index constraint, which would not be possible with `team_id` as it is nullable and would allow duplicates.
* This allows to use a foreign key constraint on `team_id` to `teams.id`, which would not be possible with a non-nullable `team_id` column (as the id `0` does not exist in the `teams` table).
* The `global_or_team_id` does not have to "leak out" of the `server/datastore/mysql` package, typically the `Datastore` methods that deal with these tables will receive a `teamID *uint` argument and will convert it internally to the `global_or_team_id` value so the caller doesn't need to know about this column, e.g.: [this code to insert a script](https://github.com/fleetdm/fleet/blob/d47bd8f626ff337badb44139d5b564cb2f640406/server/datastore/mysql/scripts.go#L327-L330). 

Note that the `global_or_team_id` name is a bit of a misnomer as it's really `no_team_or_team_id`, but it has become a bit of a convention to use that name at this point.

However, for places where both "No team" and "All teams" make sense, we use a `nil` team ID to mean "All teams" and an explicit team ID of `0` to mean "No team". For example, [the "List hosts" endpoint](https://github.com/fleetdm/fleet/blob/d47bd8f626ff337badb44139d5b564cb2f640406/server/datastore/mysql/hosts.go#L1253-L1269) can list the hosts for the "No team", but it can also list hosts for "All teams".

Because those "No team"/"All teams" concepts are special in Fleet and are not backed by an actual row in the `teams` table, [explicit checks are made when trying to create or edit a team](https://github.com/fleetdm/fleet/blob/d47bd8f626ff337badb44139d5b564cb2f640406/ee/server/service/teams.go#L78-L83) so that those names are not used.
