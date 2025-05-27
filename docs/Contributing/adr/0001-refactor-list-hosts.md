# ADR-0001: Refactor ListHosts 

## Status

**Proposed** | Accepted | Rejected | Deprecated | Superseded

## Date

2025-05-26

## Context

Any logic in Fleet that needs to operate over a set of filtered hosts has two options:

1. Use the [main `ListHosts` method](https://github.com/fleetdm/fleet/blob/725e7336b9f85720ec881b520b6d5b5950927d50/server/datastore/mysql/hosts.go#L954).
2. Use one of the other methods like [`ListHostsLiteByIDs`](https://github.com/fleetdm/fleet/blob/725e7336b9f85720ec881b520b6d5b5950927d50/server/datastore/mysql/hosts.go#L2835)

If neither of those meets the requirements at hand, you can create _another_ `ListHosts*` method, further fragmenting the codebase.  Or you can attempt to add a column or a join to something like `ListHostsLiteByIDs`, possibly expanding it beyond its intended purpose.  Or, finally, you can extend `ListHosts` itself, which is the route typically chosen.  This has lead to the current state where `ListHosts` builds a large, complex query in a way that is not easy to reason about or test, and returns many more columns than are necessarily needed by any particular use case.  

For instance, both the "Transfer hosts to a different team" and "Run a script on multiple hosts" features leverage `ListHosts` to retrieve a set of hosts, running an expensive query that returns large amounts of data when they really just need ID and a few other columns.

Future features and internal processes that operate over a filtered set of hosts will face similar issues.

Additionally, the `ListHostsInLabel` method nearly duplicates `ListHosts`, adding an extra maintenance burden whenever one or the other method is updated.

## Decision

The proposal in this ADR is to refactor `ListHosts` in the following ways:

1. Add an option allowing the caller to specify which host properties to populate.  For example, `opts.HostFieldNames = {"SeenTime", "DeviceMapping", "ScriptsEnabled"}`.  This will allow paring down the response and the related memory and data transfer pressures when we're filtering over large data sets.
1. Move all filtering and joining logic into helper methods. This will make the logic easier to maintain and test.  `applyHostFilters` already does a good job of this, and it can be enhanced further by making sure that each helper fully encapsulates any if/then logic so that `applyHostFilters` mainly iterates over a set of helper methods.  
1. Add a helper method for filtering by label, and deprecate `ListHostsInLabel`. This will alleviate the burden of maintaining essentially duplicate methods.
1. (Optional, but ideal): Track all SQL query parts in a data type and construct the SQL programatically at the end of `ListHosts`, rather than using `fmt.Sprintf`.  This will make the logic less error-prone and easier to test.  We already do this partly in `applyHostFilters` with `whereParams`, but we still have some string soup that we can improve upon.   

## Consequences

**Pros of this refactor**:

- Less data transfer and memory pressure when querying large data sets (e.g. executing a batch script to 100,000s of hosts)
- Logic that is easier to maintain, extend and test

**Cons of this refactor**:

- In [the PoC](https://github.com/fleetdm/fleet/compare/sgress454/28700-batch-execute-with-filter#diff-ec797a071df046dfb849880d689b5dc274601d19626e17e2f61e6ce663adaea9R954-R961), the reflection used to translate `Host` fields into SQL has some overhead, but I think it's pretty negligible, especially since the SQL for the "default" fields is cached.  We could have other cached sets as well.
- Using a `QueryParts` data type would be introducing a new development pattern, and the associated cognitive load.  Building SQL declaratively is a pretty time-tested practice (it's how most ORMs work), but new patterns are new patterns.

And of course, refactors are refactors, so we may need to add additional tests to the current codebase to ensure adequate regression testing.

## Alternatives considered

- Doing nothing.  This is essentially what we did for the first iteration of "execute script on a set of filtered hosts" story.  This necessitated limiting the feature to 5,000 hosts at a time, which still takes upwards of 10 seconds to complete.  
- Refactoring only the `SELECT` portion of the SQL, to add the option of specifying which fields to return.  This would unblock other use cases that want to use `ListHosts` to return large numbers of records, but wouldn't address the growing complexity of the code.  It would also not address the maintenance burden of having both `ListHosts` and `ListHostsInLabel`. The referenced [branch](https://github.com/fleetdm/fleet/compare/sgress454/28700-batch-execute-with-filter) basically implements this.

## References

- The [Batch-run scripts on hundreds of hosts](https://github.com/fleetdm/fleet/issues/28389) story and [subsequent Slack discussion of limitations](https://fleetdm.slack.com/archives/C02A8BRABB5/p1747081462299719).  This refactor definitely doesn't alleviate all of the risks and limitation, but I think it'd be a component in any eventual large-scale solution.
- A [mostly-completed branch](https://github.com/fleetdm/fleet/compare/sgress454/28700-batch-execute-with-filter) implementing the `HostFieldNames` option to select which fields to return from `ListHosts` and `ListHostsInLabel`.
