# Security policies

## Information security policy and acceptable use policy

This Information Security Policy is intended to protect Fleet Device Management Inc's employees, contractors, partners, customers, and the company from illegal or damaging actions by individuals, either knowingly or unknowingly.

Internet/Intranet/Extranet-related systems, including but not limited to computer equipment, software, operating systems, storage media, network accounts providing electronic mail, web browsing, and file transfers, are the property of Fleet Device Management Inc. These systems are to be used for business purposes in serving the interests of the company, and of our clients and customers in the course of normal operations.

Effective security is a team effort involving the participation and support of every Fleet Device Management Inc employee or contractor who deals with information and/or information systems. It is the responsibility of every team member to read and understand this policy and conduct their activities accordingly.

### Acceptable use of end-user computing
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

| Policy owner   | Effective date |
| -------------- | -------------- |
| @GuillaumeRoss | 2022-06-01     |

Fleet requires all workforce members to comply with the following acceptable use requirements and procedures, such as:

1. The use of Fleet computing systems is subject to monitoring by Fleet IT and/or Security teams.

2. Fleet team members must not leave computing devices (including laptops and smart devices) used for business purposes, including company-provided and BYOD devices, unattended in public.

3. Device encryption must be enabled for all mobile devices accessing company data, such as whole-disk encryption for all laptops.

4. Use only legal software with a valid license installed through the internal "app store" or trusted sources. Well-documented open source software can be used. If in doubt, ask in *#g-security*.  

5. Avoid sharing credentials. Secrets must be stored safely, using features such as GitHub secrets. For accounts and other sensitive data that need to be shared, use the company-provided password manager.

6. At Fleet, we are public by default. Sensitive information from logs, screenshots, or other types of data (memory dumps, for example), must be sanitized to remove any sensitive or confidential information prior to posting.

7. Anti-malware or equivalent protection and monitoring must be installed and enabled on all endpoint systems that may be affected by malware, including workstations, laptops and servers.

8. It is strictly forbidden to download or store any secrets used to sign Orbit installer updates on end-user computing devices, including laptops, workstations, and mobile devices.

9. Only company-owned and managed computers are allowed to connect directly to Fleet autoupdater production environments.

10. Fleet team members must not let anyone else use Fleet-provided and managed workstations unsupervised, including family members and support personnel of vendors. Use screen sharing instead of allowing them to access your system directly.

11. Device's operating system must be kept up to date. Fleet-managed systems will receive prompts for updates to be installed, and BYOD devices are to be updated by the team member using it or they might lose access. 

12. Team members must not store sensitive data on portable storage.

13. The use of Fleet company accounts on "shared" computers, such as hotel kiosk systems, is strictly prohibited.

### Risk management policy
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

| Policy owner   | Effective date |
| -------------- | -------------- |
| @GuillaumeRoss | 2022-06-01     |

Fleet policy requires that:

1. A thorough risk assessment must be conducted to evaluate potential threats and vulnerabilities to the confidentiality, integrity, and availability of sensitive, confidential and proprietary electronic information Fleet stores, transmits, and/or processes.

2. Risk assessments must be performed with any major change to Fleet's business or technical operations and/or supporting infrastructure, no less than once per year.

3. Strategies shall be developed to mitigate or accept the risks identified in the risk assessment process.


### Secure software development and product security policy 
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

Fleet policy requires that:

1. Fleet software engineering and product development is required to follow security best practices. Product should be "Secure by Design" and "Secure by Default".

2. Quality assurance activities will be performed.  This may include

  * peer code reviews prior to merging new code into the main development branch
    (e.g. master branch); and
  * thorough product testing before releasing to production (e.g. unit testing
    and integration testing).

3. Risk assessment activities (i.e. threat modeling) must be performed for a new product or major changes to an existing product.

4. Security requirements must be defined, tracked, and implemented.

5. Security analysis must be performed for any open source software and/or third-party components and dependencies included in Fleet software products.

6. Static application security testing (SAST) must be performed throughout development and prior to each release.

7. Dynamic application security testing (DAST) must be performed prior to each release.

8. All critical or high severity security findings must be remediated prior to each release.

9. All critical or high severity vulnerabilities discovered post release must be remediated in the next release or as per the Fleet vulnerability management policy SLAs, whichever is sooner.

10. Any exception to the remediation of a finding must be documented and approved by the security team or CTO.

### Human resources security policy
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

| Policy owner   | Effective date |
| -------------- | -------------- |
| @GuillaumeRoss | 2022-06-01     |


Fleet is committed to ensuring all workforce members participate in security and compliance in their roles at Fleet. We encourage self-management and reward the right behaviors. 

