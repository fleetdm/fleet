# Canonical examples: structural breakdown

Detailed notes on how each reference guide implements the skeleton from `SKILL.md`. Use this when the fast-path checklist isn't specific enough for the case in front of you.

## articles/deploy-fleet-on-docker-compose.md

- Opening states the outcome and time-to-complete in one sentence: "You'll have a Fleet instance running with MySQL and Redis in about 15 minutes."
- Prerequisites heading: "What you'll need."
- Steps are sequential H2 sections named as actions: "Download the configuration files" → "Configure your environment" → "Configure TLS" → "Start Fleet" → "Access Fleet."
- Branching handled with bold inline labels inside one section rather than separate headings: "**Option 1: Reverse proxy or load balancer handles TLS**" / "**Option 2: Fleet handles TLS directly**," with an explicit "Skip to 'Start Fleet' below" for readers who don't need option 2.
- Optional steps are labeled in the heading itself: "Optional: Add your license key," "Optional: Configure S3 storage."
- Troubleshooting: each item is a **bold symptom** used as a pseudo-heading ("**Permission denied errors on /logs**"), followed directly by the fix, sometimes with a code block.
- Ends with "Production considerations" — a bulleted list of hardening tips, not a summary. Still practical, not a recap.

## articles/migrate-fleet-server.md

