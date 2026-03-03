# What is Apple's Device Enrollment Program (DEP)?

Apple no longer uses the program name Device Enrollment Program (DEP) in its documentation. Apple now uses [Automated Device Enrollment (ADE)](https://support.apple.com/en-us/102300) to describe the way newly purchased devices are automatically enrolled into your MDM of choice during first boot. By enabling supervision during enrollment, ADE lets organizations enforce settings such as preventing users from removing the MDM management profile and scales to thousands of devices.

This article covers how ADE works, which devices qualify, security controls, and MDM integration.

## Automated Device Enrollment overview

ADE links devices purchased through authorized channels to your organization in Apple Business Manager (ABM) before they reach employees, allowing them to ship directly to end users. When users power on their devices for the first time and connect to the internet, automatic MDM enrollment begins. Users complete Setup Assistant screens while device configuration applies from your [MDM server](https://fleetdm.com/device-management) in the background.

Apple originally launched this capability in 2014 as the Device Enrollment Program and rebranded it to Automated Device Enrollment (ADE) in December 2019 alongside the launch of ABM. The underlying technology remained the same, but the new name better describes what the system actually does. 

For organizations managing hundreds or thousands of Apple devices, this automation eliminates the manual configuration burden that previously required IT teams to physically handle every device before distribution. Security policies apply from first boot, and devices can ship directly to remote employees without IT involvement.

## How does Automated Device Enrollment (ADE) work?

ADE relies on a series of automated steps that begin the moment you purchase devices. Each device's serial number registers in Apple's activation database when you buy through authorized Apple channels, automatically assigning it to your organization's ABM account. During first activation, the device queries Apple's servers before any user interaction. Apple's servers return your organization's MDM enrollment URL along with configuration parameters. The device then contacts your MDM server to receive configuration profiles and security policies.

Once enrolled, supervision and MDM configuration determine whether controls are permanent and whether users can remove management profiles. Factory reset doesn't eliminate controls either because enrollment status ties to the device serial number registered in ABM.

Behind the automatic enrollment process, ADE works by establishing a trusted connection between ABM and your MDM server through token-based authentication. You download a server token file from ABM and import it into your MDM server to create this secure connection. The token validates that your MDM server is authorized to manage devices assigned to your organization.

Enrollment requires internet connectivity to function properly. Setup Assistant pauses at the network selection screen until the device can reach Apple's activation servers. Without network access, devices cannot complete the automated enrollment process. This connectivity requirement ensures devices receive current enrollment profiles and security configurations before users gain access.

Failed enrollments typically result from network connectivity issues, expired certificates, or misconfigured enrollment profiles. When enrollment fails, the device displays an error message during Setup Assistant. IT teams can troubleshoot this by verifying network connectivity to Apple's servers, checking certificate expiration dates, and reviewing enrollment profile configurations in ABM.

## Benefits of Apple's Automated Device Enrollment

Zero-touch deployment reduces IT workload by eliminating physical device handling before distribution. IT teams configure enrollment profiles once in ABM, then devices automatically enroll without per-device interaction. This automation scales enrollment from dozens to thousands of devices without requiring additional IT headcount, allowing organizations to ship devices directly to distributed employees anywhere and significantly reducing onboarding timelines.

These benefits show up in practice. In this case study, Apple documented a [28% support request](https://www.apple.com/business/enterprise/success-stories/retail/rituals/) reduction when using ADE to manage thousands of devices remotely with a small IT department.

Beyond operational efficiency, ADE provides security advantages that manual enrollment cannot match. The system prevents users from bypassing or removing device management, closing security gaps that existed with previous approaches. Security policies apply automatically before users access devices, ensuring that encryption, firewall rules, and required applications deploy during initial setup. New hires receive ready-to-work devices on day one with everything pre-configured, eliminating common help desk tickets and configuration delays.

## ADE vs. manual enrollment: Key differences

Organizations have two primary methods for enrolling Apple devices into MDM systems: ADE, and manual enrollment. Understanding the differences between these two helps plan device deployment strategies.

| Factor | ADE | Manual Enrollment |
| ----- | ----- | ----- |
| **Enrollment timing** | Automatic during Setup Assistant | May require user action post-setup |
| **IT involvement** | Zero-touch (devices ship to users) | IT may need to manually configure or instruct users |
| **MDM profile removal** | Delivers an immutable, non-removable MDM enrollment profile for management | Depending on the enrollment workflow, users may be able to  remove management |
| **Device supervision** | Automatic supervision | Requires Apple Configurator |
| **Purchase requirements** | Must buy through authorized seller | Any device source works |
| **Existing device support** | Requires erase or Apple Configurator | Can work without device wipe |

ADE makes sense for new device purchases going directly to employees, organizations prioritizing security and compliance, remote workforces where IT can't physically configure devices, and large-scale deployments processing many devices annually.

Converting manually enrolled devices to ADE requires wiping them completely. The best way to handle this is timing your ADE implementation with device refresh cycles rather than forcing users through device wipes. During the transition period, organizations can run both enrollment methods side by side, using ADE for new purchases while keeping manual enrollment for devices already deployed.

## Which Apple devices work with ADE?

ADE works across [Apple's major device platforms](https://support.apple.com/en-us/102300) with specific operating system requirements that most current devices already meet.

Purchasing requirements determine which devices qualify for automatic enrollment. Devices must come through Apple directly or through ADE-participating authorized resellers to register automatically. When purchasing from resellers or carriers, you should provide your ABM Organization ID during the transaction so devices register to your account immediately. Direct consumer purchases from Apple Stores don't automatically qualify. Existing device inventory can be added through Apple Configurator, Apple's Mac app for manually configuring and supervising devices via a USB connection, though this requires physical device access and device wipes, which makes it impractical for devices already in use.

Geographic availability also matters if you operate across multiple countries. ADE works through ABM in over 35 countries across the Americas, Europe, Asia-Pacific, and the Middle East. Reseller participation varies by region, so you should check with your resellers about ADE availability in your target markets before placing large orders.

## What security controls does ADE provide?

ADE automatically supervises devices during enrollment, unlocking security restrictions that aren't available on unsupervised devices. This supervision gives you advanced management controls that protect your company data while keeping everything transparent. Users can see their device's supervision status in settings, so there's no hidden monitoring happening in the background.

This supervised mode provides security features that work together to prevent unauthorized access and data loss:

* Automatic device supervision for advanced management controls  
* Mandatory MDM enrollment that users cannot bypass or remove  
* Activation Lock bypass codes for organizational device recovery  
* Factory reset protection that maintains management through re-enrollment  
* Device identity certificates for secure MDM authentication

Activation Lock and certificate management need additional planning during deployment. Activation Lock ties devices to user Apple IDs to prevent theft but creates complications when employees leave without disabling Find My. Through ABM, ADE provides organizational bypass codes that let MDM administrators clear device activation without needing the original user's Apple ID credentials.

Certificate management requires ongoing attention because the Apple Push Notification certificate expires annually. Organizations must use the same Apple ID for renewal that was used during initial certificate creation. When certificates expire, devices and management servers lose the ability to authenticate with each other until someone completes the renewal process.

## Choosing an MDM platform for ADE deployment

ADE is an enrollment mechanism configured through ABM, not a complete management platform. ABM assigns devices to your MDM server, but the MDM server delivers the actual profiles, apps, and policies after enrollment completes. Without an MDM system, ADE provides no device management capabilities.

When evaluating MDM vendors for ADE compatibility, you need to verify several technical requirements. Check that the platform supports Apple Push Notification certificate management with annual renewal processes, offers Setup Assistant customization options that let you control the enrollment experience, and can handle multiple MDM servers if your organization needs different management systems for different regions or business units.

Another important consideration for the long term is vendor flexibility. Changing MDM vendors after deploying ADE requires wiping enrolled devices completely and re-enrolling them with the new platform. This disruption is significant enough that you should plan any MDM migrations to coincide with natural device refresh cycles rather than forcing users through unnecessary resets.

Cross-platform capabilities also matter if you manage more than just Apple devices. Organizations with mixed device environments benefit from MDM platforms that handle Mac, Windows, and [Linux](https://fleetdm.com/guides/how-to-install-osquery-and-enroll-linux-devices-into-fleet) from a single console rather than juggling separate management tools. [Fleet](http://fleetdm.com) supports ADE enrollment for Mac, iPhone, and iPad devices while also managing Windows and Linux endpoints. Its open-source model provides complete code transparency so you can verify exactly how devices are managed, and self-hosting options let you maintain full control over where device data lives.

## What do you need to set up ADE?

Setting up ADE requires some upfront preparation to ensure smooth deployment. You need to establish foundational infrastructure and verify you meet Apple's requirements before enrolling your devices.

You should start by confirming you have these essential prerequisites in place:

* Apple Business Manager account with D-U-N-S number and domain verification  
* [MDM vendor](http://fleetdm.com) supporting ADE enrollment and APNs certificate management  
* Authorized reseller relationships for automatic device registration  
* Network infrastructure permitting connections to Apple servers without SSL/TLS inspection  
* Certificate renewal procedures using the same Apple ID for annual APNs renewal

Beyond technical infrastructure, configuration planning determines how users experience enrollment. You need to define enrollment profiles that specify which Setup Assistant screens users see during initial setup, establish device naming conventions that make sense for your IT team, and create department-specific configurations for different user groups. Make sure to assign these profiles in Apple Business Manager before distributing devices so enrollment happens smoothly without last-minute troubleshooting.

## Conclusion

ADE eliminates manual enrollment work, enforces security policies through supervised mode and mandatory MDM enrollment, and scales device management to thousands of devices without proportional IT headcount increases. Organizations achieve faster deployment, better security, and reduced operational overhead through zero-touch enrollment.

[Fleet's open-source MDM platform](https://fleetdm.com/device-management) supports ADE enrollment with cross-platform visibility across Mac, Windows, and Linux devices. [Schedule a demo](https://fleetdm.com/contact) to see how Fleet's ADE-compatible MDM delivers zero-touch deployment while maintaining complete data transparency.

<meta name="articleTitle" value="What is Apple's Device Enrollment Program (DEP)?">
<meta name="authorFullName" value="Brock Walters">
<meta name="authorGitHubUsername" value="nonpunctual">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-02-26">
<meta name="description" value="Apple's Automated Device Enrollment (ADE), previously DEP, automatically enrolls devices into MDM. Learn how it works, its benefits, and requirements.">
