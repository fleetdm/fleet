# MAM isn't a Microsoft thing anymore (but your LLM still thinks it is)

*An AI assistant tried to stop me from publishing this article. It was wrong, and so is the idea that protecting work data on personal phones means buying Microsoft.*

## Key takeaways

- **Google now protects work apps the way Microsoft does.** Since 2022, Gmail, Drive, Docs, and Google's other iOS apps have the same kind of data protection as Intune app protection, with no enrollment required.
- **BYOD enrollment has been low, but that's changing.** Historically, only about 31% of personal phones enroll in MDM. Easier enrollment, clear privacy, and conditional access are pushing that number up.
- **AI is a big reason why.** A publicly traded fintech company that resisted BYOD for years just started a program so tools like Claude can work on employees' devices and reach company data, with access gated by conditional access and an enrolled device.

## The MAM, the myth, the legend

When I asked an AI assistant to publish this article, it tried to stop me. It "fact-checked" my draft and quietly rewrote it before opening the pull request. It cut Chat and Meet from my list of Google apps, watered down the rest, and flagged my claims as unverified. I made it go read Google's current docs. It found I was right and put everything back. Then it insisted it hadn't tried to stop me.

That was the second time that day. Earlier, I'd spent an afternoon researching BYOD with an AI assistant, and every answer ended up in the same place. If you want to protect work data on personal phones without enrolling them in MDM, buy Microsoft: Intune app protection, Entra Conditional Access, Outlook. Everyone else is playing catch-up.

That used to be true. It isn't anymore.

> **What is MAM?** It can be hard to see through the marketing obfuscation, but here’s my attempt to explain in simple terms: “Mobile app management” (MAM) is just a way to keep work data inside work apps: it blocks copy-paste, saving, and sharing out of apps like Outlook or Gmail. Like any data loss prevention, it never fully works, because anyone can take a picture of the screen with another phone, or write down what they saw with a pencil. You can do roughly the same thing by enrolling the phone in your MDM.
> 
> Under the hood, it comes down to two controls. First, you decide which apps can sign in. Whoever issues the mail token gets to say which apps receive one. Microsoft uses Conditional Access to let Outlook in, and Google's Admin console does the same for Gmail. Either way, the built-in iPhone Mail app is out. Second, those apps keep work data in. They can block copy-paste into personal apps, the share sheet, AirDrop, saving to Files, and printing. Microsoft calls this Intune app protection. Google has the same switches in Gmail, Drive, Docs, Sheets, Slides, Chat, and Meet on iOS.

The assistant sounded very sure of itself, so I stopped asking who does MAM best and started asking how it works:

> "You sound like you've been reading propaganda. Show me why Microsoft does this best."
>
> "Isn't the same true for Gmail? Gmail becomes the only app that passes the sign-in check."

One by one, the Microsoft advantages fell apart. The assistant gave in once it read Google's admin docs.

## Nothing but the truth

You may have heard that app protection without enrollment only works with Intune. That's false, and it has nothing to do with managing devices. Outlook and the Office apps have some lightweight data controls built in, and Microsoft calls them MAM. Google has the same controls in Gmail and its other work apps, and so does any other office suite. There's no secret deal with Apple behind any of it.

Google lets you turn these controls on in the Google Workspace Admin console, alongside the rest of your Gmail and Drive settings.   

The difference with Outlook and Office apps is that Microsoft _makes you set this up in Intune_, not in its Outlook email settings or the settings for your other Office apps.  Why make it harder to find and configure this stuff?  Because by putting these settings in Intune, you need to buy both of Microsoft’s products, which makes the bundle harder to leave.

