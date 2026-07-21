---
name: fleet-article-formatting
description: Apply Fleet's house article format and article-specific voice to Fleet ARTICLES — blog pieces with meta category "articles" or "comparison" (thought-leadership, how-to, and comparison pieces). Use when writing a NEW article or editing, refreshing, or tightening an EXISTING one, even if the user doesn't say "use the format." Governs structure and article-specific voice; pair with content-style for word-level voice. Do NOT use for guides, case studies, or announcements — those are separate content types with their own skills.
allowed-tools: Read, Grep, Glob, Edit, Write, Bash(git diff*), Bash(git status*)
effort: medium
---

# Fleet article formatting

This skill governs article **structure and article-specific voice** — the section order, the key-takeaways pattern, the CTAs, and the honest-claims guardrails — for drafting new articles and bringing older ones up to standard. For word-level voice (sentence case, em dashes, filler, Fleet terminology, positioning), run the `content-style` skill over the prose. Apply both together on every article.

The format does one job: let a busy reader get the whole argument from the top of the page, then keep reading for the proof. "Key takeaways" and the CTA button sit immediately after the dek — before the intro — so a reader who scrolls no further still leaves with the argument and a next step. Every rule below serves that.

## Content types

Fleet publishes several content types under `articles/`. This skill governs the **article** format only. Identify the type first (the `<meta name="category" ...>` value is the routing signal), then apply the matching format. The canonical list of valid `category` values lives in [`website/scripts/build-static-content.js`](../../../website/scripts/build-static-content.js) (`validArticleCategories`); a fuller per-type explainer is in [`content-style/references/content-types.md`](../content-style/references/content-types.md).

| Content type | What it is | `category` value(s) | Format governed by |
|---|---|---|---|
| **Article** | Thought-leadership, how-to, and comparison pieces in the house format (title → dek → key takeaways → CTA → body → closing) | `articles`, `comparison` | **this skill** (+ `content-style` for prose) |
| **Guide** | Step-by-step operational how-to | `guides` | `fleet-guide-formatting` |
| **Case study** | Customer story; requires summary/quote meta tags | `case study` | its own template (build-enforced) |
| **Announcement** | Product/news announcement | `announcements` | `content-style` |
| Release notes, podcasts, webinars, whitepapers, reports | Other content types | `releases`, `podcasts`, `webinar`, `whitepaper`, `report`, … | out of scope here |

**Scope follows from this table:** apply this skill when the piece is an **Article** — `category` is `articles` *or* `comparison`. Both use the same format; `comparison` differs only in routing (it sets `category=comparison` and carries extra routing meta tags — see the website build requirements). For any other type, stop and use the owning skill; flag the mismatch to the author rather than reshaping their piece.

## The structure

Use this skeleton for Fleet articles — thought-leadership posts, how-to articles, and comparison pieces. Not every article needs every section, but the order is fixed.

```
# Title (sentence case)
*Italic dek — one or two sentences.*

## Key takeaways
- **Bold lead-in.** One to three sentences, outcome-first. (5–6 bullets.)

<a purpose="cta-button" href="https://fleetdm.com/relevant-page">Short action label</a>

[Intro — keep it short (about two short paragraphs); it opens the body proper.]

## [Body section]
### [Subsection]
...

## [Closing section]
[Short recap or stakes, then the through-line.]

## See it live
[Optional next-steps block: a guide link plus one or two demo/workshop bullets.]

---
*Italic CTA line with links.*
```

### Title

Sentence case, not Title Case. Lead with the reader's outcome or the tension, not the product name. "How Fleet completes your Microsoft stack" reads better than "Fleet: The Complete Microsoft Integration Platform."

### Dek (the italic line under the title)

One or two sentences, italicized. It frames the question the piece answers or the payoff the reader gets — it does not summarize the whole article. Think of it as the reason to keep reading. If the piece has a meta description, the dek can be a sharper, more human version of it.

### Key takeaways — the heart of the format

- Place it immediately after the dek, before the intro and the first body section. Nothing but the title and dek comes before it — the reader gets the whole argument at the very top of the page.
- 5–6 bullets. Each bullet: `**Bold lead-in phrase.**` followed by one to three sentences.
- **Lead with the business outcome, not the feature.** The bold phrase should state what's true for the reader or what they get ("Fleet sees it across every OS, in real time"), and the sentences explain why it matters. Avoid bullets that just name a feature.
- **Each takeaway previews a body section** — there should be a rough one-to-one mapping. If a takeaway has no home in the body, either cut it or add the section.
- **Preview, don't echo.** Do not copy a full sentence verbatim from the body into a takeaway. Say the same idea in different words. If the reader sees the identical sentence twice within a few hundred words, the takeaway reads as padding.
- **Takeaways must stand alone.** Because they now precede the intro, they can't lean on any setup — each bullet has to make sense to a reader who has read only the title and dek.

