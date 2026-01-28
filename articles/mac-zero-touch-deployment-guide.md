# Mac zero-touch deployment: How to automate device provisioning with ADE

Shipping Mac devices to remote employees typically means either extensive IT hands-on configuration or sending users inadequately secured machines. Manual provisioning doesn't scale when device counts reach hundreds or thousands, and misconfigured devices create security gaps that persist for months. This guide covers what zero-touch deployment is, how the technical architecture works, and practical setup considerations for Mac fleets.

## What is zero-touch deployment for Mac?

Zero-touch deployment lets organizations ship Mac devices directly from vendors to end users without IT physically handling each machine. When the user opens the box and powers on the device, it automatically queries Apple's servers, receives its MDM assignment, enrolls itself, and applies the organization's security policies and configuration profiles. The entire provisioning process happens without IT teams manually intervening.

The technology centers on Apple's Automated Device Enrollment (formerly DEP), which links a device's serial number to your MDM server through Apple Business Manager. When powered on, devices query Apple's activation servers to determine their assigned MDM service and automatically enroll. For macOS 14 and later, if devices don't enroll during first setup, they display a full-screen setup experience that enforces enrollment, preventing users from bypassing organizational control.

This automation can eliminate the traditional imaging workflow where IT teams received shipments, unboxed devices, connected them to imaging stations, installed base configurations, and then shipped them to users. Instead, devices go directly from the vendor to employees, arriving pre-configured and ready for immediate use.

## Why organizations need zero-touch deployment

Remote work has fundamentally changed device provisioning expectations. Managing distributed workforces means teams can't rely on physical device access for configuration, yet security requirements have only increased. Zero-touch deployment addresses several operational and security challenges that traditional provisioning workflows can't handle efficiently.

Organizations adopting zero-touch deployment typically see improvements across several areas:

* **Faster device provisioning:** Devices ship directly from vendors to users, eliminating the delay of routing through IT facilities. Users can typically start working immediately after unboxing rather than waiting days or weeks for IT-configured devices.  
* **Consistent security baselines:** Devices receive identical configuration profiles automatically, removing the variability that manual provisioning introduces. Configuration drift becomes less likely when devices enroll with known-good settings from first boot.  
* **Reduced IT workload:** IT teams don't touch devices physically, freeing time for higher-value projects. A 500-device deployment that might have required weeks of hands-on work happens automatically.  
* **Improved remote user experience:** Users receive devices that typically work immediately without requiring technical setup knowledge. The enrollment process can skip most Setup Assistant panes, getting users to productivity faster.  
* **Better compliance audit trails:** MDM systems automatically log enrollment timestamps, applied policies, and configuration changes. Compliance teams can generate reports showing exactly when and how devices were provisioned without reconstructing manual processes.

These provisioning, security, and efficiency improvements compound when managing geographically distributed teams. Organizations with offices across multiple countries can provision devices in each location using the same automated workflow, maintaining consistent security posture regardless of where devices ship.

## How zero-touch deployment works

Mac zero-touch deployment operates through a three-tier architecture: Apple Business Manager as the enrollment authority, the MDM server as the policy distributor, and the Mac itself as an active participant in provisioning.

### Device registration and assignment

Devices must be purchased from Apple or participating Apple Authorized Resellers to be automatically registered in Apple Business Manager. The vendor registers devices to your organization's Apple Business Manager account during purchase, linking each serial number to that organization.

After devices appear in Apple Business Manager, you assign them to the MDM server through the web portal. This assignment links serial numbers with the MDM instance through a "virtual MDM server" configuration, establishing the trust relationship for automatic enrollment.

### First boot and enrollment sequence

When the user powers on the device, it queries Apple servers to determine MDM assignment. If assigned, Apple's servers direct the device to contact your MDM service, which responds with an enrollment profile and initial configuration payloads. You can configure which Setup Assistant panes to skip using the Auto Advance key.

For macOS 14 and later, devices registered to ABM that don't enroll during first setup display a full-screen experience enforcing enrollment, preventing users from bypassing organizational control.

### Profile delivery and configuration

After enrollment, your MDM server delivers configuration profiles and security settings. Bootstrap packages deploy automatically during enrollment, including essential applications, certificates, and system configurations for immediate productivity.

With Automated Device Enrollment, enrollment profiles can be configured as mandatory and non-removable, preventing users from unenrolling devices. ADE also provides access to all available Apple-defined payloads and restrictions, offering broader management capabilities than standard enrollment methods.

## Prerequisites and setup considerations

Successfully implementing zero-touch deployment depends on having the right infrastructure components. Missing prerequisites are a common source of enrollment failures, so verification before deployment saves significant troubleshooting time.

