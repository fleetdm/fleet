# Security audits
This page contains explanations of the latest external security audits performed on Fleet software.

## August 2021 security of Orbit auto-updater

Back in 2021, when Orbit was still new, alpha, and likely not used by anyone but us here at Fleet, we contracted Trail of Bits (ToB) to have them review the security of the auto-updater portion of it.

For more context around why we did this, please see this [post](https://blog.fleetdm.com/) on the Fleet blog.

**AN ACTUAL LINK TO THE POST WILL BE ADDED AS SOON AS BLOG IS PUBLIC.**

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
