# Computational research company unifies endpoint management with Fleet

A computational research company develops software that accelerates drug discovery and materials science. Its teams rely on macOS, Linux, and Windows devices across a highly technical environment, and the company is working to manage all three on a single platform.

Fleet is the foundation of that strategy. The team has rolled out Fleet for macOS first — replacing legacy Mac management with a more transparent, automation-friendly approach — with Linux and Windows planned to follow.

## At a glance

- **Industry:** Healthcare technology and computational research
- **Environment:** ~1,553 endpoints across macOS, Linux, and Windows
- **Currently rolled out with Fleet:** macOS MDM
- **Primary requirements:** Unified visibility (long-term), security enforcement, GitOps, osquery support
- **Previous challenge:** Fragmented tooling across operating systems and limited Linux visibility

## The challenge: fragmented tooling across operating systems

Before Fleet, the company relied on multiple tools across operating systems. Workspace ONE did not meet expectations for support and communication. Intune lacked some Windows capabilities the team needed, and Jamf did not provide sufficient depth for a mixed environment with a large Linux footprint.

The team decided to consolidate on a single platform that could eventually manage all three operating systems. Linux desktops and servers were the biggest long-term gap, but the most immediate opportunity was modernizing macOS management — the largest piece of the environment that was already actively managed.

## The evaluation criteria

The team focused on three priorities:

1. **Unified visibility** — A single platform that can eventually manage macOS, Windows, and Linux.
2. **Security and compliance enforcement** — Automatically apply policies and reduce manual work.
3. **GitOps and osquery support** — Manage configuration through code and use SQL-based telemetry for deeper visibility.

## The solution: macOS first, with a path to unified management

Fleet gave the team a single system for macOS policy management, device visibility, and automation — and a clear path to extend the same platform to Linux and Windows over time.

In the macOS rollout, the team uses Fleet to manage policies, map device users via SCIM, and schedule updates that reduce disruption for scientists and other technical users. Fleet also replaced parts of the previous automation stack, which reduced complexity.

The open-source model was a strong fit because it gave the team direct visibility into how the platform works and how features evolve over time — important context as they plan the next phases of rollout for Linux and Windows.

## The results

Fleet helped the team modernize macOS management and lay the groundwork for unified endpoint management across the environment.

- **Modernized macOS management:** Legacy macOS tooling has been replaced with a more transparent, automation-friendly platform.
- **Stronger vulnerability response on macOS:** Real-time telemetry and policy enforcement help the team respond faster on managed systems.
- **A foundation to build on:** With macOS in production, the team has a proven base for extending Fleet to Linux and Windows.

## Why they recommend Fleet

For this company, the biggest benefit is having a foundation for unified endpoint management. Fleet gives the team one platform to operate today on macOS — with greater transparency and automation than the legacy tools it replaced — and the same platform can extend to Linux and Windows as the team's roadmap progresses.


<meta name="articleTitle" value="Computational research company unifies endpoint management with Fleet">
<meta name="authorFullName" value="Irena Reedy">
<meta name="authorGitHubUsername" value="irenareedy">
<meta name="category" value="case study">
<meta name="publishedOn" value="2026-03-18">
<meta name="description" value="Fleet helps a research company replace multiple tools with unified endpoint management."> 
<meta name="useBasicArticleTemplate" value="true">
<meta name="cardTitleForCustomersPage" value="Computational research company">
