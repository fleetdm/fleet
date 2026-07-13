---
name: spec-story
description: Break down a Fleet GitHub story issue into implementable sub-issues with technical specs. Use when asked to "spec", "break down", or "analyze" a story or issue.
allowed-tools: Bash(gh *), Bash(git log*), Bash(git blame*), Bash(git show*), Bash(git diff*), Read, Grep, Glob, Write, Edit, Agent, WebFetch(domain:github.com), WebFetch(domain:fleetdm.com), WebSearch, mcp__claude_ai_Figma__*, mcp__claude_ai_Slack__*, mcp__claude_ai_Gong__*
model: opus
effort: high
argument-hint: "<issue-number-or-url>"
---

# Spec a Fleet Story

Break down the GitHub story into implementable sub-issues: $ARGUMENTS

## Process

This skill proceeds in **four stages** with explicit user-approval gates between each. Do not skip ahead. Do not collapse stages. After each stage, summarize what you produced and **stop** to ask the user for approval before moving on.

```
Stage 1: Understand  →  [user approves]  →
Stage 2: Iterate on the spec skeleton  →  [user approves]  →
Stage 3: Draft the sub-issues  →  [user approves]  →
Stage 4: Create in GitHub
```

If the user pushes back at any gate, stay in that stage and iterate. Do not advance until they explicitly say so.

---

## Stage 1 — Understand the story

**Goal:** build a deep, shared understanding of what's being asked, what surfaces it touches, and what context exists in the codebase, GitHub, Slack, and git history.
**Output:** a written understanding summary the user confirms before decomposition begins.

### 1.1 Gather required inputs (gating)

Before fetching the issue, exploring the codebase, or producing any output, confirm with the user that you have:

- **A Figma link** for any design surface the story touches (or explicit confirmation there is no design — e.g. backend-only stories)
- **The documentation change PR(s)** that accompany the story (or explicit confirmation there are none)

If either input is missing and the user has not explicitly waived it, **stop and ask**. Do not infer, do not proceed, do not start mapping the codebase. Phrase the question plainly and list each missing input.

Once inputs are provided:
- For each Figma URL, parse out the `fileKey` and `nodeId` and call the Figma MCP (`mcp__claude_ai_Figma__get_design_context`, `get_screenshot`, `get_metadata` as appropriate). Inspect carefully — capture every state, variant, empty/error/loading state, and any annotations or Code Connect mappings. Note design tokens used.
- **Find the Ready page — the cover is not the design.** A Fleet story's Figma link usually points at the `ℹ️ Cover` page (often `node-id=0-1`), which only links back to the GitHub issue. The actual screens and dev notes live on the `✅ Ready` page. Do not stop at the cover, and do not conclude "no design exists" if `get_metadata` (no nodeId) lists only the cover — that page listing under-reports, so a cover-only result is not proof the design is missing. Get the Ready page by node ID: ask the user for its node URL (`...?node-id=<id>`, from Figma's "Copy link to selection"), then call `get_metadata` on that node to get the structure. Screenshot the whole Ready section, and call `get_design_context` on each `Dev note` callout and every tooltip and label to extract copy verbatim. Dev notes (e.g. "Use .py file icon on software details page") frequently change the implementation and must be captured.
- For each documentation PR, fetch it with `gh pr view <number> --json title,body,files` and read the diff to understand the user-facing surface area being documented.

### 1.2 Understand the issue
- Fetch the issue with `gh issue view <number> --json title,body,labels,milestone,assignees`
- Read the full description, acceptance criteria, and any linked issues
- Identify the user-facing goal and success criteria
- If the issue references Figma designs, API docs, or external specs, fetch them
- **Find the t-shirt size on the parent story.** Check the issue's labels (e.g. `~size:s`, `~size:m`, `~size:l`) and body for an explicit t-shirt estimate. If none is present, flag it as an open question — do not invent one. Capture the size only; do not derive or assign story points from it.

### 1.3 Map the codebase impact

**Always start here — read the schema and migrations FIRST, before grepping code paths.** Fleet's database shape dictates everything downstream, so grounding the spec in the real table structure (rather than inferring it from scattered Go code) is faster and more accurate, and anchors every later decision (sub-issue boundaries, migration scope, conditions of satisfaction) in actual columns, types, and constraints. Stories that touch no tables are extremely rare; do this regardless:

- **Schema.** Grep the affected tables in `server/datastore/mysql/schema.sql` — the generated, always-current dump of every table (regenerated from migrations by `make dump-test-schema`). Pull the exact columns, types, indexes, and foreign keys for each table the story touches, e.g. ``grep -A40 'CREATE TABLE `software_installers`' server/datastore/mysql/schema.sql``. Grep the specific tables; do not load the whole file.
- **Migrations.** Skim the most recent files in `server/datastore/mysql/migrations/tables/` (sorted by timestamp) for in-flight or adjacent schema changes the dump may not reflect yet, and to read the *intent* behind recent columns (the `Up` function and its comments).

