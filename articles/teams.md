# Teams

_Available in Fleet Premium_

In Fleet, you can group hosts together in a "team" in Fleet. This way, you can apply queries, policies, scripts, and more that are tailored to a host's risk/compliance needs.

A host can only belong to one team. 

You can give users access to only some teams.

You can manage teams by selecting your avatar in the top navigation and then **Settings > Teams**.

## Best practice

Fleet's best practice teams: 
- `💻 Workstations`: End users' production work computers (macOS, Windows, and Linux)
- `💻🐣 Workstations (canary)`: IT team's test work computers. Sometimes, for demos or testing, includes end user's work computers. Used for [dogfooding](https://en.wikipedia.org/wiki/Eating_your_own_dog_food) a new workflow or feature that may or may not be rolled out to the "Workstations" team.
- `☁️ Servers`: Security team's production servers.
- `☁️🐣 Servers (canary)`: Security team's test servers.
- `Compliance exclusions`: All contributors' test work computers or virtual machines (VMs). Used for validating workflows for Fleet customers or reproducing bugs in the Fleet product.
- `📱🏢 Company-owned iPhones`: iPhones purchased by the organization that enroll to Fleet automatically via Apple Business Manager. For example, iPhones used by iOS Engineers.
- `🔳🏢 Company-owned iPads`: iPads purchased by the organization that enroll to Fleet automatically via Apple Business Manager. For example, conference-room iPads.
- `📱🔐 Personally-owned iPhones`: End users' personal iPhones that have access to company resources. For example, Slack and Gmail.

If some of your hosts don't fall under the above teams, what are these hosts for? The answer determines the the hosts' risk/compliance needs, and thus their security basline, and thus their "team" in Fleet. If the hosts' have a different compliance needs, and thus different security baseline, then it's time to create a new team in Fleet.

## Adding hosts to a team

You can add hosts to a new team in Fleet by either enrolling the host with a team's enroll secret or by transferring the host via the Fleet UI after the host has been enrolled to Fleet.

## Advanced

You can automatically enroll hosts to a specific team in Fleet by installing a fleetd with a team enroll secret. Learn more [here](https://fleetdm.com/guides/enroll-hosts#enroll-host-to-a-specific-team).

Changing the host's enroll secret after enrollment will not cause the host to be transferred to a different team.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2024-07-11">
<meta name="articleTitle" value="Teams">
<meta name="description" value="Learn how to group hosts in Fleet to apply specific queries, policies, and agent options using teams.">