### Required infrastructure

You need an Apple Business Manager account with organizational verification. Your organization must complete Apple's verification process before devices can enroll through ADE. This process requires a valid D-U-N-S number and verification of organizational details by Apple representatives. Verification is separate from ABM account creation and typically takes 24-48 hours to complete. Missing this step is a common cause of ADE enrollment failures, as devices won't appear in your ABM account until verification is approved.

Your MDM server must be added to Apple Business Manager as a virtual MDM server, establishing the trust relationship for device assignments. Your MDM tool must support Automated Device Enrollment with mandatory, non-removable enrollment profiles and the Auto Advance key for Setup Assistant customization.

Network connectivity requires internet access to contact Apple's activation servers and MDM endpoints during first boot. Fully automated deployment without any user interaction requires Ethernet connectivity during Auto Advance. Wi-Fi-only deployments typically require users to select the network during Setup Assistant, though Wi-Fi profiles can be pre-configured in enrollment settings.

### Identity provider integration

Most enterprise deployments integrate LDAP or SAML authentication during enrollment. Platform SSO can now be activated during automated device enrollment, a capability announced at WWDC 2025\. Platform SSO allows employees to access managed apps and company services through their identity provider without additional sign-ins after initial enrollment.

The MDM server requests authentication through the IdP, receives user attributes, and associates them with the enrolling device. This connection allows user-specific policies based on job role or department. Testing authentication workflows before fleet-wide deployment prevents failures where enrollment succeeds but users lack access to corporate resources.

### Prestage enrollment configuration

Prestage enrollments define which Setup Assistant panes users see during first boot. Administrators can skip screens like Location Services and Screen Time, though some panes cannot be skipped even with prestage configuration.

Bootstrap packages allow immediate software provisioning during enrollment, including essential applications, security certificates, and initial configurations. Post-enrollment automation workflows can then deliver additional applications through MDM payloads.

## Supporting Apple management technologies

Beyond the core ADE workflow, several Apple technologies work together to create automated provisioning that handles thousands of devices. Understanding how these pieces connect helps teams design enrollment workflows that take full advantage of Apple's management capabilities.

### Declarative device management

Apple's Declarative Device Management represents a strategic shift from server-managed to device-managed enforcement. Rather than MDM servers constantly polling devices to verify compliance status, DDM servers push declarations to devices, which autonomously enforce policy and proactively report status changes. This reduces constant polling and server burden by distributing enforcement logic to devices. 

Apple announced that macOS 26 will expand DDM capabilities to include package deployment for Mac devices. When released, DDM will support device-driven software distribution for App Store apps, custom apps, and .pkg files, enabling devices enrolled through ADE to use DDM-managed state for end-to-end automated provisioning. This device-driven architecture proves particularly valuable for managing large fleets, as it distributes configuration management responsibility to devices rather than centralizing it on MDM servers.

### Platform SSO integration

Platform SSO allows identity-first provisioning where authentication happens during enrollment rather than after. Platform SSO can now be activated during automated device enrollment, allowing employees to immediately access managed apps and company services without additional sign-ins. This addresses a common friction point where users complete device enrollment but then face repeated authentication prompts when accessing corporate resources.

### Apple Business Manager API capabilities

Apple Business Manager is receiving significant API enhancements that allow programmatic device management. New endpoints let administrators retrieve device management service information through *GET /v1/mdmServers*, list all devices assigned to a specific MDM server via *GET /v1/mdmServers/{id}/relationships/devices*, and programmatically assign or unassign devices from management services through *POST /v1/orgDeviceActivities*. 

These new APIs support infrastructure-as-code patterns where device assignments and MDM configurations are version-controlled and deployed through automated pipelines, aligning with emerging GitOps-based management approaches for enterprise Mac fleet management.

## Unified device management across platforms

For teams managing mixed fleets, Mac zero-touch deployment compares to similar capabilities on Windows and Linux in distinct ways. Each platform has different enrollment architectures and different maturity levels for automated provisioning.

### Platform-specific enrollment protocols

Windows Autopilot provides Microsoft's zero-touch approach through integration with cloud identity services and Windows MDM services. Windows Autopilot is a collection of technologies used to set up and pre-configure new devices, utilizing the OEM-optimized version of Windows client while simplifying the device lifecycle from initial deployment through end of life. 

Autopilot depends on specific features available in Windows client, cloud identity services, MDM services, Windows Activation services, and automatic device joining/registration to cloud identity during provisioning. Unlike Apple's Automated Device Enrollment, Windows Autopilot also supports registering existing or self-built devices that weren't purchased through participating OEM channels, by manually collecting and uploading their hardware hashes.

