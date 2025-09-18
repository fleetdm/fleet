# ADR-0006: Replace "No Team" with real default team

## Status

Approved

## Date

2025-09-11

## Context

We currently support a virtual No Team for all instances, which acts as a resting place for hosts not assigned to any other team. No Team is not a real team with its own row in the `teams` table or its own full team config (it does have partial config as of [#32129](https://github.com/fleetdm/fleet/pull/32129)). This implementation leads to several challenges:

* It is supported by thousands[^1] of lines of custom code and tests, causing a maintenance burden for developers.
* In actual use, No Team often differs from "real" teams in subtle and unintuitive ways (for example, queries cannot be assigned to No Team, but policies can). This leads to frustration for customers and support staff. 
* As noted [elsewhere](https://docs.google.com/presentation/d/1Q8u5KtgeBmm3g7emt3VJ7nochEV3dKJIm4zUJiiyXd0/edit?slide=id.g3796d19f491_0_59#slide=id.g3796d19f491_0_59), each sprint has capacity assigned to work (bugs and stories) dedicated partially or fully to supporting No Team.

## Decision

Replace the virtual No Team concept with the concept of a _default_ team, which is a team _no different to any other_ except for the following features:

1. It is marked as the default team in the database.
2. It cannot be deleted.
3. If a host is assigned to a team that is deleted, it is reassigned to the default team.
4. If a host is enrolled using the global enrollment key, it is assigned to the default team.

Existing Fleet instances would have a new default team created for them via migration, while new instances would have one created during the setup process. This will be the case for both premium- and free-tier instances, although free-tier instances will continue to hide any team-related UI (as they do now with No Team).

See the [WIP technical design document](https://docs.google.com/document/d/1tTO0ip1lGJXiL0O5vDet6DFlOzv_ufazuiZ6wqB60vY/edit?usp=sharing) for more details on implementation.

## Consequences

### Benefits

* Massive reduction in technical debt and associated future bugs.
* Removal of maintenance burden on devs, freeing up capacity in each sprint.
* Easier to document and explain to customers.
* Relies on well-worn and tested concept (Fleet teams) rather than inventing something new; this will be a huge net _reduction_ in code.
* The "new" features of the default team should be fairly straightforward to implement.
* Can be done in a way that is invisible to existing customers (especially if we name their new default team "No team"), other than that their "No team" will now have upgraded features (like the ability to add queries).

### Drawbacks

* Requires a large database migration. Not necessarily complex, but touching a lot of tables (at least 20).
* Fairly "high touch", in that it requires a surgical code removal from multiple unrelated files, although after the migration is successfully applied most of the code we want to remove will be inert. It may be possible to do this cleanup in several steps.
* Carries significant risk (we're essentially deleting a team, albeit a virtual one, and transferring all its data to another team). Will require careful planning, testing and mitigation.

## Alternatives considered

### Leaving No Team as-is, and continue to add to [`DefaultTeam` config](https://github.com/fleetdm/fleet/blob/9df8e23f7a84ea2cc1f827f0209958ba3572e6a7/server/fleet/teams.go#L191-L194)

The main benefit to this approach is the short-term risk reduction of not doing the work to migrate off of No Team. In the medium-to-long term, we will continue to accrue technical debt as we try to make our virtual team _look_ more like a regular team, and handle issue arising from the fact that it is not _actually_ a real team. The guiding principle of this ADR is that it is [riskier and costlier](https://docs.google.com/presentation/d/1Q8u5KtgeBmm3g7emt3VJ7nochEV3dKJIm4zUJiiyXd0/edit?slide=id.g3796d19f491_0_59#slide=id.g3796d19f491_0_59) to maintain the current No Team concept than it would be to eliminate it. 

### Making No Team a real team, but with ID 0

This leans into the fact that most (not all) of the data and config pointing to the current No Team abstraction uses ID `0` to represent it. By making a real team with that ID, we avoid much of the database migration, making this an attractive alternative. However, this has some significant drawbacks:

1. Using `0` as an ID in a MySQL auto-incrementing table is non-standard and requires some special config.
2. Much more significantly, we have a lot of code that checks for team ID `0`, which we'd still want to remove, except now anything we miss would have significant impact because _the code would actually run_. This could lead to some difficult-to-debug issues affecting only the new default team. If instead we use a new, non-zero ID for the default team, any leftover No Team code will be "dead" because there will be no data with team ID `0` to trigger it.

## References

* [Original presentation on removing No Team](https://docs.google.com/presentation/d/1Q8u5KtgeBmm3g7emt3VJ7nochEV3dKJIm4zUJiiyXd0/edit?slide=id.g351848d7157_0_84#slide=id.g351848d7157_0_84)
* [WIP technical design document](https://docs.google.com/document/d/1tTO0ip1lGJXiL0O5vDet6DFlOzv_ufazuiZ6wqB60vY/edit?usp=sharing)
* [Eng-initiated ticket for this work](https://github.com/fleetdm/fleet/issues/32435)
* [Example No-Team-related bug ticket that opened in the hour I was writing this ADR](https://github.com/fleetdm/fleet/issues/32876)

[^1]: A quick search for "no team" in Go files in the main Fleet repo has 790 hits in 123 files. Most of those are comments above multiple lines or functions dedicated to No Team logic. 
