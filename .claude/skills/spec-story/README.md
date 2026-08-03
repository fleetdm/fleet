# /spec-story

Breaks a Fleet story issue into implementable, parallelizable sub-issues with deep technical specs. On your approval, it creates them in GitHub and fills in the parent story's Engineering section.

## Invoke

```
/spec-story <issue-number-or-url>
```

It runs as a four-stage **Understand → Skeleton → Draft → Create** workflow and pauses for your explicit approval at each gate. Nothing is created in GitHub until you approve the drafted sub-issues.

## What it already does

These are built in, so you don't need to spell them out in your prompt:

- **Research for context:** GitHub (issues, PRs, and discussions), Slack, Gong, and git history.
- **Decomposition for parallel work:** splits by specialization (backend, frontend, fleetctl-GitOps, and agent) along code boundaries, so different engineers can work in parallel. It always adds a Documentation and QA sub-issue, and never mixes backend and frontend in one sub-issue.
- **Schema-first grounding:** reads `server/datastore/mysql/schema.sql` and recent `migrations/tables/` before it greps code paths.
- **Figma Ready page:** pulls the `Ready` page by `node-id` and copies dev notes, tooltip text, and message text verbatim. It skips the cover, which only links to the issue.
- **Independent verification:** before it presents the draft, a separate skeptical subagent re-checks `file:line` references, schema, and GitHub claims against the real sources.
- **High reasoning effort:** set in the skill's frontmatter, so you don't need to add "ultrathink".

## What to give it

Supply these up front so it doesn't have to stop and ask in Stage 1:

- **The story:** the issue number or URL. Required.
- **The Figma Ready-page link** with its `node-id`, for example `.../design/...?node-id=2-130&m=dev`. Or tell it there's no design, which is common for backend-only stories.
- **The accompanying doc-change PRs:** the REST API, GitOps, or audit-log PRs. They define the field names and shapes the spec must match, and conflicts between them need explicit reconciliation. Or tell it there are none.

## Prompt template

```
/spec-story https://github.com/fleetdm/fleet/issues/<N>

Figma (Ready page): <figma design URL with node-id>
Doc-change PRs: #<a>, #<b>, #<c>
```

That's all it needs. The skill handles research, schema grounding, decomposition, drafting, verification, and creation from there. Add a line only for story-specific scope it can't infer.

## Prerequisites

- `gh`, authenticated with `gh auth login`.
- Figma, Slack, and Gong context requires the matching MCP servers to be connected. These aren't part of the out-of-box team setup. Without them, the skill still runs but skips those sources.
