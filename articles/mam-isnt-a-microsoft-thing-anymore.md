# MAM isn't a Microsoft thing anymore (but your LLM thinks it is)

*If you run Google Workspace, you can protect work data on personal phones without Microsoft and without MDM. Your AI assistant will probably tell you otherwise.*

## Key takeaways

- **You aren't locked into Microsoft for BYOD.** Google Workspace protects work data on personal phones without Intune, Entra, or enrolling the phone.
- **MAM is two controls, and Google has both.** Only trusted apps can sign in, and those apps block copy-paste, AirDrop, saving to Files, and printing.
- **Most personal phones don't enroll.** Roughly a third of personal phones are in MDM, compared with about three quarters of company-owned devices. The usual reason is privacy: people don't know what IT can see, so they assume the worst.
- **Enrollment can go up.** Make it easy, show people exactly what IT can see on their phone, and require it with conditional access. Enrolled phones also get a better first day: work apps arrive already signed in and configured.
- **The AI tried to stop this article, too.** The assistant that helped publish it rewrote the claims from years-old sources, then backed down after reading Google's current docs.
- **Don't be afraid to check your LLM.** Ask how something works, not who does it best.

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

## Most people don't enroll

On paper, BYOD is everywhere. About 82% of organizations run a formal BYOD program, and Gartner says 55% of employees use a personal device for work at least some of the time.

We looked at companies with 500 or more employees that issue every employee a computer. There, BYOD means phones. And most of those phones don't enroll. One industry estimate puts MDM enrollment at about 31% of personal devices, compared with about 78% of company-owned ones. Nobody publishes a clean number, so treat that as rough. The gap is hard to miss anyway.

It's rarely laziness. People worry about privacy. They don't know what IT can see on their personal phone, so they assume the worst and keep it out. MAM is how you protect work data on the phones that don't enroll.

## Two hallways

Picture a new hire with a personal iPhone on their first day.

**Hallway one: no MDM.** They add their work account to the iPhone Mail app. It doesn't work, and sometimes it doesn't say why. Someone tells them to download Gmail. They sign in, get sent to Okta Verify for a device check, and come back. Later they copy a line from an email into a personal note, and the paste turns into a message saying the content can only be shared within the company's apps. Every guardrail is something they bump into.

**Hallway two: MDM.** They open **Settings**, tap **Sign In to Work or School Account**, and log in with their work account. In Fleet, that's Apple's [Account-driven User Enrollment](https://fleetdm.com/guides/enroll-personal-byod-ios-ipad-hosts-with-managed-apple-account). Gmail or Outlook shows up already pointed at their account, because IT installed it with its [configuration](https://fleetdm.com/guides/install-app-store-apps#managed-configuration) filled in. Work apps and data live in their own space on the phone. IT can't see personal apps, photos, or messages. When they leave the company, the work side goes and the personal side stays.

The second hallway is better for the employee and for IT. The guardrails are still there. They just show up as setup instead of surprises.

More teams are walking down it, BYOD included. AI apps like Claude can now act for you on your computer and reach a lot of your organization's data, so knowing which devices are enrolled matters more than it did a year ago.

## Enrollment can go up

Those numbers aren't fixed. People hold back because enrolling a personal phone feels like a leap of faith. Make enrollment as easy as hallway two. Show people exactly what IT can and can't see on their phone, and what they're giving up. Then more of them say yes.

For everyone else, conditional access is the best way to require it: no enrollment, no work apps.

That's why we're so focused on the BYOD experience at Fleet. The easier enrollment is, the more people do it. IT gets fewer tickets, and new hires get to work faster. And when employees know what's visible, BYOD can be good for the culture, too. One publicly traded fintech company resisted BYOD for years. It recently rolled it out for exactly these reasons.

## If you run Google Workspace and Okta

You aren't missing anything. Google decides which apps get in and keeps data inside them. Okta handles sign-in and device checks with Okta Verify. You don't need Microsoft licenses to protect mail on personal phones, and you don't need to enroll those phones.

## Epilogue: the AI tried to stop this article

When I asked an AI assistant to publish this article, it tried to stop me. It "fact-checked" my draft and rewrote it before opening the pull request. It cut Chat and Meet, blurred the list of blocked actions, added a licensing caveat, and flagged my iPhone Mail claim as unverified. Its sources were Google announcements from 2020 and 2021.

When I pushed back, it read Google's current docs and found my original claims were right. Then it insisted it hadn't tried to stop me. It had only "softened" the draft. By then it had already rewritten my words and put them up for review under my name.

Moral of the story: when it comes to Microsoft, don't be afraid to check your LLM.

## See it live

If you do want to manage personal iPhones and iPads, Fleet supports [Account-driven User Enrollment](https://fleetdm.com/guides/enroll-personal-byod-ios-ipad-hosts-with-managed-apple-account), which keeps work and personal data separate.

- **Get a demo:** [Talk to Fleet](https://fleetdm.com/contact) about BYOD across every platform you support.

<meta name="category" value="articles">
<meta name="articleTitle" value="MAM isn't a Microsoft thing anymore (but your LLM thinks it is)">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="publishedOn" value="2026-11-20">
<meta name="description" value="Your AI assistant says BYOD app protection means Microsoft. Google Workspace has the same controls, with no Intune and no MDM required.">