**Example of a strong takeaway (outcome-first, previews a section):**
> **Governance is code, not console clicks.** Reports and policies live in Git as YAML, get reviewed in a pull request, and deploy through CI, so your AI governance posture is auditable and reversible instead of a click someone made six months ago.

**Weaker version (feature-first, no stakes):**
> **GitOps support.** Fleet supports managing policies and reports as YAML in Git with CI/CD.

### CTA button after the key takeaways

Place a single call-to-action button directly after the key takeaways list, before the intro.

- **Syntax:** `<a purpose="cta-button" href="https://fleetdm.com/path">Short action label</a>` — an established pattern that renders as Fleet's primary (green) button in the article body. Use a full `https://fleetdm.com/…` URL, as existing articles do.
- **Make it relevant to the piece.** Match label and destination to the argument: "See config-as-code in Fleet" → `/infrastructure-as-code` for a config-as-code post, "Compare features" → `/pricing` for a comparison, "Get a demo" → `/contact` as the default.
- **One button only.** The fuller menu of next steps belongs in the closing CTA, not here. If the closing block also offers "Get a demo," vary the top button so the two CTAs aren't identical.

### Intro

The intro comes **after** the CTA button and opens the body proper. Keep it short — aim for about two short paragraphs. It can be narrative ("I've spent the last few months talking with teams…") or direct ("Your organization runs on Microsoft."), but it should end on a bridge that hands the reader into the first body section — a question, a stakes statement, or a "here's where the answer lives" line.

Because the takeaways have already summarized the argument, the intro doesn't need to — its job is to set up the problem and pull the still-reading reader into the body. If it runs past two paragraphs, fold the setup together and cut. Don't restate the takeaways in prose; that reads as padding coming right after them.

### Body

`##` for sections, `###` for subsections, all sentence case. Lead each section with its point, then support it. Use concrete, grounded specifics (real version numbers, real table names, real CVE-feed names) wherever you have them — specificity is what separates Fleet content from generic vendor copy.

