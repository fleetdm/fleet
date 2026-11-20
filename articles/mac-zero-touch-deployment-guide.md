# Mac zero-touch deployment: how to automate device provisioning with ADE

*Macs have shipped straight to employees, ready to work, for years, and Apple doesn't charge for it. Here's how Mac zero touch works, and how Fleet now brings the same zero-license-cost setup to Windows and Linux.*

## Key takeaways

- **New Macs set themselves up.** With Automated Device Enrollment (ADE), a Mac ships from Apple or a reseller to the employee, finds its MDM on first boot, and applies your settings without IT ever touching it.
- **Mac zero touch costs nothing extra.** Apple Business is free. Once your organization is verified and your MDM is connected, every eligible Mac you buy can enroll automatically.
- **Setup goes faster when the prerequisites are in place.** Organization verification, the MDM server connection, network access during Setup Assistant, and identity provider sign-in cause most failed rollouts.
- **Apple keeps adding to ADE.** Platform SSO during enrollment, declarative device management, and new Apple Business APIs mean less work after the first boot.
- **Windows and Linux get zero touch too, without Autopilot.** Dell, Lenovo, HP, and other PC makers can install Fleet's agent before the computer ships, so you don't need Autopilot, Entra ID, or Intune. For qualifying customers, Fleet pays the PC maker's fee.
- **One tool and one Git repository for every computer.** Fleet manages Macs, Windows PCs, and Linux machines from the same YAML files, reviewed and deployed like the rest of your infrastructure.

<a purpose="cta-button" href="https://fleetdm.com/contact">Get a demo</a>

When you ship a computer to a remote employee, you usually have two choices: spend IT time setting it up by hand, or send it out without the security settings it needs. Setting up computers by hand doesn't scale, and a misconfigured computer can stay that way for months.

Mac admins solved this years ago with zero touch. This guide covers how it works, what you need before your first rollout, and how the same approach now works for the Windows and Linux computers in your fleet.

## What is zero-touch deployment for Mac?

Zero-touch deployment lets organizations ship Macs directly from Apple or a reseller to end users. When the end user opens the box and turns on their new Mac, it contacts Apple's servers, learns which MDM it belongs to, enrolls, and applies your organization's settings and configuration profiles as the user walks through Setup Assistant.

Apple's Automated Device Enrollment (formerly DEP) makes this possible by linking each Mac's serial number to your MDM server in Apple Business. On macOS 14 and later, a Mac that skips enrollment during first setup shows a full-screen prompt that requires it, so end users can't bypass your organization's management.

This replaces the old imaging routine, where IT received shipments, unboxed each computer, connected it to an imaging station, installed a base configuration, and shipped it again. Now computers go straight to employees and are ready to use when they arrive.

## Why organizations use zero-touch deployment

Remote work changed how computers get set up. IT teams can't count on having the computer in hand, and security requirements keep growing. Zero touch helps in a few ways:

* **Faster setup:** Computers ship directly to end users instead of through an IT office, so people can start work the day the box arrives.
* **Consistent security settings:** Every computer gets the same configuration profiles from first boot, without the variation that comes with setting them up by hand.
* **Less IT work:** A 500-computer rollout that once took weeks of hands-on work happens on its own.
* **A better experience for remote employees:** End users don't need technical knowledge to get started, and you can skip most Setup Assistant screens.
* **Clearer compliance records:** Your MDM records when each computer enrolled and which settings it received, so you don't have to reconstruct a manual process for an audit.

These benefits grow with distributed teams. Offices in different countries can use the same automated setup, so every computer starts with the same security settings wherever it ships.

## How zero-touch deployment works

Mac zero touch has three parts: Apple Business decides where each Mac enrolls, your MDM server sends its settings, and the Mac sets itself up.

### Device registration and assignment

Macs bought from Apple or a participating Apple Authorized Reseller are added to your organization's Apple Business account automatically when you buy them. You can also add Macs bought elsewhere with Apple Configurator, though that takes a manual step for each one.

