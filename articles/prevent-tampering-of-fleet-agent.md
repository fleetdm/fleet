# Prevent tampering of Fleet Orbit on Windows

## Introduction

On Windows, users with administrative rights can remove or modify management agents like Fleet Orbit. Unlike many EDR and DLP tools with built-in tamper protection, Fleet does not attempt to hide or lock itself down. This is intentional. Fleet is built on openness and transparency, with enforcement driven by policies you can see and manage.

But that doesn’t mean you’re left unprotected. To keep the agent in place, IT teams can add guardrails using a combination of:

- A Fleet policy with automation that enforces protection by running a script if tampering occurs  
- A PowerShell script that applies hardened registry values (executed automatically by the policy)  
- A Windows configuration profile that blocks MDM unenrollment (a separate control that complements the above)  

Together, these components create a self-healing enforcement loop that ensures protections remain in place, without relying on hidden or opaque mechanisms.

## Hardening the installer

One way to harden the installer is to apply registry values through a PowerShell script. These values help prevent uninstallation or tampering of protected applications.

[Windows hardening PowerShell script](https://github.com/fleetdm/fleet/blob/main/assets/scripts/windows-fleet-hardening.ps1)

## Policies in Fleet

A Fleet policy confirms that the hardened registry key exists. If the key is missing, the policy fails, triggering automation to rerun the script. Once applied, the policy becomes compliant. If tampering occurs later, the cycle repeats.

[Fleet policy for Windows hardening](https://github.com/fleetdm/fleet/blob/main/assets/policies/windows-fleet-hardening.policies.yml)

> Note: On first run, this policy intentionally fails to ensure automation executes the hardening script.

## Blocking unenrollment

A Windows configuration profile can prevent devices from unenrolling from MDM. This is a separate measure from the policy and script but adds another layer of protection.

[Block MDM unenrollment configuration profile](https://github.com/fleetdm/fleet/blob/main/assets/configuration-profiles/BlockMDMUnenrollment.xml)

[Microsoft CSP reference](https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-csp-experience#allowmanualmdmunenrollment): The `AllowManualMDMUnenrollment` CSP is only supported on certain versions of Windows. Verify compatibility before deployment.

## Advanced approaches

Some organisations may already use additional controls to protect against tampering:

- **ADMX-backed CSPs**  
  The `ADMX_AddRemovePrograms` CSP can restrict software removal across all applications, not just Fleet.
- **Application control solutions**  
  Tools like AppLocker can block the execution of unapproved installers or uninstallers. Powerful, but they require careful design and broader adoption.

The policy and script combination provides a self-healing loop, while the configuration profile and advanced approaches add complementary protection.

## Conclusion

By combining a Fleet policy with automation, a PowerShell hardening script, and a configuration profile, admins can enforce dependable protection against tampering with the Fleet Orbit agent and installer settings.

Fleet’s open model makes enforcement visible and verifiable without relying on concealed or fragile mechanisms.

Want to learn more about how Fleet approaches transparent, cross-platform device management?  
Visit [fleetdm.com](https://fleetdm.com) or check out the other [guides for macOS, Windows, and Linux](https://fleetdm.com/guides).

<meta name="articleTitle" value="Prevent tampering of Fleet Orbit on Windows">
<meta name="authorFullName" value="Adam Baali">
<meta name="authorGitHubUsername" value="AdamBaali">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-08-29">
<meta name="description" value="Combine a Fleet policy, a PowerShell script, and a Windows configuration profile to prevent tampering with Fleet Orbit.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-windows-hardening-cover-800x450@2x.png">
