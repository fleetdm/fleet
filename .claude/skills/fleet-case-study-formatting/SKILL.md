---
name: fleet-case-study-formatting
description: Apply Fleet's house format, voice, and required meta tags to Fleet CASE STUDIES — customer success stories published under `<meta name="category" value="case study">`. Use when drafting a new case study from a customer interview/quotes, or editing, tightening, or auditing an existing one, even if the user doesn't say "case study" (triggers: "customer story", "success story", "write up [Company]'s Fleet migration", "case study for X", a request built from call notes/quotes about a customer's experience with Fleet). Governs the challenge → choosing Fleet → results structure, the `attribution-quote` and `checklist` custom divs, the case-study-specific endmatter (summaryChallenge/summarySolution/summaryKeyResults, company/quote meta tags) the website build enforces, and the non-negotiable rule that every quote is verbatim and every number is customer-sourced. Pair with content-style for word-level voice. Do NOT use for articles, guides, or announcements — those are separate content types with their own skills, even when they also tell a customer story.
allowed-tools: Read, Grep, Glob, Edit, Write, Bash(git diff*), Bash(git status*)
effort: medium
---

# Fleet case study formatting

A Fleet case study exists to let a skeptical prospect hear, in a real customer's own words, that switching to Fleet solved a specific problem. It is not an article, and it is not marketing copy dressed up with a logo — it's a structured, quote-driven narrative that the website renders into a dedicated template with a sidebar summary, a company card, and a hero pull-quote. This skill governs that **structure, custom syntax, and required meta tags**; pair it with `content-style` for word-level voice (sentence case, no em dashes, no filler, no hype).

## Scope — when this skill applies

Applies to pieces published under `<meta name="category" value="case study">` — a customer's story of adopting Fleet, told in three acts (their problem, why they chose Fleet, what changed) and anchored by real attributed quotes.

Does **not** apply to:

- **Articles** (`category` = `articles` or `comparison`) — thought-leadership, how-to, comparison pieces. Use `fleet-article-formatting`.
- **Guides** (`category` = `guides`) — step-by-step procedures. Use `fleet-guide-formatting`.
- **Announcements** (`category` = `announcements`) — even when they narrate a customer's experience. `articles/deputy-achieves-compliance-and-clarity-with-fleet.md` is a real example: it reads like a case study but is tagged `announcements` and skips the case-study meta tags entirely. If a customer-story draft doesn't have (or won't have) the summary/quote/company meta tags below, it's probably meant to be an announcement or article, not a case study — ask rather than forcing this template on it.

Before applying this format, check the `<meta name="category" ...>` value, or ask which type the piece is destined for.

## Canonical examples

Read one or two before writing a new case study — `articles/fastly.md`, `articles/stripe.md`, `articles/thumbtack.md`, `articles/mollie.md`, `articles/faire.md`, `articles/foursquare.md`, `articles/primo.md`. All follow the same skeleton with different section names suited to each story. See `references/canonical-examples.md` for a structural breakdown of each one.

## The structure

```
# Title (sentence case, names the company and the outcome)

## [Challenge section: "The Challenge" / "Why [Company] needed a change" / "The search for a solution"]
[Narrative: the old tool/process, the pain, why it became untenable.]

<div purpose="attribution-quote">

*Verbatim quote about the pain.*

**Speaker name**

Title, Company
</div>

## [Decision section: "Choosing Fleet" / "Why Fleet" / "The search for a solution"]
[Narrative: evaluation criteria, why Fleet won, how the switch went.]

<div purpose="checklist">        <!-- optional, for evaluation criteria or a short outcome list -->

First criterion or result

Second criterion or result
</div>

<div purpose="attribution-quote">
...
</div>

## [Results section: "The results" / "The outcome"]
[Narrative: quantified outcomes, what's different now.]

<div purpose="attribution-quote">
...
</div>

## [Optional closing: "Looking ahead" / "The future"]
[What's next for the partnership — a plan, an expansion, a platform they're evaluating next.]

<meta name="category" value="case study">
<meta name="articleTitle" value="...">
<meta name="description" value="...">

<meta name="publishedOn" value="YYYY-MM-DD">
<meta name="authorGitHubUsername" value="n/a">
<meta name="authorFullName" value="Fleetdm">

<meta name="companyLogoFilename" value="...">
<meta name="quoteAuthorImageFilename" value="...">
<meta name="quoteAuthorName" value="...">
<meta name="quoteAuthorJobTitle" value="...">
<meta name="quoteContent" value="...">

<meta name="companyName" value="...">
<meta name="companyInfo" value="...">
<meta name="companyInfoLineTwo" value="...">    <!-- optional -->

<meta name="summaryChallenge" value="...">
<meta name="summarySolution" value="...">
<meta name="summaryKeyResults" value="...; ...; ...">
```

### Title

Sentence case (except the company name, which keeps its own capitalization). Names the company and states the outcome, not the feature: "Fastly gains visibility into all endpoints and critical infrastructure worldwide," "Stripe moved 10,000 Macs to Fleet, saving hundreds of thousands annually," "Thumbtack migrates more than 90% of Macs with no IT intervention." "How [Company] ..." also works when the story is about a mechanism ("How Mollie manages endpoint compliance the same way it ships software," "How Primo built an HR-driven IT platform, powered by Fleet").

### The three-act narrative

Every canonical example is some variation of this arc, even though section headings differ story to story — pick headings that fit the story rather than copying one example's exact wording:

1. **The problem.** What the customer used before (or the process they had), why it broke down, and the cost of that (wasted time, risk, a specific incident, slow vendor support). Ground it in specifics: real tool names, real numbers of devices, a real incident (Thumbtack's mis-set nudge-profile date; Stripe's fragmented API workarounds).
2. **Choosing Fleet.** What they evaluated against (sometimes an explicit checklist of criteria), and why Fleet won. Tie each reason back to a capability, not a vague endorsement: API-first architecture enabling GitOps, open-source code enabling self-hosting or auditability, cross-platform coverage, responsive Slack-based support.
3. **The results.** Quantified, concrete outcomes: migration speed and percentage, cost/licensing savings, headcount or ticket-volume effects, new capabilities unlocked. This is where a `checklist` div listing bullet-style results is most common.

An optional fourth section ("Looking ahead," "The future") closes with what's next — expanded platform coverage, a deepening partnership, a new use case. Not every case study needs one; several of the canonical examples fold this into the results section instead.

### Custom syntax

Two custom `<div purpose="...">` blocks, unique to case studies (and rarely used elsewhere) — get them exactly right, the build pipeline and stylesheet depend on them:

**`attribution-quote`** — a pull quote inside the body, attributed to a named speaker:

```
<div purpose="attribution-quote">

*Italicized verbatim quote, exactly as the customer said it.*

**Speaker Name**

Title, Company
</div>
```

- The quote line is italic (`*...*`), the name is bold (`**...**`), the title/company line is plain text. Blank lines between each are required for the markdown to render correctly.
- Title and company can be one line (`Sr Manager Systems Engineering, Fastly`) or, when the company is already obvious from context, just the title.
- Use 2-4 of these per case study, spaced through the narrative at the moments they land hardest — right after describing the pain, right after the "why Fleet" reasoning, and in the results section. Don't front-load them all in one place.

**`checklist`** — a plain list of short items with no markdown bullet syntax, rendered as a checklist by the stylesheet:

```
<div purpose="checklist">

First item — a criterion, a feature, or a quantified result

Second item

Third item
</div>
```

- Items are separated by blank lines, not `-` or `*` bullets — the div's CSS supplies the checkmark styling.
- Use it for evaluation criteria (Faire's three MDM-selection priorities) or a short, punchy results list (Foursquare's cost/time savings). Longer results lists (4+ items with more explanation) can also work as plain markdown bullets in the results section prose, as Thumbtack's and Primo's "With Fleet, [Company] has:" lists do — either is fine, pick whichever reads cleaner for the story.

### Voice and honesty guardrails — stricter than other content types

Case studies carry more risk than an article: every claim is attributed to a real person at a real company, and it gets fact-checked by that company before or after publishing.

- **Quotes are verbatim, never paraphrased or invented.** If you don't have the customer's exact words for a moment the story needs, write the narrative around it without a quote, or flag it for the author to source a real quote. Do not manufacture a plausible-sounding quote and attribute it to a named person.
- **Numbers come from the customer, not from you.** Migration percentages, device counts, cost savings, timeframes — these all must trace back to what the customer or the Fleet team working with them actually said. Flag any number you're inferring rather than quoting/sourcing directly.
- **Name Fleet's own team members' contributions honestly** (e.g., a CSM's role in a migration) only if that's part of the customer's account, not invented color.
- Otherwise, the standard `content-style` rules apply in full: sentence case headings, no em dashes, no hyperbole ("revolutionary," "seamless," "game-changing"), Oxford commas, active voice. Case study prose reads slightly more narrative/conversational than a guide, closer to an article, but it's still restrained — let the customer's own quotes carry the enthusiasm; the narrator's voice around them stays measured and factual.

### Endmatter — case-study-specific meta tags (build-enforced)

Beyond the standard article endmatter (`articleTitle`, `description`, `publishedOn`, `authorFullName`, `authorGitHubUsername`, `category`), the website's markdown build (`website/scripts/build-static-content.js`) requires additional tags for `category: case study` pages, and will fail the build if they're missing:

- **`summaryChallenge`, `summarySolution`, `summaryKeyResults`** — required. These populate the sidebar "Challenge / Solution / Key results" summary box on the case study page (`website/views/pages/articles/case-study.ejs`). `summaryKeyResults` **must** be a semicolon-separated list (`"Result one; Result two; Result three"`) — a single sentence with no semicolon fails the build.
- **`companyLogoFilename`** — if present, must reference a file that actually exists in `website/assets/images/`. The build checks this and errors if the file is missing.
- **`quoteContent`** (the hero pull-quote at the top of the page) — if present, requires `quoteAuthorName`, `quoteAuthorJobTitle`, and `quoteAuthorImageFilename` alongside it, and the image file must exist in `website/assets/images/`.
- **`companyName`, `companyInfo`** (and optional `companyInfoLineTwo`) — power the "About [Company]" sidebar/mobile block. Not build-enforced, but expected by the template; include them.

**Image assets are a hard dependency, not something this skill can produce.** Before the build will succeed, someone needs to add the actual logo and headshot files to `website/assets/images/`, following the site's naming convention (`{descriptor}-{css-width}x{css-height}@2x.{ext}`, e.g. `fastly-logo-104x40@2x.png`, `dan-jackson-120x120@2x.png`). If those files don't exist yet, say so explicitly rather than writing `<meta>` tags that point at nothing — that's a silent build break waiting to happen.

An escape hatch exists (`<meta name="useBasicArticleTemplate" value="true">`) for a case study that should render with the plain article template instead of the sidebar layout; it then requires `cardTitleForCustomersPage` instead of the summary/quote/company tags. This is rare — only reach for it if the author specifically doesn't want the sidebar template.

### Related, optional step: testimonials.yml

Many (not all) published case studies also get their pull-quote added as an entry in `handbook/company/testimonials.yml`, which feeds the `<scrollable-tweets>` carousel used across the site (e.g., the `/customers` page). This is a nice-to-have, not build-enforced, and outside this skill's core job — mention it to the author as a follow-up rather than doing it unprompted, since it touches a shared file curated by the marketing team.

## Workflow: writing a new case study

1. Confirm you have real material to draw from — customer quotes (verbatim, attributed), and concrete numbers (device counts, migration timelines, cost figures). If working from raw interview notes or a call transcript, extract the quotes and facts before drafting; don't smooth a real quote into different words.
2. Identify the three-act shape for this specific story: what was broken, why Fleet, what changed. Pick section headings that fit the story (see canonical examples) rather than reusing one example's headings verbatim.
3. Draft the body: problem → decision → results, threading 2-4 `attribution-quote` divs at the moments they land, and a `checklist` div for evaluation criteria and/or headline results if the story has a natural short list.
4. Write the title (sentence case, names the company and the outcome).
5. Write the full endmatter block: standard article tags, then the case-study-specific summary/quote/company tags. Derive `summaryChallenge`/`summarySolution` as tight paraphrases of the body's first two acts; derive `summaryKeyResults` from the results section, semicolon-separated.
6. Confirm image assets (`companyLogoFilename`, `quoteAuthorImageFilename`) exist in `website/assets/images/`, or flag that they still need to be added.
7. Run the `content-style` skill over the prose for voice, sentence case, em dashes, filler, and Fleet terminology.
8. Run the self-check below.

## Workflow: auditing or editing an existing case study

1. Confirm it's tagged `category: case study`. If it's an `announcements` piece narrating a customer story (like the Deputy example), don't force this structure onto it — flag the mismatch instead.
2. Read the whole piece and check the three-act shape is intact: problem, decision, results, each grounded in specifics rather than generic praise.
3. Check every quote reads as something a person would actually say (specific, occasionally imperfect phrasing) rather than smoothed marketing copy — that's often a sign a quote was paraphrased rather than transcribed. Flag for the author to verify against the source if uncertain.
4. Check the `attribution-quote` and `checklist` divs are syntactically correct (blank lines between quote/name/title, no stray markdown bullets inside `checklist`).
5. Verify the endmatter has all case-study-required tags and that `summaryKeyResults` is semicolon-separated.
6. Verify referenced image files (`companyLogoFilename`, `quoteAuthorImageFilename`) actually exist in `website/assets/images/`.
7. Run the `content-style` skill over the prose.
8. Run the self-check below, then summarize changes and flag anything you couldn't verify (a number, a quote, a missing image asset).

## Self-check before finishing

- Title is sentence case, names the company, and states the outcome rather than a feature.
- Body follows the problem → choosing Fleet → results arc, with section headings suited to this story.
- 2-4 `attribution-quote` divs, correctly formatted (italic quote, blank line, bold name, blank line, title/company), spaced through the narrative rather than clustered.
- Every quote is verbatim and attributed to a real, named person; no invented quotes.
- Every number (migration %, timeframe, cost, device count) traces back to the customer or Fleet team, not an inference.
- A `checklist` div (or a plain bullet list) surfaces the headline results.
- Endmatter includes `summaryChallenge`, `summarySolution`, `summaryKeyResults` (semicolon-separated), and the company/quote meta tags — or, if using `useBasicArticleTemplate`, `cardTitleForCustomersPage` instead.
- `companyLogoFilename` and `quoteAuthorImageFilename` reference image files that exist in `website/assets/images/`, or the author has been told they still need to add them.
- No "osquery" in prose (say "Fleet's agent" or "fleetd"); headings sentence case; no em dashes, filler, or hyperbole.
- The `content-style` skill has been run over the prose.

## Reference

A blank, copyable skeleton lives at `assets/case-study-template.md`. A structural breakdown of each canonical example lives at `references/canonical-examples.md`.
