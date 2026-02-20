# Communications services sector scaling cross-platform device management with Fleet

A leading communications platform manages a diverse environment of approximately 3,000 endpoints, including Mac, Windows, and Linux. Seeking to eliminate management silos and improve visibility, the team turned to Fleet to provide a unified, transparent, and automated approach to device orchestration.

## At a glance

- **Endpoints:** ~3,000 (Mac, Windows, and Linux).  
- **Primary Requirement:** Unified management via a single binary/API.  
- **Key Integrations:** osquery, GitOps workflows, and BigQuery.  
- **Previous Solution:** Jamf.  

## The Challenge: Overcoming silos and "blind spots"

Before adopting Fleet, the team relied on Jamf, but faced significant hurdles. Primary frustrations included limited feature completeness for application management and compliance auditing. Furthermore, support was found to be unreliable during critical incidents. Technical gaps in managing Linux servers and remote laptops created significant "blind spots" in their infrastructure.

## The Solution: Transparency and GitOps

The team identified three top requirements for a new solution: osquery integration, GitOps workflows for configuration management, and robust support for multi-platform management—specifically Linux. Fleet’s open-source nature allowed internal reviews of the management stack, ensuring no "hidden agents" were running. This transparency also fostered trust with engineers, as they could inspect exactly how Fleet worked.

## The Results: Real-time visibility and automated workflows

By consolidating to Fleet, siloed processes were replaced with a single API and binary.

- **Real-time Visibility:** Fleet significantly improved response times for handling vulnerabilities and gathering audit evidence through unified logs.  
- **Streamlined Automation:** The team now uses Fleet’s API to automate complex tasks, such as orchestrating Linux bootstrap scripts and managing package installations via internal repositories.  
- **Advanced Telemetry:** By streaming telemetry directly to BigQuery, the security team enhanced its ability to monitor threats and detect anomalies instantly.  

<meta name="articleTitle" value="Communications services sector scaling cross-platform device management with Fleet">
<meta name="authorFullName" value="Irena Reedy">
<meta name="authorGitHubUsername" value="irenareedy">
<meta name="category" value="announcements">
<meta name="publishedOn" value="2026-02-20">
<meta name="description" value="Communications services sector scaling cross-platform device management with Fleet.">
