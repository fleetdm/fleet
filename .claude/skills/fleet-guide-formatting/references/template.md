# [Task-verb-led title in sentence case]

[One short paragraph. State the problem and what the reader ends up with. No history lesson, no "in today's landscape." If the guide doesn't cover every scenario, say so here in one sentence.]

## Prerequisites

<!-- Heading may instead be "Requirements," "What you'll need," or "Before you begin" — pick one, keep it consistent within the guide. -->

Check these before you start:

- [Concrete, checkable requirement — version, access level, or artifact in hand]
- [Another requirement. Gate by version inline if needed: "Fleet v4.86 or earlier: do X. Fleet v4.87 or later: no action needed."]

<!-- If there's a mistake that's costly or hard to undo, flag it here as a callout, not just a bullet: -->
> **Warning:** [What goes wrong, and how to avoid it. Keep it to the risk that matters most.]

## [First action, as an imperative heading — e.g. "Create a recipe override"]

[One or two sentences of setup, then the command or click-path.]

```bash
[command]
```

[What just happened, in one sentence, only if it's not obvious from the command.]

## [Second action]

<!-- If this step has two execution paths (UI vs. GitOps, mode A vs. mode B), pick one:
     - Sibling H2/H3 sections if the paths diverge enough to need their own sub-steps.
     - A numbered click-path list next to a code block as H3 siblings under one H2, if it's the same step via two methods. -->

1. [UI action, bolding the element name: Go to **Settings > Fleets**.]
2. [Next click.]

<!-- Inline gotcha tied to this specific step: -->
> **Note:** [What the reader will see if this doesn't apply to them, including the literal error text if there is one.]

## Verify

<!-- Include only if success or failure isn't obvious from the last step. Delete this section otherwise. -->

[How to confirm the change took effect.]

1. [Check one.]
2. [Check two.]

## Troubleshoot

<!-- Include only if there are known failure modes. Delete this section otherwise. -->

**[Bold symptom, e.g. "Permission denied errors"]**

[The fix, directly after the symptom. Add a code block if there's a command to run.]

**[Another bold symptom]**

[The fix.]

## Further reading

<!-- Optional. Rename to "Related resources" or "Get help" if that fits the content better — "Get help" for community-maintained tools without official support. -->

- [Link with descriptive text, not "here"]
- [Another link]

<meta name="articleTitle" value="[Must match the H1 exactly]">
<meta name="authorFullName" value="[author name]">
<meta name="authorGitHubUsername" value="[github username]">
<meta name="publishedOn" value="[YYYY-MM-DD]">
<meta name="category" value="guides">
<meta name="description" value="[1-2 sentences, 150 chars max. Factual, benefit-driven, no filler.]">
