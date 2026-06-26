# Fleet style rules (full reference)

This mirrors Fleet's canonical guidance in `handbook/company/writing.md` and `handbook/marketing/fleet-ai-writing-instructions.md`. When those files are present in the repo, they win — read them. Use this when they aren't available.

## Contents
- [Voice and tone](#voice-and-tone)
- [Sentence structure](#sentence-structure)
- [Punctuation](#punctuation)
- [Capitalization and sentence case](#capitalization-and-sentence-case)
- [Fleet naming and terminology](#fleet-naming-and-terminology)
- [Words and phrases to avoid](#words-and-phrases-to-avoid)
- [Headings](#headings)
- [Lists](#lists)
- [Links](#links)
- [Numbers, dates, and times](#numbers-dates-and-times)
- [Code and Markdown](#code-and-markdown)
- [Competitor and Fleet framing](#competitor-and-fleet-framing)
- [Anti-AI patterns](#anti-ai-patterns)

## Voice and tone

Fleet's writing philosophy is "What would Mister Rogers say?" — helpful, neighborly, respectful, and honest.

- Treat the reader (IT pros, client platform engineers, security practitioners) as an equal and an expert peer.
- No snark, condescension, edginess, or sarcasm.
- Practice radical honesty. State bugs, limitations, and mistakes plainly. Never use spin to hide technical debt.
- Clarity over cleverness. If a sentence is clever but obscures meaning, rewrite it.
- Contractions are good (they're, don't, it'll, won't). They keep the tone approachable.
- Exclamation points: use sparingly, one at a time at most.

## Sentence structure

- Active voice: "Fleet manages hosts," not "Hosts are managed by Fleet."
- Imperative mood for instructions: "Click **Save**," not "You should click Save."
- One idea per sentence. If a sentence exceeds ~20 words, split it.
- Short and punchy beats long and qualified.

## Punctuation

- **Oxford comma:** always. "macOS, Windows, and Linux."
- **Em dashes:** avoid them. Use a comma, a colon, or a new sentence. (Most common AI tell.)
- **Commas over em dashes** is the default rewrite when you see a dash.
- **Quotation marks:** place punctuation outside the quotes unless it's part of the quoted string — e.g. write "osquery", not "osquery."
- **Spacing:** exactly one space after a period.
- **Colons:** introduce a list or a phrase that adds detail. Don't use a colon when a list immediately follows a heading.
- **Hyphens:** for ranges (Monday-Friday) and compound modifiers before a noun ("three-week cadence," but "released every three weeks").
- **Ampersands (&):** only in brand names or direct quotes; otherwise write "and."

## Capitalization and sentence case

- **Sentence case for all headings, subheadings, button text, and UI labels.** "Ask questions about your servers," not "Ask Questions About Your Servers." "Host details," not "Host Details."
- Capitalize only proper nouns, acronyms, and words with their own styling.
  - "MDM commands" — MDM is an acronym.
  - "macOS uses…" — macOS keeps its lowercase m.
  - "Nudge" — proper noun, stays capitalized.

## Fleet naming and terminology

- **Fleet** or **Fleet Device Management** — the company and the product. Never "FleetDM," "fleetDM," or "fleetdm" in prose.
- **Fleeties** — core team members.
- **fleet / fleets** — lowercase when referring to a group of devices.
- **osquery** — always lowercase. If it would start a sentence, rewrite so it doesn't.
- **fleetctl**, **fleetd** — lowercase; rewrite if one would start a sentence.
- **Fleet Desktop**, **Fleet UI**, **Fleet server**, **Orbit** — capitalized.
- Preferred nouns: "hosts," "devices," "computers." Prefer "device" over "endpoint." A device can be a phone, desktop, laptop, VM, or server.
- **Avoid** "agents" and "nodes."
- **Disk encryption:** use "disk encryption" generally. Use "FileVault" or "BitLocker" only when specifically referring to macOS or Windows.

## Words and phrases to avoid

- **Filler:** very, really, actually, basically, essentially, just.
- **Corporate/formal:** facilitate (use "help"), utilize (use "use"), leverage (use "use").
- **Hyperbole / hype:** revolutionary, game-changing, seamless, powerful, robust, unprecedented, industry-leading, best-in-class, cutting-edge, world-class.
- **Vague intensifiers and superlatives** in general — let specific facts carry the weight.

## Headings

- Sentence case (see above).
- No end punctuation unless the heading is a question.
- Reference topics: use a static noun. "Log destinations."
- Guides/tasks: use a task-based verb. "Configure a log destination."
- Avoid -ing verbs: "Configure a log destination," not "Configuring a log destination."
- Avoid vague verbs: "Log destinations," not "Understand log destinations."
- Don't put code in headings.
- Hierarchy: H1 page title, H2 main sections, H3 sub, H4 sub-sub. Use standard Markdown (`#`, `##`).

## Lists

- Unordered lists use hyphens (`-`).
- Ordered lists use numbers, for sequential actions.
- Introduce a list with a colon after a complete sentence; no colon when the list directly follows a heading.
- List items that are complete sentences get end punctuation; fragments don't. Be consistent within a list.

## Links

- Use full URLs rather than relative links, so content stays movable.
- Make link text meaningful. Link the descriptive words, not "here" or "click here."

## Numbers, dates, and times

- Spell out a number at the start of a sentence; otherwise use numerals.
- Numbers over 999 get commas (1,000, not 1000).
- Times use numerals with no space (7am, 7:30pm). Specify the time zone for a global audience.

## Code and Markdown

- Backticks for code, file paths, and terminal commands.
- Bold UI elements only — never bold for emphasis. Use italics for UI navigation paths (e.g. *Organization settings*) where the handbook does.
- Standard Markdown headings only.

## Competitor and Fleet framing

Apply the same discipline to competitors and to Fleet. Credibility comes from specificity, not superlatives.

- State facts only. Never editorialize. Describe what a product does, not how well it does it.
  - "Jamf provides macOS management capabilities" — not "Jamf provides excellent macOS management."
- Don't frame a competitor as the default or obvious choice ("gold standard," "known for its excellent…").
- Competitor limitations must be verifiable and specific.
  - "Kandji does not currently offer Linux endpoint management" — not "Kandji falls short on cross-platform support."
- State Fleet's genuine differentiators (GitOps-native workflow, open source, Linux support) plainly. Let the facts do the work.
- Write as a knowledgeable practitioner, not a salesperson. The audience tunes out vendor-pitch language.

## Anti-AI patterns

These survive careful drafting — read specifically for them before finishing:

- Em dashes used as connectors. Replace with commas, colons, or new sentences.
- Over-bolding and decorative bold. Bold UI elements only.
- Throat-clearing intros: "In the rapidly evolving world of…," "In today's fast-paced…," "It's important to note that…." Cut them; lead with substance.
- Passive voice that crept in. Convert to active.
- Filler and hype words (see lists above).
- Hedging and vague qualifiers ("might potentially," "in order to" → "to").
- Final check: "Is this the simplest way to say this? Would Fred Rogers approve of this tone?"
