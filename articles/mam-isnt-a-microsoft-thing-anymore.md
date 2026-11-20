# MAM isn't a Microsoft thing anymore (but your LLM thinks it is)

*An AI assistant tried to stop me from publishing this article. It was wrong, and so is the idea that protecting work data on personal phones means buying Microsoft.*

## Key takeaways

- **You don't need Microsoft to protect work data on personal phones.** Google Workspace can do it too, without enrolling the phone.
- **MAM is two things.** Only approved apps can sign in, and those apps keep work data out of personal ones. Microsoft and Google both do both.
- **Most personal phones don't enroll.** About a third do. People worry about what IT can see, and nobody tells them.
- **That can change.** Make enrollment easy, be honest about what IT can see, and use conditional access to require it.
- **Check your LLM.** Mine was confidently wrong about this twice in one day.

When I asked an AI assistant to publish this article, it tried to stop me. It "fact-checked" my draft and quietly rewrote it before opening the pull request. It cut Chat and Meet from my list of Google apps, watered down the rest, and flagged my claims as unverified. I made it go read Google's current docs. It found I was right and put everything back. Then it insisted it hadn't tried to stop me.

That was the second time that day. Earlier, I'd spent an afternoon researching BYOD with an AI assistant, and every answer ended up in the same place. If you want to protect work data on personal phones without enrolling them in MDM, buy Microsoft: Intune app protection, Entra Conditional Access, Outlook. Everyone else is playing catch-up.

That used to be true. It isn't anymore.

Here's what mobile app management (MAM) comes down to once you strip off the branding. First, you decide which apps can sign in. Whoever issues the mail token gets to say which apps receive one. Microsoft uses Conditional Access to let Outlook in, and Google's Admin console does the same for Gmail. Either way, the built-in iPhone Mail app is out. Second, those apps keep work data in. They can block copy-paste into personal apps, the share sheet, AirDrop, saving to Files, and printing. Microsoft calls this Intune app protection. Google has the same switches in Gmail, Drive, Docs, Sheets, Slides, Chat, and Meet on iOS.

Microsoft got there first, and Intune app protection works well. But there's no secret deal with Apple behind it. Each vendor controls its own sign-in and its own apps.

The assistant sounded very sure of itself, so I stopped asking who does MAM best and started asking how it works. "You sound like you've been reading propaganda. Show me why Microsoft does this best." "Isn't the same true for Gmail? Gmail becomes the only app that passes the sign-in check." One by one, the Microsoft advantages fell apart. Even "Microsoft has clearer error messages" didn't hold up. A message saying the app isn't approved by IT helps a user about as much as Google's "Access blocked." The assistant gave in once it read Google's admin docs.

So if you run Google Workspace and Okta, you aren't missing anything. Google decides which apps get in and keeps the data inside them. Okta handles sign-in and device checks with Okta Verify. You don't need Microsoft licenses to protect mail on personal phones, and you don't need to enroll those phones.

## Most people don't enroll

BYOD is everywhere on paper. About 82% of organizations have a formal BYOD program, and Gartner says 55% of employees use a personal device for work at least some of the time.

We looked at companies with 500 or more employees that give everyone a work computer. There, BYOD means phones, and most of those phones don't enroll. One industry estimate puts MDM enrollment at about 31% of personal devices, compared with about 78% of company-owned ones. Nobody publishes a clean number, so take that with a grain of salt.

It's not laziness. People don't know what IT can see on their personal phone, so they assume the worst. I can't blame them.

Skipping enrollment also makes for a rougher first day. A new hire without MDM adds their work account to the iPhone Mail app, and it fails without saying why. Someone tells them to download Gmail. They sign in, get bounced to Okta Verify for a device check, and come back. Later, they copy a line from an email into a personal note, and it pastes as a message saying it can only be shared within the company's apps.

With MDM, they open **Settings**, tap **Sign In to Work or School Account**, and log in with their work account. (Apple calls this [Account-driven User Enrollment](https://support.apple.com/guide/deployment/dep23db2037d/web).) Gmail or Outlook shows up already signed in, because IT set it up ahead of time. Work apps and data live in their own space on the phone, and IT can't see personal apps, photos, or messages. When the employee leaves, the work side goes away and the personal side stays.

Those enrollment numbers aren't fixed. Enrolling a personal phone feels like a leap of faith. Make enrollment easy. Tell people plainly what IT can and can't see, and what they're giving up. More of them will say yes. For the rest, conditional access is the best way to require it: no enrollment, no work apps.

When enrollment is easy, more people enroll, IT gets fewer tickets, and new hires get to work faster. When people know what IT can see, BYOD can even be good for the culture. One publicly traded fintech company held out against BYOD for years and recently rolled it out for exactly these reasons. With AI apps like Claude now acting on people's computers and reaching a lot of company data, knowing which devices are enrolled matters more every month.

> **That's why at Fleet, we're working on BYOD that puts employee privacy first.** Fleet is open source, so anyone can read exactly what it collects. Employees can even ask an AI assistant what IT can see on their device. We call that [scope transparency](https://fleetdm.com/transparency). We're also working on audit transparency: making it clear when data on a device is accessed, with the controls an organization needs.

Moral of the story: when it comes to Microsoft, don't be afraid to check your LLM.

<meta name="category" value="articles">
<meta name="articleTitle" value="MAM isn't a Microsoft thing anymore (but your LLM thinks it is)">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="publishedOn" value="2026-11-20">
<meta name="description" value="Your AI assistant says BYOD app protection means Microsoft. Google Workspace has the same controls, with no Intune and no MDM required.">
