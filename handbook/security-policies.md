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

1. Fleet software engineering and product development are required to follow security best practices. The product should be "Secure by Design" and "Secure by Default."

2. Fleet performs quality assurance activities. This may include:

  * peer code reviews prior to merging new code into the main development branch
    (e.g., master branch).
  * thorough product testing before releasing it to production (e.g., unit testing
    and integration testing).

3. Risk assessment activities (i.e., threat modeling) must be performed for a new product or major changes to an existing product.

4. Security requirements must be defined, tracked, and implemented.

5. Security analysis must be performed for any open source software and/or third-party components and dependencies included in Fleet software products.

6. Static application security testing (SAST) must be performed throughout development and prior to each release.

7. Dynamic application security testing (DAST) must be performed prior to each release.

8. All critical or high severity security findings must be remediated prior to each release.

9. All critical or high severity vulnerabilities discovered post-release must be remediated in the next release or as per the Fleet vulnerability management policy SLAs, whichever is sooner.

10. Any exception to the remediation of a finding must be documented and approved by the security team or CTO.

### Human resources security policy
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

5. Fleet and its employees will take reasonable measures to ensure no sensitive data is transmitted via digital communications such as email or posted on social media outlets.

6. Fleet will maintain a list of prohibited activities that will be part of onboarding procedures and have training available if/when the list of those activities changes.

7. A fair disciplinary process will be used for employees suspected of committing breaches of security. Fleet will consider multiple factors when deciding the response, such as whether or not this was a first offense, training, business contracts, etc. Fleet reserves the right to terminate employees in the case of severe cases of misconduct.

8. Fleet will maintain a reporting structure that aligns with the organization's business lines and/or individual's functional roles. The list of employees and reporting structure must be available to [all employees](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0).

9. Employees will receive regular feedback and acknowledgment from their managers and peers. Managers will give constant feedback on performance, including but not limited to during regular one-on-one meetings.

10. Fleet will publish job descriptions for available positions and conduct interviews to assess a candidate's technical skills as well as soft skills prior to hiring.

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

2. use of high privilege accounts must only be performed when absolutely necessary.

<meta name="maintainedBy" value="guillaumeross">
