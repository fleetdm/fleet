# Security policies

## Information security policy and acceptable use policy

This Information Security Policy is intended to protect Fleet Device Management Inc's employees, contractors, partners, customers, and the company from illegal or damaging actions by individuals, either knowingly or unknowingly.

Internet/Intranet/Extranet-related systems, including but not limited to computer equipment, software, operating systems, storage media, network accounts providing electronic mail, web browsing, and file transfers, are the property of Fleet Device Management Inc. These systems are to be used for business purposes in serving the interests of the company, and of our clients and customers in the course of normal operations.

Effective security is a team effort involving the participation and support of every Fleet Device Management Inc employee or contractor who deals with information and/or information systems. It is the responsibility of every team member to read and understand this policy and conduct their activities accordingly.

All Fleet employees and long-term collaborators are expected to read and electronically sign the *acceptable use of end-user computing* policy as well as to be aware of the others and consult them as needed to make sure systems built and used are done in a compliant manner.

### Acceptable use of end-user computing
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

| Policy owner   | Effective date |
| -------------- | -------------- |
| @GuillaumeRoss | 2022-06-01     |

Fleet requires all workforce members to comply with the following acceptable use requirements and procedures, such as

1. the use of Fleet computing systems is subject to monitoring by Fleet IT and/or Security teams.

2. Fleet team members must not leave computing devices (including laptops and smart devices) used for business purposes, including company-provided and BYOD devices, unattended in public.

3. device encryption must be enabled for all mobile devices accessing company data, such as whole-disk encryption for all laptops.

4. using only legal software with a valid license installed through the internal "app store" or trusted sources. Well-documented open source software can be used. If in doubt, ask in *#g-security*.  

5. avoiding sharing credentials. Secrets must be stored safely, using features such as GitHub secrets. For accounts and other sensitive data that need to be shared, use the company-provided password manager.

6. sanitizing and removing any sensitive or confidential information prior to posting. At Fleet, we are public by default. Sensitive information from logs, screenshots, or other types of data (memory dumps, for example),should not be shared.

7. anti-malware or equivalent protection and monitoring must be installed and enabled on all endpoint systems that may be affected by malware, including workstations, laptops, and servers.

8. it being strictly forbidden to download or store any secrets used to sign Orbit installer updates on end-user computing devices, including laptops, workstations, and mobile devices.

9. only allowing company-owned and managed computers to connect directly to Fleet autoupdater production environments.

10. enforcing the policy that Fleet team members must not let anyone else use Fleet-provided and managed workstations unsupervised, including family members and support personnel of vendors. Use screen sharing instead of allowing them to access your system directly.

11. Requiring device's operating system must be kept up to date. Fleet-managed systems will receive prompts for updates to be installed, and BYOD devices are to be updated by the team member using them or they might lose access. 

12. Requiring team members must not store sensitive data on portable storage.

13. Not allowing the use of Fleet company accounts on "shared" computers, such as hotel kiosk systems, is strictly prohibited.

## Access control policy
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

| Policy owner   | Effective date |
| -------------- | -------------- |
| @GuillaumeRoss | 2022-06-01     |

Fleet requires all workforce members to comply with the following acceptable use requirements and procedures, such that:

1. Access to all computing resources, including servers, end-user computing devices, network equipment, services, and applications, must be protected by strong authentication, authorization, and auditing.

2. Interactive user access to production systems must be associated with an account or login unique to each user.

3. All credentials, including user passwords, service accounts, and access keys, must meet the length, complexity, age, and rotation requirements defined in Fleet security standards.

4. Use a strong password and two-factor authentication (2FA) whenever possible to authenticate to all computing resources (including both devices and applications).

5. 2FA is required to access any critical system or resource, including but not limited to resources in Fleet production environments.

6. Unused accounts, passwords, and access keys must be removed within 30 days.

7. A unique access key or service account must be used for different applications or user access.

8. Authenticated sessions must time out after a defined period of inactivity.

### Access authorization and termination

Fleet policy requires that:

1. access authorization shall be implemented using role-based access control (RBAC) or a similar mechanism.

2. standard access based on a user's job role may be pre-provisioned during employee onboarding. All subsequent access requests to computing resources must be approved by the requestor’s manager prior to granting and provisioning of access.

3. access to critical resources, such as production environments, must be approved by the security team in addition to the requestor’s manager.

4. access must be reviewed regularly and revoked if no longer needed.

5. upon the termination of employment, all system access must be revoked, and user accounts terminated within 24-hours or one business day, whichever is shorter.

6. all system access must be reviewed at least annually and whenever a user's job role changes.

