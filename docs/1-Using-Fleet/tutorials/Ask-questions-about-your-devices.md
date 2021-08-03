# Ask questions about your devices

> This tutorial assumes that you have a preview environment of Fleet up and running. Check out [the "Try Fleet" instructions](../../../README.md#try-fleet) on how to start a preview environment of Fleet.

This tutorial covers the following Fleet concepts:

- Where to see your devices in Fleet
- How to add Fleet's standard query library
- How to asking questions about your devices by running queries

### Devices in Fleet

Immediately after logging in to Fleet, you're presented with the **Hosts** page. In Fleet, devices are called "hosts." 

On this page you'll see 7 hosts. These hosts are simulated, Linux devices and, like the Fleet preview environment, they're running locally on your computer in Docker.

In this tutorial we'll be asking questions about these devices by running some queries against them.

### Add queries

Fleet uses queries to determine the information to return from your devices. Put simply, a query is a specific question you can ask about your devices. 

> Fleet facilitates providing the answer to these questions by communicating to the osquery agent that runs on any device. To learn more about osquery, check out [the osquery documentation](https://osquery.readthedocs.io/en/stable/).

In Fleet, select the "Queries" tab from the top navigation bar. You should see an empty "Queries" table. This is because you don't have any queries yet in your Fleet.

We'll now populate your Fleet with Fleet's standard query library.

First, head to the following file in the fleetdm/fleet GitHub repository: https://github.com/fleetdm/fleet/blob/main/docs/1-Using-Fleet/standard-query-library/standard-query-library.yml

Copy the contents of this file into a new file on you local computer called `standard-query-library.yml`. Please note where this file is located in your local filesystem.

> You can manage your Fleet with configuration files in yaml syntax. This concept might be familiar to those that have used Kubernetes or another tool that offers yaml configuration files. Checkout [the configuration file documentation](../configuration-files/README.md) for more information on managing Fleet with configuration files.

Now, we're going to use the `fleetctl` command-line tool to import the queries into Fleet.

Login to `fleetctl` by running the following command in your terminal window:

```
fleetctl login
```

Use the same `admin@example.com` email and `admin123#` password when prompted.

Import the queries into fleet by running the following command:

```
fleetctl apply -f standard-query-library.yml
```

> If you received a message that looks like `open standard-query-library.yml: no such file or directory` you may need to confirm the absolute path to your `standard-query-library.yml` file and change this in the command above.

Success! Now, refresh the **Queries** page in the Fleet. You should see the "Queries" table populated with Fleet's standard query library.

### Asking questions by running queries

Now, you're going to ask the following questions about the simulated, Linux hosts connected to your Fleet:

1. What version of OpenSSL is installed on each device if any?

2. Do these devices have a high severity vulnerable version of OpenSSL installed?

To answer these questions with Fleet, you're going to run the following query: "Detect Linux hosts with high severity vulnerable versions of OpenSSL." 

On the **Queries** page, find this query by entering the query's name in the search bar. In the "Queries" table, select this query to navigate to the **Edit or run query** page.

Next, select the "Select targets" dropdown and then select the purple "+" icon to the right of "All hosts." This means we'll be attempting to run this query against all hosts connected to your Fleet.

Next, select the "Run" button to run the query. The query may take several seconds to complete because Fleet must wait for the osquery agents to respond with results.

> Fleet's query response time is inherently variable because of osquery's heartbeat response time. This helps prevent performance issues on hosts.

When the query has finished, you should see 4 columns and several rows in the "Results" table:
- The "hostname" column answers: which device responded for a given row of results? 

- The "name" column answers: what is the name of the installed software item? The query you just ran asked for all software items that contain "openssl" in their name, so each row in this column should contain "openssl."

- The "source" column answers: what osquery table is the result coming from? For more information on the table's available in osquery, check out the [osquery schema documentation](https://osquery.io/schema).

- The "version" column answers: which version of the software item was detected on this device?

The "Results" table presented in Fleet answers our first question of interest which was "What version of OpenSSL is installed on each device if any?"

Next, you can compare the results in the "version" column to table below which includes the high severity vulnerabilities reported by [OpenSSL](https://www.openssl.org/news/vulnerabilities.html).


| OpenSSL version range                                                  | Vulnerability (CVE)                                                                           |
| --------------------------------------------------------- | ----------------------------------------------------------------------------- |
| 1.1.1h-1.1.1j                                             | [CVE-2021-3450](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2021-3450) |
| 1.1.1-1.1.1j                                              | [CVE-2021-3449](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2021-3449) |
| 1.1.1-1.1.1h and 1.0.2-1.0.2w                             | [CVE-2020-1971](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2020-1971) |
| 1.1.1d-1.1.1f                                             | [CVE-2020-1967](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2020-1967) |
| 1.1.1-1.1.1d and 1.0.2-1.0.2t                             | [CVE-2019-1551](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2019-1551) |
| 1.1.1-1.1.1c                                              | [CVE-2019-1549](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2019-1549) |
| 1.1.0-1.1.0d                                              | [CVE-2017-3733](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2017-3733) |
| 1.1.0-1.1.0b                                              | [CVE-2016-7054](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-7054) |
| 1.1.0 and 1.0.2-1.0.2h and 1.0.1-1.0.1t                   | [CVE-2016-6304](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-6304) |
| 1.0.2-1.0.2b and 1.0.1-1.0.1n                             | [CVE-2016-2108](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-2108) |
| 1.0.2-1.0.2f and 1.0.1-1.0.1r                             | [CVE-2016-0800](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-0800) |
| 1.0.2 and 1.0.1-1.0.1l and 1.0.0-1.0.0q and 0.9.8-0.9.8ze | [CVE-2016-0703](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-0703) |
| 1.0.2-1.0.2e                                              | [CVE-2016-0701](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-0701) |
| 1.0.2b-1.0.2c and 1.0.1n-1.0.1o                           | [CVE-2015-1793](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2015-1793) |
| 1.0.2                                                     | [CVE-2015-0291](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2015-0291) |
| 1.0.1-1.0.1i                                              | [CVE-2014-3513](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2014-3513) |
| 1.0.1-1.0.1h                                              | [CVE-2014-3511](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2014-3511) |
| 1.0.1-1.0.1h                                              | [CVE-2014-3511](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2014-3511) |

Do any of the simulated, Linux hosts have a high severity vulnerable version of OpenSSL installed? If the answer is yes, don't worry. The devices are running in a simulated Docker environment and do not provide any additional vectors for performing malicious actions against your device.