Then map the rest of the surface:
- Find existing implementations of related features (Grep for key terms)
- Identify the service methods, API endpoints, and frontend pages involved; use `server/fleet/datastore.go` for the datastore interface (what methods exist)
- Trace the request flow: API endpoint → service method → datastore → frontend

### 1.4 Research prior art and history

For each relevant codepath identified in 1.3, and for the feature concept itself, gather context that will not appear in the issue body:

- **GitHub.** Search issues, PRs, and discussions for prior mentions of the feature, related bugs, and earlier attempts:
  - `gh issue list --search "<keywords>" --state all --limit 20`
  - `gh pr list --search "<keywords>" --state all --limit 20`
  - `gh search issues "<keywords>" --repo fleetdm/fleet --limit 20`
  - Pay special attention to **in-flight doc PRs** (REST API, GitOps YAML, audit log, usage stats) — they often define field names and shapes that the implementation must match. Identify all of them; note which fields they introduce and where they place them. Conflicts between doc PRs are common and require explicit reconciliation in Stage 3.
  - Read the most relevant results in full; capture any decisions, constraints, or rejected approaches.
- **Slack.** Use `mcp__claude_ai_Slack__slack_search_public_and_private` (or `slack_search_public` if private access is unavailable) to find conversations about the feature. Look for product, eng, and customer threads. Read full threads with `slack_read_thread` when something looks load-bearing — design rationale, customer asks, or pushback worth surfacing.
- **Gong.** Find related meetings where the feature, customer ask, or its constraints were discussed. Use `mcp__claude_ai_Gong__search_calls` to locate sales, customer success, or product calls by keyword, then `mcp__claude_ai_Gong__search_transcript` or `summarize_transcript` to pull the relevant moments (and `retrieve_transcripts` for the full text when a call is load-bearing). Customer rationale, commitments, and prioritization context often live in calls, not in the issue or Slack — surface anything that reshapes scope.
- **Git history.** For every file or directory you expect to touch, inspect history to understand why the current shape exists:
  - `git log --oneline -- <path>` for the change list
  - `git log -p -- <path>` for the full diff history when scoping a refactor
  - `git blame <path>` for line-level provenance on tricky sections
  - Read commit messages and linked PRs for prior intent — Fleet conventions and reversed decisions are often explained in commit bodies, not in code.
- **Subject-matter experts (SMEs).** From the affected surfaces and the PR/Slack history, identify the engineer(s) closest to the system being changed: the Apple MDM engineer for AccountConfiguration / SCEP / DEP work; the Windows MDM engineer for MS-MDM / Autopilot / Azure AD work; the agent (orbit) lead for fleetd changes; the frontend lead for new pages or major component changes. Use `git blame` and recent PR authorship as signals. Maintain a list — these are the SMEs the user must consult in Stage 2.3 before the spec is finalized. The SMEs almost always raise points that reshape the decomposition.

### 1.5 Stage 1 gate — present understanding and pause

Write an understanding summary that proves you have **synthesized** the inputs — not paraphrased them. The summary must include:

- **Story restatement** — the user story in one sentence, verbatim from the parent issue.
- **Plain-language synthesis** (2–4 paragraphs) — what the feature does end-to-end, in your own words, surfacing the load-bearing details an implementer needs to know (which command, which surface, which platform constraint, which permission tier).
- **Critical scoping decisions, with rationale** — pair every constraint with the *why*. Format each as a short bolded callout, e.g.:
  - **ADE only.** The `AccountConfiguration` command's `AutoSetupAdminAccounts` key creates the account during Setup Assistant — only possible on ADE-enrolled devices.
  - **Premium only.** `UpdateMDMAppleSetup` returns `ErrMissingLicense` in core; enterprise implementation in `ee/server/service/`.
  - **No Secure Token in v1.** Apple grants Secure Token only when plaintext is sent. We send a hash; this is acceptable because <reason>. May revisit post-v1.
- **Affected surfaces** — UI pages, API endpoints, services, datastore methods, migrations, MDM commands, CLI/GitOps, agent — listed concretely.
- **Mermaid sequence diagram** — required for any feature spanning multiple systems (UI ↔ API ↔ DB ↔ worker ↔ external service ↔ device). Show every actor on its own swimlane and the operations between them in order. Group operations under `note over X,Y: <step>` blocks. For pure-frontend or pure-backend single-surface stories where a sequence diagram adds no information, document why it was omitted; otherwise produce one.
- **Parent t-shirt size** — the parent story's t-shirt size from its labels/body (or "not set — open question"). Do not convert it to points.
- **Prior decisions, constraints, and related work** surfaced in 1.4 — including in-flight doc PRs and known field-name conflicts.
- **SMEs to consult in Stage 2.3** — by name and area.
- **Open questions** that block decomposition.

**Stop and ask the user to confirm or correct your understanding.** Do not move to Stage 2 until they explicitly approve. If the user supplies missing context (a hidden constraint, a wrong scoping decision, an additional affected surface), update the summary in place and re-confirm.

---

## Stage 2 — Iterate on the spec skeleton

