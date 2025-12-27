# Declarative device management: a primer

Declarative device management (DDM) changes how managed settings are deployed and enforced on Apple devices. Device state can be maintained via DDM by enabling devices to independently apply configurations based on describing criteria rather than constantly polling management servers for pushed configuration updates. 

This guide goes deeper into what declarative device management is, how it differs from legacy management approaches, and how to implement it in your organization.

## Understanding declarative device management (DDM)

Apple introduced declarative device management at the 2021 Worldwide Developer Conference. The DDM protocol has been added to the existing Apple device management protocol for easy adoption in practice, and the DDM protocol includes the entire current MDM protocol's range of preference domains for backwards compatibility. 

This means DDM and MDM profiles can be delivered to devices simultaneously without management conflicts, and organizations can adopt DDM gradually, running declarations alongside their current MDM configuration profiles while they transition to DDM over time.

### How DDM changes device management

Traditional MDM follows a reactive pattern where servers drive every action through constant back-and-forth communication. When an administrator wants to deploy an application to managed devices, the server sends an installation command and waits for acknowledgement from each device. The server then checks back to see if installation has started, receives progress updates, and continues this conversation until it can verify that every device finished the task.

This same pattern repeats for every management action, whether checking compliance status, updating configurations, or gathering device information. Each interaction requires network communication, server processing, and device attention at multiple stages.

### What is DDM and how does it change Apple device management?

DDM changes the relationship between management servers and the devices they oversee. Administrators define a desired state for their devices. The devices then take responsibility for reaching and maintaining that state on their own.

This might sound like a small distinction, but it changes how management actually works in practice. A device running DDM receives a “declaration” that describes what should be true about its configuration. From that point forward, the device checks its own state against those declarations and makes adjustments when needed.

### Why Apple built DDM: Solving scalability challenges

The MDM approach works well enough when managing dozens or even hundreds of devices, but, problems can show up when organizations scale to many thousands of devices, need faster rollout times, or less congestion during critical configuration delivery (e.g., configuration profiles that contain certificates). 

A server managing many thousands of devices might handle millions of individual requests per day using legacy MDM. This creates significant load on both the management server and the network. 

Apple built DDM to solve these problems by shifting responsibility to the devices themselves. DDM reduces the need for servers to constantly poll for updates since devices themselves are responsible for maintaining their compliance state.

### Supported Apple platforms and requirements

DDM support has expanded across Apple's device ecosystem since its 2021 introduction:

* iOS 16 and later (all enrollment types)  
* iPadOS 15 and later (all enrollment types)  
* macOS 13 Ventura and later  
* tvOS 16 and later  
* watchOS 10 and later

These requirements mean DDM needs relatively recent operating systems. Organizations running current OS versions can start implementing DDM immediately.

## The three pillars of declarative device management

Apple describes DDM as built on three core concepts that work together:

### Declarations: Defining device policy

Declarations are the building blocks of DDM. There are four distinct types of declarations:

* **Configurations:** These work similarly to existing configuration profiles, defining settings, restrictions, and account details.  
* **Assets:** These represent reference data that configurations need, like credentials or certificates, letting you define information once and reference it from multiple places rather than duplicating data across configurations.  
* **Activations:** These determine when and where configurations should apply by grouping them together with conditions that must be met before they take effect.  
* **Management declarations:** These convey organizational information and server capabilities to devices.

This flexibility lets organizations build management policies that adapt to different device contexts without duplicating configuration data. A single configuration can be referenced by multiple activations, and a single activation can include multiple configurations.

### Status channel: Proactive device reporting

The status channel changes how devices communicate state information back to management servers. Rather than waiting for servers to poll them for updates, devices report changes as they happen.

Management servers subscribe to specific status items they want to track, such as OS version, device type, or compliance state. Once subscribed, the server receives an initial status report. After that, devices only send updates when subscribed items actually change. A device doesn't need to report that its OS version is still iOS 17.2 every time the server checks in. It only reports when the attribute changes.

Status items also work in activation predicates, which lets devices decide what configurations to apply based on their current state. When a device updates from iOS 16 to iOS 17, it can immediately check whether any configurations requiring iOS 17 should now activate.

### Extensibility: Future-proof management

