# ADR-0003: Switching to long-lived forks to manage actively maintained third-party dependencies

## Status

Rejected

## Date

2025-07-21

## Context

Fleet uses a number of third-party open-source dependencies, and in some cases needs to make modifications to those
dependencies to conform to our needs. These modifications may or may not make sense in the context of the upstream
project, but in many cases the modifications require editing code rather than wrapping it.

Previously, in cases where it didn't make sense for Fleet to wait for changes to be merged upstream, engineers would
copy libraries to into the monorepo. However, this made it quite difficult to pull in beneficial upstream changes,
including security fixes. It also meant that changes useful beyond Fleet were difficult for other members of the
community to benefit from, even though this arrangement meets the letter of libraries' licenses.

## Decision

Going forward, for actively maintained GitHub-hosted third-party dependencies where Fleet needs to make its own
modification on top, that work will be done on a fork of the appropriate repository in the `fleet` GitHub organization.
See [Impact](#impact) for exact implementation details.

## Consequences

### Benefits

* It's simpler to incorporate upstream fixes and improvements for libraries as those changes are made, increasing the
likelihood that Fleet will pull those improvements in and decreasing duplicated work.
* It's easier to push more broadly useful changes upstream, as those change sets will live in a feature branch prior to
being merged to our "working branch." This allows us to be better open source citizens, and will increase Fleet's
visibility in relevant circles.
* This practice more closely aligns with industry standard approaches to maintaining workload-specific patches on top of
third-party open source dependencies, allowing for easier onboarding of internal and external contributors.

### Drawbacks

#### More exceptions to the monorepo rule

This decision adds additional nuance to the monorepo "Why this way" entry, and results in additional repositories being
created under the fleetdm organization, potentially causing confusion for external users.

Mitigations:

1. Pay close attention to revised "Why this way" verbiage, including a reference back to this ADR for a comprehensive
explanation of what we're doing and the context for it.
2. Continue to keep internal self-contained dependencies inside the monorepo.
3. Ensure that library forks are *not* pinned at the organization level.
4. Archive (but don't delete) dependency repos that are no longer relevant.

#### More workflow friction

Editing both a dependency and the code that depends on it requires managing two repositories, compared to one when code
is copied into the monorepo.

Mitigation:

1. Use git submodules rather than heavier-duty package management
2. Allow pre-merge branches in the `fleet` repo to point at pre-merge commits in dependency forks so upstream and
downstream can be modified concurrently in response to review feedback.

#### More ways of doing the same thing

To avoid incurring significant development effort as soon as this ADR is accepted, existing dependencies are not
required to immediately conform to this workflow. This creates additional inconsistencies in the codebase (including
naming on the `third-party` directory).

Mitigation:

1. Create an engineering-initiated issue for inventorying dependencies, with an eye to migrating them to the fork-based
workflow if they're actively developed upstream, as soon as this ADR is accepted.
2. Prioritize library cleanup as low-hanging fruit as part of new-hire onboarding, with each library getting its own
engineering-initiated story under the issue created in the first mitigation step. Smaller/easier-to-migrate libraries
should be prioritized over larger ones to gain momentum.

### Impact

For new cases, where Fleet needs to make changes to a new actively maintained[^1] GitHub-hosted third-party dependency,
Fleet will fork the dependency into the `fleet` GitHub organization. Upon forking, the repo description must be
modified to indicate that the library incorporates Fleet-specific changes, for inclusion in a Fleet product
(e.g. Fleet server or fleetd).

Changes to the dependency must be pushed to a branch distinct from the upstream repo's branches, allowing our fork to be
easily synced with upstream. Changes must reviewed before merging into this branch by a Fleetie, similar to merging into
`main` in `fleet/fleetdm` (e.g. Go changes must be reviewed by a backend engineer).

Periodically, branches mirroring the upstream repo should be synced to that repo. If upstream changes make sense to
merge into the Fleet-specific branch, an issue (engineering-initiated or otherwise as appropriate) should be created
to merge upstream changes in. Merges from upstream require the same level of review as a normal code contribution: the
person doing the review must be distinct from the person doing the merge.

Forks may be imported into the monorepo via git submodules (instead of using go dep), into the `/third-party`[^2]
directory. For work merged into `main` or a release branch, submodules must reference a merged commit that
has passed peer review. Open PRs may point to commits that are themselves on a PR branch to allow for quick iteration of
both monorepo and dependency code.

Fleeties should attempt to upstream their changes. If the Fleet-specific fork of a dependency is no longer needed (e.g.
because all relevant changes were successfully upstreamed) the consumer of the dependency should be updated to refer to
the upstream package directly via normal dependency management mechanisms (e.g. go dep or npm). Once this change has
been merged to `main` in the monorepo, the fork should be archived, with the repo description updated to indicate that
the upstream repo should be used instead.

### Future considerations

This change applies to any future third-party dependency usage. Existing dependencies that would qualify for this path
if they were imported today should be migrated to this workflow via engineering-initiated issues. An
engineering-initiated issue will be created to inventory these dependencies when this ADR is accepted, and that issue
will spawn one engineering-initiated issue per library for the migration itself.

The expected final state is that all actively-maintained[^1] third-party dependencies are managed via forks rather than
by being inlined in the code, but this ADR doesn't prescribe how long that process should take.

## Alternatives considered

### Status quo: inlined dependencies

Our existing approach for dependencies needing modification was to copy-paste code into the repository.

#### Pros

* Quick
* Status quo
* Dependencies can be revised concurrently with downstream code

#### Cons

* Inconsistent conventions on where inlined dependencies live
* Difficult to incorporate upstream improvements
* Difficult to quickly determine drift between our version of the dependency and upstream
* Upstream version history is not reflected in our version of the code
* Contributing upstream is hard enough that it's unlikely to happen

#### Why not selected

Code is read more than it's written, and the maintenance burden of effectively creating a lightly maintained
point-in-time fork creates subtle risks that turn in to time bombs for critical code. We don't want to increase risks
by continuing to add dependencies this way.

### Forks managed by package manager

We could implement this ADR, but with package management rather than git submodules as our import mechanism.

#### Pros

* Consistent handling of dependencies, whether they're forked or not

#### Cons

* Significant overhead for either tagging releases or pinning versions, slowing down the pace of development

#### Why not selected

We want to implement this change in a way that reduces friction rather than creating it, to avoid the temptation to
rewrite dependencies instead of importing them at all.

### Immediate migration of all dependencies

We could implement this ADR immediately on existing dependencies rather than just new ones. In effect, this would follow
the same implementation plan, but with much higher priority for existing library migration work.

#### Pros

* We get to a consistent dependency management state sooner

#### Cons

* Work that's in effect a refactor that should have no immediate customer-facing impact evicts customer-facing work,
increasing changes of business pushback on the entire idea.

#### Why not selected

Improving dependency management going forward strikes the best balance of iterative improvement (and not increasing
risks as more dependencies are added and tweaked) with velocity on business feature and bugfix priorities. This more
relaxed pacing also allows us to take learnings from one library migration at a time in case processes need to be
revised, rather than performing a suboptimal fix all at once.

## References

None.

[^1]: For purposes of this ADR, "actively maintained" means "having commits within the last three months to a branch Fleet is
interested in consuming." If an upstream repo has been abandoned, following this workflow (rather than copying the code
into the monorepo) is optional.
[^2]: Note `-`, not `_`, for consistency with other directories in the `fleet monorepo.