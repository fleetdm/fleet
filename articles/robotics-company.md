# Robotics company unifies Mac, Windows, Linux, and Android devices

A robotics company managing specialized hardware like industrial-grade tablets in excavators alongside traditional developer desktops required a tool that could handle the complexity of Linux workstations and diverse hardware.

## At a glance

- **Endpoints:** 117 (Mac, Windows, Linux, Android).  
- **Primary requirement:** multi-platform management and GitOps workflows.  
- **Key integrations:** osquery, Tailscale, and Google Credential Provider.  
- **Previous solution:** limited manual management for Linux.  

## The challenge

Before Fleet, Linux desktops were a significant "blind spot". High configuration complexity and Nvidia driver conflicts made it nearly impossible to scale or manage these devices effectively. The team needed a way to manage WiFi profiles and kiosk configurations across a varied fleet.

## The solution

Fleet met the requirement for a single point of truth across macOS, Windows, and Linux. The team implemented GitOps workflows and used Fleet’s open-source transparency to build confidence in the reliability of their stack.

## The results

- **Google Auth on Windows:** The team automated the deployment of the Google Credential Provider for Windows, removing the need for Active Directory dependencies.  
- **Real-time network access:** By integrating host vitals with Tailscale, the team now makes real-time network access decisions based on device health.  
- **Proactive IT:** Cross-platform automation and policy checks have allowed the team to shift from reactive troubleshooting to proactive management.  


<meta name="articleTitle" value="Robotics company unifies Mac, Windows, Linux, and Android devices">
<meta name="authorFullName" value="Irena Reedy">
<meta name="authorGitHubUsername" value="irenareedy">
<meta name="category" value="case study">
<meta name="publishedOn" value="2026-02-22">
<meta name="description" value="A robotics company unified Mac, Windows, Linux, and Android with Fleet, enabling GitOps, proactive security, and real-time device control."> 
<meta name="useBasicArticleTemplate" value="true">
<meta name="cardTitleForCustomersPage" value="Robotics company">