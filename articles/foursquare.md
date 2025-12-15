# Foursquare cut costs and gained 114% ROI with Fleet

## Why Foursquare needed a change

Foursquare supports more than 200 Macs and around 15 Windows devices in a hybrid environment, where developers are able to work on the platforms of their choice. That flexibility means the IT team must support macOS and Windows with the same level of consistency and security.

The team faced two main challenges: the complexity of managing endpoints across different MDMs, and the time required to maintain them. As a small team, they were running two separate MDM platforms — an Apple-focused MDM for Macs and another for Windows. The overhead of maintaining both systems was overwhelming and added unnecessary complexity.

To minimize this complexity, the team began automating their processes. Automation required API-driven configurability across systems, but their previous MDM’s fragmented API made it difficult to build reliable DevOps workflows. That limitation pushed the team to evaluate alternatives.

They also found that maintaining multiple MDM tools pulled time away from supporting users. They wanted to refocus their resources on serving their users’ needs instead of “looking after the tools that managed their devices.”

## The search for a solution

The team at Foursquare began evaluating alternatives. Their main goal was to consolidate their device management into a single platform that supported both macOS and Windows. Any replacement also had to match the controls they already relied on in their existing MDMs.

They needed a platform with a developer-friendly API — something simple, reliable, and easy to work with. A clear API would let the team invest more in automation, which hadn’t been practical while managing two different MDMs.

They also wanted the migration to be quick and seamless. With their past experience, the team knew that MDM changes could be disruptive, and they wanted to minimize any impact on end users.

## Choosing Fleet

The Foursquare team had met Fleet at a DevOps conference. After hearing positive feedback and conducting their own research, they decided to evaluate the platform. Fleet checked all the boxes they were looking for: macOS and Windows support, a developer-friendly API, and a GitOps-first approach.

During the evaluation, the team was particularly impressed with Fleet’s Open Source roots. The transparency gave them confidence early on. Everything from bugs to roadmaps and source code was available for review.

<div purpose="attribution-quote">

There are no surprises - you can see what they are working on at any time.

**Mike Meyer**

Senior Manager, Corporate Engineer

</div>

A key question was whether Fleet could do everything their current MDMs. The Fleet team helped them translate their current configurations into Fleet equivalents. In some cases, they discovered new ways to manage these settings that were easier to automate. Throughout the process, the Fleet team stayed closely engaged and provided strong support.

The evaluation also showed that Fleet fit well with their telemetry pipeline. They could send device data directly into their SIEM and data lake without staging it separately. Having device data alongside identity and authentication information in the same repository gave the team more flexibility in how they worked with that data.

When Foursquare decided to migrate, moving their macOS and Windows devices to Fleet was seamless. The Fleet team provided ready-to-use tools and guidance that made the transition straightforward.

<div purpose="attribution-quote">

One of the easiest, quickest, smoothest migrations I’ve ever done.

**Mike Meyer**

Senior Manager, Corporate Engineer

</div>

## Clearer visibility and stronger automation

After six months, Foursquare began to see clear benefits.

<div purpose="checklist">

They reduced time spent on endpoint maintenance by 50%. Instead of “caring for the tools,” the team could focus more on solving their customers’ needs.

They reduced MDM spend by 24% by consolidating two separate platforms into Fleet and eliminating redundant licensing.

</div>

Foursquare also gained real-time visibility into their endpoints. They could dispatch queries and send results directly into their SIEM, giving them a clearer understanding of device state. This level of visibility helped them make data-driven decisions and deploy updates or configuration changes with confidence.

Fleet has also become a key part of their automation workflow. Managing configurations through automation — instead of through a UI — gave the team a level of confidence they didn’t have before.

<div purpose="attribution-quote">

Fleet is the best tool for Foursquare - it helps my team be more creative and effective.

**Mike Meyer**

Senior Manager, Corporate Engineer

</div>


<meta name="category" value="case study">
<meta name="companyLogoFilename" value="foursquare-logo-212x40@2x.png">
<meta name="articleTitle" value="Foursquare cut costs and gained 114% ROI with Fleet">


<meta name="publishedOn" value="2025-12-11">
<meta name="authorGitHubUsername" value="n/a">
<meta name="authorFullName" value="Fleetdm">

<meta name="quoteContent" value="“One of the easiest, quickest, smoothest migrations I’ve ever done.”">
<meta name="quoteAuthorImageFilename" value="mike-meyer-120x120@2x.png">
<meta name="quoteAuthorName" value="Mike Meyer">
<meta name="quoteAuthorJobTitle" value="Senior Manager, Corporate Engineer">

<meta name="companyName" value="Foursquare">
<meta name="companyInfo" value="Foursquare is the industry's leading geospatial technology platform, designed to help businesses make smarter decisions and create more engaging customer experiences.">
<meta name="companyInfoLineTwo" value="Powered by deep machine learning and a privacy-forward approach, their technology and solutions are redefining how organizations derive value from location intelligence.">

<meta name="summaryChallenge" value="Foursquare wanted to simplify how they managed Macs and Windows devices. Their devices were split across two different MDMs, which created extra work and made automation harder for a small IT team.">
<meta name="summarySolution" value="Fleet provided Foursquare with a single MDM platform that supports both Mac and Windows. Migrating from their previous MDMs was seamless and easy, thanks to Fleet-provided out-of-the-box tools that added speed and simplicity.">
<meta name="summaryKeyResults" value="Cut endpoint maintenance effort by 50%; Reduced licensing costs by 24%; Achieved 114% ROI by removing duplicate tools; Real-time visibility with direct integration to their SIEM, with no latency; Increased reliability of security controls with GitOps">
