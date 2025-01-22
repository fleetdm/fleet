# Teams

> Teams are available in Fleet Premium.

In Fleet, you can organize hosts into 'teams' to apply queries, policies, scripts, and other configurations tailored to their specific risk and compliance requirements.

To manage teams:

1. Select your avatar in the top navigation.
2. Choose **Settings > Teams**.

> **Note:** 
> - A host can only belong to one team. 
> - You can give users access to only some teams.

## Best practice

Fleet's best practice teams: 
- `💻 Workstations`: End users' production work computers (macOS, Windows, and Linux)
- `💻🐣 Workstations (canary)`: IT team's test work computers. Sometimes, for demos or testing, includes end user's work computers. Used for [dogfooding](https://en.wikipedia.org/wiki/Eating_your_own_dog_food) a new workflow or feature that may or may not be rolled out to the "Workstations" team.
- `☁️ Servers`: Security team's production servers.
- `☁️🐣 Servers (canary)`: Security team's test servers.
- `Compliance exclusions`: All contributors' test work computers or virtual machines (VMs). Used for validating workflows for Fleet customers or reproducing bugs in the Fleet product.
- `📱🏢 Company-owned iPhones`: iPhones purchased by the organization that enroll to Fleet automatically via Apple Business Manager. For example, iPhones used by iOS Engineers.
- `🔳🏢 Company-owned iPads`: iPads purchased by the organization that enroll to Fleet automatically via Apple Business Manager. For example, conference-room iPads.
- `📱🔐 Personally-owned iPhones`: End users' personal iPhones, like those enrolled through a BYOD program, that have access to company resources.

If some of your hosts don't fit into the teams listed above, consider their purpose. This will help determine their risk and compliance requirements, which in turn define their security baseline and appropriate team in Fleet. If these hosts have distinct compliance needs and security baselines, it's advisable to create a new team in Fleet.

## Add hosts to a team

You can add hosts to a team in Fleet by either enrolling the host with a team's enroll secret or by transferring the host via Fleet UI after the host has been enrolled to Fleet.

### Enroll hosts with a team's enroll secret

1. In Fleet UI, navigate to **Settings > Teams** and select the team you wish to add a host to.
2. Select **Add hosts** and follow the on-screen instructions.

> Quick tip: When viewing a specific team (from the **Teams** dropdown), Selecting **Add hosts** will display instructions to add new hosts directly to that team.

### Transfer a host

1. In FleetUI, navigate to the **Hosts** page and select the host you wish to transfer.
2. From the host details page, press **Actions > Transfer** and follow the on-screen instructions.

> Quick tip: You can hit the checkbox next to the host you wish to transfer to access its quick menu. From there, select **Transfer** and follow the on-screen instructions.

## Advanced

You can automatically enroll hosts to a specific team in Fleet by installing a fleetd with a team enroll secret. Learn more [here](https://fleetdm.com/guides/enroll-hosts#enroll-host-to-a-specific-team).

Changing the host's enroll secret after enrollment will not cause the host to be transferred to a different team.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2024-07-11">
<meta name="articleTitle" value="Teams">
<meta name="description" value="Learn how to group hosts in Fleet to apply specific queries, policies, and agent options using teams.">
