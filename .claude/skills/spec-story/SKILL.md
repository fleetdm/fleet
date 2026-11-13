---
name: spec-story
description: Break down a Fleet GitHub story into sub-issues with a technical spec, then create the sub-issues, link them to the parent story, and fill in its Engineering section. Use whenever someone asks to "spec", "break down", "decompose", "plan the engineering work for", or "write sub-issues for" a fleetdm/fleet story, or hands over a story and asks how to split it up, even if they never say "spec". Not for bug triage, bug fixes, PR review, or OpenSpec change proposals.
allowed-tools: Bash(gh *), Bash(git log*), Bash(git blame*), Bash(git show*), Bash(git diff*), Bash(mktemp*), Read, Grep, Glob, Write, Edit, Agent, Skill, WebFetch(domain:github.com), WebFetch(domain:fleetdm.com), WebSearch, mcp__claude_ai_Figma__*, mcp__claude_ai_Slack__*, mcp__claude_ai_Gong__*
model: opus
effort: high
argument-hint: "<issue-number-or-url>"
---

# Spec a Fleet story

Break down this GitHub story into sub-issues: $ARGUMENTS


## Outputs

- **The spec doc** is for the engineers who build the story. It holds the HOW: checked `file:line` anchors, code shapes, migration plans, sequence diagrams, and the reason behind each decision. Save it where the user says, outside the repo. It's working notes, so it doesn't ship.
- **The GitHub sub-issues** are for everyone who tracks the story, including product, design, and QA. They hold the WHAT: what ships, open questions, and how to tell it's done. `fleetdm/fleet` is public and code anchors go stale. So keep file paths, line numbers, code, internal links, and customer names out of them.

Read [references/spec-doc.md](references/spec-doc.md) before you draft the spec doc. Read [references/github.md](references/github.md) before you write sub-issue bodies and before Stage 4.


## Stages and gates

```
Stage 1: Understand  →  [gate]  →  Stage 2: Skeleton  →  [gate]  →
Stage 3: Draft  →  [gate]  →  Stage 4: Create in GitHub (dry run → gate → create)
```

Stop at each gate. Sum up what you made and wait for the user to approve it. If they push back, stay in that stage and keep working. Each stage builds on the last. A wrong read of the story leads to a wrong split, and a wrong split wastes a full draft.

**Batch mode.** The user can waive the gates up front, which is common for a batch of S-and-below stories specced in parallel. Then run Stages 1 through 3 without stopping. Still do the design review, both checks, and the Stage 4 dry run. Note any skipped SME review in Expert review notes. Creating issues in GitHub still needs a clear go-ahead, which the user can give as part of the waiver. Without a waiver, every gate applies.


## Stage 1: Understand the story

Build a shared picture of what the story asks for, which parts of Fleet it touches, and what context lives outside the issue.


### 1.1 Gather required inputs

Read the issue body first (`gh issue view <n> --json body`). Stories often link their Figma file and doc PRs. Then make sure you have these:

- **A Figma link** to the `✅ Ready` page for every design surface, or word that there's no design. Backend-only stories often have none.
- **The doc change PRs** for the REST API, GitOps YAML, audit log, and usage stats, or word that there are none.
- **Where to save the spec doc**, unless the user already said.

If anything's still missing and not waived, stop and ask for each missing input. Don't guess, and don't start mapping the code. Doc PRs set the field names the code must match. Figma dev notes often change how the code works. Speccing without them leads to rework.

**Figma.** If the Figma MCP server isn't connected, tell the user. With their go-ahead, keep going without it. Mark the spec's Figma section as not extracted, and list the missing copy and dev notes as open questions.

When it's connected, call `create_session` with the file key first, and reuse its session ID on later calls. A story's Figma link usually opens the `ℹ️ Cover` page (often `node-id=0-1`), which only links back to the issue. The screens and dev notes live on the Ready page. A `get_metadata` page list with no node ID often shows only the cover, even when a Ready page exists. So a cover-only result doesn't mean there's no design. Ask the user for the Ready page's node URL (Figma's "Copy link to selection"), then:

1. Call `get_metadata` on that node to get its structure.
2. Call `get_screenshot` on the whole Ready section.
3. Call `get_design_context` on each `Dev note` callout, tooltip, and label to copy the text word for word. If a response comes back flagged as sparse, request its visible child nodes in one parallel batch.

You're only pulling copy and notes, so skip Figma's design-to-code workflow. Capture every state: empty, error, loading, disabled, and GitOps mode.

**Doc PRs.** Run `gh pr view <n> --json title,body,files` and `gh pr diff <n>`.


### 1.2 Read the issue

