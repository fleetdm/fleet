# ADR-0001: OpenSpec Adoption

## Status

Rejected

## Date

2026-05-21

## Context

[OpenSpec](https://openspec.dev/) (by Fission AI, YC W26) is a spec-driven development framework. Teams commit Markdown specification files into an `openspec/` directory in the repo. Each feature or change gets a folder containing a proposal, design doc, task list, and delta specs (Given/When/Then scenarios). When a change ships, deltas are archived and merged into a "spec library" that represents the current state of the system.

The goal is to align AI coding agents and human developers on what to build before code is generated, and to maintain a living specification of system behavior alongside the code.

We evaluated whether Fleet should adopt OpenSpec and begin committing spec files to the repo.

## Decision

We will not adopt OpenSpec.

Our existing workflow already provides the structured-thinking benefit that OpenSpec targets:

- **User stories with detailed acceptance criteria** define what to build and how to validate it.
- **AI-generated PR descriptions** document the problem, approach, affected layers, and alternatives considered.
- **`CLAUDE.md` conventions** guide AI agents on Fleet-specific patterns, error handling, authorization, and the request lifecycle.

The GitHub issue remains the source of truth for each change. Specs, acceptance criteria, and design context live there, linked directly to the PR that implements them. Adopting OpenSpec would create a parallel specification layer that largely restates what already exists in the issue, but in committed files that require ongoing maintenance and can drift from both the issue and the implementation.

## Consequences

**Positive:**
- No additional files added to PRs, keeping review surface area unchanged.
- No new maintenance burden for keeping spec files in sync with the codebase.
- No workflow dependency on a young, single-maintainer tool.

**Negative:**
- We forgo a standardized, machine-readable spec format that could improve AI agent context in the future.
- No centralized "spec library" describing current system behavior. This role is filled by user stories and docs, which are less structured.

**Future considerations:**
- Revisit if our current workflow breaks down at scale.
- Revisit if OpenSpec adds automated spec-to-implementation verification (turning specs into enforceable checks rather than documentation).
- Revisit if the tool matures significantly (stable API, broader maintainer base).

## Alternatives considered

### Adopt OpenSpec fully

Commit the `openspec/` directory with spec library and per-change artifacts. Rejected because:

- **Code review burden.** Every PR would include hundreds of additional lines of spec Markdown. Reviewers would need to either verify specs match the implementation (doubling review work) or skip them (leaving specs unverified). Review throughput is already a bottleneck.
- **Spec drift.** Keeping specs accurate requires running `openspec archive` after every change and updating specs when implementation pivots mid-task. Both are manual steps that depend on consistent discipline across all contributors. When drift occurs, reconciliation competes with shipping features.
- **Overhead for small changes.** Community experience shows OpenSpec generates substantial artifacts even for simple bug fixes. Fleet ships a mix of trivial and complex changes, and the overhead is disproportionate for the former.

### Adopt OpenSpec for large features only

Use OpenSpec selectively for multi-component features while skipping it for small changes. Rejected because:

- Inconsistent adoption creates confusion about when specs are required.
- Large, multi-component features are exactly the cases where specs drift fastest during implementation.
- The spec library would only cover a subset of the system, limiting its value as a source of truth.

### Require a structured "Approach" section in PR descriptions

Add a lightweight template (Problem, Change, Why this layer, What I considered) to PR descriptions for complex changes. Not adopted as a formal process because AI-generated PR descriptions already cover this well without a mandated format.

## References

- [#44402](https://github.com/fleetdm/fleet/issues/44402) - Evaluation issue
- [OpenSpec documentation](https://openspec.dev/)
- [ThoughtWorks Technology Radar, Vol. 34](https://www.thoughtworks.com/radar/tools/openspec) - "Assess" ring
