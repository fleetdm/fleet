# Scaling financial security with GitOps and RBAC

A leading digital payments provider focused on making international money transfers faster and more transparent. Operating in the highly regulated financial services industry, they required a management solution that could match the rigor of their security and compliance standards.

## At a glance

- **Endpoints:** 662 (macOS and Windows).  
- **Primary Requirement:** GitOps workflows and granular RBAC.  
- **Key Integrations:** Windows Autopilot, Okta, and Fleet Cloud.  
- **Previous Solution:** Workspace ONE and self-managed osquery.  

## The Challenge: Restrictive "magic" systems

Their previous experience with Workspace ONE was defined by high restrictions and a lack of granular Role-Based Access Control (RBAC). This made it impossible to safely delegate tasks to the help desk. Additionally, running a self-managed, on-premise osquery instance led to fragmented deployments and operational silos.

## The Solution: Fleet Cloud and Configuration-as-Code

They are unifying fragmented deployments into a single Fleet Cloud environment. This transition focuses on a GitOps-first approach, treating physical hardware like cloud infrastructure. By using Fleet’s granular RBAC, they can finally grant specific, limited access to support staff without compromising the entire management stack.

## The Results: Professionalized deployment controls

- **GitOps Adoption:** All configurations are now version-controlled and peer-reviewed, replacing "magic" backend changes with predictable, code-driven workflows.  
- **Automated Labeling:** The team uses the Fleet API for dynamic device grouping and bulk operations, replacing what was previously a manual, error-prone process.  
- **Unified Compliance:** Integrating Windows Autopilot and Okta into a single API has removed the technical debt of managing disconnected silos, providing instant data for financial audits.  

<meta name="articleTitle" value="Scaling financial security with GitOps and RBAC">
<meta name="authorFullName" value="Irena Reedy">
<meta name="authorGitHubUsername" value="irenareedy">
<meta name="category" value="announcements">
<meta name="publishedOn" value="2026-02-23">
<meta name="description" value="A leading digital payments provider focused on making international money transfers faster and more transparent. Operating in the highly regulated financial services industry, they required a management solution that could match the rigor of their security and compliance standards.">
