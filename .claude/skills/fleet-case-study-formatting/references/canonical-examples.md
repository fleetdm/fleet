# Canonical examples: structural breakdown

Detailed notes on how each reference case study implements the skeleton from `SKILL.md`. Use this when the fast-path checklist isn't specific enough for the case in front of you.

## articles/fastly.md

- Sections: "The Challenge" → "Choosing Fleet" → "The results." Three `attribution-quote` divs, one per section, each from a different Fastly speaker (a manager and a Sr Client Platform Engineer), showing that quotes don't all have to come from one person.
- No `checklist` div — the results section is pure narrative with numbers threaded into sentences (99.9% of Macs migrated within 45 days, ~1,200 employee devices, 100+ POPs).
- `companyInfo` covers what Fastly does generally; no `companyInfoLineTwo`.
- Closes without a "looking ahead" section — the last quote (on GitOps/cost/agility) serves as the closing beat.

## articles/faire.md

- Sections: "Why Faire needed a change" → "The search for a solution" → "Choosing Fleet" → "The future." Four sections instead of three — the evaluation criteria get their own section, separate from "choosing Fleet."
- Uses a `checklist` div for the three MDM-selection priorities (API-first architecture, comprehensive Apple support, a reputable vendor) — this is the clearest example of `checklist` used for *decision criteria* rather than results.
- Only one `attribution-quote` div, placed in the "Choosing Fleet" section.
- Ends with an outbound link ("Read Faire's article...") to the customer's own externally-published account — a legitimate closing move when one exists.
- `companyInfoLineTwo` used to add headcount/geography/device-mix detail that didn't fit in the first `companyInfo` sentence.

## articles/stripe.md

- Sections: "Why Stripe needed a change" → "The search for a solution" → "Choosing Fleet" → "The results." Same four-part shape as Faire, but "the search for a solution" here is Stripe's own evaluation criteria narrated in prose (not a `checklist` div) — three paragraphs, each opening with "First," "Next," "Finally."
- Four `attribution-quote` divs, all from the same speaker (Staff Client Platform Engineer) — short, punchy quotes (one is a single sentence) rather than long multi-sentence ones. Demonstrates that quotes can be very short and still carry a section.
- Numbers are specific and dramatic: 10,000 Macs migrated in 10 days, "hundreds of thousands" saved annually — repeated in the title, a quote, and the results section, which is acceptable because it's the single headline number, not incidental repetition.

## articles/thumbtack.md

- Sections: "The Challenge" → "Why Fleet" → "The outcome" → "Looking ahead." Four sections including a closing look-ahead.
- Opens the challenge section with a specific incident (a misconfigured nudge-profile date forcing an immediate OS update company-wide) before generalizing to the broader problem — concrete-example-first is a strong pattern when the customer has a good story to tell.
- Four `attribution-quote` divs from one speaker, densely packed through "Why Fleet" (three in a row) — shows quotes can cluster in the section they're most relevant to, not just one per section.
- Uses a plain bullet list ("With Fleet, Thumbtack has:") instead of a `checklist` div for the five headline results — a legitimate alternative when each bullet needs slightly more than a two-to-four-word phrase.
- "Looking ahead" doesn't just gesture at the future — it names concrete next steps (deeper CI/CD integration, broader policy coverage) and lands a closing line restating the through-line, mirroring how `fleet-article-formatting` handles closings.

## articles/mollie.md

- Similar four-part shape, distinguished by a compliance/regulatory angle (PCI, DNB, DORA) — shows the skeleton adapts to a security/compliance-driven story, not just a migration-speed story.
- `companyInfo` explicitly states the regulatory context ("As a regulated fintech, Mollie operates under DNB and DORA oversight") since it's load-bearing for the whole narrative.

## articles/primo.md

- Sections: "The challenge" → "Why Fleet" → "The outcome" → "Looking ahead." Distinctive because Primo is a platform built *on top of* Fleet (an MSP-style customer), not a typical IT-team-manages-its-own-fleet story — the skeleton still holds, but "the outcome" describes what Primo's *customers* experience, one layer removed from Primo itself.
- Three `attribution-quote` divs, all the founder/CEO — appropriate for a smaller company where one person is the primary voice.
- `summaryKeyResults` mixes quantified results ("two to three minutes instead of three to four hours") with a scale statement ("400 customers, 30,000 devices, one platform") as its final item — a good closing beat for a summary list.

## articles/foursquare.md

- Sections: "The Challenge" → (evaluation) → "Choosing Fleet" → results narrative. Shortest of the canonical set.
- `checklist` div holds exactly two results (50% less maintenance effort, 24% lower spend) — proof the div works fine with just two items, not only longer lists.
- `companyInfoLineTwo` used for a second descriptive sentence about the company's positioning, distinct from Faire's use of it for scale/geography facts — the field is flexible, use it for whatever second fact matters most.

## Anti-pattern: a customer story that isn't a case study

`articles/deputy-achieves-compliance-and-clarity-with-fleet.md` looks like a case study — it uses a `checklist` div, has a Challenge/Solution/Results shape, and is about a customer's experience with Fleet. But it's tagged `<meta name="category" value="announcements">`, not `case study`, and has none of the case-study-specific meta tags (`summaryChallenge`, `summarySolution`, `summaryKeyResults`, `companyName`, `quoteContent`, etc.). It also uses inline markdown links in prose ("leveraged Fleet's robust [API]...") and Title Case-ish short headings ("## Challenge," "## Solution") rather than the fuller sentence-case headings the current pattern uses.

Treat this as a real, in-repo example of the boundary case called out in `SKILL.md`'s scope section: a customer narrative that was published as an announcement rather than a full case study, likely because it didn't have the dedicated logo/headshot image assets or the fuller summary content the case-study template requires. If asked to "fix" a piece like this into the case-study format, the right move is to ask whether the author wants to invest in the missing assets and summary meta tags to promote it to a real case study, not to silently add `category: case study` without those.
