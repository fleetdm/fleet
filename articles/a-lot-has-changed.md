# A lot has changed.

Twelve months of MDM progress at Fleet — and what it means for your device stack.

If you looked at Fleet a year ago, you might not recognize Fleet today.

This isn't a release notes post. It's a summary of our push to close the distance between "good for Mac-heavy orgs" and "full-stack device management for teams that run everything." Here's what shipped, why it matters, and what it looks like in practice.

## First, the foundation: devices managed like infrastructure

The most important thing that changed at Fleet wasn't a feature. It was the architecture becoming the product.

Fleet has always been open source. We're transparent about almost everything. But over the past year, the declarative, GitOps-native management model moved from a design principle to a production reality for many teams. Configuration lives in YAML. Changes go through pull requests. CI/CD validates and deploys. The change advisory board — the approval bottleneck that slows down every IT org operating at scale — gets replaced by peer review.

That shift matters because it's what makes everything else possible. When your device management layer is expressed as code, you can audit it, version it, roll it back, and wire it into the same automation pipelines your infrastructure team already uses. Dan Jackson at Fastly put it simply: "The shift to infrastructure as code has modernized our operations."

It's also the foundation for AI workflows at the device layer — because agents need reliable, queryable device data and auditable change mechanisms. Fleet provides both.

## A genuinely cross-platform MDM

For a long time, "cross-platform MDM" was marketing language for "macOS-first with varying degrees of Windows support." Fleet spent the past year making the claim real.

Windows and Linux are now first-class citizens. Zero-touch enrollment, configuration profiles, policy enforcement, script deployment, and software management all work across macOS, Windows, and Linux. Mobile — iOS, iPadOS, and Android — has seen the most accelerated development.

On iOS and iPadOS, Fleet added support for company-owned apps (including in-house apps your team builds internally), account-based BYOD enrollment, self-service software catalogs, and customizable first-time device setup. On Android, Fleet now manages personal BYOD devices with work profile enforcement and is shipping full support for company-owned, fully managed Android devices.

The practical result: one platform, every OS. The device management stack stops being a collection of tools for each operating system and starts being infrastructure — with a single pane of glass, a single data model, and a single automation layer.

## BYOD and department-level control

Two specific capabilities landed in the past year that address a problem most IT teams know intimately: the gap between what gets enforced company-wide and what should only apply to specific groups.

Department-based targeting is now in Fleet. Different teams can receive different OS settings, apps, configuration profiles, and policies — without requiring separate MDM instances or manual segmentation work. An engineering org running Linux servers and macOS laptops can have different baselines than a sales team on Windows. That configuration lives in version control, not in a human's head.

BYOD enrollment — account-based personal device enrollment for Apple, work-profile-based management for Android — is fully supported. Employees get a self-service experience. IT teams get the visibility and enforcement they need. The personal/work boundary is maintained by the platform, not by policy documents nobody reads.

## Software management that actually works at scale

Fleet's software management capabilities crossed a threshold in the past year. The Fleet-maintained app catalog — a library of common applications that Fleet tests, packages, and keeps current — now covers Chrome, Office, Firefox, Slack, Zoom, and more, across macOS and Windows. New apps are shipping regularly.

But the more important story is what happens around software. Fleet connects patch status to CVE data, so you can query which devices are running a version with a known vulnerability and automate the response. Patch policies now auto-fill minimum version requirements — you define the rule, Fleet resolves the version. Compliance evidence that used to require a manual audit process can be generated automatically and exported to whatever ticketing or reporting system your team uses.

The self-service layer matters too. Employees can now install approved apps on macOS, Linux, and Windows from a Fleet-managed portal — without filing a ticket. IT capacity stays on harder problems. The 75% patch completion gap that haunts most organizations starts to close because the human friction in the loop is reduced.

## Identity-aware device management

One of the most significant additions in early 2026: Okta integration for conditional access on macOS, plus Okta as a certificate authority for dynamic challenge authentication. This closes the loop between identity posture and device posture — a capability that's been on the wishlist of every security team running zero trust architecture.

The practical implication: device compliance state can be used as a condition for access decisions. A device that's out of patch compliance or missing a required configuration can be blocked at the identity layer, not just flagged in a dashboard. The enforcement is automatic, not dependent on someone reviewing a report.

## What's coming next

The [roadmap in the next 180 days](https://fleetdm.com/announcements/roadmap-preview-april-2026) includes local admin account management on macOS, patch policies, auto-install apps on iOS, android (lock, wipe, clear), and least-privilege API-only user roles, and more. Each of these addresses a specific gap that security-conscious teams have flagged.

The direction is consistent: more automation, less manual work, tighter integration with the identity and infrastructure layers that modern IT teams already run.

If you're running a device management stack that feels like it was designed for a different era — one tool per OS, manual change processes, compliance as an annual exercise — a lot has changed, and Fleet is worth a look.  


<meta name="articleTitle" value="A lot has changed.">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="category" value="announcements">
<meta name="publishedOn" value="2026-04-16">
<meta name="description" value="An update on our progress to support mdm for everything. Here's what shipped at Fleet, why it matters, and what it looks like.">
