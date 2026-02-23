# Open-source software organization

## Aligning open-source values with cross-platform reliability

The organization behind Firefox manages a global fleet of over 1,500 devices. They required a solution that respected privacy and offered equal depth for Mac, Windows, and Linux.

## At a glance

- **Endpoints:** 1,556 (Mac, Windows, and Linux).  
- **Primary Requirement:** True multi-OS support and open-source core.  
- **Key Integrations:** Splunk and Google BigQuery.  
- **Previous Solution:** Legacy tools that treated Linux as a "second-class citizen".  

## The Challenge: Management Silos

They lacked a unified solution for remote wipe and deep logging across their diverse OS environment. Linux servers and BYOD units were major blind spots with limited compliance visibility.

## The Solution: Cultural Acceptance

Fleet’s open-source nature aligned perfectly with their values, easing cultural acceptance and reducing pushback on MDM adoption. They utilized GitOps and osquery for real-time compliance checks as code.

## The Results: Revolutionized Threat Monitoring

- **Minutes vs. Days:** Response times to new vulnerabilities shifted from days to minutes through real-time visibility and targeted patching.  
- **Automated Reporting:** The team uses the API to programmatically track fleet-maintained app versions and patch compliance.  
- **Unified Ecosystem:** Streaming telemetry into Splunk and BigQuery allowed them to correlate endpoint events with broader infrastructure logs.  

<meta name="articleTitle" value="Open-source software organization">
<meta name="authorFullName" value="Irena Reedy">
<meta name="authorGitHubUsername" value="irenareedy">
<meta name="category" value="announcements">
<meta name="publishedOn" value="2026-02-22">
<meta name="description" value="As an "The organization behind Firefox manages a global fleet of over 1,500 devices. They required a solution that respected privacy and offered equal depth for Mac, Windows, and Linux.">
