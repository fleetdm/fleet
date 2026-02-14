# GitOps for device management: Audit trails, compliance, and configuration as code

Managing device configurations across thousands of devices raises familiar questions during compliance audits: who changed this policy, when did it happen, and why? MDM consoles handle the day-to-day work, but reconstructing change history from audit logs can be tedious. GitOps workflows offer one way to solve this by bringing version control to device management. This guide covers what GitOps is, how it applies to device management, and when it makes sense for your team.

## What is GitOps?

GitOps is an operational framework that uses Git repositories as the single source of truth for infrastructure and system configurations. Instead of making changes directly through admin consoles or command-line tools, teams define what infrastructure should look like in version-controlled files stored in Git. Automated agents can then periodically compare that desired state against reality and correct configuration drift, depending on how the workflow is implemented.

The approach rests on four core principles:

* **Declarative configuration:** Configurations describe the end state rather than the steps to get there.  
* **Version-controlled state:** Configuration definitions live in Git, creating a detailed history of changes.  
* **Automated deployment:** When changes are approved, they can be deployed automatically or via a controlled sync step.  
* **Continuous reconciliation:** Agents reconcile actual state with desired state, helping catch configuration drift before it causes problems.

Every modification flows through Git commits and pull requests, creating a tamper-evident change history. With branch protection, audit logs, and signed commits, Git can provide strong author attribution and review chains. When auditors ask who changed a policy and why, much of that evidence lives in Git history rather than scattered logs. In many GitOps systems, agents pull desired state from Git; in device management, devices typically check in to fetch updates, which can reduce reliance on long-lived credentials in external CI systems.

## How does GitOps change the way you manage devices and MDM?

Most device management happens through an MDM console. Administrators click through settings and changes propagate across the fleet. This works well for day-to-day operations, but tracking who changed what over time requires digging through audit logs that may lack context about why changes happened. For teams managing macOS, Windows, and Linux fleets who want stronger change tracking, GitOps offers an alternative.

MDM tools that support GitOps workflows expand what's possible. When your MDM provides API coverage and Git-based configuration management, you get both: a console for visibility and quick changes, plus version-controlled configurations for auditability and automation.

GitOps adds version control to this model. Device profiles, security policies, and software configurations live as YAML or JSON files in Git repositories. When a policy needs updating, someone creates a pull request, gets peer review, and merges the change. The MDM tool then applies configuration changes defined in Git to targeted devices when a sync workflow runs.

### Practical advantages for IT teams

This shift can bring several practical advantages for IT teams managing large fleets:

* **Version-controlled configurations:** Device policies can be defined as reviewable files in Git. You can see exactly what changed between versions, who approved it, and roll back problematic changes with a simple git revert.  
* **Collaborative change management:** Team members propose policy changes through pull requests, creating natural review gates. You can still make quick changes in the console when needed, but major policy updates go through version control.  
* **Consistent multi-platform policies:** Defining configurations in code lets you enforce the same security baselines across different operating systems using standardized workflows.  
* **Disaster recovery simplification:** Most of your device management configuration can be reconstructed from repository contents if something goes wrong.

Together, these capabilities turn device management into a repeatable, auditable process that scales with your fleet while preserving the flexibility of console access when you need it.

### How GitOps workflows operate

These advantages compound when your team adopts consistent Git-based workflows. Engineers define device profiles and policies in version-controlled files, changes go through pull request reviews, and CI/CD pipelines or native integrations apply approved changes to targeted device groups. The device management system can then monitor compliance on an ongoing basis and trigger remediation when devices drift from the desired state.

For organizations already using infrastructure-as-code practices, extending Git-based workflows to device management creates consistency across teams. This approach lets device management teams adopt the same modern practices as platform engineering teams, adding code-based workflows alongside their existing console access.

## When GitOps makes sense for compliance and audits

For organizations facing frequent compliance audits, GitOps can simplify evidence collection. Auditors ask for evidence of change controls, access reviews, and configuration baselines. Reconstructing this history from console audit logs is possible but time-consuming. GitOps addresses this by building compliance evidence into everyday workflows.

