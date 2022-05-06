# Security policies

## Information security policy and acceptable use policy

This Information Security Policy is intended to protect Fleet Device Management Inc's employees, contractors, partners, customers and the company from illegal or damaging actions by individuals, either knowingly or unknowingly.

Internet/Intranet/Extranet-related systems, including but not limited to computer equipment, software, operating systems, storage media, network accounts providing electronic mail, web browsing, and file transfers, are the property of Fleet Device Management Inc. These systems are to be used for business purposes in serving the interests of the company, and of our clients and customers in the course of normal operations.

Effective security is a team effort involving the participation and support of every Fleet Device Management Inc employee or contractor who deals with information and/or information systems. It is the responsibility of every team member to read and understand this policy, and to conduct their activities accordingly.

### Acceptable Use of End-user Computing
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

| Policy owner   | Effective date |
| -------------- | -------------- |
| @GuillaumeRoss | 2022-06-01     |

Fleet requires all workforce members to comply with the following acceptable use requirements and procedures, such that:

1. Use of Fleet computing systems is subject to monitoring by Fleet IT and/or Security teams.

2. Fleet team members must not leave computing devices (including laptops and smart devices) used for business purpose, including company-provided and BYOD devices, unattended in public.

3. Device encryption must be enabled for all mobile devices accessing company data, such as whole-disk encryption for all laptops.

4. Use only legal software with a valid license installed through the internal "app store" or trusted sources. Well-documented open source software can be used. If in doubt, ask in *#g-security*.  

5. Avoid sharing credentials. Secrets must be stored safely, using features such as GitHub secrets. For accounts and other sensitive data that need to be shared, use the company-provided password manager.

6. At Fleet, we are public by default. Sensitive information from logs, screenshots or other types of data (memory dumps for example), must be sanitized to remove any sensitive or confidential information prior to posting.

7. Anti-malware or equivalent protection and monitoring must be installed and enabled on all endpoint systems that may be affected by malware, including workstations, laptops and servers.

8. It is strictly forbidden to download or store any secrets used to sign Orbit installer updates on end-user computing devices, including laptops, workstations and mobile devices.

9. Only company owned and managed computers are allowed to connect directly to Fleet auto updater production environments.

10. Fleet team members must not let anyone else use Fleet provided and managed workstations unsupervised, including family members and support personnel of vendors. Use screen sharing instead of allowing them to access your system directly.

11. Device operating system must be kept up to date. Fleet managed systems will receive prompts for updates to be installed, and BYOD devices are to be updated by the team member using it, or might lose access. 

12. Team members must not store sensitive data on portable storage.

13. The use of Fleet company accounts on "shared" computers, such as hotel kiosk systems, is strictly prohibited.

### Access control policy

Fleet requires all workforce members to comply with the following acceptable use requirements and procedures, such that:

1. Access to all computing resources, including servers, end-user computing devices, network equipment, services and applications, must be protected by strong authentication, authorization, and auditing.

2. Interactive user access to production systems must be associated to an account or login unique to each user.

3. All credentials, including user passwords, service accounts, and access keys, must meet the length, complexity, age, and rotation requirements defined in Fleet security standards.

4. Use strong password and two-factor authentication (2FA) whenever possible to authenticate to all computing resources (including both devices and applications).

5. 2FA is required to access any critical system or resource, including but not limited to resources in Fleet production environments.

6. Unused accounts, passwords, access keys must be removed within 30 days.

7. A unique access key or service account must be used for different application or user access.

8. Authenticated sessions must time out after a defined period of inactivity.

#### Access authorization and termination

Fleet policy requires that

1. Access authorization shall be implemented using role-based access control (RBAC) or similar mechanism.

2. Standard access based on a user's job role may be pre-provisioned during employee onboarding. All subsequent access requests to computing resources must be approved by the requestor’s manager, prior to granting and provisioning of access.

3. Access to critical resources, such as production environments, must be approved by the security team in addition to the requestor’s manager.

4. Access must be reviewed on a regular basis and revoked if no longer needed.

5. Upon termination of employment, all system access must be revoked and user accounts terminated within 24 hours or one business day, whichever is shorter.

6. All system access must be reviewed at least annually and whenever a user's job role changes.

#### Shared secrets management

Fleet policy requires that

1. Use of shared credentials/secrets must be minimized.

2. If required by business operations, secrets/credentials must be shared securely and stored in encrypted vaults that meet the Fleet data encryption standards.

#### Privileged access management

Fleet policy requires that

1. Automation with service accounts must be used to configure production systems when technically feasible.

2. Use of high privilege accounts must only be performed when absolutely necessary.

## Asset management policy
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

| Policy owner   | Effective date |
| -------------- | -------------- |
| @GuillaumeRoss | 2022-06-01     |

You can't protect what you can't see. Therefore, Fleet must maintain an accurate and up-to-date inventory of its physical and digital assets.

Fleet policy requires that:

1. IT and/or security must maintain an inventory of all critical company assets, both physical and logical.

2. All assets should have identified owners and be tagged with a risk/data classification.

3. All company-owned computer purchases must be tracked.

## Business continuity and disaster recovery policy
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

| Policy owner   | Effective date |
| -------------- | -------------- |
| @GuillaumeRoss | 2022-06-01     |

The Fleet business continuity and disaster recovery plan establishes procedures to recover Fleet following a disruption resulting from a disaster. 

Fleet policy requires that:

1. A plan and process for business continuity and disaster recovery (BCDR), including the backup and recovery of critical systems and data, will be defined and documented.

2. BCDR shall be simulated and tested at least once a year. 

