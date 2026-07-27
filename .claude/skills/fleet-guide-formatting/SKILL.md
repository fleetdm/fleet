---
name: fleet-guide-formatting
description: Ensure Fleet how-to guides (articles/ with meta category "guides") follow the concise, step-by-step structure established by Fleet's best guides — short problem statement, prerequisites, inline gotcha callouts, task-based or numbered steps, optional verify/troubleshoot sections, no filler. Use when writing a new guide, converting a draft into a guide, or auditing/retrofitting an existing guide's structure. Trigger on requests like "write a guide for X," "format this as a guide," "check guide formatting," "audit our guides," "does this follow our guide structure," or when editing a file under articles/ tagged category "guides". This skill governs STRUCTURE — what sections exist, in what order, how steps are shown. For voice, grammar, and word choice, use the content-style skill instead; the two are meant to be used together. Do NOT use this for articles, case studies, or announcements — those are different meta categories with their own conventions (articles use the fleet-article-formatting skill). If the piece's meta category is anything other than "guides", this format does not apply. A strong signal this skill applies: the draft reads like an opinion piece, roundup, or narrative with no concrete steps an admin could follow — that's the exact anti-pattern this skill exists to catch.
allowed-tools: Read, Grep, Glob, Edit, Write, Bash(git diff*), Bash(git status*)
effort: medium
---

# Fleet guide formatting

A Fleet guide gets an admin to step 1, step 2, done. It is not a thought piece, a roundup, or an essay that happens to live in `articles/`. This skill exists to do one job: let an admin find the exact step they need without reading past it. Every rule below serves that — this skill captures the structural skeleton established across Fleet's best guides and gives a checklist for writing new guides or auditing existing ones.

This skill is about **structure only**: which sections exist, in what order, how steps are shown. For voice, tone, and grammar mechanics (sentence case, em dashes, filler words, Fleet terminology), use the `content-style` skill and its `references/style-rules.md` — apply both together when writing or reviewing a guide.

## Scope — when this skill applies

This format is for Fleet **guides only** — pieces published under `<meta name="category" value="guides">`: step-by-step procedures an admin follows to accomplish one task.

It does **not** apply to:

- **Articles** (`category` = `articles`) — thought-leadership, how-to essays, and comparison pieces. Use the `fleet-article-formatting` skill instead.
- **Case studies** (`category` = `success stories`)
- **Announcements** (`category` = `announcements`)

Before applying this format, check the piece's `<meta name="category" ...>` value (or ask the author which category it's destined for). If it isn't `guides`, stop and don't impose this structure — flag the mismatch instead (see the mistagged-piece check in the audit checklist below).

## Canonical examples

These are the reference guides this skill is derived from. Read one or two before writing a new guide if you want the pattern in context:

- `articles/deploy-fleet-on-docker-compose.md` — task-headed sections in doing-order, "Optional:" labeled steps, a Troubleshooting section with bold symptom lead-ins.
- `articles/migrate-fleet-server.md` — "Before you begin" prerequisites with inline risk callouts, sequential H2 steps, a "Verify the migration" section, Troubleshooting at the end.
- `articles/enforce-macos-updates-per-major-version.md` — explicit "Step 1 / Step 2 / Step 3" H2 headings because the count itself matters, inline `>` Note/Warning callouts placed exactly where they bite, a numbered UI click-path nested inside a step.
- `articles/set-device-hostname-via-fleet-api.md` — tight prerequisites, numbered click-path-style steps for an API workflow, bold endpoint/header labels instead of prose.
- `articles/manage-bootstrap-package-with-gitops.md` — the shortest possible version of the skeleton: intro, prerequisites, three action-headed steps, a "More information" link, done.
- `articles/autopkg-with-fleet.md` — branching steps (direct mode vs. GitOps mode) handled as sibling H2 sections, each self-contained; a "Get help" section instead of "Further reading" because the tool is community-maintained.
- `articles/canary-fleet-for-fleetd-updates.md` — leads with the *problem* before the fix, a `>` callout for a licensing gotcha, numbered steps under one H2 "Set up your canary fleet" rather than one H2 per step.
- `articles/managed-migration-assistant-mac-to-mac-migration-with-fleet.md` — "Requirements" then "What transfers and what doesn't" (a reference table-in-prose the reader needs before touching config) before any steps; branches for GitOps vs. UI paths; "Further reading" at the end.

