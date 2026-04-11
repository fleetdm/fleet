## Overview

### Fleet

Fleet is an Apple-oriented, modern, transparent device management solution with multi-platform support for Linux, macOS, iOS, iPadOS, Windows, Android and Chromebook devices. Fleet has an API-first design with built-in GitOps console management. Fleet is based on open-source technology providing near real-time reporting, comprehensive device control and automated remediation capabilities.

### Jamf

Jamf has evolved over two decades as a management solution focused on Apple devices. Jamf Pro added Android and Chromebook management in the past, removed it, and recently announced support for Android again. Jamf sells a range of products that integrate with Jamf Pro for an additional cost to the Jamf Pro license. Jamf has a large customer base and long history in the Apple device management space.


## Key differences

Fleet and Jamf serve different strategic purposes based on fleet composition and workflow needs.


### Platform support
 
<table style="border-collapse:collapse;width:100%">
  <thead>
    <tr>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px"></th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Fleet</th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Jamf Pro</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">macOS management</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Full MDM lifecycle</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — 20+ year track record</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">iOS / iPadOS management</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Windows management</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#f8d7da">No</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Linux management</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Native osquery agent</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#f8d7da">No</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Android management</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Partner developed solution</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Chromebook management</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#f8d7da">No</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">tvOS / visionOS management</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#f8d7da">No</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Device scoping &amp; targeting</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Dynamic labels, Manual labels, and Host vitals labels</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Smart Groups + Static Groups</td>
    </tr>
  </tbody>
</table>
 
### Enrollment and provisioning
 
<table style="border-collapse:collapse;width:100%">
  <thead>
    <tr>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px"></th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Fleet</th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Jamf Pro</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Zero-touch deployment (ABM/ASM)</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — ABM/ASM + Autopilot</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — ABM/ASM; deep Apple integration</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">End-user IdP auth at Setup Assistant</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — SAML SSO during OOBE; local account pre-filled from IdP</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#fff3cd">Partial — Platform SSO available but less integrated</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Bootstrap apps &amp; scripts during Setup Assistant</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Configure required apps and scripts before device release</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#fff3cd">Partial — PreStage enrollment triggers policies, less granular gating</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">BYOD enrollment</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Incl. Android work profiles</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — User-initiated enrollment</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">MDM migration from another vendor</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Built-in migration workflow</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#fff3cd">Partial — Possible but no built-in migration tool</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Identity provider integration at enrollment</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Okta, Entra, Azure AD, etc.</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Platform SSO; Simplified Setup</td>
    </tr>
  </tbody>
</table>
 
### Identity and access
 
<table style="border-collapse:collapse;width:100%">
  <thead>
    <tr>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px"></th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Fleet</th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Jamf Pro</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">SAML SSO for admin console</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — SP- and IdP-initiated flows</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — SSO for Jamf Pro console</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">SCIM user provisioning &amp; attribute sync</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Provision/deprovision via SCIM with attribute sync</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#fff3cd">Partial — Limited SCIM; primarily manual user management</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">IdP user-to-host mapping</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Sync IdP user attributes to hosts via SCIM</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#fff3cd">Partial — Manual or LDAP-based; no automatic mapping</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Role-based access control (RBAC)</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">SCEP certificate deployment (e.g., Okta Verify + FastPass)</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Deploy SCEP cert profiles for device trust</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — SCEP via AD CS or third-party CA</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Conditional access integration (IdP policy-based block)</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Policy failures trigger IdP conditional access blocks</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#fff3cd">Partial — Requires Jamf Connect or third-party integration</td>
    </tr>
  </tbody>
</table>
 
### Configuration management
 
<table style="border-collapse:collapse;width:100%">
  <thead>
    <tr>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px"></th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Fleet</th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Jamf Pro</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Configuration profile delivery with full confirmation</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Upload custom profiles</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#f8d7da">No</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Declarative Device Management (DDM)</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#fff3cd">Partial — Blueprints framework (Jamf Cloud)</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Enforce disk encryption (FileVault/BitLocker)</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Mac + Windows</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Mac only (FileVault)</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Disk encryption key escrow and recovery</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Keys escrowed in Fleet, retrievable via host details</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — FileVault key escrow in Jamf Pro, retrievable by admin</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Enforce OS updates</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Mac, iOS, Windows</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Mac, iOS; managed software updates</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">OS update ring groups (canary/staged rollout)</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Fleets for Ring 0 and Ring 1 with DDM enforcement</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#fff3cd">Partial — Smart Groups approximate rings, no built-in concept</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Device scoping &amp; targeting</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Labels (dynamic via osquery) + fleets</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Smart Groups + Static Groups</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Local admin account creation and password escrow</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Script-based, credentials retrievable</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#fff3cd">Partial — Requires Jamf Connect, not built into Pro</td>
    </tr>
  </tbody>