- Opening explicitly scopes the guide down: "Every environment is different, so this guide focuses on the essential steps rather than trying to cover every possible scenario." This lets the guide skip edge cases without apologizing for it later.
- Prerequisites heading: "Before you begin," bulleted, each bullet bolds the action verb ("**Back up your database.**", "**Plan for downtime.**", "**Save your `FLEET_SERVER_PRIVATE_KEY`.**").
- The single highest-risk gotcha (losing the private key) is stated in the prerequisites bullet, then repeated verbatim as its own numbered item inside the "Set up the new Fleet instance" step, and repeated a third time in Troubleshooting. Repetition at the point of action is intentional for genuinely destructive mistakes — don't treat "don't repeat yourself" as an absolute in this case.
- Steps are sequential H2 sections: "Stop the Fleet server" → "Back up the MySQL database" → "Set up the new Fleet instance" → "Import the database" → "Configure S3 storage (if applicable)" → "Start Fleet on the new instance" → "Update DNS."
- Explicit "Verify the migration" section, itself a numbered list of checks, not just "you're done."
- "Additional notes" section between Verify and Troubleshooting holds true-but-not-actionable-right-now facts (Redis doesn't need migration, secrets live in MySQL). This is a legitimate fourth slot when a guide has caveats that aren't gotchas tied to a specific step and aren't failure modes either.
- Troubleshooting: bold symptom lead-ins as sub-headings within prose, each followed by a bulleted fix list.

## articles/enforce-macos-updates-per-major-version.md

- Prerequisites bullets are conditioned on Fleet version ("Fleet v4.86 or earlier: ... Fleet v4.87 or later: this flag is enabled by default. No action needed.") — version-gating lives inline in the bullet, not as a separate compatibility table.
- A `> **Warning:**` callout sits directly after prerequisites because using this guide's approach alongside a conflicting built-in feature breaks devices — the warning is positioned before the reader can make the mistake, not after.
- A short "How it works" H2 explains the mechanism in two sentences before any steps — this is a legitimate extra section when the "why this works" isn't obvious from the task name alone.
- Steps use explicit "Step 1: ...", "Step 2: ...", "Step 3: ..." H2 headings because the guide is fundamentally "create N things, once per OS version," and the count is the organizing structure.
- A `> **Note:**` callout is nested inside Step 1, immediately after the content that would trigger the problem it describes (a version-already-current error), including the literal error text the reader will see.
- A numbered UI click-path list is nested inside Step 3 for the "Using the Fleet UI" path, sitting next to a code block for the "Using GitOps" path as a sibling H3 — same step, two execution methods, not two different steps.
- "Verify" is its own H2 with a numbered click-path.
- Ends with "Related resources" as a plain link list.

## articles/set-device-hostname-via-fleet-api.md

- Prerequisites are three bullets, all concrete artifacts the reader must already have (token, serial number, enrollment state) — no soft prerequisites like "familiarity with APIs."
- Steps are sequential H2 sections matching the literal API call sequence: "Get the host UUID" → "Create the rename command" → "Base64 encode the command" → "Send the command."
- Bold labels replace sub-headings for structured request/response data: "**Endpoint:**", "**Headers:**", "**Body:**" — this is the right pattern for API guides specifically, in place of prose description of the HTTP call.
- A callout about a strict requirement (`CommandUUID` must be unique) is placed as a **bold-lead sentence inline**, not a blockquote — blockquotes aren't mandatory for every gotcha; a bold lead sentence works when the gotcha is one sentence and directly inside the step it affects.
- No Verify or Troubleshooting section — appropriate because the guide is a single API call with an obvious pass/fail (the request either 200s or it doesn't), and there's nothing failure-prone enough to warrant one. Don't add sections the task doesn't need.

## articles/manage-bootstrap-package-with-gitops.md

- The shortest example: intro (2 sentences) → one `>` Note callout (fleets can't share bootstrap packages) → Prerequisites → three action-headed H2 steps → "More information" link. No Verify, no Troubleshooting.
- Demonstrates that the skeleton compresses cleanly for a small task — don't pad a three-step guide with a Verify or Troubleshooting section just to look complete.

## articles/autopkg-with-fleet.md

- Opening explains what the third-party tool is before anything else, since the reader may not know it, and explicitly disclaims official support: "It's not an official Fleet product and isn't directly supported by Fleet."
- Two execution modes ("Direct mode" and "GitOps mode") are siblings H2s, each self-contained with its own prerequisites subsection ("Additional prerequisites for GitOps mode") and its own steps — rather than one shared step list with branches inside it. Use sibling H2 branches (as here) when the two paths diverge enough to need their own sub-steps; use inline bold-labeled options (as in the Docker Compose TLS example) when the branch is a single short choice.
- A `>` callout justifying a design decision ("Why S3?") is placed where the reader would otherwise ask "why not just upload directly," answering the objection instead of ignoring it.
- Ends with "Get help" instead of "Further reading" because the tool is community-maintained — the section name should match what the reader actually needs (support channels, not background reading).

## articles/canary-fleet-for-fleetd-updates.md

- Opens by naming the problem (EDRs flagging fleetd) for three sentences before naming the fix — appropriate when the reader may not yet believe they need this guide. Contrast with `set-device-hostname-via-fleet-api.md`, which states the task in sentence one because there's no motivating problem to sell.
- A `>` callout for a licensing gate ("`update_channels` is only available in Fleet Premium.") sits right after the concept explanation and before the steps, so a Free-tier reader doesn't follow steps that won't work for them.
- Steps are a numbered list nested under a single H2 ("Set up your canary fleet") rather than one H2 per step — appropriate for a short, three-item sequence that doesn't need step-level anchors.
- Closing section ("Start small, catch problems early") reads like a summary but earns its place by adding new practical framing (pick one device per platform, watch for updates) rather than restating prior sentences. This is the narrow exception to "no conclusion section" — allowed only when the closing paragraph still tells the reader what to do next, not just that they've reached the end.

## articles/managed-migration-assistant-mac-to-mac-migration-with-fleet.md

- "Requirements" (not "Prerequisites") holds version and enrollment-method constraints.
- A dedicated "What transfers and what doesn't" H2 sits between Requirements and the first configuration step — this is reference material the reader needs in their head before they touch config, not a step itself. Legitimate as its own section when steps would be misconfigured without it.
- GitOps vs. UI paths are H3 siblings under "Configure Managed Migration Assistant in Fleet," each a short numbered list — same pattern as the enforce-macos-updates guide's Step 3.
- An `> **Warning:**`-equivalent constraint stated as a bold-lead sentence inline ("One constraint from Apple: the **Restore** pane ... cannot be hidden") — again, inline bold works for a single-sentence gotcha; reserve full blockquotes for gotchas that need more than one sentence or a code sample.
- Closes with "End-to-end flow": a numbered list walking the full process across both Macs. This is a legitimate closing section distinct from a summary — it's a sequence diagram in prose, useful because the actual steps were split across two systems (source Mac, destination Mac, Fleet) and the reader needs to see them stitched together once.
- "Further reading" link list at the very end, before endmatter.

## Anti-pattern: an article wearing the guides tag

Watch for pieces tagged `category: guides` that are structurally articles. The tells:

- No Prerequisites/Requirements section at all.
- No numbered steps and no task-headed H2 sections. Headings are topic nouns describing changes or themes ("TLS requirements are getting stricter," "Intel Mac support timeline"), not actions the reader takes.
- Body paragraphs are multi-sentence analysis and framing, not procedure. This can be entirely voice-compliant prose — the problem is structural, not a style violation.
- Closes with a numbered "recap" or "priorities" list whose items are strategic takeaways ("start the budget conversation for X"), not steps of one task working toward a shared goal. A numbered list alone doesn't make something a procedure.
- If asked to "fix" a piece like this, the right move is not to force prerequisites and steps onto it. Flag that it's mistagged and recommend `category: articles`, or ask whether the intent was actually a guide — in which case it needs a real procedure written, not a reformat of the existing prose.