Fleet policy requires all workforce members to comply with the HR Security Policy.

Fleet policy requires that:

1. Background verification checks on candidates for employees and contractors with production access to the Fleet automatic updater service must be carried out in accordance with relevant laws, regulations and ethics, and proportional to the business requirements, the classification of the information to be accessed, and the perceived risk.

2. Employees, contractors and third-party users must agree and sign the terms and conditions of their employment contract and comply with acceptable use.

3. Employees will perform an onboarding process that familiarizes them with the environments, systems, security requirements, and procedures Fleet has in place. Employees will also have ongoing security awareness training that is audited.

4. Employee offboarding will include reiterating any duties and responsibilities still valid after terminations, verifying that access to any Fleet systems has been removed, and ensuring that all company-owned assets are returned.

5. Fleet and its employees will take reasonable measures to ensure no sensitive data is transmitted via digital communications such as email or posted on social media outlets.

6. Fleet will maintain a list of prohibited activities that will be part of onboarding procedures and have training available if/when the list of those activities changes.

7. A fair disciplinary process will be used for employees that are suspected of committing breaches of security. Multiple factors will be considered when deciding the response, such as whether or not this was a first offense, training, business contracts, etc. Fleet reserves the right to terminate employees in the case of serious cases of misconduct.

8. Fleet will maintain a reporting structure that aligns with the organization's business lines and/or individual's functional roles. The list of employees and reporting structure must be available to [all employees](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0).

9. Employees will receive regular feedback and acknowledgment from their managers and peers. Managers will give constant feedback on performance, including but not limited to during regular one-on-one meetings.

10. Fleet will publish job descriptions for available positions and conducts interviews to assess a candidate's technical skills as well as soft skills prior to hiring.

11. Background checks of an employee or contractor must be performed by operations and/or the hiring team prior to the start date of employment.
 
### Encryption policy
*Created from [JupiterOne/security-policy-templates](https://github.com/JupiterOne/security-policy-templates). [CC BY-SA 4 license](https://creativecommons.org/licenses/by-sa/4.0/)*

| Policy owner   | Effective date |
| -------------- | -------------- |
| @GuillaumeRoss | 2022-06-01     |

Fleet requires all workforce members to comply with the encryption policy, such that:

1. The storage drives of all Fleet-owned workstations must be encrypted, and enforced by the IT and/or security team.

2. Confidential data must be stored in a manner that supports user access logs.

3. All Production Data at rest is stored on encrypted volumes.

4. Volume encryption keys and machines that generate volume encryption keys are protected from unauthorized access. Volume encryption key material is protected with access controls such that the key material is only accessible by privileged accounts.

5. Encrypted volumes use strong cipher algorithms, key strength, and key management process as defined below.

6. Data is protected in transit using recent TLS versions with ciphers recognized as secure.

#### Local disk/volume encryption

Encryption and key management for local disk encryption of end-user devices follow the defined best practices for Windows, macOS, and Linux/Unix operating systems, such as Bitlocker and FileVault. 

#### Protecting data in transit

1. All external data transmission is encrypted end-to-end. This includes, but is not limited to, cloud infrastructure and third-party vendors and applications.

2. Transmission encryption keys and systems that generate keys are protected from unauthorized access. Transmission encryption key materials are protected with access controls and may only be accessed by privileged accounts.

3. TLS endpoints must score at least an "A" on SSLLabs.com.

4. Transmission encryption keys are limited to use for one year and then must be regenerated.

### Access control policy
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

#### Access authorization and termination

Fleet policy requires that:

1. access authorization shall be implemented using role-based access control (RBAC) or a similar mechanism.

2. standard access based on a user's job role may be pre-provisioned during employee onboarding. All subsequent access requests to computing resources must be approved by the requestor’s manager prior to granting and provisioning of access.

3. access to critical resources, such as production environments, must be approved by the security team in addition to the requestor’s manager.

4. access must be reviewed on regularly and revoked if no longer needed.

5. upon termination of employment, all system access must be revoked, and user accounts terminated within 24 hours or one business day, whichever is shorter.

6. all system access must be reviewed at least annually and whenever a user's job role changes.

#### Shared secrets management

Fleet policy requires that:

1. use of shared credentials/secrets must be minimized.

2. if required by business operations, secrets/credentials must be shared securely and stored in encrypted vaults that meet the Fleet data encryption standards.

#### Privileged access management

Fleet policy requires that:

1. automation with service accounts must be used to configure production systems when technically feasible.

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

## Security policy management policy
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
=======
2. use of high privilege accounts must only be performed when absolutely necessary.

<meta name="maintainedBy" value="guillaumeross">