**Goal:** agree with the user on the sub-issue breakdown and dependency graph before writing any prose.
**Output:** an approved skeleton — sub-issue index (titles, layers, labels, type, depends-on) and dependency graph. **No full sub-issue bodies yet.** Sub-issues are not pointed — the skeleton carries no story-point estimates. The parent story's t-shirt size, if set, is captured for context.

### 2.1 Identify sub-issues

Decomposition is shaped by **two forces in tension**. Hold both at once — getting either wrong produces an unworkable spec.

**Force 1 — Cohesion: keep tightly-coupled work together.**
If two pieces can only be implemented and tested as a unit, they belong in one sub-issue. Splitting them produces fake parallelism: the second sub-issue can't start, can't be tested in isolation, and yields an unmergeable PR. Examples of work that should stay together:
- A new service method and the datastore call it requires
- An API endpoint and its request/response struct types
- A migration and the goqu queries that read the new column

**Force 2 — Specialization: split by skill so each sub-issue has one natural owner.**
Backend (Go, MySQL, services, datastore, API) and frontend (React/TypeScript, pages, components) must not be mixed in the same sub-issue. The team has a limited number of people with each skill, the reviewers are different, and the PRs ship independently behind feature flags. A sub-issue that requires both backend and frontend expertise can't be assigned cleanly.

**The synthesis.** Split along specialization boundaries first — backend, frontend, fleetctl/GitOps, agent (orbit), and the combined docs/QA — then within each specialization, keep tightly-coupled work in a single sub-issue. Migrations bundle with their owning backend sub-issue; they are not their own specialization. Don't atomize within a specialization for its own sake.

**Heuristics for testing your decomposition:**
- If sub-issue B's PR description would naturally say "requires sub-issue A merged first to compile/run/test," merge them — that's a cohesion failure.
- If a single reviewer would need both backend and frontend expertise to approve a sub-issue, split it — that's a specialization failure.
- A backend sub-issue should ship a working, tested API surface that the frontend can mock against. A frontend sub-issue should consume that contract and be reviewable on its own.
- See https://github.com/fleetdm/fleet/issues/31138 for an anti-pattern: that spec splits work that can only be done together (cohesion failure) and mixes frontend with backend within sub-issues (specialization failure). Do not produce a spec shaped like that.

**Common Fleet specializations — group work into sub-issues by these:**
- **Backend (Go)** — migrations, datastore methods, service layer, API endpoints, MDM commands. Bundle these into a single backend sub-issue when they're tightly coupled (a typical full-stack story produces one foundational backend sub-issue and one integration backend sub-issue, not four flat layers).
- **Frontend (React/TypeScript)** — pages, components, frontend services. Split by surface (e.g., one sub-issue for the Controls page, one for Host details) when surfaces are independent.
- **fleetctl/GitOps (Go)** — CLI and GitOps YAML support, including round-trip export.
- **Agent / orbit (Go)** — agent-side changes.
- **Documentation and engineering QA (`docs/QA`)** — combined into a single mandatory final-gate sub-issue (see below).

Within a single specialization, prefer one sub-issue that delivers an end-to-end vertical slice of that layer (e.g., "datastore methods + service layer + API endpoint for X" can be one backend sub-issue if the pieces are tightly coupled and a single backend engineer would naturally do them in one PR) over three brittle sub-issues that block each other.

**Mandatory sub-issue for every story — `Documentation and engineering QA`.**

Every spec ends with a single combined sub-issue covering:
- REST API docs (`docs/REST API/rest-api.md`)
- Audit log reference (`docs/Contributing/reference/audit-logs.md`) for any new activity types
- Usage statistics guide (`articles/fleet-usage-statistics.md`) for any new toggles
- Feature guide updates (e.g., `articles/`, `https://fleetdm.com/guides/...`)
- End-to-end engineering QA on a real device, performed once all implementation PRs (1..N-1) have merged

Layer: `docs/QA`. Labels: `#g-software`, `~sub-task`, no `~frontend`/`~backend` (this sub-issue verifies the whole story across surfaces, and is owned by whoever is shipping the feature, not by a frontend or backend specialist). Type: `Task`.

Required even if the story looks small. This is the final-gate sub-issue, depending on every implementation sub-issue.

### 2.2 Produce the dependency graph
Show which sub-issues depend on which. A typical specialization-first decomposition (foundational backend → integration backend → frontend surfaces in parallel → docs/QA gate) looks like:
```
                              ┌──►  Frontend: <surface A>
                              │
[1] Backend foundation  →  [2] Backend integration  ──►  [N] Documentation and engineering QA
                              │
                              ├──►  Frontend: <surface B>
                              │
                              └──►  fleetctl/GitOps
```
The exact shape depends on the story. Frontend surfaces, fleetctl/GitOps, and other parallel tracks all unblock once the backend integration sub-issue establishes the API contract — they can begin development from the contract while sub-issue 2 is in review. The docs/QA sub-issue is always the final gate; it depends on every implementation sub-issue.

### 2.3 Consult the subject-matter expert(s)

