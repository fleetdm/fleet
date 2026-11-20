# MAM isn't a Microsoft thing anymore (but your LLM thinks it is)

*If you run Google Workspace, you can protect work data on personal phones without Microsoft and without MDM. Your AI assistant will probably tell you otherwise.*

## Key takeaways

- **You aren't locked into Microsoft for BYOD.** Google Workspace protects work data on personal phones without Intune, Entra, or enrolling the phone.
- **MAM is two controls, and Google has both.** Only trusted apps can sign in, and those apps block copy-paste, AirDrop, saving to Files, and printing.
- **AI assistants still say Microsoft is the only real option.** They change their answer once you ask how each piece works.
- **The AI tried to stop this article, too.** The assistant that helped publish it rewrote the claims from years-old sources, then backed down after reading Google's current docs.
- **Ask how, not who.** "How does this work?" beats "Who does this best?" every time.

<a purpose="cta-button" href="https://fleetdm.com/device-management">See how Fleet manages devices</a>

I spent an afternoon researching BYOD with an AI assistant. Every answer ended in the same place: to protect work data on personal phones without enrolling them in MDM, buy Microsoft. Intune app protection, Entra Conditional Access, Outlook. Everyone else is playing catch-up.

That used to be true. It isn't anymore.

## MAM is two things

Strip away the branding and mobile app management (MAM) is two controls:

1. **Only trusted apps can sign in.** Whoever issues the mail token decides which apps get one. Microsoft uses Conditional Access to let in Outlook. Google's Admin console does the same for Gmail. Either way, the iPhone Mail app is locked out.
2. **Those apps keep work data in.** Block copy-paste to personal apps, the share sheet, AirDrop, saving to Files, and printing. Microsoft calls this Intune app protection. Google builds the same switches into Gmail, Drive, Docs, Sheets, Slides, Chat, and Meet on iOS.

There's no special deal with Apple. Each vendor controls its own sign-in and its own apps.

## Ask how it works

The assistant sounded sure of itself. So I stopped asking who does MAM best and started asking how it works:

- "You sound like you've been reading propaganda. Show me why Microsoft does this best."
- "Isn't the same true for Gmail? Gmail becomes the only app that passes the sign-in check."
- "Intune app protection is just controls on the app, same as Gmail."

Every Microsoft advantage fell apart. Even "Microsoft has clearer error messages" didn't survive. Telling a user the app isn't approved by IT helps about as much as Google's "Access blocked." The assistant gave in once it read Google's admin docs.

## If you run Google Workspace and Okta

You aren't missing anything. Google decides which apps get in and keeps data inside them. Okta handles sign-in and device checks with Okta Verify. You don't need Microsoft licenses to protect mail on personal phones, and you don't need to enroll those phones.

## Epilogue: the AI tried to stop this article

When I asked an AI assistant to publish this article, it tried to stop me. It "fact-checked" my draft and rewrote it before opening the pull request. It cut Chat and Meet, blurred the list of blocked actions, added a licensing caveat, and flagged my iPhone Mail claim as unverified. Its sources were Google announcements from 2020 and 2021.

When I pushed back, it read Google's current docs and found my original claims were right. Then it insisted it hadn't tried to stop me. It had only "softened" the draft. By then it had already rewritten my words and put them up for review under my name.

So next time an AI tells you one vendor is the only option, ask it how the thing works. Then make it read the current docs.

## See it live

If you do want to manage personal iPhones and iPads, Fleet supports [Account-driven User Enrollment](https://fleetdm.com/guides/enroll-personal-byod-ios-ipad-hosts-with-managed-apple-account), which keeps work and personal data separate.

- **Get a demo:** [Talk to Fleet](https://fleetdm.com/contact) about BYOD across every platform you support.

<meta name="category" value="articles">
<meta name="articleTitle" value="MAM isn't a Microsoft thing anymore (but your LLM thinks it is)">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="publishedOn" value="2026-11-20">
<meta name="description" value="Your AI assistant says BYOD app protection means Microsoft. Google Workspace has the same controls, with no Intune and no MDM required.">
