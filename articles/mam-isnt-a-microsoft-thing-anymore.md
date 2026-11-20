# MAM isn't a Microsoft thing anymore (but your LLM thinks it is)

*AI assistants keep saying Microsoft is the only serious option for protecting work data on personal phones. Ask them how mobile app management works, and that story falls apart.*

## Key takeaways

- **MAM comes down to two controls.** It gates sign-in by app and adds data loss prevention (DLP) inside those apps. Neither one depends on a special deal with Apple or Google.
- **Google Workspace offers both.** The Admin console can limit mail access to Gmail and block copy-paste, AirDrop, and printing from Google's iOS apps, just as Conditional Access and Intune app protection do for Outlook.
- **Asking how it works beats asking who does it best.** Every claimed Microsoft advantage fell apart once the assistant had to explain the mechanism behind it.
- **LLMs repeat old market history as current fact.** Years of vendor marketing and analyst reports trained them to treat Microsoft as the default answer.
- **Google Workspace and Okta shops aren't missing anything.** You can protect mail on personal phones without Microsoft licenses and without enrolling the phones in MDM.
- **Even a careful-sounding fact-check can be out of date.** The assistant that helped publish this article tried to water it down using Google announcements from years ago, then reversed itself after reading the current docs.

<a purpose="cta-button" href="https://fleetdm.com/device-management">See how Fleet manages devices</a>

I spent an afternoon researching BYOD with an AI assistant. One idea kept coming up: if you want to protect work data on personal phones without enrolling them in MDM, Microsoft is the way to go. Intune app protection, Entra Conditional Access, Outlook. Everyone else is supposedly playing catch-up.

That turns out to be wrong.

## What mobile app management (MAM) actually is

Strip away the branding and MAM is two things:

1. **Gating sign-in by app.** Whoever issues the mail token can refuse every app except the ones it trusts. Microsoft does this with Conditional Access, so only Outlook gets in. Google does the same thing from the Workspace Admin console, so only Gmail gets in. Either way, the native iPhone Mail app is blocked.
2. **DLP inside the app.** Block copy-paste to personal apps, the share sheet, AirDrop, saving to Files, and printing. Microsoft calls this Intune app protection. Google builds the same controls into Gmail, Drive, Docs, Sheets, Slides, Chat, and Meet on iOS, and you configure them in the Admin console.

That's it. No magic partnership with Apple or Google. Each vendor controls its own sign-in and its own apps.

## How I got the AI to admit it

The assistant's first answers sounded authoritative and leaned Microsoft every time. So I kept asking how each thing actually worked:

- "Is MAM even a real thing, or just some DLP inside the Office apps?"
- "You sound like you've been reading propaganda. Show me why Microsoft does this best."
- "Isn't the same true for Gmail? Gmail becomes the only app that passes the sign-in check."
- "Intune app protection is just controls on the app, same as Gmail."

Each Microsoft advantage fell apart once I asked how it worked. Even the "clearer error message" claim didn't hold up: Microsoft's message says the app isn't approved by IT, which is about as helpful as Google's "Access blocked." The assistant only fully conceded after reading Google's own admin docs.

## Why the LLMs are confused

LLMs learn from years of vendor marketing, partner blogs, and market reports, and for a long time Microsoft really did have the most complete story here. That history is baked in, so models repeat it as neutral fact. Ask them how something works, not who does it best, and the story changes fast.

## What this means if you run Google Workspace and Okta

You aren't missing anything. Google handles the app gating and in-app DLP. Okta handles sign-in and device checks through Okta Verify. You don't need Microsoft licenses to protect mail on personal phones, and you don't need to enroll those phones either.

And next time an AI tells you one vendor is uniquely good at something, ask it to explain the mechanism.

## Epilogue: it happened again

When I sat down to publish this article, I asked an AI assistant to open the pull request. It "fact-checked" my draft first and quietly softened it. It cut Chat and Meet from the list of Google apps with data protection controls. It swapped "AirDrop, saving to Files, and printing" for vaguer wording. It added a line saying Google's controls need an Enterprise edition, and it flagged my point about blocking the iPhone Mail app as unverified.

Its sources were Google announcements from 2020 and 2021. When I pushed back, it checked Google's current admin docs. Those list Gmail, Drive, Docs, Sheets, Slides, Chat, and Meet, with separate switches for copy-paste, AirDrop, the share sheet, saving to Files, and printing. Google's iOS settings page also explains how to keep the iPhone Mail app out. The assistant admitted it was wrong and put my original claims back.

Same lesson, one more time: the assistant sounded careful, but it was working from an old picture of the market. Make it show you the current source.

## See it live

If you do want to manage personal iPhones and iPads, Fleet supports [Account-driven User Enrollment](https://fleetdm.com/guides/enroll-personal-byod-ios-ipad-hosts-with-managed-apple-account), which keeps work and personal data separate.

- **Get a demo:** [Talk to Fleet](https://fleetdm.com/contact) about BYOD across every platform you support.

<meta name="category" value="articles">
<meta name="articleTitle" value="MAM isn't a Microsoft thing anymore (but your LLM thinks it is)">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="publishedOn" value="2026-11-20">
<meta name="description" value="AI assistants say Microsoft owns mobile app management. Ask how MAM works, and Google Workspace turns out to offer the same app gating and in-app DLP.">