The user (the spec lead) is one reviewer. Domain experts are another, and their input often reshapes the decomposition more than any other input. Before the Stage 2 gate, prompt the user to share the skeleton + understanding summary with the SMEs identified in 1.4 (Apple/Windows MDM lead, agent lead, frontend lead, etc.).

When the user returns with SME feedback:
- Capture each point as a numbered note with the SME's name in parentheses (e.g., "1. **Password hash, not plaintext in MDM command.** ... (Jordan)").
- Incorporate the points into the spec where they affect decomposition, sub-issue scope, or technical approach. Common reshapes: a sub-issue is bundled because the SME flags hidden coupling that defeats parallelism; a sub-issue is split because two pieces are owned by different SMEs; an open question is resolved by SME knowledge that wasn't in the issue body; a v1/v2 boundary is drawn around a concern (e.g., Secure Token, password rotation) the SME confirms is acceptable to defer.
- **Preserve the SME's points verbatim** in a section that will appear in the final spec doc as **Expert review notes** — even when fully incorporated. This serves as a record of the decisions and the reasoning behind them, and prevents future re-litigation.

If the user indicates SME consultation is unnecessary (e.g., the change is trivial or the user is the SME), document that explicitly with a one-line rationale. Otherwise, do not skip this step.

### 2.4 Stage 2 gate — present the skeleton and iterate

Present the skeleton to the user as a draft:
- **Parent t-shirt size** — note the parent story's t-shirt size (or "not set"). No story points anywhere — sub-issues are not pointed.
- **Sub-issue index** — title (using the `<feature>: <area>` format from 3.2), layer (`backend`, `frontend`, `backend/CLI`, or `docs/QA`), labels (`#g-software` and `~sub-task` always; plus `~frontend` or `~backend` for implementation sub-issues; the `docs/QA` sub-issue gets neither surface label), type (`Task`), depends-on, parallel-with
- **Dependency graph** — ordering and parallelism
- **Multi-engineer plan** — for stories spanning ≥4 sub-issues, sketch how the work parallelizes for the realistic team sizes (typically 2 and 3 engineers) so the user can validate the plan against actual headcount
- **Expert review notes** captured in 2.3 (or "SME consultation deferred — <reason>")
- **Open questions** — anything still ambiguous

Expect pushback on scope, decomposition boundaries, or ordering. Iterate with the user — don't defend the first cut. **Do not draft full sub-issue bodies until the user explicitly approves the skeleton.**

---

## Stage 3 — Draft the sub-issues

**Goal:** produce the full spec document — synthesis, mermaid diagram, Figma extraction, expert notes, engineering checklist answers, deep technical narratives, sub-issues with rich Task and Condition of Satisfaction sections, dependency graph, PR strategy, multi-engineer plans, and resolved/open questions.
**Output:** a complete spec document at the depth of `37141-spec-managed-local-account.md`, awaiting the user's approval.

### 3.1 Apply Fleet's writing style

Before drafting any sub-issue prose, fetch and apply Fleet's writing guidance:
- https://fleetdm.com/handbook/company/writing — general voice, tone, and structure conventions
- https://fleetdm.com/handbook/marketing/fleet-ai-writing-instructions — AI-specific writing rules (hedging, clichés, formatting, banned phrases)

Apply these to the Task and Condition of satisfaction sections, and to every other piece of prose in the spec output (summary, open questions, PR strategy).

### 3.2 Render each sub-issue

Each sub-issue has **two presentations** — one for the spec doc the user reviews, one for the body posted to GitHub via `gh issue create`. Generate both.

**Title format.** Every sub-issue title leads with a common feature short-name and a colon, then a specific area suffix. Example: `"Managed local account: DB migration, types, datastore, and MDM command primitives"`. The prefix groups sub-issues in the issue tracker; the suffix says exactly what this one ships.

**Spec-doc presentation** (what the user reviews):

```markdown
## Sub-issue N: <title>

**Related user story:** #<parent>
**Depends on:** <sub-issue numbers, or "none">
**Parallel with:** <sub-issue numbers, or "none">

<Description paragraph — 2–4 sentences summarizing what this sub-issue ships, in plain language. This is the issue body's opening paragraph when posted to GitHub.>

### Task

<Detailed task content — see "Task section depth" below>

### Condition of Satisfaction

<Bulleted checklist — see "Condition of Satisfaction depth" below>
```

**GitHub-filed body** (what `gh issue create --body-file` posts). Read the canonical sub-task template at `.github/ISSUE_TEMPLATE/sub-task.md` and fill in its sections — do not reproduce it from memory, so the body always matches the current template if its headings or HTML comments change. Preserve the template's comments verbatim as read from the file, and map the spec-doc content onto its sections:

- **Related user story** → `#<parent>`
- **Task** → the description paragraph from the spec-doc presentation, then the detailed task content
- **Condition of satisfaction** → the bulleted checklist

Labels and issue type are applied separately in Stage 4.1.

The `Depends on` and `Parallel with` metadata lines appear in the spec doc only — not in the GitHub body. The description paragraph from the spec doc becomes the opening paragraph of the GitHub Task section, above any sub-headings or code blocks.

