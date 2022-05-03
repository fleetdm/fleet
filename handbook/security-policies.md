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


<meta name="maintainedBy" value="guillaumeross">