### Shared secrets management

Fleet policy requires that:

1. use of shared credentials/secrets must be minimized.

2. if required by business operations, secrets/credentials must be shared securely and stored in encrypted vaults that meet the Fleet data encryption standards.

### Privileged access management

Fleet policy requires that:

1. automation with service accounts must be used to configure production systems when technically feasible.

2. use of high privilege accounts must only be performed when absolutely necessary.

## Asset management policy
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

| Policy owner   | Effective date |
| -------------- | -------------- |
| @GuillaumeRoss | 2022-06-01     |

You can't protect what you can't see. Therefore, Fleet must maintain an accurate and up-to-date inventory of its physical and digital assets.

Fleet policy requires that:

1. IT and/or security must maintain an inventory of all critical company assets, both physical and logical.

2. All assets should have identified owners and a risk/data classification tag.

3. All company-owned computer purchases must be tracked.

## Business continuity and disaster recovery policy
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

| Policy owner   | Effective date |
| -------------- | -------------- |
| @GuillaumeRoss | 2022-06-01     |

The Fleet business continuity and disaster recovery plan establishes procedures to recover Fleet following a disruption resulting from a disaster. 

Fleet policy requires that:

1. A plan and process for business continuity and disaster recovery (BCDR), will be defined and documented including the backup and recovery of critical systems and data,.

2. BCDR shall be simulated and tested at least once a year. 

3. Security controls and requirements will be maintained during all BCDR activities.

### Business continuity plan

#### Line of Succession

The following order of succession to make sure that decision-making authority for the Fleet Contingency Plan is uninterrupted. The Chief Executive Officer (CEO) is responsible for ensuring the safety of personnel and the execution of procedures documented within this Fleet Contingency Plan. The CTO is responsible for the recovery of Fleet technical environments. If the CEO or Head of Engineering cannot function as the overall authority or choose to delegate this responsibility to a successor, the board of directors shall serve as that authority or choose an alternative delegate. 

### Response Teams and Responsibilities

The following teams have been developed and trained to respond to a contingency event affecting Fleet infrastructure and systems.

1. **Infrastructure** is responsible for recovering the Fleet automatic update service hosted environment. The team includes personnel responsible for the daily IT operations and maintenance. The team reports to the CTO.

2. **People Ops** is responsible for ensuring the physical safety of all Fleet personnel and coordinating the response to incidents that could impact it. Fleet has no physical site to recover. The team reports to the CEO.

3. **Security** is responsible for assessing and responding to all cybersecurity-related incidents according to Fleet Incident Response policy and procedures. The security team shall assist the above teams in recovery as needed in non-cybersecurity events. The team leader is the Head of Security.

Members of the above teams must maintain local copies of the contact information of the BCDR succession team. Additionally, the team leads must maintain a local copy of this policy in the event Internet access is not available during a disaster scenario.

All executive leadership shall be informed of any and all contingency events.

Current Fleet continuity leadership team members include the Head of Security, CEO, and CTO.

### General Disaster Recovery Procedures

#### Notification and Activation Phase

This phase addresses the initial actions taken to detect and assess the damage inflicted by a disruption to Fleet Device Management or the Fleet automatic updater service. Based on the assessment of the Event, sometimes, according to the Fleet Incident Response Policy, the Contingency Plan may be activated by either the CEO or CTO.  The Contingency Plan may also be triggered by the Head of Security in the event of a cyber disaster.

The notification sequence is listed below:

* The first responder is to notify the CTO. All known information must be relayed.
* The CTO is to contact the Response Teams and inform them of the event. The CTO or delegate is responsible to beginning the assessment procedures.
* The CTO is to notify team members and direct them to complete the assessment procedures outlined below to determine the extent of the issue and estimated recovery time. 
* The Fleet Contingency Plan is to be activated if one or more of the following criteria are met:

    * Fleet automatic update service will be unavailable for more than 48 hours.
    * Cloud infrastructure service is damaged and will be unavailable for more than 24 hours.
    * Other criteria, as appropriate and as defined by Fleet.

* If the plan is to be activated, the CTO is to notify and inform team members of the event details.
* Upon notification from the CTO, group leaders and managers must notify their respective teams. Team members are to be informed of all applicable information and prepared to respond and relocate if necessary.
* The CTO is to notify the remaining personnel and executive leadership on the general status of the incident.
* Notification can be via Slack, email, or phone.
* The CTO posts a blog post explaining that the service is down and recovery is in progress.

#### Reconstitution Phase

