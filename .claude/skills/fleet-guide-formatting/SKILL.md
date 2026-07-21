---
name: fleet-guide-formatting
description: Ensure Fleet how-to guides (articles/ with meta category "guides") follow the concise, step-by-step structure established by Fleet's best guides: short problem statement, prerequisites, inline gotcha callouts, task-based or numbered steps, optional verify/troubleshoot sections, no filler. Use when writing a new guide, converting a draft into a guide, or auditing/retrofitting an existing guide's structure. Trigger on requests like "write a guide for X", "format this as a guide", "check guide formatting", "audit our guides", "does this follow our guide structure", or when editing a file under articles/ tagged category "guides". This skill governs STRUCTURE: what sections exist, in what order, how steps are shown. It does NOT replace the content-style skill. Always run content-style over the prose as part of using this skill, in the same session, before calling a guide done. Do NOT use this for articles, case studies, or announcements, which are different meta categories with their own conventions (articles use the fleet-article-formatting skill). If the piece's meta category is anything other than "guides", this format does not apply. A strong signal this skill applies: the draft reads like an opinion piece, roundup, or narrative with no concrete steps an admin could follow. That's the exact anti-pattern this skill exists to catch.
allowed-tools: Read, Grep, Glob, Edit, Write, Bash(git diff*), Bash(git status*)
effort: medium
---

# Fleet guide formatting

A Fleet guide gets an admin to step 1, step 2, done. It is not a thought piece, a roundup, or an essay that happens to live in `articles/`. This skill exists to do one job: let an admin find the exact step they need without reading past it. Every rule below serves that. This skill captures the structural skeleton established across Fleet's best guides and gives a checklist for writing new guides or auditing existing ones.

This skill is about **structure only**: which sections exist, in what order, how steps are shown. Voice, tone, and grammar mechanics live in the `content-style` skill.

## Required: run content-style in the same session

Structure alone is not enough to ship a guide. **Every time you use this skill, invoke the `content-style` skill over the guide's prose before you hand the draft back.** Don't treat it as a suggestion the author can pick up later, and don't substitute your own recollection of the rules for loading the skill.

- **Writing a new guide:** load `content-style` before drafting, so the prose is right the first time, then re-run its review pass on the finished draft.
- **Auditing or retrofitting a guide:** run `content-style` over the file as part of the same audit. Report style findings alongside the structural ones.

## Scope: when this skill applies

This format is for Fleet **guides only**, meaning pieces published under `<meta name="category" value="guides">`: step-by-step procedures an admin follows to accomplish one task.

It does **not** apply to:

- **Articles** (`category` = `articles`): thought-leadership, how-to essays, and comparison pieces. Use the `fleet-article-formatting` skill instead.
- **Case studies** (`category` = `success stories`)
- **Announcements** (`category` = `announcements`)

Before applying this format, check the piece's `<meta name="category" ...>` value, or ask the author which category it's destined for. If it isn't `guides`, stop and don't impose this structure. Flag the mismatch instead (see the mistagged-piece check in the audit checklist below).

## Canonical examples

These are the reference guides this skill is derived from. Read one or two before writing a new guide if you want the pattern in context:

- `articles/deploy-fleet-on-docker-compose.md`: task-headed sections in doing-order, "Optional:" labeled steps, a Troubleshooting section with bold symptom lead-ins.
- `articles/migrate-fleet-server.md`: "Before you begin" prerequisites with inline risk callouts, sequential H2 steps, a "Verify the migration" section, Troubleshooting at the end.
- `articles/enforce-macos-updates-per-major-version.md`: explicit "Step 1 / Step 2 / Step 3" H2 headings because the count itself matters, inline `>` Note/Warning callouts placed exactly where they bite, a numbered UI click-path nested inside a step.
- `articles/set-device-hostname-via-fleet-api.md`: tight prerequisites, numbered click-path-style steps for an API workflow, bold endpoint/header labels instead of prose.
- `articles/manage-bootstrap-package-with-gitops.md`: the shortest possible version of the skeleton. Intro, prerequisites, three action-headed steps, a "More information" link, done.
- `articles/autopkg-with-fleet.md`: branching steps (direct mode vs. GitOps mode) handled as sibling H2 sections, each self-contained, and a "Get help" section instead of "Further reading" because the tool is community-maintained.
- `articles/canary-fleet-for-fleetd-updates.md`: leads with the *problem* before the fix, a `>` callout for a licensing gotcha, numbered steps under one H2 "Set up your canary fleet" rather than one H2 per step.
- `articles/managed-migration-assistant-mac-to-mac-migration-with-fleet.md`: "Requirements" then "What transfers and what doesn't" (a reference table-in-prose the reader needs before touching config) before any steps, branches for GitOps vs. UI paths, and "Further reading" at the end.

**Watch for the mistagged case:** a piece tagged `category: guides` with no prerequisites, no numbered or task-headed steps, and a closing "recap" or "priorities" list instead of stopping after the last practical action. That's an article that got the guides tag, not a guide. Use this as the litmus test in the audit checklist below. See `references/canonical-examples.md` for a full breakdown of the pattern.

## The skeleton

