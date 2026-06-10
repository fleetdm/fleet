
*How Fleet's continuous evidence architecture maps to your framework, your workflow, and your team*

Audit preparation does not have to consume weeks of IT capacity. When compliance evidence is produced continuously as a byproduct of normal device management, the pre-audit scramble shrinks to a verification step.

[Part 1](https://fleetdm.com/articles/audit-ready-every-day-part-1) made the case that audit preparation is painful because of a structural mismatch: compliance frameworks demand continuous evidence that controls are operating effectively, but most endpoint management platforms produce point-in-time reports. We covered the three architectural principles that close this gap (live device state, continuous policy evaluation, and unified multi-platform coverage), the GitOps model that turns change management into a byproduct of normal workflow, and why speed is becoming a compliance dimension in its own right.

This part covers what that architecture looks like in practice. How it supports compliance with specific frameworks. What auditors actually look for in continuous evidence. How the audit preparation workflow changes. And what the teams involved experience day-to-day.

## The audit frameworks Fleet supports, and how

Different compliance frameworks have different specific requirements, but they share a common underlying need: verifiable evidence that specific controls are operating as described across the organization's device fleet. Fleet helps address each major framework with the same continuous monitoring foundation.

### SOC 2

SOC 2 audits evaluate whether an organization's security controls meet the AICPA's Trust Services Criteria. Every SOC 2 report covers the Security category (also called the Common Criteria); organizations can also include Availability, Processing Integrity, Confidentiality, or Privacy depending on the services they provide and the commitments they've made to customers. For IT teams, the endpoint-relevant controls fall primarily under Security and typically cover device encryption, software patching, access management, and monitoring.

Fleet addresses the endpoint-relevant controls directly. Encryption state is continuously monitored and recorded across every managed device. Patch status and software inventory are continuously maintained. Policy compliance, covering the specific configurations required by the organization's SOC 2 control set, is continuously evaluated and historically recorded.

Fleet's scope is the endpoint. It covers device encryption, patch status, configuration compliance, and software inventory. It does not cover network traffic logs, identity provider logs, application logs, or physical security, which other systems in the SOC 2 evidence chain provide. Fleet's job is to make the endpoint portion of that evidence continuous and complete.

When the SOC 2 auditor asks for evidence that "all endpoints are encrypted and policy compliant," the Fleet answer is not a manually assembled export. It is a continuous compliance record showing encryption state and policy compliance for every managed device throughout the audit period, with trend data showing any gaps and their remediation.

Auditors usually probe the mechanics. What is your evaluation interval? If a device falls out of compliance at 2:00 PM, when do you detect it? How long is historical data retained? Fleet evaluates policies on a configurable interval, hourly by default, so a device that drifts is caught at the next evaluation rather than at the next periodic scan. Detection is measured in minutes to an hour, not weeks. Historical compliance data covers the full audit period, with retention set by the organization's own storage and data policies.

### ISO 27001

ISO 27001 requires organizations to establish and maintain an information security management system (ISMS) supported by a defined set of security controls. The current version, ISO 27001:2022, organizes 93 reference controls in Annex A into four themes: organizational, people, physical, and technological. The endpoint-relevant controls span the organizational and technological themes and include asset inventory (A.5.9), configuration management (A.8.9), technical vulnerability management (A.8.8), and monitoring activities (A.8.16).

Fleet's continuously maintained device inventory satisfies the asset management control with live, accurate data rather than a spreadsheet that is always somewhat out of date. Configuration management evidence comes from Fleet's continuous policy evaluation. Patch management evidence comes from Fleet's software inventory and vulnerability tracking. Monitoring evidence comes from Fleet's query history and alert records.

For ISO 27001, the requirement to demonstrate a managed, documented ISMS is directly supported by Fleet's continuous operational record. Not a set of reports assembled at audit time, but an ongoing operational practice that produces audit evidence as a natural byproduct.

### HIPAA

HIPAA's Security Rule requires covered entities and business associates to implement technical safeguards protecting electronic protected health information, including access controls, audit controls, integrity controls, person or entity authentication, and transmission security. For endpoint management, the relevant requirements cover device encryption, access management, authentication policy enforcement, and audit logging.

Fleet's continuous monitoring of device encryption state, access control configuration, authentication policy enforcement, and audit logging configuration provides the technical evidence required by HIPAA's Security Rule. More importantly, Fleet's historical compliance records support HIPAA's requirement for ongoing monitoring and review, not just implementing controls but demonstrating that they are continuously operating.

### PCI-DSS

The current version, PCI-DSS 4.0.1, has been fully mandatory since March 31, 2025 and includes endpoint-relevant requirements covering malware protection, patch and vulnerability management, secure configurations, access control, and logging and monitoring. Fleet's continuous software inventory and vulnerability tracking provide the patch management evidence PCI-DSS requires. Policy compliance monitoring covers the secure configuration requirements. The continuous nature of Fleet's monitoring directly supports PCI-DSS 4.0.1's emphasis on continuous security as an operational practice rather than a point-in-time assessment.

### FedRAMP

FedRAMP authorization requires implementing NIST SP 800-53 Rev 5 security controls, and FedRAMP's Continuous Monitoring (ConMon) program, based on NIST SP 800-137, requires authorized cloud service providers to maintain ongoing visibility into their security posture, with monthly deliverables covering inventory updates, vulnerability scanning, and remediation tracking.

Fleet is not itself a FedRAMP-authorized service. What Fleet provides is the continuous endpoint evidence that underpins the configuration management, vulnerability management, and continuous monitoring controls a provider must demonstrate. And because Fleet can be self-hosted, including on-prem and air-gapped, teams can run it inside their own authorization boundary rather than depending on a separate vendor authorization. For organizations building toward or maintaining FedRAMP, that produces the kind of ongoing operational visibility ConMon expects, beyond what periodic scanning alone provides.

### NIST and CIS frameworks

The NIST Cybersecurity Framework (CSF) 2.0 provides guidance across six functions, with continuous monitoring central to the Detect function and configuration management addressed throughout Identify and Protect. CIS benchmarks specify prescriptive configuration baselines for operating systems and platforms. Both work best when device configuration state is continuously verified against the baseline rather than checked periodically. Fleet's pre-built CIS benchmark policies for macOS and Windows, combined with osquery-based verification of actual device state, produce the continuous, verifiable evidence organizations need to operate effectively under these frameworks.

### Working alongside your existing MDM

Fleet runs alongside Intune, Jamf, or other existing MDMs. The existing MDM continues to manage device configuration. Fleet adds an osquery-based observation layer that produces continuous control evidence, historical policy compliance records, and granular software inventory at a level of detail and frequency that traditional MDM reporting was not designed for. For organizations that need audit readiness immediately but aren't ready to switch device management vendors, this is the faster path. For organizations that ultimately choose to consolidate on Fleet, the transition happens on their own timeline, not under audit pressure.

## What auditors actually look for, and how Fleet delivers it

Understanding what auditors evaluate helps illustrate why Fleet's architecture is well-suited to audit support.

### Evidence of continuous monitoring

Auditors understand the difference between organizations that continuously monitor their security controls and organizations that run reports before audits and present them as continuous monitoring evidence. The tell is in the timestamps, the granularity of historical data, and whether the evidence shows any failures and their remediation, or suspiciously clean results that suggest point-in-time reporting.

Fleet's continuous policy evaluation produces the kind of evidence that demonstrates genuine continuous monitoring. The historical record shows compliance state over time, including failures when they occurred and remediation when it was applied. This evidence pattern, showing real operational history including imperfections and their correction, is more credible to auditors than a clean snapshot report.

### Completeness of coverage

Auditors want to know that the evidence covers the entire population of devices in scope, not a sample, not the devices that happened to be enrolled in the primary MDM, but every device that processes, stores, or transmits the data the compliance framework protects.

Fleet makes it straightforward to demonstrate complete population coverage by reconciling enrolled hosts against your source of truth for expected devices, such as Apple or Android enrollment records or your identity provider. A device that would otherwise fall through the cracks shows up as an enrollment gap rather than an unknown unknown. Auditors respond well to organizations that can account for every device in scope rather than presenting evidence for a subset and hoping the auditor doesn't ask about the rest.

### Precision of evidence

Auditors evaluating specific technical controls want evidence that reflects the actual state of the control being evaluated, not a proxy measure that might or might not correlate with the control's actual operation.

Fleet's osquery-based evidence is precisely about what it claims to be about. When a Fleet compliance report says disk encryption is enabled on a device, that claim is backed by an osquery query that returned the actual encryption state from the device. Not "an encryption policy was pushed to this device" but "the device's encryption state was queried and returned this result." This precision matters to auditors who understand the difference.

### Remediation evidence

Compliance frameworks don't require perfection. They require that when controls fail, failures are detected and remediated. Evidence of effective remediation, showing that the organization detected a control failure and corrected it in a reasonable timeframe, is often more valuable to auditors than evidence of a perfect compliance record that may not reflect real operational history.

Fleet's remediation tracking provides this evidence. When a device falls out of compliance, the failure is timestamped. When it is remediated, the return to compliance is timestamped. The remediation timeline, how long between failure detection and confirmed remediation, is visible and documentable. This evidence pattern demonstrates an effective compliance program rather than just a clean audit result.

## The audit preparation workflow, before and after Fleet

The operational difference between audit preparation with traditional tools and with Fleet is significant enough to walk through concretely.

### Before Fleet: the six-week scramble

Week one: The compliance team leading the audit identifies the controls and specific evidence requirements for the upcoming audit. The list goes to the IT team. The IT team begins identifying which systems contain the relevant data.

Week two: Data collection begins. MDM exports for device inventory. Vulnerability scanner exports for patch status. Manual spreadsheets reconciling device lists that don't agree across systems. Discovery that several devices appear in one system but not another and nobody knows why.

Week three: Data normalization. The exports from different tools use different device identifiers, different field names, different date formats. A spreadsheet wizard spends three days building a unified view. Several devices fall out of scope for unclear reasons.

Week four: Gap discovery. The reconciled inventory reveals devices that don't appear to be enrolled in any management system. Urgent outreach to device owners. Some gaps are resolved. Others remain unexplained.

Week five: Evidence package assembly. Reports are formatted, annotated, and organized into the evidence package structure the auditors expect. Last-minute questions from the compliance officer generate additional data requests.

Week six: Final review and delivery. The evidence package reflects device state as of whatever date the exports were pulled. It is presented as representing continuous compliance, which it does not actually demonstrate.

Throughout the entire window: late nights, frustrated engineers, escalations to management, and the lingering uncertainty that the evidence package will survive auditor scrutiny.

### After Fleet: continuous readiness

The audit window opens. The compliance officer requests the evidence package. The IT team opens Fleet's compliance dashboard.

Fleet's policy compliance history for the audit period is available immediately, showing compliance state for every managed device across every relevant policy check, with trend data and remediation records. The device inventory is current and comprehensive. The software inventory covers every managed device. The historical record shows continuous monitoring throughout the audit period.

The evidence package is assembled in hours rather than weeks. The data comes from one system rather than being reconciled across multiple. The historical record demonstrates continuous monitoring rather than point-in-time assessment. The remediation timeline for any compliance failures is documented.

Underneath that speed are three concrete mechanics. Policies evaluate every hour, with alerting when a policy fails or triggers an automation, so compliance state is current rather than reconstructed. Every policy lives in a YAML file in Git, where changes require pull request approval and the audit trail already includes author, reviewer, timestamp, and commit message. And the evidence comes from a single console covering macOS, Windows, Linux, iOS, and Android, with osquery providing live state verification on macOS, Windows, and Linux and MDM compliance state serving as the evidence source on iOS and Android.

The audit preparation that used to consume weeks of IT team capacity now takes a day. The evidence is more credible because it is more accurate and more continuous. The IT team that used to dread audit season now approaches it as a routine operational activity.

## Fleet's audit evidence capabilities in detail

### Real-time and historical device inventory

Fleet maintains a continuously updated inventory of every enrolled device, including hardware details, OS version, assigned user, enrollment date, and management state. This inventory is the foundation of audit evidence for asset management controls, and it is always current.

For auditors requiring evidence of device inventory state during the audit period, Fleet's historical records support reconstructing inventory state from continuously captured device data rather than relying on retrospective estimates.

### Policy compliance history

Every Fleet policy evaluation is recorded with a timestamp. The full history of pass/fail status for every device against every policy is queryable. For audits requiring evidence that specific controls operated continuously throughout a period, this history provides the documentation without requiring any special evidence collection process.

### Software inventory and vulnerability tracking

Fleet's continuously maintained software inventory supports patch management controls across all major audit frameworks. The inventory reflects what is actually installed on devices, not what was deployed according to software management records, but what osquery finds present on the device. Vulnerability identification and remediation tracking provide the patch management evidence that auditors require.

### Query history and audit logs

Fleet's query history documents every query run against the fleet: when it was run, by whom, and against which devices. Query results are captured in Fleet's standard reporting and policy compliance history. This audit trail supports access control and monitoring requirements in multiple compliance frameworks, demonstrating that the IT team actively monitors fleet state rather than relying on passive collection.

### Integration with GRC platforms

Fleet integrates with Vanta, sending host and user data so compliance evidence is collected automatically rather than exported by hand at audit time. The Vanta integration covers macOS and Windows hosts and is available on Fleet Premium. For other platforms, and for other GRC systems, the same evidence is available through Fleet's REST API and webhooks, so it can flow into the compliance management system where auditors expect it without manual transformation.

## The auditor conversation that changes

Audit conversations with IT teams that run Fleet have a different character than those with teams relying on traditional tools. The difference is visible from the first evidence submission.

When an auditor asks "how do you verify that disk encryption is enabled on all endpoint devices," the Fleet-equipped IT team answers: "Through continuous osquery-based queries that run on every managed device on a scheduled interval. Here is the policy definition, here is the query it executes, here is the historical compliance record for the audit period showing per-device pass/fail status, and here is the remediation record for any failures that occurred."

That answer communicates several things at once. The verification method is specific and technical, not vague and aspirational. The monitoring is continuous, not periodic. The evidence covers every device, not a sample. The failures and their remediation are documented, demonstrating an effective compliance program rather than just a clean record.

The result is a more efficient audit process. Not because Fleet makes difficult questions easier to answer, but because Fleet ensures the difficult questions have already been answered through continuous operational practice before the audit begins.

## The cross-team dividend: when audit readiness benefits everyone

Fleet's audit readiness capabilities don't just benefit the IT team during audit season. They create a continuous operational improvement that benefits every team involved in the compliance program year-round.

The security team that previously received compliance evidence only at audit time now has continuous visibility into fleet compliance state. Gaps are visible as they occur. Remediation can be prioritized based on current compliance data rather than the last audit finding.

The compliance officer who previously managed a frantic pre-audit evidence collection process now manages a continuously maintained evidence base. Compliance posture is visible at any point, not just after the evidence scramble completes.

The CISO and CIO who previously faced uncertainty about whether the evidence package would survive scrutiny now have confidence grounded in continuous monitoring data rather than optimism about point-in-time exports.

The executive team that previously received a "we passed the audit" report now receives continuous compliance metrics that reflect the organization's actual security posture. The audit becomes a verification of what is already known rather than a discovery of what was true.

## The bottom line: audit season ends when evidence collection never stops

The annual audit scramble is not inevitable. It is a symptom of an endpoint management approach that treats compliance evidence as something to be assembled rather than something to be maintained.

Fleet treats compliance evidence as a continuous operational product. Every policy evaluation, every device query, every software inventory update, every compliance state change is recorded and available. The evidence auditors need to verify control operation is produced continuously as a byproduct of normal fleet management, not assembled manually in the weeks before an audit.

This changes how IT teams relate to audits. In the periodic model, compliance is demonstrated to auditors at intervals, and between audits the actual compliance posture is uncertain. In the continuous model, compliance is maintained and measured at all times. Audits become verification checkpoints in an ongoing program rather than high-stakes tests with uncertain outcomes.

Organizations that operate continuous compliance practices are more secure than organizations that operate compliance events, because the controls they claim to have are continuously verified rather than periodically reported. The audit becomes a reflection of operational reality.

Audit season ends when you stop treating audits as seasons.

*Missed [part 1](https://fleetdm.com/articles/audit-ready-every-day-part-1)? It covers the architectural foundations of continuous audit readiness: why traditional tools produce the wrong kind of evidence, and what changes when configuration, monitoring, and change management are designed for continuous compliance from the start.*

*Fleet's continuous compliance monitoring, historical policy records, and unified multi-platform inventory make audit preparation a routine operational task rather than a recurring crisis.* [*See how Fleet supports your compliance framework*](https://fleetdm.com/guides/stay-on-course-with-your-security-compliance-goals) *or* [*talk to us*](https://fleetdm.com/) *about your audit requirements.*

*Fleet runs its own compliance program on these same principles. See live control status, SOC 2 status, and third-party penetration test reports at [trust.fleetdm.com](https://trust.fleetdm.com).*

<meta name="articleTitle" value="Audit-ready every day, part 2: continuous compliance in practice">
<meta name="authorFullName" value="Ashish Kuthiala">
<meta name="authorGitHubUsername" value="akuthiala">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-06-02">
<meta name="description" value="How Fleet's continuous evidence architecture maps to SOC 2, ISO 27001, HIPAA, PCI-DSS, FedRAMP, NIST, and CIS, and what auditors actually look for.">
