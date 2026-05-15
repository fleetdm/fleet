Managing device configurations across thousands of devices raises familiar questions during compliance audits: who changed this policy, when did it happen, and why? MDM consoles handle the day-to-day work, but reconstructing change history from audit logs can be challenging. GitOps workflows solve this by bringing version control to device management.

GitOps grew out of infrastructure-as-code practices that became common in teams managing cloud services, where configurations moved from admin consoles and one-off commands into version-controlled files. Many organizations already have platform or cloud teams working this way, even if device management teams still rely mostly on MDM consoles. Applying GitOps to device management brings those same code-based practices to device configuration and compliance workflows.

This guide covers what GitOps is, how it applies to device management, and when it makes sense for IT and security teams.

## What is GitOps?

GitOps is an operational framework that uses Git repositories as the single source of truth for infrastructure and system configurations. Instead of making changes directly through admin consoles or command-line tools, teams define what infrastructure should look like in version-controlled files stored in Git. Automated workflows can then apply that desired state and help correct configuration drift, depending on how the workflow is implemented.

The approach rests on four core principles:

- **Declarative configuration:** Configurations describe the end state rather than the steps to get there.
- **Version-controlled state:** Configuration definitions live in Git, creating a detailed history of changes.
- **Automated deployment:** When changes are approved, they can be deployed automatically or via a controlled sync step.
- **Continuous reconciliation:** Ongoing checks compare actual device state against the desired state defined in Git, surfacing or correcting drift between audit cycles.

Every change flows through Git commits and pull requests, creating a tamper-evident change record. When auditors ask who changed a configuration and why, the answer lives in Git history rather than scattered logs.

## How GitOps changes device management and MDM workflows

Most device management happens through an MDM console in what's often called a click-ops workflow: administrators click through settings and changes propagate across the fleet. This works well for day-to-day operations, but tracking who changed what over time requires digging through audit logs that may lack context about why changes happened. For teams managing macOS, Windows, and Linux fleets who want stronger change tracking, GitOps offers an alternative to this click-ops model.

MDM solutions that support GitOps workflows expand what's possible here. When your MDM provides Git-based configuration management, you get both: a console for visibility and quick changes, plus version-controlled configurations for auditability and automation.

GitOps adds version control to this model. Configuration profiles and software settings live as YAML or JSON files in Git repositories. When a configuration needs updating, someone creates a pull request, gets peer review, and merges the change. The MDM then applies configuration changes defined in Git to targeted devices when a sync workflow runs.

### Practical advantages for IT teams

This shift can bring several practical advantages for IT teams managing large fleets:

- **Version-controlled configurations:** Device configurations can be defined as reviewable files in Git. You can see exactly what changed between versions, who approved it, and roll back problematic changes with a simple git revert.
- **Collaborative change management:** Team members propose configuration changes through pull requests, creating natural review gates. You can still make quick changes in the console when needed, but major configuration changes go through version control.
- **Consistent multi-platform configurations:** Defining configurations in code lets you enforce the same security baselines across different operating systems using standardized workflows.
- **Disaster recovery coverage:** Many device management configurations can be reconstructed from repository contents, though certificates, tokens, and enrollment state stored outside Git still require separate backup processes.

### How GitOps workflows operate

These advantages compound when your team adopts consistent Git-based workflows. Engineers define configurations in version-controlled files, changes go through pull request reviews, and CI/CD pipelines or native integrations apply approved changes to targeted device groups. The device management system can then monitor compliance on an ongoing basis and trigger remediation when devices drift from the desired state.

For organizations already using infrastructure-as-code practices, extending Git-based workflows to device management creates consistency across teams.

## When GitOps makes sense for compliance and audits

GitOps is most valuable in environments with frequent configuration changes or active compliance requirements. Teams managing small fleets with infrequent changes may find console audit logs sufficient, and the overhead of pull request workflows hard to justify. For organizations facing regular compliance audits, auditors ask for evidence of change controls, access reviews, and configuration baselines. Reconstructing this history from console audit logs is possible but time-consuming. GitOps addresses this by building compliance evidence into everyday workflows.

### Immutable audit trails

When branch protection rules and signed commits are in place, Git's commit history uses cryptographic hashes to create a tamper-evident record of configuration changes. Each commit captures who made the change, when it happened, what specifically changed, and (through commit messages and pull request discussions) why it was necessary.

