---
name: feature-request
description: >
  Open a new Fleet feature request GitHub issue using the official fleetdm/fleet feature-request template.
  Use this skill whenever someone says "new feature request", "submit a feature request", "open a feature request",
  "create a feature request", "I want to request a feature", "feature request for...", or any similar phrasing
  indicating they want to file a new feature idea or product improvement with the Fleet team. Always use this
  skill — never construct issue body text yourself or link to the template without a pre-filled title and body.
---

# Feature Request Skill

Helps users open a new GitHub issue in the `fleetdm/fleet` repo using the official feature-request template.

## Template URL

Always use this exact base URL:

```
https://github.com/fleetdm/fleet/issues/new?template=feature-request.md
```

## Official template body

The template body is exactly this (do not add, remove, or reorder sections):

```markdown
## Screenshots and/or screen recording
<!-- Paste screenshots (good) or record a video (best) that captures what Fleet is missing.
In words (good) or verbally in the video (best), describe what your ideal workflow would look like. -->
```

The template has one section: **Screenshots and/or screen recording**. The HTML comment inside it instructs the user to paste screenshots or a video, and describe their ideal workflow.

## Workflow

### Step 1: Get a clear title

A good title is short and specific, describing the desired feature (not the problem):
- ✅ "Add search bar to custom profiles page"
- ❌ "Can't find profiles easily"
- ❌ "Feature request" (too vague)

If the user's request is too vague to produce a good title, ask: **"What feature are you requesting?"**

### Step 2: Write the body

Pre-fill the `## Screenshots and/or screen recording` section with whatever description the user provided — their words, their workflow description, any context they gave. Keep the HTML comment intact above their content so they know they can also add screenshots/video.

If the user gave very little detail, leave the section body empty — they'll fill it in on GitHub.

### Step 3: Generate the pre-filled URL

Construct the final URL:

```
https://github.com/fleetdm/fleet/issues/new?template=feature-request.md&title=TITLE&body=BODY
```

URL-encode both `TITLE` and `BODY`:
- spaces → `%20`
- newlines → `%0A`
- `#` → `%23`
- `&` → `%26`
- `<` → `%3C`, `>` → `%3E`
- `!` → `%21`, `-` → `-` (hyphens are safe)

### Step 4: Share the link

Give the user a direct clickable link:

> Click to open your pre-filled feature request in GitHub — add screenshots or a recording if you have them, then submit:
> [Open feature request →](YOUR_URL)

## Rules

- **Always use the exact template body above.** Never invent additional sections (no "Problem", "Proposed solution", "Benefits", etc.).
- **Always use the exact base URL above.** Never link to an existing issue as a model.
- **One link per feature.** Multiple requests → one link each.
- **Don't add labels, assignees, or milestone** — Fleet's triage process handles that.