This section discusses activities necessary for restoring full Fleet automatic updater service operations at the original or new site. The goal is to restore full operations within 24 hours of a disaster or outage. The goal is to provide a seamless transition of operations.

1. Contact Partners and Customers affected to begin initial communication - CTO
2. Assess damage to the environment - Infrastructure
3. Create a new production environment using new environment bootstrap automation - Infrastructure
4. Make sure secure access to the new environment - Security
5. Begin code deployment and data replication using pre-established automation - DevOps
6. Test new environment and applications using pre-written tests - DevOps
7. Test logging, security, and alerting functionality - DevOps and Security
8. Assure systems and applications are appropriately patched and up to date -DevOps
9. Update DNS and other necessary records to point to the new environment - DevOps
10. Update Partners and Customers affected through established channels - DevOps

#### Plan Deactivation

If the Fleet automatic updater environment has been restored, the continuity plan can be deactivated. If the disaster impacted the company and not the service or both, make sure that any leftover systems created temporarily are destroyed.

## Data management policy
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

This policy outlines the requirements and controls/procedures Fleet has implemented to manage the end-to-end data lifecycle, from data creation/acquisition to retention and deletion.

Additionally, this policy outlines requirements and procedures to create and maintain retrievable exact copies of electronically protected health information(ePHI), PII, and other critical customer/business data.

Data backup is an important part of the day-to-day operations of Fleet. To protect the confidentiality, integrity, and availability of sensitive and critical data, both for Fleet and Fleet Customers, complete backups are done daily to assure that data remains available when needed and in case of a disaster.

Fleet policy requires that:

1. Data should be classified at the time of creation or acquisition.

2. Fleet must maintain an up-to-date inventory and data flows mapping of all critical data.

3. All business data should be stored or replicated to a company-controlled repository.

4. Data must be backed up according to the level defined in Fleet data classification.

5. Data backup must be validated for integrity.

6. The data retention period must be defined and comply with any and all applicable regulatory and contractual requirements.  More specifically,

  * data and records belonging to Fleet platform customers must be retained
    per Fleet product terms and conditions and/or specific contractual
    agreements.

7. By default, all security documentation and audit trails are kept for a minimum of seven years unless otherwise specified by Fleet data classification, specific regulations, or contractual agreement.

### Data Classification Model

Fleet defines the following four data classifications:

* **Critical**
* **Confidential**
* **Internal**
* **Public**

As Fleet is an open company by default, most of our data falls into **public**.

#### Definitions and Examples

**Critical** data includes data that must be protected due to regulatory requirements, privacy, and/or security sensitivities.

Unauthorized disclosure of critical data may result in major disruption to business operations, significant cost, irreparable reputation damage, and/or legal prosecution of the company.

External disclosure of critical data is strictly prohibited without an approved process and agreement in place.

*Example Critical Data Types* include

* PII (personal identifiable information)
* ePHI (electronically protected health information)
* Production security data, such as
    - Production secrets, passwords, access keys, certificates, etc.
    - Production security audit logs, events, and incident data


**Confidential** and proprietary data represents company secrets and is of significant value to the company.

Unauthorized disclosure may result in disruption to business operations and loss of value.

Disclosure requires the signing of NDA and management approval.

*Example Confidential Data Types* include

* Business plans
* Employee/HR data
* News and public announcements (pre-announcement)
* Patents (pre-filing)
* Non-production security data, including
    - Non-prod secrets, passwords, access keys, certificates, etc.
    - Non-prod security audit logs, events, and incident data

**Internal** data contains information used for internal operations.

Unauthorized disclosure may cause undesirable outcomes to business operations.

Disclosure requires management approval.  NDA is usually required but may be waived on a case-by-case basis.

**Public** data is Information intended for public consumption. Although
non-confidential, the integrity and availability of public data should be
protected.

*Example Internal Data Types* include

* Fleet source code.
* news and public announcements (post-announcement).
* marketing materials.
* product documentation.
* content posted on the company website(s) and social media channel(s).

#### Data Handling Requirements Matrix

Requirements for data handling, such as the need for encryption and the duration of retention, are defined according to the Fleet data classification.

