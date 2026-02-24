# Financial technology company

## Aligning MDM with DevOps practices at scale

A financial technology company manages a massive environment of roughly 15,000 endpoints. They required a solution that could keep up with their product velocity and integrate seamlessly with their "configuration as code" philosophy.

## At a glance

- **Endpoints:** ~11,000 Mac, ~864 Windows, ~3,000 Linux.  
- **Primary Requirement:** GitOps workflows and high-quality API.  
- **Key Integrations:** Salt-based config management, Digicert, and SIEM.  
- **Previous Solution:** Workspace ONE.  

## The Challenge: Vendor responsiveness and inconsistency

The team’s primary frustration with Workspace ONE was its inconsistency, particularly with delays in shipping configuration profiles and poor API quality. Linux servers also remained a silo, requiring separate scripting and tooling because of limited MDM support.

## The Solution: Configuration as Code

The team prioritized GitOps workflows and API compatibility to align with their existing Salt-based configuration management. Fleet’s public issue tracking and open communication provided the transparency needed to trust the platform's development cycle.

## The Results: Operational efficiency and real-time security

- **Rapid Migration:** The team planned a migration of 250 hosts per day to minimize disruption.  
- **MDM Proxy:** They developed an MDM proxy to replicate Salt proxy minion workflows, enabling true configuration-as-code.  
- **Security Pipeline:** Integrating Fleet with their SIEM via a Firehose pipeline enhanced threat detection and compliance monitoring.  

<meta name="articleTitle" value="Financial technology company">
<meta name="authorFullName" value="Irena Reedy">
<meta name="authorGitHubUsername" value="irenareedy">
<meta name="category" value="announcements">
<meta name="publishedOn" value="2026-02-22">
<meta name="description" value="A financial technology company manages a massive environment of roughly 15,000 endpoints."> 
