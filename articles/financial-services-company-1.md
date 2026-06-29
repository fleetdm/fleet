# Financial services company reduces tool sprawl with Fleet

A financial services company supports investment operations, management, and processing for institutions and high-net-worth clients. Its environment includes a large Windows fleet along with macOS and Linux systems.

Before Fleet, device management was fragmented across multiple tools. Fleet gives the team a more consistent and auditable way to manage devices across operating systems.

## At a glance

* **Industry:** Financial services and security operations

* **Devices managed:** ~9,000-10,000 devices

* **Primary requirements:** Performance stability, strict change control, unified API

* **Previous challenge:** Too many tools and limited visibility into Linux and network devices

## The challenge

Before Fleet, the company managed devices with several separate tools, including Jamf, SCCM, and Intune.

This created operational overhead and made it difficult to maintain consistent workflows. Linux systems and network devices remained difficult to monitor, leaving security teams without the visibility they needed.

The team wanted a platform that could support strict change control, avoid performance issues, and provide a reliable API for automation.

## The evaluation criteria

The team prioritized three requirements:

1. **Performance stability**  
    Avoid unnecessary impact on device performance.

2. **Strict change control**  
    Pin specific versions of osquery and Orbit.

3. **Unified API**  
    Automate workflows across macOS, Windows, and Linux.

## The solution

Fleet gave the team a single platform with version control, flexible scheduling, and better telemetry.

The company integrated Fleet API calls into Airflow jobs to automate data collection and reporting. This supported security hunting and audit workflows without relying on disconnected tools.

Fleet also helped the team tailor data collection by device group, reducing noise and making security operations more targeted.

## The results

Fleet simplified management across a large environment.

* **Reduced vendor sprawl:** The team can consolidate multiple management tools into one platform.

* **Faster audit readiness:** Compliance data for thousands of devices is easier to access.

* **Better visibility:** Linux systems are no longer as isolated from broader security workflows.

## Why they recommend Fleet

For this company, the biggest benefit is consolidation with control.

Fleet provides the team with a single, open platform that meets strict operational requirements while improving visibility across its entire environment.


<meta name="articleTitle" value="Financial services company reduces tool sprawl with Fleet">
<meta name="authorFullName" value="Irena Reedy">
<meta name="authorGitHubUsername" value="irenareedy">
<meta name="category" value="case study">
<meta name="publishedOn" value="2026-03-18">
<meta name="description" value="A financial services company replaces multiple tools with Fleet, improving visibility and control across macOS, Windows, and Linux."> 
<meta name="useBasicArticleTemplate" value="true">
<meta name="cardTitleForCustomersPage" value="Financial services company">