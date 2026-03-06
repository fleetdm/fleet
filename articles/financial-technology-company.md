# Financial technology company manages 15,000 devices with GitOps

A financial technology company managing a massive environment of roughly 15,000 endpoints required a solution that could keep up with their product velocity and integrate seamlessly with their "configuration as code" philosophy.

## At a glance

- **Endpoints:** ~11,000 Mac, ~864 Windows, ~3,000 Linux.  
- **Primary requirement:** GitOps workflows and high-quality API.  
- **Key integrations:** Salt-based config management, Digicert, and SIEM.  
- **Previous solution:** Workspace ONE.  

## The Challenge

The team’s primary frustration with Workspace One was its inconsistency, particularly with delays in shipping configuration profiles and poor API quality. Linux servers also remained a silo, requiring separate scripting and tooling because of limited MDM support.

## The Solution

The team prioritized GitOps workflows and API compatibility to align with their existing Salt-based configuration management. Fleet’s public issue tracking and open communication provided the transparency needed to trust the platform's development cycle.

## The Results

- **Rapid migration:** The team planned a migration of 250 hosts per day to minimize disruption.  
- **MDM proxy:** They developed an MDM proxy to replicate Salt proxy minion workflows, enabling true configuration-as-code.  
- **Security pipeline:** Integrating Fleet with their SIEM via a Firehose pipeline enhanced threat detection and compliance monitoring.

## About Fleet

Fleet is the single endpoint management platform for macOS, iOS, Android, Windows, Linux, ChromeOS, and cloud infrastructure. Trusted by over 1,300 organizations, Fleet empowers IT and security teams to accelerate productivity, build verifiable trust, and optimize costs.
By bringing infrastructure-as-code (IaC) practices to device management, Fleet ensures endpoints remain secure and operational, freeing engineering teams to focus on strategic initiatives.
Fleet offers total deployment flexibility: on-premises, air-gapped, container-native (Docker and Kubernetes), or cloud-agnostic (AWS, Azure, GCP, DigitalOcean). Organizations can also choose fully managed SaaS via Fleet Cloud, ensuring complete control over data residency and legal jurisdiction.

<meta name="articleTitle" value="Financial technology company manages 15,000 devices with GitOps">
<meta name="authorFullName" value="Irena Reedy">
<meta name="authorGitHubUsername" value="irenareedy">
<meta name="category" value="case study">
<meta name="publishedOn" value="2026-02-22">
<meta name="description" value="A financial technology company manages a massive environment of roughly 15,000 endpoints."> 
<meta name="useBasicArticleTemplate" value="true">
