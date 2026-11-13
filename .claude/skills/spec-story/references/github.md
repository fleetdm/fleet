# GitHub sub-issues


## Body

Read `.github/ISSUE_TEMPLATE/sub-task.md` and fill in its sections. Don't write the template from memory. Reading it keeps the body in step when the template changes. Keep its HTML comments exactly as they are.

The body says WHAT ships and how to tell it's done. `fleetdm/fleet` is public, and product, design, and QA read these issues too. So leave out file paths, line numbers, code or SQL blocks, function names, and why you didn't pick another option. All of that lives in the spec doc. Leave out customer names, Slack links, and Gong links too. Table names, endpoint paths, YAML keys, and JSON field names are fine, because they're the contract. Aim for 250 to 450 words, and follow the style rules in SKILL.md 3.1.

```markdown
## Related user story

#<parent>
<template comment>

## Task

<One paragraph on what ships: how it works today, and how it'll work after.>

What changes:

- <What's different once it ships>

Out of scope: <what this sub-issue leaves alone on purpose>

### Open questions

- **<Short bold lead.>** <One or two sentences.> <Who confirms.>

<One line saying where the implementation notes live, if the user wants it.>

<template comment>

## Condition of satisfaction

- <Something you can test, in plain words>

<template comment>
```

See #53637 for a real example. Write "None." under Open questions when there aren't any. The `Depends on` and `Parallel with` lines only go in the spec doc. Strip file paths from the Engineering answers before they go into the parent story too.


## Labels and type

Every sub-issue gets:

- The parent story's `#g-*` product group label, from Stage 1.2.
- `~sub-task`.
- Type `Task`, set with `--type Task`. It's an issue type, not a label.
- `~backend` for Go work (services, datastore, migrations, API, fleetctl and GitOps, agent) or `~frontend` for React and TypeScript work. Never both.
- No milestone. The milestone belongs to the parent story, and sub-issues follow its schedule.
- No t-shirt size or estimate. Sizes only go on the parent story.

The docs and QA sub-issue gets neither `~backend` nor `~frontend`.


## 4.1 Dry run

Write each body to a temp file with `mktemp`, and print every planned command:

```bash
gh issue create --repo fleetdm/fleet \
  --title "<feature>: <area>" \
  --body-file "$BODY" \
  --label "<parent's #g-* label>" \
  --label "~sub-task" \
  --label "~backend" \
  --type "Task"
```

Add the planned parent story edit from 4.4 to the dry run.


## 4.2 Gate

Wait for the user to approve the dry run. Don't run `gh issue create` before then. If the user already authorized creation up front, like in batch mode, print the dry run and keep going.


## 4.3 Create

Run the approved commands, and note each new issue number.


## 4.4 Link sub-issues and update the parent

Link each issue as a native sub-issue, so it shows up in the parent's Sub-issues list. The REST endpoint needs the child's database ID, not its issue number:

```bash
CHILD_ID=$(gh api repos/fleetdm/fleet/issues/<child> --jq .id)
gh api -X POST repos/fleetdm/fleet/issues/<parent>/sub_issues -F sub_issue_id="$CHILD_ID"
```

Check the links with `gh api repos/fleetdm/fleet/issues/<parent>/sub_issues --jq '.[] | "#\(.number) \(.title)"'`.

Then write the Engineering answers into the parent:

1. Fetch the body with `PARENT=$(mktemp)` and `gh issue view <parent> --json body -q .body > "$PARENT"`.
2. Tick the checkboxes, replace each `TODO` with its answer, and fill in the Risk assessment. Keep every HTML comment and the test plan exactly as they are.
3. Push it back with `gh issue edit <parent> --body-file "$PARENT"`.

Don't add a list of sub-issues to the parent. GitHub already shows native sub-issues under it. A short link inside one answer, like "tracked in #N," is fine.

Report each new issue's number and URL.
