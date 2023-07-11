# Embracing the future: Declarative Device Management 

![Embracing the future: Declarative Device Management](../website/assets/images/articles/embracing-the-future-declarative-device-management@2x.png)

As a Mac administrator, managing a fleet of Apple devices across your organization requires consistency and airtight security. With a variety of system services and background tasks to oversee, the challenge is not only to maintain uniform configurations but also to keep the organization's data secure. Recognizing these challenges, Apple has advanced a powerful new approach - Declarative Device Management (DDM).

DDM is a paradigm shift in device management, enabling a more efficient and secure administration of macOS devices. It allows for tamper-resistant configurations and ensures simplified monitoring of system services and background tasks.

In this blog post, we dive into Apple's forthcoming DDM in macOS Sonoma. Specifically, we'll explore how it will alter the way you manage system services, certificates and identities, and how it transitions you from traditional Mobile Device Management (MDM) systems. Whether you're an experienced Mac admin or just getting started, hopefully, this guide will provide some insights into DDM for you and your organization. Let's dive in!


## Declarative device management for system services

DDM paves the way for a secure and reliable mechanism to manage system services. Using tamper-resistant system configuration files for different system services ensures uniform and secure configurations across all devices. Declarative Device Management provides an added layer of protection against accidental changes by users.

For instance, system services like sshd, sudo, PAM, CUPS, Apache httpd, bash and Z-shells will be able to adopt managed service configuration files to ensure consistency and compliance. The configuration files reference a data asset that provides a ZIP archive of SSH keys that is downloaded and expanded into a tamper-resistant, service-specific location when required conditions are met—for example, FileVault is enabled—and are always prioritized over any default or overridden system configuration.


## Monitoring and compliance rules for background tasks

DDM provides an excellent way of keeping track of background tasks. A new status item in this coming release reports the list of installed background tasks, making it easier to verify that required tasks are running and unwanted tasks aren't.

In addition, the FileVault enabled state of the macOS boot volume is reported, allowing you to install sensitive configurations only when it is safe to proceed. With these features, you can ensure compliance and consistency across all macOS devices in your organization.


## Secure access with certificates and identities

Certificates and identities play a crucial role in ensuring secure access to organizational resources. In this context, DDM provides a more efficient mechanism for managing certificates and identities using its declaration data model.

Certificates and identities are defined as asset declarations, which various configurations can reference. This eliminates the need for duplicating certificates and identities across multiple profiles, thereby reducing management overhead.


## A new paradigm: software updates

Apple's DDM introduces a redefined software update process, which marks another significant step forward in device management.

Traditionally, administrators have faced considerable challenges in managing software updates. However, with DDM, this process has been dramatically simplified. The Declarative model handles scheduling and applying updates, allowing administrators to specify the desired state – for instance, maintaining the latest software version – and leave the rest to DDM.

To improve upon this functionality, Fleet, with its osquery integration, allows admins to monitor the status of these updates in real time. It provides critical insights about the update process, such as software versions, pending updates, and the update history. These features make the software update process significantly more manageable and transparent.

DDM represents an important advancement in how we manage and understand software updates. It not only will streamline administrative tasks but also elevates the overall security, performance, and integrity of the devices Mac admins manage.


## Seamless transition from MDM to DDM

Transitioning from traditional MDM to DDM will be a challenge. However, DDM provides a smooth transition without causing disruption or leaving a management gap. This is achieved by allowing DDM to take over the management of already installed MDM profiles without the need to remove them.


## Fleet + osquery + DDM = 💗

The innovations introduced with DDM, including the new software update process, represent a paradigm shift in device management. Fleet's MDM solution, powered by osquery, complements these changes and offers a GitOps-driven management platform for Mac admins.

As we continue to navigate this evolving landscape, we have tools that equip us better than ever to handle the challenges and complexities of modern device management. This new era presents opportunities for enhanced security, control, and efficiency in managing our devices.

Fleet is transforming how we manage and secure devices. Offering an open-core, cross-platform solution, Fleet is committed to empowering Mac admins with the tools they need to meet the challenges of today's and tomorrow's device management. Through its powerful and versatile platform, Fleet is illuminating the path forward in device management.



<meta name="category" value="announcements">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="authorFullName" value="JD Strong">
<meta name="publishedOn" value="2023-07-06">
<meta name="articleTitle" value="Embracing the future: Declarative Device Management">
<meta name="articleImageUrl" value="../website/assets/images/articles/embracing-the-future-declarative-device-management@2x.png">
<meta name="description" value="Explore the transformative impact of Declarative Device Management (DDM), Fleet, and osquery for MacAdmins.">
