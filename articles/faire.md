# Faire secures Macs with CIS benchmarks and Fleet


## Why Faire needed a change

Faire’s IT team is highly technical and prefers vendors that provide features through APIs. They’re builders who extend applications when out-of-the-box features fall short, and their workflows are highly automated and managed as code.

They had used their previous MDM for several years, but frustration grew as it lagged in adopting new MDM APIs, including Apple’s Declarative Device Management. Faire’s workflow, based on ‘config-as-code’, also proved challenging. 

With all their other IT systems, like their IDP, productivity software, and cloud infrastructure, all configured through code with automated CI/CD pipelines, maintaining an MDM that had to be managed via the UI became increasingly painful.


One team member spent months trying to build a Terraform provider for the previous MDM, but gave up because of bugs and inconsistent APIs.

Support also became a problem. Feature requests often felt ignored, and responses were slow, which didn’t work for a company of Faire's size.

## The search for a solution

Faire focused on three priorities when selecting a new MDM:

<div purpose="checklist">
    
API-first architecture: A first-class API built for deep integration, not an API layered on top of a UI

Comprehensive Apple support: End-to-end lifecycle management for macOS and iPadOS, including automated enrollment, policy controls, and software distribution

A reputable vendor: A well-established partner they could rely on for the long term
</div>

## Choosing Fleet

Faire selected Fleet after a bake-off with three other MDM vendors, including an open-source option. They were already using Fleet to manage osquery telemetry, and the availability of a SaaS deployment matched their move away from self-hosting.

All Fleet features are accessible through an API, with examples that show how to automate tasks through GitOps. Fleet integrated directly into Faire’s onboarding workflows, and their engineers appreciate managing device configurations through GitOps and pull requests.

Fleet also gives Faire more flexibility for managing Macs and iPads. They use custom profiles and can tap into native MDM APIs that Fleet exposes, even when those APIs are not yet implemented as built-in features.

In addition, Fleet helps Faire monitor Macs against CIS benchmarks. This improves their device security posture and gives IT the ability to take remediation actions when they find issues.

Fleet’s reputation in the Mac Admins community also matters to Faire. Industry experts highly regard the team and product, and Fleet’s open-source foundation gives Faire confidence that the underlying IP will always remain available.

<div purpose="attribution-quote">

*My team is loving managing devices via GitOps with Fleet.*

**Jeremy Baker**

Engineering Manager

</div>

## The future

Faire continues to be impressed by Fleet’s proactive support. For example, Fleet reached out when they detected a downed webhook on Faire’s end. In another instance, an engineer got on a video call 10 minutes after Faire filed a critical migration ticket.

Telemetry from Fleet remains important for device posture. It gives Faire the visibility they need to assess device signals alongside their IDP. Fleet’s integrated MDM then provides a path to remediation and risk mitigation when they find issues.

Fleet is working with Faire to explore managing other platforms, including Windows, Linux, and potentially BYOD mobile devices.

Interested to learn more? [Read Faire’s article](https://craft.faire.com/using-observability-to-reduce-chaos-in-an-mdm-migration-20a0056a48e7) about their MDM migration.




<meta name="category" value="case study">
<meta name="articleTitle" value="Faire secures Macs with CIS benchmarks and Fleet">


<meta name="publishedOn" value="2025-12-11">
<meta name="authorGitHubUsername" value="n/a">
<meta name="authorFullName" value="Fleetdm">


<meta name="companyLogoFilename" value="faire-logo-192x40@2x.png">
<meta name="quoteAuthorImageFilename" value="jeremy-baker-120x120@2x.png">
<meta name="quoteAuthorName" value="Jeremy Baker">
<meta name="quoteAuthorJobTitle" value="Engineering Manager">
<meta name="quoteContent" value="“Fleet has opened a lot of opportunities for us. My team is loving managing devices via GitOps, and the built-in support for CIS benchmarks has made it easy to enforce these on our devices.”">

<meta name="companyName" value="Faire">
<meta name="companyInfo" value="Faire is a global online wholesale marketplace that connects independent retailers with emerging and established brands. Retailers use the platform to discover products and manage inventory, and brands use it to reach a wider audience and streamline wholesale operations.">
<meta name="companyInfoLineTwo" value="Faire operates in the US, Canada, and the UK, with about 1,000 employees. Most of their endpoints run macOS, with some Windows laptops, and they also manage 300 iPads used for Zoom Rooms.">

<meta name="summaryChallenge" value="Faire’s previous MDM slowed them down. It lacked timely support for new OS controls, its APIs made config-as-code workflows hard to automate, and support responses were too slow to keep up with their needs.">
<meta name="summarySolution" value="Fleet replaced their previous MDM for managing Macs and iPads, and supports the GitOps workflows they use to manage their infrastructure.">
<meta name="summaryKeyResults" value="Enforced CIS Level 1 benchmarks to keep laptops secure; Migrated 1,000 Macs and iPads to Fleet without disruption; Moved device management workflows into GitOps; Received proactive support from Fleet">