- Run `gh issue view <n> --json title,body,labels,milestone,assignees`. Read the description, acceptance criteria, test plan, and linked issues.
- Note the parent's `#g-*` product group label. Every sub-issue gets the same one. If the parent has none, or more than one, ask which to use.
- Look up the t-shirt size. It's the `T-shirt size` field on the "🗺️ Release planning" project, not a label:
  ```bash
  gh api graphql -F n=<parent> -f query='query($n:Int!){repository(owner:"fleetdm",name:"fleet"){issue(number:$n){projectItems(first:20){nodes{project{title} fieldValues(first:50){nodes{... on ProjectV2ItemFieldSingleSelectValue{field{... on ProjectV2SingleSelectField{name}} name}}}}}}}}' \
    --jq '[.data.repository.issue.projectItems.nodes[] | select(.project.title | test("Release planning")) | .fieldValues.nodes[] | select(.field?.name=="T-shirt size") | .name] | first'
  ```
  If it's `null`, list it as an open question. Don't make up a size. Sizes only go on the story, so don't size sub-issues or add story points to them.


### 1.3 Map the code

Start with the schema. The shape of the database decides sub-issue lines, migration scope, and conditions of satisfaction. Reading it is faster and more accurate than piecing it together from Go code.

- **Schema.** Grep each table you'll touch in `server/datastore/mysql/schema.sql`. It's a generated dump of the current schema, rebuilt by `make dump-test-schema`. For example, run ``grep -A40 'CREATE TABLE `software_installers`' server/datastore/mysql/schema.sql``. Don't load the whole file.
- **Migrations.** `server/datastore/mysql/migrations/tables/` starts with the Fleet 5 baseline (`20261103120000_Fleet5Baseline.go`, which loads `baseline_schema.sql`). After that, it only holds migrations added since. Skim the newest ones for changes to the same tables. To learn why a column added before Fleet 5 exists, find its first migration in git history with `git log --all -S '<column>' -- server/datastore/mysql/migrations/tables/`.
- **Everything else.** Find related code, the datastore methods in `server/fleet/datastore.go`, service methods, endpoints, and frontend pages. Trace the flow from endpoint to service to datastore to frontend.


### 1.4 Research prior work

Split the research across parallel read-only subagents, one per source group below. Ask each one for findings with links, not raw search output. Useful findings include decisions, limits, rejected ideas, field names, and SMEs. This keeps your own context free for putting the story together. Wait for every subagent to report before you write the Stage 1 summary. If you're running as a subagent yourself, run them in the foreground, because ending your turn while they run ends your task. If an MCP server isn't connected, skip that source and say so in your summary.

- **GitHub.** Run `gh issue list --search "<keywords>" --state all --limit 20` and `gh pr list --search "<keywords>" --state all --limit 20`. Closed and merged attempts tell you as much as open ones. Run `gh search issues "<keywords>" --repo fleetdm/fleet` too. Find every in-flight doc PR, and note the fields each adds and where it puts them. Doc PRs often conflict, and Stage 3 has to settle each conflict.
- **Slack and Gong.** Search Slack with `slack_search_public_and_private`, or `slack_search_public` if that's all you have. Read promising threads with `slack_read_thread`. Gong only answers questions about one CRM account or deal, and it has no keyword search. When the story names a customer, ask `ask_account` (with the full CRM account name) what that customer needs from the feature. Set a date range that covers the request. If there's no named customer, skip Gong.
- **Git history.** For each file you expect to touch, run `git log --oneline -- <path>`. Use `git log -p` when a refactor is likely, and `git blame` on tricky code. Commit messages and linked PRs often explain conventions and reversed decisions.
- **SMEs.** Use the affected areas and PR authors to name the engineers who know each system best. Think Apple MDM, Windows MDM, Android, fleetd, or frontend. You'll check with them in Stage 2.


### 1.5 Stage 1 gate

Show the user a summary that pulls the inputs together in your own words:

- **Story.** Quote the user story from the parent issue.
- **Summary.** In 2 to 4 short paragraphs, explain what the feature does from start to finish. Name the command, surface, platform limits, and license tier.
- **Scope calls.** Write each as a bold callout with its reason, for example **Premium only.** `UpdateMDMAppleSetup` returns `ErrMissingLicense` in core.
- **Affected areas.** List the UI pages, endpoints, services, datastore methods, migrations, MDM commands, CLI and GitOps, and agent changes.
- **Sequence diagram.** Draw a Mermaid sequence diagram for any feature that spans systems. For a single-surface story, say why you left it out.
- **Label and size.** Give the product group label and the story's t-shirt size.
- **Prior work.** List related decisions and work, including in-flight doc PRs and known field-name conflicts.
- **SMEs.** Name each one and their area.
- **Open questions.** List anything that blocks the split.


## Stage 2: Agree on the skeleton