</table>
 
### Software management
 
<table style="border-collapse:collapse;width:100%">
  <thead>
    <tr>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px"></th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Fleet</th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Jamf Pro</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">App deployment</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Fleet-maintained apps + custom packages</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — App Catalog + custom packages</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Self-service app installation</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Self Service+ (recently enhanced)</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Volume Purchase Program (VPP / Apps &amp; Books)</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Patch management</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Vulnerability-driven; cross-platform</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — App Installers; macOS &amp; iOS focused</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Pre/post-install scripts for app deployment</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">App install/uninstall/reinstall from admin UI</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Per-host from host details</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Via device management actions</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Script execution</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Cross-platform (Mac, Win, Linux)</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Mac scripts; Bash, Python, etc.</td>
    </tr>
  </tbody>
</table>
 
### Security and compliance
 
<table style="border-collapse:collapse;width:100%">
  <thead>
    <tr>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px"></th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Fleet</th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Jamf Pro</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Vulnerability detection (CVEs)</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Built-in; CISA KEV; cross-platform</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#fff3cd">Partial — Basic in Pro; deep scanning requires Jamf Protect ($)</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Compliance benchmarks (CIS / STIG)</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — CIS queries publicly available</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Compliance Benchmarks (mSCP) in Pro</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Compliance policy dashboard (per-host pass/fail)</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Per-host pass/fail on Policies page</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#fff3cd">Partial — Smart Groups imply compliance, no unified dashboard</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Endpoint detection / threat monitoring</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes (built-in)</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#fff3cd">Partial — Requires Jamf Protect (separate purchase)</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">File integrity monitoring (FIM)</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes - evented tables (built-in)</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#fff3cd">Partial — Requires Jamf Protect</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">SIEM integration</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Custom log destinations; included</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Pro event logs; richer with Protect ($)</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Lock / wipe commands</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes</td>
    </tr>
  </tbody>
</table>
 
### Visibility and reporting
 
<table style="border-collapse:collapse;width:100%">
  <thead>
    <tr>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px"></th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Fleet</th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Jamf Pro</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Real-time device queries</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes - Live queries</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#fff3cd">Partial — Inventory on check-in schedule</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Hardware &amp; software inventory</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Extensive</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Comprehensive Apple inventory</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Application inventory and patch status view</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Per-host and fleet-wide; flags hosts below target version</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — App inventory; patch status via App Installers</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Custom data collection</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Custom SQL queries across 300+ tables (built-in)</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#fff3cd">Partial — Extension attributes (scripts)</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Offline device alerting (webhooks)</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Configurable offline threshold, alerts fire automatically</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#fff3cd">Partial — Webhook notifications available, less granular thresholds</td>
    </tr>
  </tbody>
</table>
 
### Remediation and automation
 
<table style="border-collapse:collapse;width:100%">
  <thead>
    <tr>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px"></th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Fleet</th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Jamf Pro</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Policy-triggered auto-remediation</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Attach remediation script to policy, auto-executes on failure</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#fff3cd">Partial — Smart Groups trigger policies, no direct policy→script link</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">On-demand script execution from admin UI</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Per-host from host details, real-time output</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Remote commands available for macOS</td>
    </tr>
  </tbody>
</table>
 
### Offboarding and lifecycle
 
<table style="border-collapse:collapse;width:100%">
  <thead>
    <tr>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px"></th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Fleet</th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Jamf Pro</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">User deprovisioning via IdP (SCIM)</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — SCIM removes host-user mapping and revokes access</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#fff3cd">Partial — Manual user deletion, limited IdP-driven deprovisioning</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Device re-assignment between users/teams</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Transfer device to new fleet, profiles auto-applied</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Move between sites/groups, profiles re-applied</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">End-user transparency</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Scope transparency; open source</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#fff3cd">Partial — Limited native transparency features</td>
    </tr>
  </tbody>
