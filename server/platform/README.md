# Platform packages

This directory contains **infrastructure and cross-cutting technical concerns** that are independent of Fleet's business domain. These packages provide foundational capabilities used across the codebase.

## Platform vs domain

Following separation of concerns, we distinguish:

- **Platform (infrastructure)**: Technical concerns like database connectivity, HTTP utilities, middleware, and transport-level error handling. These packages have no knowledge of Fleet's business domain.
- **Domain (business logic)**: Feature-specific code organized into bounded contexts. Domain packages depend on platform packages, not the reverse.

## Guidelines

- Platform packages must not import domain packages
- Platform packages should be general-purpose and reusable
- Architectural boundaries are enforced by `arch_test.go`