| Data             | Labeling or Tagging | Segregated Storage | Endpoint Storage | Encrypt At Rest | Encrypt In Transit | Encrypt In Use | Controlled Access | Monitoring | Destruction at Disposal | Retention Period | Backup Recovery |
|------------------|---------------------|--------------------|------------------|-----------------|--------------------|----------------|-------------------|------------|------------------------|------------------|-----------------|
| **Critical**     | Required            | Required           | Prohibited       | Required        | Required           | Required       | Access is blocked to end users by default; Temporary access for privileged users only | Required   | Required   | seven years for audit trails; Varies for customer-owned data† | Required   |
| **Confidential** | Required            | N/R                | Allowed          | Required        | Required           | Required       | All access is based on need-to-know | Required   | Required   | seven years for official documentation; Others vary based on business need | Required   |
| **Internal**     | Required            | N/R                | Allowed          | N/R             | N/R                | N/R            | All employees and contractors (read); Data owners and authorized individuals (write) | N/R | N/R | Varies based on business need | Optional   |
| **Public**       | N/R                 | N/R                | Allowed          | N/R             | N/R                | N/R            | Everyone (read); Data owners and authorized individuals (write) | N/R     | N/R     | Varies based on business need | Optional   |

N/R = Not Required

† Customer-owned data is stored for as long as they remain as a Fleet customer, or as required by regulations, whichever is longer. Customers may request their data to be deleted at any time; unless retention is required by law.

Most Fleet data is **public** yet retained and backed up not due to our data handling requirements but simply business requirements.


## Encryption policy
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

| Policy owner   | Effective date |
| -------------- | -------------- |
| @GuillaumeRoss | 2022-06-01     |

Fleet requires all workforce members to comply with the encryption policy, such that:

1. The storage drives of all Fleet-owned workstations must be encrypted and enforced by the IT and/or security team.

2. Confidential data must be stored in a manner that supports user access logs.

3. All Production Data at rest is stored on encrypted volumes.

4. Volume encryption keys and machines that generate volume encryption keys are protected from unauthorized access. Volume encryption key material is protected with access controls such that the key material is only accessible by privileged accounts.

5. Encrypted volumes use strong cipher algorithms, key strength, and key management process as defined below.

6. Data is protected in transit using recent TLS versions with ciphers recognized as secure.

### Local disk/volume encryption

Encryption and key management for local disk encryption of end-user devices follow the defined best practices for Windows, macOS, and Linux/Unix operating systems, such as Bitlocker and FileVault. 

### Protecting data in transit

1. All external data transmission is encrypted end-to-end. This includes, but is not limited to, cloud infrastructure and third-party vendors and applications.

2. Transmission encryption keys and systems that generate keys are protected from unauthorized access. Transmission encryption key materials are protected with access controls and may only be accessed by privileged accounts.

3. TLS endpoints must score at least an "A" on SSLLabs.com.

4. Transmission encryption keys are limited to use for one year and then must be regenerated.

## Human resources security policy
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

| Policy owner   | Effective date |
| -------------- | -------------- |
| @GuillaumeRoss | 2022-06-01     |


Fleet is committed to ensuring all workforce members participate in security and compliance in their roles at Fleet. We encourage self-management and reward the right behaviors. 

Fleet policy requires all workforce members to comply with the HR Security Policy.

Fleet policy requires that:

1. Background verification checks on candidates for employees and contractors with production access to the Fleet automatic updater service must be carried out in accordance with relevant laws, regulations, and ethics. These checks should be proportional to the business requirements, the classification of the information to be accessed, and the perceived risk.

2. Employees, contractors, and third-party users must agree to and sign the terms and conditions of their employment contract and comply with acceptable use.

3. Employees will perform an onboarding process that familiarizes them with the environments, systems, security requirements, and procedures that Fleet already has in place. Employees will also have ongoing security awareness training that is audited.

4. Employee offboarding will include reiterating any duties and responsibilities still valid after terminations, verifying that access to any Fleet systems has been removed, and ensuring that all company-owned assets are returned.

5. Fleet and its employees will take reasonable measures to make sure no sensitive data is transmitted via digital communications such as email or posted on social media outlets.

6. Fleet will maintain a list of prohibited activities that will be part of onboarding procedures and have training available if/when the list of those activities changes.

7. A fair disciplinary process will be used for employees suspected of committing security breaches. Fleet will consider multiple factors when deciding the response, such as whether or not this was a first offense, training, business contracts, etc. Fleet reserves the right to terminate employees in the case of severe cases of misconduct.

8. Fleet will maintain a reporting structure that aligns with the organization's business lines and/or individual's functional roles. The list of employees and reporting structure must be available to [all employees](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0).

9. Employees will receive regular feedback and acknowledgment from their managers and peers. Managers will give constant feedback on performance, including but not limited to during regular one-on-one meetings.

10. Fleet will publish job descriptions for available positions and conduct interviews to assess a candidate's technical skills as well as soft skills prior to hiring.

11. Background checks of an employee or contractor must be performed by operations and/or the hiring team before we grant the new employee or contractor access to the Fleet automatic updater environment.

