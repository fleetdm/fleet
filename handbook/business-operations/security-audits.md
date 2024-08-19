# Security audits
This page contains explanations of the latest external security audits performed on Fleet software.

## June 2024 penetration testing of Fleet 4.50.1
In June 2024, [Latacora](https://www.latacora.com/) performed an application penetration assessment of the application from Fleet. 

An application penetration test captures a point-in-time assessment of vulnerabilities, misconfigurations, and gaps in applications that could allow an attacker to compromise the security, availability, processing integrity, confidentiality, and privacy (SAPCP) of sensitive data and application resources. An application penetration test simulates the capabilities of a real adversary, but accelerates testing by using information provided by the target company.

Latacora identified a few medium and low severity risks, and Fleet is prioritizing and responding to those within SLAs. Once all action has been taken, a summary will be provided.

You can find the full report here: 2024-06-14-fleet-penetration-test.pdf

## June 2023 penetration testing of Fleet 4.32 
In June 2023, [Latacora](https://www.latacora.com/) performed an application penetration assessment of the application from Fleet. 

An application penetration test captures a point-in-time assessment of vulnerabilities, misconfigurations, and gaps in applications that could allow an attacker to compromise the security, availability, processing integrity, confidentiality, and privacy (SAPCP) of sensitive data and application resources. An application penetration test simulates the capabilities of a real adversary, but accelerates testing by using information provided by the target company.

Latacora identified a few issues, the most critical ones we have addressed in 4.33. These are described below.

You can find the full report here: [2023-06-09-fleet-penetration-test.pdf](https://github.com/fleetdm/fleet/raw/main/docs/files/2023-06-09-fleet-penetration-test.pdf).

### Findings
#### 1 - Stored cross-site scripting (XSS) in tooltip
| Type                | Latacora Severity |
| ------------------- | -------------- |
| Cross-site scripting| High risk      |

All tooltips using the "tipContent" tag are set using "dangerouslySetInnerHTML". This allows manipulation of the DOM without sanitization. If a user can control the content sent to this function, it can lead to a cross-site scripting vulnerability. 

This was resolved in version release [4.33.0](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.33.0) with [implementation of DOMPurify library](https://github.com/fleetdm/fleet/pull/12229) to remove dangerous dataset.

#### 2 - Broken authorization leads to observers able to add hosts
| Type                | Latacora Severity |
| ------------------- | -------------- |
| Authorization issue | High risk      |

Observers are not supposed to be able to add hosts to Fleet. Via specific endpoints, it becomes possible to retrieve the certificate chains and the secrets for all teams, and these are the information required to add a host. 

This was resolvedin version release [4.33.0](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.33.0) with [updating the observer permissions](https://github.com/fleetdm/fleet/pull/12216).

## April 2022 penetration testing of Fleet 4.12 
In April 2022, we worked with [Lares](https://www.lares.com/) to perform penetration testing on our Fleet instance, which was running 4.12 at the time. 

Lares identified a few issues, the most critical ones we have addressed in 4.13. Other less impactful items remain. These are described below.

As usual, we have made the full report (minus redacted details such as email addresses and tokens) available.

You can find the full report here: [2022-04-29-fleet-penetration-test.pdf](https://github.com/fleetdm/fleet/raw/main/docs/files/2022-04-29-fleet-penetration-test.pdf).

### Findings
#### 1 - Broken access control & 2 - Insecure direct object reference
| Type                | Lares Severity |
| ------------------- | -------------- |
| Authorization issue | High risk      |

This section contains a few different authorization issues, allowing team members to access APIs out of the scope of their teams. The most significant problem was that a team administrator was able to add themselves to other teams. 

This is resolved in 4.13, and an [advisory](https://github.com/fleetdm/fleet/security/advisories/GHSA-pr2g-j78h-84cr) has been published before this report was made public.
We are also planning to add [more testing](https://github.com/fleetdm/fleet/issues/5457) to catch potential future mistakes related to authorization.

#### 3 - CSV injection in export functionality
| Type      | Lares Severity |
| --------- | -------------- |
| Injection | Medium risk    |

It is possible to create or rename an existing team with a malicious name, which, once exported to CSV, could trigger code execution in Microsoft Excel. We assume there are other ways that inserting this type of data could have similar effects, including via osquery data. For this reason, we will evaluate the feasibility of [escaping CSV output](https://github.com/fleetdm/fleet/issues/5460).

Our current recommendation is to review CSV contents before opening in Excel or other programs that may execute commands.

#### 4 - Insecure storage of authentication tokens
| Type                   | Lares Severity |
| ---------------------- | -------------- |
| Authentication storage | Medium risk    |

This issue is not as straightforward as it may seem. While it is true that Fleet stores authentication tokens in local storage as opposed to cookies, we do not believe the security impact from that is significant. Local storage is immune to CSRF attacks, and cookie protection is not particularly strong. For these reasons, we are not planning to change this at this time, as the changes would bring minimal security improvement, if any, and change always carries the risk of creating new vulnerabilities.

#### 5 - No account lockout
| Type           | Lares Severity |
| -------------- | -------------- |
| Authentication | Medium risk    |

Account lockouts on Fleet are handled as a “leaky bucket” with 10 available slots. Once the bucket is full, a four second timeout must expire before another login attempt is allowed. We believe that any longer, including full account lockout, could bring user experience issues as well as denial of service issues without improving security, as brute-forcing passwords at a rate of one password per 4 seconds is very unlikely.

We have additionally added very prominent activity feed notifications of failed logins that make brute forcing attempts apparent to Fleet admins.

#### 6 - Session timeout - insufficient session expiration
| Type               | Lares Severity |
| ------------------ | -------------- |
| Session expiration | Medium risk    |

Fleet sessions are currently [configurable](https://fleetdm.com/docs/deploying/configuration#session-duration). However, the actual behavior, is different than the expected one. We [will switch](https://github.com/fleetdm/fleet/issues/5476) the behavior so the session timeout is based on the length of the session, not on how long it has been idle. The default will remain five days, which will result in users having to log in at least once a week, while the current behavior would allow someone to remain logged in forever. If you have any reason to want a shorter session duration, simply configure it to a lower value.

#### 7 - Weak passwords allowed
| Type           | Lares Severity |
| -------------- | -------------- |
| Weak passwords | Medium risk    |

The default password policy in Fleet requires passwords that are seven characters long. We have [increased this to 12](https://github.com/fleetdm/fleet/issues/5477) while leaving all other requirements the same. As per NIST [SP 800-63B](https://pages.nist.gov/800-63-3/sp800-63b.html), we believe password length is the most important requirement. If you have additional requirements for passwords, we highly recommend implementing them in your identity provider and setting up [SSO](https://fleetdm.com/docs/deploying/configuration#configuring-single-sign-on-sso).

#### 8 - User enumeration
| Type        | Lares Severity |
| ----------- | -------------- |
| Enumeration | Low risk       |

User enumeration by a logged-in user is not a critical issue. Still, when done by a user with minimal privileges (such as a team observer), it is a leak of information, and might be a problem depending on how you use teams. For this reason, only team administrators are able to enumerate users as of Fleet 4.31.0.

#### 9 - Information disclosure via default content
| Type                   | Lares Severity |
| ---------------------- | -------------- |
| Information disclosure | Informational  |

This finding has two distinct issues. 

The first one is the /metrics endpoint, which contains a lot of information that could potentially be leveraged for attacks. We had identified this issue previously, and it was [fixed in 4.13](https://github.com/fleetdm/fleet/issues/2322) by adding authentication in front of it.

The second one is /version. While it provides some minimal information, such as the version of Fleet and go that is used, it is information similar to a TCP banner on a typical network service. For this reason, we are leaving this endpoint available. 

If this endpoint is a concern in your Fleet environment, consider that the information it contains could be gleaned from the HTML and JavaScript delivered on the main page. If you still would like to block it, we recommend using an application load balancer.

#### The GitHub issues that relate to this test are:
[Security advisory fixed in Fleet 4.13](https://github.com/fleetdm/fleet/security/advisories/GHSA-pr2g-j78h-84cr)

[Add manual and automated test cases for authorization #5457](https://github.com/fleetdm/fleet/issues/5457)

[Evaluate current CSV escaping and feasibility of adding if missing #5460](https://github.com/fleetdm/fleet/issues/5460)

[Set session duration to total session length #5476](https://github.com/fleetdm/fleet/issues/5476)

[Increase default minimum password length to 12 #5477](https://github.com/fleetdm/fleet/issues/5477)

[Add basic auth to /metrics endpoint #2322](https://github.com/fleetdm/fleet/issues/2322)

[Ensure only team admins can list other users #5657](https://github.com/fleetdm/fleet/issues/5657)

## August 2021 security of Orbit auto-updater

Back in 2021, when Orbit was still new, alpha, and likely not used by anyone but us here at Fleet, we contracted Trail of Bits (ToB) to have them review the security of the auto-updater portion of it.

For more context around why we did this, please see this [post](https://blog.fleetdm.com/security-testing-at-fleet-orbit-auto-updater-audit-7e3e99152a25) on the Fleet blog.

You can find the full report here: [2021-04-26-orbit-auto-updater-assessment.pdf](https://github.com/fleetdm/fleet/raw/3ad02fc697e196b5628bc07e807fbc2db3086393/docs/files/2021-04-26-orbit-auto-updater-assessment.pdf)

### Findings

#### 1 - Unhandled deferred file close operations
| Type               | ToB Severity |
| ------------------ | ------------ |
| Undefined Behavior | Low          |

This issue was addressed in PR [1679](https://github.com/fleetdm/fleet/issues/1679) and merged on August 17, 2021.

The fix is an improvement to cleanliness, and though the odds of exploitation were very low, there is no downside to improving it. 

This finding did not impact the auto-update mechanism but did impact Orbit installations.

#### 2 - Files and directories may pre-exist with too broad permissions
| Type            | ToB Severity |
| --------------- | ------------ |
| Data Validation | High         |

This issue was addressed in PR [1566](https://github.com/fleetdm/fleet/pull/1566) and merged on August 11, 2021

Packaging files with permissions that are too broad can be hazardous. We fixed this in August 2021. We also recently added a [configuration](https://github.com/fleetdm/fleet/blob/f32c1668ae3bc57d33c31eb30eb1959f65963a0a/.golangci.yml#L29) to our [linters](https://en.wikipedia.org/wiki/Lint_(software)) and static analysis tools to throw an error any time permissions on a file are above 0644 to help avoid future similar issues. We rarely change these permissions. When we do, they will have to be carefully code-reviewed no matter what, so we have also enforced code reviews on the Fleet repository.

This finding did not impact the auto-update mechanism but did impact Orbit installations.

#### 3 - Possible nil pointer dereference 
| Type            | ToB Severity  |
| --------------- | ------------- |
| Data Validation | Informational |

We did not do anything specific for this informational recommendation. However, we did deploy multiple SAST tools, such as [gosec](https://github.com/securego/gosec), mentioned in the previous issue, and [CodeQL](https://codeql.github.com/), to catch these issues in the development process.

This finding did not impact the auto-update mechanism but did impact Orbit installations.

#### 4 - Forcing empty passphrase for keys encryption
| Type         | ToB Severity |
| ------------ | ------------ |
| Cryptography | Medium       |

This issue was addressed in PR [1538](https://github.com/fleetdm/fleet/pull/1538) and merged on August 9, 2021.

We now ensure that keys do not have empty passphrases to prevent accidents.

#### 5 - Signature verification in fleetctl commands
| Type            | ToB Severity |
| --------------- | ------------ |
| Data Validation | High         |

Our threat model for the Fleet updater does not include the TUF repository itself being malicious. We currently assume that if the TUF repository is compromised and that the resulting package could be malicious. For this reason, we keep the local repository used with TUF offline (except for the version we publish and never re-sign) with the relevant keys, and why we add target files directly rather than adding entire directories to mitigate this risk. 

We consider the security of the TUF repository itself out of the threat model of the Orbit auto-updater at the moment, similarly to how we consider the GitHub repository out of scope. We understand that if the repository was compromised, an attacker could get malicious code to be signed, and so we have controls at the GitHub level to prevent this from happening. For TUF, currently, our mitigation is to keep the files offline.

We plan to document our update process, including the signature steps, and improve them to reduce risk as much as possible. 

#### 6 - Redundant online keys in documentation
| Type            | ToB Severity |
| --------------- | ------------ |
| Access Controls | Medium       |

Using the right key in the right place and only in the right place is critical to the security of the update process. 

This issue was addressed in PR [1678](https://github.com/fleetdm/fleet/pull/1678) and merged on August 15, 2021. 

#### 7 - Lack of alerting mechanism 
| Type          | ToB Severity |
| ------------- | ------------ |
| Configuration | Medium       |

We will make future improvements, always getting better at detecting potential attacks, including the infrastructure and processes used for the auto-updater.

#### 8 - Key rotation methodology is not documented
| Type         | ToB Severity |
| ------------ | ------------ |
| Cryptography | Medium       |

This issue was addressed in PR [2831](https://github.com/fleetdm/fleet/pull/2831) and merged on November 15, 2021

#### 9 - Threshold and redundant keys 
| Type         | ToB Severity  |
| ------------ | ------------- |
| Cryptography | Informational |


We plan to document our update process, including the signature steps, and improve them to reduce risk as much as possible. We will consider multiple role keys and thresholds, so specific actions require a quorum, so the leak of a single key is less critical.

#### 10 - Database compaction function could be called more times than expected
| Type               | ToB Severity  |
| ------------------ | ------------- |
| Undefined Behavior | Informational |

This database was not part of the update system, and we [deleted](http://hrwiki.org/wiki/DELETED) it.

#### 11 - All Windows users have read access to Fleet server secret
| Type            | ToB Severity |
| --------------- | ------------ |
| Access Controls | High         |

While this did not impact the security of the update process, it did affect the security of the Fleet enrollment secrets if used on a system where non-administrator accounts were in use. 

This issue was addressed in PR [21](https://github.com/fleetdm/orbit/pull/21) of the old Orbit repository and merged on April 26, 2021. As mentioned in finding #2, we also deployed tools to detect weak permissions on files.

#### 12 - Insufficient documentation of SDDL permissions
| Type                 | ToB Severity |
| -------------------- | ------------ |
| Auditing and Logging | Low          |

While SDDL strings are somewhat cryptic, we can decode them with [PowerShell](https://docs.microsoft.com/en-us/powershell/module/microsoft.powershell.utility/convertfrom-sddlstring?view=powershell-7.2). We obtained SDDL strings from a clean Windows installation with a new osquery installation. We then ensure that users do not have access to secret.txt, to resolve finding #11. 

We have documented the actual permissions expected on April 26, 2021, as you can see in this [commit](https://github.com/fleetdm/fleet/commit/79e82ebcb653b435c6753c68a42cadaa083115f7) in the same PR [21](https://github.com/fleetdm/orbit/pull/21) of the old Orbit repository as for #11.

### Summary
ToB identified a few issues, and we addressed most of them. Most of these impacted the security of the resulting agent installation, such as permission-related issues.

Our goal with this audit was to ensure that our auto-updater mechanism, built with
[TUF](https://theupdateframework.io/), was sound. We believe it is, and we are planning future
improvements to make it more robust and resilient to compromise.

<meta name="pageOrderInSection" value="790">
<meta name="description" value="Explanations of the latest external security audits performed on Fleet software.">
<meta name="maintainedBy" value="hollidayn">
