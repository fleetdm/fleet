# Stripe moved 10,000 Macs to Fleet, saving hundreds of thousands annually

## Why Stripe needed a change

Stripe manages about 10,000 Macs and 5,000 Chromebooks for employees and support partners. All employee Macs are provisioned automatically with Apple Automated Device Enrollment, and users authenticate into the device during onboarding. Chromebooks have a lower risk profile and are used by support partners.

Stripe has a strong focus on security and automation. Their previous MDM relied on manual, UI-driven workflows, which didn’t fit their “infrastructure as code” approach. Fragmented APIs made it hard to automate even simple tasks, and the team had to build workarounds to fill gaps.

New Apple OS releases introduce features, commands, and profiles. Stripe wanted more visibility and control than their MDM provided. Support for the latest OS only arrived after public release, and they couldn’t inject raw payloads or inspect device responses.

Slow vendor support made these issues worse. Stripe often waited a long time for ticket responses, delaying resolutions. Lower-impact bugs sometimes went unfixed.

<div purpose="attribution-quote">

*The ability to get heard [with our previous MDM] was just impossible.*

**Wes Whetstone**

Staff Client Platform Engineer
</div>

## The search for a solution

When their previous MDM came up for renewal, Stripe decided to look for alternatives. They evaluated several MDMs against a clear set of requirements.

First, any replacement had to support all critical use cases for Stripe’s Macs. Employee Macs are “business only,” and third-party software must be approved as “business critical” before installation. Stripe’s security teams also use MDM profiles as part of their device-trust decisions. Getting core MDM functionality right was essential.

Next, the MDM had to fit Stripe’s automation workflows. The team focuses on infrastructure-as-code and wanted the MDM to integrate into existing workflows, not introduce another isolated tool. Stripe also prefers open-source tools because they offer more visibility into how the product works.

Finally, the MDM had to support on-premise deployment. Stripe didn’t consider cloud-only MDMs because they view MDM as critical infrastructure. With control over 10,000 Macs, they wanted full control over who could access the system.

## Choosing Fleet

During evaluation, Stripe needed to confirm that Fleet could handle their core MDM use cases, including Automated Device Enrollment and deploying security profiles.

Fleet met those requirements, but the open-source model gave Stripe additional confidence. Being able to see how profiles were built and deployed made them comfortable managing Macs with Fleet.

<div purpose="attribution-quote">

*The openness of the whole management stack … is more valuable to us.*

**Wes Whetstone**

Staff Client Platform Engineer
</div>

When issues arise, Stripe can also inspect what went wrong themselves. Instead of uploading logs and waiting for a vendor response, they can typically identify where issues originate, which leads to faster resolutions.

Fleet’s API-first design let Stripe automate their entire MDM workflow. They migrated 10,000 Macs in 10 days, and custom profiles allowed them to set up Macs in video conferencing rooms with true zero-touch deployment — something their previous MDM could not support.

<div purpose="attribution-quote">

*We saved a couple of hundred thousand dollars a year.*

**Wes Whetstone**

Staff Client Platform Engineer
</div>

## The results

Using Fleet as the MDM for Macs gives Stripe’s security and IT teams more control over a critical piece of infrastructure. It has removed manual work and allowed the team to build custom solutions, such as enabling remote screen sharing in meeting rooms without requiring someone on site. Core MDM capabilities are now a pillar of Stripe’s zero-trust device attestation.

Stripe is enthusiastic about Fleet’s support, stating that “[Fleet’s] customer success manager played a key role in the successful migration and rollout.”

Stripe also plans to evaluate the capabilities offered by Fleet’s open-source MDM for other platforms they manage, including Windows VMs and BYOD programs.

<div purpose="attribution-quote">

*Fleet’s Customer Success Manager for Stripe is the best!*

**Wes Whetstone**

Staff Client Platform Engineer
</div>

<meta name="category" value="case study">
<meta name="articleTitle" value="Stripe moved 10,000 Macs to Fleet, saving hundreds of thousands annually">


<meta name="publishedOn" value="2025-12-11">
<meta name="authorGitHubUsername" value="n/a">
<meta name="authorFullName" value="Fleetdm">

<meta name="companyLogoFilename" value="stripe-logo-96x40@2x.png">
<meta name="quoteAuthorImageFilename" value="wes-whetstone-120x120@2x.png">
<meta name="quoteAuthorName" value="Wes Whetstone">
<meta name="quoteAuthorJobTitle" value="Staff Client Platform Engineer">
<meta name="quoteContent" value="“The ability to get heard [with our previous MDM] was just impossible.”">

<meta name="companyName" value="Stripe">
<meta name="companyInfo" value="Stripe is a financial technology company that provides a developer-friendly platform for businesses to accept and manage online payments. It offers a broad set of financial infrastructure products, including tools for subscription management, fraud prevention, business incorporation, and issuing physical and virtual cards.">

<meta name="summaryChallenge" value="As a leading financial services provider, Stripe wanted more automation, transparency, and flexibility from its MDM. Their previous tool relied on manual processes, and slow support left the team dissatisfied.">
<meta name="summarySolution" value="Stripe already used Fleet for device visibility with osquery. Fleet’s open-source, automation-first approach gave Stripe the transparency they wanted and let them replace manual tasks with automated scripts. The option to deploy Fleet on-prem also set it apart from other MDMs, many of which are cloud-first or cloud-only.">
<meta name="summaryKeyResults" value="Seamless migration of 10,000 Macs in 10 days.; Saved “hundreds of thousands” each year in licensing costs.; Gained the ability to deploy custom profiles for full control, including setting up Macs in Zoom Rooms without user interaction.">