</table>
 
### Architecture and operations
 
<table style="border-collapse:collapse;width:100%">
  <thead>
    <tr>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px"></th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Fleet</th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Jamf Pro</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">GitOps / infrastructure as code</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — First-class; YAML/Git-based</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#fff3cd">Partial — IBM Terraform-based, not all functionality available</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">API-first architecture</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Unified REST API; all features</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#fff3cd">Partial — Multiple APIs; GUI-first design</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Self-hosted deployment</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — On-prem, cloud, air-gapped</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#fff3cd">Partial — functionality not as complete as cloud</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Managed cloud hosting (SaaS)</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Jamf Cloud</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Open-source / source-available code</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — 100% on GitHub</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#f8d7da">No — Proprietary</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Audit logging</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes</td>
    </tr>
  </tbody>
</table>
 
### Pricing and licensing
 
<table style="border-collapse:collapse;width:100%">
  <thead>
    <tr>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px"></th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Fleet</th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Jamf Pro</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Free tier available</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Core features; unlimited hosts</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#f8d7da">No — 14-day free trial only</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Pricing model</td>
      <td style="border:1px solid #ccc;padding:8px">$7/host/month (Premium); all features included</td>
      <td style="border:1px solid #ccc;padding:8px">~$3.67–$7.89/device/month; varies by device type</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">All-inclusive security (vuln, EDR, FIM)</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Single license covers everything</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#f8d7da">No — Protect, Connect, ETP sold separately</td>
    </tr>
  </tbody>
</table>
 
### Support and ecosystem
 
<table style="border-collapse:collapse;width:100%">
  <thead>
    <tr>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px"></th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Fleet</th>
      <th style="width:33.3%;border:1px solid #ccc;padding:8px">Jamf Pro</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Vendor support channels</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Email, phone, video (Premium); community Slack</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Chat, email, phone; premium services available</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Community &amp; ecosystem maturity</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Growing — Active open-source communities &amp; ecosystems</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Mature — Large user base; Jamf Nation; 20+ years</td>
    </tr>
    <tr>
      <td style="border:1px solid #ccc;padding:8px">Apple relationship &amp; day-zero OS support</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Apple-oriented; tracks releases</td>
      <td style="border:1px solid #ccc;padding:8px;background-color:#d4edda">Yes — Close Apple partnership; historically day-zero</td>
    </tr>
  </tbody>
</table>


## Device management workflow comparisons

### Enrollment and provisioning

Both Fleet and Jamf Pro support Apple Business / School Manager integration for zero-touch deployment (typically meaning that devices ship directly to end users and enroll via an automated process on first boot.)

Both solutions also provide options for deploying MDM enrollment profiles via supervision and settings that prevent end users from removing management and MDM configuration profiles without authorization, giving organizations strong enforcement controls to match requirements and comply with standards.

### Configuration management

Jamf allows admins to create Smart or Static groups as the mechanism for controlling the scope of management automations and configuration profile delivery. Jamf includes configuration profile templates for building profiles to deliver common settings.

Fleet directs Apple device admins to iMazing Profile Creator for building configuration profiles. Fleet uses fleets and labels to assign and deliver configuration profiles to devices. Labels can be manual (e.g., arbitrary assignment by serial number), dynamic (based on device state assessed) or set via "Host vitals" (i.e., using server-side attributes of a device like IdP group membership.) Validation of configuration profile delivery is obtained separately from MDM for complete assurance of device state.

### Software management

Jamf provides an App Catalog and integrated Apps and Books distribution for volume purchasing with scoping based on Smart or Static Groups.

Fleet provides software management through Fleet-maintained apps and also includes Apps and Books distribution for volume purchasing from App Stores.

Both solutions provide the ability to upload custom software packages for installation and scripting capabilities for automation. This ensures that complex software (e.g., security applications like [CrowdStrike](/guides/deploying-crowdstrike-with-fleet)) can be customized during installation.

### Security and compliance

Jamf Pro is Jamf's flagship device management solution but it is not an out-of-the-box security solution. Jamf Pro enables management of FileVault disk encryption, Gatekeeper, and other Apple features which help to keep devices secure, however, Jamf's advanced security offerings like Jamf Protect and Jamf Executive Threat Protection are separate products from Jamf Pro that must be purchased separately at additional cost.