Get the user to approve a sub-issue list and dependency graph. Don't write full bodies yet.


### 2.1 Split the work

Balance cohesion and specialization:

- **Cohesion.** Keep work together when it can only be built and tested together. A service method goes with the datastore call it needs. An endpoint goes with its request and response types. A migration goes with the queries that read its new column. Splitting these only looks parallel. If sub-issue B's PR would say "needs A merged first to compile or test," merge them.
- **Specialization.** Never put backend (Go, MySQL, API) and frontend (React, TypeScript) work in one sub-issue. They have different owners and reviewers, and the PRs ship on their own. If a reviewer would need both skills to approve it, split it.

Split by skill first: backend, frontend, fleetctl and GitOps, and agent. Then keep tightly linked work inside each one together. A migration goes with the backend sub-issue that owns it. A typical full-stack story has a backend foundation sub-issue and a backend integration sub-issue. It adds one frontend sub-issue per independent surface, plus the docs and QA sub-issue. A backend sub-issue ships a tested API that frontend can mock.

#31138 shows what to avoid, because it splits linked work and mixes frontend with backend.

Every spec ends with one **Documentation and engineering QA** sub-issue in the `docs/QA` layer. It covers the REST API docs (`docs/API/rest-api.md`), audit log docs (`docs/API/global-audit-logs.md`), usage stats (`articles/fleet-usage-statistics.md`), and feature guides. It also covers QA on a real device once every other PR has merged. It depends on every other sub-issue. It gets no `~frontend` or `~backend` label, since it checks the whole story. Small stories get one too.


### 2.2 Draw the dependency graph

Draw it in ASCII. Parallel tracks like frontend surfaces and GitOps usually open up once the backend integration sub-issue settles the API contract. They can start from that contract while it's still in review.


### 2.3 Review the design

Aim for the smallest design that fully solves the story, with the fewest new tables, endpoints, and moving parts. First drafts usually split too finely and reach too far. Before you show the skeleton, review it in five areas:

- **Data model.** Reuse tables and columns before you add new ones. Pick the narrowest type, and set nullability and defaults on purpose. Add the indexes your queries need and no others. Don't store what you can derive, and don't hide fields you'll filter on inside JSON. Add foreign keys unless the table's big enough that one would force a full table copy.
- **Security.** Name who can call each new endpoint and see each new field. That means the service-layer auth check, fleet scoping, role permissions, and API-only users. Treat input from devices and users as untrusted. Keep secrets out of logs, activities, and API responses. For changes to auth, MDM, enrollment, or user data, run the `fleet-security-auditor` agent on the plan.
- **Scalability.** Plan for deployments with hundreds of thousands of hosts. Avoid per-host loops and N+1 queries, and make sure every new query uses an index. Know what runs on each host check-in versus in a cron job. If the load profile changes, say so in the Engineering answers for load testing.
- **Simplicity.** Ask whether each sub-issue and schema change needs to exist. Look for a table, column, helper, endpoint, or component to reuse. See if a new table could be a column, or a new endpoint a field on an existing one. Merge sub-issues when it doesn't cost parallel work.
- **Scope.** Spec only what the story asks for. Trace every sub-issue and condition of satisfaction back to the story's goal or acceptance criteria. Cut anything that doesn't trace back, or list it under Follow-ups in the spec doc. That covers nearby bugs, refactors, and nice-to-haves.

Write up the results in a **Design review** section. Cover what you cut, reused, simplified, or kept in each area, and why. On #47904, the simplicity review took the first draft from 8 sub-issues and a five-table migration down to 6 sub-issues. The migration became one ALTER plus three DROPs, and the plan kept all its parallel work.


### 2.4 Check the skeleton's claims

Start one fresh read-only subagent (general-purpose or `Explore`). Give it a numbered list of every claim the split depends on. That includes `file:line` anchors, symbols, tables, columns, GitHub links, and how the code behaves today. Add a few open questions too. For example, ask what else reads `<table>`, or whether the GitOps delete path handles this. Ask for a CONFIRMED, REFUTED, or PARTIAL table with evidence.

A fresh agent doesn't share your assumptions, so it catches mistakes a reread misses. The open questions often turn up facts that change the plan. Put the results in a **Verification** section, and fix every refuted claim where it appears.


### 2.5 Check with the SMEs

Ask the user to share the summary and skeleton with the SMEs from 1.4. SMEs often change the split. They spot hidden links between pieces, ownership lines, and what's safe to put off. When feedback comes back, write each point as a numbered note with the SME's name in parentheses, and work it into the plan. Keep the notes word for word in the spec's **Expert review notes**, even after you've used them, so nobody reopens settled decisions later. If the user skips SME review, write down why in one line.