1. **H1 title.** Sentence case, task-verb-led: "Deploy Fleet with Docker Compose", "Migrate Fleet server to a new deployment", "Manage bootstrap packages with GitOps". When introducing a named Apple or Fleet feature, "Feature name: task" also works: "Managed Migration Assistant: Mac-to-Mac migration with Fleet".
2. **Opening, no heading.** One short paragraph, rarely two. States the problem and what the reader ends up with. No history lesson, no "in today's landscape". State scope limits up front if the guide doesn't cover every scenario.
3. **Prerequisites.** The heading is "Prerequisites", "Requirements", "What you'll need", or "Before you begin". A bulleted list of concrete, checkable requirements: versions, access level, and artifacts in hand. Version-dependent requirements go inline in the bullet, not in a separate paragraph.
4. **Gotcha callouts, threaded inline.** Use `> **Note:**` or `> **Warning:**` blockquotes placed right next to the step or requirement they affect. Never a standalone "Gotchas" section collecting them all at the top.
5. **Steps.** Pick the shape that fits the task, don't force one pattern:
   - Sequential H2 sections named as actions, in doing-order, each with H3 sub-steps if needed.
   - Explicit "Step 1: ...", "Step 2: ..." H2 headings when the count of steps itself matters.
   - A numbered click-path list inside one section, when the action is "go click through these screens". Bold the UI element names.
   Every step: imperative mood, active voice, one action per step or paragraph.
6. **Verify** (when success isn't obviously visible). A short section confirming the change took effect, often itself a numbered click-path.
7. **Troubleshooting** (when failure modes are known). The heading is "Troubleshoot" or "Troubleshooting". Each item leads with a **bold symptom** acting as a pseudo-heading, followed immediately by the fix.
8. **Further reading / Related resources / Get help** (optional). A short link list at the end, before the endmatter.
9. **Endmatter.** Required, and you write it. See "Endmatter is not optional" below.

What guides never have: a "Conclusion", "Summary", or "Wrapping up" section that restates what was said. The guide ends after the last practical section.

## Endmatter is not optional

Every guide ends with the `<meta>` block from `.claude/skills/content-style/references/content-types.md`. **Emit it yourself as part of the draft.** A guide handed back without endmatter is incomplete, and the author shouldn't have to notice it's missing and paste it in. `references/template.md` ends with the block already filled in for guides. Keep it there.

Fill it in like this:

- `articleTitle`: matches the H1 exactly, character for character.
- `category`: always `guides` for this skill. If it should be anything else, this skill doesn't apply (see Scope).
- `description`: 1-2 sentences, 150 characters max, factual and benefit-driven. Write this one. It's the only field you can derive from the guide itself.
- `authorFullName`, `authorGitHubUsername`, and `publishedOn`: **never fabricate these.** If you don't know the author or the intended publish date, leave the placeholder in place and tell the author which fields they need to fill in.

## Write a new guide

1. Confirm it's a guide: is there a concrete task with real prerequisites and steps? If the content is analysis, opinion, or a roundup with no procedure, it belongs in `category: articles`, not `guides`. Say so rather than forcing the skeleton onto it.
2. Load the `content-style` skill now, before drafting, so the prose is right the first time.
3. Copy `references/template.md` as a starting skeleton and fill it in section by section, endmatter included.
4. Write the opening last if it helps. It's easier to state the problem precisely once the steps are settled.
5. Run the `content-style` review pass over the finished prose. Search for `—` and rewrite every hit.
6. Self-check against the audit checklist below.

## Audit or retrofit an existing guide

Read the file, run the `content-style` skill over it, then check each item. Report findings by section, don't just say "needs work":

- [ ] H1 is sentence case and task-verb-led, or "Feature name: task".
- [ ] Opening is one short paragraph, two at most, states the problem and the outcome, and has no throat-clearing intro.
- [ ] Has a prerequisites or requirements section if the task depends on a version, access level, or artifact.
- [ ] Gotchas are `>` callouts placed next to the step they affect, not buried in a paragraph or dumped in their own section.
- [ ] Steps are numbered or task-headed, not narrated as flowing prose the reader has to parse for actions.
- [ ] Each step is imperative mood, one action.
- [ ] Bold is used only for UI elements, field and file names, and troubleshooting symptom lead-ins. Never decorative.
- [ ] Has a Verify section if success or failure isn't obvious from the last step.
- [ ] Troubleshooting entries, if present, lead with a bold symptom, not a generic "Issue:" label.
- [ ] Ends after the last practical section. No summary or conclusion coda.
- [ ] Endmatter present and complete: all six `<meta>` tags, `category` is `guides`, `articleTitle` matches the H1 exactly, and `description` is under 150 characters. Author and date are real, or flagged as needing the author's input. Never invented.
- [ ] `content-style` was run over the prose in this session, and its findings are reported alongside the structural ones.
- [ ] **If it has no prerequisites and no concrete steps**, it's not a guide. Recommend recategorizing to `articles` or restructuring around a real procedure. Don't just reshuffle headings on a piece that has no steps to number.

For a deeper structural breakdown of each canonical example and the mistagged-article anti-pattern, see `references/canonical-examples.md`.
