# Learn how to use Fleet

> This tutorial assumes that you have a preview environment of Fleet up and running. If you haven't already done so, check out our [Get Started](https://fleetdm.com/get-started) guide for instructions on how to start a preview environment of Fleet.

In this tutorial, we'll cover the following concepts:

- [Where to see your device in Fleet](#where-to-see-your-device-in-fleet)
- [How to ask questions about your device](#how-to-ask-questions-about-your-devices)

### Where to see your devices in Fleet

Once you log into Fleet, you are presented with the **Home** page.

On this page you'll see that your own device has been added to Fleet.

>In Fleet, devices are referred to as "hosts."

In the background, Fleet ran several checks to assess the security hygiene of your device.

>In Fleet, these checks are referred to as "policies."

### How to ask questions about your devices

With osquery and Fleet, you can ask a multitude of questions to help you manage, monitor, and identify threats on your devices, but if you are just starting out, and unsure of what to ask, Fleet comes baked in with a [query library](https://fleetdm.com/queries) of common questions.

So, let's start by asking the following question about your device:

* What is the operating system installed on my device and what is its version?

This question can easily be answered, by running this simple query: "Get the version of the resident operating system." 

On the **Queries** page, enter the query name, "Get the version of the resident operating system," in the search box, and select it to enter the **query console**. Then from the **query console**, hit "Run query", and from the "Select targets" page, select "All hosts," to run this query against all hosts enrolled in your Fleet. Then hit the "Run" button to execute the query.

The query may take several seconds to complete, because Fleet has to wait for the osquery agents to respond with results.

> Fleet's query response time is inherently variable because of osquery's heartbeat response time. This helps prevent performance issues on hosts.

When the query has finished, you should see several columns in the "Results" table:

- The "name" column answers: which operating system is installed on my device? 

- The "version" column answers: which version of the installed operating system is my device running?"