This provides strong evidence for change management requirements across multiple compliance frameworks.

For SOC 2 assessments, Git commit history satisfies Common Criteria around logical access controls and change management. The pull request approval chain demonstrates that changes went through proper review before deployment. For HIPAA technical safeguards, the commit history shows how configurations protecting electronic protected health information evolved over time.

### Continuous compliance validation

Rather than treating compliance as a periodic audit activity, GitOps lets you validate configurations continuously at two points. Pre-merge checks can evaluate proposed configurations against compliance rules and catch violations of security baselines during pull request review. Post-deployment, device-state policies can evaluate whether deployed configurations match the expected baselines on a recurring cadence, surfacing drift between audit cycles.

### Framework-specific alignment

Different compliance frameworks emphasize different controls, but GitOps provides evidence applicable across many of them simultaneously. Examples of how GitOps evidence can map to common controls include:

- **SOC 2 Type II:** Git commit history and pull request approvals provide evidence for change management and logical access controls requirements.
- **HIPAA:** Commit history documents how configurations affecting protected health information evolved over time, supporting access control and audit controls requirements.
- **FedRAMP:** Versioned declarative state and pull request workflows address configuration baseline and change control requirements, with commit logs satisfying audit event documentation needs.

Security teams managing multiple compliance frameworks often find that GitOps can reduce duplicated effort when they standardize change management through Git. A single Git commit history provides evidence for change management controls across SOC 2, HIPAA, and FedRAMP without maintaining separate documentation systems.

### Drift detection and remediation

Configuration drift occurs when actual device state diverges from documented baselines. This divergence creates compliance risk that can be difficult to detect through periodic manual checks. Manual changes made outside normal processes or devices missing updates can leave fleets in an inconsistent state that fails audit validation.

GitOps addresses drift through continuous policy evaluation paired with automated remediation. Policies running on a recurring schedule compare actual device state against expected baselines and surface deviations between audit cycles. When a device fails a policy, remediation workflows can run a corrective script or install required software automatically, rather than waiting for an administrator to investigate. This closes the loop from detection to fix and helps maintain more consistent configurations between assessment cycles.

## How GitOps principles apply to multi-platform MDM and enterprise compliance

