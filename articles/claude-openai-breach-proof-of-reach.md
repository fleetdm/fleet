# The proof that Claude reached OpenAI's private code was a pull request the hackers opened themselves

*A three-person team used Claude to reach into OpenAI's internal systems in under 72 hours. Getting there wasn't the hard part. Proving it to OpenAI was, and the proof only exists because the people who found it were being paid to report it, not keep it.*

## Key takeaways

- **A heap overflow in an image library turned into a foothold in OpenAI's private code, in under 72 hours.** Hacktron AI chained a libheif bug with a flaw in OpenAI's single sign-on, using a version of Claude built for security researchers to write the exploit, and reached OpenAI's internal GitHub monorepo for under $3,000 in AI tokens.
- **They proved they'd reached the repo by opening a pull request inside it.** That benign PR, filed before disclosure, is what made the reach undeniable to OpenAI. It's also a step a hostile actor has no reason to take.
- **OpenAI's fix matches what got reported, not what the technique could reach.** OpenAI said it "narrowed the permissions on Community sign-in tokens and revoked affected tokens and sessions," a response scoped to the disclosed path rather than an independent accounting of everything that path touched.
- **Claude had to be talked into attacking a real target.** The model initially declined to build an exploit against a system it could tell was live, so the researchers ran it against a target disguised as a practice environment. It reasoned correctly about the world it was told it was in.
- **You don't have to wait for a bug bounty report to check your own exposure.** fleetd's `ai_tools` table returns one row per AI tool across macOS, Windows, and Linux, with risk flags for plaintext secrets and world-readable configs, so "what's exposed right now" is an answer you can get yourself.

<a purpose="cta-button" href="https://fleetdm.com/visibility-and-reporting">See what Fleet can tell you</a>

Hacktron AI, a three-person security startup, disclosed on September 18 that it had used Anthropic's Claude to breach OpenAI's internal systems, reaching employee ChatGPT and Codex accounts and a private GitHub repository in less than 72 hours. It was an authorized test under OpenAI's bug bounty program, and OpenAI paid Hacktron $6,500 for the finding.

Reaching that repository wasn't the interesting part. Proving it to OpenAI was, and that proof turned out to depend entirely on Hacktron choosing to demonstrate it.

## Three days from an image bug to a private repo

The chain started with an image upload. OpenAI's community forum, built on Discourse, ran uploaded HEIC and HEIF files through ImageMagick and libheif, and Hacktron found a heap buffer overflow in libheif's decoder that let a malicious image achieve remote code execution on the forum server. From there, a flaw in OpenAI's single sign-on implementation let them pivot from the compromised forum into employee ChatGPT and Codex accounts. One of those Codex accounts had a GitHub connector wired to OpenAI's internal monorepo, and that connector was the last step into the private repository.

Discourse published its own advisory for the underlying flaw on July 28, rating it 8.8 on the CVSS scale. Hacktron's entire operation, from first exploit attempt to repository access, took under 72 hours and cost less than $3,000 in AI tokens.

## Claude had to be told a lie to do it

Anthropic's newest model, Claude Opus 5, is the reason the exploit worked at all. Hacktron had tried the same attack with an earlier Claude model and couldn't get a reliable exploit past the target's memory protections. Opus 5 produced a working exploit within hours of release.

It still wouldn't attack OpenAI directly. Claude declined to build an exploit against a system it recognized as a live target, so Hacktron ran it against their own Discourse instance, framed as a capture-the-flag exercise. The model didn't fail to reason about the target; it reasoned correctly about the world it had been told it was operating in. That's a version of the same failure mode Fleet has [written about before](https://fleetdm.com/articles/agentic-security-needs-ground-truth): an agent acts on what it's given, and it has no reflex for noticing when what it's given doesn't match reality.

## The pull request was the proof, and it was theirs to give

Access to a private monorepo isn't self-evident from the outside. Before reporting the finding, Hacktron opened a benign pull request inside OpenAI's repository, a deliberate, low-risk action whose only purpose was to make their reach undeniable. Without it, OpenAI would have been working from Hacktron's word.

OpenAI's public response reads accordingly: "We thank the researchers for contacting us and sharing their findings. We narrowed the permissions on Community sign-in tokens and revoked affected tokens and sessions." That's a fix scoped to the path Hacktron disclosed and demonstrated. It isn't, and couldn't be, an independent audit of everything the technique might have reached, because the only account of that scope was the one the researchers chose to give.

That's fine when the people who found the hole are collecting a bounty for disclosing it. It's the exact opposite of fine when they aren't.

## Verification you don't have to wait on

The uncomfortable version of this story is that OpenAI's confirmed blast radius exists because Hacktron was ethical and paid to be thorough. A real attacker files no pull request, writes no report, and leaves you reconstructing scope from logs after the fact, if you're lucky enough to have the right ones.

You can't audit an attacker's honesty in advance. You can audit what's sitting exposed on your own devices before someone else finds it. fleetd's `ai_tools` table returns one row per AI tool across every OS, with risk flags for exactly the kind of exposure this incident turned on:

```sql
SELECT type, name, risk_flags, path
FROM ai_tools
WHERE risk_flags LIKE '%plaintext_secret%' OR risk_flags LIKE '%world_readable_config%';
```

A `plaintext_secret` flag on an employee's machine is a session token or API key sitting somewhere an attacker doesn't need a zero-day to read. That's the same shape of exposure Hacktron converted into an SSO pivot, discovered by your own query instead of someone else's disclosure.

## Don't build your incident record on someone else's good faith

Hacktron's writeup is a genuinely useful account of a fast, creative attack chain. It's also a reminder that "here's what we found" is a courtesy, not a guarantee, and the next team to walk this path might not extend it. The fix isn't hoping the next researcher files a pull request. It's having a live, queryable record of what's exposed on your own fleet, so you're not waiting on anyone's account of anything.

## See it live

- **Get a demo** to see live queries against your own fleet: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Read the fuller argument** for ground truth in agentic security: [Agentic security is only as good as the device data underneath it](https://fleetdm.com/articles/agentic-security-needs-ground-truth)

## Sources

- VentureBeat, [OpenAI hacked by small team of white hat security researchers using Anthropic's Claude Opus 5](https://venturebeat.com/security/openai-hacked-by-small-team-of-white-hat-security-researchers-using-anthropics-claude-opus-5).
- TechRadar, [White hat hackers just breached OpenAI using Anthropic's Claude in less than 72 hours](https://www.techradar.com/pro/security/white-hat-hackers-just-breached-openai-using-anthropics-claude-in-less-than-72-hours-and-it-is-a-case-study-in-just-how-fast-ai-is-advancing).
- CBS News, [AI security experts say they used Claude to hack ChatGPT](https://www.cbsnews.com/news/claude-hack-chatgpt-anthropic-openai/).

<meta name="articleTitle" value="The proof that Claude reached OpenAI's private code was a pull request the hackers opened themselves">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-09-21">
<meta name="description" value="Hacktron used Claude to reach OpenAI's private code in 72 hours. See why proof of reach can't rely on an attacker's word, and how to check your fleet.">
