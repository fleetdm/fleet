# Communications platform unifies device management across 3,000 devices

A leading communications platform manages a diverse environment of approximately 3,000 endpoints, including Mac, Windows, and Linux. Seeking to eliminate management silos and improve visibility, the team turned to Fleet to provide a unified, transparent, and automated approach to device orchestration.

## At a glance

- **Endpoints:** ~3,000 (Mac, Windows, and Linux).  
- **Primary requirement:** unified management via a single binary/API.  
- **Key integrations:** osquery, GitOps workflows, and BigQuery.  
- **Previous solution:** Jamf.  

## The challenge

Before adopting Fleet, the team relied on Jamf, but faced significant hurdles. Primary frustrations included limited feature completeness for application management and compliance auditing. Furthermore, support was found to be unreliable during critical incidents. Technical gaps in managing Linux servers and remote laptops created significant "blind spots" in their infrastructure.

## The solution

The team identified three top requirements for a new solution: osquery integration, GitOps workflows for configuration management, and robust support for multi-platform management—specifically Linux. Fleet’s open-source nature allowed internal reviews of the management stack, ensuring no "hidden agents" were running. This transparency also fostered trust with engineers, as they could inspect exactly how Fleet worked.

## The results

By consolidating to Fleet, siloed processes were replaced with a single API and binary.

- **Real-time visibility:** Fleet significantly improved response times for handling vulnerabilities and gathering audit evidence through unified logs.  
- **Streamlined automation:** the team now uses Fleet’s API to automate complex tasks, such as orchestrating Linux bootstrap scripts and managing package installations via internal repositories.  
- **Advanced telemetry:** by streaming telemetry directly to BigQuery, the security team enhanced its ability to monitor threats and detect anomalies instantly.  


<meta name="articleTitle" value="Communications platform unifies device management across 3,000 devices">
<meta name="authorFullName" value="Irena Reedy">
<meta name="authorGitHubUsername" value="irenareedy">
<meta name="category" value="case study">
<meta name="publishedOn" value="2026-02-20">
<meta name="description" value="Communications platform unifies 3,000 devices with real-time visibility, GitOps automation, and transparent cross-platform management.">
<meta name="useBasicArticleTemplate" value="true">
<meta name="cardTitleForCustomersPage" value="Communications platform">
