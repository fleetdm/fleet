# Fleet positioning and messaging brief

Use this to align tone and framing for marketing and sales enablement content, and any copy that needs to land Fleet's value. It's distilled from two canonical sources:

- **Implicating the pain** (positioning doc): https://docs.google.com/document/d/1h8N9Icow6g-08EZPQMnbH-bDKDbXoPy-lwlUm9DiPJA/edit
- **Fleet for IT engineers and admins** (sales deck): https://docs.google.com/presentation/d/1WTyGrmA4pSB7H8BeT14BF7peozBceToW8TK__doyQTg/edit

> **Read this before lifting anything from it.** The positioning doc is an internal, in-progress strategy document. It contains raw language, TODOs, and specifics that are not cleared for public use (internal deal sizes, unverified stats, AI-vendor implementation details). Mine it for **framing and voice**, not for sentences, quotes, or numbers. Never publish a stat, customer quote, or claim from here without verifying it against a public, approved Fleet source. This is the same no-fabrication discipline the rest of the skill enforces.

## Audience and their pains

The reader is a high-agency "doer" — an IT admin or IT leader (often at a 700+ employee company), plus the CIOs, CISOs, and MacAdmins around them. They already know device management matters. They don't need to be sold on the category; they need it to be faster and easier. Their emotional core is the fear of looking slow or ineffective, and the desire to be the person who made the company faster.

Concrete pains the docs name (useful as the "before" in any narrative):

- **Slow to change** — afraid to "press the button"; no rollback, no peer review, no record of who changed what.
- **The last mile** — "push and pray"; rollouts stall at 75%, and the exposed 25% are often execs, legal, and finance.
- **Left behind** — pressure to use AI for real, but click-ops tools have no safe "human in the loop" path to it.
- **Roach motel** — vendor lock-in; cloud-only restrictions that clash with data-residency needs; shrinking human support.
- **Meshy tools** — separate tools, teams, and APIs per OS. "Two or more sources of truth" means no source of truth.
- **Busywork** — large chunks of team time spent gathering audit and leadership evidence, or waiting on slow scripts.
- **Off limits / stumped** — can't see into acquisitions and subsidiaries; can't easily pull current or historical data on AI usage, shadow IT, vuln exposure, or compliance posture.
- **Trust gap** — employees won't take "trust me" that IT isn't spying; buyers won't take "trust me" on a closed-source vendor.

## The narrative method: implicate the pain

This is the core move, and it's the opposite of a feature tour. Lead with the reader's pain, then their desire. Cost savings is justification, not desire — it's the rationalization made *after* someone already wants Fleet, so don't lead with it.

The arc:
1. Surface the pain with concrete, lived scenarios, not abstractions.
2. Quantify and amplify it (how much time, who else is affected).
3. Bridge to a relevant outcome ("that reminds me of…").
4. Reframe **Old IT vs. New IT**.
5. Future-pace: if you could do this tomorrow, how would you know you're better off, and who else benefits?

The **Old IT vs. New IT** contrast is the recurring engine. Old IT: slow, afraid to change, locked in, meshy, push-and-pray, siloed. New IT: "you can just do things," freedom at every level, see reality clearly, open by design — manage devices like DevOps, with a human in the loop. The throughline is **operational speed**, tied to the AI moment: don't be left behind; be the real deal.

The four "promised land" pillars: (1) you can just do things (access), (2) freedom at every level (flexibility), (3) see reality clearly (clarity), (4) open by design.

## Differentiators (state them plainly, let facts do the work)

- One source of truth across macOS, Windows, Linux, and mobile — replaces a stack of per-OS tools.
- Source-available and open by design — transparency for both employees and buyers.
- GitOps and infrastructure-as-code — version control, peer review, rollbacks, audit trail, and the safe path to AI-assisted change.
- Real-time visibility via osquery — query any device in seconds; treat the fleet as a live database.
- Clear diagnostics — see *why* a rollout or profile failed without dragging the end user onto a call.
- Deployment flexibility — on-prem or cloud, no lock-in, supports data-residency needs.
- A single modern API across every OS.
- Fast, agnostic migration — enroll through an existing MDM for instant visibility without taking over.

## Messaging do's

- Lead with pain, then desire. Keep speed as the throughline.
- Use sharp, concrete Old IT vs. New IT contrasts.
- Tell customer outcomes through specific scenarios, not adjectives.
- Speak to transparency — "open by design," no "trust me."
- Tie to the AI moment honestly: a human in the loop, manage devices like DevOps.
- Acknowledge real objections (migration has historically been painful) and reframe rather than dodge.

## Messaging don'ts

- No feature bake-offs or "harbor tour" demos divorced from pain.
- Don't lead with "replace Jamf to save money" — it frames Fleet defensively.
- Don't promise unbuilt features. Redirect to Fleet's pace and openness.
- Watch hyperbole — the source doc itself flags its own overstatements as inefficient. Specificity beats superlatives.
- Don't overclaim where Fleet isn't differentiated (e.g. end-user UX is roughly on par with incumbents today). Lean on visibility, diagnostics, speed, and openness instead.
- "Heart" arguments (no lock-in, open source for the long term) deepen conviction but rarely close on their own — use them to reinforce, not to carry the pitch.

## Proof points

The docs reference customer stories (e.g. Stripe, Foursquare, NVIDIA, Reddit) and supporting stats. **Do not reuse any specific number, quote, or named claim without verifying it against a public, approved source** such as a published case study on fleetdm.com. Many specifics in the positioning doc are internal or unverified. When you need a proof point and can't verify it, ask the user rather than reaching for one from here.
