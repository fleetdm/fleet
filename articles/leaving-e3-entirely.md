# Leaving E3 entirely

Remember your first PC? Maybe it was a beige tower with a Pentium III and Encarta on CD, or something you built yourself with a screwdriver and a forum thread. Computers were exciting, and they felt like yours.

These days you keep everyone else's computers running, and renewal dates have replaced the excitement. This post is about getting some of it back.

Our [previous post](https://fleetdm.com/articles/intune-isnt-free-what-the-microsoft-365-bundle-really-costs) covered stepping down from E5 to E3. This time we priced what it costs a 1,000-person company to leave Microsoft 365 E3 entirely. We expected to argue that independence is worth a premium. Over three years, that's not how the numbers came out.

## The bundle tax

If you manage Windows today, some of this probably sounds familiar:

- A configuration fails, and the only clue is an error code you have to look up.
- You wait hours to find out whether a profile applied.
- You can't tell whether a report is current.
- Devices don't behave the way you set them up.
- Leaving feels out of reach, because profiles don't export and switching tools sounds like wiping every PC.

Meanwhile, the price keeps rising. On July 1, Microsoft 365 E3 went up to $39 per user per month. That seat includes Intune, Entra ID P1, Windows Enterprise, and Defender, whether you use them or not.

What keeps most teams stuck is Windows setup. Zero touch has always run through Autopilot, which requires Entra, so even companies on Okta or Google keep paying Microsoft for identity just to set up laptops. For 1,000 people, Entra ID P1 alone costs $252,000 over three years. Bill Gates shouldn't decide what software you buy just because you like his operating system.

## What changes with Fleet

Fleet removes the Windows setup problem and gives you the choice back:

- Zero touch, $0 means new PCs arrive already managed. When you order, forward an email from Fleet to Dell, Lenovo, or HP, and they install Fleet before the device ships. You don't need Autopilot, Entra, Intune, or USB sticks, and the same approach works for Linux.
- AI-native endpoint engineering replaces the guesswork. Every setting in Fleet is a file, and every feature has an API, so you can ask for a change like "get our Windows PCs to CIS Level 2," review it, and approve it. You see what's actually on each device right now, not what it reported hours ago.
- You choose your own tools. Pick the identity provider, productivity suite, and EDR that suit your company. Fleet manages Windows, Mac, and Linux in one place.

## What our example company wanted

Picture a 1,000-person company with 650 Windows PCs, 300 Macs, and 50 Linux machines. It pays for Microsoft 365 E3 for everyone, plus Jamf for the Macs. It wasn't trying to save money. It wanted to choose its own tools and was ready to pay a premium to do it. Here's what it picked:

- Okta takes over identity from Entra, for Windows setup and every app.
- Google Workspace covers almost everyone. About 50 people in finance and leadership keep desktop Excel and PowerPoint through Microsoft 365 Apps for enterprise.
- Fleet replaces both Intune and Jamf, so every Mac, PC, and Linux machine is managed in one place.
- New computers follow a simple rule: MacBook Pros for engineers, MacBook Neos for almost everyone else, and Windows or Linux PCs for anyone who prefers them. New PCs ship from Lenovo with Fleet already installed, and Fleet pays the setup fee.
- Windows Enterprise and Defender for Endpoint Plan 1 stay, bought on their own instead of inside E3. Defender also replaces the Mac security tool that came with Jamf.

## The twist: it gets cheaper

Then they added it up. At today's mix of computers, their new setup costs $534,500 a year, compared with $513,000 for Microsoft 365 E3 and Jamf. That's the premium they expected, about 4%, mostly from buying Windows Enterprise for 650 PCs. The surprise came three years out: the premium is gone by year one, and by year three they save $181,500 a year.

Here's what each piece costs at list price, per user per month unless noted:

| Component | List price | Annual cost |
| --- | --- | --- |
| Microsoft 365 E3 (today) | $39 | $468,000 |
| Jamf for Mac (today) | $12.50 per Mac | $45,000 |
| Fleet Premium | $7 per host | $84,000 |
| Okta Workforce Identity Starter (SSO, MFA, directory) | $6 | $72,000 |
| Windows Enterprise E3, 650 PC users | about $7.60 | $59,280 |
| Defender for Endpoint Plan 1 (what E3 includes) | $3 | $36,000 |
| Office 365 E3 | $26 | $312,000 |
| Microsoft 365 Apps for enterprise + Exchange Online Plan 2 | $12 + $8 | $240,000 |
| Google Workspace Enterprise Standard, plus Microsoft 365 Apps for 50 people | about $23, plus $12 | $283,200 |

Their setup isn't the only way out. Each option below replaces Intune and Jamf with Fleet and Entra with Okta, and keeps Windows Enterprise and Defender. At today's mix of computers, here's how they compare:

| Setup | Annual cost | Compared with today |
| --- | --- | --- |
| Today: Microsoft 365 E3 + Jamf | $513,000 | — |
| Their pick: Okta + Google Workspace, Excel and PowerPoint for 50 | $534,500 | +$21,500 (+4%) |
| Like-for-like: Okta + Office 365 E3 | $563,300 | +$50,300 (+10%) |
| Lowest cost: Okta + Microsoft 365 Apps + Exchange Online | $491,300 | −$21,700 (−4%) |

A few things to note:

- The lowest-cost option is cheaper from day one. It saves $21,700 a year while keeping desktop Office apps, Windows Enterprise, and Defender. It drops SharePoint, Teams, and some of Office 365 E3's compliance tools, so check those first.
- The like-for-like option keeps the same apps and compliance tools for about 10% more.
- Windows Enterprise is the biggest new cost, at $59,280 a year for 650 PCs. Existing hybrid-joined PCs keep activating it the way they do today, and new PCs get a volume license key, which Fleet can apply. PCs that don't need it can stay on Windows Pro, which comes with the hardware.
- Zero touch, $0 matters in every option. Keeping Autopilot means keeping Entra ID P1, which would add $84,000 a year to each one.

These are list prices, so treat the totals as estimates. The one-time cost of the move is covered below.

Prices: [Fleet](https://fleetdm.com/pricing), [Microsoft 365 July 2026 pricing](https://fleetdm.com/articles/intune-isnt-free-what-the-microsoft-365-bundle-really-costs), [Okta](https://www.okta.com/pricing), [Jamf](https://www.spotsaas.com/blog/jamf-pricing), [Defender for Endpoint](https://costbench.com/software/endpoint-security/microsoft-defender-endpoint), [Windows Enterprise](https://ifeeltech.com/windows-11-security-guide), [Google Workspace Enterprise](https://www.redresscompliance.com/google-workspace-pricing-enterprise-guide-2026.html).

## The next three years

The premium disappears as the company grows. Over three years it hires 200, then 400, then 350 people, and its computers shift toward Macs.

Every new computer, for a new hire or a refresh, follows the buying rule above: 25% MacBook Pros, 60% MacBook Neos, 12% Windows PCs, and 3% Linux machines. The [MacBook Neo](https://www.macworld.com/article/219909/mac-buying-guide.html) starts at $699, less than most business laptops. PCs are replaced every four years and Macs every five, since Fleet's experience scores for each hardware model show the Macs lasting longer. Here's how the fleet and the software bill change:

|  | Today | Year 1 | Year 2 | Year 3 |
| --- | --- | --- | --- | --- |
| Employees | 1,000 | 1,200 | 1,600 | 1,950 |
| Macs | 300 | 609 | 1,088 | 1,524 |
| Windows PCs | 650 | 540 | 454 | 362 |
| Linux machines | 50 | 51 | 58 | 64 |
| Staying on Microsoft 365 E3 and Jamf | $513,000 | $653,000 | $912,000 | $1,141,200 |
| Staying on Microsoft 365 E3, with Fleet for Macs | $493,200 | $612,800 | $840,200 | $1,040,600 |
| Their setup with Fleet | $534,500 | $619,500 | $801,700 | $959,700 |
| Savings vs. E3 and Jamf | −$21,500 | $33,500 | $110,300 | $181,500 |
| Savings vs. E3 with Fleet for Macs | −$41,300 | −$6,700 | $38,500 | $80,900 |

That's about $325,000 saved over three years, and most of it comes from Macs. In their setup, a Mac user costs $468 a year for Google Workspace, Okta, Defender, and Fleet, compared with $618 for Microsoft 365 E3 plus Jamf. A Windows user costs a little more, mainly because of Windows Enterprise, but 85% of new computers are Macs.

A tougher comparison is keeping E3 and moving only the Macs to Fleet, which saves money from day one. Against that, leaving E3 costs a little more in year one, pulls ahead in year two, and is about $81,000 a year ahead by year three.

The table also leaves out support. IBM [reported](https://www.networkworld.com/article/944412/windows-centric-ibm-changes-its-tune-on-mac-deployments.html) that 5% of its Mac users called the help desk, compared with 40% of PC users, and that each Mac saved at least $265 over four years. By year three, this company has about 940 more Macs than it would with today's mix, worth roughly $62,000 a year at IBM's rate.

The model assumes one computer per employee and the same computers in every scenario. About 5% of staff keep desktop Excel and PowerPoint, and new Macs set themselves up for free through Apple Business Manager.

## How they got there

The move takes six to nine months. Evaluation and purchase take 8 to 12 weeks, including a 30-day proof of value on a few dozen devices. Then identity moves to Okta and devices move to Fleet in batches, with no PC wiped. For most employees, the only visible change is a new sign-in page.

The main one-time cost is overlap: Fleet and Okta start while E3 is still paid for, at about $13,000 a month at list prices. Signing a few months before the Microsoft renewal keeps that to roughly $50,000 to $80,000, plus the team's time. [Fast Track](https://fleetdm.com/fast-track) shortens both. Fleet experts work on site with the team for three to five days and leave behind a working deployment, tested enrollment, and a plan for the rest of the move.

### Other things to plan for

- Active Directory keeps running during the move. Existing PCs stay hybrid-joined, new PCs skip the domain, Group Policy moves into Fleet, and the domain controllers retire as old PCs are replaced. Microsoft 365 E3 includes [Windows Server CAL rights](https://www.directionsonmicrosoft.com/sites/default/files/PDFs_public/dom-licensing-reference-set-sample_windows_server.pdf) that Office 365 E3 and Google Workspace don't, so budget for CALs until AD is gone. Everyone also keeps an account in the free Microsoft tenant for Defender and Microsoft 365 Apps licenses, which Okta creates automatically.
- Configuration Manager still runs alongside Intune on about 100 lab and shop-floor PCs, as it does at 62% of the organizations in [Recast's 2026 survey](https://itdaily.com/news/software/state-of-intune-recast-2026/). Its [license rights](https://learn.microsoft.com/configmgr/core/understand/product-and-licensing-faq) end with E3, so those PCs move to Fleet too, which adds a week or two. Servers covered by separate System Center licenses can stay.
- Personal devices finally get enrolled. Few employees put Intune on their phones, but Fleet shows them exactly what it collects and keeps work data separate, so more people sign up. Okta uses Fleet's device checks when deciding who gets in, so only devices that pass them reach company apps.

Afterward, the IT team's work looks different. Instead of waiting on sync cycles and looking up error codes, they ask Fleet for what they need and review the change before it goes out. The two engineers who led the move now run the company's device strategy, and their project comes up whenever leadership talks about what IT delivered that year.

## Who's already doing this

Companies rarely publish stories about cutting Microsoft licenses, since it can complicate their next renewal. But every piece of this plan is already running at scale.

One of the world's top medical technology companies runs thousands of Windows PCs with Fleet, Okta, and Lenovo Ready to Provision instead of Autopilot.

New hires get their laptop at home on the Friday before they start, and IT activates their Okta account the same day. Setup takes three screens: a name, a local password, and Fleet sign-in with Okta. Fleet then installs the standard apps. On a good connection, the whole process takes 20 to 30 minutes, and most of that is Windows updates.

When a laptop comes back, a disposition partner holds it for 60 days, wipes it, and then restocks, recycles, or resells it depending on its age. Their IT team calls the setup straightforward and says they're happy with it.

Other Fleet customers show what the rest of the plan looks like in practice:

- [Abridge](https://fleetdm.com/case-study/abridge), a healthcare AI company, built zero-touch onboarding with Fleet's support team. Setting up an engineer's laptop used to take a whole day, and now it takes about 45 minutes. Abridge manages more than 700 devices as code.
- [Stripe](https://fleetdm.com/case-study/stripe) moved 10,000 Macs to Fleet in 10 days and saves hundreds of thousands of dollars a year in licensing.
- A systems support specialist at another Fleet customer put it simply: "I'm spending about 20 hours a week less on managing PCs than I was with Intune."

## FAQ

**Do I need Autopilot to use Fleet?** No. Autopilot requires Entra, so it adds a Microsoft license you may not otherwise need. If you already have Entra and want to keep Autopilot, Fleet fully supports it.

**What Microsoft license does zero-touch setup need?** None. Forward the email from Fleet to Dell, Lenovo, or HP.

**Intune comes with E3. Why pay for Fleet?** You'll save money. Switching from Microsoft 365 E3 to Office 365 E3 saves about $13 per user per month, almost double Fleet's price. You also choose your own identity provider and EDR instead of maintaining Entra and Intune for the bundle's sake.

**How do employees sign in to Windows without Entra?** Your identity provider handles it. Okta, Google, and Duo all offer Windows sign-in tools.

**Can I migrate from Intune without wiping devices?** Yes. Fleet's automatic Windows migration moves PCs from Intune or another MDM without wiping them, and end users don't have to do anything.

**Which versions of Windows does Fleet support?** Windows 10 and 11 get full management, including enrollment, profiles, BitLocker, software, and OS updates. Windows Server gets visibility, vulnerability reporting, and scripts. Windows on ARM is supported too.

**Does Fleet replace my EDR?** No. Fleet manages and checks your devices, and it works alongside the EDR you choose, like Defender, CrowdStrike, or SentinelOne.

## Don't settle

You don't have to stay on a bundle because it's the default. Price the pieces, choose the ones you want, and let Fleet connect them. Few IT projects have such a clear before and after: a smaller bill, faster first days for new hires, and a team known for moving the company forward.

And you're never locked in again. Fleet supports Entra and Autopilot, so you can go back anytime. It works with any EDR, so if you later switch from Defender to CrowdStrike, Fleet rolls out the new agent to every computer.

You won't be doing it alone, either. Fleet's support team helps with the whole E3 move, including questions about Okta, Google Workspace, Defender, and Active Directory that go beyond Fleet itself. Stripe credits its Fleet customer success manager with a key role in its rollout.

Remember when computers were exciting? They can be again.

[Get a demo](https://fleetdm.com/contact?talkToUs) · [See Fleet's pricing](https://fleetdm.com/pricing)

---

*Fine print: the zero touch, $0 offer.* Available to new customers who purchase Fleet Premium, or start a proof of value for Fleet Premium, to fully replace Intune on at least 500 PCs (Windows, Linux, or both) before December 31, 2026. For the term of the Fleet Premium subscription covering those PCs, Fleet pays the one-time fee for Dell Ready Image, Lenovo Ready to Provision, or the equivalent service from HP, Acer, and other manufacturers on any new PCs you buy, up to the average discounted, negotiated price Fleet sees among customers of similar size. To redeem, forward the email you received from Fleet to Dell, Lenovo, or your hardware maker. Other prices in this post are November 2026 list prices.

<meta name="articleTitle" value="Leaving E3 entirely">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-11-03">
<meta name="description" value="What it costs a 1,000-person company to leave Microsoft 365 E3 for Fleet, Okta, and Google Workspace, modeled over three years.">
