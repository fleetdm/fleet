# Engineering Spec Review Checklist

This document is a checklist for **reviewing design docs and translating them into well-specified engineering subtasks** in the Fleet repository.

The goal is to ensure issues are actionable, testable, and complete before implementation begins, reducing ambiguity and rework.

---

## Global Checklist (Applies to All Sub-Issues)

Before reviewing individual areas, confirm:

- [ ] **Problem & goal are clearly stated**
  - What user or system behavior is changing?
  - What does “done” look like?
- [ ] **Out of scope is explicitly defined**
- [ ] **Assumptions are listed**
- [ ] **Backward compatibility is considered**
- [ ] **Failure modes & edge cases are acknowledged**
- [ ] **Rollout and migration plan exists** (if applicable)
- [ ] **Acceptance criteria are concrete and testable**
- [ ] **Documentation impact is noted** (Fleet docs, API docs, changelog)
- [ ] **Security & permissions implications are reviewed**
- [ ] **Observability impact is noted** (logs, metrics, errors)

---

## Frontend Sub-Issue Checklist

### UX & Behavior

- [ ] Entry points in the UI are defined
- [ ] User roles & permissions are specified
- [ ] Happy path UX is described
- [ ] Error, loading, and empty states are defined
- [ ] Copy/text is provided or guidance is given
- [ ] Feature flag usage is specified (if applicable)

### Data & API Interaction

- [ ] Required API endpoints are listed
- [ ] Request and response shapes are documented
- [ ] Pagination, sorting, and filtering behavior is specified
- [ ] Caching or refetching strategy is noted
- [ ] Real-time updates or polling expectations are defined

### Design System & Consistency

- [ ] Existing components are identified or new ones are justified
- [ ] Consistency with Fleet UI patterns is verified
- [ ] Accessibility considerations are noted (ARIA, keyboard, contrast)

### Testing

- [ ] Unit test expectations are defined
- [ ] Integration / UI test expectations are defined
- [ ] Manual QA steps are outlined

---

## Backend Sub-Issue Checklist (Including REST APIs)

### API Design

- [ ] New vs existing endpoints are clearly identified
- [ ] HTTP methods and routes are specified
- [ ] Request schema is fully defined
- [ ] Response schema is fully defined
- [ ] Error responses and status codes are listed
- [ ] API versioning strategy is noted (if modifying existing APIs)
- [ ] Idempotency considerations are addressed (where applicable)

### Authorization & Security

- [ ] Required permissions and roles are explicitly stated
- [ ] Authentication context usage is defined
- [ ] Input validation rules are documented
- [ ] Potential abuse or misuse scenarios are considered

### Data Model & Storage

- [ ] Database schema changes are specified (tables, columns, indexes)
- [ ] Migrations are described (forward and rollback)
- [ ] Data lifecycle considerations are defined
- [ ] Performance impact is considered (queries, indexing)

### Business Logic

- [ ] State transitions are clearly described
- [ ] Edge cases and invalid states are defined
- [ ] Side effects are documented (events, async jobs, webhooks)
- [ ] Consistency guarantees are explained

### Testing & Docs

- [ ] Unit test expectations are defined
- [ ] Integration test expectations are defined
- [ ] API documentation updates are noted
- [ ] Backward compatibility tests are considered

---

## Agent Sub-Issue Checklist

### Behavior & Compatibility

- [ ] Agent behavior changes are clearly described
- [ ] OS/platform scope is specified (macOS, Windows, Linux, etc.)
- [ ] Minimum supported versions are noted
- [ ] Compatibility with older Fleet servers is considered

### Config & Communication

- [ ] New configuration options are documented
- [ ] Default behavior is specified
- [ ] Server ↔ agent contract changes are explicitly listed
- [ ] Failure and retry behavior is defined

### Performance & Reliability

- [ ] Performance impact is considered (CPU, memory, disk, network)
- [ ] Error handling and logging behavior is defined
- [ ] Offline or degraded-mode behavior is specified

### Rollout & Safety

- [ ] Feature flags or gradual rollout strategy is defined
- [ ] Upgrade and migration behavior is described
- [ ] Recovery strategy for failures is noted

### Testing

- [ ] Unit test expectations are defined
- [ ] Integration or end-to-end test expectations are defined
- [ ] Manual testing steps are included

---

## GitOps Sub-Issue Checklist

### Configuration & Schema

- [ ] YAML schema changes are fully specified
- [ ] Before/after examples are provided
- [ ] Validation rules are defined
- [ ] Default values are clarified

### Sync & Drift Behavior

- [ ] How changes are detected and applied is described
- [ ] Conflict resolution rules are specified
- [ ] Drift detection behavior is explained

### Errors & Safety

- [ ] Failure behavior is defined (partial apply, rollback, stop)
- [ ] Error messages are clear and actionable
- [ ] Safeguards against destructive changes are considered

### Compatibility & Migration

- [ ] Backward compatibility is addressed
- [ ] Migration path is documented
- [ ] Interaction with existing Fleet GitOps flows is reviewed

### Testing & Docs

- [ ] Unit or integration test expectations are defined
- [ ] Example repositories or fixtures are referenced
- [ ] Documentation updates are called out

---

## Reviewer Red Flags

Reviewers should push back if a spec includes:

- ❌ “Implementation details TBD”
- ❌ Missing API request/response shapes
- ❌ “Handle errors” without specifics
- ❌ No migration plan for data or configuration changes
- ❌ Implicit behavior changes not explicitly acknowledged
- ❌ No clear acceptance criteria

---

## Usage

This checklist should be used when:
- Reviewing design docs
- Creating or refining engineering subtasks
- Assessing readiness for implementation

Specs do not need to be verbose, but **they must be complete**.
