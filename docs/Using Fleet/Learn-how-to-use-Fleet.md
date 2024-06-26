# Learn how to use Fleet

- [How to add your device to Fleet](#how-to-add-your-device-to-fleet)
- [How to ask questions about your device](#how-to-ask-questions-about-your-device)

### Overview

In this guide, we'll cover the following concepts:
- How to add your device to Fleet
- How to ask questions about your device

### How to add your device to Fleet

Once you log into Fleet, you are presented with the **Home** page.

To add your device: 

1. Select **Add hosts**. In Fleet, devices are referred to as "hosts."
2. Select your device's platform.
3. Select **Download** to download Fleet's agent (fleetd). The download may take several seconds.
4. Open fleetd and follow the installation steps.

> It may take several seconds for Fleet osquery to send your device's data to Fleet.

In the background, Fleet ran several checks to assess the security hygiene of your device.

> In Fleet, these checks are referred to as "policies."

### How to ask questions about your device

With Fleet, you can ask a multitude of questions to help you manage, monitor, and identify threats on your devices, but if you are just starting out, and unsure of what to ask, Fleet comes baked in with a [query library](https://fleetdm.com/queries) of common questions.

So, let's start by asking the following question about your device:

* What operating system is installed on my device and what is its version?

This question can easily be answered by running this simple query: "Get operating system information." 

To run this query on your device:

1. Select **Queries** in the top navigation.
2. Select **Create new query** (or browse your organization's queries for "operating system information" in the search bar).
3. Type the query you would like to run, `SELECT * FROM os_version;`.
4. Select **Run query**, then select **All hosts** (your device may be the only host added to Fleet), and finally select **Run** to execute the query.

The query may take several seconds to complete, because Fleet has to wait for the Fleet's agent (fleetd) to respond with results. Only online hosts will respond with results to a live query.

> Fleet's query response time is inherently variable because of osquery's heartbeat response time. This helps prevent performance issues on hosts.

When the query has finished, you should see several columns in the "Results" table:

- The "name" column answers: "What operating system is installed on my device?" 

- The "version" column answers: "What version of the installed operating system is on my device?"

<meta name="pageOrderInSection" value="100">
<meta name="description" value="Get started with using Fleet by learning how to enroll your device into a Fleet instance and run queries to ask questions about it.">
<meta name="navSection" value="hidden">