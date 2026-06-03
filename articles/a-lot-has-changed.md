# A lot has changed.

Twelve months of MDM progress at Fleet, and what it means for your device stack.

If you looked at Fleet a year ago, you might not recognize Fleet today.

This isn't a release notes post. It's a summary of our push to close the distance between "good for Mac-heavy orgs" and "full-stack device management for teams that run everything." Here's what shipped, why it matters, and what it looks like in practice.

## Devices managed like infrastructure

The most important change at Fleet isn't a feature. It is the architecture becoming the product.

Fleet has always been open source. We're transparent about almost everything. But over the past year, the declarative, GitOps-native management model moved from a design principle to a production reality for many teams. Configuration lives in YAML. Changes go through pull requests. CI/CD validates and deploys. The change advisory board, the approval bottleneck that slows down every IT org operating at scale, now, peer reviews can replace the CAB.

That shift matters because it's what makes everything else possible. When your device management layer is expressed as code, you can audit it, version it, and roll it back. You can also wire it into your infrastructure team's existing automation pipelines. [Dan Jackson at Fastly put it simply](https://fleetdm.com/case-study/fastly): "The shift to infrastructure as code has modernized our operations."

It's also the foundation for AI workflows at the device layer, because agents need reliable, queryable device data and auditable change mechanisms. Fleet provides both.

## Cross-platform MDM

For a long time, "cross-platform MDM" was marketing language for "macOS-first with varying degrees of Windows support." Fleet spent the past year making the claim real.

iOS/iPadOS, Windows, Android, and Linux are now first-class citizens. Zero-touch enrollment, configuration profiles, policy enforcement, script deployment, and software management all work across macOS, iOS/iPadOS, Windows, Android, and Linux. Mobile (iOS/iPadOS and Android) has seen the most accelerated development.

On iOS and iPadOS, Fleet added Account-based User Enrollment for BYOD devices, self-service software catalogs, and customizable first-time device setup. Fleet also supports company-owned apps, including in-house apps your team builds internally. On Android, Fleet now manages personal BYOD devices with work profile enforcement and full support for company-owned, fully managed Android devices.

The practical result is one platform, every OS. The device management stack stops being a collection of tools for each operating system. It becomes infrastructure: one interface, one data model, and one automation layer.

## BYOD and department-level control

Two capabilities landed in the past year that address a problem most IT teams know well. What gets enforced company-wide and what should apply only to specific groups are two different things, and Fleet now handles both.

Department-based targeting is now in Fleet. Different teams can receive different OS settings, apps, configuration profiles, and policies, without requiring separate MDM instances or manual segmentation work. An engineering org running Linux servers and macOS laptops can have different baselines than a sales team on Windows. That configuration lives in version control, not in a human's head.

BYOD enrollment: Fleet fully supports account-based personal device enrollment for Apple and work-profile-based management for Android. Employees get a self-service experience. IT teams get the visibility and enforcement they need. The platform maintains the personal/work boundary, not policy documents nobody reads.

## Software management that works at scale

Fleet's software management capabilities crossed a threshold in the past year. The [Fleet-maintained app catalog](https://fleetdm.com/software-catalog) covers Chrome, Office, Firefox, Slack, Zoom, and more across macOS and Windows. Fleet tests, packages, and keeps each app current. New apps are shipping regularly.

But the more important story is what happens around software. Fleet connects patch status to CVE data. You can query which devices are running a vulnerable version and automate the response. Patch policies now auto-fill minimum version requirements. You define the rule, and Fleet resolves the version. Fleet can automatically generate compliance evidence and export it to whatever ticketing or reporting system your team uses.

The self-service layer matters too. Employees can now install approved apps on macOS, Linux, and Windows from a Fleet-managed portal, without filing a ticket. IT capacity stays on harder problems. Patch rates improve as IT removes the manual steps from the process.

## Identity-aware device management

One of the most significant additions in early 2026 was the integration of Okta for conditional access on macOS. Fleet also added Okta as a certificate authority for dynamic challenge authentication. This closes the loop between identity posture and device posture, a capability that security teams running zero-trust architecture have long needed.

The practical implication is that you can use device compliance state as a condition for access decisions. A device out of patch compliance or missing a required configuration gets blocked at the identity layer. Fleet doesn't just flag it in a dashboard. The enforcement is automatic, not dependent on someone reviewing a report.

## What's coming next

The [roadmap in the next 180 days](https://fleetdm.com/announcements/roadmap-preview-april-2026) includes local admin account management on macOS, patch policies, auto-install apps on iOS, Android (lock, wipe, clear), and least-privilege API-only user roles, and more. Each of these addresses a specific gap that security-conscious teams have flagged.

The direction is consistent — more automation, less manual work, tighter integration with the identity and infrastructure layers that modern IT teams already run.

If your device management stack feels like it was designed for a different era, a lot has changed here. One tool per OS, manual change processes, compliance as an annual exercise — none of that is required anymore. Fleet is worth a look.



<meta name="articleTitle" value="A lot has changed.">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="category" value="announcements">
<meta name="publishedOn" value="2026-04-16">
<meta name="description" value="Fleet now manages macOS, Windows, Linux, iOS, Android, and ChromeOS from a single GitOps-native platform. Here's what shipped in the last year.">
