# Which AI model works best for generating configuration profiles?

If you've ever stared at a blank `.mobileconfig` file or tried to hand-write a Windows CSP from scratch, you know the pain. The schema is fussy, Apple's docs are scattered across half a dozen pages, and Microsoft's ADMX backing makes you feel like you need a second monitor just to keep tabs open. So naturally, many admins (myself included) have started leaning on AI for the first pass.

The real question is: which model actually produces a profile that works on the first try?

I gave a handful of recent frontier models the same set of prompts. A macOS profile to enforce FileVault with deferral options, a Windows CSP to disable the Edge first-run experience, and an ADMX-backed PowerShell logging policy. Same prompt each time, no follow-ups. Here's what shook out.

**Claude Opus 4.6** was the most reliable across the board. It nailed the macOS plist structure, including the right `PayloadType` strings and UUID handling, and didn't invent keys that don't exist. On the Windows side, it actually understood the difference between `<Add>` and `<Replace>` verbs and used CDATA correctly for ADMX-backed settings. When I asked it to explain why it picked a particular `LocURI`, the answer lined up with Microsoft's docs.

**GPT-5** came in a close second. The macOS profile was clean and validated through `plutil` without complaint. The Windows CSP, however, kept defaulting to integer formats when it should've used `chr`, and twice it produced a key that looked plausible but isn't in the Policy CSP. Fixable with a follow-up prompt, but not what you want if you're cranking out profiles at scale.

**Gemini 2.5 Pro** is the dark horse. Its long context window means you can drop an entire `.admx` file in and ask it to derive a working profile, which the others don't handle quite as gracefully. The downside is that it likes to over-explain, and the XML it ships with sometimes carries extra whitespace or stray comments that break stricter parsers. Strip those out, and the underlying profile is solid.

If you're picking one for daily use, Opus wins on accuracy, Gemini wins when you're working straight from raw vendor documentation, and GPT-5 is the fastest to iterate with once you already know what you want.

One thing worth saying, no matter which model you use: validate the output before pushing to production. Run `plutil -lint` for macOS, and for Windows, deploy to a single test host and check the `DeviceManagement-Enterprise-Diagnostics-Provider` event log. AI is great for scaffolding, but it doesn't know your fleet, your test ring, or your rollback plan. That part is still on you.

Once you have a profile you trust, Fleet handles the rest. Custom profiles work across macOS, Windows, iOS, and Android from one place. Drop the file in, scope it to a team or label, and ship it.  

<meta name="articleTitle" value="Which AI model works best for generating configuration profiles?">
<meta name="authorFullName" value="Harry	Ravazzolo">
<meta name="authorGitHubUsername" value="harrisonravazzolo">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-05-20">
<meta name="description" value="Compare Claude Opus 4.6, GPT-5, and Gemini 2.5 Pro for generating macOS .mobileconfig and Windows CSP profiles.">
