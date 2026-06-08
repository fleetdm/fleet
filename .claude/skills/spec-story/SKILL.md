---
name: spec-story
description: Break down a Fleet GitHub story issue into implementable sub-issues with technical specs. Use when asked to "spec", "break down", or "analyze" a story or issue.
allowed-tools: Bash(gh *), Read, Grep, Glob, Write, Edit, WebFetch(domain:github.com), WebFetch(domain:fleetdm.com), WebSearch
model: opus
effort: high
argument-hint: "<issue-number-or-url>"
---

# Spec a Fleet Story

Break down the GitHub story into implementable sub-issues: $ARGUMENTS

## Process

### 1. Understand the Story
- Fetch the issue with `gh issue view <number> --json title,body,labels,milestone,assignees`
- Read the full description, acceptance criteria, and any linked issues
- Identify the user-facing goal and success criteria
- If the issue references Figma designs, API docs, or external specs, fetch them

### 2. Map the Codebase Impact
Search the codebase to understand what exists and what needs to change:
- Find existing implementations of related features (Grep for key terms)
- Identify the tables, service methods, API endpoints, and frontend pages involved
- Check migration files and `server/fleet/datastore.go` for relevant schema
- Trace the request flow: API endpoint → service method → datastore → frontend

### 3. Identify Sub-Issues
Decompose into atomic, implementable units. Each sub-issue should be:
- Completable independently (or with clearly stated dependencies)
- Testable with specific acceptance criteria
- Scoped to one layer when possible (backend, frontend, or migration)

Common decomposition patterns for Fleet:
- **Database migration** — new tables or columns needed
- **Datastore methods** — new or modified query functions
- **Service layer** — business logic, authorization, validation
- **API endpoint** — new or modified HTTP endpoints
- **Frontend page/component** — UI changes
- **fleetctl/GitOps** — CLI and GitOps YAML support
- **Tests** — integration test coverage for the feature
- **Documentation** — REST API docs, user-facing docs

### 4. Write Each Sub-Issue Spec

For each sub-issue, write:

```markdown
## Sub-issue N: [Title]

**Depends on:** [sub-issue numbers, or "none"]
**Layer:** [migration | datastore | service | API | frontend | CLI | docs | tests]
**Estimated scope:** [small: <2h | medium: 2-8h | large: >8h]

### What
[1-3 sentences describing the change]

### Why
[How this contributes to the parent story's goal]

### Technical Approach
- [Specific files to create or modify]
- [Key functions, types, or patterns to follow]
- [Reference existing similar implementations]

### Acceptance Criteria
- [ ] [Testable criterion 1]
- [ ] [Testable criterion 2]
- [ ] [Tests pass: specific test commands]

### Open Questions
- [Any ambiguity that needs product/design input]
```

### 5. Produce the Dependency Graph
Show which sub-issues depend on which:
```
Migration → Datastore → Service → API → Frontend
                                      → CLI/GitOps
                                      → Docs
```
Note which sub-issues can be parallelized.

### 6. Write the Output
Create a spec document with:
1. **Summary** — one paragraph overview
2. **Sub-issues** — each with the template above
3. **Dependency graph** — visual ordering
4. **Open questions** — anything that needs clarification before implementation begins
5. **Suggested PR strategy** — single PR vs multiple, review order

## Rules
- Every sub-issue must reference specific files and patterns from the codebase
- No vague specs: "implement the backend" is not a sub-issue
- If you find ambiguity in the story, flag it as an open question rather than guessing
- Check for related existing issues with `gh issue list --search "keyword" --limit 10`
- Consider Fleet's multi-platform nature: does this affect macOS, Windows, Linux, iOS, Android?
- Consider enterprise vs core: does this need license checks?
