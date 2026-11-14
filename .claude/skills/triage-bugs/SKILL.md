---
name: triage-bugs
description: Triage the bug Inbox on a Fleet product group's GitHub project board the way the Engineering Manager does at standup. Finds Inbox bugs already fixed by merged PRs, asks openers for missing expected behavior or repro steps, moves bugs that need a design call to Drafting for the Product Designer, and moves complete technical bugs to Ready. Use for "/triage-bugs", "triage the board", "clean up the Inbox", "go through the aging bugs", "which Inbox bugs are already fixed", "move the bugs along", or any request to route bugs on a fleetdm `#g-*` board (e.g. https://github.com/orgs/fleetdm/projects/109). Don't use it to investigate or fix a single bug.
argument-hint: "[board URL or project number] [sweep|asks|drafting|ready|all]"
effort: high
---

# Triage bugs

Move a product group's Inbox bugs to their next column, following `handbook/company/product-groups.md` ("How issues move" and "All bugs > Inbox") as this team runs it. Every comment, column move, assignee change, and body edit is drafted in chat first. Nothing posts until the user approves that batch.


## Inputs

- **Board.** The project number from `$ARGUMENTS` or a URL like `https://github.com/orgs/fleetdm/projects/109/views/7`. With none given, ask once.
- **Group and Product Designer.** Find the board number in `handbook/company/product-groups.md`. Each group's section links its board and `#g-*` label, and its team table names the Product Designer (PD). That PD is who Drafting comments mention and who gets assigned there.
- **Phase.** `sweep`, `asks`, `drafting`, `ready`, or `all`. With none given, run `all`: sweep first, then classify everything left. Present the drafts grouped by action so the user can approve one group at a time.

Keep working files (snapshots, issue bodies, drafts) in the session scratchpad.


## Ground rules

Each rule has a reason.

- **Inbox only.** Bugs in Drafting, Ready, or later are already in the flow and close through it. Listing them as fixed or blocked is noise.
- **Skip bugs owned elsewhere.** Leave out `:reproduce` (QA board), `:release` (already scheduled), `:help-*` (another team), `~investigate` (still with CSEs), `security` (private track), and closed issues.
- **No QA reproduction requests by default.** Many groups reproduce bugs themselves, even though the handbook describes `:reproduce`. Don't add the label or write "I'll send this to QA" unless the user says their group sends bugs to QA. Asking the opener to re-check their own report is fine.
- **Assignees change only on a Drafting move**, where the PD gets assigned. Designers work from Drafting items assigned to them, so an unassigned bug there sits untouched. Ready needs no assignee: engineers pick unassigned bugs from Ready. Don't unassign anyone.
- **Milestones only matter for `~unreleased bug`.** Those need the release that ships the feature and the feature's engineer (`handbook/engineering/releases.md`). Flag gaps in the report. Don't set them unless asked.
- **No customer names** in anything posted. Labels like `customer-*` and `#fleet-at-*` channel names count.
- **Re-snapshot before every report and every apply.** The user moves and closes bugs by hand between turns. Mention anything that moved without you ("You moved #N to Ready").


## Board helper

`scripts/board.sh` reads and moves board items over the REST API. Status names match without their emoji.

```bash
S=.claude/skills/triage-bugs/scripts/board.sh   # run from the repo root
$S bugs 109 inbox                 # TSV: number, status, assignees, labels, title
$S bugs 109 | cut -f2 | sort | uniq -c   # bug count per column
$S move 109 drafting 51493 53393  # move one or more issues
```

Avoid `gh project`. It runs on GraphQL, which parallel agents can exhaust in minutes, and then it fails with "unknown owner type". When `gh issue view` fails the same way, read issues with `gh api repos/fleetdm/fleet/issues/N` and `.../comments`.

Read an issue with `gh issue view N -R fleetdm/fleet --json title,author,assignees,labels,milestone,createdAt,body,comments`. Strip `<!-- -->` template comments before judging whether a section is filled in.


## Workflow


### 1. Snapshot

List the Inbox bugs, apply the exclusions, and record the count. That count is the "before" number for the final report. Note which carry `~aging bug` (90+ days old), since those are the priority.


### 2. Fixed-by-PR sweep

Find Inbox bugs that a merged PR already fixed. Read `references/sweep-brief.md` for the method and the subagent brief. In short:

1. Fetch `origin/main`. Pull every PR merged since the oldest Inbox bug was filed. Split the date range so no query returns 1,000, since `gh pr list --search` caps there.
2. Pull each bug's timeline cross-references with GraphQL.
3. Hand about 10 bugs to each read-only subagent. Each checks the code on `origin/main`, then returns FIXED, IN FLIGHT, or NOT ADDRESSED with a confidence level and the PR.