Apple continuously updates its operating systems and adds new management declaration types. DDM handles these changes through capability awareness between devices and management servers. When a device updates to a newer OS, it tells the server what new features it supports, and servers inform devices about newly available declaration types. This means organizations can [adopt new capabilities](https://fleetdm.com/announcements/mdm-just-got-better) as they become available without coordinating major upgrades.

## The benefits of declarative device management

Organizations should see improvements by using DDM in two key areas:

### Performance and scalability gains

Management server loads decrease because devices no longer need constant polling. A server managing 10,000 devices might have handled millions of check-in requests per day under traditional MDM. With DDM, devices only communicate when their state actually changes. Network traffic also decreases, and setting enforcement happens more efficiently because devices don't wait for polling cycles.

### Improved IT operations

IT teams should see several improvements due to DDM's design:

* Administrative overhead decreases as devices handle routine compliance enforcement more efficiently
* The JSON-based declaration format enables configurations that are easier to build and maintain

## Implementing DDM in your organization

Understanding these benefits helps organizations plan their DDM adoption approach.

Moving to DDM will currently work best as a gradual migration, transitioning configuration profiles slowly and as-needed rather than reformatting everything at once. 

### Coexistence with legacy MDM

DDM was designed to work alongside MDM rather than replace it. Configuration profiles and DDM declarations can exist on the same device simultaneously. MDM configuration profiles that an organization currently is delivering to devices can be reformatted as DDM or kept as-is.

However, when both an MDM configuration profile and a DDM declaration contain the same setting, DDM takes precedence. This precedence applies specifically to software update configurations and app management. Organizations can migrate policies to DDM selectively, converting software update policies to declarations while leaving other configurations as traditional profiles. The coexistence period can last as long as necessary since Apple intends to continue supporting legacy MDM profiles as it expands DDM capabilities.

### Planning your migration

Organizations should identify which policies will benefit most from migration first. Software updates make a good starting point since Apple has built strong DDM support for updates and the benefits are real. 

Whatever starting point you choose, plan your testing approach early in the process. Use pilot groups or a staging environment that represents how your fleet will behave without affecting all end users and use activation predicates to target specific device types or OS versions if your management solution supports them.

**Getting started**

Once you've planned your approach, the actual implementation follows a straightforward progression:

1. Verify that your devices meet OS requirements by checking your device inventory  
2. Choose a simple, high-impact use case like software update enforcement for your first implementation  
3. Deploy to a small pilot group and monitor status channel reports  
4. Expand gradually to larger device groups based on early results

These initial deployments give you hands-on experience with DDM before rolling it out more broadly.

### Monitoring and measuring success

After you've deployed DDM, you should measure its impact by comparing the new approach against the old one. If supported, you'll see status channel reports quantify the reduction in device communication. Server metrics will show infrastructure benefits through lower CPU utilization, reduced memory consumption, and decreased network bandwidth. Faster control enforcement also may be noticeable in your reporting.

## How Fleet supports declarative device management

[Fleet](http://fleetdm.com) is an open-source device management platform that enables administrators to send DDM payloads directly to macOS, iOS, and iPadOS devices. DDM profiles can live in git repository management solutions like GitHub, GitLab, or Bitbucket where teams can peer review changes, track history, or roll back updates through automated GitOps workflows. The osquery capabilities of Fleet add near real-time querying capabilities giving security teams a "double check" on device management state on top of MDM / DDM: reporting of richly-detailed data, on-demand device information for investigations and fast compliance checks.

For organizations looking to implement DDM with full transparency and no vendor lock-in, Fleet provides an open alternative. [Schedule a demo](https://fleetdm.com/contact) to see how it works.

## Frequently asked questions

### What happens if DDM and legacy MDM profiles conflict?

Apple handles this through precedence rules. When both a configuration profile and a DDM declaration contain the same setting, DDM takes precedence for software update configurations and app management specifically. For other settings, Apple merges conflicting policies and enforces the strictest configuration, similar to how traditional MDM handles multiple profiles with overlapping settings.

### Why aren't my DDM declarations applying to devices?

Most issues trace back to three common causes. First, verify your devices run OS versions that support the declaration types you're using. Second, check that devices have successfully enabled DDM through the activation command during enrollment. And third, review activation predicates carefully since declarations only apply when all associated activations evaluate to true.

### How long does migration from traditional MDM to DDM typically take?

Migration timelines vary based on organization size and complexity. Most organizations take a gradual approach, starting with software updates over one to two months, then moving to account configurations and security policies. The coexistence of DDM and legacy MDM means there's no forced timeline. You can migrate quickly or slowly as it makes sense for your organization, your technical resources, and your risk tolerance.

### What's the easiest way to start implementing DDM?

Start with software update enforcement on a small pilot group since it delivers immediate benefits and has strong Apple support. [Fleet](https://fleetdm.com/device-management) simplifies this by letting you send DDM JSON payloads directly to devices, manage declarations in version control, and monitor responses through real-time osquery integration. [Schedule a demo](https://fleetdm.com/contact) to see how Fleet implements DDM with full transparency.

<meta name="articleTitle" value="Declarative device management: a primer">
<meta name="authorFullName" value="Brock Walters">
<meta name="authorGitHubUsername" value="nonpunctual">
<meta name="category" value="articles">
<meta name="publishedOn" value="2025-11-27">
<meta name="description" value="Learn how Apple's declarative device management works, its benefits over traditional MDM, and how to implement DDM in your organization.">