For **integration or comparison pieces**, the cooperative "Fleet + X" section framing works well (e.g. "Fleet + Microsoft Intune," "Fleet + your SIEM"): name the other tool, give it genuine credit for what it does well, then show precisely where Fleet adds depth. This reads as confident rather than defensive. (Thought-leadership and how-to pieces don't need this framing — use plain topic headers.)

### Closing

A short section that restates the stakes ("The risk of waiting is real") and lands the through-line. Do **not** re-list every key takeaway here — that's the most common bloat. One or two synthesizing sentences, then the closing line.

### CTA

Fleet pieces typically carry two calls to action, and they play different roles:

- **The post-takeaways button** (above) — one button, high on the page, for the reader who's already convinced.
- **The closing CTA** — at the foot of the article, a fuller menu of next steps. This can be a short "See it live" block (a guide link plus one or two bullets such as **Get a demo** → `/contact` and **Join a GitOps training session** → `/gitops-workshop`) and/or an italic CTA line with links. Keep it to the actions that genuinely fit the piece; real links only.

## Voice and terminology

Fleet's voice is confident, specific, and honest. It respects the reader's intelligence and never oversells.

### Terminology rules

- **Say "Fleet's agent" or "fleetd," not "osquery,"** in marketing and customer-facing content. "Queryable," "query," and "live query" are fine as descriptors; in product-facing CTAs Fleet often prefers "report." Don't expose raw upstream project names where "Fleet's agent" reads cleaner.
- **Sentence case** for all headings and for credential/product names ("Certified Fleet expert," not "Certified Fleet Expert").
- Name competitors and other tools plainly and fairly — no scare quotes, no snark. The argument should win on substance.

### Honest-claims guardrails

Fleet content earns trust by being accurate. This is non-negotiable and applies whether drafting or editing.

- **Never invent an integration, capability, or parity claim.** If you're not certain Fleet does something, don't assert it. Verify against Fleet's docs (fleetdm.com), the changelog, or ask the author.
- **Hedge where the truth is partial.** Prefer "tends to wave it through" over "is completely blind to"; prefer "you can pipe Fleet data into Sentinel via your existing pipeline" over implying a turnkey native connector that may not exist. Accurate-but-modest beats impressive-but-wrong every time.
- **Ground specifics.** Tie capability claims to real versions/features when you can ("landed for macOS in Fleet 4.70.0, extended to Windows in 4.84.0"). Flag any claim you couldn't verify so a human can check it before publishing.
- Cross-platform coverage (macOS, Windows, Linux, and beyond) is usually Fleet's strongest differentiator — surface it, but only where it's genuinely relevant to the point.

### Formatting restraint

Default to prose. Use bullets and tables only where a list genuinely earns its place — enumerations (e.g. a fixed checklist you're contrasting against), step sequences, code, or scannable reference. Don't bullet a narrative. Don't bold half the sentence. The key takeaways are the one section that's intentionally list-heavy; the body should breathe.

### De-duplication

Each distinct point gets one primary home. A claim that appears five times across a piece loses force and bloats length. The takeaways preview the body, but beyond that, if you find the same idea stated in two sections, keep it in the stronger one and cut or cross-reference the other. When a point legitimately belongs in two places (takeaway + body), vary the wording so it reads as deliberate, not duplicated.

## Workflow: writing a new piece

1. Confirm the piece is an **article** (not a case study, announcement, or guide) and pin down the article type (thought-leadership, how-to, or comparison) and the single main argument.
2. Draft the body sections first — that's where the substance is. Ground every claim; flag anything unverified.
3. Write the dek and the intro. Keep the intro to about two short paragraphs; it opens the body, so end it on a bridge into the first section.
4. Derive the **key takeaways** from the finished body: one outcome-first bullet per major section, 5–6 total, previewing without echoing. Place them immediately after the dek, before the intro — each bullet must stand alone with no setup.
5. Add the **post-takeaways CTA button** (`<a purpose="cta-button" href="https://fleetdm.com/path">…</a>`) directly after the takeaways, before the intro, with a label and destination relevant to the piece.
6. Write the closing and the closing CTA.
7. Run the `content-style` skill over the whole draft for voice, sentence case, em dashes, filler, and Fleet terminology.
8. Run the self-check below.

## Workflow: updating and enhancing existing content

Use this to bring older Fleet content up to the current format. Work through it in order.

1. **Confirm it's an article.** Check the `<meta name="category" ...>` value against the Content types table. If it's `articles` or `comparison`, proceed. Otherwise stop — this format doesn't apply; tell the author rather than reshaping their piece.
2. **Read the whole piece** and identify its main argument and its natural section breaks.
3. **Add a dek** if there isn't one — an italic one-or-two-sentence framing under the title.
4. **Insert a "Key takeaways" section** immediately after the dek, before the intro. Derive 5–6 outcome-first bullets, roughly one per major section. Make them preview the body without copying sentences out of it, and make sure each stands alone — the reader hasn't seen the intro yet.
5. **Add the post-takeaways CTA button** (`<a purpose="cta-button" href="https://fleetdm.com/path">…</a>`) directly after the takeaways, before the intro, with a label/destination relevant to the piece.
6. **Tighten the intro** to about two short paragraphs. It now follows the CTA button and opens the body, so it should set up the problem and bridge into the first section — not restate the takeaways.
7. **Sweep terminology**: replace "osquery" with "Fleet's agent"/"fleetd" in prose, fix Title Case headings to sentence case, fix capitalized credential/product names.
8. **De-duplicate**: find claims repeated across sections and keep each in its strongest single home.
9. **Verify claims**: check every capability/integration/parity statement against Fleet's docs or the changelog. Soften or correct anything unsupported; flag anything you can't confirm for the author.
10. **Trim over-formatting**: convert bullet-soup back to prose where a list isn't earning its place; reduce stray bolding.
11. **Preserve structural metadata** (meta tags, author fields, frontmatter, existing links) exactly — don't drop them in the rewrite.
12. **Run the `content-style` skill** over the revised prose — it owns the voice, sentence case, em-dash, filler, and terminology rules referenced in the steps above; let it be the final word on word-level style.
13. Run the self-check below, then summarize the changes you made and list any claims you flagged for human verification.

## Self-check before finishing

- Title is sentence case and outcome-led; there's an italic dek that frames rather than summarizes.
- "Key takeaways" sits immediately after the dek, before the intro: 5–6 outcome-first bullets, each previewing a section, each standing alone, none echoing a body sentence verbatim.
- A single CTA button follows the key takeaways and precedes the intro, with a label and destination relevant to the piece.
- The intro is short (about two short paragraphs), doesn't restate the takeaways, and ends on a bridge into the first body section.
- Body sections lead with their point and use grounded specifics; comparison pieces credit the other tool before adding depth.
- No "osquery" in customer-facing prose; headings and credential names are sentence case.
- Every capability claim is true and grounded; partial truths are hedged; unverified claims are flagged.
- No idea is repeated across sections without a reason; formatting is restrained outside the takeaways.
- Closing lands the through-line without re-listing the takeaways; the top button and closing CTA aren't identical, and all links are real.
- The `content-style` skill has been run over the prose, and its voice, grammar, and terminology guidance is satisfied.

## Reference

A blank, copyable skeleton lives at `assets/article-template.md`. Start new articles from it when helpful.