### 2.6 Stage 2 gate

Show the user:

- **Sub-issue list.** Give each one's title (`<feature>: <area>`), layer (`backend`, `frontend`, `backend/CLI`, or `docs/QA`), labels, what it depends on, and what it can run alongside.
- **Dependency graph.** Include the graph from 2.2.
- **Checks.** Include the Design review and Verification results.
- **Team plan.** For 4 or more sub-issues, show how 2 engineers would split the work, and 3 if the story can use them.
- **Expert review notes.** Include the notes, or why SME review was skipped.
- **Size and questions.** Give the story's t-shirt size and the open questions. Sub-issues don't get a size.

Expect pushback. Rework the split instead of defending your first draft. Don't write full bodies until the user approves the skeleton.


## Stage 3: Draft the spec

Write the full spec doc and a GitHub body for every sub-issue.


### 3.1 Write

1. Read `handbook/company/writing.md` and follow the style rules below.
2. Draft the spec doc using [references/spec-doc.md](references/spec-doc.md).
3. Write each sub-issue's GitHub body using [references/github.md](references/github.md).

Fleet's style is plain and casual. Use it in the spec doc, the GitHub bodies, and the Engineering answers:

- Put the main point first, in each paragraph and each sentence.
- Use short sentences, short words, active voice, and contractions.
- State each point directly. Don't announce it first, and don't open with a phrase that tells the reader a point is notable or important.
- Say what a part does instead of labeling it critical or structural.
- Use commas or periods instead of em dashes. Use colons sparingly, and don't glue one to a label.
- Skip filler adverbs and marketing adjectives.
- Use bold sparingly.

As you draft, cover these areas:

- **Migrations.** Any sub-issue with a migration follows `.claude/rules/fleet-database.md` and `docs/Contributing/guides/migrations.md`. Its Task in the spec doc covers guards that make it safe to retry. It also covers staying independent of migrations on other branches, since they can run out of order. Seed or default data goes in a new table migration, because the baseline is never edited. Run the big-table checks on any table that grows with host count. If the migration changes or adds data, its condition of satisfaction includes an `applyUpToPrev`/`applyNext` test. Migrations that only add a table, column, or index don't need one. If it touches a big table, add a timing note on a large dataset.
- **Apple MDM commands.** Follow "Adding a new command or a new reference" in `docs/Contributing/mdm/apple/apple-mdm-command-lifecycle.md`. It covers new commands Fleet sends on its own, and new columns that store a command UUID Fleet reads later. Write down the cleanup choice (short, standard, or unlisted) and whether the inactive-purge denylist applies. Add any reference guard and index the feature needs.
- **API and YAML changes.** Run `/api-design-check` on any new endpoint or field. If a sub-issue needs an API or YAML change that the story's doc PRs don't cover, list it in Open questions. Then the tech lead can send it to the product designer.
- **Coverage.** Think through every platform (macOS, Windows, Linux, iOS, iPadOS, Android, and ChromeOS), Free and Premium, and GitOps mode. When something's unclear, add an open question instead of guessing.


### 3.2 Check the draft

Start a fresh read-only subagent with the spec doc and GitHub bodies. Tell it to treat every claim as wrong until it confirms it against the source. It should report only what fails or what it can't confirm. Have it check that:

- Every `file:line` and symbol exists in the current code, and every datastore method exists in `server/fleet/datastore.go`.
- Every table and column exists in `schema.sql` with the stated type and constraints, and every planned migration fits the table as it is today. Constraints include nullability, defaults, unique keys, indexes, and the foreign keys the plan relies on.
- Every issue and PR number resolves, and doc PR field names match the spec.
- The graph has no cycles, no sub-issue mixes backend and frontend, and the docs and QA sub-issue is there.
- Every doc PR or field-name conflict has a clear decision.
- GitHub bodies have no file paths, line numbers, code blocks, function names, internal links, or customer names, and they agree with the spec doc.
- The design holds up on data model, security, scalability, simplicity, and scope, as described in 2.3. Flag any new endpoint without a named auth check, any query that can't use an index, and any work the story didn't ask for.
- The writing follows the style rules in 3.1.
- No affected area is missing a sub-issue. Ask it to name any it finds.

Fix everything it flags. If you can't confirm a claim, list it as an open question instead of stating it as fact.


### 3.3 Stage 3 gate

Show the user the spec doc and the GitHub bodies. Don't create anything in GitHub until they approve.


## Stage 4: Create the issues in GitHub

Follow [references/github.md](references/github.md). Print a dry run of every `gh issue create`, and wait for approval unless the user already authorized creation up front. Then create the issues, and link each one as a native sub-issue of the parent. Write the Engineering answers into the parent story. Report each new issue's number and URL.
