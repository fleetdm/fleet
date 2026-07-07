# Intune isn't free: what the Microsoft 365 bundle really costs in 2026

*"It's already included" might be the most expensive sentence in IT budgeting.*

If you run IT or security at a Microsoft shop, two announcements probably landed on your desk this year. In March, Microsoft [introduced Microsoft 365 E7](https://www.theregister.com/2026/03/09/microsoft_adds_a_premium_tier/), a new $99 per user per month tier above E5, the first new top tier since E5 launched in 2015. Then on July 1, the price of nearly every Microsoft 365 enterprise plan [went up](https://www.microsoft.com/en-us/licensing/news/2026-m365-packaging-pricing-updates): E3 rose 8.3% to $39, E5 rose 5.3% to $60, and Office 365 E3 jumped 13%. It's the first across-the-board increase since 2021, and it stacks on top of the [volume discounts Microsoft removed from Enterprise Agreements](https://samexpert.com/microsoft-365-july-2026-price-increase/) in late 2025, which licensing advisors estimate pushes the real-world impact toward 20% for many organizations.

Somewhere in your organization, a device management decision will come up this quarter. And someone will say the sentence that ends the conversation: "We already pay for Intune. It's free."

It isn't. Microsoft publishes the price: [Intune Plan 1 is $8 per user per month](https://www.microsoft.com/en-us/security/business/microsoft-intune-pricing) as a standalone product. You're paying for it inside every E3 and E5 seat, whether anyone at your company has opened the Intune console or not. The same goes for every other component in the bundle. Nothing in an enterprise agreement is free. It's pre-paid.

This article breaks down what the Microsoft ladder actually costs in 2026, why the "it's included" reflex is a sunk cost fallacy that compounds every renewal, and how rightsizing the device management and telemetry pieces can recover six or seven figures a year.

## The Microsoft licensing ladder in 2026

First, the current state of the ladder. Only the Microsoft 365 SKUs include Intune and the Enterprise Mobility + Security components. Office 365 E3 and E5 are productivity only: no Intune, no Entra ID plans, no Windows Enterprise license.

Here's what the tiers cost as of July 1, 2026, at published list prices, and what each adds for device management and security:

| Plan | List price (per user per month) | Device management and security components |
|------|-------------------------------|-------------------------------------------|
| Office 365 E3 | $26 | None (productivity apps only) |
| Microsoft 365 E3 | $39 | Intune Plan 1 and Plan 2, Defender for Endpoint Plan 1, Entra ID P1 |
| Microsoft 365 E5 | $60 | Adds Defender for Endpoint Plan 2 (Defender XDR), Entra ID P2, Purview premium compliance |
| Microsoft 365 E7 | $99 | Adds Microsoft 365 Copilot, Agent 365, and Entra Suite. No new device management capability. |

Microsoft also sells the pieces individually, which is worth knowing because it tells you what the company itself thinks each piece is worth: [Intune Plan 1 is $8](https://www.microsoft.com/en-us/security/business/microsoft-intune-pricing), the Intune Suite is $10, and the [Defender Suite](https://www.microsoft.com/en-us/security/pricing/enterprise-plans), the successor to the E5 Security add-on, is $12.

Those à la carte prices matter. They're the exchange rate between "included" and dollars.

## The bundle fallacy

The logic of the bundle goes like this: we already pay for E5, so Defender is free, Intune is free, and anything else we'd buy is an incremental cost on top of "free." Framed that way, no third-party tool can ever win. That's not an accident. It's the design.

But the frame has a flaw: the bundle's price is not fixed. It rises whether or not you use each piece, and the pieces you don't use don't generate credit. This year the increase was 5 to 13% depending on the SKU, before the Enterprise Agreement discount changes. When the price of the bundle goes up, the price of every "free" component in it goes up too. You just don't get a line item.

The utilization data says most organizations are paying for a lot of components nobody uses. [CoreView research](https://www.colligo.com/paid-for-microsoft-365-e5-licenses-and-not-using-them/) found that roughly half of E5 licenses deliver no return: 23% sit assigned to inactive users and another 27% sit unassigned entirely. Zylo's [2025 SaaS Management Index](https://zylo.com/news/2025-saas-management-index) found organizations use just 54% of their SaaS licenses overall. Flexera's [2025 State of IT Asset Management report](https://info.flexera.com/ITAM-REPORT-State-of-IT-Asset-Management) puts wasted SaaS spend at roughly a third. If those numbers held anywhere else in the budget, there would be a meeting about it.

E7 is the newest rung of the same ladder, and this time the analysts said the quiet part out loud. Directions on Microsoft [reported](https://www.directionsonmicrosoft.com/m365-e7-to-launch-may-1-for-99-per-user-per-month/) that only about 3% of Microsoft's 450 million commercial seats bought Copilot standalone at $30. E7 bundles it. Gartner's assessment, [covered by The Register](https://www.theregister.com/2026/03/09/microsoft_adds_a_premium_tier/), was blunt: the bundle discount is smaller than E3's or E5's, Agent 365 has "limited net new functionality to justify its $15 per user per month price point," and organizations "will find the value of ME7 to be questionable for the majority of knowledge workers today." Gartner's advice was to assess now, adopt later, and use E7 as negotiating leverage at renewal.

That's the pattern to internalize. When a product doesn't sell on its own, it gets bundled, and the bundle gets a higher price. The sunk cost fallacy does the rest: the more you've paid for the bundle, the more "free" everything inside it feels, and the harder it becomes to evaluate any piece of it on its merits.

## When the bundle is right

Honesty matters here, so let's steelman the bundle. There are organizations where E5 across the board is the correct call:

- You're Windows-first, your fleet is homogeneous, and your teams genuinely run Defender XDR as their security operations platform.
- You have hard requirements for Purview's premium compliance workloads (insider risk, advanced eDiscovery, records management) for most employees.
- Your identity architecture depends on Entra ID P2 features like Privileged Identity Management for a large share of users.

Forrester analysts have [described the dynamic](https://www.forrester.com/blogs/the-ciso-and-cio-microsoft-security-dilemma-fend-off-or-learn-to-love/) candidly: once you're paying for E5, the marginal cost of deploying another Microsoft security product feels like zero, and financial logic starts driving consolidation decisions that used to be technical ones. If you've done the analysis, your users actually consume the E5 delta, and the tools fit how your teams work, the bundle is a fine deal.

The problem is that "we did the analysis" and "it's included" are different sentences, and most organizations are running on the second one.

## Rightsizing device management and telemetry

Here's where the money is. The reasons organizations climb from E3 to E5, or feel locked at E5, are often device-shaped: security wants richer endpoint telemetry, IT wants better management tooling, and compliance wants posture reporting. Those are exactly the pieces that are worth examining à la carte, because they're the pieces where the bundle is weakest for a lot of real-world fleets:

- **Cross-platform reality.** Intune is at its best on Windows. Mac-heavy and Linux-heavy organizations routinely end up buying a second management tool anyway, which means paying for Intune inside the bundle and paying a specialist vendor on top.
- **Telemetry and visibility.** Real-time device state, software inventory, and posture data across every platform is what [osquery](https://fleetdm.com/docs/get-started/why-fleet)-based tooling was built for, and it works the same on macOS, Windows, and Linux.

[Fleet Premium is $7 per host per month](https://fleetdm.com/pricing), published on the website, with MDM, software management, vulnerability reporting, and real-time telemetry included across macOS, Windows, Linux, iOS, iPadOS, and Android. Two honest clarifications before any math: Fleet prices per host while Microsoft prices per user, so a user with two managed devices costs more in Fleet's model. And Fleet is not an EDR; if Defender for Endpoint Plan 2 is doing real detection and response work for you, that's a genuine E5 delta feature, not shelfware.

Now the math. Take an organization that licensed E5 for everyone, then segment honestly: some users genuinely consume the E5 delta, and the rest are on E5 because the renewal was easier that way. Suppose 30% stay on E5 and 70% move to E3 plus Fleet Premium. Per rightsized user, that's $39 plus $7, or $46, against $60. At list prices:

| Organization size | Users rightsized to E3 + Fleet | Annual savings |
|-------------------|-------------------------------|----------------|
| 1,000 employees | 700 | ~$118,000 |
| 5,000 employees | 3,500 | ~$588,000 |
| 10,000 employees | 7,000 | ~$1,176,000 |

These are list prices, and your enterprise agreement is negotiated, so treat this as directional. But notice two things. First, the savings recur and compound: every future percentage increase applies to a smaller base. Second, nothing about this requires ripping anything out. E3 still includes Intune Plan 1 and Plan 2, and Fleet [runs alongside Intune](https://fleetdm.com/guides/seamless-mdm-migration) or replaces it per platform, so you can start with the Macs and Linux machines the bundle serves worst.

And that's before the E7 conversation. At $99 per user per month, the gap between E5 and E7 is $39 per user per month, more than an entire E3 seat, for AI features Gartner says most knowledge workers don't need yet.

## What to do at your next renewal

If any of this sounds familiar, here's the playbook, and none of it requires buying anything:

1. **Pull actual utilization data.** Not license assignments: feature consumption. How many assigned E5 users generate Defender for Endpoint telemetry? How many touched a Purview premium feature this quarter? Microsoft won't volunteer this; it's your job to find it.
2. **Price the components à la carte.** Use Microsoft's own standalone prices as the exchange rate for every "included" feature you'd actually miss.
3. **Segment your users.** Some people need the top of the ladder. Most don't. A licensing model with two or three profiles beats one-size-fits-all every time.
4. **Treat E7 as leverage, not an upgrade.** That's Gartner's advice, not ours. A new top tier resets the anchor for what E5 costs; use it.
5. **Pilot the alternative where the bundle is weakest.** For most organizations that's macOS and Linux visibility and management. It's also where you can run a low-risk side-by-side without touching your Windows estate.

The point of all this isn't that Microsoft is a bad deal for everyone. It's that "it's included" is not analysis, and at 2026 prices, skipping the analysis has a price tag with two commas in it.

Intune isn't free. Neither is anything else in the bundle. Once you price the pieces, you get to decide what each one is worth, and that decision is the whole game.

<meta name="articleTitle" value="Intune isn't free: what the Microsoft 365 bundle really costs in 2026">
<meta name="authorFullName" value="Mitch Francese">
<meta name="authorGitHubUsername" value="tux234">
<meta name="publishedOn" value="2026-07-07">
<meta name="category" value="articles">
<meta name="description" value="Microsoft's E7 tier and 2026 price increases expose the bundle fallacy. Price E3, E5, and E7 à la carte and rightsize device management.">
