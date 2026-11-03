# Your AI assistant says you can't unbundle Microsoft. Check the premise.

*An AI assistant told me Windows device management needed an $84,000 a year Entra add-on. It was describing one path as if it were the only one.*

## Key takeaways

- **AI assistants state a premise as fact, then do the math on it.** The arithmetic can be perfect while the starting assumption is wrong. Check the assumption first.
- **Entra ID P1 is a requirement of Microsoft's enrollment path, not of Windows device management.** Entra automatic enrollment and Autopilot need it. Devices that get Fleet's agent from the factory or from a provisioning package don't use that path.
- **A one-line correction changed the answer.** Once the assistant was reminded how zero-touch enrollment works without Entra, it dropped the cost it had added and said splitting the bundle was cheaper, even at list price.
- **Ask what requires each cost.** "What requires this, and is there a way around it?" is a better prompt than "is this possible?"
- **Ask for list prices, then verify them.** Have the assistant show its prices and its date, then check them against Microsoft's current price list and your own agreement before you act.

<a purpose="cta-button" href="https://fleetdm.com/windows">See Windows zero-touch setup</a>

I was working through an unbundling question with an AI assistant: what does it cost to move a 1,000-person company off the Microsoft 365 bundle's device management and onto something else? The assistant gave a confident answer. The answer was wrong, and the way it was wrong is common enough that it's worth walking through.

If you're a buyer, a finance partner, or an IT leader using an assistant to sanity-check a Microsoft renewal, this is for you.

## What the assistant said

The assistant said this about Windows enrollment:

> "Windows automatic MDM enrollment through Entra, including Autopilot, needs Entra P1. That adds $7 per user, about $84K a year. Without it, you're limited to manual or provisioning-package enrollment."

That was for 1,000 users. At $7 per user per month, $84,000 a year is the right multiplication. It then used that $84,000 to argue that leaving the bundle would not save much.

## Where the premise breaks

The first half of that statement is accurate, and Fleet's own docs say so. Connecting Fleet to Entra for automatic enrollment or Autopilot requires a [Microsoft Entra ID P1 license for each end user](https://fleetdm.com/guides/windows-mdm-setup#microsoft-licenses-you-need). That's Microsoft's enrollment path, and Microsoft prices it that way.

The second half is the problem. "Without it, you're limited to manual or provisioning-package enrollment" treats every other option as a downgrade, and leaves out the factory path.

I told the assistant it was wrong. Fleet customers run thousands of Windows PCs where the computer maker installs Fleet's agent before the device ships, using Dell ReadyImage or Lenovo RTP. The device enrolls the first time it's online. There's no Entra automatic enrollment and no Autopilot in that flow. Fleet describes the same approach on its [Windows page](https://fleetdm.com/windows): forward Fleet's order email to Dell, Lenovo, or HP, and they install Fleet before the device ships.

## What happened when I pushed back

The assistant agreed:

> "Dell ReadyImage and Lenovo RTP put Fleet on the device at the factory, so it enrolls zero-touch without Entra auto-enrollment or Autopilot. That means Entra P1 isn't needed... Entra P1 is only needed if a customer chooses to use Microsoft's own enrollment path, and Fleet customers don't have to."

Then it redid the comparison. These are the assistant's figures, not mine. They are for 1,000 users, at annual list prices as the assistant understood them in July 2026.

| Scenario | Microsoft 365 bundle | Unbundled | Difference |
| --- | --- | --- | --- |
| Office apps and Windows Enterprise only | M365 E3 (no Teams): $365,400 | Microsoft 365 Apps + Windows E3: $259,560 | $105,840 |
| Same, against E3 with Teams | M365 E3: $468,000 | Microsoft 365 Apps + Windows E3: $259,560 | $208,440 |
| With Exchange and SharePoint, no Teams | M365 E3 (no Teams): $365,400 | Office 365 E3 (no Teams) + Windows E3: $300,960 | $64,440 |
| With Exchange, SharePoint, and Teams | M365 E3: $468,000 | Office 365 E3 + Windows E3: $336,300 | $131,700 |

Its summary was that even if the company needs Microsoft email, splitting the bundle still saves about $64,000 to $132,000 a year, because the extra cost it had added was Entra P1, which isn't needed.

Please don't take these numbers from me. I haven't verified Microsoft's prices, and prices and packaging change. Check them against [Microsoft's current price list](https://www.microsoft.com/en-us/licensing/news/2026-m365-packaging-pricing-updates) and your own agreement. The table also leaves out what you'd spend on a device management tool to replace the one in the bundle. [Fleet's pricing](https://fleetdm.com/pricing) is published, so add your own number before you compare.

## Why the assistant got it wrong

It wasn't making things up. It described Microsoft's enrollment path accurately, then treated that path as the only one. My guess is that Microsoft's own enrollment path is the one it reaches for first, because it's the one most documented.

It also stopped when it found a cost. Once "Entra P1, $84K" was in the answer, it was a fact in the conversation, and the rest of the reasoning was built on it. Nothing prompted it to ask whether the cost was avoidable.

When I reminded it how factory enrollment works, it behaved. It agreed unbundling was possible, and cheaper, even at list price. The knowledge was there. It just wasn't the default.

## What to do

Check the premise, not only the arithmetic. If an assistant tells you a cost is required, ask three things:

1. **What requires this cost?** Make it name the product, the feature, and the license, so you can check it.
2. **Is there a way around it?** Ask for the alternatives, including ones that don't go through the vendor's own tooling.
3. **Show me list prices and the date.** Have it list every price it used and where it got it, then verify them yourself.

Tell it what you know, too. If you've seen a setup work in practice, say so and ask it to redo the numbers. In this case, one correction was enough.

Zero-touch enrollment without Entra does have trade-offs. Devices that enroll this way aren't joined to Entra ID, and some identity-driven flows work differently. Fleet's [Autopilot without Autopilot](https://fleetdm.com/articles/autopilot-without-autopilot) article covers them. If you already pay for Entra ID P1 through Microsoft 365 E3 or E5, the license savings don't apply to you.

## The assumption is the thing to audit

An assistant that's wrong with confidence is more expensive than one that's unsure, because it ends the conversation. A bundle works the same way: once "required" is in the answer, nobody looks for the exit. Make the assistant, and the vendor, show what requires each line item.

## See it live

- Read how it works: [Autopilot without Autopilot](https://fleetdm.com/articles/autopilot-without-autopilot)
- Compare the numbers: [Intune isn't free](https://fleetdm.com/articles/intune-isnt-free-what-the-microsoft-365-bundle-really-costs)
- **Get a demo:** [talk to Fleet](https://fleetdm.com/contact) about Windows zero-touch setup.

<meta name="articleTitle" value="Your AI assistant says you can't unbundle Microsoft. Check the premise.">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-11-03">
<meta name="description" value="AI assistants often say Windows device management forces an $84K Entra add-on. Check the premise before you accept the cost of unbundling Microsoft.">
