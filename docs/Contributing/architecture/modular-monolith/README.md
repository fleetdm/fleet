# Modular monolith architecture

This is the central documentation for Fleet's transition to a modular monolith architecture. It provides an overview of the approach and the current status.

## Overview

Fleet's Go codebase has grown to over 600,000 lines of code across 2,300+ Go files. To improve maintainability, team scalability, and code quality, we are transitioning from a traditional layered architecture to a **modular monolith** with well-defined **bounded contexts**.

### What is a modular monolith?

A modular monolith is an architectural pattern where:
- Code is organized into cohesive, loosely-coupled bounded contexts
- Each bounded context owns its complete vertical slice: handlers → service logic → data access
- Bounded contexts communicate through well-defined public interfaces, not internal implementation details
- The application still deploys as a single binary

This approach provides the benefits of strong boundaries (clear ownership, better testability, reduced coupling) without the operational complexity of distributed systems.

### Industry precedents

| Company | Codebase | Approach |
|---------|----------|----------|
| **GitLab** | Ruby, 2.2M+ LOC | [Modular monolith](https://handbook.gitlab.com/handbook/engineering/architecture/design-documents/modular_monolith) with bounded contexts |
| **Kubernetes** | Go, 500K+ LOC | [30+ controllers](https://github.com/kubernetes/kubernetes/tree/master/pkg/controller) in single binary with logical separation |
| **Shopify** | Ruby, 3M+ LOC | [Modular monolith](https://shopify.engineering/shopify-monolith) with component boundaries |

## Current status

### Bounded contexts

| Context | Status | ADR | Implementation |
|---------|--------|-----|----------------|
| **Activity** (audit) | Pilot in progress | [ADR-0007](../../adr/0007-pilot-activity-bounded-context.md) | [#36452](https://github.com/fleetdm/fleet/issues/36452) |

### Candidates for future extraction

These bounded contexts are candidates for future extraction, based on domain boundaries and team ownership:

- **Identity** - Users, sessions, and authentication
- **Vulnerabilities** - Vulnerability detection and management
- **Software** - Software inventory, installation, and updates
- **Policies** - Policy definitions and compliance
- **Scripts** - Script execution and batch activities
- **Queries** - Query management and scheduling

## Architecture principles

### Bounded context ownership

Each bounded context owns its **complete vertical slice**.

**Key principles:**
- Other contexts can only import the root package (public interface), not the `service/` or `mysql/` subdirectories
- The context that owns the API endpoint is responsible for orchestrating operations that span multiple bounded contexts

### Database guidelines

| Guideline | Details |
|-----------|---------|
| **Single shared database** | All contexts share one MySQL database |
| **Exclusive write ownership** | Each context owns writing to its tables exclusively |
| **Table prefixes** | Tables use context name as prefix (e.g., `activity_past`, `software_titles`) |
| **Cross-context reads allowed** | Joins for read operations are permitted but indicate coupling |
| **No cross-context transactions** | Each transaction must be scoped to a single context's tables |

### Communication patterns

- Use public service layer methods only (e.g., `activity.NewActivity()`)
- Never call another context's datastore methods directly
- Async communication patterns to be defined in future ADR

### Architectural enforcement

We enforce boundaries using architecture tests that validate import restrictions.

These tests run as part of the regular CI test suite.

## Reference patterns

The following patterns emerged from the Activity bounded context pilot. Bounded context owners are free to adopt these if they find them useful. They are not requirements.

### Directory structure

Use Go's `internal` directory to enforce public vs private boundaries:

```text
/server/{context}/
├── bootstrap/                 # Public: dependency injection entry point
│   └── bootstrap.go
├── api/                       # Public: service interface and request/response types
│   └── service.go
└── internal/                  # Private: implementation details (cannot be imported externally)
    ├── types/                 # Internal interfaces (e.g., Datastore)
    ├── service/               # Service implementation and HTTP handlers
    ├── mysql/                 # Database implementation
    ├── tests/                 # Integration tests
    └── testutils/             # Test utilities
```

The `internal` directory is enforced by the Go compiler. Code outside the bounded context cannot import it.

The `bootstrap` package implements the **composition root** pattern. It is the single place where all dependencies are wired together. Code outside the bounded context calls `bootstrap.New()` to instantiate the fully-configured service.

### Anti-corruption layer (ACL)

When a bounded context needs data from another context (or legacy Fleet code), you may use an ACL to translate between domains. This isolates the bounded context from external types and provides a single integration point.

ACLs are temporary by design. Once the dependency exposes a clean public API that does not bring in large coupling or transitive dependencies, the ACL can be removed and replaced with a direct import.

Location: `/server/acl/{context}acl/`

```go
type FleetServiceAdapter struct {
    svc fleet.UserLookupService  // External dependency
}

func (a *FleetServiceAdapter) UsersByIDs(ctx context.Context, ids []uint) ([]*activity.User, error) {
    users, err := a.svc.UsersByIDs(ctx, ids)  // Call external service
    // ... translate fleet.User to activity.User ...
    return result, nil
}
```

Benefits:
- Single point of integration with external code
- Bounded context remains decoupled from external types
- Easy to replace when a clean API becomes available

### Dependency injection for authentication and authorization

Bounded contexts should not own authentication or authorization. These are injected from outside.

**Authentication** (verifying who the user is) is injected as HTTP middleware. The bootstrap function returns a routes function that accepts the auth middleware:

```go
// bootstrap/bootstrap.go returns a function that accepts auth middleware
func New(...) (api.Service, func(authMiddleware endpoint.Middleware) eu.HandlerRoutesFunc) {
    // ...
    routesFn := func(authMiddleware endpoint.Middleware) eu.HandlerRoutesFunc {
        return service.GetRoutes(svc, authMiddleware)
    }
    return svc, routesFn
}
```

**Authorization** (checking permissions) is injected as a service dependency. The bounded context calls the authorizer but does not implement it:

```go
// bootstrap/bootstrap.go receives authorizer as a parameter
func New(
    dbConns *platform_mysql.DBConnections,
    authorizer platform_authz.Authorizer,  // Injected from outside
    // ...
) {
    svc := service.NewService(authorizer, ds, userProvider, logger)
}
```

### Test organization

**Unit tests**: Colocated with implementation files, use mocks for all dependencies.

**Integration tests**: In `internal/tests/` directory, use real database and HTTP server with mock auth.

## Glossary

| Term | Definition |
|------|------------|
| **Bounded context** | A DDD concept representing a cohesive domain with clear boundaries. In Fleet, a group of Go packages under a common directory. |
| **Cross-cutting concern** | Functionality that affects multiple parts of the system (e.g., activity logging, authentication). |
| **Modular monolith** | An architectural pattern where code is organized into bounded contexts within a single deployable binary. Not related to Go modules. |
| **Ubiquitous language** | Using the same terminology consistently across code and business domains. |
| **Vertical slice** | The complete implementation stack for a feature: handlers → service → datastore → database. |

## FAQ

### How do I handle functionality that spans multiple contexts?

The context that owns the API endpoint orchestrates the operation. If writes to multiple contexts are needed, perform them as separate transactions. Use compensating transactions (Saga pattern) only as a last resort for critical consistency requirements.

### What happens if I need to import implementation details from another context?

This is a sign that either:
1. The functionality belongs in a shared package
2. The bounded context boundaries need adjustment
3. A public interface should be added to the other context

Discuss with the tech leads before bypassing boundaries.

### What's the difference between a module and a bounded context?

A module is a generic term for organizing cohesive code. A bounded context has hard boundaries enforced through public APIs, clear ownership, and restricted imports. We use "bounded context" to emphasize that these are not just convenient code groupings, but intentional boundaries designed to allow teams to work independently as the codebase scales.