### Immutable audit trails

Git's commit history uses cryptographic hashes to create a tamper-evident record of configuration changes. In practice, repository configuration and access controls determine whether history can be rewritten or deleted. Each commit captures who made the change, when it happened, what specifically changed, and (through commit messages and pull request discussions) why it was necessary. 

This provides useful evidence for change management requirements across multiple compliance frameworks, though auditors still evaluate surrounding processes like access governance, approval workflows, and logging retention.

For SOC 2 assessments, Git commit history can help satisfy Common Criteria around logical access controls and change management. The pull request approval chain demonstrates that changes went through proper review before deployment. For HIPAA technical safeguards, the commit history shows how configurations protecting electronic protected health information evolved over time.

### Continuous compliance validation

Rather than treating compliance as a periodic audit activity, GitOps lets you validate configurations continuously. Policy-as-code tools can evaluate proposed configurations against compliance rules before changes reach production. If someone tries to merge a configuration that violates security baselines, automated checks can catch it during the pull request review.

### Framework-specific alignment

Different compliance frameworks emphasize different controls, but GitOps provides evidence applicable across many of them simultaneously. Examples of how GitOps evidence can map to common controls include:

* **SOC 2 Type II:** Git repository access logs combined with pull request approvals can help satisfy CC6 (Logical and Physical Access Controls) requirements. Git commit history with mandatory pull request reviews supports CC8 (Change Management) practices. Reconciliation logs from GitOps operators can address CC7 (System Operations and Monitoring) requirements.  
* **HIPAA:** Git history records modifications to configurations affecting protected health information (ePHI), supporting §164.312(a)(1) Access Control and §164.312(b) Audit Controls. Git-based RBAC combined with audit logs provides technical safeguards for ePHI access and modifications.  
* **FedRAMP:** Versioned declarative state addresses CM-2 (Baseline Configuration) requirements. Pull request workflows with branch protection can support CM-3 (Configuration Change Control) and CM-9 (Configuration Management Plan) processes. Git commit logs with author attribution and timestamps help satisfy AU-2 (Audit Events) and AU-3 (Content of Audit Records) control requirements.

Security teams managing multiple compliance frameworks often find that GitOps can reduce duplicated effort when they standardize change management through Git. A single Git commit history provides evidence for change management controls across SOC 2, HIPAA, and FedRAMP without maintaining separate documentation systems.

### Drift detection and remediation

Configuration drift occurs when actual device state diverges from documented baselines. This divergence creates compliance risk that can be difficult to detect through periodic manual checks. Manual changes made outside normal processes or devices missing updates can leave fleets in an inconsistent state that fails audit validation.

GitOps can address this challenge through reconciliation workflows: automated agents compare desired state (defined in Git) with actual state (observed in production) and, when configured to do so, correct drift automatically or surface it for investigation. This helps maintain more consistent configurations between audit cycles instead of discovering issues only during assessments.

## How GitOps principles apply to multi-platform MDM and enterprise compliance

