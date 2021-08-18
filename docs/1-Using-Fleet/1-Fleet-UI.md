# Fleet UI
- [Scheduling queries](#scheduling-queries)
- [Configuring agent options](#configuring-agent-options)

## Scheduling queries

The Fleet application allows you to schedule queries. This way these queries will run on an ongoing basis against the hosts that you have installed osquery on. To schedule specific queries in Fleet, you can organize these queries into "Query Packs". To view all saved packs and perhaps create a new pack, select "Schedule" from the top nav.
![Manage Packs](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/manage-packs.png)

If you select a pack from the list, you can quickly enable and disable the entire pack, or you can configure it further.

![Manage Packs With Pack Selected](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/manage-packs-with-pack-selected.png)

When you edit a pack, you can decide which targets you would like to execute the pack. This is a similar selection experience to the target selection process that you use to execute a new query.

![Edit Pack Targets](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/edit-pack-targets.png)

To add queries to a pack, use the right-hand sidebar. You can take an existing scheduled query and add it to the pack. You can also define a few key details such as:

- interval: how often should the query be executed?
- logging: which osquery logging format would you like to use?
- platform: which operating system platforms should execute this query?
- minimum osquery version: if the table was introduced in a newer version of osquery, you may want to ensure that only sufficiently recent version of osquery execute the query.
- shard: from 0 to 100, what percent of hosts should execute this query?

![Schedule Query Sidebar](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/schedule-query-sidebar.png)


Once you've scheduled queries and curated your packs, you can read our guide to [Working With Osquery Logs](../1-Using-Fleet/5-Osquery-logs.md).

## Configuring agent options

The Fleet application allows you to specify options returned to osqueryd when it checks for configuration. See the [osquery documentation](https://osquery.readthedocs.io/en/stable/deployment/configuration/#options) for the available options.

### Global agent options

Global agent options are applied to all hosts enrolled in Fleet.

Only user's with the Admin role can edit global agent options.

To configure global agent options, head to **Settings > Organization settings > Global agent options**.

![Global agent options](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/global-agent-options.png)
 
### Team level agent options

`Applies only to Fleet Basic`

```
ℹ️  In Fleet 4.0, Teams were introduced.
```

Team agent options are applied to all hosts assigned to a specific team in Fleet.

Team agent options *override* global agent options. 

Let's say you have two teams in Fleet. One team is named "Workstations" and the other named "Servers." If you edit the agent options for the "Workstations" team, the hosts assigned to this team will now receive these agent options *instead of* the global agent options. The hosts assigned to the "Servers" team will still receive the global agent options.

To configure team agent options, head to **Settings > Teams > `Team-name-here` > Agent options**.

![Team agent options](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/team-agent-options.png)

<meta name="title" value="Fleet UI">
