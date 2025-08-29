# Prevent tampering of Fleet agent and installer settings on Windows

## Introduction
On Windows, users with administrative rights can remove or modify management agents like **Fleet Orbit**.  
Unlike many EDR and DLP tools with built-in tamper protection, Fleet doesn’t attempt to hide or lock itself down. This is intentional—Fleet is built on **openness and transparency**, with enforcement driven by policies you can see and manage.

To keep the agent in place, IT teams can add their own guardrails using a combination of:

- A PowerShell script that sets hardened registry values  
- A Fleet policy that checks those values and reapplies them if missing  
- A Windows configuration profile that blocks MDM unenrollment  

Together, these components create a **self-healing enforcement loop** that ensures protections remain in place, without relying on hidden or opaque mechanisms.

---

## Hardening the installer
Apply registry values via a PowerShell script to help prevent uninstallation or tampering of protected applications.

[Windows hardening PowerShell script](https://github.com/fleetdm/fleet/blob/main/assets/scripts/windows-fleet-hardening.ps1)

---

## Policies in Fleet
A Fleet policy confirms that the hardened registry key exists. If it’s missing, the policy fails, triggering automation to run the script again. Once applied, the policy becomes compliant. If tampering occurs later, the cycle repeats.

[Fleet policy for Windows hardening](https://github.com/fleetdm/fleet/blob/main/assets/policies/windows-fleet-hardening.policies.yml)

**Note:** On first run, this policy intentionally fails to ensure automation executes the hardening script.

---

## Blocking unenrollment
Deploy a configuration profile to prevent devices from unenrolling from MDM.

[Block MDM unenrollment configuration profile](https://github.com/fleetdm/fleet/blob/main/assets/configuration-profiles/BlockMDMUnenrollment.xml)

**Microsoft CSP reference**  
The `AllowManualMDMUnenrollment` CSP is only supported on certain versions of Windows—verify compatibility before deployment.

---

## More advanced approaches
More mature organisations may already be using additional controls to protect against tampering. For example:

- **ADMX-backed CSPs** such as `ADMX_AddRemovePrograms`, which can restrict software removal across the board.  
  - Provides strong protection, but applies to *all* applications, not just the Fleet agent.  

- **AppLocker** or similar application control solutions, which can explicitly block execution of unapproved installers or uninstallers.  
  - Powerful, but requires more careful design and typically broader organisational adoption.  

The **Fleet script + policy + profile** approach offers a lightweight, targeted way to enforce persistence without restricting unrelated applications.

---

## Conclusion
Combining a PowerShell hardening script, a Fleet policy with automation, and a configuration profile gives you dependable protection against tampering with the Fleet agent and other critical installer settings.

Fleet’s open model makes enforcement visible and verifiable—without relying on concealed or fragile mechanisms.

---

Want to learn more about how Fleet approaches transparent, cross-platform device management?  
Visit [fleetdm.com](https://fleetdm.com) or check out the other guides for macOS, Windows, and Linux.

<meta name="articleTitle" value="Prevent tampering of Fleet agent and installer settings on Windows.">
<meta name="authorFullName" value="Adam Baali">
<meta name="authorGitHubUsername" value="AdamBaali">
<meta name="category" value="articles">
<meta name="publishedOn" value="2025-08-29">
<meta name="description" value="On Windows, users with administrative rights can remove or modify management agents like **Fleet Orbit**.">
