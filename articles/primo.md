# How Primo built an HR-driven IT platform, powered by Fleet


## The challenge: IT orchestration can’t run through a console

Primo is the all-in-one IT platform that brings identity, device management, SaaS administration, and asset lifecycle together in one place. The team acts as the “IT cloud in a box” for more than 400 modern companies, managing 30,000 active devices across customer environments that range from five-person startups to organizations of 500-plus employees.

Primo’s core insight runs against the standard enterprise IT architecture. Most platforms treat the identity provider as the source of truth for IT policy. Primo treats the HR system as the source of truth - BambooHR, Gusto, and the other modern HR platforms that have become the system of record for most SMBs over the last decade. When an employee is hired, promoted, transferred, or leaves, the entire IT lifecycle follows automatically from the HR data, on the dates the HR system already knows.

<div purpose="attribution-quote">

*Our core philosophy is that HR data should be the ultimate driver of IT policy. We want a platform where onboarding or offboarding takes two to three minutes instead of three to four hours.*

**Martin Pannier**

Founder and CEO, Primo

</div>

That model only works if every layer underneath it runs through code. Onboarding cannot mean three to four hours of someone in IT clicking through MDM, IDP, SaaS, and procurement consoles. It has to take three minutes, and it has to be repeatable across hundreds of customers without adding headcount. For Primo, a clickops MDM was not a feature problem. It was an existential one.

## Why Fleet: multi-OS, open-source, self-hosted, API-first

Primo evaluated device management platforms against a single test: can we build on top of it the way we build software? Fleet was the only option that answered yes.

<div purpose="attribution-quote">

*A few things make Fleet unique: multi-OS support, open-source nature, self-hosted capabilities, and an API-first approach. That combination is what makes the platform we built possible.*

**Martin Pannier**

Founder and CEO, Primo

</div>

Fleet’s API-first architecture lets Primo run its own orchestration layer in front of every customer environment. Device configuration, policies, software, scheduled wipes and locks - all of it flows through Primo’s internal tooling, which drives Fleet programmatically. The MDM becomes part of the same orchestration sequence that touches IDP, SaaS, and procurement. When HR flags an employee for offboarding three weeks out, Primo can schedule the entire sequence, including the device wipe, to fire automatically on Friday at 5:00 PM, with no one in IT needing to touch a console.

Fleet’s self-hostable model gave Primo control over how it isolates each customer’s data, policies, and configuration. Cross-platform coverage came from a single binary and one API across macOS, Windows, and Linux. And Fleet’s open-source code gave Primo the confidence to build on top of a foundation it could audit, extend, and contribute back to - a non-negotiable for a company whose customers depend on Primo for their entire IT operation.

<div purpose="attribution-quote">

*This seamless experience simply would not be possible without Fleet. You could try to cobble it together with cloud-only MDM providers, but Fleet’s open-source nature, API-first approach, and self-hosting capabilities make it a uniquely powerful engine for our use case. It’s a core pillar of our value proposition.*

**Martin Pannier**

Founder and CEO, Primo

</div>

## The outcome: 400 customers, 30,000 devices, one platform

Fleet is not a tool Primo runs alongside its product. It is part of the product. Primo’s customers - internal IT teams at roughly 80% of accounts, MSP partners at the remaining 20% - get cross-platform device management without ever standing up MDM themselves. They experience Primo’s interface. Fleet runs underneath, driven entirely by Primo’s automation.

With Fleet, Primo has:

- Onboarding and offboarding sequences that take two to three minutes instead of three to four hours
- 100% Fleet coverage of MDM across every customer environment Primo operates
- Cross-platform device management across macOS, Windows, and Linux from a single API
- Scheduled, HR-triggered actions - device wipes, locks, software deployments - that fire automatically on the dates the HR system already knows

The IT platform finally moves at the speed of the HR data that drives it.

## Looking ahead

Primo and Fleet are strengthening their partnership. Primo continues to provide dedicated Fleet instances to each customer as the integration they are building deepens. Primo will be Fleet’s partner of choice for smaller deployments as Fleet continues to focus its GTM strategy on larger enterprises. As Primo extends its reach through partnerships with major HR platforms, the device fleet under management is on track to grow several times over.

For a company built on the principle that HR data should drive IT policy, the role of Fleet is clear: not a console for IT to click through, but a programmable MDM engine that runs the way the rest of Primo runs. Fleet powers the device management layer of Primo’s platform, and Primo powers IT orchestration for 400+ modern companies.

<meta name="category" value="case study">
<meta name="articleTitle" value="How Primo built an HR-driven IT platform, powered by Fleet">
<meta name="description" value="Primo chose Fleet to power the MDM layer inside its HR-driven IT orchestration platform, scaling to 400 customers and 30,000 devices.">


<meta name="publishedOn" value="2026-07-16">
<meta name="authorGitHubUsername" value="n/a">
<meta name="authorFullName" value="Fleetdm">

<meta name="companyLogoFilename" value="logo-primo-174x40@2x.png">
<meta name="quoteAuthorImageFilename" value="martin-pannier-120x120@2x.jpeg">
<meta name="quoteAuthorName" value="Martin Pannier">
<meta name="quoteAuthorJobTitle" value="Founder and CEO, Primo">
<meta name="quoteContent" value="“This seamless experience simply would not be possible without Fleet. You could try to cobble it together with cloud-only MDM providers, but Fleet’s open-source nature, API-first approach, and self-hosting capabilities make it a uniquely powerful engine for our use case. It’s a core pillar of our value proposition.”">

<meta name="companyName" value="Primo">
<meta name="companyInfo" value="Primo is an all-in-one IT orchestration platform headquartered in Paris, France. The company acts as an “IT cloud in a box” for more than 400 modern companies, bringing identity, device management, SaaS administration, and asset lifecycle together in one platform driven by HR data.">
<meta name="companyInfoLineTwo" value="Primo manages 30,000 active devices across customer environments that range from five-person startups to organizations of 500-plus employees, spanning Apple, Windows, Linux, and Android.">

<meta name="summaryChallenge" value="Primo built its platform on the principle that HR data should drive IT policy, with onboarding and offboarding measured in minutes, not hours. That only works if every layer underneath runs through code. A clickops MDM that IT had to click through was an existential problem, not a feature gap.">
<meta name="summarySolution" value="Fleet powers the MDM layer inside Primo’s platform. Fleet’s API-first, self-hostable, open-source, multi-OS design lets Primo drive device configuration, policies, software, and scheduled wipes and locks programmatically across every customer environment, as part of the same orchestration sequence that touches IDP, SaaS, and procurement.">
<meta name="summaryKeyResults" value="Onboarding and offboarding sequences that take two to three minutes instead of three to four hours; 100% Fleet coverage of MDM across every customer environment Primo operates; Cross-platform device management across macOS, Windows, and Linux from a single API; Scheduled, HR-triggered actions that fire automatically on the dates the HR system already knows">