3. Security controls and requirements will be maintained during all BCDR activities.

## Information security roles and responsibilities
*Created from [Vanta](https://www.vanta.com/) policy templates.*

| Policy owner   | Effective date |
| -------------- | -------------- |
| @GuillaumeRoss | 2022-06-01     |

Fleet Device Management is committed to conducting business in compliance with all applicable laws, regulations, and company policies. Fleet has adopted this policy to outline the security measures required to protect electronic information systems and related equipment from unauthorized use.

| Role                                            | Responsibilities                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| ----------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Board of directors                              | Oversight over risk and internal control for information security, privacy and compliance<br/> Consults with executive leadership and head of security to understand Fleet's security mission and risks and provides guidance to bring them into alignment                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| Executive leadership                            | Approves capital expenditures for information security<br/> Oversight over the execution of the information security risk management program<br/> Communication path to Fleet's board of directors<br/> Aligns information security policy and posture based on Fleet's mission, strategic objectives and risk appetite                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
CTO                                             | Oversight over information security in the software development process<br/>  Responsible for the design, development, implementation, operation, maintenance and monitoring of development and commercial cloud hosting security controls<br/> Responsible for oversight over policy development <br/>Responsible for implementing risk management in the development process                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| Head of security                                | Oversight over the implementation of information security controls for infrastructure and IT processes<br/>  Responsible for the design, development, implementation, operation, maintenance and monitoring of IT security controls<br/> Communicate information security risks to executive leadership<br/> Report information security risks annually to Fleet's leadership and gains approvals to bring risks to acceptable levels<br/>  Coordinate the development and maintenance of information security policies and standards<br/> Work with applicable executive leadership to establish an information security framework and awareness program<br/>  Serve as liaison to the board of directors, law enforcement and legal department.<br/>  Oversight over identity management and access control processes |
| System owners                                   | Manage the confidentiality, integrity and availability of the information systems for which they are responsible in compliance with Fleet policies on information security and privacy.<br/>  Approve of technical access and change requests for non-standard access                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| Employees, contractors, temporary workers, etc. | Acting at all times in a manner which does not place at risk the security of themselves, colleagues, and of the information and resources they have use of<br/>  Helping to identify areas where risk management practices should be adopted<br/>  Adhering to company policies and standards of conduct Reporting incidents and observed anomalies or weaknesses                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| Head of people operations                       | Ensuring employees and contractors are qualified and competent for their roles<br/>  Ensuring appropriate testing and background checks are completed<br/>  Ensuring that employees and relevant contractors are presented with company policies <br/>  Ensuring that employee performance and adherence to values is evaluated<br/>  Ensuring that employees receive appropriate security training                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| Head of business operations                     | Responsible for oversight over third-party risk management process Responsible for review of vendor service contracts                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |

## Operations security and change management policy
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

| Policy owner   | Effective date |
| -------------- | -------------- |
| @GuillaumeRoss | 2022-06-01     |

Fleet policy requires that:

1. All production changes, including but not limited to software deployment, feature toggle enablement, network infrastructure changes, and access control authorization updates, must be invoked through approved change management process.

2. Each production change must maintain complete traceability to fully document the request, including requestor, date/time of change, actions taken and results.

3. Each production change must include proper approval.

  * The approvers are determined based on the type of change.
  * Approvers must be someone other than the author/executor of the change, unless they are the DRI for that system.
  * Approvals may be automatically granted if certain criteria is met.
    The auto-approval criteria must be pre-approved by the Security Officer and
    fully documented and validated for each request.


## Third-party management policy
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

| Policy owner   | Effective date |
| -------------- | -------------- |
| @GuillaumeRoss | 2022-06-01     |

Fleet makes every effort to assure all third party organizations are
compliant and do not compromise the integrity, security, and privacy of Fleet
or Fleet Customer data. Third Parties include Vendors, Customers, Partners,
Subcontractors, and Contracted Developers.

1. A list of approved vendors/partners must be maintained and reviewed annually.

2. Approval from management, procurement and security must be in place before onboarding any new vendor or contractor with impacton on Fleet production systems. Additionally, all changes to existing contract agreements must be reviewed and approved before implementation.

3. For any technology solution that needs to be integrated with Fleet production environment or operations, a Vendor Technology Review must be performed by the security team to understand and approve the risk.  Periodic compliance assessment and SLA review may be required.

4. Fleet Customers or Partners should not be allowed access outside of their own environment, meaning they cannot access, modify, or delete any data belonging to other 3rd parties.

5. Additional vendor agreements are obtained as required by applicable regulatory compliance requirements.

## Security policy management
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

| Policy owner   | Effective date |
| -------------- | -------------- |
| @GuillaumeRoss | 2022-06-01     |

Fleet policy requires that:

1. Fleet policies must be developed and maintained to meet all applicable compliance requirements adhere to security best practices, including but not limited to:

- SOC 2

2. All policies must be reviewed at least annually.

3. All policy changes must be approved by Fleet's head of security. Additionally,

  * Major changes may require approval by Fleet CEO or designee;
  * Changes to policies and procedures related to product development may
    require approval by the CTO.

3. All policy documents must be maintained with version control.

4. Policy exceptions are handled on a case-by-case basis.

  * All exceptions must be fully documented with business purpose and reasons
    why the policy requirement cannot be met.
  * All policy exceptions must be approved by both Fleet Security Officer and CEO.
  * An exception must have an expiration date no longer than one year from date
    of exception approval and it must be reviewed and re-evaluated on or before
    the expiration date.

<meta name="maintainedBy" value="guillaumeross">
