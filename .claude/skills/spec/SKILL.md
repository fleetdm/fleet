---
name: spec
description: Spec a GitHub ticket by creating a short behavioral contract through interactive Q&A, then propose subtask breakdown. Use when the user says "spec", "spec a ticket", or "spec this".
user_invocable: true
---

# Spec a Ticket

Create a short, clear behavioral contract for a GitHub ticket so there is a clear path to implementation.

## Inputs

The user provides a GitHub issue URL or number. If just a number, assume it's in the `fleetdm/fleet` repo.

## Process

### Phase 1: Understand the ticket

1. Fetch the ticket using `gh issue view`.
2. Read the ticket title, description, comments, and labels.
3. If needed, explore the relevant parts of the codebase (DB schema, API endpoints, UI components, existing patterns) to ground your understanding in what actually exists.
4. Summarize your understanding back to the user in 2-3 sentences.

### Phase 2: Interactive Q&A (the contract)

Build a short behavioral contract through conversation:

1. Present a **draft contract** with a "Human section" explaining what needs to be done. This should be:
   - Written in plain, direct English
   - Short -- aim for one page or less (unless genuinely complex)
   - Focused on behavior: what happens, not how to code it
   - Deterministic: no ambiguity about expected outcomes
   - For bug tickets, include a **Root cause** section identifying the specific code and failure mechanism

2. Below the draft, list **open questions** -- only things that materially affect behavior. Use this format:
   - Decision needed
   - Why it matters
   - Your recommended default

3. The user will answer, ask you questions, or ask your opinion. Go back and forth until the contract is solid.

**Rules during Q&A:**
- If you must fill a gap yourself, flag it inline (e.g., "Assumed: X -- let me know if this should be different")
- Include 1-2 short examples (input/output or happy path/failure) ONLY when the ticket involves non-obvious logic. Skip for straightforward work.
- Do not ask broad brainstorming questions. Only ask what affects behavior.
- Do not over-specify. If something is obvious from context or existing code patterns, don't spell it out.

### Phase 3: Subtask breakdown

Once the contract is agreed upon:

1. Propose a division into subtasks. Typical splits:
   - Backend / Frontend
   - By code area (API, DB, UI, tests)
   - By topic or feature slice
2. Discuss with the user until the breakdown is agreed upon.

### Phase 4: Write to ticket

1. Ask the user for explicit permission before writing.
2. Once approved, edit the issue description using `gh issue edit --body` to **append** the spec after the existing content. Never replace or reorder the original template content.
3. Ask the user whether to create actual sub-issues or just list them as "Proposed subtasks". If creating sub-issues, use `gh issue create` for each, linking back to the parent ticket.
4. Format for the parent ticket -- add a `---` separator, then:

```
---

## Spec

[The behavioral contract]

## Subtasks

- [ ] #issue_number - Subtask 1
- [ ] #issue_number - Subtask 2
- [ ] ...
```
