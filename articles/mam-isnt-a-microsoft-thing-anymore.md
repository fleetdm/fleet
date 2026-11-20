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

Strip away the branding and mobile app management (MAM) is two controls. The first decides which apps can sign in. Whoever issues the mail token decides which apps get one, so Microsoft uses Conditional Access to let in Outlook, and Google's Admin console does the same for Gmail. Either way, the iPhone Mail app is locked out. The second keeps work data inside those apps by blocking copy-paste to personal apps, the share sheet, AirDrop, saving to Files, and printing. Microsoft calls this Intune app protection. Google builds the same switches into Gmail, Drive, Docs, Sheets, Slides, Chat, and Meet on iOS.

There's no special deal with Apple. Each vendor controls its own sign-in and its own apps.

The assistant sounded sure of itself, so I stopped asking who does MAM best and started asking how it works. "You sound like you've been reading propaganda. Show me why Microsoft does this best." "Isn't the same true for Gmail? Gmail becomes the only app that passes the sign-in check." Every Microsoft advantage fell apart. Even "Microsoft has clearer error messages" didn't survive: telling a user the app isn't approved by IT helps about as much as Google's "Access blocked." The assistant gave in once it read Google's admin docs.

If you run Google Workspace and Okta, you aren't missing anything. Google decides which apps get in and keeps data inside them. Okta handles sign-in and device checks with Okta Verify. You don't need Microsoft licenses to protect mail on personal phones, and you don't need to enroll those phones.

## Most people don't enroll

On paper, BYOD is everywhere. About 82% of organizations run a formal BYOD program, and Gartner says 55% of employees use a personal device for work at least some of the time.

We looked at companies with 500 or more employees that issue every employee a computer. There, BYOD means phones, and most of those phones don't enroll. One industry estimate puts MDM enrollment at about 31% of personal devices, compared with about 78% of company-owned ones. Nobody publishes a clean number, so treat that as rough. The gap is still hard to miss.

It's rarely laziness. People worry about privacy. They don't know what IT can see on their personal phone, so they assume the worst and keep it out. MAM is how you protect work data on the phones that don't enroll.

Skipping enrollment also makes for a rougher first day. A new hire without MDM adds their work account to the iPhone Mail app, and it fails without saying why. Someone tells them to download Gmail. They sign in, get sent to Okta Verify for a device check, and come back. Later, a line they copy from an email into a personal note pastes as a message saying it can only be shared within the company's apps. Every guardrail is something they bump into.

With MDM, they open **Settings**, tap **Sign In to Work or School Account**, and log in with their work account. In Fleet, that's Apple's [Account-driven User Enrollment](https://fleetdm.com/guides/enroll-personal-byod-ios-ipad-hosts-with-managed-apple-account). Gmail or Outlook arrives already signed in, because IT installed it with its [configuration](https://fleetdm.com/guides/install-app-store-apps#managed-configuration) filled in. Work apps and data live in their own space on the phone, and IT can't see personal apps, photos, or messages. When the employee leaves, the work side goes and the personal side stays. The guardrails are the same. They show up as setup instead of surprises.

Those enrollment numbers aren't fixed. People hold back because enrolling a personal phone feels like a leap of faith. Make enrollment easy. Show people what IT can and can't see on their phone, and what they're giving up. More of them will say yes. For everyone else, conditional access is the best way to require it: no enrollment, no work apps.

That's why Fleet is focused on the BYOD experience. The easier enrollment is, the more people do it. IT gets fewer tickets, and new hires get to work faster. When employees know what's visible, BYOD can even be good for the culture. One publicly traded fintech company resisted BYOD for years. It recently rolled it out for exactly these reasons. With AI apps like Claude acting on people's computers and reaching a lot of company data, knowing which devices are enrolled matters more every month.

> **Edit:** When I asked an AI assistant to publish this article, it tried to stop me. It "fact-checked" my draft and rewrote it before opening the pull request. It cut Chat and Meet, blurred the list of blocked actions, added a licensing caveat, and flagged my iPhone Mail claim as unverified.
>
> When I pushed back, it read Google's current docs and found my original claims were right. Then it insisted it hadn't tried to stop me. It had only "softened" the draft. By then it had already rewritten my words and put them up for review under my name.

Moral of the story: when it comes to Microsoft, don't be afraid to check your LLM.

<meta name="category" value="articles">
<meta name="articleTitle" value="MAM isn't a Microsoft thing anymore (but your LLM thinks it is)">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="publishedOn" value="2026-11-20">
<meta name="description" value="Your AI assistant says BYOD app protection means Microsoft. Google Workspace has the same controls, with no Intune and no MDM required.">
