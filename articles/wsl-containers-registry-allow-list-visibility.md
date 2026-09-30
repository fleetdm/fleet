# A registry allow list for WSL containers only helps if you can see what Windows devices actually pull

*WSL containers reached general availability on September 29, with Intune controls and Defender for Endpoint visibility. Both help. Neither tells you which of your devices are running containers today.*

## Key takeaways

- **Linux containers are now a supported part of the Windows attack surface.** With WSL containers generally available, developers can build and run Linux containers on Windows without a separate container engine, so expect them to appear on more devices, sooner.
- **Microsoft shipped two admin controls at launch.** Intune can enable or disable the feature, and a "WSL containers registry allow list" restricts image pulls to approved registries.
- **A control is a statement of intent, not a measurement.** A device that was offline, unenrolled, or configured by hand may not match the policy you think you deployed.
- **Defender for Endpoint adds activity visibility, not an inventory.** It surfaces process, file, and network activity from containers and ties it to the Windows host, which helps an investigation once you know where to look.
- **Fleet's agent can report device state on Windows.** Installed software, running processes, and configuration are queryable across your devices, so you can compare what is actually present against the policy you set.
- **Answer the inventory question before the incident.** Knowing which devices run containers, and which registries they pull from, is much cheaper to work out now than during a response.

<a purpose="cta-button" href="https://fleetdm.com/device-management">See how Fleet manages Windows devices</a>

Microsoft's WSL containers feature went generally available on September 29. It adds a `wslc.exe` command line tool, aliased as `container.exe`, and an API for running Linux containers from native Windows applications. For IT teams, the announcement that matters most is the enterprise controls that arrived with it.

Those controls are welcome, and they leave a question open. A policy that restricts where containers can pull from tells you what should happen. It doesn't tell you which devices are running containers right now.

## What Microsoft shipped for admins

Microsoft describes two Intune settings. The first enables or disables WSL containers entirely. The second is a "WSL containers registry allow list," which limits the registries developers can pull images from. An internal registry can be allowed while a public one stays blocked.

On the security side, Microsoft says the Defender for Endpoint plugin for WSL now covers containers. It surfaces process, file, and network activity from a container and connects that activity to the Windows host. Microsoft also notes that Docker Compose isn't supported yet, and calls it the top feature request.

## What a policy can't tell you

Intune deploys a setting, and Defender watches activity. Neither is designed to answer the inventory question: which devices have containers running, and which registries did they pull from?

A few things make that gap real. Devices that haven't checked in yet don't have the setting. A developer with local admin rights can change how their machine is configured. And a policy applied to a group only covers the devices that are in the group. Each of these is ordinary, and each leaves you with a policy that says one thing and a device that does another.

## Checking the device itself

Fleet's agent reports state from each Windows device, so you can compare it against your intent instead of assuming the two match. Installed software, running processes, and configuration are all queryable across the fleet, and a report can run on a schedule.

What Fleet can see about containers specifically depends on the query you write, so test it on a device that runs WSL containers before you rely on it. A sensible starting point is to check whether the feature's components are present, whether container processes are running, and whether the setting Intune deployed has landed on the device. Confirm where that setting is stored on a test machine before you write the check.

Once a report gives you the answer, a policy can keep asking. If a device drifts from the configuration you expect, it shows up as failing rather than staying invisible.

## Where this leaves you

WSL containers are going to spread through engineering teams because they remove friction for developers. The Intune controls give you a way to shape that. Defender gives you a place to investigate.

The piece in between is knowing what is actually running. Answer that while nothing is wrong, and the allow list becomes a control you can verify instead of one you hope is working.

## See it live

- **Get a demo** to see Windows device state reported across your own fleet: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Read how Fleet handles device management:** [fleetdm.com/device-management](https://fleetdm.com/device-management)

<meta name="articleTitle" value="A registry allow list for WSL containers only helps if you can see what Windows devices actually pull">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-09-30">
<meta name="description" value="WSL containers are generally available, with Intune and Defender controls. Here's how to check which Windows devices are actually running them.">