Applying GitOps principles to device management requires a tool designed for declarative configuration and strong API support. [Fleet provides GitOps workflows](https://fleetdm.com/fleet-gitops) for multi-platform device management, where device profiles, security policies, and configurations are defined as YAML files in Git repositories. Fleet is API-first and supports managing configurations via UI, REST API, and GitOps, enabling robust automation alongside console access. This approach gives your team declarative control over devices while maintaining the audit trails and version control that compliance frameworks require.

### Declarative device configuration

Fleet's Git-based workflow lets you [define device configurations](https://fleetdm.com/docs/configuration/yaml-files) declaratively in YAML files stored in Git. Engineers propose changes through pull requests, teams review and approve modifications, and Fleet applies the approved configuration when your workflow runs. Devices receive updated settings when they next check in.

For macOS, you can define configuration profiles, OS update policies with deadlines and minimum versions, and bootstrap packages. Windows configurations support similar patterns for profiles and OS update management. Linux systems support script-based management and configuration deployment.

### Enforcing security baselines across platforms

This declarative model extends to security baselines. Fleet can enforce disk encryption on macOS and Windows, and you can manage that enforcement through UI, API, or GitOps workflows. Using teams and labels, these configurations can be scoped to specific device groups based on operating system type, department, or other criteria.

For example, a macOS configuration profile can be version-controlled alongside Windows security policies in the same repository, with Fleet applying the appropriate configuration to each device type based on your targeting rules.

### Label-based targeting for staged rollouts

Fleet provides labels that let you group devices by operating system version, department, hardware type, or custom attributes. You can roll out new macOS profiles exclusively to engineering devices running a specific OS version, scope configurations to specific teams, or target devices based on custom definitions. This targeting is defined in your GitOps YAML files, making rollout strategies version-controlled.

### Maintaining compliance across large fleets

For organizations managing compliance across frameworks like SOC 2, FedRAMP, and PCI-DSS, Git-based approaches provide the audit trail and change control documentation these programs require. With Fleet's GitOps workflow, configuration changes are managed via Git with full attribution through commit history and pull request reviews. Fleet applies configurations when your workflow runs, and you can schedule these workflows to keep device configurations closely aligned with your Git repository as devices check in.

Fleet's osquery foundation provides over 300 queryable data tables for generating audit evidence, from installed software and running processes to security configurations and user accounts. In Fleet Premium, the platform can check devices against CIS Benchmarks and custom policies continuously, not just during audits, so compliance posture stays visible between assessment cycles. 

Dashboards show which devices meet policy requirements and which have drifted from expected configurations, giving leadership confidence in security posture without periodic scrambles.

## Bring GitOps workflows to your device fleet

Modern device management can benefit from many of the same infrastructure-as-code practices that transformed application deployment. GitOps workflows provide the audit trails, change controls, and version-controlled configurations that compliance frameworks demand while giving IT teams familiar tools for managing device configurations.

Fleet is an open-source device management tool that provides osquery-based device visibility, vulnerability management, and MDM capabilities across macOS, Windows, and Linux. Because Fleet is open source, security teams can inspect exactly how policies are enforced and verify compliance logic rather than trusting a black box. 

Define security policies, configuration profiles, and update schedules in version-controlled YAML files, then let Fleet apply them across your device fleet through GitOps workflows. [Try Fleet](https://fleetdm.com/try-fleet) to see how GitOps-based device management works for your team.

## Frequently asked questions

### What is the difference between GitOps and legacy CI/CD?

Legacy CI/CD pipelines push changes to production by connecting external systems to infrastructure. GitOps inverts this model, with agents inside the environment typically pulling desired state from Git repositories. This pull-based approach can improve security because production systems often avoid exposing write endpoints or sharing long-lived credentials with external pipelines.

### How long does it take to implement GitOps for device management?

Implementation timelines vary based on fleet size and existing processes. Organizations typically start by migrating a subset of configurations to version control, then expand coverage over several weeks. Teams already comfortable with Git workflows often find the transition faster since the concepts translate directly from application deployment practices.

### What compliance frameworks does GitOps support?

GitOps provides audit evidence that can support SOC 2, HIPAA, FedRAMP, and PCI-DSS requirements around change management, access controls, and configuration baselines when combined with appropriate technical and process controls. The Git commit history creates tamper-evident records with author attribution and timestamps that help satisfy multiple frameworks simultaneously.

### Can I use GitOps with my current MDM tool?

GitOps principles apply to any MDM that supports declarative configuration and API-driven management. Some MDM tools offer limited API coverage that restricts what can be managed through code, so GitOps capabilities vary by vendor. Fleet supports native GitOps workflows where configurations are defined in YAML files and applied automatically when your workflow runs. [Explore Fleet's docs](https://fleetdm.com/docs) to evaluate whether its approach fits your GitOps strategy.

<meta name="articleTitle" value="What is GitOps? How it transforms device management and compliance">
<meta name="authorFullName" value="Brock Walters">
<meta name="authorGitHubUsername" value="nonpunctual">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-02-14">
<meta name="description" value="Learn what GitOps is, how it changes device management through version control and automation & why compliance teams rely on immutable audit trails.">
