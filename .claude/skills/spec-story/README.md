# /spec-story

`/spec-story` splits a Fleet story into sub-issues that engineers can build in parallel. It writes a technical spec doc for the engineers and a short WHAT-only body for each sub-issue. Once you approve, it creates the sub-issues, links them under the parent story, and fills in the parent's Engineering section.


## Run it

```
/spec-story <issue-number-or-url>
```

It works in four stages: Understand, Skeleton, Draft, and Create. It stops for your approval after each one. Nothing's created in GitHub until you approve the dry run.


## What it does

You don't need to ask for any of these:

- **Research.** It searches GitHub issues and PRs, Slack, git history, and Gong (when the story names a customer). Each source gets its own parallel subagent.
- **Schema first.** It reads `server/datastore/mysql/schema.sql` and the migrations since the Fleet 5 baseline before it digs into code.
- **Figma.** It pulls the `Ready` page by `node-id` and copies dev notes, tooltips, and messages word for word. It skips the cover page, which only links to the issue.
- **Split.** It splits work by skill (backend, frontend, fleetctl and GitOps, agent) and keeps tightly linked work together. It always adds a Documentation and engineering QA sub-issue.
- **Checks.** Before the skeleton gate, it reviews the design for data model, security, scalability, simplicity, and scope. A separate subagent rechecks every `file:line`, schema, and GitHub claim, first on the skeleton and again on the draft.
- **Labels.** Every sub-issue gets the parent story's `#g-*` product group label. It reads the story's t-shirt size from the Release planning project. Sub-issues don't get a size.
- **Two outputs.** The spec doc holds the HOW, like code anchors, code shapes, and reasons. GitHub bodies hold the WHAT, since `fleetdm/fleet` is public and code anchors go stale.
- **Style.** It writes in Fleet's plain, casual style from `handbook/company/writing.md`.


## What to give it

Include these up front so it doesn't have to stop and ask:

- **The story.** Give the issue number or URL. This one's required.
- **The Figma Ready page link** with its `node-id`, like `.../design/...?node-id=2-130&m=dev`. Or say there's no design.
- **The doc change PRs** for the REST API, GitOps, or audit log. They set the field names the spec has to match. Or say there aren't any.
- **Where to save the spec doc.** Pick any place outside the repo.


## Prompt template

```
/spec-story https://github.com/fleetdm/fleet/issues/<N>

Figma (Ready page): <figma design URL with node-id>
Doc change PRs: #<a>, #<b>
Spec doc: <path outside the repo>
```

To spec a batch of small stories without stopping at each gate, say so up front. For example, "batch mode, skip the gates, SME review waived." It still stops before it creates anything in GitHub, unless you tell it to create the issues too.


## Prerequisites

- `gh`, signed in with `gh auth login`.
- Figma, Slack, and Gong need their MCP servers connected. Without them, the skill still runs and tells you which sources it skipped.
