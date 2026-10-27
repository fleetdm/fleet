---
name: handbook-pr
description: Make a change to Fleet's handbook and open a pull request for it. Use whenever someone asks to add, update, remove, reword, or fix anything under handbook/ (Markdown pages, *.rituals.yml, open-positions.yml), including requests that arrive from a Slack thread or a GitHub mention, even if they never say "handbook" or "PR" (for example "document that IT owns laptop refreshes" or "add a note about time off for support roles"). Covers reading Fleet's writing guidelines, finding the right page, rules that break the fleetdm.com build or existing links, keeping private details off a public page, the PR title and description, and requesting the one right reviewer. Not for docs/, articles/, or website/ changes.
allowed-tools: Read, Grep, Glob, Edit, Write, Bash(git *), Bash(gh pr *), Bash(gh api *), Bash(bash .claude/skills/handbook-pr/scripts/*)
effort: medium
---

# Handbook pull requests

Fleet is a [handbook-first](https://fleetdm.com/handbook/company/why-this-way#why-handbook-first-strategy) company. The handbook is public at [fleetdm.com/handbook](https://fleetdm.com/handbook), and anything merged to `main` goes live right away. Your goal is a small, correct PR that the page maintainer can merge without edits.

## 1. Read the style guidelines

Fleet's style guidelines are the source of truth for every word you write. Read them from the repo at the start of each handbook change. They change often, so don't rely on memory or on this skill:

- [Writing at Fleet](https://fleetdm.com/handbook/company/writing) (`handbook/company/writing.md`). Read "Writing style," "Writing mechanics," and the "Editing and publishing" checklist.
- [Brand](https://fleetdm.com/handbook/company/brand) (`handbook/company/brand.md`) for naming and voice.
- The `content-style` skill. Load it and use its checklist on all new or reworded prose.

Link to these pages. Don't copy their rules into the handbook page or the PR description.

## 2. Understand the request

- Pin down exactly what should change and why. If the request came from Slack, read the whole thread, not just the message that tagged you.
- If you're missing a fact (a URL, a GitHub username, a cadence, a date, a tool name, who owns something), ask the requester. Never invent one. An open question is better than a confident guess on a public page.
- Keep one logical change per PR. If the request touches unrelated topics or pages with different maintainers, split it into separate PRs or ask.
- If the request follows up on a merged PR, start a new branch from `main`. If it changes an open PR, push to that PR's branch and update its description instead of opening a second PR.

## 3. Find the right place

- Search before you write: `git grep -n -i "<keyword>" -- handbook/`. If the topic is already covered, update that section instead of adding a new one.
- Link instead of duplicating. If a concept lives on another page, link to that section.
- Rituals belong in the division's `*.rituals.yml` file, which renders as the rituals table on that page. Change the YAML entry and don't repeat the ritual as prose.
- Most changes belong on an existing page. Only create a new page if the requester confirms it. Then follow ["Adding a new handbook page"](https://fleetdm.com/handbook/company/writing#adding-a-new-handbook-page): use the kebab-case, lowercase page title as the filename, add an H1 that matches it, and copy the meta tags from the parent page.

## 4. Make the edit

Match the page around you: heading levels, list style, bold lead-ins, and link format. Then follow the rules below.

**Build and CI rules.** Breaking these fails `website/scripts/build-static-content.js` or `.github/workflows/docs.yml`:

- Every handbook Markdown page needs `<meta name="maintainedBy" value="<github-username>">`. A `title` meta tag must be 40 characters or fewer, and emoji count as 2.
- No `@fleetdm.com` email addresses. Link to a Slack channel or handbook section, or use `@example.com` in examples.
- No `{{ }}` outside inline code, no code block nested inside another code block, and a blank line before any HTML comment that follows a list.
- No "here" or "click here" as link text. Link descriptive words.
- Images need absolute URLs, like `https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/<file>.png`.
- Every ritual needs `task`, `startedOn` (YYYY-MM-DD), `frequency`, `description`, and `dri`. If it has `autoIssue`, `frequency` must be `Daily`, `Weekly`, `Triweekly`, `Monthly`, `Quarterly`, or `Annually`. `dri` must be a real GitHub username: copy it from an existing entry or `website/config/custom.js`, and check new ones with `gh api users/<username>`.

**Link rules** from writing.md:

- Don't rename or remove headings unless the request requires it. fleetdm.com builds each anchor from the heading text, and Slack messages, GitHub issues, and other pages link to those anchors. If you must rename one, update every link to the old anchor in the same PR and call out the rename in the PR description.
- Use full `https://fleetdm.com/handbook/...` URLs, even for sections on the same page. Don't use relative links like `(#section)` or `(../page.md)`.

**Keep the diff minimal:**

- Change only what the request needs. Don't reflow paragraphs, fix unrelated wording, or reformat lists you didn't touch.
- Write each paragraph as one line, with a blank line between paragraphs. Don't put each sentence on its own line.
- Keep existing whitespace and line endings.

## 5. Check that it's safe to publish

The handbook is public, and requests often come from DMs or private channels. Before you commit, remove anything that shouldn't be on fleetdm.com:

- Names of customers or prospects that aren't already public, deal or pricing details, and anyone's compensation, performance, or HR details
- Security details, like internal hostnames, credentials, unpatched vulnerabilities, or detection gaps
- Private Slack messages quoted word for word, and links to private docs that the handbook doesn't already link to

If you're not sure something is public, leave it out and ask in the thread.

## 6. Check your work

1. From the repo root, run `bash .claude/skills/handbook-pr/scripts/check-handbook-diff.sh`. It compares your changes with `origin/main` (run `git fetch origin main` first if needed) and reports errors that break the build or CI, and warnings for broken anchors, relative links, em dashes, and whitespace churn. Fix every error. Fix each warning, or explain it in the PR description.
2. Read `git diff origin/main` line by line, as writing.md asks. Every changed line should be intentional.
3. Run the `content-style` review pass over the new or changed prose.

## 7. Open the PR

**Title:** `Handbook: <what changed>`, in sentence case and under about 70 characters. For example: `Handbook: request Figma access through Vanta`.

**Description:** Don't use `.github/pull_request_template.md` for a handbook-only PR. It's a checklist for code changes, and the `check-pr-template` CI check doesn't run on `handbook/` changes. Use this instead, and delete lines that don't apply:

```markdown
_Requested by <name> · [Slack thread](<link>)_

**Related issue:** #<number>

## What changed

<One or two sentences: what changed and why.>

**Before:**
> <old text, if wording changed>

**After:**
> <new text>

**Page:** https://fleetdm.com/handbook/<path>#<section>

## Notes for reviewer

<Only if needed: renamed headings and the links you updated, facts you couldn't verify, follow-ups.>

## AI

**AI:** <tool> (<model ID>)
```

- For the AI line, use the tool you're running in (for example `Claude Code`) and the exact model ID your environment reports, or `unknown`. Keep the format exactly as shown so people can search for it.
- Don't add a "Generated with" footer if your environment already adds one. One is enough.
- Write the body to a file and pass it with `--body-file`, so HTML comments and backticks come through as written.

**Reviewer:** Request exactly one reviewer, the page maintainer. That's the person whose face appears on the page at fleetdm.com:

1. Use the `maintainedBy` meta tag at the bottom of the page you changed.
2. For YAML files, or pages without the tag, use the closest matching path in `githubRepoDRIByPath` in `website/config/custom.js`.

If you're opening the PR from the maintainer's own GitHub account, you can't request yourself. Leave the reviewer off and say so when you report back.

```bash
gh pr create --title "Handbook: <what changed>" --body-file <path-to-body.md> --reviewer <maintainer>
```

Don't merge the PR yourself.

## 8. Report back

Reply where the request came from, in the Slack thread or the GitHub comment. Include:

- The PR link
- One sentence on what changed
- Who will review it
- Anything you left out or couldn't verify, and why
