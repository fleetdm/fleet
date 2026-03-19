# Fleets

> Fleets are available in Fleet Premium.

In Fleet, you can organize hosts into fleets to apply queries, policies, scripts, and other configurations tailored to their specific risk and compliance requirements.

A host can only belong to one fleet. You can give users access to mutiple fleet.

To manage fleets:

1. Select your avatar in the top navigation.
2. Choose **Settings > Fleets**.

## Best practice

Fleet's best practice fleets:
- `💻 Workstations`: End users' production work computers (macOS, Windows, and Linux)
- `☁️ IT servers`: Production servers used to host internal tools like certificate authorities (CAs).
- `📱🔐 Personal mobile devices`: iPhones, iPads, and Android devices owned by employees that can access company data.
- `📱🏢 Employee-issued mobile devices`: iPhones, iPads, and Android devices issued to employees that can access company data.
- `🖥️ Dedicated devices`: iPads or iPhones for dedicated or shared use. If some of your devices have different use cases, break this fleet into separate fleets (ex. `🖥️ Kiosk devices` and `🎥 Zoom room devices`).

## Add hosts to a fleet

You can add hosts to a fleet in Fleet by either enrolling the host with a fleet's enroll secret or by transferring the host via Fleet UI after the host has been enrolled to Fleet.

### Transfer a host

1. In Fleet UI, navigate to the **Hosts** page and select the host you wish to transfer.
2. From the host details page, press **Actions > Transfer** and follow the on-screen instructions.

> Quick tip: You can hit the checkbox next to the host you wish to transfer to access its quick menu. From there, select **Transfer** and follow the on-screen instructions.

## Advanced

You can automatically enroll hosts to a specific fleet in Fleet by installing a fleetd agent with a [fleet enroll secret](https://fleetdm.com/guides/enroll-hosts#enroll-host-to-a-specific-fleet).

Changing the host's enroll secret after enrollment will not cause the host to be transferred to a different fleet.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2024-07-11">
<meta name="articleTitle" value="Fleets">
<meta name="description" value="Learn how to group hosts in Fleet to apply specific queries, policies, and agent options using fleets.">
