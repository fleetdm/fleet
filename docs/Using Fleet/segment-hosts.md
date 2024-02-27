# Segment hosts

`Applies only to Fleet Premium`

```
ℹ️  In Fleet 4.0, Teams were introduced.
```

- [Overview](#overview)
- [Best practice](#best-practice)
- [Transfer hosts to a team](#transfer-hosts-to-a-team)

## Overview

In Fleet, you can group hosts together in a team.

Then, you can give users access to only some teams.

This means you manage permissions so that some users can only run queries and manage hosts on the teams these users have access to.

You can manage teams in the Fleet UI by selecting **Settings** > **Teams** in the top navigation. From there, you can add or remove teams, manage user access to teams, transfer hosts, or modify team settings.

## Best practice

The best practice is to create these teams: `Workstations`, `Workstations (canary)`, `Servers`, and `Servers (canary)`.



## Adding hosts to a team

Hosts can only belong to one team in Fleet.

You can add hosts to a new team in Fleet by either enrolling the host with a team's enroll secret or by transferring the host via the Fleet UI after the host has been enrolled to Fleet.

To automatically add hosts to a team in Fleet, check out the [**Adding hosts** documentation](https://fleetdm.com/docs/using-fleet/adding-hosts#automatically-adding-hosts-to-a-team).

> If a host was previously enrolled using a global enroll secret, changing the host's osquery enroll
> secret will not cause the host to be transferred to the desired team. You must delete the
> `osquery/osquery.db` file on the host, which forces the host to re-enroll
> using the new team enroll secret. Alternatively, you can transfer the host via the Fleet UI, the
> fleetctl CLI using `fleetctl hosts transfer`, or the [transfer host API endpoint](https://fleetdm.com/docs/using-fleet/rest-api#transfer-hosts-to-a-team).



<meta name="pageOrderInSection" value="1000">
<meta name="description" value="Learn how to group hosts in Fleet to apply specific queries, policies, and agent options using teams.">
<meta name="navSection" value="The basics">
