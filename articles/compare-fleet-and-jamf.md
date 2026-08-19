## Overview

### Fleet

Fleet is an Apple-oriented, modern, transparent device management solution with multi-platform support for Linux, macOS, iOS, iPadOS, Windows, Android and Chromebook devices. Fleet has an API-first design with built-in GitOps console management. Fleet is based on open-source technology providing near real-time reporting, comprehensive device control and automated remediation capabilities.

### Jamf

Jamf has evolved over two decades as a management solution focused on Apple devices. Jamf Pro added Android and Chromebook management in the past, removed it, and recently announced support for Android again. Jamf sells a range of products that integrate with Jamf Pro for an additional cost to the Jamf Pro license. Jamf has a large customer base and long history in the Apple device management space.


## Key differences

Fleet and Jamf serve different strategic purposes based on fleet composition and workflow needs.


### Platform support
 
| | Fleet | Jamf Pro |
| --- | --- | --- |
| **macOS management** | ✅ Full MDM lifecycle | ✅ 20+ year track record |
| **iOS / iPadOS management** | ✅ | ✅ |
| **Windows management** | ✅ | ❌ |
| **Linux management** | ✅ Native osquery agent | ❌ |
| **Android management** | ✅ | ✅ Partner developed solution |
| **Chromebook management** | ✅ | ❌ |
| **tvOS / visionOS management** | ❌ | ✅ |
| **Device scoping & targeting** | ✅ Dynamic labels, Manual labels, and Host vitals labels | ✅ Smart Groups + Static Groups |
 
### Enrollment and provisioning
 
| | Fleet | Jamf Pro |
| --- | --- | --- |
| **Zero-touch deployment (ABM/ASM)** | ✅ ABM/ASM + Autopilot | ✅ ABM/ASM; deep Apple integration |
| **End-user IdP auth at Setup Assistant** | ✅ SAML SSO during OOBE; local account pre-filled from IdP | ⚠️ Platform SSO available but less integrated |
| **Bootstrap apps & scripts during Setup Assistant** | ✅ Configure required apps and scripts before device release | ⚠️ PreStage enrollment triggers policies, less granular gating |
| **BYOD enrollment** | ✅ Incl. Android work profiles | ✅ User-initiated enrollment |
| **MDM migration from another vendor** | ✅ Built-in migration workflow | ⚠️ Possible but no built-in migration tool |
| **Identity provider integration at enrollment** | ✅ Okta, Entra, Azure AD, etc. | ✅ Platform SSO; Simplified Setup |
 
### Identity and access
 
| | Fleet | Jamf Pro |
| --- | --- | --- |
| **SAML SSO for admin console** | ✅ SP- and IdP-initiated flows | ✅ SSO for Jamf Pro console |
| **SCIM user provisioning & attribute sync** | ✅ Provision/deprovision via SCIM with attribute sync | ⚠️ Limited SCIM; primarily manual user management |
| **IdP user-to-host mapping** | ✅ Sync IdP user attributes to hosts via SCIM | ⚠️ Manual or LDAP-based; no automatic mapping |
| **Role-based access control (RBAC)** | ✅ | ✅ |
| **SCEP certificate deployment (e.g., Okta Verify + FastPass)** | ✅ Deploy SCEP cert profiles for device trust | ✅ SCEP via AD CS or third-party CA |
| **Conditional access integration (IdP policy-based block)** | ✅ Policy failures trigger IdP conditional access blocks | ⚠️ Requires Jamf Connect or third-party integration |
 
### Configuration management
 
| | Fleet | Jamf Pro |
| --- | --- | --- |
| **Configuration profile delivery with full confirmation** | ✅ Upload custom profiles | ❌ |
| **Declarative Device Management (DDM)** | ✅ | ⚠️ Blueprints framework (Jamf Cloud) |
| **Enforce disk encryption (FileVault/BitLocker)** | ✅ Mac + Windows | ✅ Mac only (FileVault) |
| **Disk encryption key escrow and recovery** | ✅ Keys escrowed in Fleet, retrievable via host details | ✅ FileVault key escrow in Jamf Pro, retrievable by admin |
| **Enforce OS updates** | ✅ Mac, iOS, Windows | ✅ Mac, iOS; managed software updates |
| **OS update ring groups (canary/staged rollout)** | ✅ Fleets for Ring 0 and Ring 1 with DDM enforcement | ⚠️ Smart Groups approximate rings, no built-in concept |
| **Device scoping & targeting** | ✅ Labels (dynamic via osquery) + fleets | ✅ Smart Groups + Static Groups |
| **Local admin account creation and password escrow** | ✅ Script-based, credentials retrievable | ⚠️ Requires Jamf Connect, not built into Pro |
 
### Software management
 