Jamf's security products make use of Apple's native Endpoint Security Framework for EDR and telemetry collection enabling security monitoring and SIEM integration capabilities, but, this potentially means detection and compliance are more expensive when using Jamf's full product line.

Fleet approaches security and compliance through built-in software vulnerability detection and the power of built-in osquery reporting combined with automation capabilities for enforcing and remediating controls on top of complete support for Apple's MDM specification (which includes control over basic security features like FileVault and Gatekeeper.)

These combined Fleet capabilities make it straight-forward to enforce compliance baselines using frameworks like [CIS](/guides/cis-benchmarks) or STIG. Threat detection in Fleet works through the creation of queries to find attributes, device processes, file systems, network configurations, malware detection via [YARA-based signature matching](/guides/remote-yara-rules), and vulnerability intelligence. Security monitoring, data collection, SIEM integration, and all other Fleet capabilities are included under a single license at no additional cost. Fleet provides visibility into software inventories, file system events, connected hardware, firewall status, and virtually any imaginable attribute of any device via the [Fleet osquery data table schema](/tables).

## Single-platform vs. multi-platform support

Whether or not your device management solution has multi-platform support capability determines if consolidation of your device management tooling is possible. Maintaining multiple single-platform solutions can be complex and expensive. Multiple solutions may mean multiple, separate IT teams and it definitely means managing multiple contract renewals.

Jamf provides purpose-built management capabilities across Apple's device range but really only specializes in Apple, with recently announced Android support.

Fleet offers comprehensive multi-platform coverage for Linux, macOS, iOS, iPadOS, Windows, Android and Chromebook devices from a single console.

## FAQ

#### What is the main difference between a single-platform device management solution and a multi-platform device management solution?

Specialized MDM solutions focus on one device ecosystem. multi-platform MDM solutions provide unified management across different operating systems from a single console. [Try Fleet](/try-fleet) to see how multi-platform management can work in your environment.

#### Can multi-platform device management solutions manage Apple devices as effectively as Apple-specialized platforms?

Fleet is an Apple-oriented device management solution. Though it is multi-platform, Fleet provides management capabilities at parity with solutions like Jamf for most use cases including zero-touch, automated enrollment through Apple Business or School Manager, delivery of MDM configuration profiles, MDM commands, Declarative Device Management support, software management, script execution and strict control over scoping management objects to the right devices.

#### What should I consider when comparing MDM costs?

Both Fleet and Jamf Pro offer per-device subscription pricing with costs varying based on fleet size and requirements. Organizations should consider implementation effort, training needs, and ROI savings through tool consolidation when choosing to move to a new device management solution. More specialized training and support may be required when maintaining multiple device management solutions. multi-platform device management solutions enable tool consolidation that can offset per-device costs.

In addition to device management feature parity with Jamf, Fleet includes capabilities that Jamf does not like GitOps console management, software vulnerability reporting, osquery data collection, and SIEM integration under a single license per device at no additional cost. These inclusions may allow an organization to trim costs even further when consolidating tools by moving to Fleet.

#### How long does it take to implement device management across different platforms?

Implementation and migration timelines vary based on fleet size and organizational requirements. Fleet offers world-class customer support and professional services to assist organizations with migration. End user migration / enrollment workflows are available for all computer platforms Fleet supports (mobile device MDM migrations are limited by product vendor capabilities and can therefore be more challenging to do.) [Schedule a demo](/contact) to discuss specific implementation timelines for your environment.



<meta name="articleTitle" value="Fleet vs. Jamf">
<meta name="authorFullName" value="Brock Walters">
<meta name="authorGitHubUsername" value="nonpunctual">
<meta name="articleSlugInCategory" value="jamf-vs-fleet"> 
<meta name="introductionTextBlockOne" value="Organizations managing Apple devices face a choice: pick one of a number of available Apple device management solutions, or, a solution with multi-platform capabilities."> 
<meta name="introductionTextBlockTwo" value="This guide compares and contrasts the capabilities of Fleet with Jamf Pro, highlighting deployment approaches and buying decision criteria."> 
<meta name="category" value="comparison">
<meta name="publishedOn" value="2026-01-27">
<meta name="description" value="This guide compares and contrasts the capabilities of Fleet with Jamf Pro, highlighting deployment approaches and buying decision criteria.">