Once a Mac appears in Apple Business, you assign it to your MDM server in the Apple Business web portal. You can also set a default MDM server so new purchases are assigned automatically.

## Prerequisites and setup considerations

Missing prerequisites cause most enrollment failures, so check these before your first rollout.

### Required infrastructure

You need a verified Apple Business account. Apple asks for your organization's D-U-N-S number (a unique business identifier from Dun & Bradstreet) and confirms your details before Macs can enroll through ADE. Verification is separate from creating the account and can take a few days, and Macs won't appear in Apple Business until it's done.

You also need to connect your MDM server in Apple Business so it can receive Mac assignments. Your MDM needs to support Automated Device Enrollment, enrollment profiles that end users can't remove, and Setup Assistant customization.

During first boot, the Mac needs internet access to reach Apple's activation servers and your MDM. For setup with no end user interaction at all, use Ethernet with Auto Advance. With Wi-Fi, the end user usually picks the network during Setup Assistant.

### Identity provider integration

Most organizations ask end users to sign in with their identity provider (IdP) during enrollment. Your MDM gets the user's details from the IdP and links them to the Mac, so you can apply settings based on role or department. Test sign-in before a fleet-wide rollout, so enrollment doesn't succeed while employees still can't reach company apps.

## Apple management technologies that build on ADE

Several other Apple technologies build on ADE and reduce the work after first boot.

### Declarative device management

With declarative device management (DDM), the MDM sends the Mac a description of how it should be set up, and the Mac enforces it and reports back when something changes. That means less back-and-forth with the server and faster, more reliable results. macOS 26 extended DDM to software, so App Store apps, custom apps, and .pkg files can be installed and kept in place the same way.

### Platform SSO

Starting with macOS 26, Platform SSO can be set up during Automated Device Enrollment. End users sign in with their IdP once, during setup, and can then open managed apps and company services without signing in again. This removes a common annoyance where enrollment finishes but every app still asks for a password.

### Apple Business API

The Apple Business API lets you check which MDM server each Mac is assigned to, list the Macs assigned to a server, and assign or unassign Macs without using the web portal. That makes it possible to manage Mac assignments as code, alongside the rest of your configuration.

## Zero touch for Windows and Linux

Mac zero touch has never needed an extra license. Until recently, Windows and Linux didn't have anything comparable.

### Windows without Autopilot

The usual Windows zero-touch path is Windows Autopilot. Autopilot requires Microsoft Entra ID, so even companies that use Okta or Google for everything else keep paying Microsoft for identity just to set up laptops. For a 1,000-person company, Entra ID P1 alone costs about $84,000 a year.

Fleet takes another route. When you order PCs, the PC maker installs Fleet's agent (fleetd) before the computer ships, using Dell Ready Image, Lenovo Ready to Provision, or the equivalent service from HP, Acer, and others. The PC enrolls in Fleet the first time it's online. You don't need Autopilot, Entra ID, Intune, or USB sticks. End users sign in with your IdP, whether that's Okta, Google Workspace, Entra ID, or another IdP Fleet supports.

