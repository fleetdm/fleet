## Overview

### Fleet

Fleet is an Apple-oriented, modern, transparent device management solution with cross-platform support for Linux, macOS, iOS, iPadOS, Windows, Android and Chromebook devices. Fleet has an API-first design with built-in GitOps console management. Fleet is based on open-source technology providing near real-time reporting, comprehensive device control and automated remediation capabilities.

### Jamf

Jamf has evolved over two decades as a management solution focused on Apple devices. Jamf Pro added Android and Chromebook management in the past, removed it, and recently announced support for Android again. Jamf sells a range of products that integrate with Jamf Pro for an additional cost to the Jamf Pro license. Jamf has a large customer base and long history in the Apple device management space.


## Key differences

Fleet and Jamf serve different strategic purposes based on fleet composition and workflow needs.

|                    | Fleet                                                                                                     | Jamf                                                                                          |
| ------------------ | --------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| Architecture       | API-first design, unified API, osquery validation and data collection                                     | GUI-first, multiple APis                                                                      |
| Development        | Open-core, public code, contributions welcome                                                             | Proprietary, slow customer feature request intake                                             |
| Console Management | GUI or GitOps / configuration-as-code console management                                                  | GUI-first, no comprehensive built-in version history or console state management with code    |
| Platform Support   | Linux, macOS, iOS, iPadOS, Windows, Android, Chromebook                                                   | macOS, iOS, iPadOS, tvOS, visionOS, Android                                                   |
| Apple MDM          | MDM + DDM protocol, API supports any arbitrary MDM command                                                | MDM protocol, DDM protocol supported in Jamf Cloud only, only specific MDM commands available |
| Security           | On-demand osquery data collection, YARA rules, event monitoring, CIS benchmark queries publicly available | Deep security features require additional product purchases                                   |



## Device management workflow comparisons

### Enrollment and provisioning

Both Fleet and Jamf Pro support Apple Business / School Manager integration for zero-touch deployment (typically meaning that devices ship directly to end users and enroll via an automated process on first boot.)

Both solutions also provide options for deploying MDM enrollment profiles via supervision and settings that prevent end users from removing management and MDM Configuration Profiles without authorization, giving organizations strong enforcement controls to match requirements and comply with standards.

### Configuration management

Jamf allows admins to create Smart or Static groups as the mechanism for controlling the scope of management automations and Configuration Profile delivery. Jamf includes Configuration Profile templates for building profiles to deliver common settings.

Fleet directs Apple device admins to iMazing Profile Creator for building Configuration Profiles. Fleet uses Teams and Labels to assign and deliver Configuration Profiles to devices. Labels can be manual (e.g., arbitrary assignment by serial number), dynamic (based on device state assessed via osquery) or set via "Host vitals" (i.e., using server-side attributes of a device like IdP group membership.) Validation of Configuration Profile delivery is obtained separately from MDM via osquery for complete assurance of device state.

### Software management

Jamf provides an App Catalog and integrated Apps and Books distribution for volume purchasing with scoping based on Smart or Static Groups.

Fleet provides software management through Fleet-maintained apps and also includes Apps and Books distribution for volume purchasing from App Stores.

Both solutions provide the ability to upload custom software packages for installation and scripting capabilities for automation. This ensures that complex software (e.g., security applications like CrowdStrike) can be customized during installation.

### Security and compliance

Jamf Pro is Jamf's flagship device management solution but it is not an out-of-the-box security solution. Jamf Pro enables management of FileVault disk encryption, Gatekeeper and other Apple features which help to keep devices secure, however, Jamf's advanced security offerings like Jamf Protect and Jamf Executive Threat Protection are separate products from Jamf Pro that must be purchased separately at additional cost.

Jamf's security products make use of Apple's native Endpoint Security Framework for EDR and telemetry collection enabling security monitoring and SIEM integration capabilities, but, this potentially means detection and compliance are more expensive when using Jamf's full product line.

