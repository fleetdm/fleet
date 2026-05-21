# National research lab scales host visibility with Fleet

A national research laboratory supports high-performance computing and advanced scientific research. Its infrastructure includes physical servers, compute nodes, and large Linux environments that require careful operational control.

Fleet helps the lab manage these systems by providing faster access to device data and reducing manual work.

## At a glance

* **Industry:** National research and high-performance computing

* **Devices managed:** ~2,000 hosts

* **Primary requirements:** Self-hosting, GitOps workflows, team segmentation

* **Previous challenge:** Manual reporting and limited visibility into specialized environments

## The challenge

Before Fleet, reporting was manual.

Teams generated PDFs and CSV files, then passed them between groups. That process took time and made it harder to respond quickly to audits or operational questions.

The lab also needed better visibility into HPC clusters and specialized Linux servers that were not easy to manage with traditional tools.

## The evaluation criteria

The team focused on three capabilities:

1. **Self-hosting**  
    Maintain full control of infrastructure for security and compliance reasons.

2. **GitOps workflows**  
    Manage configuration changes with peer review and version control.

3. **Team segmentation**  
    Support different host groups across the lab with granular control.

## The solution

Fleet gave the team direct access to device data without relying on manual reporting cycles.

The lab migrated from vanilla osquery to Fleet Orbit in phases, which helped reduce disruption while bringing more systems under management. Fleet Desktop also helped improve the feedback loop between infrastructure teams and researchers by giving users visibility into the state of their hosts.

Fleet data was streamed to Splunk, which supported monitoring without relying on local log files that created extra disk churn.

## The results

Fleet replaced manual reporting with a more automated and scalable approach.

* **Faster audit response:** Teams can check software inventory and policy status on demand.

* **Lower administrative overhead:** Automated policies and queries reduce manual reporting work.

* **Minimal migration impact:** Phased rollout helped protect performance in compute-heavy environments.

## Why they recommend Fleet

For this lab, the biggest benefit is scalable visibility. Fleet gives the team a faster, more direct way to manage thousands of hosts across specialized research environments.


<meta name="articleTitle" value="National research lab scales host visibility with Fleet">
<meta name="authorFullName" value="Irena Reedy">
<meta name="authorGitHubUsername" value="irenareedy">
<meta name="category" value="case study">
<meta name="publishedOn" value="2026-03-17">
<meta name="description" value="A national research lab scales host visibility with Fleet, improving reporting and reducing manual work.">
<meta name="useBasicArticleTemplate" value="true">
<meta name="cardTitleForCustomersPage" value="National research lab">