That's **zero touch, $0**. For new Fleet Premium customers replacing Intune on at least 500 PCs, Fleet pays the PC maker's one-time setup fee on new PCs for the term of the subscription. To order, you forward an email from Fleet to Dell, Lenovo, or your PC maker. See the [full terms](https://fleetdm.com/articles/leaving-e3-entirely) for details, since this is a limited-time offer.

You can [preinstall Fleet's agent with a provisioning package](https://fleetdm.com/guides/preinstall-fleets-agent-on-windows-with-a-provisioning-package) today. A [Fleet factory image](https://fleetdm.com/guides/windows-zero-touch-without-entra-or-autopilot), coming in Fleet 5.2, goes further: end users sign in with your IdP and get their software before they reach the Windows desktop, much like a Mac in Setup Assistant. If you already use Entra ID and Autopilot, Fleet supports those too.

### Linux

Linux has no vendor-provided enrollment service like ADE or Autopilot. Teams have relied on tools like cloud-init, Ansible, and PXE boot, which work but take ongoing effort to build and maintain for each distribution. With Fleet, the same factory approach works for Linux: the PC maker installs fleetd before the computer ships, and zero touch, $0 covers Linux PCs too. Learn more about [automated provisioning for Linux desktops](https://fleetdm.com/articles/automated-provisioning-for-Linux-desktop-in-the-enterprise).

| Platform | Zero-touch approach | What you need | Extra license cost |
| ----- | ----- | ----- | ----- |
| macOS | Automated Device Enrollment (ADE) | Apple Business, Macs bought from Apple or a participating reseller | None |
| Windows | Fleet agent installed by the PC maker | A participating PC maker (Dell, Lenovo, HP, and others) | None, and Fleet pays the PC maker's fee for qualifying customers |
| Windows (alternative) | Windows Autopilot | Microsoft Entra ID, a Windows MDM | Entra ID P1 or a bundle that includes it |
| Linux | Fleet agent installed by the PC maker | A participating PC maker | None, and Fleet pays the PC maker's fee for qualifying customers |

### One set of settings after enrollment

Each platform still starts differently, but after enrollment Fleet manages them the same way. You can set security baselines, check compliance, and install software for Macs, Windows PCs, and Linux machines in one place, instead of running a separate tool and workflow for each.

## GitOps-based Mac fleet management

More IT teams now manage computers the way infrastructure teams manage servers: as code. With GitOps, your configuration lives as files in a Git repository on GitHub or GitLab.

Every change goes through a code review, so you have a record of who changed what and why. You can test changes on a few computers before rolling them out, and roll back with a revert. If your infrastructure team already works this way, they can use the same habits for computers.

With Fleet GitOps, YAML files define bootstrap packages, Setup Assistant customization, IdP sign-in, policies, software, configuration profiles, and scripts for every platform. Those files live alongside your other infrastructure code and go through the same review and deployment process.

## Zero touch for every computer

Zero touch used to be a Mac advantage. Now new Windows and Linux computers can arrive ready to work too, without Autopilot, Entra ID, or an extra license, and Fleet sets up all of them from one place.

## See it live

- [Get a demo](https://fleetdm.com/contact) to see Fleet set up a new Mac, Windows PC, or Linux computer.
- Read [Leaving E3 entirely](https://fleetdm.com/articles/leaving-e3-entirely) to see what zero touch, $0 means for a 1,000-person company's software bill.

## Frequently asked questions

### Can we implement zero-touch deployment without Apple Business?

No. Automated Device Enrollment requires Macs registered in Apple Business or Apple School Manager. Macs bought from Apple or a participating reseller are added automatically, and you can add others with Apple Configurator. Other options, like having end users enroll themselves, take manual steps and don't give you the same control.

### Does zero touch for Windows and Linux need a Microsoft license?

No. The PC maker installs Fleet's agent before the computer ships, so you don't need Autopilot, Entra ID, or Intune. For qualifying customers, Fleet pays the PC maker's setup fee.

### What happens if a Mac fails to enroll during first boot?

Network problems, MDM server settings, and missing assignments in Apple Business cause most failures. Check that the Mac is assigned to the right MDM server in Apple Business, that your MDM has synced recently, and that the Mac can reach Apple's activation servers and your MDM.

### How does zero-touch deployment affect device security and compliance?

Every computer gets the same security settings from first boot, and enrollment profiles can be set so end users can't remove them. Your MDM's enrollment and configuration records make compliance audits simpler.

### Does zero-touch deployment work for device refresh cycles?

Yes. Replacement computers enroll through the same process as new ones. [Try Fleet](https://fleetdm.com/device-management) to see how it handles new computers and refreshes.

<meta name="articleTitle" value="Mac zero-touch deployment: how to automate device provisioning with ADE">
<meta name="authorFullName" value="Brock Walters">
<meta name="authorGitHubUsername" value="nonpunctual">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-02-26">
<meta name="description" value="How Mac zero-touch deployment works with Apple Business and ADE, and how Fleet brings zero touch, $0 to Windows and Linux without Autopilot.">