**Watch for the mistagged case:** a piece tagged `category: guides` with no prerequisites, no numbered or task-headed steps, and a closing "recap" or "priorities" list instead of stopping after the last practical action. That's an article that got the guides tag, not a guide. Use this as the litmus test in the audit checklist below — see `references/canonical-examples.md` for a full breakdown of the pattern.

## The skeleton

1. **H1 title** — sentence case, task-verb-led: "Deploy Fleet with Docker Compose," "Migrate Fleet server to a new deployment," "Manage bootstrap packages with GitOps." When introducing a named Apple/Fleet feature, "Feature name: task" also works: "Managed Migration Assistant: Mac-to-Mac migration with Fleet."
2. **Opening — no heading** — one short paragraph (rarely two). States the problem and what the reader ends up with. No history lesson, no "in today's landscape." State scope limits up front if the guide doesn't cover every scenario.
3. **Prerequisites** — heading is "Prerequisites," "Requirements," "What you'll need," or "Before you begin." A bulleted list of concrete, checkable requirements (versions, access level, artifacts in hand). Version-dependent requirements go inline in the bullet, not a separate paragraph.
4. **Gotcha callouts, threaded inline** — `> **Note:**` or `> **Warning:**` blockquotes placed right next to the step or requirement they affect. Never a standalone "Gotchas" section collecting them all at the top.
5. **Steps** — pick the shape that fits the task, don't force one pattern:
   - Sequential H2 sections named as actions, in doing-order, each with H3 sub-steps if needed.
   - Explicit "Step 1: ...", "Step 2: ..." H2 headings when the count of steps itself matters.
   - A numbered click-path list inside one section, when the action is "go click through these screens" — bold the UI element names.
   Every step: imperative mood, active voice, one action per step or paragraph.
6. **Verify** (when success isn't obviously visible) — short section confirming the change took effect, often itself a numbered click-path.
7. **Troubleshooting** (when failure modes are known) — heading "Troubleshoot" or "Troubleshooting." Each item leads with a **bold symptom** acting as a pseudo-heading, followed immediately by the fix.
8. **Further reading / Related resources / Get help** (optional) — a short link list at the very end, before the endmatter.
9. **Endmatter** — required. Matches the template in `content-style/references/content-types.md`, `category` set to `guides`, `articleTitle` matching the H1 exactly.

What guides never have: a "Conclusion," "Summary," or "Wrapping up" section that restates what was just said. The guide ends after the last practical section.

## Writing a new guide

1. Confirm it's actually a guide: is there a concrete task with real prerequisites and steps? If the content is analysis, opinion, or a roundup with no procedure, it belongs in `category: articles`, not `guides` — say so rather than forcing the skeleton onto it.
2. Copy `references/template.md` as a starting skeleton and fill it in section by section.
3. Write the opening last if it helps — it's easier to state the problem precisely once the steps are settled.
4. Run the `content-style` skill's checklist over the prose (voice, sentence case, no em dashes, no filler, Fleet terminology) before finishing.
5. Self-check against the audit checklist below.

## Auditing or retrofitting an existing guide

Read the file, then check each item. Report findings by section, don't just say "needs work":

- [ ] H1 is sentence case and task-verb-led (or "Feature name: task").
- [ ] Opening is one short paragraph (two at most), states problem + outcome, no throat-clearing intro.
- [ ] Has a prerequisites/requirements section if the task depends on a version, access level, or artifact.
- [ ] Gotchas are `>` callouts placed next to the step they affect, not buried in a paragraph or dumped in their own section.
- [ ] Steps are numbered or task-headed, not narrated as flowing prose the reader has to parse for actions.
- [ ] Each step is imperative mood, one action.
- [ ] Bold is used only for UI elements, field/file names, and troubleshooting symptom lead-ins. Never decorative.
- [ ] Has a Verify section if success or failure isn't obvious from the last step.
- [ ] Troubleshooting entries (if present) lead with a bold symptom, not a generic "Issue:" label.
- [ ] Ends after the last practical section. No summary/conclusion coda.
- [ ] Endmatter present and correct: category is `guides`, `articleTitle` matches the H1 exactly.
- [ ] **If it has no prerequisites and no concrete steps**, it's not a guide. Recommend recategorizing to `articles` or restructuring around an actual procedure — don't just reshuffle headings on a piece that has no steps to number.

For a deeper structural breakdown of each canonical example and the mistagged-article anti-pattern, see `references/canonical-examples.md`.
