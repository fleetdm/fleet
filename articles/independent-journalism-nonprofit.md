# Independent journalism nonprofit

## Managing journalism infrastructure with modern DevOps

An independent journalism nonprofit that builds infrastructure for independent journalism and manages a remote-first fleet of macOS and Linux devices required a developer-centric tool that aligned with their open-source values.

## At a glance

- **Endpoints:** ~39 (Mac and Linux).  
- **Primary requirement:** open-source core and GitOps capabilities.  
- **Key integrations:** osquery, internal team dashboards.  
- **Previous solution:** traditional "black-box" MDMs.  

## The challenge: value misalignment

Traditional MDMs felt like "black boxes" that were misaligned with their mission. They specifically struggled with maintaining consistent visibility into Linux workstations.

## The solution: declarative configuration

They chose Fleet for its ability to treat Linux and macOS with equal visibility via osquery. They manage their device state via version-controlled repositories (GitOps), which allows them to stay lean.

## The results: high-trust remote management

- **Invisible transition:** the migration was nearly invisible to users, leveraging declarative configurations to manage states without disruption.  
- **Transparent metrics:** device compliance status is synced directly to internal team dashboards via the API.  
- **Live Posture checks:** the team can run live queries across the entire foundation to confirm security status in seconds.

- <meta name="articleTitle" value="Independent journalism nonprofit">
<meta name="authorFullName" value="Irena Reedy">
<meta name="authorGitHubUsername" value="irenareedy">
<meta name="category" value="announcements">
<meta name="publishedOn" value="2026-02-22">
<meta name="description" value="Journalism nonprofit manages Mac and Linux devices with open source, GitOps workflows, and transparent, real-time compliance visibility.">
