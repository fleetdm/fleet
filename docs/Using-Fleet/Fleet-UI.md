# Fleet UI
- [Create a query](#create-a-query)
- [Run a query](#run-a-query)
- [Schedule a query](#schedule-a-query)

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/1VNvg3_drow" allowfullscreen></iframe>
</div>

## Create a query

Queries in Fleet allow you to ask a multitude of questions to help you manage, monitor, and identify threats on your devices. 

If you're unsure of what to ask, head to Fleet's [query library](https://fleetdm.com/queries). There you'll find common queries that have been tested by members of our community.

How to create a query:

1. In the top navigation, select **Queries**.

2. Select **Create new query** to navigate to the query console.

3. In the **Query** field, enter your query. If you're just starting out and unsure of what to ask, head to Fleet's [query library](https://fleetdm.com/queries) of common queries.

4. Select **Save**, enter a name an description for your query, and select **Save query**.

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

Fleet allows you to schedule queries. Scheduled queries will run and send results to your log destination automatically.

How to schedule a query:

1. In the top navigation, select **Schedule**.

2. Select **Schedule a query**.

3. Select the **Select query** dropdown and choose the query that you'd like to run on a schedule. 

4. Select the **Frequency** dropdown and choose how often you'd like the query to run and send results to your log destination. **Every hour** is a good frequency to start. You can change this later.

5. Select **Schedule**.

To see which log destinations are available in Fleet, head to the [osquery logs guide](../Using-Fleet/Osquery-logs.md).

<meta name="title" value="Fleet UI">

<meta name="pageOrderInSection" value="200">
