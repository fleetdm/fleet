# How Mollie manages endpoint compliance the same way it ships software


## The challenge: an audit that never ends

Mollie is the Dutch fintech behind payment infrastructure used by 250,000 merchants across Europe. The company powers payments for online stores from independent shops to Spotify, and runs on a motto that everyone at Mollie can recite: always be shipping. Speed and quality, every day.

Shipping fast in a regulated European fintech creates a quieter obligation behind the scenes. PCI DSS, SOC 2, the Dutch National Bank, and EU resilience rules under DORA all require Mollie to prove the controls on every endpoint, every day, on demand. An audit, at its core, is straightforward: tell us what you do, show us how you do it, and prove you are doing it that way. The hard part is the proof - getting an accurate sample of the entire device population, with confidence, every time.

<div purpose="attribution-quote">

*Compliance as code is really why we were attracted to Fleet. Endpoint is very scrutinized in audit - access management, controls, least privileged application usage. All of this can be demonstrated in code from an endpoint perspective.*

**Sam Clarke**

Senior Infrastructure Manager

</div>

For Mac, Mollie had Jamf. For Linux, which a growing share of engineers preferred, there was no real solution. Canonical Landscape gave Mollie some basic management, but not the depth or the visibility required to demonstrate PCI compliance for developers with access to cardholder data. Mollie needed a Linux platform that could run policies and scripts on demand, prove device state to an auditor, and remediate drift without manual intervention.

If Mollie ships software through code review, automation, and version control, the device fleet running that engineering needs to be managed the same way. A regulated fintech cannot run compliance through clickops.

<div purpose="attribution-quote">

*We only have a daily recon for Jamf. To get data off a device through Fleet is much quicker - we can choose granularity down to five minutes. It gives us insight into the device we didn’t otherwise have.*

**Jacob Burley**

Staff Engineer

</div>

EU data residency made the choice clear. As a fintech under DNB and DORA oversight, Mollie must host its critical infrastructure in Europe and answer for vendor lock-in as part of business continuity. Fleet’s self-hosting model meant Mollie could run the platform in its own GCP environment, on Kubernetes, with its own cloud identity and service account model. No sub-processor reviews. No reliance on a US vendor’s data residency roadmap. No feature gap between a SaaS edition and an on-prem one.

<div purpose="attribution-quote">

*Being able to self-host Fleet means we're not locked in and can stay in control of our own resiliency, which is huge in the EU for fintechs. You have to answer for business continuity. Knowing there are real options out there gives a lot of peace of mind.*

**Sam Clarke**

Senior Infrastructure Manager

</div>

## The outcome: control, on Mollie’s terms

Fleet now sits at the center of Mollie’s endpoint compliance program. The team manages its Linux end-user devices on Fleet Premium, and runs Fleet’s community edition alongside Jamf on Mac to ship osquery telemetry into BigQuery for security reporting.

With Fleet, Mollie has:

- Linux device management that meets PCI requirements for developers with access to cardholder data
- Compliance policies versioned in Git, peer-reviewed, and deployed the same way Mollie ships product code
- Near real-time visibility into device state, with query granularity down to five minutes, instead of waiting on a daily recon
- Self-healing drift remediation, with signed policies enforced locally on each endpoint, even when air-gapped
- Fleet running on Mollie’s own GCP infrastructure in Europe, meeting DNB and DORA requirements without a third-party sub-processor in the path

What changed is not just the tooling. The IT team now operates the way Mollie’s product engineers operate: in code, in version control, in the same review and deployment workflow that built the rest of the business. Compliance is no longer a clickops problem.

## Looking ahead

Mollie continues to expand its use of Fleet - new policy webhooks feeding real-time non-compliance alerts into Slack, deeper integration with platform SSO rollouts on Mac, and pull requests back to the project where the team needs something the wider community will benefit from too.

For a regulated EU fintech that has to prove compliance every day, the appeal of Fleet is in what it is not: not a black box, not a SaaS-only option, not a vendor that decides Mollie’s roadmap for it. Fleet is the platform Mollie controls, on infrastructure Mollie owns, working the way Mollie builds software.

<meta name="category" value="case study">
<meta name="articleTitle" value="How Mollie manages endpoint compliance the same way it ships software">
<meta name="description" value="Mollie runs Fleet on its own GCP infrastructure in Europe to manage Linux devices and prove endpoint compliance to PCI, DNB, and DORA auditors.">


<meta name="publishedOn" value="2026-06-12">
<meta name="authorGitHubUsername" value="n/a">
<meta name="authorFullName" value="Fleetdm">


<meta name="companyLogoFilename" value="mollie-logo-136x40@2x.png">
<meta name="quoteAuthorImageFilename" value="sam-clarke-120x120@2x.png">
<meta name="quoteAuthorName" value="Sam Clarke">
<meta name="quoteAuthorJobTitle" value="Infrastructure Manager">
<meta name="quoteContent" value="“Being able to self-host Fleet means we're not locked in and can stay in control of our own resiliency, which is huge in the EU for fintechs. You have to answer for business continuity. Knowing there are real options out there gives a lot of peace of mind.”">

<meta name="companyName" value="Mollie">
<meta name="companyInfo" value="Mollie is a payment service provider headquartered in Amsterdam, founded in 2004. The company serves more than 250,000 merchants across Europe and powers online and in-person payments for businesses of all sizes. As a regulated fintech, Mollie operates under DNB and DORA oversight.">

<meta name="summaryChallenge" value="Mollie needed Linux device management that could satisfy PCI DSS auditors on demand. Canonical Landscape covered basic management, but lacked the visibility and policy control required to prove compliance for developers with access to cardholder data.">
<meta name="summarySolution" value="Fleet replaced Canonical Landscape for Linux device management, giving Mollie a single tool to enforce CIS Level 2 baselines, run reports, and ship telemetry to BigQuery, all managed in Git and deployed through pull requests. Mollie self-hosts Fleet on its own GCP infrastructure in Europe, meeting DNB and DORA requirements without a third-party sub-processor.">
<meta name="summaryKeyResults" value="Compliance-as-code endpoint management with full data sovereignty on Mollie's own EU infrastructure; Achieved PCI-compliant Linux device management for developers with access to cardholder data; Moved compliance policies into version control, with peer review and the same deployment workflow used to ship product code; Reduced device query intervals from daily to five minutes with on-demand visibility into device state">
