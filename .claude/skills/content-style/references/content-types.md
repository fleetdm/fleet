# Content types and format discipline

Each content type has its own shape. Identify the type first, then apply the matching format. All types share the voice and mechanics in `style-rules.md`.

## Website and UI copy

- Benefit-focused: lead with what the reader can do, not what Fleet is.
- Scannable: short sentences, bullet points, clear headers.
- Concise: remove every unnecessary word. No hype, no filler adjectives.
- Direct, specific calls to action.
- Product UI text: sentence case for labels and buttons, plain and short. Mirror the terminology already used in the UI.

## Guides and tutorials

- Explain the "why" of the task before the steps.
- Use numbered lists for sequential actions; one action per step.
- Imperative mood, active voice.
- Bold UI elements only (e.g. "Navigate to **Settings > Hosts**"). Never bold for emphasis.
- Surface the simple, high-level steps first; put advanced details lower down.
- Requires the article endmatter below with `category` set to `guides`.
- For the full section-by-section skeleton (prerequisites, inline gotcha callouts, step shapes, verify/troubleshoot, a fill-in template) and an audit checklist for existing guides, use the `guide-structure` skill.

## Articles and blog posts

- Conversational yet professional. Provide context — the "why," not just the "what."
- Explain jargon for the reader.
- Use `##` (H2) and `###` (H3) to break up long sections.
- Stay aligned with Fleet values; no vendor-pitch language.
- Requires the article endmatter below.

## Announcements and release notes

- Lead with the news in the first sentence.
- Keep it short (roughly 2-4 sentences) and factual. No hype, no build-up.
- Requires the article endmatter below with `category` set to `announcements`.

## Docs reference pages

- Static-noun headings ("Log destinations"), not task verbs.
- Precise and complete; favor tables and lists over prose where they're clearer.
- Full URLs for links so pages stay movable.

## Marketing and sales enablement

- Same honesty and anti-hype discipline as everything else — the audience is technical and tunes out spin.
- Read `references/positioning.md` for the audience pains, the "implicate the pain" narrative, Fleet's differentiators, and messaging do's and don'ts. Lead with pain then desire; keep operational speed as the throughline; never lift unverified stats, quotes, or named claims from the source docs.
- Follow the competitor and Fleet framing rules in `style-rules.md`: state facts, name specific differentiators, never editorialize.

## Article endmatter template

Articles, guides, and announcements end with this YAML block. Match `articleTitle` to the H1 exactly. Don't fabricate the author, username, or date — ask if you don't have them.

```
<meta name="articleTitle" value="[Must match the H1 exactly]">
<meta name="authorFullName" value="[author name]">
<meta name="authorGitHubUsername" value="[github username]">
<meta name="publishedOn" value="[YYYY-MM-DD]">
<meta name="category" value="[releases|security|engineering|case study|announcements|guides|podcasts|comparison|whitepaper|articles]">
<meta name="description" value="[1-2 sentences. 150 chars max. Factual, benefit-driven, no filler.]">
```