| | Fleet | Jamf Pro |
| --- | --- | --- |
| **App deployment** | ✅ Fleet-maintained apps + custom packages | ✅ App Catalog + custom packages |
| **Self-service app installation** | ✅ | ✅ Self Service+ (recently enhanced) |
| **Volume Purchase Program (VPP / Apps & Books)** | ✅ | ✅ |
| **Patch management** | ✅ Vulnerability-driven; cross-platform | ✅ App Installers; macOS & iOS focused |
| **Pre/post-install scripts for app deployment** | ✅ | ✅ |
| **App install/uninstall/reinstall from admin UI** | ✅ Per-host from host details | ✅ Via device management actions |
| **Script execution** | ✅ Cross-platform (Mac, Win, Linux) | ✅ Mac scripts; Bash, Python, etc. |
 
### Security and compliance
 
| | Fleet | Jamf Pro |
| --- | --- | --- |
| **Vulnerability detection (CVEs)** | ✅ Built-in; CISA KEV; cross-platform | ⚠️ Basic in Pro; deep scanning requires Jamf Protect ($) |
| **Compliance benchmarks (CIS / STIG)** | ✅ CIS queries publicly available | ✅ Compliance Benchmarks (mSCP) in Pro |
| **Compliance policy dashboard (per-host pass/fail)** | ✅ Per-host pass/fail on Policies page | ⚠️ Smart Groups imply compliance, no unified dashboard |
| **Endpoint detection / threat monitoring** | ✅ Built-in | ⚠️ Requires Jamf Protect (separate purchase) |
| **File integrity monitoring (FIM)** | ✅ evented tables (built-in) | ⚠️ Requires Jamf Protect |
| **SIEM integration** | ✅ Custom log destinations; included | ✅ Pro event logs; richer with Protect ($) |
| **Lock / wipe commands** | ✅ | ✅ |
 
### Visibility and reporting
 
| | Fleet | Jamf Pro |
| --- | --- | --- |
| **Real-time device queries** | ✅ Live queries | ⚠️ Inventory on check-in schedule |
| **Hardware & software inventory** | ✅ Extensive | ✅ Comprehensive Apple inventory |
| **Application inventory and patch status view** | ✅ Per-host and fleet-wide; flags hosts below target version | ✅ App inventory; patch status via App Installers |
| **Custom data collection** | ✅ Custom SQL queries across 300+ tables (built-in) | ⚠️ Extension attributes (scripts) |
| **Offline device alerting (webhooks)** | ✅ Configurable offline threshold, alerts fire automatically | ⚠️ Webhook notifications available, less granular thresholds |
 
### Remediation and automation
 
| | Fleet | Jamf Pro |
| --- | --- | --- |
| **Policy-triggered auto-remediation** | ✅ Attach remediation script to policy, auto-executes on failure | ⚠️ Smart Groups trigger policies, no direct policy→script link |
| **On-demand script execution from admin UI** | ✅ Per-host from host details, real-time output | ✅ Remote commands available for macOS |
 
### Offboarding and lifecycle
 
| | Fleet | Jamf Pro |
| --- | --- | --- |
| **User deprovisioning via IdP (SCIM)** | ✅ SCIM removes host-user mapping and revokes access | ⚠️ Manual user deletion, limited IdP-driven deprovisioning |
| **Device re-assignment between users/teams** | ✅ Transfer device to new fleet, profiles auto-applied | ✅ Move between sites/groups, profiles re-applied |
| **End-user transparency** | ✅ Scope transparency; open source | ⚠️ Limited native transparency features |
 
### Architecture and operations
 
| | Fleet | Jamf Pro |
| --- | --- | --- |
| **GitOps / infrastructure as code** | ✅ First-class; YAML/Git-based | ⚠️ IBM Terraform-based, not all functionality available |
| **API-first architecture** | ✅ Unified REST API; all features | ⚠️ Multiple APIs; GUI-first design |
| **Self-hosted deployment** | ✅ On-prem, cloud, air-gapped | ⚠️ Functionality not as complete as cloud |
| **Managed cloud hosting (SaaS)** | ✅ | ✅ Jamf Cloud |
| **Open-source / source-available code** | ✅ 100% on GitHub | ❌ Proprietary |
| **Audit logging** | ✅ | ✅ |
 
### Pricing and licensing
 
| | Fleet | Jamf Pro |
| --- | --- | --- |
| **Free tier available** | ✅ Core features; unlimited hosts | ❌ 14-day free trial only |
| **Pricing model** | $7/host/month (Premium); all features included | ~$3.67–$7.89/device/month; varies by device type |
| **All-inclusive security (vuln, EDR, FIM)** | ✅ Single license covers everything | ❌ Protect, Connect, ETP sold separately |
 
### Support and ecosystem
 
| | Fleet | Jamf Pro |
| --- | --- | --- |
| **Vendor support channels** | ✅ Email, phone, video (Premium); community Slack | ✅ Chat, email, phone; premium services available |
| **Community & ecosystem maturity** | ✅ Growing — Active open-source communities & ecosystems | ✅ Mature — Large user base; Jamf Nation; 20+ years |
| **Apple relationship & day-zero OS support** | ✅ Apple-oriented; tracks releases | ✅ Close Apple partnership; historically day-zero |
 

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