12. A list of employees and contractors will be maintained, including their titles and managers, and made available to everyone internally.

13. An [anonymous](https://docs.google.com/forms/d/e/1FAIpQLSdv2abLfCUUSxFCrSwh4Ou5yF80c4V2K_POoYbHt3EU1IY-sQ/viewform?vc=0&c=0&w=1&flr=0&fbzx=4276110450338060288) form to report unethical behavior will be provided to employees.

## Incident response policy
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/). Based on the SANS incident response process.*

Fleet policy requires that:

1. All computing environments and systems must be monitored in accordance with Fleet policies and procedures specified in the Fleet handbook.
2. Alerts must be reviewed to identify security incidents.
3. Incident response procedures are invoked upon discovery of a valid security incident.
4. Incident response team and management must comply with any additional requests by law enforcement in the event of a criminal investigation or national security, including but not limited to warranted data requests, subpoenas, and breach notifications.

### Incident response plan
#### Security Incident Response Team (SIRT)

The Security Incident Response Team (SIRT) is responsible for

* reviewing analyzing, and logging all received reports and tracking their statuses.
* performing investigations, creating and executing action plans, and post-incident activities.
* collaboration with law enforcement agencies.

Current members of the Fleet SIRT:

* Head of Security
* CTO
* CEO

#### Incident Management Process
Fleet's incident response classifies security-related events into the following categories:

* **Events** - Any observable computer security-related occurrence in a system
  or network with a negative consequence. Examples:

    * Hardware component failing, causing service outages.
    * Software error causing service outages.
    * General network or system instability.

* **Precursors** - A sign that an incident may occur in the future. Examples:

    * Monitoring system showing unusual behavior.
    * Audit log alerts indicated several failed login attempts.
    * Suspicious emails that target specific Fleet staff members with
      administrative access to production systems.
    * Alerts raised from a security control source based on its monitoring
      policy, such as

        - Google Workspace (user authentication activities)
        - Fleet (internal instance)
        - Syslog events from servers

* **Indications** - A sign that an incident may have occurred or may be
  occurring at the present time. Examples:

    * Alerts for modified system files or unusual system accesses.
    * Antivirus alerts for infected files or devices.
    * Excessive network traffic directed at unexpected geographic locations.

* **Incidents** - A confirmed attack/indicator of compromise or a validated
  violation of computer security policies or acceptable use policies, often
  resulting in data breaches. Examples:

    * Unauthorized disclosure of sensitive data
    * Unauthorized change or destruction of sensitive data
    * A data breach accomplished by an internal or external entity
    * A Denial-of-Service (DoS) attack causing a critical service to become
      unreachable

Fleet employees must report any unauthorized or suspicious activity seen on
production systems or associated with related communication systems (such as
email or Slack). In practice, this means keeping an eye out for security events
and letting the Security team know about any observed precursors or indications
as soon as they are discovered.

Incidents of a severity/impact rating higher than **MINOR** shall trigger the response process.

#### I - Identification and Triage

1. Immediately upon observation, Fleet members report suspected and known
   Events, Precursors, Indications, and Incidents in one of the following ways:

    1. Direct report to management, the Head of Security, CTO, CEO, or
       other
    2. Email
    3. Phone call
    4. Slack

2. The individual receiving the report facilitates the collection of additional
   information about the incident, as needed, and notifies the Head of Security
   (if not already done).

