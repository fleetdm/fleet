# Deploy Santa with Fleet GitOps and skip the sync server

### Links to article series:

- Part 1: Deploy Santa with Fleet GitOps and skip the sync server
- Part 2: [How we deployed Santa at Fleet](https://fleetdm.com/guides/how-we-deployed-santa-at-fleet)

[Santa](https://github.com/northpolesec/santa) is a binary authorization system for macOS. It has become important to organizations serious about application blocking and control. However, the traditional Santa deployment model comes with operational overhead at scale, primarily centered around the need for a dedicated Santa sync server.

In the conventional setup, Santa requires a custom sync server to:

- Distribute allow / deny rules across your fleet
- Collect execution events and blocked binary reports
- Manage configuration changes and rule updates

At the time of writing, there are currently three off-the-shelf sync server solutions available:

- [Moroz](https://github.com/groob/moroz) - A golang server that serves hardcoded rules from simple configuration files.
- [Rudolph](https://github.com/airbnb/rudolph) - An AWS-based serverless sync service built on API GW, DynamoDB, and Lambda components.
- [Zentral](https://github.com/zentralopensource/zentral) - An event hub to gather, process, and monitor system events and link them to an inventory.

Running any of these solutions may incur additional infrastructure costs and upkeep. You also might have to adopt an unfamiliar configuration language specific to the solution. 

But, what if you could get all the benefits and functionality of a sync server using your existing device management solution?

## Enter Fleet + GitOps + Santa

The combination of Fleet's device management platform, GitOps principles, and Santa's binary authorization creates a powerful alternative that eliminates the need for a traditional Santa sync server entirely.

## How Fleet replaces the Santa sync server

Fleet acts as a modern, API-driven replacement for traditional Santa sync servers by using:

### Configuration as code management

Fleet's GitOps workflow allows you to manage Santa configurations stored in Git repositories. Instead of hosting sync server infrastructure, you define Santa rules and configurations declaratively through familiar XML (mobileconfig) files.

### Automated rule distribution

Fleet's agent (fleetd) and MDM automatically applies Santa configurations across your macOS devices. Changes pushed to your Git repository trigger automatic deployment through Fleet's GitOps pipeline.

### Event collection and monitoring

Fleet's osquery integration captures Santa events, eliminating the need for custom event collection endpoints.

## Implementation overview

Here is how the Fleet + GitOps + Santa workflow operates in practice:

1. **Configuration Definition:** Security and IT teams define Santa rules in files within a Git repository
2. **Change Management:** Rule updates go through standard pull request review processes
3. **Automated Deployment:** Fleet GitOps detects changes and applies configurations
4. **Real-time Monitoring:** osquery tables provide visibility into Santa events
5. **Incident Response:** Fleet's queries and policies trigger automated workflows for investigation or remediation

## The bottom line

Fleet believes in reducing complexity. Fleet's GitOps-native approach provides the functionality of a custom Santa sync server while adding enterprise device management, operational simplicity, and modern change management capabilities while eliminating infrastructure maintenance. It's a more scalable and secure approach to binary authorization that aligns with modern infrastructure practices.

Ready to modernize your Santa deployment? Fleet's open-source platform makes it easier than ever. 

Stay tuned into the progress and discussion on a native Santa + Fleet integration currently in design by viewing this [Fleet feature request on GitHub](https://github.com/fleetdm/fleet/issues/24910)

[Fleet](https://fleetdm.com/device-management) is an open-source device management platform that provides GitOps-native configuration management, comprehensive device visibility, and enterprise-grade security for organizations managing thousands of endpoints.

The [next article](https://fleetdm.com/guides/how-we-deployed-santa-at-fleet) in this series is a step-by-step guide showing how we implemented this deployment model for Santa internally at Fleet.

<meta name="articleTitle" value="Deploy Santa with Fleet GitOps and skip the sync server">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-02-04">
<meta name="description" value="Part 1 of 2 - Learn to manage Santa in a whole new way with less complexity and overhead.">