Applying GitOps principles to device management requires a solution designed for Git-based configuration management. [Fleet provides GitOps workflows](https://fleetdm.com/fleet-gitops) for multi-platform device management, where configuration profiles and settings are defined as YAML files in Git repositories. Fleet supports managing configurations via UI, REST API, and GitOps. The key to its GitOps workflow is that the same configurations applied in the UI can also be defined in YAML. This approach gives your team declarative control over devices while maintaining the audit trails and version control that compliance frameworks require.

Many MDM platforms expose APIs you can script against. API access alone doesn't make GitOps viable, though, because teams still have to build and maintain the glue between Git and the MDM. Fleet provides declarative YAML configurations and a built-in GitOps execution engine, so the workflow runs without custom scripting.

### Declarative device configuration

Fleet's Git-based workflow lets you [define device configurations](https://fleetdm.com/docs/configuration/yaml-files) declaratively in YAML files stored in Git. Engineers propose changes through pull requests, teams review and approve modifications, and Fleet applies the approved configuration when your workflow runs. Devices receive updated settings when they next check in.

For macOS, you can define configuration profiles, OS update settings with deadlines and minimum versions, and scripts that run at enrollment. Windows and Linux support similar Git-based workflows for profiles, OS update management, scripts, and configuration deployment, even though not every operating system uses the same declarative device configuration model. Fleet's agent runs on each device and reports actual state back. Fleet can then reconcile configurations against the desired state in Git, whether the underlying OS uses a declarative or imperative configuration model.

### Enforcing security baselines across platforms

This declarative model extends to security baselines. Fleet can enforce disk encryption (FileVault on macOS, BitLocker on Windows), and you can manage that enforcement through UI, API, or GitOps workflows. Using Fleets and labels, these configurations can be scoped to specific device groups based on operating system type, department, or other criteria.

### Label-based targeting for staged rollouts

Fleet provides labels that let you group devices by operating system version, department, hardware type, or custom attributes. You can roll out new macOS profiles exclusively to engineering devices running a specific OS version, scope configurations to specific Fleets, or target devices based on custom definitions. This targeting is defined in your GitOps YAML files, making rollout strategies version-controlled. For compliance, your rollout scope is auditable directly from Git: which devices received which configuration, and when, alongside the configuration content itself.

### Maintaining compliance across large fleets

For organizations managing compliance across frameworks like SOC 2, FedRAMP, and PCI-DSS, Git-based approaches provide the audit trail and change control documentation these programs require. With Fleet's GitOps workflow, configuration changes are managed via Git with full attribution through commit history and pull request reviews.

Fleet Premium provides pre-built CIS Benchmark policies that administrators can import and apply, alongside custom Fleet Policies that evaluate device state on a configurable cadence. When a device fails a policy check, Fleet can automatically run a remediation script or install required software. Up to three retry attempts run before the device is flagged for manual review. This closes the loop from drift detection to remediation between audit cycles.

Because Fleet is open source, security and compliance teams can inspect how reconciliation, policy evaluation, and remediation logic work. That matters for audit defense in regulated industries. Fleet's REST API and webhook automations also let you feed compliance evidence into common GRC platforms and ticketing systems rather than maintaining a separate evidence pipeline.

Dashboards show which devices meet configured requirements and which have drifted from expected configurations, giving leadership confidence in security posture without periodic scrambles.

## Bring GitOps workflows to your device fleet

Modern device management can benefit from many of the same infrastructure-as-code practices that reshaped application deployment. GitOps workflows provide the audit trails, change controls, and version-controlled configurations that compliance frameworks demand while giving IT teams familiar tools for managing device configurations.

Fleet is an open-source device management solution that provides device visibility and vulnerability management across macOS, Windows, and Linux, with MDM capabilities extending to iOS, iPadOS, and Android.

Define Fleet Policies, configuration profiles, and update schedules in version-controlled YAML files, then let Fleet apply them across your device fleet through GitOps workflows. [Get a demo](https://fleetdm.com/contact) to see how GitOps-based device management works for your team.

## Frequently asked questions

### What is the difference between GitOps and legacy CI/CD?

Legacy CI/CD pipelines push changes to production by connecting external systems to infrastructure. GitOps inverts this model by shifting change management from click-ops and direct console changes to approved definitions stored in Git. Automated workflows then apply those changes from version-controlled files. This approach can improve security because teams manage changes through pull requests and controlled sync workflows instead of making ad hoc changes in production.

### How long does it take to implement GitOps for device management?

Implementation timelines vary based on fleet size and existing processes. Organizations typically start by migrating a subset of configurations to version control, then expand coverage over several weeks. Teams already comfortable with Git workflows often find the transition faster since the concepts translate directly from application deployment practices.

### What compliance frameworks does GitOps support?

GitOps provides audit evidence directly applicable to SOC 2, HIPAA, FedRAMP, and PCI-DSS. For SOC 2, Git commit history and pull request approvals satisfy change management and logical access controls requirements. For HIPAA, commit history demonstrates how configurations affecting protected health information evolved over time. For FedRAMP, versioned declarative state and pull request workflows address configuration baseline and change control requirements. Because a single Git commit history provides evidence across frameworks simultaneously, teams managing multiple compliance programs avoid maintaining separate documentation systems for each.

### Can I use GitOps with my current MDM solution?

GitOps principles apply best to MDMs that support Git-based, declarative configuration management rather than only API-driven scripting. Some MDM solutions offer limited API coverage that restricts what can be managed through code, so GitOps capabilities vary by vendor. Fleet supports native GitOps workflows where configurations are defined in YAML files and applied automatically when your workflow runs. [Explore Fleet's docs](https://fleetdm.com/docs) to see how the workflow operates, or [contact us](https://fleetdm.com/contact) to discuss whether it fits your environment.

<meta name="articleTitle" value="GitOps for device management: Audit trails, compliance, and configuration as code">
<meta name="authorFullName" value="Ashish Kuthiala">
<meta name="authorGitHubUsername" value="akuthiala">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-05-15">
<meta name="description" value=" Learn what GitOps is, how it changes device management through version control and automation & why compliance teams rely on immutable audit trails.">