Draft this comment on each FIXED bug with high or medium confidence. The user verifies and closes them.

```markdown
I think this is fixed in #NNNNN. Need to confirm.
```

List IN FLIGHT bugs with their open PR. They may belong in In progress, but leave that to the user.


### 3. Classify what's left

Read each remaining bug's body, comments, and recent timeline (`gh api repos/fleetdm/fleet/issues/N/timeline`), then put it in exactly one bucket. Bugs bounce between groups through label changes with no comment, so the last comment alone can be stale.

| Bucket | When | Action |
|---|---|---|
| Ask the opener | Expected behavior or repro steps are missing or `TODO`, and the opener can supply it | Comment mentioning the opener; stays in Inbox |
| Needs product input | The fix depends on a product choice | Drafting, PD assigned, plain-language question |
| Ready | Expected behavior and repro steps are filled in (or the defect is plain from the code), and the fix needs no product choice | Ready, no comment |
| Hold | An open PR already targets it, fresh discussion in the last day, the bug belongs to another group, the opener is the user, or the user said to skip it | No action; list it with the reason |

Ask for the version only when the bug has neither `~released bug` nor `~unreleased bug`. Those labels already say what the version would, and a blank "Discovered" field alone isn't worth a round trip.

These signs point to a product choice:

- Expected behavior is `TODO` and someone in the thread called it a product call.
- Commenters disagree on what should happen.
- The fix needs new UI states, new copy, or a change to documented behavior.
- The expected behavior contradicts the report.

When several bugs hang on one decision, say so in each comment ("Same call as #X and #Y") and group them in the draft.

Expected behavior can be plain from existing behavior, like an error the product already shows on a sibling path. In that case, propose a one-line `### 🛠️ Expected behavior` section for the issue body. Insert it before `### 🧑‍💻  Steps to reproduce` and show the diff, so the bug can go to Ready without a round trip.

When the opener is the right person to ask but someone already couldn't reproduce the bug, say who tried and why it failed.


### 4. Draft

Show every draft in chat, grouped by action, with one table per group (bug, short title, who's mentioned, column change, assignee change). Then ask for approval. Approval covers only the batch shown. When the user edits or skips an item, apply that change and re-show only what changed.

Every comment is one to three sentences. It's plain language, with no file paths or code for the PD. Hedge inferences with "I think", and name the choices when asking for a decision. Use these templates, which the user has approved:

- **Ask the opener:** `@<opener> could you <ask>? We need it before we can move this out of the Inbox. Thanks!` The asks are "fill in the expected behavior" and "add steps to reproduce." Combine them as needed, and add "and the version you found this on" when the version is missing.
- **Needs product input:** `@<pd> <question naming the options>`. For example: "@<pd> should skipped installs get their own "Skipped" count on the Library card, or drop out of the Failed count? The same call covers #53393 and #55053." When part of the bug is fixed and a list of design calls remains, open with "I think this needs a design call, so I'm moving this to Drafting."
- **Can't reproduce:** `@<person> can you still reproduce this on <version>? If not, I'll close it in favor of #NNNNN.`
- **Ready:** no comment. Add one only when the investigation turned up something the engineer needs, such as a suggested fix that misses a case.


### 5. Apply

After approval:

```bash
gh issue comment N -R fleetdm/fleet --body-file draft_N.md
$S move <project> drafting N...          # or: ready N...
gh issue edit N -R fleetdm/fleet --add-assignee <pd>     # Drafting moves only
gh issue edit N -R fleetdm/fleet --body-file body_N.md   # approved body edits only
```

Write each comment to a file and post it with `--body-file`, so quoting and backticks survive. Then re-snapshot and check each bug's column and assignees on GitHub.


### 6. Report

Lead with the Inbox count before and after, then what moved. For example: "Inbox went from 80 to 6. You closed 6, 19 went to Drafting, and 47 went to Ready."

- Link each comment you posted.
- Show what's left in the Inbox as a table with a "Waiting on" column. Fill it with the person and what they were asked, or "Us" and the check to run.
- Flag `~unreleased bug` items moved to Ready that are missing a milestone or assignee.
- Give the date to close asked bugs with no reply. The handbook allows one week, so that's the ask date plus 7 days.

If the user asks how many bugs they've closed, filter the issue `closed` events by their login (`gh api repos/fleetdm/fleet/issues/N/events`).