**Task section depth.** A Task section is implementation guidance, not a paraphrase of the issue. It should include:

- **Sub-headings (`####`)** breaking the section into work areas — e.g., "Enrollment worker", "MDM ack handler", "Enterprise settings toggle", "API endpoint", "Host detail response enrichment". One sub-heading per area for any sub-issue larger than a page.
- **Exact file paths with line numbers** where they exist: `server/fleet/app.go:545`, `server/worker/apple_mdm.go:237`, `server/datastore/mysql/apple_mdm.go:6109`. Verify line numbers from current code with Read/Grep; do not guess. If line numbers are likely to drift before implementation, anchor on a function or symbol name as well.
- **Code blocks** in the relevant language for every non-trivial change: Go for backend, SQL for migrations, TypeScript for frontend, XML/plist for MDM commands, JSON for API responses. Show actual structs, function signatures, query bodies, and request/response shapes — not paraphrases.
- **References to existing patterns** the implementer should follow: ``"follow `getHostRecoveryLockPasswordEndpoint` at `server/service/hosts.go:3939`"``, ``"same pattern as `SetHostsRecoveryLockPasswords` at `apple_mdm.go:7460`"``. Pattern references compress hundreds of lines of unwritten guidance.
- **"Why" rationale, including negative-space reasoning.** When a design choice diverges from an apparent alternative, document why the alternative was not chosen. Use bolded headings like ``"**Why a new table (not extending `host_recovery_key_passwords`):**"``, `"**Why no `username` column:**"`, `"**Why no `ExpandHostSecrets` is needed:**"`. This prevents future implementers from re-litigating decisions that have already been settled.
- **Edge cases and guards** identified during research — platform guards (`isMacOS(args.Platform)`), nil-safety (`appCfg may be nil at this point because…`), license checks (`license.IsPremium(ctx)`), pre-existing host behavior, ack handlers that must distinguish this command from a sibling use of the same MDM verb, etc.
- **Required follow-up commands** when interface or generation changes are introduced (e.g., ``"Run `make generate-mock` after datastore interface changes"``).

Heuristic: if the Task section reads like a generic todo list ("implement the endpoint", "add the migration"), it is not deep enough. The bar is the example at https://github.com/fleetdm/fleet/issues/37141 with notes at `37141-spec-managed-local-account.md` — if your sub-issue's Task section is materially shorter or less concrete than the example's, revise.

**Condition of Satisfaction depth.** A bulleted checklist of testable behaviors and tests, grouped by surface or scenario (with bolded sub-group labels: "**Enrollment + ack:**", "**Settings:**", "**API + host response:**", "**End-to-end integration test:**"). Include:

- **Specific test commands** with environment variables — `MYSQL_TEST=1 go test ./server/datastore/mysql/...`, `MYSQL_TEST=1 REDIS_TEST=1 go test ./server/service/... ./server/worker/...`, `yarn test`.
- **Behavioral assertions** with concrete inputs/outputs — ``"`end_user_local_account_type: \"standard\"` returns validation error; only `\"admin\"` accepted"``.
- **Negative cases** — manual enrollment hidden, hosts enrolled before feature enabled, non-darwin hosts, non-premium tier, GitOps disabled state.
- **End-to-end integration** when applicable — enrollment → ack → API → host detail.
- **Snapshot/golden assertions** when byte-for-byte stability matters (e.g., MDM plist with no admin account is byte-for-byte unchanged from today).

### 3.3 Assemble the spec document