3. The Head of Security determines if the issue is an Event, Precursor,
   Indication, or Incident.

   1. If the issue is an event, indication, or precursor, the Head of Security
      forwards it to the appropriate resource for resolution.

      1. Non-Technical Event (minor infringement): the Head of Security of the
         designee creates an appropriate issue in GitHub and further investigates
         the incident as needed.
      2. Technical Event: Assign the issue to a technical resource for
         resolution. This resource may also be a contractor or outsourced
         technical resource in the event of a lack of resource or expertise in
         the area.

   2. If the issue is a security incident, the Head of Security activates the
      Security Incident Response Team (SIRT) and notifies senior leadership by
      email.

       1. If a non-technical security incident is discovered, the SIRT completes
          the investigation, implements preventative measures, and resolves the
          security incident.
       2. Once the investigation is completed, progress to Phase V, Follow-up.
       3. If the issue is a technical security incident, commence to Phase II:
          Containment.
       4. The Containment, Eradication, and Recovery Phases are highly
          technical. It is important to have them completed by a highly
          qualified technical security resource with oversight by the SIRT team.
       5. Each individual on the SIRT and the technical security resource
          document all measures taken during each phase, including the start and
          end times of all efforts.
       6. The lead member of the SIRT team facilitates the initiation of an Incident
          ticket in GitHub Security Project and documents all findings and details
          in the ticket.

           * The intent of the Incident ticket is to provide a summary of all
             events, efforts, and conclusions of each Phase of this policy and
             procedures.
           * Each Incident ticket should contain sufficient details following
             the [SANS Security Incident Forms templates](https://www.sans.org/score/incident-forms/),
             as appropriate.

3. The Head of Security, Privacy Officer, or Fleet representative appointed
   notifies any affected Customers and Partners. If no Customers and Partners
   are affected, notification is at the discretion of the Security and Privacy
   Officer.

4. In the case of a threat identified, the Head of Security is to form a team to
   investigate and involve necessary resources, both internal to Fleet and
   potentially external.

#### II - Containment (Technical)

In this Phase, Fleet's engineers and security team attempt to contain the
security incident. It is essential to take detailed notes during the
security incident response process. This provides that the evidence gathered
during the security incident can be used successfully during prosecution, if
appropriate.

1. Review any information that has been collected by the Security team or any
   other individual investigating the security incident.
2. Secure the blast radius (i.e., a physical or logical network perimeter or
   access zone).
3. Perform the following forensic analysis preparation, as needed:

    1. Securely connect to the affected system over a trusted connection.
    2. Retrieve any volatile data from the affected system.
    3. Determine the relative integrity and the appropriateness of backing the
       system up.
    4. As necessary, take a snapshot of the disk image for further forensic,
       and if appropriate, back up the system.
    5. Change the password(s) to the affected system(s).
    6. Determine whether it is safe to continue operations with the affected
       system(s).
    7. If it is safe, allow the system to continue to functioning; and move to
       Phase V, Post Incident Analysis and Follow-up.
    8. If it is NOT safe to allow the system to continue operations, discontinue
       the system(s) operation and move to Phase III, Eradication.
    9. The individual completing this phase provides written communication to
       the SIRT.

4. Complete any documentation relative to the security incident containment on the
   Incident ticket, using
   [SANS IH Containment Form](https://www.sans.org/media/score/incident-forms/IH-Containment.pdf)
   as a template.
5. Continuously apprise Senior Management of progress.
6. Continue to notify affected Customers and Partners with relevant updates as
   needed.

#### III - Eradication (Technical)

The Eradication Phase represents the SIRT's effort to remove the cause and the
resulting security exposures that are now on the affected system(s).

1. Determine symptoms and cause related to the affected system(s).
2. Strengthen the defenses surrounding the affected system(s), where possible (a
   risk assessment may be needed and can be determined by the Head of Security).
   This may include the following:

    1. An increase in network perimeter defenses.
    2. An increase in system monitoring defenses.
    3. Remediation ("fixing") any security issues within the affected system,
       such as removing unused services/general host hardening techniques.

3. Conduct a detailed vulnerability assessment to verify all the holes/gaps that
   can be exploited are addressed.

    1. If additional issues or symptoms are identified, take appropriate
       preventative measures to eliminate or minimize potential future
       compromises.

4. Update the Incident ticket with Eradication details, using
   [SANS IH Eradication Form](https://www.sans.org/media/score/incident-forms/IH-Eradication.pdf)
   as a template.
5. Update the documentation with the information learned from the vulnerability
   assessment, including the cause, symptoms, and the method used to fix the
   problem with the affected system(s).
6. Apprise Senior Management of the progress.
7. Continue to notify affected Customers and Partners with relevant updates as
   needed.
8. Move to Phase IV, Recovery.

#### IV - Recovery (Technical)

The Recovery Phase represents the SIRT's effort to restore the affected
system(s) to operation after the resulting security exposures, if any, have
been corrected.

1. The technical team determines if the affected system(s) have been changed in
   any way.

    1. If they have, the technical team restores the system to its proper,
       intended functioning ("last known good").
    2. Once restored, the team validates that the system functions the way it
       was intended/had functioned in the past. This may require the involvement
       of the business unit that owns the affected system(s).
    3. If the operation of the system(s) had been interrupted (i.e., the system(s)
       had been taken offline or dropped from the network while triaged),
       restart the restored and validated system(s) and monitor for behavior.
    4. If the system had not been changed in any way but was taken offline
       (i.e., operations had been interrupted), restart the system and monitor
       for proper behavior.
    5. Update the documentation with the detail that was determined during this
       phase.
    6. Apprise Senior Management of progress.
    7. Continue to notify affected Customers and Partners with relevant updates
       as needed.
    8. Move to Phase V, Follow-up.

#### V - Post-Incident Analysis (Technical and Non-Technical)

The Follow-up phase represents the review of the security incident to look for
"lessons learned" and determine whether the process could have
been improved. It is recommended all security incidents be reviewed
shortly after resolution to determine where response could be improved.
Timeframes may extend to one to two weeks post-incident.

1. Responders to the security incident (SIRT Team and technical security
   resource) meet to review the documentation collected during the security
   incident.
2. A "lessons learned" section is written and attached to the Incident ticket.

    1. Evaluate the cost and impact of the security incident on Fleet using
       the documents provided by the SIRT and the technical security resource.
    2. Determine what could be improved. This may include:

        * Systems and processes adjustments
        * Awareness training and documentation
        * Implementation of additional controls

    3. Communicate these findings to Senior Management for approval and
       implementation of any recommendations made post-review of the security
       incident.
    4. Carry out recommendations approved by Senior Management; sufficient
       budget, time, and resources should be committed to this activity.

3. Ensure all incident-related information is recorded and retained as described
   in Fleet Auditing requirements and Data Retention standards.
4. Close the security incident.

#### Periodic Evaluation

It is important to note that the security incident response processes
should be periodically reviewed and evaluated for effectiveness. This
also involves appropriate training of resources expected to respond to security
incidents, as well as the training of the general population regarding
Fleet's expectations for them relative to security responsibilities. We test the
incident response plan annually.


### Information security roles and responsibilities
*Created from [Vanta](https://www.vanta.com/) policy templates.*

| Policy owner   | Effective date |
| -------------- | -------------- |
| @GuillaumeRoss | 2022-06-01     |

Fleet Device Management is committed to conducting business in compliance with all applicable laws, regulations, and company policies. Fleet has adopted this policy to outline the security measures required to protect electronic information systems and related equipment from unauthorized use.

| Role                                            | Responsibilities                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| ----------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Board of directors                              | Oversight over risk and internal control for information security, privacy, and compliance<br/> Consults with executive leadership and head of security to understand Fleet's security mission and risks and provides guidance to bring them into alignment                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| Executive leadership                            | Approves capital expenditures for information security<br/> Oversight over the execution of the information security risk management program<br/> Communication path to Fleet's board of directors. Meets with the board regularly, including at least one official meeting a year<br/> Aligns information security policy and posture based on Fleet's mission, strategic objectives, and risk appetite                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
CTO                                             | Oversight over information security in the software development process<br/>  Responsible for the design, development, implementation, operation, maintenance and monitoring of development and commercial cloud hosting security controls<br/> Responsible for oversight over policy development <br/>Responsible for implementing risk management in the development process                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| Head of security                                | Oversight over the implementation of information security controls for infrastructure and IT processes<br/>  Responsible for the design, development, implementation, operation, maintenance, and monitoring of IT security controls<br/> Communicate information security risks to executive leadership<br/> Report information security risks annually to Fleet's leadership and gains approvals to bring risks to acceptable levels<br/>  Coordinate the development and maintenance of information security policies and standards<br/> Work with applicable executive leadership to establish an information security framework and awareness program<br/>  Serve as liaison to the board of directors, law enforcement and legal department.<br/>  Oversight over identity management and access control processes |
| System owners                                   | Manage the confidentiality, integrity, and availability of the information systems for which they are responsible in compliance with Fleet policies on information security and privacy.<br/>  Approve of technical access and change requests for non-standard access                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| Employees, contractors, temporary workers, etc. | Acting at all times in a manner that does not place at risk the security of themselves, colleagues, and the information and resources they have use of<br/>  Helping to identify areas where risk management practices should be adopted<br/>  Adhering to company policies and standards of conduct Reporting incidents and observed anomalies or weaknesses                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| Head of people operations                       | Ensuring employees and contractors are qualified and competent for their roles<br/>  Ensuring appropriate testing and background checks are completed<br/>  Ensuring that employees and relevant contractors are presented with company policies <br/>  Ensuring that employee performance and adherence to values is evaluated<br/>  Ensuring that employees receive appropriate security training                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| Head of business operations                     | Responsible for oversight over third-party risk management process; responsible for review of vendor service contracts                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |

## Operations security and change management policy
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

| Policy owner   | Effective date |
| -------------- | -------------- |
| @GuillaumeRoss | 2022-06-01     |

Fleet policy requires

1. all production changes, including but not limited to software deployment, feature toggle enablement, network infrastructure changes, and access control authorization updates, must be invoked through the approved change management process.

2.each production change must maintain complete traceability to fully document the request, including the requestor, date/time of change, actions taken, and results.

3. each production change must include proper approval.

  * The approvers are determined based on the type of change.
  * Approvers must be someone other than the author/executor of the change unless they are the DRI for that system.
  * Approvals may be automatically granted if specific criteria are met.
    The auto-approval criteria must be pre-approved by the Head of Security and
    fully documented and validated for each request.

## Risk management policy
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

| Policy owner   | Effective date |
| -------------- | -------------- |
| @GuillaumeRoss | 2022-06-01     |

Fleet policy requires:

1. a thorough risk assessment must be conducted to evaluate potential threats and vulnerabilities to the confidentiality, integrity, and availability of sensitive, confidential, and proprietary electronic information Fleet stores, transmits, and/or processes.

2. risk assessments must be performed with any major change to Fleet's business or technical operations and/or supporting infrastructure no less than once per year.

3. strategies shall be developed to mitigate or accept the risks identified in the risk assessment process.

4.  The risk register is monitored quarterly to assess compliance with the above policy, and document newly discovered or created risks.

### Acceptable Risk Levels

Risks that are either low impact or low probability are generally considered acceptable.

All other risks must be individually reviewed and managed.

### Risk corrective action timelines

| Risk Level | Corrective action timeline |
| ---------- | ------------------- |
| Low        | Best effort         |
| Medium     | 120 days            |
| High       | 30 days             |


## Secure software development and product security policy 
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

Fleet policy requires that:

1. Fleet software engineering and product development are required to follow security best practices. The product should be "Secure by Design" and "Secure by Default."

2. Fleet performs quality assurance activities. This may include

  * peer code reviews prior to merging new code into the main development branch
    (e.g., master branch).
  * thorough product testing before releasing it to production (e.g., unit testing
    and integration testing).

3. Risk assessment activities (i.e., threat modeling) must be performed for a new product or extensive changes to an existing product.

4. Security requirements must be defined, tracked, and implemented.

5. Security analysis must be performed for any open source software and/or third-party components and dependencies included in Fleet software products.

6. Static application security testing (SAST) must be performed throughout development and before each release.

7. Dynamic application security testing (DAST) must be performed before each release.

8. All critical or high severity security findings must be remediated before each release.

9. All critical or high severity vulnerabilities discovered post-release must be remediated in the next release or as per the Fleet vulnerability management policy SLAs, whichever is sooner.

10. Any exception to the remediation of a finding must be documented and approved by the security team or CTO.

## Security policy management policy
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

| Policy owner   | Effective date |
| -------------- | -------------- |
| @GuillaumeRoss | 2022-06-01     |

Fleet policy requires that:

1. Fleet policies must be developed and maintained to meet all applicable compliance requirements and adhere to security best practices, including but not limited to:

- SOC 2

2. Fleet must annually review all policies.

3. Fleet maintains all policy changes must be approved by Fleet's head of security. Additionally,

  * Major changes may require approval by Fleet CEO or designee;
  * Changes to policies and procedures related to product development may
    require approval by the CTO.

3. Fleet maintains all policy documents with version control.

4. Policy exceptions are handled on a case-by-case basis.

  * All exceptions must be fully documented with business purpose and reasons
    why the policy requirement cannot be met.
  * All policy exceptions must be approved by Fleet Head of Security and CEO.
  * An exception must have an expiration date no longer than one year from date
    of exception approval and it must be reviewed and re-evaluated on or before
    the expiration date.

## Third-party management policy
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

| Policy owner   | Effective date |
| -------------- | -------------- |
| @GuillaumeRoss | 2022-06-01     |

Fleet makes every effort to assure all third-party organizations are
compliant and do not compromise the integrity, security, and privacy of Fleet
or Fleet Customer data. Third Parties include Vendors, Customers, Partners,
Subcontractors, and Contracted Developers.

1. A list of approved vendors/partners must be maintained and reviewed annually.

2. Approval from management, procurement, and security must be in place before onboarding any new vendor or contractor that impacts Fleet production systems. Additionally, all changes to existing contract agreements must be reviewed and approved before implementation.

3. For any technology solution that needs to be integrated with Fleet production environment or operations, the security team must perform a Vendor Technology Review to understand and approve the risk. Periodic compliance assessment and SLA review may be required.

4. Fleet Customers or Partners should not be allowed access outside of their own environment, meaning they cannot access, modify, or delete any data belonging to other third parties.

5. Additional vendor agreements are obtained as required by applicable regulatory compliance requirements.

<meta name="maintainedBy" value="guillaumeross">
<meta name="title" value="📜 Security policies">
