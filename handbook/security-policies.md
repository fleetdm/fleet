# Security policies

## Information security policy and acceptable use policy

This Information Security Policy is intended to protect Fleet Device Management Inc's employees, contractors, partners, customers, and the company from illegal or damaging actions by individuals, either knowingly or unknowingly.

Internet/Intranet/Extranet-related systems, including but not limited to computer equipment, software, operating systems, storage media, network accounts providing electronic mail, web browsing, and file transfers, are the property of Fleet Device Management Inc. These systems are to be used for business purposes in serving the interests of the company, and of our clients and customers in the course of normal operations.

Effective security is a team effort involving the participation and support of every Fleet Device Management Inc employee or contractor who deals with information and/or information systems. It is the responsibility of every team member to read and understand this policy and conduct their activities accordingly.

### Acceptable Use of End-user Computing
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


<meta name="maintainedBy" value="guillaumeross">
