# ADR-NNNN: Title

## Status

Proposed 

## Date

2025-10-22

## Context

In June 2025, we decided in [ADR-0001: Pilot splitting service layer into separate Go packages](0001-pilot-service-layer-packages.md) to pilot splitting the service layer into separate Go packages, using the Android platform as a proof-of-concept. This ADR proposes to supersede this, changing aspects of how the Android module integrates with the rest of Fleet.

### What works well 

Several aspects of the Android module work well, and should be kept and iterated upon:

- Having a separate Android module provides a good developer experience (DX) in that it:
  - means most implementation work can take place within the module
  - test-dev-test iterations are fast
  - ...
- 

### What doesn't work well

Some aspects of the Android module don't work well, and should be revisited:

- Though modularity is desirable, making the codebase modular along platform lines doesn't align with the product's goal of a single API approach for features across all platforms. This causes integration challenges and makes it difficult to maintain the codebase.
- 

## Decision

Given the above points, we decided to:

- tk

Explain the solution chosen and why it was selected over alternatives.

## Consequences

Describe the consequences of the decision, both positive and negative. This should include:

- Benefits of implementing this decision
- Drawbacks or technical debt incurred
- Impact on existing systems or processes
- Future considerations or follow-up decisions needed

## Alternatives considered

Describe alternative solutions that were considered and why they were not chosen.

For each alternative:
- Describe the alternative approach
- List pros and cons
- Explain why it was not selected

## References

List any references, such as:
- Links to related issues or discussions
- External articles or documentation
- Research or data that influenced the decision