---
name: content-style
description: Write, edit, and review any public-facing or customer/prospect-facing Fleet content so it follows Fleet's writing, brand, voice, and style guidelines. Use this whenever you create or change the words in website copy (fleetdm.com), handbook pages, docs, guides, tutorials, articles, blog posts, announcements, release notes, product UI text and microcopy, GitHub issues about content, or marketing and sales enablement material, even if the user never says "style guide." Trigger on edits under website/, handbook/, docs/, articles/, and on requests like "write a blog post," "draft an announcement," "review this guide for style," "make this sound like Fleet," "tighten this up," or "clean up this copy." This skill is for authoring and editing prose for voice, brand, and style. It does not apply to writing code, tests, scripts, commit messages, or internal team chat, even when those happen to mention articles, announcements, blog posts, or UI strings.
allowed-tools: Read, Grep, Glob, Edit, Write, Bash(git diff*), Bash(git status*)
effort: medium
---

# Fleet content style

Produce and refine written content that sounds like Fleet: radically honest, technically precise, and "Mister Rogers" kind. The reader is an IT professional, client platform engineer, or security practitioner. Write to them as a helpful expert peer, never as a salesperson.

This skill works both inside the Fleet repo and outside it. The rules below are self-contained, but when the handbook is present it is the source of truth — read it first so you always reflect the latest guidance.

## When you start

1. **Identify the content type and the mode.** Type drives format discipline: website copy, guide/tutorial, article/blog, announcement, docs reference, product UI text, or marketing/sales enablement. Mode is either *writing new* content or *reviewing/editing existing* content. If the type or intended placement is unclear, ask — don't guess, because format rules differ by type.

2. **Load the canonical guidelines if they exist.** When working in a repo that contains them, read these before writing, since they may have been updated since this skill was written:
   - `handbook/marketing/fleet-ai-writing-instructions.md` — the token-optimized ruleset (start here)
   - `handbook/company/writing.md` — the full writing guide (headings, links, lists, numbers)
   - `handbook/company/brand.md` — visual brand, naming, imagery
   If they aren't present (e.g. drafting external copy), use the embedded rules below.

3. **For marketing and sales content, align positioning.** Read `references/positioning.md` — a brief distilled from the "Implicating the pain" positioning doc and the "Fleet for IT engineers and admins" sales deck (links inside). It covers the audience's pains, the "implicate the pain" narrative method, Fleet's differentiators, and messaging do's and don'ts. Treat its proof points as needing verification: never publish a stat, customer quote, or named claim without confirming it against a public, approved source, and ask the user if you can't.

4. **Read the full rules for anything non-trivial.** `references/style-rules.md` has the complete mechanics with examples; `references/content-types.md` has per-type format discipline and the article/guide/announcement endmatter templates. The checklist below is the fast path, not the whole story.

## The core of Fleet's voice

- **Plain English, short sentences, active voice.** One idea per sentence. If a sentence runs past ~20 words, split it. "Fleet manages hosts," not "Hosts are managed by Fleet."
- **Imperative mood for instructions.** "Click **Save**," not "You should click Save."
- **Clarity over cleverness.** If a sentence is clever but obscures meaning, rewrite it. The goal is for the reader to understand, not for the writer to look smart.
- **Radical honesty.** State bugs, limitations, and gaps plainly. Never use marketing spin to hide them.
- **No snark, no hype.** Treat the reader as an equal. Never condescending, edgy, or sarcastic.

## High-frequency rules (the fast checklist)

These are the rules content most often gets wrong. Apply them on every pass.

- **Sentence case everywhere** — headings, subheadings, buttons, UI labels. "Host details," not "Host Details." Only proper nouns, acronyms, and self-styled names (macOS, osquery) keep their casing.
- **Commas, not em dashes.** Avoid em dashes entirely. Use a comma, a colon, or a new sentence. This is the single most common AI tell to strip.
- **Oxford comma, always.** "macOS, Windows, and Linux."
- **Bold UI elements only.** "Navigate to **Hosts**." Never bold for emphasis or to decorate. Excessive bolding is an AI tell.
- **Cut filler.** Delete "very," "really," "actually," "basically," "essentially," and "just." Use common words: "help," not "facilitate."
- **No hyperbole.** Strip "revolutionary," "game-changing," "seamless," "powerful," "robust," "unprecedented," "industry-leading," "best-in-class."
- **Fleet naming.** The product and company are **Fleet** (or Fleet Device Management) — never "FleetDM," "fleetDM," or "fleetdm" in prose. Lowercase `osquery`, `fleetctl`, `fleetd` (rewrite the sentence if one would start it). Capitalize Fleet Desktop and Orbit.
- **Preferred terms.** Use "hosts," "devices," "computers," "Fleet UI," "Fleet server." Avoid "agents" and "nodes." Prefer "device" over "endpoint."
- **No fabrication.** Never invent features, URLs, names, dates, version numbers, or CLI commands. If you don't have something you need, ask — it's better not to know than to make it up.
- **Strip AI intros.** No "In the rapidly evolving world of…," no throat-clearing. Lead with the substance.

## Writing mode

Draft directly in the target format and apply the rules as you write — don't write loose prose and clean it up after. Lead with the "why" before the "how." Match the format discipline for the content type (see `references/content-types.md`): guides get numbered steps, announcements lead with the news, website copy is scannable and benefit-focused, articles are conversational but professional.

Before presenting a draft, do one self-review pass against the checklist above, reading specifically for em dashes, over-bolding, filler, hype, and sentence-case slips — these survive even careful first drafts.

## Review / edit mode

When the user hands you existing content (or you're editing files in a diff):

1. Read the content and identify its type.
2. Go through it against the checklist and the full rules, noting each issue with its location.
3. Apply the fixes directly (Edit), or if the user wants a critique, report findings as a short list grouped by rule — quote the original, give the corrected version, and name the rule. Keep it specific and actionable; don't pad with praise.
4. Preserve the author's meaning and technical accuracy. Style edits must never change facts. If a sentence is factually unclear, flag it rather than rewriting it into something that might be wrong.

When in doubt, simplify.