Thankfully, now there’s another way.  You don’t need Intune, and Microsoft can’t force you to use Intune since they don’t control any major mobile operating system (though not for [lack of trying](https://en.wikipedia.org/wiki/Windows_Phone)). If you actually need to stop files and copied text from moving out of work apps, you can enroll the phone in your MDM, like Fleet or Jamf.

This works for _every_ app IT installs, Outlook and Gmail included, and it uses  Apple's and Android’s standard device management settings, an open standard any MDM can use. Once you're using them, you aren't tied to the limits of one company's office suite and its device management tool.

It's also worth asking how much this matters to you in practice. In my conversations with companies, most that have the license for MAM don't lean on it much. Some turned it on because it came in their Microsoft bundle, and some never turned it on at all.

Bottom line: If you run Google Workspace/Nextcloud, or Okta/Ping, or Fleet/Jamf, you aren't missing anything. Your office suite decides which apps get in and keeps the data inside them. Your identity provider handles sign-in and device checks with Okta Verify. You don't need Microsoft licenses to protect mail on personal phones, and for many companies you may not even need to enroll those phones in an MDM at all.

That said, there are plenty of other security-driven (not just DLP) reasons to enroll employees’ personal smartphones in an MDM, everything from conditional access, to preventing keyloggers by encouraging patching, to lost mode, and more.

But let’s talk about the other factors, not just security.

## Enrolling employee smartphones speeds up time to productivity
Skipping MDM enrollment makes for a rougher first day. For example, when new hires try to add their work account to the iPhone Mail app, it fails without saying why. They create a ticket or ask their manager, and someone tells them to download Gmail or Outlook. They sign in, get bounced to Okta Verify or Entra for a device check, and come back. 

Compare that with modern MDM, on iOS for example, where an employee simply opens **Settings**, taps **Sign In to Work or School Account**, and logs in with their standard SSO credentials. (Apple calls this [Account-driven User Enrollment](https://support.apple.com/guide/deployment/dep23db2037d/web).)  Then, Gmail or Outlook magically shows up already configured and partitioned, because IT set it up ahead of time.

This way, work apps and data live in their own space on the phone, and IT can't see personal apps, photos, or messages. When the employee leaves, the work side goes away and the personal side stays.

![Enrolling a personal iPhone with a work account keeps work and personal data separate](../website/assets/images/articles/apple-account-driven-user-enrollment-800x400@2x.png)

Android is just as quick. The employee opens Fleet's enrollment page, taps **Enroll**, and Android sets up a work profile that keeps work apps separate from personal ones.

![Fleet's enrollment page for a personal Android phone](../website/assets/images/articles/android-byod-enroll-page-206x200@2x.png)

No-brainer, right?

## Most people don't enroll

On paper, BYOD is everywhere. About 82% of organizations have a formal BYOD program, and Gartner says 55% of employees use a personal device for work at least some of the time.

In practice, it's mostly unmanaged. We looked at companies with 500 or more employees that give everyone a work computer. There, BYOD means phones, and most of those phones don't enroll. One industry estimate puts MDM enrollment at about 31% of personal devices. Nobody publishes a clean number, so take that with a grain of salt.

It's not laziness. People don't know what IT can see on their personal phone, so they assume the worst. I don't blame them. I wouldn't enroll my phone in an MDM that isn't open source either, because I couldn't tell what it reads. My text messages? My browser history? What [can it see](https://fleetdm.com/transparency)? At a lot of companies, the answer is 🤷.

The result is high friction, low adoption, and a culture where even executives are reticent to give up their privacy and risk [their kids’ pictures getting randomly wiped by Intune](https://www.bankinfosecurity.com/microsoft-intune-mdm-gains-notoriety-after-stryker-hack-a-31084).

## BYOD enrollment is increasing because of AI

But this is changing, and AI is speeding it up. For example, one publicly traded fintech company held out against BYOD for years, but recently started a program because it wants tools like Claude to work on employees' devices and reach company data, with conditional access and an enrolled device deciding how much each one gets.  This is an increasingly common story among other mid-sized and large Fleet customers, and as of late 2026, the trend is accelerating.

Enrolling a personal phone shouldn’t require a leap of faith. Employees should understand plainly what IT can and can't see, and what it can and can’t do to their personal data. Otherwise people rightly will assume the worst.

(This is why we work so hard to put scope transparency and employee transparency first at Fleet. Fleet is open source, so anyone can read exactly what it collects, and employees can ask an AI assistant what IT can see on their device and find out in seconds, with blistering accuracy. We're also [working on audit transparency](https://github.com/fleetdm/fleet/issues/57438), which makes it clear when data on a device is accessed while keeping the controls an organization needs.  Feedback welcome!)

When enrollment is easy, more people enroll, IT gets fewer tickets, and new hires get to work faster. When people know what IT can see, BYOD can even be good for the culture.

## No M’AM?
Whether or not you run a BYOD program for employee phones today, and whether or not you lock down email and office apps, you are free.  You don’t need Intune, E3, Microsoft, or any other particular vendor to provide enterprise-grade security, provisioning, or even data-loss prevention, whether you have 100,000 employees or 100, and whether or not you are in a regulated industry.  Plenty of other organizations do this today with or without Microsoft, with or without Intune, and with or without MAM, and it’s easy to set up.  (If you could use a hand, [drop us a line](https://fleetdm.com/support).)

In the meantime, remember to keep squinting until you see the truth.  And when it comes to Microsoft… don't be afraid to check your AI’s work.

_**Thanks** to [Deelan Sabido](https://www.linkedin.com/in/deelansabido/), [Allen Houchins](https://www.linkedin.com/in/allenhouchins/), [Kitzy](https://www.linkedin.com/in/kitzy), and [Mike Thomas](https://www.linkedin.com/in/mike-thomas-52277938) for reading drafts of this essay._

<meta name="category" value="articles">
<meta name="articleTitle" value="MAM isn't a Microsoft thing anymore (but your LLM still thinks it is)">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="publishedOn" value="2026-11-20">
<meta name="description" value="Your AI assistant says BYOD app protection means Microsoft. Google Workspace has the same controls, with no Intune and no MDM required.">
