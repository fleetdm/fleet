# Fleet UI
- [Creating a query](#create-a-query)
- [Running a query](#run-a-query)
- [Scheduling a query](#schedule-a-query)
- [Update agent options](#update-agent-options)

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/1VNvg3_drow" allowfullscreen></iframe>
</div>

## Create a query

Queries in Fleet allow you to ask a multitude of questions to help you manage, monitor, and identify threats on your devices. 

If you're unsure of what to ask, head to Fleet's [query library](https://fleetdm.com/queries). There you'll find common queries that have been tested by members of our community.

How to create a query:

1. In the top navigation, select **Queries**.

2. Select **Create new query** to navigate to the query console.

3. In the **Query** field, enter your query. Remember, you can find common queries in [Fleet's library](https://fleetdm.com/queries).

4. Select **Save**, enter a name and description for your query, and select **Save query**.

## Run a query

Run a live query to get answers for all of your online hosts.

> Offline hosts won’t respond to a live query because they may be shut down, asleep, or not connected to the internet.

How to run a query:

1. In the top navigation, select **Queries**.

2. In the **Queries** table, find the query you'd like to run and select the query's name to navigate to the query console.

3. Select **Run query** to navigate to the target picker. Select **All hosts** and select **Run**. This will run the query against all your hosts.

The query may take several seconds to complete because Fleet has to wait for the hosts to respond with results.

> Fleet's query response time is inherently variable because of osquery's heartbeat response time. This helps prevent performance issues on hosts.

## Schedule a query

Fleet allows you to schedule queries. Scheduled queries will send data to your log destination automatically.

The default log destination, **filesystem**, is good to start. With this set, data is sent to the `/var/log/osquery/osqueryd.snapshots.log` file on each host’s filesystem. To see which log destinations are available in Fleet, head to the [log destinations page](https://fleetdm.com/docs/using-fleet/log-destinations).

How to schedule a query:

1. In the top navigation, select **Schedule**.

2. Select **Schedule a query**.

3. Select the **Select query** dropdown and choose the query that you'd like to run on a schedule. 

4. Select the **Frequency** dropdown and choose how often you'd like the query to run and send results to your log destination. **Every hour** is a good frequency to start. You can change this later.

5. Select **Schedule**.

With Fleet Premium, you can schedule queries for groups of hosts using [the teams feature](https://fleetdm.com/docs/using-fleet/teams). This allows you to collect different data for each group.

> In Fleet Premium, groups of hosts are called "teams."

How to use teams to schedule queries for a group of hosts:

1. If you haven't already, first [create a team](https://fleetdm.com/docs/using-fleet/teams#create-a-team) and [transfer hosts](https://fleetdm.com/docs/using-fleet/teams#transfer-hosts-to-a-team) to the team.

2. In the **Teams** dropdown below the top navigation, select the team.

3. Follow the "How to schedule a query" instructions above.

## Update agent options

<!-- Heading is kept so that the link from the Fleet UI still works -->
<span id="configuring-agent-options" name="configuring-agent-options"></span>

Fleet allows you to update the settings of the agent installed on all your hosts at once. In Fleet, these settings are called "agent options."

The default agent options are good to start. 

How to update agent options:

1. In the top navigation, select your avatar and select **Settings**. Only users with the [admin role](https://fleetdm.com/docs/using-fleet/permissions) can access the pages in **Settings**.

2. On the Organization settings page, select **Agent options** on the left side of the page.

3. Use Fleet's YAML editor to configure your osquery options, decorators, or set command line flags.

To see all agent options, head to the [agent options documentation](https://fleetdm.com/docs/using-fleet/configuration-files#agent-options).

4. Place your new setting one level below the `options` key. The new setting's key should be below and one tab to the right of `options`.

5. Select **Save**.

The agents may take several seconds to update because Fleet has to wait for the hosts to check in. Additionally, hosts enrolled with removed enroll secrets must properly rotate their secret to have the new changes take effect.

<meta name="title" value="Fleet UI">
<meta name="pageOrderInSection" value="200">
<meta name="description" value="Learn how to create, run, and schedule queries, as well as update agent options in the Fleet user interface.">
<meta name="navSection" value="The basics">