The spec document the user reviews follows this structure. Sections marked "(when applicable)" are conditional; everything else is required for any non-trivial story. The bar is the example spec at `37141-spec-managed-local-account.md` (against https://github.com/fleetdm/fleet/issues/37141).

1. **Story** — restate the user story verbatim from the parent issue (preserve the `As a... I want... so that...` form), then write 2–4 paragraphs of plain-language synthesis. Surface critical scoping decisions upfront with rationale, formatted as bolded callouts: `**ADE only.** <why>`, `**Fleet Premium only.** <why>`, `**No Secure Token.** <why this is acceptable for v1>`. Each callout pairs the constraint with the reason — never a bare constraint.
2. **Feature design** — the Mermaid sequence diagram from Stage 1.5 (refined as decomposition firms up). Required for multi-system features. Use `note over X,Y: <step number and label>` to mark phase transitions (1. Enable feature, 2. Enrollment, 3. Device picks up command, 4. Device acknowledges, 5. Admin retrieves password). Include all real actors — `Admin`, `UI`, `API`, `EE`, `DB`, `Worker`, `Cmdr`, `Nano`, `Mac` (or the equivalent for the feature). Show actual function/endpoint names on the arrows.
3. **Figma dev notes, tooltip text, and message text** (when applicable) — extract every UI string from Figma verbatim. **One table per surface** (Controls page, Host details Actions dropdown, Modal, Activity feed). Columns: `Element` | `Text`. Include section titles, checkbox labels, tooltips, descriptions, flash messages (enable/disable/error), button labels, error states, and disabled-state tooltips. Beneath each table, capture **Dev notes** as bullet points: visibility rules, ADE-only guards, GitOps disabled state, default values, character/UID constraints. **Do not paraphrase Figma copy** — copy it verbatim, including punctuation.
4. **Expert review notes** — the SME feedback captured in Stage 2.3, preserved as a numbered list with the SME's name in parentheses on the heading line ("MDM engineering review notes (Jordan)"). Even when fully incorporated into the spec, keep the notes here as a record of the decisions and reasoning. If consultation was deferred, state the reason in one line and move on.
5. **Engineering section answers** — reproduce each engineering checklist item from the parent story body and answer it directly. Typical items: **Test plan finalized**, **Contributor API changes**, **Feature guide changes**, **Database schema migrations**, **Load testing**, **Load testing/osquery-perf improvements**, **This is a premium only feature**. These answers are written back to the parent story's Engineering section in Stage 4.4 — they are the single source of truth.
6. **API design note** (when applicable) — when multiple in-flight artifacts (REST API doc PRs, GitOps PRs, audit log PRs) interact with the spec, render a table: `PR | Endpoint | What changes`. Then call out conflicts (e.g., a field placed at the wrong nesting level) and the chosen resolution. Flag any item that still needs confirmation with a specific team. Document any intentionally divergent field names (e.g., REST API `enable_managed_local_account` vs GitOps YAML `enable_create_local_admin_account`) and the `renameto` tag pattern that handles the mapping.
7. **Deep technical narrative** (when applicable) — standalone H2 section(s) for the trickiest aspects of the implementation, with descriptive titles like "How the password reaches the device". One section per topic; not buried inside a sub-issue. Include diagrams, plist/XML examples, struct shapes, and "why this design over the alternative" reasoning. These sections are the future implementer's lifeline when the code-level "why" is non-obvious.
8. **Sub-issues summary** — a single table: `# | GitHub issue title | Layer | Depends on`. Use the title format from 3.2. Layer values: `backend`, `frontend`, `backend/CLI`, `docs/QA`.
9. **Sub-issues** — each rendered with the spec-doc presentation from 3.2 (metadata header → description paragraph → `### Task` → `### Condition of Satisfaction`). Bundle all backend pieces that are tightly coupled into one backend sub-issue, per Stage 2.1. End with the mandatory `Documentation and engineering QA` sub-issue.
10. **Dependency graph** — ASCII visual graph showing dependencies between sub-issues. Make parallel branches visually clear with `┌──►` / `└──►` connectors. Beneath the graph, write 1–3 sentences explaining the critical path and which sub-issues can begin from spec/API contract before their dependencies merge.
11. **PR strategy** — table mapping each PR to dependencies and parallelizable peers: `PR | Sub-issues | Can start after | Parallel with`.
12. **Multi-engineer scenarios** — Gantt-style ASCII for the realistic team sizes. At minimum produce a "With 2 engineers" plan showing one backend track and one frontend track; for larger stories also produce "With 3 engineers". Required for stories spanning ≥4 sub-issues; recommended otherwise.
13. **Within-sub-issue parallelization** (when applicable) — for sub-issues large enough to split between two people on a shared branch (typical of the foundational backend sub-issue), list the independent pieces with their files and start conditions in a `Piece | Files | Can start` table.
14. **Open questions** — anything still ambiguous or blocked on a decision. Each item should be actionable: who decides, what they need to decide. If there are no open questions, write "None." — do not omit the section.
15. **Resolved questions** — decisions made during spec review (often via SME consultation), preserved as a record. Each entry pairs the question with the decision and the rationale, sometimes with forward-compatibility notes ("adding Secure Token later does not require account deletion or recreation because…"). This section prevents re-litigation when implementers ask "wait, why don't we do X?".
16. **Testing tips** (when applicable) — concrete CLI snippets for manually verifying the feature once implemented: `fleetctl api -X PATCH ...`, `dscl . -read /Users/...`, `sysadminctl -secureTokenStatus ...`, `gh issue view ...`. Numbered steps the engineer or QA can follow on a real device.

### 3.4 Verify the draft with an independent subagent

Before presenting, verify the draft's concrete claims using a **separate verification subagent** (via the `Agent` tool) — not an in-context re-read. A fresh agent doesn't share the drafting context's assumptions and has no stake in the draft being right, so it starts skeptical; that independence is what makes the check catch real errors instead of rubber-stamping them.

Spawn one read-only subagent (a general-purpose or `Explore` agent). Give it the drafted spec and this instruction: *treat every concrete claim as wrong until re-confirmed against the source — re-read the actual files, schema, and GitHub; do not trust the spec's prose. Return only what fails or is uncertain.* It must check:
- **References resolve.** Every `file:line` and symbol still exists in current code (Read/Grep); every datastore method named exists in `server/fleet/datastore.go`.
- **Schema is accurate.** Every table and column cited exists in `server/datastore/mysql/schema.sql` with the stated type/constraints, and every proposed migration is consistent with the current table shape.
- **GitHub claims exist.** Every referenced issue/PR number resolves (`gh`); in-flight doc-PR field names match what the spec assumes.
- **Decomposition holds.** The dependency graph is acyclic; no sub-issue mixes backend and frontend; each is a clean vertical slice; the mandatory `Documentation and engineering QA` sub-issue is present.
- **Conflicts resolved.** Every doc-PR / field-name conflict surfaced in research has an explicit resolution in the spec — none silently dropped.
- **Completeness.** Name any affected surface (UI page, endpoint, service, datastore method, migration, MDM command, CLI/GitOps, agent) the decomposition does not cover. Phrase this to find gaps, not to bless coverage.

Fix everything the subagent flags before the gate. If a claim cannot be confirmed, present it as an open question rather than as fact.

### 3.5 Stage 3 gate — pause for approval

Present the full spec to the user. Wait for explicit approval. **Do not create any GitHub issues until the user explicitly approves the drafted sub-issues.** If they ask for revisions, stay in Stage 3 and iterate.

---

## Stage 4 — Create the sub-issues in GitHub

**Goal:** create the issues in GitHub with the correct labels and type, wire them up as native sub-issues of the parent story, and fill in the parent story's Engineering section.
**Output:** created issues wired to the parent, the parent's Engineering section completed, and all numbers and URLs reported back to the user.

### 4.1 Plan and dry-run

For each sub-issue, prepare:
- A title
- A body file containing the three-section template, fully filled in
- The required labels and type (see below)

**Required on every sub-issue:**
- Labels: `#g-software`, `~sub-task`
- Type: `Task` (GitHub issue type, set via `--type Task`, not a label)
- **No milestone.** Do not pass `--milestone`, and do not add sub-issues to a milestone afterward. The milestone lives on the parent story only; sub-tasks inherit their schedule from it.

**Layer-conditional labels:**
- Frontend sub-issues (React/TypeScript pages and components, frontend services): add `~frontend`
- Backend sub-issues (Go services, datastore, migrations, API endpoints, fleetctl/GitOps, server-side tests, agent/orbit): add `~backend`
- The combined `Documentation and engineering QA` sub-issue (layer `docs/QA`): add neither `~frontend` nor `~backend`. It owns verification of the whole story across surfaces and is not aligned to a single specialization.

Print every planned `gh issue create` invocation as a dry-run summary. Example:

```
gh issue create \
  --title "<sub-issue title>" \
  --body-file <path-to-rendered-body.md> \
  --label "#g-software" \
  --label "~sub-task" \
  --label "~backend" \
  --type "Task"
```

### 4.2 Stage 4 gate — final confirmation

Ask the user to approve the dry-run. Wait for explicit go-ahead. Do not run any `gh issue create` commands before approval.

### 4.3 Create

Run the approved invocations (no `--milestone`). Capture each new issue number and node ID.

### 4.4 Wire up sub-issues and update the parent story

After the issues are created, do both of these — do not leave them to the user:

**Wire each sub-issue as a native task off the parent story.** Use the GitHub sub-issues API so the children appear under the parent's Sub-issues / Tasks list, not just as a markdown reference. For each child:
```bash
PARENT_ID=$(gh issue view <parent> --repo fleetdm/fleet --json id -q .id)
CHILD_ID=$(gh issue view <child> --repo fleetdm/fleet --json id -q .id)
gh api graphql -H "GraphQL-Features: sub_issues" -f query='
  mutation($parent:ID!,$child:ID!){ addSubIssue(input:{issueId:$parent, subIssueId:$child}){ subIssue { number } } }' \
  -f parent="$PARENT_ID" -f child="$CHILD_ID"
```
Confirm with `gh issue view <parent> --json subIssuesSummary` (expect `total` equal to the number of sub-issues created).

**Fill in the parent story's Engineering section.** Write the Stage 3.3 (item 5) engineering answers directly into the parent issue: tick the checkboxes, replace each `TODO` with the answer, and resolve the Risk assessment items. Fetch the body (`gh issue view <parent> --json body -q .body > /tmp/parent_body.md`), edit it in place (preserve all HTML comments and the test plan verbatim), and push it back (`gh issue edit <parent> --body-file /tmp/parent_body.md`). Add the sub-issue links at the top of the Engineering section.

Finally, report each created issue with its number and URL.

## Rules

### Bar
- The paradigmatic example is `37141-spec-managed-local-account.md` against https://github.com/fleetdm/fleet/issues/37141. Every spec you produce should have comparable depth, structure, and concreteness. If your output is materially shorter or less concrete than the example, revise.

### Process gating
- **Always** gate the entire process on Stage 1.1: ask for the Figma link(s) and documentation change PR(s) up front, and stop until the user provides them or explicitly confirms none exist. Do not start codebase mapping, history search, or spec drafting before this is settled.
- **Always** stop at each stage gate (1.5, 2.4, 3.5, 4.2) and wait for explicit user approval before advancing. Do not collapse stages, do not advance on implicit cues, and do not create GitHub issues until Stage 4.2 approval is given.
- **Always** research prior art before finalizing sub-issues: inspect Figma carefully via the Figma MCP, search GitHub issues/PRs/discussions, Slack, and Gong calls for prior mentions, and read git history (`git log`, `git blame`) for every codepath you expect to touch.
- **Always** consult the relevant SME(s) (Apple/Windows MDM lead, agent lead, frontend lead, etc.) in Stage 2.3 before finalizing the spec, and preserve their feedback verbatim in an "Expert review notes" section. If consultation is deferred, document why in one line.

### Decomposition
- **Decompose along specialization boundaries first, then keep tightly-coupled work within a specialization in one sub-issue.** Never mix frontend and backend in the same sub-issue (specialization failure). Never split work that can only be done together into separate sub-issues (cohesion failure). https://github.com/fleetdm/fleet/issues/31138 is the reference anti-pattern.
- Every sub-issue must reference specific files, line numbers, and patterns from the codebase. No vague specs: "implement the backend" is not a sub-issue.
- **Always** include a single combined `Documentation and engineering QA` sub-issue (layer `docs/QA`, no `~frontend`/`~backend` label) as the final-gate sub-issue in every spec.
- **Always** prefix sub-issue titles with a common feature short-name and a colon (e.g., `"<feature>: <specific area>"`).
- For stories spanning ≥4 sub-issues, **always** include a "With 2 engineers" multi-engineer plan; produce "With 3 engineers" when the story can absorb a third.

### Estimation
- **Capture the parent story's t-shirt size** from its labels/body. If it has none, do not invent one — flag it as an open question.
- **Do not assign story points.** Fleet does not point sub-tasks, and this skill does not produce point estimates or t-shirt-to-point conversions.

### Spec doc structure
- **Always** include a Mermaid sequence diagram in the Feature design section for any feature spanning multiple systems (UI ↔ API ↔ DB ↔ worker ↔ external service ↔ device). Show every actor and the order of operations, with `note over X,Y` blocks marking phase transitions. Omit only when the flow is genuinely single-surface trivial; document why.
- **Always** extract Figma UI strings verbatim into per-surface tables (Element | Text) — section titles, checkbox/button labels, tooltips, descriptions, flash messages (enable/disable/error), disabled-state tooltips. Do not paraphrase Figma copy.
- **Always** answer every engineering checklist item from the parent story body in an "Engineering section answers" section.
- **Always** include "Open questions" and "Resolved questions" sections. Use "None." rather than omitting an empty section. Resolved questions preserve decisions made during spec review and prevent re-litigation.
- **Always** reconcile in-flight doc PRs in an "API design note" section when more than one PR shapes the API or YAML surface for the feature. Show the PR table, the conflicts, and the resolution.

### Sub-issue body
- The sub-issue's spec-doc presentation includes a metadata header (`Depends on`, `Parallel with`) and a description paragraph; the GitHub-filed body uses the strict three-section template (`Related user story` / `Task` / `Condition of satisfaction`) with HTML comments preserved verbatim. The description paragraph from the spec doc becomes the opening paragraph of the GitHub Task section, above any sub-headings or code blocks.
- **Task sections** must contain: sub-headings for work areas, exact file paths with line numbers (verified with Read/Grep, not guessed), full code blocks in the relevant language, references to existing patterns (`"follow X at file:line"`), and **negative-space "why" rationale** — when a design choice diverges from an apparent alternative, document why the alternative was not chosen.
- **Condition of Satisfaction** sections must group assertions by surface or scenario with bolded sub-group labels, include specific test commands with env vars, cover negative cases, and include end-to-end integration when applicable.

### Style
- **Always** apply Fleet's writing style from https://fleetdm.com/handbook/company/writing and https://fleetdm.com/handbook/marketing/fleet-ai-writing-instructions to all prose in the spec.
- If you find ambiguity in the story, flag it as an open question rather than guessing.
- Consider Fleet's multi-platform nature: does this affect macOS, Windows, Linux, iOS, Android?
- Consider enterprise vs core: does this need license checks?

### GitHub
- **Always** apply `#g-software` and `~sub-task` labels and the `Task` type when creating sub-issues in GitHub. Add `~frontend` or `~backend` based on the surface the sub-issue touches — never both. The `docs/QA` sub-issue gets neither `~frontend` nor `~backend`. Confirm with the user via a dry-run summary before running any `gh issue create` commands.
- **Never set a milestone on sub-issues.** Do not pass `--milestone`, and do not add them to a milestone afterward — the milestone belongs to the parent story only.
- **Always wire sub-issues as native tasks off the parent story** via the sub-issues GraphQL API (`addSubIssue` with the `GraphQL-Features: sub_issues` header and the child's node ID), then verify with `subIssuesSummary`. A markdown reference is not enough — they must appear under the parent's Sub-issues list.
- **Always update the parent story's Engineering section** in Stage 4.4: write the Stage 3.3 engineering answers into the parent body, tick the checkboxes, resolve the Risk assessment, and add the sub-issue links. Preserve all HTML comments and the test plan verbatim.
