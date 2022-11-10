# Fleet 4.23.0 | Better insight into inherited policies, improved host vitals, and more configuration visibility

Fleet 4.23.0 is up and running. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.23.0) or continue reading to get the highlights.

For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights
- Know which hosts on a team comply with inherited policies.
- Separate private and public IP addresses.
- See host disk encryption status.
- See team configuration file edits.

## Know which hosts on a team comply with inherited policies
**Available in Fleet Premium**

One of the benefits of Fleet teams is that you can assign [policies](https://fleetdm.com/securing/what-are-fleet-policies) to specific groups of devices, empowering you to [refine your compliance approach](https://fleetdm.com/securing/stay-on-course-with-your-security-compliance-goals). But your organization may have policies that apply to every device — regardless of teams or departments. We call these “inherited” policies.

Fleet 4.23.0 makes it easy to see which hosts on a team comply with inherited policies. Below the team’s policies, you’ll see the option to show inherited policies. Selecting this will show all applicable inherited policies.

Like team policies, you’ll see how many hosts are passing or failing inherited policies. Clicking the host count will generate a list of devices that are passing or failing a particular policy.

## Separate private and public IP addresses
**Available in Fleet Free and Fleet Premium**

In previous releases of Fleet, the Hosts page listed the IP address for every device, but it didn’t specify whether these IP addresses were private or public. We’ve cleared up that confusion in Fleet 4.23.0 with updates to the Hosts page.

We changed the title of the “IP address” column to “Private IP address” and added a “Public IP address” column. Now, users can easily see both private and public IP addresses.

## See host disk encryption status
**Available in Fleet Free and Fleet Premium**

Fleet 4.23.0 gives you the ability to see if a host has disk encryption enabled.

We’ve added a disk encryption field to the Host details page. This field displays an “On” or “Off” status. Hovering over the disk encryption status will bring up tooltips tailored to each operating system.

For Linux hosts, disk encryption status is only displayed if disk encryption is “On.” 
This is because Fleet detects if the `/dev/dm-1` drive is encrypted. This drive is commonly used as the location for the root file system on the Ubuntu Linux distribution.

## See team configuration file edits
**Available in Fleet Premium**

Unexpected changes to your agent options are concerning to say the least. Fleet 4.23.0 will put you at ease.

The activity feed on the Dashboard page now includes edits to team configuration files. You can see who edited the configuration file and when the edits were made.

If the edits apply to a single team, you’ll see the team’s name in the activity feed. Otherwise, the notification will mention that multiple teams have been edited.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.23.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Chris McGillicuddy">
<meta name="authorGitHubUsername" value="chris-mcgillicuddy">
<meta name="publishedOn" value="2022-11-10">
<meta name="articleTitle" value="Fleet 4.23.0 | Better insight into inherited policies, improved host vitals, and more configuration visibility">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.22.0-cover-800x450@2x.jpg">
