# Ask questions about your devices

> This tutorial assumes that you have a preview environment of Fleet up and running. Check out [the "Try Fleet" instructions](../../../README.md#try-fleet) on how to start a preview environment of Fleet.

This tutorial covers the following Fleet concepts:

- Where your devices are presented in Fleet
- How to add queries to Fleet by importing Fleet's standard query library
- How to asking questions about your devices by running queries

### Devices in Fleet

Immediately after logging in to Fleet, you're presented with the "Hosts" page. In Fleet, devices are called "hosts." 

On this page you'll see 7 hosts. These hosts are simulated devices and, like the Fleet preview environment, they're running locally on your computer in Docker.

In this tutorial we'll be asking questions about these devices by running several queries.

### Add queries

Fleet uses queries to determine the information to return from your devices. Put simply, a query is a specific question you can ask about your devices. 

> Fleet facilitates providing the answer to these questions by communicating to the osquery agent that rungs on any device. To learn more about osquery, check out [the osquery documentation](https://osquery.readthedocs.io/en/stable/).

In the Fleet, select the "Query" button from the top navigation bar. You should see an empty table. This is because you don't have any queries yet in your Fleet.

We'll now populate your Fleet with Fleet's standard query library.

First, head to the following file in the fleetdm/fleet GitHub repository: https://github.com/fleetdm/fleet/blob/main/docs/1-Using-Fleet/standard-query-library/standard-query-library.yml

Copy the contents of this file into a new file on you local computer called `standard-query-library.yml`. Please take note on where this new file is located in your local filesystem.

> You can manage your Fleet with configuration files in yaml syntax. This concept might be familiar to those that have used Kubernetes or a tool that offers configuration files. Checkout [the configuration file documentation](../configuration-files/README.md) for more information on managing Fleet with configuration files.

Now, we're going to use `fleetctl` to import the queries into Fleet.

Login to `fleetctl` by running the following command in your terminal window:

```
fleetctl login
```

Use the same `admin@example.com` email and `admin123#` password when prompted.

Import the queries into fleet by running the following command:

```
fleetctl apply -f standard-query-library.yml
```

If you received a message that looks like `open standard-query-library.yml: no such file or directory` you may need to confirm the absolute path to your `standard-query-library.yml` file.