| Platform | Zero-Touch Approach | Key Requirements | Enrollment Method |
| ----- | ----- | ----- | ----- |
| macOS | Automated Device Enrollment (ADE) | Apple Business Manager, participating vendors | Cloud-based automatic enrollment via Setup Assistant |
| Windows | Windows Autopilot | Cloud identity services, Windows MDM, participating OEMs | Cloud-based device registration and provisioning |
| Linux | Custom automation | Configuration management tools, IT infrastructure | No standardized vendor-provided protocol |

While Linux supports automated provisioning through tools like cloud-init, Ansible, and PXE boot, it lacks a vendor-provided MDM enrollment protocol comparable to ADE or Autopilot. These deployments rely on configuration management infrastructure rather than MDM-based enrollment, and implementations remain distribution-specific since macOS and Windows are the only platforms with vendor-supported zero-touch enrollment using standardized protocols. Building Linux zero-touch provisioning requires substantial ongoing investment in custom automation.

### Post-enrollment management convergence

True cross-platform zero-touch deployment doesn't exist at the enrollment layer. Each platform requires separate enrollment infrastructure (Apple Business Manager for macOS, Windows Autopilot for Windows, and custom automation for Linux). However, modern MDM tools can provide unified management after enrollment completes. Administrators configure policies once and deploy them across platforms, with the MDM translating intent into platform-specific implementations.

This architectural reality means IT teams need to architect separate enrollment strategies per platform while seeking unified visibility and control post-enrollment. A practical approach involves accepting platform-specific enrollment workflows while standardizing on security baselines and compliance monitoring that work across operating systems.

## GitOps-based Mac fleet management

Infrastructure-as-code patterns are gaining adoption for Mac fleet management. Some enterprises have implemented GitOps workflows in production environments, treating device configuration as declarative code stored in version control systems.

This approach provides several advantages for IT teams. Configuration changes go through code review before deployment, creating an audit trail of who changed what and why. Teams can test configuration updates in staging environments before applying them to production fleets. If your infrastructure team is already familiar with GitOps patterns for server and application management, they can apply the same methodologies to device management.

Organizations can define macOS deployment configuration through YAML files that specify bootstrap packages, Setup Assistant customization, enrollment authentication requirements, and security policies. These configuration files live in GitHub repositories alongside other infrastructure code, receiving the same review and deployment workflows as other infrastructure components.

## Open-source Mac fleet management

Implementing zero-touch deployment gives your team consistent device provisioning while reducing manual workload. The right tool makes the difference between automation that works and automation that creates new problems. This is where Fleet comes in.

Fleet integrates with Apple Business Manager for automated device enrollment and provides the cross-platform visibility you need when managing mixed Mac, Windows, and Linux fleets. [Schedule a demo](https://fleetdm.com/contact) to see how Fleet simplifies Mac fleet management.

## Frequently asked questions

### Can we implement zero-touch deployment without Apple Business Manager?

Automated Device Enrollment requires devices registered in Apple Business Manager or Apple School Manager, which is only possible when purchased from Apple or participating Apple Authorized Resellers. Alternative methods like user-initiated enrollment or Apple Configurator require manual steps and don't provide the same capabilities, since MDM enrollment profiles can be removed. Devices acquired through unauthorized channels can't use Automated Device Enrollment.

### What happens if a device fails to enroll during first boot?

For macOS 14 and later, devices registered to Apple Business Manager that don't enroll during first setup display a full-screen experience enforcing enrollment that users cannot bypass. Common failure causes include network connectivity issues, MDM server configuration errors, or device assignment problems. Troubleshooting involves checking MDM synchronization status, verifying device assignment in Apple Business Manager, and ensuring access to Apple's activation servers and MDM endpoints.

### How does zero-touch deployment affect device security and compliance?

Zero-touch deployment strengthens security by ensuring devices receive consistent baseline configurations from first boot. Enrollment profiles can be configured as non-removable, and the automated workflow reduces configuration variability that manual provisioning introduces, making compliance audits simpler through complete enrollment and configuration logs.

### Does zero-touch deployment work for device refresh cycles?

Yes. Replacement devices enroll automatically through the same workflow as new devices. Apple's Return to Service workflow automates device erasure and re-enrollment for device redeployment. Starting with iOS 26, iPadOS 26, and visionOS 26, Return to Service can also preserve managed apps during device preparation for reuse. [Try Fleet](https://fleetdm.com/device-management) to see how declarative configuration automates enrollment workflows for both new devices and refresh scenarios.
