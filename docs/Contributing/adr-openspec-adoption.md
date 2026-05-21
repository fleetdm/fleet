# ADR: OpenSpec Adoption

- **Status:** Rejected
- **Date:** 2026-05-21
- **Issue:** [#44402](https://github.com/fleetdm/fleet/issues/44402)

## Context

[OpenSpec](https://openspec.dev/) (by Fission AI, YC W26) is a spec-driven development framework. Teams commit Markdown specification files into an `openspec/` directory in the repo. Each feature or change gets a folder containing a proposal, design doc, task list, and delta specs (Given/When/Then scenarios). When a change ships, deltas are archived and merged into a "spec library" that represents the current state of the system.

The goal is to align AI coding agents and human developers on what to build before code is generated, and to maintain a living specification of system behavior.

We evaluated whether Fleet should adopt OpenSpec and commit spec files alongside our code.

## Decision

We will not adopt OpenSpec.

## Rationale

### Code review is already a bottleneck

Every PR would include hundreds of additional lines of spec Markdown (proposal, design, tasks, delta specs) alongside the actual code changes. Reviewers would face a choice: verify that specs match the implementation (doubling review work) or skip the specs (making them unverified documentation). Neither outcome is acceptable when review throughput is already constrained.

### Spec drift is inevitable and unfunded

OpenSpec requires running an `archive` command after each change ships to merge delta specs into the spec library. This demands consistent discipline across all contributors. Known failure modes:

- If anyone skips the archive step, specs diverge from reality immediately.
- Mid-implementation pivots are not reflected in the spec. This is a documented limitation.
- Cross-cutting refactors require updating many spec folders simultaneously.
- No one owns spec maintenance. When drift occurs, reconciliation competes with shipping features.

### Our existing workflow already provides the key benefit

OpenSpec's primary value is structured thinking before implementation. Fleet already achieves this through:

- **User stories with detailed acceptance criteria** that define what to build and how to validate it.
- **AI-generated PR descriptions** that document the problem, approach, affected layers, and alternatives considered.
- **`CLAUDE.md` conventions** that guide AI agents on Fleet-specific patterns, error handling, auth, and the request lifecycle.

Adopting OpenSpec would create a parallel specification layer that largely restates what already exists in these artifacts, but in committed files that require ongoing maintenance.

### Tool maturity and ecosystem risk

OpenSpec is at v1.2 with a single maintainer at Fission AI. Adopting it as a core workflow dependency introduces bus-factor risk for a tool that could stall, pivot, or introduce breaking changes.

### Overhead disproportionate for small changes

Research and community experience show that OpenSpec generates substantial spec artifacts even for small bug fixes. Fleet ships a mix of trivial and complex changes. The overhead is only potentially justified for large, multi-component features, but those are exactly the cases where specs drift fastest during implementation.

## Consequences

- No `openspec/` directory or spec files will be committed to the Fleet repo.
- We continue using user stories, AI-generated PR descriptions, and `CLAUDE.md` as our primary alignment tools for AI-assisted development.
- We will revisit this decision if: (a) our current workflow breaks down at scale, (b) OpenSpec adds automated spec-to-implementation verification, or (c) the tool matures significantly (stable API, broader maintainer base).