Fleet approaches security and compliance through built-in software vulnerability detection and the power of osquery reporting combined with automation capabilities for enforcing and remediating controls on top of complete support for Apple's MDM specification (which includes control over basic security features like FileVault and Gatekeeper.)

These combined Fleet capabilities make it straight-forward to enforce compliance baselines using frameworks like CIS or STIG. Threat detection in Fleet works through the creation of queries to find attributes, device processes, file systems, network configurations, malware detection via YARA-based signature matching, and vulnerability intelligence. Security monitoring, data collection, SIEM integration and all other Fleet capabilities are included under a single license at no additional cost. Fleet provides visibility into software inventories, file system events, connected hardware, firewall status and virtually any imaginable attribute of any device via the Fleet osquery data table schema.

## Single-platform vs. cross-platform support

Whether or not your device management solution has cross-platform support capability determines if consolidation of your device management tooling is possible. Maintaining multiple single-platform solutions can be complex and expensive. Multiple solutions may mean multiple, separate IT teams and it definitely means managing multiple contract renewals.

Jamf provides purpose-built management capabilities across Apple's device range but really only specializes in Apple, with recently announced Android support.

Fleet offers comprehensive cross-platform coverage for Linux, macOS, iOS, iPadOS, Windows, Android and Chromebook devices from a single console.

## FAQ

#### What's the main difference between a single-platform device management solution and a cross-platform device management solution?

Specialized MDM solutions focus on one device ecosystem. Cross-platform MDM solutions provide unified management across different operating systems from a single console. [Try Fleet](/register) to see how cross-platform management can work in your environment.

#### Can cross-platform device management solutions manage Apple devices as effectively as Apple-specialized platforms?

Fleet is an Apple-oriented device management solution. Though it is cross-platform, Fleet provides management capabilities at parity with solutions like Jamf for most use cases including zero-touch, automated enrollment through Apple Business or School Manager, delivery of MDM Configuration Profiles, MDM commands, Declarative Device Management support, software management, script execution and strict control over scoping management objects to the right devices.

#### What should I consider when comparing MDM costs?

Both Fleet and Jamf Pro offer per-device subscription pricing with costs varying based on fleet size and requirements. Organizations should consider implementation effort, training needs, and ROI savings through tool consolidation when choosing to move to a new device management solution. More specialized training and support may be required when maintaining multiple device management solutions. Cross-platform device management solutions enable tool consolidation that can offset per-device costs.

In addition to device management feature parity with Jamf, Fleet includes capabilities that Jamf does not like GitOps console management, software vulnerability reporting, osquery data collection and SIEM integration under a single license per device at no additional cost. These inclusions may allow an organization to trim costs even further when consolidating tools by moving to Fleet.

#### How long does it take to implement device management across different platforms?

Implementation and migration timelines vary based on fleet size and organizational requirements. Fleet offers world-class customer support and professional services to assist organizations with migration. End user migration / enrollment workflows are available for all computer platforms Fleet supports (mobile device MDM migrations are limited by product vendor capabilities and can therefore be more challenging to do.) [Schedule a demo](/contact) to discuss specific implementation timelines for your environment.



<meta name="articleTitle" value="Fleet vs. Jamf"> 
<meta name="articleSubtitle" value="How to choose the right MDM">
<meta name="authorFullName" value="FleetDM">
<meta name="authorGitHubUsername" value="fleet-release">
<meta name="articleSlugInCategory" value="jamf"> 
<meta name="introductionTextBlockOne" value="Organizations managing Apple devices face a choice: pick one of a number of available Apple device management solutions, or, a solution with cross-platform capabilities."> 
<meta name="introductionTextBlockTwo" value="This guide compares and contrasts the capabilities of Fleet with Jamf Pro, highlighting deployment approaches and buying decision criteria."> 
<meta name="category" value="compare">
<meta name="publishedOn" value="2026-01-27">
<meta name="description" value="This guide compares and contrasts the capabilities of Fleet with Jamf Pro, highlighting deployment approaches and buying decision criteria.">
