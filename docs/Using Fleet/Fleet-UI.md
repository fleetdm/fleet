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
> Avoid using dot notation (".") for column names in your queries as it can cause results to render incorrectly in Fleet UI. Please see [issue #15446](https://github.com/fleetdm/fleet/issues/15446) for more details. 

4. Select **Save**, enter a name and description for your query, select the frequency that the query should run at, and select **Save query**.

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

*In Fleet 4.35.0, the "Schedule" page was removed, and query automations are now configured on the "Queries" page. Instructions for scheduling queries in earlier versions of Fleet can be found [here](https://github.com/fleetdm/fleet/blob/ac797c8f81ede770853c25fd04102da9f5e109bf/docs/Using-Fleet/Fleet-UI.md#schedule-a-query).*

Fleet allows you to schedule queries to run at a set frequency. Scheduled queries will send data to Fleet and/or your [log destination](https://fleetdm.com/docs/using-fleet/log-destinations) automatically. 

By default, queries that run on a schedule will only target platforms compatible with that query. This behavior can be overridden by setting the platforms in **Advanced options** when saving a query.

**How to send data to your log destination:**

*Only users with the [admin role](https://fleetdm.com/docs/using-fleet/manage-access#admin) can manage query automations.*

1. In the top navigation, select **Queries**.

2. Select **Manage automations**.

3. Check the box next to the queries you want to send data to your log destination, and select **Save**. (The frequency that queries run at is set when a query is created.)

> Note: When viewing a specific [team](https://fleetdm.com/docs/using-fleet/segment-hosts) in Fleet Premium, only queries that belong to the selected team will be listed. When configuring query automations for all hosts, only global queries will be listed.

## Update agent options

<!-- Heading is kept so that the link from the Fleet UI still works -->
<span id="configuring-agent-options" name="configuring-agent-options"></span>

> This content was relocated on 31st August 2023. 

See "[Agent configuration](https://fleetdm.com/docs/configuration/agent-configuration)" to learn how to simultaneously update agent options from the Fleet UI or fleetctl command line tool.

<meta name="title" value="Fleet UI">
<meta name="pageOrderInSection" value="200">
<meta name="description" value="Learn how to create, run, and schedule queries, as well as update agent options in the Fleet user interface.">
<meta name="navSection" value="The basics">
