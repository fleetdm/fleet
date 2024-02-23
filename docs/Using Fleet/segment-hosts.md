# Segment hosts

`Applies only to Fleet Premium`

```
ℹ️  In Fleet 4.0, Teams were introduced.
```

- [Overview](#overview)
- [Naming conventions](#naming-conventions)
- [Transfer hosts to a team](#transfer-hosts-to-a-team)

## Overview

In Fleet, you can group hosts together in a team.

With hosts segmented into exclusive teams, you can apply specific queries, policies, and agent options to each team.

Then you can:

- Enroll hosts to one team using team specific enroll secrets

- Apply unique agent options to each team

- Schedule queries that target one or more teams

- Run live queries against one or more teams

- Grant users access to one or more team

You can manage teams in the Fleet UI by selecting **Settings** > **Teams** in the top navigation. From there, you can add or remove teams, manage user access to teams, transfer hosts, or modify team settings.

## Naming conventions

One recommended approach is to create a team for each type of system in your organization. For example, you may have teams named: `Workstations`, `Workstations - sandbox`, `Servers`, and `Servers - sandbox`.

(A popular pattern is to end a team’s name with "- sandbox" to test new queries and configuration with staging hosts or volunteers acting as canaries.)


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
